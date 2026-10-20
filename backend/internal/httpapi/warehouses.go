package httpapi

import (
	"math"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/ledger"
)

type warehouseJSON struct {
	ID       int64           `json:"id"`
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	X        float64         `json:"x"`
	Y        float64         `json:"y"`
	RadiusM  float64         `json:"radius_m"`
	Capacity *int64          `json:"capacity"`
	Used     decimal.Decimal `json:"used"`
}

type allocationJSON struct {
	LocationID int64           `json:"location_id"`
	Code       string          `json:"location_code"`
	Quantity   decimal.Decimal `json:"quantity"`
	DistanceM  float64         `json:"distance_m"`
}

func (s *Server) listWarehouses(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListWarehouses(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := []warehouseJSON{}
	for _, row := range rows {
		data = append(data, warehouseJSON{
			ID:       row.ID,
			Code:     row.Code,
			Name:     row.Name,
			X:        row.X,
			Y:        row.Y,
			RadiusM:  row.RadiusM,
			Capacity: row.Capacity,
			Used:     row.Used,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) previewPlacement(w http.ResponseWriter, r *http.Request) {
	var req ledger.PreviewRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	preview, err := s.ledger.PreviewPlacement(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPreviewJSON(preview))
}

func (s *Server) previewPick(w http.ResponseWriter, r *http.Request) {
	var req ledger.PreviewRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	preview, err := s.ledger.PreviewPick(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPreviewJSON(preview))
}

func toPreviewJSON(preview ledger.Preview) map[string]any {
	allocations := []allocationJSON{}
	for _, a := range preview.Plan.Allocations {
		allocations = append(allocations, allocationJSON{
			LocationID: a.Bin.ID,
			Code:       a.Bin.Code,
			Quantity:   domain.FromUnits(a.Quantity),
			DistanceM:  math.Round(a.Distance*100) / 100,
		})
	}
	return map[string]any{
		"warehouse_id":        preview.Plan.Warehouse.ID,
		"unit":                preview.Product.Unit,
		"start_location_id":   preview.Plan.Start.ID,
		"start_location_code": preview.Plan.Start.Code,
		"reason":              preview.Plan.Reason,
		"allocations":         allocations,
	}
}
