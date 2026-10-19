package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type LocationHandler struct {
	q *sqlc.Queries
}

func NewLocationHandler(q *sqlc.Queries) *LocationHandler {
	return &LocationHandler{q: q}
}

func (h *LocationHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/locations", h.create)
	mux.HandleFunc("GET /api/locations", h.list)
	mux.HandleFunc("GET /api/locations/{id}", h.get)
}

type createLocationRequest struct {
	Code        string  `json:"code"`
	Description string  `json:"description"`
	WarehouseID int64   `json:"warehouse_id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Capacity    *int64  `json:"capacity"`
	AutoPlace   *bool   `json:"auto_place"`
}

type locationResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	WarehouseID int64     `json:"warehouse_id"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Capacity    *int64    `json:"capacity"`
	AutoPlace   bool      `json:"auto_place"`
	Used        *int64    `json:"used,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func toLocationResponse(l sqlc.Location) locationResponse {
	return locationResponse{
		ID:          l.ID,
		Code:        l.Code,
		Description: l.Description,
		WarehouseID: l.WarehouseID,
		X:           l.X,
		Y:           l.Y,
		Capacity:    idPtr(l.Capacity),
		AutoPlace:   l.AutoPlace,
		CreatedAt:   l.CreatedAt.Time.UTC(),
	}
}

func (h *LocationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createLocationRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.Description = strings.TrimSpace(req.Description)

	fields := map[string]string{}
	if req.Code == "" || len(req.Code) > 64 {
		fields["code"] = "is required, max 64 characters"
	} else if strings.ContainsAny(req.Code, " \t\n") {
		fields["code"] = "must not contain spaces"
	}
	if len(req.Description) > 500 {
		fields["description"] = "max 500 characters"
	}
	if req.WarehouseID < 0 {
		fields["warehouse_id"] = "must be a positive integer"
	}
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	warehouse, err := h.warehouseFor(r, req.WarehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "warehouse not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	if !valid(w, CheckBinLayout(warehouse, Point{req.X, req.Y}, req.Capacity)) {
		return
	}

	location, err := h.q.CreateLocation(r.Context(), sqlc.CreateLocationParams{
		Code:        req.Code,
		Description: req.Description,
		WarehouseID: warehouse.ID,
		X:           req.X,
		Y:           req.Y,
		Capacity:    optCapacity(req.Capacity),
		AutoPlace:   req.AutoPlace == nil || *req.AutoPlace,
	})
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "DUPLICATE", "a location with this code already exists")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toLocationResponse(location))
}

func (h *LocationHandler) warehouseFor(r *http.Request, id int64) (sqlc.Warehouse, error) {
	if id == 0 {
		return h.q.GetDefaultWarehouse(r.Context())
	}
	return h.q.GetWarehouseByID(r.Context(), id)
}

func (h *LocationHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListLocations(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]locationResponse, len(rows))
	for i, row := range rows {
		data[i] = toLocationResponse(sqlc.Location{
			ID: row.ID, Code: row.Code, Description: row.Description, CreatedAt: row.CreatedAt,
			WarehouseID: row.WarehouseID, X: row.X, Y: row.Y, Capacity: row.Capacity, AutoPlace: row.AutoPlace,
		})
		data[i].Used = &row.Used
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *LocationHandler) get(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, toLocationResponse(location))
}

func CheckBinLayout(warehouse sqlc.Warehouse, position Point, capacity *int64) map[string]string {
	fields := map[string]string{}
	if Distance(position, Point{warehouse.X, warehouse.Y}) > warehouse.RadiusM {
		fields["x"] = "bin must be within the warehouse perimeter"
	}
	if capacity != nil {
		switch {
		case *capacity <= 0:
			fields["capacity"] = "must be greater than zero, or left empty for no limit"
		case warehouse.Capacity.Valid && *capacity > warehouse.Capacity.Int64:
			fields["capacity"] = "must not exceed the warehouse capacity"
		}
	}
	return fields
}

func optCapacity(capacity *int64) pgtype.Int8 {
	if capacity == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *capacity, Valid: true}
}
