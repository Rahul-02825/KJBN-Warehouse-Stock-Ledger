package ledger

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type CostCheckRow struct {
	ProductID         int64           `json:"product_id"`
	SKU               string          `json:"sku"`
	Name              string          `json:"name"`
	StoredAverage     decimal.Decimal `json:"stored_average"`
	RecomputedAverage decimal.Decimal `json:"recomputed_average"`
	Difference        decimal.Decimal `json:"difference"`
	Matches           bool            `json:"matches"`
}

func (l *Ledger) CostCheck(ctx context.Context) ([]CostCheckRow, int, error) {
	q := sqlc.New(l.pool)
	movements, err := q.ListMovementsForCostCheck(ctx)
	if err != nil {
		return nil, 0, err
	}
	products, err := q.ListProductCosts(ctx)
	if err != nil {
		return nil, 0, err
	}

	quantity := map[int64]decimal.Decimal{}
	average := map[int64]decimal.Decimal{}
	for _, m := range movements {
		id := m.ProductID
		if m.MovementType == "IN" {
			average[id] = domain.WeightedAverage(quantity[id], average[id], m.Quantity, *m.UnitCost)
			quantity[id] = quantity[id].Add(m.Quantity)
		} else if m.MovementType == "REVERSAL" && m.ReversedType == "IN" {
			quantity[id] = quantity[id].Sub(m.Quantity)
			average[id], _ = domain.BackOutAverage(quantity[id], average[id], m.Quantity, *m.UnitCost)
		} else {
			quantity[id] = quantity[id].Add(m.NetDelta)
		}
	}

	rows := []CostCheckRow{}
	mismatches := 0
	for _, p := range products {
		row := CostCheckRow{
			ProductID:         p.ID,
			SKU:               p.Sku,
			Name:              p.Name,
			StoredAverage:     p.AverageCost,
			RecomputedAverage: average[p.ID],
			Difference:        p.AverageCost.Sub(average[p.ID]),
			Matches:           p.AverageCost.Equal(average[p.ID]),
		}
		if !row.Matches {
			mismatches++
		}
		rows = append(rows, row)
	}
	return rows, mismatches, nil
}
