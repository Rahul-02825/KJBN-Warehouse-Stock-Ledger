package ledger

import (
	"context"
	"slices"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type CountLine struct {
	ProductID int64           `json:"product_id"`
	Counted   decimal.Decimal `json:"counted"`
}

type StockTakeRequest struct {
	LocationID int64       `json:"location_id"`
	Counts     []CountLine `json:"counts"`
	Note       string      `json:"note"`
	Reference  string      `json:"reference"`
	CreatedBy  string      `json:"created_by"`
}

type StockTakeResult struct {
	Adjustments []sqlc.Movement
	Unchanged   int
}

func (r *StockTakeRequest) Validate() error {
	fields := map[string]string{}
	requireID(fields, "location_id", r.LocationID)
	if len(r.Counts) == 0 {
		fields["counts"] = "add at least one counted product"
	}
	if len(r.Counts) > domain.MaxCountLines {
		fields["counts"] = "at most " + strconv.Itoa(domain.MaxCountLines) + " products per count"
	}

	seen := map[int64]bool{}
	for i, line := range r.Counts {
		field := "counts." + strconv.Itoa(i)
		if line.ProductID < 1 {
			fields[field] = "product_id must be a positive integer"
		} else if seen[line.ProductID] {
			fields[field] = "each product can be counted once"
		} else if msg := domain.ValidateCount(line.Counted); msg != "" {
			fields[field] = msg
		}
		seen[line.ProductID] = true
	}

	r.Note = checkNote(fields, r.Note)
	r.Reference = checkReference(fields, r.Reference)
	r.CreatedBy = checkCreatedBy(fields, r.CreatedBy)
	return result(fields)
}

func (l *Ledger) StockTake(ctx context.Context, req StockTakeRequest) (StockTakeResult, error) {
	if err := req.Validate(); err != nil {
		return StockTakeResult{}, err
	}
	reference := req.Reference
	if reference == "" {
		reference = domain.DefaultStockTakeReference
	}

	counted := map[int64]decimal.Decimal{}
	productIDs := []int64{}
	for _, line := range req.Counts {
		counted[line.ProductID] = line.Counted
		productIDs = append(productIDs, line.ProductID)
	}
	slices.Sort(productIDs)

	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return StockTakeResult{}, err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)

	location, err := getLocation(ctx, q, req.LocationID)
	if err != nil {
		return StockTakeResult{}, err
	}

	var result StockTakeResult
	for _, productID := range productIDs {
		product, err := lockProduct(ctx, q, productID, counted[productID])
		if err != nil {
			return StockTakeResult{}, err
		}
		balance, err := q.GetStockBalance(ctx, sqlc.GetStockBalanceParams{ProductID: product.ID, LocationID: location.ID})
		if err != nil {
			return StockTakeResult{}, err
		}

		difference := counted[productID].Sub(balance.Quantity)
		if difference.IsZero() {
			result.Unchanged++
			continue
		}

		row := adjustmentRow(product, location.ID, difference, domain.StockTakeReason, req.Note, reference, req.CreatedBy)
		changes := []stockChange{{LocationID: location.ID, Delta: difference}}
		movement, err := saveMovement(ctx, q, product, row, changes)
		if err != nil {
			return StockTakeResult{}, err
		}
		result.Adjustments = append(result.Adjustments, movement)
	}

	if err := tx.Commit(ctx); err != nil {
		return StockTakeResult{}, err
	}
	return result, nil
}
