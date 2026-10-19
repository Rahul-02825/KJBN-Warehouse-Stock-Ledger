package api

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

const unitScale = 10000

func toUnits(d decimal.Decimal) int64 { return d.Shift(4).IntPart() }

func fromUnits(n int64) decimal.Decimal { return decimal.New(n, -4) }

func placementStep(unit string) int64 {
	if unit == "pcs" {
		return unitScale
	}
	return 1
}

type noSpaceError struct {
	place     string
	fit       int64
	requested int64
}

func (e noSpaceError) Error() string {
	return fmt.Sprintf("cannot accommodate %s units in %s: only %s fit", fromUnits(e.requested), e.place, fromUnits(e.fit))
}

type binPlan struct {
	Warehouse   sqlc.Warehouse
	Start       Bin
	Reason      string
	Allocations []Allocation
}

func (p binPlan) changes() []change {
	out := make([]change, len(p.Allocations))
	for i, a := range p.Allocations {
		out[i] = change{locationID: a.Bin.ID, delta: fromUnits(a.Quantity)}
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
		return notFoundError{"warehouse"}
	}
	return err
}

func loadBins(ctx context.Context, q *sqlc.Queries, warehouseID, productID int64) ([]Bin, error) {
	rows, err := q.ListBinUsage(ctx, sqlc.ListBinUsageParams{WarehouseID: warehouseID, ProductID: productID})
	if err != nil {
		return nil, err
	}
	bins := make([]Bin, len(rows))
	for i, r := range rows {
		bins[i] = Bin{
			ID:              r.ID,
			Code:            r.Code,
			Position:        Point{r.X, r.Y},
			Capacity:        capacityUnits(r.Capacity),
			Used:            toUnits(toDecimal(r.Used)),
			ProductQuantity: toUnits(toDecimal(r.ProductQuantity)),
			AutoPlace:       r.AutoPlace,
		}
	}
	return bins, nil
}

func capacityUnits(capacity pgtype.Int8) *int64 {
	if !capacity.Valid {
		return nil
	}
	units := capacity.Int64 * unitScale
	return &units
}

func planReceipt(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (binPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return binPlan{}, err
	}
	quantity := toUnits(amount)
	step := placementStep(product.Unit)
	place := "warehouse " + warehouse.Code
	start, reason, ok := startingBin(bins, warehouse, locationID, step)
	if !ok {
		return binPlan{}, noSpaceError{place: place, requested: quantity}
	}

	allocations, fit := Plan(start, bins, quantity, step)
	fit = min(fit, warehouseFree(warehouse, bins))
	if fit < quantity {
		return binPlan{}, noSpaceError{place: place, fit: fit, requested: quantity}
	}
	return binPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func startingBin(bins []Bin, warehouse sqlc.Warehouse, locationID, step int64) (Bin, string, bool) {
	if locationID > 0 {
		i := slices.IndexFunc(bins, func(b Bin) bool { return b.ID == locationID })
		return bins[i], ReasonChosen, true
	}
	return ChooseStart(bins, Point{warehouse.X, warehouse.Y}, step)
}

func warehouseFree(warehouse sqlc.Warehouse, bins []Bin) int64 {
	if !warehouse.Capacity.Valid {
		return Bin{}.Free()
	}
	var used int64
	for _, b := range bins {
		used += b.Used
	}
	return max(warehouse.Capacity.Int64*unitScale-used, 0)
}

func ensureSpace(ctx context.Context, q *sqlc.Queries, product sqlc.Product, changes []change) error {
	netByWarehouse := map[int64]int64{}
	addsByWarehouse := map[int64][]change{}
	for _, c := range changes {
		location, err := getLocation(ctx, q, c.locationID)
		if err != nil {
			return err
		}
		netByWarehouse[location.WarehouseID] += toUnits(c.delta)
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
		bin := bins[slices.IndexFunc(bins, func(b Bin) bool { return b.ID == c.locationID })]
		if wanted := toUnits(c.delta); wanted > bin.Free() {
			return noSpaceError{place: "bin " + bin.Code, fit: bin.Free(), requested: wanted}
		}
	}
	if free := warehouseFree(warehouse, bins); net > free {
		return noSpaceError{place: "warehouse " + warehouse.Code, fit: free, requested: net}
	}
	return nil
}
