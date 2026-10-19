package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

// Every posting runs in one transaction and starts by locking the product row.
// That lock serialises all postings for a product, so balance checks and the
// weighted-average calculation always see a consistent state. Conditional
// balance updates are a second guard against negative stock.

var maxAmount = decimal.New(1, 9) // 1,000,000,000

var adjustmentReasons = map[string]bool{
	"damaged": true, "expired": true, "count_correction": true, "theft": true, "other": true,
}

type validationError map[string]string

func (e validationError) Error() string { return "validation failed" }

type notFoundError struct{ what string }

func (e notFoundError) Error() string { return e.what + " not found" }

type insufficientStockError struct {
	location  string
	available decimal.Decimal
	requested decimal.Decimal
}

func (e insufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock at %s: available %s, requested %s", e.location, e.available, e.requested)
}

type notReversibleError struct{ reason string }

func (e notReversibleError) Error() string { return e.reason }

var errAlreadyReversed = errors.New("movement has already been reversed")

// errReplay means the Idempotency-Key was already used; the handler returns the original movement.
var errReplay = errors.New("idempotency key already used")

// WeightedAverage returns the new average cost after receiving qty at cost.
func WeightedAverage(oldQty, oldAvg, qty, cost decimal.Decimal) decimal.Decimal {
	if !oldQty.IsPositive() {
		return cost.Round(4)
	}
	totalValue := oldQty.Mul(oldAvg).Add(qty.Mul(cost))
	return totalValue.Div(oldQty.Add(qty)).Round(4)
}

// ValidateQuantity checks a positive amount: quantity or cost.
func ValidateQuantity(d decimal.Decimal) string {
	switch {
	case !d.IsPositive():
		return "must be greater than zero"
	case !d.Equal(d.Round(4)):
		return "must have at most 4 decimal places"
	case d.GreaterThan(maxAmount):
		return "must not exceed 1000000000"
	}
	return ""
}

// BackOutAverage removes a reversed receipt (qty at cost) from the average.
// remaining is the stock left after the receipt's quantity was taken out.
// It returns false when the receipt's value has already been shipped out.
func BackOutAverage(remaining, avg, qty, cost decimal.Decimal) (decimal.Decimal, bool) {
	if !remaining.IsPositive() {
		return avg, true // nothing left; keep the last average
	}
	value := remaining.Add(qty).Mul(avg).Sub(qty.Mul(cost))
	if value.IsNegative() {
		return avg, false
	}
	return value.Div(remaining).Round(4), true
}

func ValidReason(r string) bool { return adjustmentReasons[r] }

type Ledger struct {
	pool *pgxpool.Pool
}

func NewLedger(pool *pgxpool.Pool) *Ledger {
	return &Ledger{pool: pool}
}

// change is one bin's stock effect: positive adds, negative removes.
type change struct {
	locationID int64
	delta      decimal.Decimal
}

func (l *Ledger) PostIn(ctx context.Context, req InRequest) (m sqlc.Movement, err error) {
	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		product, err := lockProduct(ctx, q, req.ProductID, req.Quantity)
		if err != nil {
			return err
		}
		startID, changes, err := receiptChanges(ctx, q, product, req)
		if err != nil {
			return err
		}
		oldQty, err := q.GetProductTotalQuantity(ctx, product.ID)
		if err != nil {
			return err
		}
		newAvg := WeightedAverage(toDecimal(oldQty), toDecimal(product.AverageCost), req.Quantity, req.UnitCost)

		m, err = applyChanges(ctx, q, product, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "IN",
			ToLocationID:   optID(startID),
			Quantity:       toNumeric(req.Quantity),
			UnitCost:       toNumeric(req.UnitCost),
			Reference:      req.Reference,
			CreatedBy:      req.CreatedBy,
			IdempotencyKey: optText(req.IdempotencyKey),
		}, changes)
		if err != nil {
			return err
		}
		_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: product.ID, AverageCost: toNumeric(newAvg)})
		return err
	})
	return m, err
}

// PostOut values the shipment at the current average and stores that cost on the movement.
func (l *Ledger) PostOut(ctx context.Context, req OutRequest) (m sqlc.Movement, err error) {
	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		product, err := lockProduct(ctx, q, req.ProductID, req.Quantity)
		if err != nil {
			return err
		}
		warehouse, err := resolveWarehouse(ctx, q, req.LocationID, req.WarehouseID, false)
		if err != nil {
			return err
		}
		plan, err := planPick(ctx, q, warehouse, product, req.Quantity, req.LocationID)
		if err != nil {
			return err
		}
		m, err = applyChanges(ctx, q, product, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "OUT",
			FromLocationID: optID(plan.Allocations[0].Bin.ID),
			Quantity:       toNumeric(req.Quantity),
			UnitCost:       product.AverageCost,
			Reference:      req.Reference,
			CreatedBy:      req.CreatedBy,
			IdempotencyKey: optText(req.IdempotencyKey),
		}, plan.pickChanges())
		return err
	})
	return m, err
}

