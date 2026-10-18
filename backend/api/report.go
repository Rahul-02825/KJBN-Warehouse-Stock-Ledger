package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type ReportHandler struct {
	q *sqlc.Queries
}

func NewReportHandler(q *sqlc.Queries) *ReportHandler {
	return &ReportHandler{q: q}
}

func (h *ReportHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/reports/low-stock", h.lowStock)
	mux.HandleFunc("GET /api/reports/valuation", h.valuation)
	mux.HandleFunc("GET /api/reports/reconciliation", h.reconciliation)
	mux.HandleFunc("GET /api/products/{id}/ledger", h.ledger)
}

type lowStockRow struct {
	ProductID     int64           `json:"product_id"`
	SKU           string          `json:"sku"`
	Name          string          `json:"name"`
	Unit          string          `json:"unit"`
	ReorderLevel  decimal.Decimal `json:"reorder_level"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	Shortfall     decimal.Decimal `json:"shortfall"`
}

// lowStock lists products whose total stock across all bins is at or below the reorder level.
func (h *ReportHandler) lowStock(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListLowStockProducts(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]lowStockRow, len(rows))
	for i, p := range rows {
		data[i] = lowStockRow{
			ProductID:     p.ProductID,
			SKU:           p.Sku,
			Name:          p.Name,
			Unit:          p.Unit,
			ReorderLevel:  toDecimal(p.ReorderLevel),
			TotalQuantity: toDecimal(p.TotalQuantity),
			Shortfall:     toDecimal(p.Shortfall),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

type valuationRow struct {
	ProductID     int64           `json:"product_id"`
	SKU           string          `json:"sku"`
	Name          string          `json:"name"`
	Unit          string          `json:"unit"`
	TotalQuantity decimal.Decimal `json:"total_quantity"`
	AverageCost   decimal.Decimal `json:"average_cost"`
	TotalValue    decimal.Decimal `json:"total_value"`
}

// valuation is quantity x average cost per product, plus the grand total.
func (h *ReportHandler) valuation(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.GetInventoryValuation(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	total := decimal.Zero
	data := make([]valuationRow, len(rows))
	for i, p := range rows {
		data[i] = valuationRow{
			ProductID:     p.ProductID,
			SKU:           p.Sku,
			Name:          p.Name,
			Unit:          p.Unit,
			TotalQuantity: toDecimal(p.TotalQuantity),
			AverageCost:   toDecimal(p.AverageCost),
			TotalValue:    toDecimal(p.TotalValue),
		}
		total = total.Add(data[i].TotalValue)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "total_value": total})
}

type reconciliationRow struct {
	ProductID      int64           `json:"product_id"`
	SKU            string          `json:"sku"`
	LocationID     int64           `json:"location_id"`
	LocationCode   string          `json:"location_code"`
	CachedQuantity decimal.Decimal `json:"cached_quantity"`
	LedgerQuantity decimal.Decimal `json:"ledger_quantity"`
	Difference     decimal.Decimal `json:"difference"`
}

// reconciliation recomputes every balance from the ledger and compares it with
// the cached balance. It only reports; it never changes data.
// ?only_drift=true returns just the rows that differ.
func (h *ReportHandler) reconciliation(w http.ResponseWriter, r *http.Request) {
	onlyDrift := r.URL.Query().Get("only_drift") == "true"
	rows, err := h.q.ListReconciliation(r.Context(), onlyDrift)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	driftCount := 0
	data := make([]reconciliationRow, len(rows))
	for i, c := range rows {
		data[i] = reconciliationRow{
			ProductID:      c.ProductID,
			SKU:            c.Sku,
			LocationID:     c.LocationID,
			LocationCode:   c.LocationCode,
			CachedQuantity: toDecimal(c.CachedQuantity),
			LedgerQuantity: toDecimal(c.LedgerQuantity),
			Difference:     toDecimal(c.Difference),
		}
		if !data[i].Difference.IsZero() {
			driftCount++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "drift_count": driftCount})
}

type ledgerRow struct {
	MovementID      int64            `json:"movement_id"`
	CreatedAt       time.Time        `json:"created_at"`
	Type            string           `json:"type"`
	Reference       string           `json:"reference"`
	Reason          *string          `json:"reason"`
	CreatedBy       string           `json:"created_by"`
	LocationID      int64            `json:"location_id"`
	LocationCode    string           `json:"location_code"`
	QuantityDelta   decimal.Decimal  `json:"quantity_delta"`
	UnitCost        *decimal.Decimal `json:"unit_cost"`
	RunningTotal    decimal.Decimal  `json:"running_total"`
	RunningLocation decimal.Decimal  `json:"running_location_balance"`
}

// ledger lists every stock change for a product, oldest first, with running
// balances, so any current quantity can be explained line by line.
func (h *ReportHandler) ledger(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	product, err := h.q.GetProductByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	rows, err := h.q.GetProductLedger(r.Context(), id)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	data := make([]ledgerRow, len(rows))
	for i, e := range rows {
		data[i] = ledgerRow{
			MovementID:      e.MovementID,
			CreatedAt:       e.CreatedAt.Time.UTC(),
			Type:            e.MovementType,
			Reference:       e.Reference,
			Reason:          textPtr(e.Reason),
			CreatedBy:       e.CreatedBy,
			LocationID:      e.LocationID,
			LocationCode:    e.LocationCode,
			QuantityDelta:   toDecimal(e.QuantityDelta),
			UnitCost:        numPtr(e.UnitCost),
			RunningTotal:    toDecimal(e.RunningTotal),
			RunningLocation: toDecimal(e.RunningLocationBalance),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": product.ID,
		"sku":        product.Sku,
		"unit":       product.Unit,
		"data":       data,
	})
}
