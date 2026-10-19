package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

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

type productJSON struct {
	ID           int64           `json:"id"`
	SKU          string          `json:"sku"`
	Name         string          `json:"name"`
	Unit         string          `json:"unit"`
	ReorderLevel decimal.Decimal `json:"reorder_level"`
	AverageCost  decimal.Decimal `json:"average_cost"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func toProductJSON(p sqlc.Product) productJSON {
	return productJSON{
		ID:           p.ID,
		SKU:          p.Sku,
		Name:         p.Name,
		Unit:         p.Unit,
		ReorderLevel: p.ReorderLevel,
		AverageCost:  p.AverageCost,
		CreatedAt:    p.CreatedAt.UTC(),
		UpdatedAt:    p.UpdatedAt.UTC(),
	}
}

func (req *createProductRequest) validate() error {
	req.SKU = strings.TrimSpace(req.SKU)
	req.Name = strings.TrimSpace(req.Name)
	fields := map[string]string{}
	if req.SKU == "" || len(req.SKU) > 64 {
		fields["sku"] = "is required, max 64 characters"
	}
	if req.Name == "" || len(req.Name) > 200 {
		fields["name"] = "is required, max 200 characters"
	}
	if !domain.ValidUnit(req.Unit) {
		fields["unit"] = domain.UnitsMessage()
	}
	if msg := domain.CheckReorderLevel(req.ReorderLevel, req.Unit); msg != "" {
		fields["reorder_level"] = msg
	}
	if len(fields) > 0 {
		return domain.ValidationError(fields)
	}
	return nil
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, r, err)
		return
	}

	product, err := s.q.CreateProduct(r.Context(), sqlc.CreateProductParams{
		Sku:          req.SKU,
		Name:         req.Name,
		Unit:         req.Unit,
		ReorderLevel: req.ReorderLevel,
	})
	if postgres.IsUniqueViolation(err) {
		writeErrorBody(w, http.StatusConflict, errorBody{Code: "DUPLICATE", Message: "a product with this sku already exists"})
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProductJSON(product))
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	product, err := s.findProduct(r, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductJSON(product))
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, size := pageParams(query)
	search := nilIfEmpty(strings.TrimSpace(query.Get("search")))

	rows, err := s.q.ListProducts(r.Context(), sqlc.ListProductsParams{
		Search:     search,
		PageSize:   int32(size),
		PageOffset: int32((page - 1) * size),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	total, err := s.q.CountProducts(r.Context(), search)
	if err != nil {
		writeError(w, r, err)
		return
	}

	data := []productJSON{}
	for _, p := range rows {
		data = append(data, toProductJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": page, "page_size": size, "total": total})
}

func (s *Server) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req updateProductRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if req.Name == nil && req.ReorderLevel == nil {
		writeError(w, r, domain.ValidationError{"body": "send name and/or reorder_level"})
		return
	}
	current, err := s.findProduct(r, id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	params := sqlc.UpdateProductDetailsParams{ID: id}
	fields := map[string]string{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 200 {
			fields["name"] = "must not be blank, max 200 characters"
		}
		params.Name = &name
	}
	if req.ReorderLevel != nil {
		if msg := domain.CheckReorderLevel(*req.ReorderLevel, current.Unit); msg != "" {
			fields["reorder_level"] = msg
		}
		params.ReorderLevel = req.ReorderLevel
	}
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}

	product, err := s.q.UpdateProductDetails(r.Context(), params)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProductJSON(product))
}

func (s *Server) findProduct(r *http.Request, id int64) (sqlc.Product, error) {
	product, err := s.q.GetProductByID(r.Context(), id)
	if postgres.IsNotFound(err) {
		return sqlc.Product{}, domain.NotFoundError{What: "product"}
	}
	return product, err
}
