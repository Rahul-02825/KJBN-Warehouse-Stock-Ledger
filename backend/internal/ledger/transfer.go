package ledger

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type TransferRequest struct {
	ProductID      int64           `json:"product_id"`
	FromLocationID int64           `json:"from_location_id"`
	ToLocationID   int64           `json:"to_location_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`
	IdempotencyKey string          `json:"-"`
}

func (r *TransferRequest) Validate() error {
	fields := map[string]string{}
	requireID(fields, "product_id", r.ProductID)
	requireID(fields, "from_location_id", r.FromLocationID)
	requireID(fields, "to_location_id", r.ToLocationID)
	if r.FromLocationID != 0 && r.FromLocationID == r.ToLocationID {
		fields["to_location_id"] = "must be different from from_location_id"
	}
	checkAmount(fields, "quantity", r.Quantity)
	r.Reference = checkReference(fields, r.Reference)
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	return result(fields)
}

func (l *Ledger) Transfer(ctx context.Context, req TransferRequest) (sqlc.Movement, bool, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Movement{}, false, err
	}
	movement, err := l.transfer(ctx, req)
	if errors.Is(err, errReplay) {
		return l.findReplay(ctx, req.IdempotencyKey, "TRANSFER", req.ProductID)
	}
	return movement, false, err
}

func (l *Ledger) transfer(ctx context.Context, req TransferRequest) (sqlc.Movement, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return sqlc.Movement{}, err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)

	product, err := lockProduct(ctx, q, req.ProductID, req.Quantity)
	if err != nil {
		return sqlc.Movement{}, err
	}
	from, err := getLocation(ctx, q, req.FromLocationID)
	if err != nil {
		return sqlc.Movement{}, err
	}
	to, err := getLocation(ctx, q, req.ToLocationID)
	if err != nil {
		return sqlc.Movement{}, err
	}

	changes := []stockChange{
		{LocationID: from.ID, Delta: req.Quantity.Neg()},
		{LocationID: to.ID, Delta: req.Quantity},
	}
	movement, err := saveMovement(ctx, q, product, sqlc.CreateMovementParams{
		ProductID:      product.ID,
		MovementType:   "TRANSFER",
		FromLocationID: &from.ID,
		ToLocationID:   &to.ID,
		Quantity:       req.Quantity,
		UnitCost:       &product.AverageCost,
		Reference:      req.Reference,
		CreatedBy:      req.CreatedBy,
		IdempotencyKey: nilIfEmpty(req.IdempotencyKey),
	}, changes)
	if err != nil {
		return sqlc.Movement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Movement{}, err
	}
	return movement, nil
}
