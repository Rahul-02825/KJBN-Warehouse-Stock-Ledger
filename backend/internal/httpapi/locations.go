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

type createLocationRequest struct {
	Code        string  `json:"code"`
	Description string  `json:"description"`
	WarehouseID int64   `json:"warehouse_id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Capacity    *int64  `json:"capacity"`
	AutoPlace   *bool   `json:"auto_place"`
}

type locationJSON struct {
	ID          int64            `json:"id"`
	Code        string           `json:"code"`
	Description string           `json:"description"`
	WarehouseID int64            `json:"warehouse_id"`
	X           float64          `json:"x"`
	Y           float64          `json:"y"`
	Capacity    *int64           `json:"capacity"`
	AutoPlace   bool             `json:"auto_place"`
	Used        *decimal.Decimal `json:"used,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

func toLocationJSON(l sqlc.Location) locationJSON {
	return locationJSON{
		ID:          l.ID,
		Code:        l.Code,
		Description: l.Description,
		WarehouseID: l.WarehouseID,
		X:           l.X,
		Y:           l.Y,
		Capacity:    l.Capacity,
		AutoPlace:   l.AutoPlace,
		CreatedAt:   l.CreatedAt.UTC(),
	}
}

func (req *createLocationRequest) validate() error {
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
		return domain.ValidationError(fields)
	}
	return nil
}

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request) {
	var req createLocationRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, r, err)
		return
	}

	warehouse, err := s.findWarehouse(r, req.WarehouseID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	site := domain.Site{Center: domain.Point{X: warehouse.X, Y: warehouse.Y}, RadiusM: warehouse.RadiusM, Capacity: warehouse.Capacity}
	if fields := domain.CheckBinLayout(site, domain.Point{X: req.X, Y: req.Y}, req.Capacity); len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}

	autoPlace := true
	if req.AutoPlace != nil {
		autoPlace = *req.AutoPlace
	}
	location, err := s.q.CreateLocation(r.Context(), sqlc.CreateLocationParams{
		Code:        req.Code,
		Description: req.Description,
		WarehouseID: warehouse.ID,
		X:           req.X,
		Y:           req.Y,
		Capacity:    req.Capacity,
		AutoPlace:   autoPlace,
	})
	if postgres.IsUniqueViolation(err) {
		writeErrorBody(w, http.StatusConflict, errorBody{Code: "DUPLICATE", Message: "a location with this code already exists"})
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toLocationJSON(location))
}

func (s *Server) findWarehouse(r *http.Request, id int64) (sqlc.Warehouse, error) {
	var warehouse sqlc.Warehouse
	var err error
	if id == 0 {
		warehouse, err = s.q.GetDefaultWarehouse(r.Context())
	} else {
		warehouse, err = s.q.GetWarehouseByID(r.Context(), id)
	}
	if postgres.IsNotFound(err) {
		return sqlc.Warehouse{}, domain.NotFoundError{What: "warehouse"}
	}
	return warehouse, err
}

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.q.ListLocations(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := []locationJSON{}
	for _, row := range rows {
		used := row.Used
		data = append(data, locationJSON{
			ID:          row.ID,
			Code:        row.Code,
			Description: row.Description,
			WarehouseID: row.WarehouseID,
			X:           row.X,
			Y:           row.Y,
			Capacity:    row.Capacity,
			AutoPlace:   row.AutoPlace,
			Used:        &used,
			CreatedAt:   row.CreatedAt.UTC(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) getLocation(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, toLocationJSON(location))
}

func (s *Server) findLocation(r *http.Request, id int64) (sqlc.Location, error) {
	location, err := s.q.GetLocationByID(r.Context(), id)
	if postgres.IsNotFound(err) {
		return sqlc.Location{}, domain.NotFoundError{What: "location"}
	}
	return location, err
}
