package ledger

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type ShipRequest struct {
	ProductID      int64           `json:"product_id"`
	LocationID     int64           `json:"location_id"`
	WarehouseID    int64           `json:"warehouse_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`
	IdempotencyKey string          `json:"-"`
}

func (r *ShipRequest) Validate() error {
	fields := map[string]string{}
	requireID(fields, "product_id", r.ProductID)
	optionalID(fields, "location_id", r.LocationID)
	optionalID(fields, "warehouse_id", r.WarehouseID)
	checkAmount(fields, "quantity", r.Quantity)
	r.Reference = checkReference(fields, r.Reference)
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	return result(fields)
}

func (l *Ledger) Ship(ctx context.Context, req ShipRequest) (sqlc.Movement, bool, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Movement{}, false, err
	}
	movement, err := l.ship(ctx, req)
	if errors.Is(err, errReplay) {
		return l.findReplay(ctx, req.IdempotencyKey, "OUT", req.ProductID)
	}
	return movement, false, err
}

func (l *Ledger) ship(ctx context.Context, req ShipRequest) (sqlc.Movement, error) {
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
	warehouse, err := findWarehouse(ctx, q, req.LocationID, req.WarehouseID, false)
	if err != nil {
		return sqlc.Movement{}, err
	}
	plan, err := planPick(ctx, q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		return sqlc.Movement{}, err
	}

	movement, err := saveMovement(ctx, q, product, sqlc.CreateMovementParams{
		ProductID:      product.ID,
		MovementType:   "OUT",
		FromLocationID: &plan.Allocations[0].Bin.ID,
		Quantity:       req.Quantity,
		UnitCost:       &product.AverageCost,
		Reference:      req.Reference,
		CreatedBy:      req.CreatedBy,
		IdempotencyKey: nilIfEmpty(req.IdempotencyKey),
	}, plan.pickChanges())
	if err != nil {
		return sqlc.Movement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Movement{}, err
	}
	return movement, nil
}
