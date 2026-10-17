package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

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
	Code        string `json:"code"`
	Description string `json:"description"`
}

type locationResponse struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func toLocationResponse(l sqlc.Location) locationResponse {
	return locationResponse{
		ID:          l.ID,
		Code:        l.Code,
		Description: l.Description,
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
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	location, err := h.q.CreateLocation(r.Context(), sqlc.CreateLocationParams{
		Code:        req.Code,
		Description: req.Description,
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

// list is not paginated; a warehouse has a small, bounded number of bins.
func (h *LocationHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListLocations(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]locationResponse, len(rows))
	for i, l := range rows {
		data[i] = toLocationResponse(l)
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
