package ledger

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type BinPlan struct {
	Warehouse   sqlc.Warehouse
	Start       domain.Bin
	Reason      string
	Allocations []domain.Allocation
}

func (p BinPlan) receiptChanges() []stockChange {
	changes := []stockChange{}
	for _, a := range p.Allocations {
		changes = append(changes, stockChange{LocationID: a.Bin.ID, Delta: domain.FromUnits(a.Quantity)})
	}
	return changes
}

func (p BinPlan) pickChanges() []stockChange {
	changes := []stockChange{}
	for _, a := range p.Allocations {
		changes = append(changes, stockChange{LocationID: a.Bin.ID, Delta: domain.FromUnits(a.Quantity).Neg()})
	}
	return changes
}

func findWarehouse(ctx context.Context, q *sqlc.Queries, locationID, warehouseID int64, lock bool) (sqlc.Warehouse, error) {
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
		if postgres.IsNotFound(err) {
			return sqlc.Warehouse{}, domain.NotFoundError{What: "warehouse"}
		}
		if err != nil {
			return sqlc.Warehouse{}, err
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
	if postgres.IsNotFound(err) {
		return sqlc.Warehouse{}, domain.NotFoundError{What: "warehouse"}
	}
	return warehouse, err
}

func loadBins(ctx context.Context, q *sqlc.Queries, warehouseID, productID int64) ([]domain.Bin, error) {
	rows, err := q.ListBinUsage(ctx, sqlc.ListBinUsageParams{WarehouseID: warehouseID, ProductID: productID})
	if err != nil {
		return nil, err
	}
	bins := []domain.Bin{}
	for _, row := range rows {
		bins = append(bins, domain.Bin{
			ID:              row.ID,
			Code:            row.Code,
			Position:        domain.Point{X: row.X, Y: row.Y},
			Capacity:        capacityInUnits(row.Capacity),
			Used:            domain.ToUnits(row.Used),
			ProductQuantity: domain.ToUnits(row.ProductQuantity),
			AutoPlace:       row.AutoPlace,
		})
	}
	return bins, nil
}

func capacityInUnits(capacity *int64) *int64 {
	if capacity == nil {
		return nil
	}
	units := *capacity * domain.UnitScale
	return &units
}

func findBin(bins []domain.Bin, id int64) (domain.Bin, bool) {
	for _, bin := range bins {
		if bin.ID == id {
			return bin, true
		}
	}
	return domain.Bin{}, false
}

func warehouseFree(warehouse sqlc.Warehouse, bins []domain.Bin) int64 {
	if warehouse.Capacity == nil {
		return domain.Bin{}.Free()
	}
	var used int64
	for _, bin := range bins {
		used += bin.Used
	}
	return max(*warehouse.Capacity*domain.UnitScale-used, 0)
}

func planReceipt(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (BinPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return BinPlan{}, err
	}
	quantity := domain.ToUnits(amount)
	step := domain.PlacementStep(product.Unit)
	place := "warehouse " + warehouse.Code

	var start domain.Bin
	var reason string
	if locationID > 0 {
		start, _ = findBin(bins, locationID)
		reason = domain.ReasonChosen
	} else {
		var ok bool
		start, reason, ok = domain.ChooseStart(bins, domain.Point{X: warehouse.X, Y: warehouse.Y}, step)
		if !ok {
			return BinPlan{}, domain.NoSpaceError{Place: place, Requested: amount}
		}
	}

	allocations, fit := domain.Plan(start, bins, quantity, step)
	fit = min(fit, warehouseFree(warehouse, bins))
	if fit < quantity {
		return BinPlan{}, domain.NoSpaceError{Place: place, Fit: domain.FromUnits(fit), Requested: amount}
	}
	return BinPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func planPick(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (BinPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return BinPlan{}, err
	}
	sources := binsHoldingProduct(bins)
	quantity := domain.ToUnits(amount)
	step := domain.PlacementStep(product.Unit)
	place := "warehouse " + warehouse.Code

	var start domain.Bin
	var reason string
	if locationID > 0 {
		start, reason = chosenPickBin(bins, sources, locationID)
	} else {
		var ok bool
		start, reason, ok = domain.ChooseStart(sources, domain.Point{}, step)
		if !ok {
			return BinPlan{}, domain.InsufficientStockError{Location: place, Available: decimal.Zero, Requested: amount}
		}
	}

	allocations, fit := domain.Plan(start, sources, quantity, step)
	if fit < quantity {
		return BinPlan{}, domain.InsufficientStockError{Location: place, Available: domain.FromUnits(fit), Requested: amount}
	}
	return BinPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func binsHoldingProduct(bins []domain.Bin) []domain.Bin {
	sources := []domain.Bin{}
	for _, bin := range bins {
		if bin.ProductQuantity > 0 {
			held := bin.ProductQuantity
			sources = append(sources, domain.Bin{
				ID:              bin.ID,
				Code:            bin.Code,
				Position:        bin.Position,
				Capacity:        &held,
				ProductQuantity: held,
				AutoPlace:       true,
			})
		}
	}
	return sources
}

func chosenPickBin(bins, sources []domain.Bin, locationID int64) (domain.Bin, string) {
	if bin, ok := findBin(sources, locationID); ok {
		return bin, domain.ReasonChosen
	}
	bin, _ := findBin(bins, locationID)
	empty := int64(0)
	return domain.Bin{ID: bin.ID, Code: bin.Code, Position: bin.Position, Capacity: &empty}, domain.ReasonChosen
}

func ensureSpace(ctx context.Context, q *sqlc.Queries, product sqlc.Product, changes []stockChange) error {
	netByWarehouse := map[int64]int64{}
	addsByWarehouse := map[int64][]stockChange{}
	for _, change := range changes {
		location, err := getLocation(ctx, q, change.LocationID)
		if err != nil {
			return err
		}
		netByWarehouse[location.WarehouseID] += domain.ToUnits(change.Delta)
		if change.Delta.IsPositive() {
			addsByWarehouse[location.WarehouseID] = append(addsByWarehouse[location.WarehouseID], change)
		}
	}

	warehouseIDs := []int64{}
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

func ensureWarehouseSpace(ctx context.Context, q *sqlc.Queries, productID, warehouseID int64, adds []stockChange, net int64) error {
	warehouse, err := q.LockWarehouse(ctx, warehouseID)
	if err != nil {
		return err
	}
	bins, err := loadBins(ctx, q, warehouseID, productID)
	if err != nil {
		return err
	}
	for _, change := range adds {
		bin, _ := findBin(bins, change.LocationID)
		if domain.ToUnits(change.Delta) > bin.Free() {
			return domain.NoSpaceError{Place: "bin " + bin.Code, Fit: domain.FromUnits(bin.Free()), Requested: change.Delta}
		}
	}
	if free := warehouseFree(warehouse, bins); net > free {
		return domain.NoSpaceError{Place: "warehouse " + warehouse.Code, Fit: domain.FromUnits(free), Requested: domain.FromUnits(net)}
	}
	return nil
}
