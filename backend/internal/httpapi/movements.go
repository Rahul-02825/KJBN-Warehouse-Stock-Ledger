package httpapi

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/ledger"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type movementJSON struct {
	ID                 int64            `json:"id"`
	Type               string           `json:"type"`
	ProductID          int64            `json:"product_id"`
	FromLocationID     *int64           `json:"from_location_id"`
	ToLocationID       *int64           `json:"to_location_id"`
	Quantity           decimal.Decimal  `json:"quantity"`
	UnitCost           *decimal.Decimal `json:"unit_cost"`
	TotalValue         *decimal.Decimal `json:"total_value"`
	Reference          string           `json:"reference"`
	Reason             *string          `json:"reason"`
	Note               string           `json:"note"`
	CreatedBy          string           `json:"created_by"`
	ReversesMovementID *int64           `json:"reverses_movement_id"`
	CreatedAt          time.Time        `json:"created_at"`
}

type movementDetailJSON struct {
	ID                 int64            `json:"id"`
	Type               string           `json:"type"`
	ProductID          int64            `json:"product_id"`
	FromLocationID     *int64           `json:"from_location_id"`
	ToLocationID       *int64           `json:"to_location_id"`
	Quantity           decimal.Decimal  `json:"quantity"`
	UnitCost           *decimal.Decimal `json:"unit_cost"`
	TotalValue         *decimal.Decimal `json:"total_value"`
	Reference          string           `json:"reference"`
	Reason             *string          `json:"reason"`
	Note               string           `json:"note"`
	CreatedBy          string           `json:"created_by"`
	ReversesMovementID *int64           `json:"reverses_movement_id"`
	CreatedAt          time.Time        `json:"created_at"`
	SKU                string           `json:"sku"`
	ProductName        string           `json:"product_name"`
	Unit               string           `json:"unit"`
	FromLocationCode   *string          `json:"from_location_code"`
	ToLocationCode     *string          `json:"to_location_code"`
	IsReversed         bool             `json:"is_reversed"`
	ReversedByID       *int64           `json:"reversed_by_id"`
	Entries            []entryJSON      `json:"entries,omitempty"`
}

type entryJSON struct {
	LocationID    int64            `json:"location_id"`
	QuantityDelta decimal.Decimal  `json:"quantity_delta"`
	UnitCost      *decimal.Decimal `json:"unit_cost"`
}

func totalValue(quantity decimal.Decimal, unitCost *decimal.Decimal) *decimal.Decimal {
	if unitCost == nil {
		return nil
	}
	total := quantity.Mul(*unitCost).Round(domain.Decimals)
	return &total
}

func toMovementJSON(m sqlc.Movement) movementJSON {
	return movementJSON{
		ID:                 m.ID,
		Type:               m.MovementType,
		ProductID:          m.ProductID,
		FromLocationID:     m.FromLocationID,
		ToLocationID:       m.ToLocationID,
		Quantity:           m.Quantity,
		UnitCost:           m.UnitCost,
		TotalValue:         totalValue(m.Quantity, m.UnitCost),
		Reference:          m.Reference,
		Reason:             m.Reason,
		Note:               m.Note,
		CreatedBy:          m.CreatedBy,
		ReversesMovementID: m.ReversalOfID,
		CreatedAt:          m.CreatedAt.UTC(),
	}
}

func toMovementDetailJSON(d sqlc.GetMovementDetailRow) movementDetailJSON {
	return movementDetailJSON{
		ID:                 d.ID,
		Type:               d.MovementType,
		ProductID:          d.ProductID,
		FromLocationID:     d.FromLocationID,
		ToLocationID:       d.ToLocationID,
		Quantity:           d.Quantity,
		UnitCost:           d.UnitCost,
		TotalValue:         totalValue(d.Quantity, d.UnitCost),
		Reference:          d.Reference,
		Reason:             d.Reason,
		Note:               d.Note,
		CreatedBy:          d.CreatedBy,
		ReversesMovementID: d.ReversalOfID,
		CreatedAt:          d.CreatedAt.UTC(),
		SKU:                d.Sku,
		ProductName:        d.ProductName,
		Unit:               d.Unit,
		FromLocationCode:   d.FromLocationCode,
		ToLocationCode:     d.ToLocationCode,
		IsReversed:         d.IsReversed,
		ReversedByID:       d.ReversedByID,
	}
}

func writeMovement(w http.ResponseWriter, r *http.Request, movement sqlc.Movement, replayed bool, err error, fields ...any) {
	if err != nil {
		writeError(w, r, err, fields...)
		return
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
		writeJSON(w, http.StatusOK, toMovementJSON(movement))
		return
	}
	slog.Info("movement posted", "request_id", requestID(r), "movement_id", movement.ID, "type", movement.MovementType, "product_id", movement.ProductID)
	writeJSON(w, http.StatusCreated, toMovementJSON(movement))
}