// PostTransfer moves stock between two bins as one movement with two entries.
// The average cost does not change.
func (l *Ledger) PostTransfer(ctx context.Context, req TransferRequest) (m sqlc.Movement, err error) {
	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		product, from, err := loadProductAndLocation(ctx, q, req.ProductID, req.FromLocationID, req.Quantity)
		if err != nil {
			return err
		}
		to, err := getLocation(ctx, q, req.ToLocationID)
		if err != nil {
			return err
		}
		m, err = applyChanges(ctx, q, product, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "TRANSFER",
			FromLocationID: optID(from.ID),
			ToLocationID:   optID(to.ID),
			Quantity:       toNumeric(req.Quantity),
			UnitCost:       product.AverageCost,
			Reference:      req.Reference,
			CreatedBy:      req.CreatedBy,
			IdempotencyKey: optText(req.IdempotencyKey),
		}, []change{{from.ID, req.Quantity.Neg()}, {to.ID, req.Quantity}})
		return err
	})
	return m, err
}

// PostAdjustment records a counted correction. The quantity is signed. It is
// valued at the current average and never changes the average.
func (l *Ledger) PostAdjustment(ctx context.Context, req AdjustmentRequest) (m sqlc.Movement, err error) {
	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		product, location, err := loadProductAndLocation(ctx, q, req.ProductID, req.LocationID, req.Quantity.Abs())
		if err != nil {
			return err
		}
		row := adjustmentRow(product, location.ID, req.Quantity, req.Reason, req.Note, req.Reference, req.CreatedBy)
		row.IdempotencyKey = optText(req.IdempotencyKey)
		m, err = applyChanges(ctx, q, product, row, []change{{location.ID, req.Quantity}})
		return err
	})
	return m, err
}

func adjustmentRow(product sqlc.Product, locationID int64, signed decimal.Decimal, reason, note, reference, createdBy string) sqlc.CreateMovementParams {
	row := sqlc.CreateMovementParams{
		ProductID:    product.ID,
		MovementType: "ADJUSTMENT",
		Quantity:     toNumeric(signed.Abs()),
		UnitCost:     product.AverageCost,
		Reason:       pgtype.Text{String: reason, Valid: true},
		Note:         note,
		Reference:    reference,
		CreatedBy:    createdBy,
	}
	if signed.IsPositive() {
		row.ToLocationID = optID(locationID)
	} else {
		row.FromLocationID = optID(locationID)
	}
	return row
}

// PostReversal undoes a movement by applying its entries with the signs flipped.
// The original row is never changed.
func (l *Ledger) PostReversal(ctx context.Context, movementID int64, req ReverseRequest) (m sqlc.Movement, err error) {
	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		original, err := q.LockMovementForReversal(ctx, movementID)
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundError{"movement"}
		}
		if err != nil {
			return err
		}
		if err := checkKey(ctx, q, req.IdempotencyKey); err != nil {
			return err
		}
		if original.MovementType == "REVERSAL" {
			return notReversibleError{"a reversal cannot itself be reversed"}
		}
		_, err = q.GetReversalForMovement(ctx, optID(original.ID))
		if err == nil {
			return errAlreadyReversed
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		product, err := q.LockProductForCostUpdate(ctx, original.ProductID)
		if err != nil {
			return err
		}
		entries, err := q.GetEntriesForReversal(ctx, original.ID)
		if err != nil {
			return err
		}
		var changes []change
		for _, e := range entries {
			changes = append(changes, change{e.LocationID, toDecimal(e.QuantityDelta).Neg()})
		}

		m, err = applyChanges(ctx, q, product, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "REVERSAL",
			FromLocationID: original.ToLocationID,
			ToLocationID:   original.FromLocationID,
			Quantity:       original.Quantity,
			UnitCost:       original.UnitCost,
			Reference:      fmt.Sprintf("Reversal of movement #%d", original.ID),
			Note:           req.Note,
			CreatedBy:      req.CreatedBy,
			ReversalOfID:   optID(original.ID),
			IdempotencyKey: optText(req.IdempotencyKey),
		}, changes)
		if isUniqueViolation(err) {
			return errAlreadyReversed
		}
		if err != nil {
			return err
		}

		if original.MovementType == "IN" {
			return backOutReceipt(ctx, q, product, original)
		}
		return nil
	})
	return m, err
}

