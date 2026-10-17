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

type InInput struct {
	ProductID, LocationID int64
	Quantity, UnitCost    decimal.Decimal
	Reference, CreatedBy  string
}

type OutInput struct {
	ProductID, LocationID int64
	Quantity              decimal.Decimal
	Reference, CreatedBy  string
}

type TransferInput struct {
	ProductID, FromLocationID, ToLocationID int64
	Quantity                                decimal.Decimal
	Reference, CreatedBy                    string
}

// AdjustmentInput.Quantity is signed: positive adds stock, negative removes it.
type AdjustmentInput struct {
	ProductID, LocationID              int64
	Quantity                           decimal.Decimal
	Reason, Note, Reference, CreatedBy string
}

type ReversalInput struct {
	MovementID      int64
	Note, CreatedBy string
}

type Ledger struct {
	pool *pgxpool.Pool
}

func NewLedger(pool *pgxpool.Pool) *Ledger {
	return &Ledger{pool: pool}
}

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

func ValidReason(r string) bool { return adjustmentReasons[r] }

func (l *Ledger) PostIn(ctx context.Context, in InInput) (sqlc.Movement, error) {
	var m sqlc.Movement
	err := l.inTx(ctx, func(q *sqlc.Queries) error {
		product, location, err := loadProductAndLocation(ctx, q, in.ProductID, in.LocationID, in.Quantity)
		if err != nil {
			return err
		}

		oldQty, err := q.GetProductTotalQuantity(ctx, product.ID)
		if err != nil {
			return err
		}
		newAvg := WeightedAverage(toDecimal(oldQty), toDecimal(product.AverageCost), in.Quantity, in.UnitCost)

		m, err = q.CreateMovement(ctx, sqlc.CreateMovementParams{
			ProductID:    product.ID,
			MovementType: "IN",
			ToLocationID: optID(location.ID),
			Quantity:     toNumeric(in.Quantity),
			UnitCost:     toNumeric(in.UnitCost),
			Reference:    in.Reference,
			CreatedBy:    in.CreatedBy,
		})
		if err != nil {
			return err
		}
		if err := addEntry(ctx, q, m, location.ID, in.Quantity, in.UnitCost); err != nil {
			return err
		}
		if err := addStock(ctx, q, product.ID, location.ID, in.Quantity); err != nil {
			return err
		}
		_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: product.ID, AverageCost: toNumeric(newAvg)})
		return err
	})
	return m, err
}

// PostOut values the shipment at the current average and stores that cost on the movement.
func (l *Ledger) PostOut(ctx context.Context, in OutInput) (sqlc.Movement, error) {
	var m sqlc.Movement
	err := l.inTx(ctx, func(q *sqlc.Queries) error {
		product, location, err := loadProductAndLocation(ctx, q, in.ProductID, in.LocationID, in.Quantity)
		if err != nil {
			return err
		}
		if err := takeStock(ctx, q, product.ID, location.ID, in.Quantity); err != nil {
			return err
		}

		cost := toDecimal(product.AverageCost)
		m, err = q.CreateMovement(ctx, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "OUT",
			FromLocationID: optID(location.ID),
			Quantity:       toNumeric(in.Quantity),
			UnitCost:       toNumeric(cost),
			Reference:      in.Reference,
			CreatedBy:      in.CreatedBy,
		})
		if err != nil {
			return err
		}
		return addEntry(ctx, q, m, location.ID, in.Quantity.Neg(), cost)
	})
	return m, err
}

// PostTransfer moves stock between two bins as one movement with two entries.
// The average cost does not change.
func (l *Ledger) PostTransfer(ctx context.Context, in TransferInput) (sqlc.Movement, error) {
	var m sqlc.Movement
	err := l.inTx(ctx, func(q *sqlc.Queries) error {
		product, from, err := loadProductAndLocation(ctx, q, in.ProductID, in.FromLocationID, in.Quantity)
		if err != nil {
			return err
		}
		to, err := getLocation(ctx, q, in.ToLocationID)
		if err != nil {
			return err
		}
		if err := takeStock(ctx, q, product.ID, from.ID, in.Quantity); err != nil {
			return err
		}
		if err := addStock(ctx, q, product.ID, to.ID, in.Quantity); err != nil {
			return err
		}

		cost := toDecimal(product.AverageCost)
		m, err = q.CreateMovement(ctx, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "TRANSFER",
			FromLocationID: optID(from.ID),
			ToLocationID:   optID(to.ID),
			Quantity:       toNumeric(in.Quantity),
			UnitCost:       toNumeric(cost),
			Reference:      in.Reference,
			CreatedBy:      in.CreatedBy,
		})
		if err != nil {
			return err
		}
		if err := addEntry(ctx, q, m, from.ID, in.Quantity.Neg(), cost); err != nil {
			return err
		}
		return addEntry(ctx, q, m, to.ID, in.Quantity, cost)
	})
	return m, err
}

