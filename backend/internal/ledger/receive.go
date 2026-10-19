package ledger

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type ReceiveRequest struct {
	ProductID      int64           `json:"product_id"`
	LocationID     int64           `json:"location_id"`
	WarehouseID    int64           `json:"warehouse_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	UnitCost       decimal.Decimal `json:"unit_cost"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`
	IdempotencyKey string          `json:"-"`
}

func (r *ReceiveRequest) Validate() error {
	fields := map[string]string{}
	requireID(fields, "product_id", r.ProductID)
	optionalID(fields, "location_id", r.LocationID)
	optionalID(fields, "warehouse_id", r.WarehouseID)
	checkAmount(fields, "quantity", r.Quantity)
	checkAmount(fields, "unit_cost", r.UnitCost)
	r.Reference = checkReference(fields, r.Reference)
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	return result(fields)
}

func (l *Ledger) Receive(ctx context.Context, req ReceiveRequest) (sqlc.Movement, bool, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Movement{}, false, err
	}
	movement, err := l.receive(ctx, req)
	if errors.Is(err, errReplay) {
		return l.findReplay(ctx, req.IdempotencyKey, "IN", req.ProductID)
	}
	return movement, false, err
}

func (l *Ledger) receive(ctx context.Context, req ReceiveRequest) (sqlc.Movement, error) {
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
	warehouse, err := findWarehouse(ctx, q, req.LocationID, req.WarehouseID, true)
	if err != nil {
		return sqlc.Movement{}, err
	}
	plan, err := planReceipt(ctx, q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		return sqlc.Movement{}, err
	}

	oldQuantity, err := q.GetProductTotalQuantity(ctx, product.ID)
	if err != nil {
		return sqlc.Movement{}, err
	}
	newAverage := domain.WeightedAverage(oldQuantity, product.AverageCost, req.Quantity, req.UnitCost)

	movement, err := saveMovement(ctx, q, product, sqlc.CreateMovementParams{
		ProductID:      product.ID,
		MovementType:   "IN",
		ToLocationID:   &plan.Start.ID,
		Quantity:       req.Quantity,
		UnitCost:       &req.UnitCost,
		Reference:      req.Reference,
		CreatedBy:      req.CreatedBy,
		IdempotencyKey: nilIfEmpty(req.IdempotencyKey),
	}, plan.receiptChanges())
	if err != nil {
		return sqlc.Movement{}, err
	}

	_, err = q.UpdateProductAverageCost(ctx, sqlc.UpdateProductAverageCostParams{ID: product.ID, AverageCost: newAverage})
	if err != nil {
		return sqlc.Movement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.Movement{}, err
	}
	return movement, nil
}