func (s *Server) receiveStock(w http.ResponseWriter, r *http.Request) {
	key, err := readKey(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req ledger.ReceiveRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Receive(r.Context(), req)
	writeMovement(w, r, movement, replayed, err,
		"product_id", req.ProductID, "location_id", req.LocationID, "quantity", req.Quantity)
}

func (s *Server) shipStock(w http.ResponseWriter, r *http.Request) {
	key, err := readKey(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req ledger.ShipRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Ship(r.Context(), req)
	writeMovement(w, r, movement, replayed, err,
		"product_id", req.ProductID, "location_id", req.LocationID, "quantity", req.Quantity)
}

func (s *Server) transferStock(w http.ResponseWriter, r *http.Request) {
	key, err := readKey(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req ledger.TransferRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Transfer(r.Context(), req)
	writeMovement(w, r, movement, replayed, err,
		"product_id", req.ProductID, "from_location_id", req.FromLocationID, "to_location_id", req.ToLocationID, "quantity", req.Quantity)
}

func (s *Server) adjustStock(w http.ResponseWriter, r *http.Request) {
	key, err := readKey(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req ledger.AdjustRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Adjust(r.Context(), req)
	writeMovement(w, r, movement, replayed, err,
		"product_id", req.ProductID, "location_id", req.LocationID, "quantity", req.Quantity, "reason", req.Reason)
}

func (s *Server) reverseMovement(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	key, err := readKey(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req ledger.ReverseRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Reverse(r.Context(), id, req)
	writeMovement(w, r, movement, replayed, err, "movement_id", id)
}

func (s *Server) stockTake(w http.ResponseWriter, r *http.Request) {
	var req ledger.StockTakeRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.ledger.StockTake(r.Context(), req)
	if err != nil {
		writeError(w, r, err, "location_id", req.LocationID, "lines", len(req.Counts))
		return
	}
	adjustments := []movementJSON{}
	for _, m := range result.Adjustments {
		adjustments = append(adjustments, toMovementJSON(m))
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"location_id": req.LocationID,
		"adjustments": adjustments,
		"unchanged":   result.Unchanged,
	})
}

func (s *Server) getMovement(w http.ResponseWriter, r *http.Request) {
	id, err := readID(r, "id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	detail, err := s.q.GetMovementDetail(r.Context(), id)
	if postgres.IsNotFound(err) {
		writeError(w, r, domain.NotFoundError{What: "movement"})
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	entries, err := s.q.ListMovementEntries(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	movement := toMovementDetailJSON(detail)
	for _, e := range entries {
		movement.Entries = append(movement.Entries, entryJSON{
			LocationID:    e.LocationID,
			QuantityDelta: e.QuantityDelta,
			UnitCost:      e.UnitCost,
		})
	}
	writeJSON(w, http.StatusOK, movement)
}

func (s *Server) listMovements(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fields := map[string]string{}
	filter := sqlc.CountMovementHistoryParams{
		ProductID:  queryID(fields, query, "product_id"),
		LocationID: queryID(fields, query, "location_id"),
		FromDate:   queryTime(fields, query, "from", false),
		ToDate:     queryTime(fields, query, "to", true),
	}
	movementType := strings.ToUpper(query.Get("type"))
	if movementType != "" {
		if domain.ValidMovementType(movementType) {
			filter.MovementType = &movementType
		} else {
			fields["type"] = domain.MovementTypesMessage()
		}
	}
	filter.Reference = nilIfEmpty(strings.TrimSpace(query.Get("reference")))
	if len(fields) > 0 {
		writeError(w, r, domain.ValidationError(fields))
		return
	}

	page, size := pageParams(query)
	asCSV := query.Get("format") == "csv"
	if asCSV {
		page, size = 1, csvMaxRows
	}
	rows, err := s.q.ListMovementHistory(r.Context(), sqlc.ListMovementHistoryParams{
		ProductID:    filter.ProductID,
		MovementType: filter.MovementType,
		LocationID:   filter.LocationID,
		Reference:    filter.Reference,
		FromDate:     filter.FromDate,
		ToDate:       filter.ToDate,
		PageSize:     int32(size),
		PageOffset:   int32((page - 1) * size),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	data := []movementDetailJSON{}
	for _, row := range rows {
		data = append(data, toMovementDetailJSON(sqlc.GetMovementDetailRow(row)))
	}
	if asCSV {
		writeMovementsCSV(w, data)
		return
	}

	total, err := s.q.CountMovementHistory(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": page, "page_size": size, "total": total})
}

func writeMovementsCSV(w http.ResponseWriter, data []movementDetailJSON) {
	rows := [][]string{}
	for _, m := range data {
		rows = append(rows, []string{
			strconv.FormatInt(m.ID, 10), m.CreatedAt.Format(time.RFC3339), m.Type,
			m.SKU, csvText(m.ProductName), m.Unit, csvPtr(m.FromLocationCode), csvPtr(m.ToLocationCode),
			m.Quantity.String(), csvDecimal(m.UnitCost), csvDecimal(m.TotalValue),
			csvText(m.Reference), csvPtr(m.Reason), csvText(m.Note), csvText(m.CreatedBy),
			csvID(m.ReversesMovementID), csvID(m.ReversedByID),
		})
	}
	writeCSV(w, "movements.csv", []string{
		"id", "created_at", "type", "sku", "product_name", "unit", "from_location", "to_location",
		"quantity", "unit_cost", "total_value", "reference", "reason", "note", "created_by",
		"reverses_movement_id", "reversed_by_id",
	}, rows)
}
