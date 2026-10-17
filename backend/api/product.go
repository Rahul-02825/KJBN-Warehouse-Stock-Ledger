package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type ProductHandler struct {
	q *sqlc.Queries
}

func NewProductHandler(q *sqlc.Queries) *ProductHandler {
	return &ProductHandler{q: q}
}

func (h *ProductHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/products", h.create)
	mux.HandleFunc("GET /api/products", h.list)
	mux.HandleFunc("GET /api/products/{id}", h.get)
	mux.HandleFunc("PATCH /api/products/{id}", h.update)
}

type createProductRequest struct {
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	ReorderLevel decimal.Decimal `json:"reorder_level"`
}

type updateProductRequest struct {
	Name         *string          `json:"name"`
	ReorderLevel *decimal.Decimal `json:"reorder_level"`
}

type productResponse struct {
	ID           int64           `json:"id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	ReorderLevel decimal.Decimal `json:"reorder_level"`
	AverageCost  decimal.Decimal `json:"average_cost"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func toProductResponse(p sqlc.Product) productResponse {
	return productResponse{
		ID:           p.ID,
		SKU:          p.Sku,
		Name:         p.Name,
		Unit:         p.Unit,
		ReorderLevel: toDecimal(p.ReorderLevel),
		AverageCost:  toDecimal(p.AverageCost),
		CreatedAt:    p.CreatedAt.Time.UTC(),
		UpdatedAt:    p.UpdatedAt.Time.UTC(),
	}
}

func (h *ProductHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.SKU = strings.TrimSpace(req.SKU)
	req.Name = strings.TrimSpace(req.Name)

	fields := map[string]string{}
	if req.SKU == "" || len(req.SKU) > 64 {
		fields["sku"] = "is required, max 64 characters"
	}
	if req.Name == "" || len(req.Name) > 200 {
		fields["name"] = "is required, max 200 characters"
	}
	if !ValidUnit(req.Unit) {
		fields["unit"] = "must be one of pcs, kg, litre"
	}
	if msg := CheckReorderLevel(req.ReorderLevel, req.Unit); msg != "" {
		fields["reorder_level"] = msg
	}
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	product, err := h.q.CreateProduct(r.Context(), sqlc.CreateProductParams{
		Sku:          req.SKU,
		Name:         req.Name,
		Unit:         req.Unit,
		ReorderLevel: toNumeric(req.ReorderLevel),
	})
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, "DUPLICATE", "a product with this sku already exists")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProductResponse(product))
}

func (h *ProductHandler) get(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) list(w http.ResponseWriter, r *http.Request) {
	page, size := pageParams(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))

	searchText := strings.TrimSpace(r.URL.Query().Get("search"))
	search := pgtype.Text{String: searchText, Valid: searchText != ""}

	rows, err := h.q.ListProducts(r.Context(), sqlc.ListProductsParams{
		Search:     search,
		PageSize:   int32(size),
		PageOffset: int32((page - 1) * size),
	})
	if err != nil {
		writeInternalError(w, err)
		return
	}
	total, err := h.q.CountProducts(r.Context(), search)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	data := make([]productResponse, len(rows))
	for i, p := range rows {
		data[i] = toProductResponse(p)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":      data,
		"page":      page,
		"page_size": size,
		"total":     total,
	})
}

// update changes only name and reorder_level; sku and unit are fixed once created.
func (h *ProductHandler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var req updateProductRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == nil && req.ReorderLevel == nil {
		writeValidationError(w, map[string]string{"body": "send name and/or reorder_level"})
		return
	}

	current, err := h.q.GetProductByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "product not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}

	params := sqlc.UpdateProductDetailsParams{ID: id}
	fields := map[string]string{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 200 {
			fields["name"] = "must not be blank, max 200 characters"
		}
		params.Name = pgtype.Text{String: name, Valid: true}
	}
	if req.ReorderLevel != nil {
		if msg := CheckReorderLevel(*req.ReorderLevel, current.Unit); msg != "" {
			fields["reorder_level"] = msg
		}
		params.ReorderLevel = toNumeric(*req.ReorderLevel)
	}
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	product, err := h.q.UpdateProductDetails(r.Context(), params)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(product))
}

func ValidUnit(u string) bool {
	return u == "pcs" || u == "kg" || u == "litre"
}

func CheckReorderLevel(d decimal.Decimal, unit string) string {
	if d.IsNegative() {
		return "must not be negative"
	}
	if !d.Equal(d.Round(4)) {
		return "must have at most 4 decimal places"
	}
	if unit == "pcs" && !d.Equal(d.Truncate(0)) {
		return "must be a whole number for pcs"
	}
	return ""
}

func readID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	return readPathID(w, r, "id")
}

func readPathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", name+" must be a positive integer")
		return 0, false
	}
	return id, true
}
