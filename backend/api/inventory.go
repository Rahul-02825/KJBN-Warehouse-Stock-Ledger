package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

// Balances are read from the stock_balances cache. The ledger stays the source of truth;
// the reconciliation report compares the two.

type InventoryHandler struct {
	q *sqlc.Queries
}

func NewInventoryHandler(q *sqlc.Queries) *InventoryHandler {
	return &InventoryHandler{q: q}
}

func (h *InventoryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/inventory", h.grid)
	mux.HandleFunc("GET /api/inventory/{productId}/{locationId}", h.cell)
	mux.HandleFunc("GET /api/products/{id}/balances", h.byProduct)
	mux.HandleFunc("GET /api/locations/{id}/balances", h.byLocation)
}

type cellResponse struct {
	ProductID    int64           `json:"product_id"`
	SKU          string          `json:"sku"`
	ProductName  string          `json:"product_name"`
	Unit         string          `json:"unit"`
	LocationID   int64           `json:"location_id"`
	LocationCode string          `json:"location_code"`
	Quantity     decimal.Decimal `json:"quantity"`
	AverageCost  decimal.Decimal `json:"average_cost"`
}

// grid returns one row per product x location, with zero where nothing is stored.
func (h *InventoryHandler) grid(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListStockOverview(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]cellResponse, len(rows))
	for i, c := range rows {
		data[i] = cellResponse{
			ProductID:    c.ProductID,
			SKU:          c.Sku,
			ProductName:  c.ProductName,
			Unit:         c.Unit,
			LocationID:   c.LocationID,
			LocationCode: c.LocationCode,
			Quantity:     toDecimal(c.Quantity),
			AverageCost:  toDecimal(c.AverageCost),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *InventoryHandler) cell(w http.ResponseWriter, r *http.Request) {
	productID, ok := readPathID(w, r, "productId")
	if !ok {
		return
	}
	locationID, ok := readPathID(w, r, "locationId")
	if !ok {
		return
	}
	c, err := h.q.GetStockBalance(r.Context(), sqlc.GetStockBalanceParams{ID: productID, ID_2: locationID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "product or location not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cellResponse{
		ProductID:    c.ProductID,
		SKU:          c.Sku,
		ProductName:  c.ProductName,
		Unit:         c.Unit,
		LocationID:   c.LocationID,
		LocationCode: c.LocationCode,
		Quantity:     toDecimal(c.Quantity),
		AverageCost:  toDecimal(c.AverageCost),
	})
}

type productBalance struct {
	LocationID   int64           `json:"location_id"`
	LocationCode string          `json:"location_code"`
	Quantity     decimal.Decimal `json:"quantity"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (h *InventoryHandler) byProduct(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.q.ListProductBalances(r.Context(), id)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	total := decimal.Zero
	data := make([]productBalance, len(rows))
	for i, b := range rows {
		qty := toDecimal(b.Quantity)
		total = total.Add(qty)
		data[i] = productBalance{
			LocationID:   b.LocationID,
			LocationCode: b.LocationCode,
			Quantity:     qty,
			UpdatedAt:    b.UpdatedAt.Time.UTC(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id":     product.ID,
		"sku":            product.Sku,
		"unit":           product.Unit,
		"total_quantity": total,
		"data":           data,
	})
}

type locationBalance struct {
	ProductID   int64           `json:"product_id"`
	SKU         string          `json:"sku"`
	ProductName string          `json:"product_name"`
	Unit        string          `json:"unit"`
	Quantity    decimal.Decimal `json:"quantity"`
	AverageCost decimal.Decimal `json:"average_cost"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (h *InventoryHandler) byLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	location, err := h.q.GetLocationByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "location not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	rows, err := h.q.ListLocationBalances(r.Context(), id)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	data := make([]locationBalance, len(rows))
	for i, b := range rows {
		data[i] = locationBalance{
			ProductID:   b.ProductID,
			SKU:         b.Sku,
			ProductName: b.ProductName,
			Unit:        b.Unit,
			Quantity:    toDecimal(b.Quantity),
			AverageCost: toDecimal(b.AverageCost),
			UpdatedAt:   b.UpdatedAt.Time.UTC(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location_id":   location.ID,
		"location_code": location.Code,
		"data":          data,
	})
}
