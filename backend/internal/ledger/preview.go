package ledger

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type PreviewRequest struct {
	ProductID   int64           `json:"product_id"`
	LocationID  int64           `json:"location_id"`
	WarehouseID int64           `json:"warehouse_id"`
	Quantity    decimal.Decimal `json:"quantity"`
}

type Preview struct {
	Product sqlc.Product
	Plan    BinPlan
}

func (r *PreviewRequest) Validate() error {
	fields := map[string]string{}
	requireID(fields, "product_id", r.ProductID)
	optionalID(fields, "location_id", r.LocationID)
	optionalID(fields, "warehouse_id", r.WarehouseID)
	checkAmount(fields, "quantity", r.Quantity)
	return result(fields)
}

func (l *Ledger) PreviewPlacement(ctx context.Context, req PreviewRequest) (Preview, error) {
	q := sqlc.New(l.pool)
	product, warehouse, err := l.previewSetup(ctx, q, req)
	if err != nil {
		return Preview{}, err
	}
	plan, err := planReceipt(ctx, q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		return Preview{}, err
	}
	return Preview{Product: product, Plan: plan}, nil
}

func (l *Ledger) PreviewPick(ctx context.Context, req PreviewRequest) (Preview, error) {
	q := sqlc.New(l.pool)
	product, warehouse, err := l.previewSetup(ctx, q, req)
	if err != nil {
		return Preview{}, err
	}
	plan, err := planPick(ctx, q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		return Preview{}, err
	}
	return Preview{Product: product, Plan: plan}, nil
}

func (l *Ledger) previewSetup(ctx context.Context, q *sqlc.Queries, req PreviewRequest) (sqlc.Product, sqlc.Warehouse, error) {
	if err := req.Validate(); err != nil {
		return sqlc.Product{}, sqlc.Warehouse{}, err
	}
	product, err := q.GetProductByID(ctx, req.ProductID)
	if postgres.IsNotFound(err) {
		return sqlc.Product{}, sqlc.Warehouse{}, domain.NotFoundError{What: "product"}
	}
	if err != nil {
		return sqlc.Product{}, sqlc.Warehouse{}, err
	}
	if msg := domain.CheckWholeQuantity(req.Quantity, product.Unit); msg != "" {
		return sqlc.Product{}, sqlc.Warehouse{}, domain.ValidationError{"quantity": msg}
	}
	warehouse, err := findWarehouse(ctx, q, req.LocationID, req.WarehouseID, false)
	if err != nil {
		return sqlc.Product{}, sqlc.Warehouse{}, err
	}
	return product, warehouse, nil
}