// PostAdjustment records a counted correction. It is valued at the current
// average and never changes the average.
func (l *Ledger) PostAdjustment(ctx context.Context, in AdjustmentInput) (sqlc.Movement, error) {
	var m sqlc.Movement
	qty := in.Quantity.Abs()
	err := l.inTx(ctx, func(q *sqlc.Queries) error {
		product, location, err := loadProductAndLocation(ctx, q, in.ProductID, in.LocationID, qty)
		if err != nil {
			return err
		}

		params := sqlc.CreateMovementParams{
			ProductID:    product.ID,
			MovementType: "ADJUSTMENT",
			Quantity:     toNumeric(qty),
			UnitCost:     product.AverageCost,
			Reason:       pgtype.Text{String: in.Reason, Valid: true},
			Note:         in.Note,
			Reference:    in.Reference,
			CreatedBy:    in.CreatedBy,
		}
		if in.Quantity.IsPositive() {
			params.ToLocationID = optID(location.ID)
			err = addStock(ctx, q, product.ID, location.ID, qty)
		} else {
			params.FromLocationID = optID(location.ID)
			err = takeStock(ctx, q, product.ID, location.ID, qty)
		}
		if err != nil {
			return err
		}

		m, err = q.CreateMovement(ctx, params)
		if err != nil {
			return err
		}
		return addEntry(ctx, q, m, location.ID, in.Quantity, toDecimal(product.AverageCost))
	})
	return m, err
}

// PostReversal undoes a movement by posting its inverse. The original row is never changed.
func (l *Ledger) PostReversal(ctx context.Context, in ReversalInput) (sqlc.Movement, error) {
	var m sqlc.Movement
	err := l.inTx(ctx, func(q *sqlc.Queries) error {
		original, err := q.LockMovementForReversal(ctx, in.MovementID)
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundError{"movement"}
		}
		if err != nil {
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
		for _, e := range entries {
			delta := toDecimal(e.QuantityDelta).Neg()
			if delta.IsPositive() {
				err = addStock(ctx, q, product.ID, e.LocationID, delta)
			} else {
				err = takeStock(ctx, q, product.ID, e.LocationID, delta.Neg())
			}
			if err != nil {
				return err
			}
		}

		if original.MovementType == "IN" {
			if err := backOutReceipt(ctx, q, product, original); err != nil {
				return err
			}
		}

		m, err = q.CreateMovement(ctx, sqlc.CreateMovementParams{
			ProductID:      product.ID,
			MovementType:   "REVERSAL",
			FromLocationID: original.ToLocationID,
			ToLocationID:   original.FromLocationID,
			Quantity:       original.Quantity,
			UnitCost:       original.UnitCost,
			Reference:      fmt.Sprintf("Reversal of movement #%d", original.ID),
			Note:           in.Note,
			CreatedBy:      in.CreatedBy,
			ReversalOfID:   optID(original.ID),
		})
		if isUniqueViolation(err) {
			return errAlreadyReversed
		}
		if err != nil {
			return err
		}
		_, err = q.CreateReversalEntries(ctx, sqlc.CreateReversalEntriesParams{
			ReversalMovementID: m.ID,
			OriginalMovementID: original.ID,
		})
		return err
	})
	return m, err
}

// backOutReceipt removes a reversed receipt from the average cost.
// Called after the stock has already been taken out.
func backOutReceipt(ctx context.Context, q *sqlc.Queries, product sqlc.Product, receipt sqlc.Movement) error {
	remainingNum, err := q.GetProductTotalQuantity(ctx, product.ID)
	if err != nil {
		return err
	}
	remaining := toDecimal(remainingNum)
	if !remaining.IsPositive() {
		return nil // nothing left; keep the last average
	}

	qty := toDecimal(receipt.Quantity)
	avg := toDecimal(product.AverageCost)
	value := remaining.Add(qty).Mul(avg).Sub(qty.Mul(toDecimal(receipt.UnitCost)))
	if value.IsNegative() {
		return notReversibleError{"stock valued from this receipt has already been used; post an adjustment instead"}
	}
	newAvg := value.Div(remaining).Round(4)
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
	product, err := q.LockProductForCostUpdate(ctx, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Product{}, sqlc.Location{}, notFoundError{"product"}
	}
	if err != nil {
		return sqlc.Product{}, sqlc.Location{}, err
	}
	location, err := getLocation(ctx, q, locationID)
	if err != nil {
		return sqlc.Product{}, sqlc.Location{}, err
	}
	if product.Unit == "pcs" && !qty.Equal(qty.Truncate(0)) {
		return sqlc.Product{}, sqlc.Location{}, validationError{"quantity": "must be a whole number for pcs"}
	}
	return product, location, nil
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

func addEntry(ctx context.Context, q *sqlc.Queries, m sqlc.Movement, locationID int64, delta, cost decimal.Decimal) error {
	_, err := q.CreateMovementEntry(ctx, sqlc.CreateMovementEntryParams{
		MovementID:    m.ID,
		ProductID:     m.ProductID,
		LocationID:    locationID,
		QuantityDelta: toNumeric(delta),
		UnitCost:      toNumeric(cost),
	})
	return err
}

func optID(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: true} }

func isNumericOverflow(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22003"
}
