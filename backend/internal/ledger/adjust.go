package ledger

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type AdjustRequest struct {
	ProductID      int64           `json:"product_id"`
	LocationID     int64           `json:"location_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	Reason         string          `json:"reason"`
	Note           string          `json:"note"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`
	IdempotencyKey string          `json:"-"`
}

func (r *AdjustRequest) Validate() error {
	fields := map[string]string{}
	r.Reason = strings.TrimSpace(r.Reason)
	requireID(fields, "product_id", r.ProductID)
	requireID(fields, "location_id", r.LocationID)
	checkAmount(fields, "quantity", r.Quantity.Abs())
	if !domain.ValidReason(r.Reason) {
		fields["reason"] = domain.ReasonsMessage()
	}
	r.Note = checkNote(fields, r.Note)
	if domain.NeedsNote(r.Reason) && r.Note == "" {
		fields["note"] = "is required when reason is " + r.Reason
	}
	r.Reference = checkReference(fields, r.Reference)
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	return result(fields)
}

func (l *Ledger) Adjust(ctx context.Context, req AdjustRequest) (sqlc.Movement, bool, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Movement{}, false, err
	}
	movement, err := l.adjust(ctx, req)
	if errors.Is(err, errReplay) {
		return l.findReplay(ctx, req.IdempotencyKey, "ADJUSTMENT", req.ProductID)
	}
	return movement, false, err
}

func (l *Ledger) adjust(ctx context.Context, req AdjustRequest) (sqlc.Movement, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return sqlc.Movement{}, err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)

	product, err := lockProduct(ctx, q, req.ProductID, req.Quantity.Abs())
	if err != nil {
		return sqlc.Movement{}, err
	}
	location, err := getLocation(ctx, q, req.LocationID)
	if err != nil {
		return sqlc.Movement{}, err
	}

	row := adjustmentRow(product, location.ID, req.Quantity, req.Reason, req.Note, req.Reference, req.CreatedBy)
	row.IdempotencyKey = nilIfEmpty(req.IdempotencyKey)
	changes := []stockChange{{LocationID: location.ID, Delta: req.Quantity}}
	movement, err := saveMovement(ctx, q, product, row, changes)
	if err != nil {
		return sqlc.Movement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Movement{}, err
	}
	return movement, nil
}

func adjustmentRow(product sqlc.Product, locationID int64, signed decimal.Decimal, reason, note, reference, createdBy string) sqlc.CreateMovementParams {
	row := sqlc.CreateMovementParams{
		ProductID:    product.ID,
		MovementType: "ADJUSTMENT",
		Quantity:     signed.Abs(),
		UnitCost:     &product.AverageCost,
		Reason:       &reason,
		Note:         note,
		Reference:    reference,
		CreatedBy:    createdBy,
	}
	if signed.IsPositive() {
		row.ToLocationID = &locationID
	} else {
		row.FromLocationID = &locationID
	}
	return row
}
