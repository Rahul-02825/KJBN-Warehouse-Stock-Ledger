package api

import (
	"context"
	"slices"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

func planPick(ctx context.Context, q *sqlc.Queries, warehouse sqlc.Warehouse, product sqlc.Product, amount decimal.Decimal, locationID int64) (binPlan, error) {
	bins, err := loadBins(ctx, q, warehouse.ID, product.ID)
	if err != nil {
		return binPlan{}, err
	}
	sources := stockedBins(bins)
	quantity := domain.ToUnits(amount)
	step := domain.PlacementStep(product.Unit)

	start, reason, ok := pickStart(bins, sources, locationID, step)
	if !ok {
		return binPlan{}, domain.InsufficientStockError{Location: "warehouse " + warehouse.Code, Available: decimal.Zero, Requested: amount}
	}
	allocations, fit := domain.Plan(start, sources, quantity, step)
	if fit < quantity {
		return binPlan{}, domain.InsufficientStockError{Location: "warehouse " + warehouse.Code, Available: domain.FromUnits(fit), Requested: amount}
	}
	return binPlan{Warehouse: warehouse, Start: start, Reason: reason, Allocations: allocations}, nil
}

func stockedBins(bins []domain.Bin) []domain.Bin {
	var out []domain.Bin
	for _, b := range bins {
		if b.ProductQuantity > 0 {
			held := b.ProductQuantity
			out = append(out, domain.Bin{ID: b.ID, Code: b.Code, Position: b.Position, Capacity: &held, ProductQuantity: held, AutoPlace: true})
		}
	}
	return out
}

func pickStart(bins, sources []domain.Bin, locationID, step int64) (domain.Bin, string, bool) {
	if locationID == 0 {
		return domain.ChooseStart(sources, domain.Point{}, step)
	}
	if i := slices.IndexFunc(sources, func(b domain.Bin) bool { return b.ID == locationID }); i >= 0 {
		return sources[i], domain.ReasonChosen, true
	}
	i := slices.IndexFunc(bins, func(b domain.Bin) bool { return b.ID == locationID })
	empty := int64(0)
	return domain.Bin{ID: bins[i].ID, Code: bins[i].Code, Position: bins[i].Position, Capacity: &empty}, domain.ReasonChosen, true
}

func (p binPlan) pickChanges() []change {
	out := make([]change, len(p.Allocations))
	for i, a := range p.Allocations {
		out[i] = change{locationID: a.Bin.ID, delta: domain.FromUnits(a.Quantity).Neg()}
	}
	return out
}
