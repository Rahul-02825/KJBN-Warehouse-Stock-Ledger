package api

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type binPlan struct {
	Warehouse   sqlc.Warehouse
	Start       domain.Bin
	Reason      string
	Allocations []domain.Allocation
}

func (p binPlan) changes() []change {
	out := make([]change, len(p.Allocations))
	for i, a := range p.Allocations {
		out[i] = change{locationID: a.Bin.ID, delta: domain.FromUnits(a.Quantity)}
	}
	return out
}

func resolveWarehouse(ctx context.Context, q *sqlc.Queries, locationID, warehouseID int64, lock bool) (sqlc.Warehouse, error) {
	id := warehouseID
	if locationID > 0 {
		location, err := getLocation(ctx, q, locationID)
		if err != nil {
			return sqlc.Warehouse{}, err
		}
		id = location.WarehouseID
	}
	if id == 0 {
		fallback, err := q.GetDefaultWarehouse(ctx)
		if err != nil {
			return sqlc.Warehouse{}, warehouseErr(err)
		}
		id = fallback.ID
	}

	var warehouse sqlc.Warehouse
	var err error
	if lock {
		warehouse, err = q.LockWarehouse(ctx, id)
	} else {
		warehouse, err = q.GetWarehouseByID(ctx, id)
	}
	return warehouse, warehouseErr(err)
}

func warehouseErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.NotFoundError{What: "warehouse"}
	}
	return err
}

func loadBins(ctx context.Context, q *sqlc.Queries, warehouseID, productID int64) ([]domain.Bin, error) {
	rows, err := q.ListBinUsage(ctx, sqlc.ListBinUsageParams{WarehouseID: warehouseID, ProductID: productID})
	if err != nil {
		return nil, err
	}
	bins := make([]domain.Bin, len(rows))
	for i, r := range rows {
		bins[i] = domain.Bin{
			ID:              r.ID,
			Code:            r.Code,
			Position:        domain.Point{X: r.X, Y: r.Y},
			Capacity:        capacityUnits(r.Capacity),
			Used:            domain.ToUnits(r.Used),
			ProductQuantity: domain.ToUnits(r.ProductQuantity),
			AutoPlace:       r.AutoPlace,
		}
	}
	return bins, nil
}

func capacityUnits(capacity *int64) *int64 {
	if capacity == nil {
		return nil
	}
	units := *capacity * domain.UnitScale
	return &units
}

func planReceipt(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (binPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return binPlan{}, err
	}
	quantity := domain.ToUnits(amount)
	step := domain.PlacementStep(product.Unit)
	place := "warehouse " + warehouse.Code
	start, reason, ok := startingBin(bins, warehouse, locationID, step)
	if !ok {
		return binPlan{}, domain.NoSpaceError{Place: place, Requested: amount}
	}

	allocations, fit := domain.Plan(start, bins, quantity, step)
	fit = min(fit, warehouseFree(warehouse, bins))
	if fit < quantity {
		return binPlan{}, domain.NoSpaceError{Place: place, Fit: domain.FromUnits(fit), Requested: amount}
	}
	return binPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func startingBin(bins []domain.Bin, warehouse sqlc.Warehouse, locationID, step int64) (domain.Bin, string, bool) {
	if locationID > 0 {
		i := slices.IndexFunc(bins, func(b domain.Bin) bool { return b.ID == locationID })
		return bins[i], domain.ReasonChosen, true
	}
	return domain.ChooseStart(bins, domain.Point{X: warehouse.X, Y: warehouse.Y}, step)
}

func warehouseFree(warehouse sqlc.Warehouse, bins []domain.Bin) int64 {
	if warehouse.Capacity == nil {
		return domain.Bin{}.Free()
	}
	var used int64
	for _, b := range bins {
		used += b.Used
	}
	return max(*warehouse.Capacity*domain.UnitScale-used, 0)
}

func ensureSpace(ctx context.Context, q *sqlc.Queries, product sqlc.Product, changes []change) error {
	netByWarehouse := map[int64]int64{}
	addsByWarehouse := map[int64][]change{}
	for _, c := range changes {
		location, err := getLocation(ctx, q, c.locationID)
		if err != nil {
			return err
		}
		netByWarehouse[location.WarehouseID] += domain.ToUnits(c.delta)
		if c.delta.IsPositive() {
			addsByWarehouse[location.WarehouseID] = append(addsByWarehouse[location.WarehouseID], c)
		}
	}

	warehouseIDs := make([]int64, 0, len(addsByWarehouse))
	for id := range addsByWarehouse {
		warehouseIDs = append(warehouseIDs, id)
	}
	slices.Sort(warehouseIDs)

	for _, id := range warehouseIDs {
		if err := ensureWarehouseSpace(ctx, q, product.ID, id, addsByWarehouse[id], netByWarehouse[id]); err != nil {
			return err
		}
	}
	return nil
}

func ensureWarehouseSpace(ctx context.Context, q *sqlc.Queries, productID, warehouseID int64, adds []change, net int64) error {
	warehouse, err := q.LockWarehouse(ctx, warehouseID)
	if err != nil {
		return err
	}
	bins, err := loadBins(ctx, q, warehouseID, productID)
	if err != nil {
		return err
	}
	for _, c := range adds {
		bin := bins[slices.IndexFunc(bins, func(b domain.Bin) bool { return b.ID == c.locationID })]
		if wanted := domain.ToUnits(c.delta); wanted > bin.Free() {
			return domain.NoSpaceError{Place: "bin " + bin.Code, Fit: domain.FromUnits(bin.Free()), Requested: c.delta}
		}
	}
	if free := warehouseFree(warehouse, bins); net > free {
		return domain.NoSpaceError{Place: "warehouse " + warehouse.Code, Fit: domain.FromUnits(free), Requested: domain.FromUnits(net)}
	}
	return nil
}
