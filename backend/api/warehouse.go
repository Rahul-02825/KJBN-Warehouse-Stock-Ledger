package api

import (
	"errors"
	"math"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type WarehouseHandler struct {
	q *sqlc.Queries
}

func NewWarehouseHandler(q *sqlc.Queries) *WarehouseHandler {
	return &WarehouseHandler{q: q}
}

func (h *WarehouseHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/warehouses", h.list)
	mux.HandleFunc("POST /api/placements/preview", h.preview)
}

type warehouseResponse struct {
	ID       int64           `json:"id"`
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	X        float64         `json:"x"`
	Y        float64         `json:"y"`
	RadiusM  float64         `json:"radius_m"`
	Capacity *int64          `json:"capacity"`
	Used     decimal.Decimal `json:"used"`
}

func (h *WarehouseHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListWarehouses(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]warehouseResponse, len(rows))
	for i, row := range rows {
		data[i] = warehouseResponse{
			ID: row.ID, Code: row.Code, Name: row.Name, X: row.X, Y: row.Y,
			RadiusM: row.RadiusM, Capacity: idPtr(row.Capacity), Used: toDecimal(row.Used),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

type PreviewRequest struct {
	ProductID   int64           `json:"product_id"`
	LocationID  int64           `json:"location_id"`
	WarehouseID int64           `json:"warehouse_id"`
	Quantity    decimal.Decimal `json:"quantity"`
}

func (r *PreviewRequest) validate() map[string]string {
	f := map[string]string{}
	checkIDs(f, map[string]int64{"product_id": r.ProductID})
	checkOptionalIDs(f, map[string]int64{"location_id": r.LocationID, "warehouse_id": r.WarehouseID})
	checkAmount(f, "quantity", r.Quantity)
	return f
}

type allocationResponse struct {
	LocationID int64           `json:"location_id"`
	Code       string          `json:"location_code"`
	Quantity   decimal.Decimal `json:"quantity"`
	DistanceM  float64         `json:"distance_m"`
}

func (h *WarehouseHandler) preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	product, err := h.q.GetProductByID(r.Context(), req.ProductID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	if product.Unit == "pcs" && !req.Quantity.Equal(req.Quantity.Truncate(0)) {
		writeValidationError(w, map[string]string{"quantity": "must be a whole number for pcs"})
		return
	}

	warehouse, err := resolveWarehouse(r.Context(), h.q, req.LocationID, req.WarehouseID, false)
	if err != nil {
		writePostingError(w, r, err)
		return
	}
	plan, err := planReceipt(r.Context(), h.q, warehouse, product, req.Quantity, req.LocationID)
	if err != nil {
		writePostingError(w, r, err)
		return
	}

	allocations := make([]allocationResponse, len(plan.Allocations))
	for i, a := range plan.Allocations {
		allocations[i] = allocationResponse{
			LocationID: a.Bin.ID,
			Code:       a.Bin.Code,
			Quantity:   fromUnits(a.Quantity),
			DistanceM:  math.Round(a.Distance*100) / 100,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"warehouse_id":        warehouse.ID,
		"unit":                product.Unit,
		"start_location_id":   plan.Start.ID,
		"start_location_code": plan.Start.Code,
		"reason":              plan.Reason,
		"allocations":         allocations,
	})
}
