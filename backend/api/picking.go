package api

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

func planPick(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (binPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return binPlan{}, err
	}
	sources := stockedBins(bins)
	quantity := toUnits(amount)
	step := placementStep(product.Unit)

	start, reason, ok := pickStart(bins, sources, locationID, step)
	if !ok {
		return binPlan{}, insufficientStockError{location: "warehouse " + warehouse.Code, available: decimal.Zero, requested: amount}
	}
	allocations, fit := Plan(start, sources, quantity, step)
	if fit < quantity {
		return binPlan{}, insufficientStockError{location: "warehouse " + warehouse.Code, available: fromUnits(fit), requested: amount}
	}
	return binPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func stockedBins(bins []Bin) []Bin {
	var out []Bin
	for _, b := range bins {
		if b.ProductQuantity > 0 {
			held := b.ProductQuantity
			out = append(out, Bin{ID: b.ID, Code: b.Code, Position: b.Position, Capacity: &held, ProductQuantity: held, AutoPlace: true})
		}
	}
	return out
}

func pickStart(bins, sources []Bin, locationID, step int64) (Bin, string, bool) {
	if locationID == 0 {
		return ChooseStart(sources, Point{}, step)
	}
	if i := slices.IndexFunc(sources, func(b Bin) bool { return b.ID == locationID }); i >= 0 {
		return sources[i], ReasonChosen, true
	}
	i := slices.IndexFunc(bins, func(b Bin) bool { return b.ID == locationID })
	empty := int64(0)
	return Bin{ID: bins[i].ID, Code: bins[i].Code, Position: bins[i].Position, Capacity: &empty}, ReasonChosen, true
}

func (p binPlan) pickChanges() []change {
	out := make([]change, len(p.Allocations))
	for i, a := range p.Allocations {
		out[i] = change{locationID: a.Bin.ID, delta: fromUnits(a.Quantity).Neg()}
	}
	return out
}