// applyChanges updates each bin's balance, then records the movement and one
// ledger entry per change. Balance and ledger are always written together.
// The product is already locked, so a retry with the same key waits here and
// then finds the first posting.
func applyChanges(ctx context.Context, q *sqlc.Queries, product sqlc.Product, row sqlc.CreateMovementParams, changes []change) (sqlc.Movement, error) {
	if err := checkKey(ctx, q, row.IdempotencyKey.String); err != nil {
		return sqlc.Movement{}, err
	}
	if err := ensureSpace(ctx, q, product, changes); err != nil {
		return sqlc.Movement{}, err
	}
	for _, c := range changes {
		var err error
		if c.delta.IsPositive() {
			err = addStock(ctx, q, row.ProductID, c.locationID, c.delta)
		} else {
			err = takeStock(ctx, q, row.ProductID, c.locationID, c.delta.Neg())
		}
		if err != nil {
			return sqlc.Movement{}, err
		}
	}

	m, err := q.CreateMovement(ctx, row)
	if constraintName(err) == "movements_idempotency_key_key" {
		return sqlc.Movement{}, errReplay // same key used at the same moment for another product
	}
	if err != nil {
		return sqlc.Movement{}, err
	}
	for _, c := range changes {
		_, err := q.CreateMovementEntry(ctx, sqlc.CreateMovementEntryParams{
			MovementID:    m.ID,
			ProductID:     m.ProductID,
			LocationID:    c.locationID,
			QuantityDelta: toNumeric(c.delta),
			UnitCost:      row.UnitCost,
		})
		if err != nil {
			return sqlc.Movement{}, err
		}
	}
	return m, nil
}

// backOutReceipt removes a reversed receipt from the average cost.
// Called after the stock has already been taken out.
func backOutReceipt(ctx context.Context, q *sqlc.Queries, product sqlc.Product, receipt sqlc.Movement) error {
	remainingNum, err := q.GetProductTotalQuantity(ctx, product.ID)
	if err != nil {
		return err
	}
	newAvg, ok := BackOutAverage(toDecimal(remainingNum), toDecimal(product.AverageCost),
		toDecimal(receipt.Quantity), toDecimal(receipt.UnitCost))
	if !ok {
		return notReversibleError{"stock valued from this receipt has already been used; post an adjustment instead"}
	}
	_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: product.ID, AverageCost: toNumeric(newAvg)})
	return err
}

func (l *Ledger) inTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op after a successful commit

	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// loadProductAndLocation locks the product, checks the location exists, and
// rejects fractional quantities for products measured in pieces.
func loadProductAndLocation(ctx context.Context, q *sqlc.Queries, productID, locationID int64, qty decimal.Decimal) (sqlc.Product, sqlc.Location, error) {
	product, err := lockProduct(ctx, q, productID, qty)
	if err != nil {
		return sqlc.Product{}, sqlc.Location{}, err
	}
	location, err := getLocation(ctx, q, locationID)
	if err != nil {
		return sqlc.Product{}, sqlc.Location{}, err
	}
	return product, location, nil
}

func lockProduct(ctx context.Context, q *sqlc.Queries, productID int64, qty decimal.Decimal) (sqlc.Product, error) {
	product, err := q.LockProductForCostUpdate(ctx, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Product{}, notFoundError{"product"}
	}
	if err != nil {
		return sqlc.Product{}, err
	}
	if product.Unit == "pcs" && !qty.Equal(qty.Truncate(0)) {
		return sqlc.Product{}, validationError{"quantity": "must be a whole number for pcs"}
	}
	return product, nil
}

func receiptChanges(ctx context.Context, q *sqlc.Queries, product sqlc.Product, req InRequest) (int64, []change, error) {
	warehouse, err := resolveWarehouse(ctx, q, req.LocationID, req.WarehouseID, true)
	if err != nil {
		return 0, nil, err
	}
	plan, err := planReceipt(ctx, q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		return 0, nil, err
	}
	return plan.Start.ID, plan.changes(), nil
}

func getLocation(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.Location, error) {
	location, err := q.GetLocationByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Location{}, notFoundError{"location"}
	}
	return location, err
}

func addStock(ctx context.Context, q *sqlc.Queries, productID, locationID int64, qty decimal.Decimal) error {
	err := q.EnsureStockBalanceExists(ctx, sqlc.EnsureStockBalanceExistsParams{ProductID: productID, LocationID: locationID})
	if err != nil {
		return err
	}
	_, err = q.IncreaseStockBalance(ctx, sqlc.IncreaseStockBalanceParams{
		Amount: toNumeric(qty), ProductID: productID, LocationID: locationID,
	})
	return err
}

// takeStock decreases a balance only if enough stock is there.
func takeStock(ctx context.Context, q *sqlc.Queries, productID, locationID int64, qty decimal.Decimal) error {
	n, err := q.DecreaseStockBalanceIfAvailable(ctx, sqlc.DecreaseStockBalanceIfAvailableParams{
		Amount: toNumeric(qty), ProductID: productID, LocationID: locationID,
	})
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	bal, err := q.GetStockBalance(ctx, sqlc.GetStockBalanceParams{ID: productID, ID_2: locationID})
	if err != nil {
		return err
	}
	return insufficientStockError{location: bal.LocationCode, available: toDecimal(bal.Quantity), requested: qty}
}

// checkKey returns errReplay when a movement with this idempotency key already exists.
func checkKey(ctx context.Context, q *sqlc.Queries, key string) error {
	if key == "" {
		return nil
	}
	_, err := q.GetMovementByIdempotencyKey(ctx, optText(key))
	if err == nil {
		return errReplay
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func optID(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: true} }

func optText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

func isNumericOverflow(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22003"
}
