package api

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

const maxCountLines = 500

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

func (r *StockTakeRequest) validate() map[string]string {
	r.Note = strings.TrimSpace(r.Note)
	f := map[string]string{}
	checkIDs(f, map[string]int64{"location_id": r.LocationID})
	switch {
	case len(r.Counts) == 0:
		f["counts"] = "add at least one counted product"
	case len(r.Counts) > maxCountLines:
		f["counts"] = "at most " + strconv.Itoa(maxCountLines) + " products per count"
	}
	seen := map[int64]bool{}
	for i, c := range r.Counts {
		field := "counts." + strconv.Itoa(i)
		switch {
		case c.ProductID < 1:
			f[field] = "product_id must be a positive integer"
		case seen[c.ProductID]:
			f[field] = "each product can be counted once"
		case domain.ValidateCount(c.Counted) != "":
			f[field] = domain.ValidateCount(c.Counted)
		}
		seen[c.ProductID] = true
	}
	if len(r.Note) > 1000 {
		f["note"] = "max 1000 characters"
	}
	r.Reference, r.CreatedBy = checkText(f, r.Reference, r.CreatedBy)
	return f
}

type stockTakeResult struct {
	Adjustments []sqlc.Movement
	Unchanged   int
}

func (l *Ledger) PostStockTake(ctx context.Context, req StockTakeRequest) (result stockTakeResult, err error) {
	counts := slices.Clone(req.Counts)
	slices.SortFunc(counts, func(a, b CountLine) int { return cmp.Compare(a.ProductID, b.ProductID) })
	reference := req.Reference
	if reference == "" {
		reference = "Stock take"
	}

	err = l.inTx(ctx, func(q *sqlc.Queries) error {
		location, err := getLocation(ctx, q, req.LocationID)
		if err != nil {
			return err
		}
		for _, c := range counts {
			product, err := lockProduct(ctx, q, c.ProductID, c.Counted)
			if err != nil {
				return err
			}
			balance, err := q.GetStockBalance(ctx, sqlc.GetStockBalanceParams{ProductID: product.ID, LocationID: location.ID})
			if err != nil {
				return err
			}
			difference := c.Counted.Sub(balance.Quantity)
			if difference.IsZero() {
				result.Unchanged++
				continue
			}
			row := adjustmentRow(product, location.ID, difference, domain.StockTakeReason, req.Note, reference, req.CreatedBy)
			m, err := applyChanges(ctx, q, product, row, []change{{location.ID, difference}})
			if err != nil {
				return err
			}
			result.Adjustments = append(result.Adjustments, m)
		}
		return nil
	})
	return result, err
}

func (h *MovementHandler) stockTake(w http.ResponseWriter, r *http.Request) {
	var req StockTakeRequest
	if !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	result, err := h.ledger.PostStockTake(r.Context(), req)
	if err != nil {
		writePostingError(w, r, err)
		return
	}
	adjustments := make([]movementResponse, len(result.Adjustments))
	for i, m := range result.Adjustments {
		adjustments[i] = toMovementResponse(m)
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"location_id": req.LocationID,
		"adjustments": adjustments,
		"unchanged":   result.Unchanged,
	})
}
