package httpapi

import (
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type stockCellJSON struct {
	ProductID    int64           `json:"product_id"`
	SKU          string          `json:"sku"`
	ProductName  string          `json:"product_name"`
	Unit         string          `json:"unit"`
	LocationID   int64           `json:"location_id"`
	LocationCode string          `json:"location_code"`
	Quantity     decimal.Decimal `json:"quantity"`
	AverageCost  decimal.Decimal `json:"average_cost"`
}

func (s *Server) stockGrid(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListStockOverview(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]stockCellJSON, len(rows))
	for i, c := range rows {
		data[i] = stockCellJSON{
			ProductID:    c.ProductID,
			SKU:          c.Sku,
			ProductName:  c.ProductName,
			Unit:         c.Unit,
			LocationID:   c.LocationID,
			LocationCode: c.LocationCode,
			Quantity:     c.Quantity,
			AverageCost:  c.AverageCost,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) stockCell(w http.ResponseWriter, r *http.Request) {
	productID, err := readID(r, "productId")
	if err != nil {
		writeError(w, r, err)
		return
	}
	locationID, err := readID(r, "locationId")
	if err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.q.GetStockBalance(r.Context(), sqlc.GetStockBalanceParams{ProductID: productID, LocationID: locationID})
	if postgres.IsNotFound(err) {
		writeError(w, r, domain.NotFoundError{What: "product or location"})
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stockCellJSON{
		ProductID:    c.ProductID,
		SKU:          c.Sku,
		ProductName:  c.ProductName,
		Unit:         c.Unit,
		LocationID:   c.LocationID,
		LocationCode: c.LocationCode,
		Quantity:     c.Quantity,
		AverageCost:  c.AverageCost,
	})
}

type productBalanceJSON struct {
	LocationID   int64           `json:"location_id"`
	LocationCode string          `json:"location_code"`
	Quantity     decimal.Decimal `json:"quantity"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (s *Server) productBalances(w http.ResponseWriter, r *http.Request) {
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
	rows, err := s.q.ListProductBalances(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	total := decimal.Zero
	data := make([]productBalanceJSON, len(rows))
	for i, b := range rows {
		qty := b.Quantity
		total = total.Add(qty)
		data[i] = productBalanceJSON{
			LocationID:   b.LocationID,
			LocationCode: b.LocationCode,
			Quantity:     qty,
			UpdatedAt:    b.UpdatedAt.UTC(),
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

type locationBalanceJSON struct {
	ProductID   int64           `json:"product_id"`
	SKU         string          `json:"sku"`
	ProductName string          `json:"product_name"`
	Unit        string          `json:"unit"`
	Quantity    decimal.Decimal `json:"quantity"`
	AverageCost decimal.Decimal `json:"average_cost"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (s *Server) locationBalances(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	location, err := s.findLocation(r, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	rows, err := s.q.ListLocationBalances(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	data := make([]locationBalanceJSON, len(rows))
	for i, b := range rows {
		data[i] = locationBalanceJSON{
			ProductID:   b.ProductID,
			SKU:         b.Sku,
			ProductName: b.ProductName,
			Unit:        b.Unit,
			Quantity:    b.Quantity,
			AverageCost: b.AverageCost,
			UpdatedAt:   b.UpdatedAt.UTC(),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location_id":   location.ID,
		"location_code": location.Code,
		"data":          data,
	})
}

type stockAsOfJSON struct {
	ProductID    int64           `json:"product_id"`
	SKU          string          `json:"sku"`
	ProductName  string          `json:"product_name"`
	Unit         string          `json:"unit"`
	LocationID   int64           `json:"location_id"`
	LocationCode string          `json:"location_code"`
	Quantity     decimal.Decimal `json:"quantity"`
}

func (s *Server) stockAsOf(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fields := map[string]string{}
	at := queryTime(fields, query, "date", true)
	if query.Get("date") == "" {
		fields["date"] = "is required"
	}
	productID := queryID(fields, query, "product_id")
	locationID := queryID(fields, query, "location_id")
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}

	rows, err := s.q.ListStockAsOf(r.Context(), sqlc.ListStockAsOfParams{
		AsOf: *at, ProductID: productID, LocationID: locationID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]stockAsOfJSON, len(rows))
	for i, c := range rows {
		data[i] = stockAsOfJSON{
			ProductID:    c.ProductID,
			SKU:          c.Sku,
			ProductName:  c.ProductName,
			Unit:         c.Unit,
			LocationID:   c.LocationID,
			LocationCode: c.LocationCode,
			Quantity:     c.Quantity,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"as_of": at.UTC(), "data": data})
}
