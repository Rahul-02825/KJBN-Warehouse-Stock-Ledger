package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type MovementHandler struct {
	q      *sqlc.Queries
	ledger *Ledger
}

func NewMovementHandler(q *sqlc.Queries, ledger *Ledger) *MovementHandler {
	return &MovementHandler{q: q, ledger: ledger}
}

func (h *MovementHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/movements/in", h.postIn)
	mux.HandleFunc("POST /api/movements/out", h.postOut)
	mux.HandleFunc("POST /api/movements/transfer", h.postTransfer)
	mux.HandleFunc("POST /api/movements/adjustment", h.postAdjustment)
	mux.HandleFunc("POST /api/movements/{id}/reverse", h.reverse)
	mux.HandleFunc("GET /api/movements", h.list)
	mux.HandleFunc("GET /api/movements/{id}", h.get)
}

// Requests. Each validate method trims text fields and returns one message per invalid field.

type InRequest struct {
	ProductID   int64           `json:"product_id"`
	LocationID  int64           `json:"location_id"`
	WarehouseID int64           `json:"warehouse_id"`
	Quantity    decimal.Decimal `json:"quantity"`
	UnitCost    decimal.Decimal `json:"unit_cost"`
	Reference   string          `json:"reference"`
	CreatedBy   string          `json:"created_by"`

	IdempotencyKey string `json:"-"` // from the Idempotency-Key header
}

func (r *InRequest) validate() map[string]string {
	f := map[string]string{}
	checkIDs(f, map[string]int64{"product_id": r.ProductID})
	checkOptionalIDs(f, map[string]int64{"location_id": r.LocationID, "warehouse_id": r.WarehouseID})
	checkAmount(f, "quantity", r.Quantity)
	checkAmount(f, "unit_cost", r.UnitCost)
	r.Reference, r.CreatedBy = checkText(f, r.Reference, r.CreatedBy)
	return f
}

type OutRequest struct {
	ProductID  int64           `json:"product_id"`
	LocationID int64           `json:"location_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Reference  string          `json:"reference"`
	CreatedBy  string          `json:"created_by"`

	IdempotencyKey string `json:"-"`
}

func (r *OutRequest) validate() map[string]string {
	f := map[string]string{}
	checkIDs(f, map[string]int64{"product_id": r.ProductID, "location_id": r.LocationID})
	checkAmount(f, "quantity", r.Quantity)
	r.Reference, r.CreatedBy = checkText(f, r.Reference, r.CreatedBy)
	return f
}

type TransferRequest struct {
	ProductID      int64           `json:"product_id"`
	FromLocationID int64           `json:"from_location_id"`
	ToLocationID   int64           `json:"to_location_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`

	IdempotencyKey string `json:"-"`
}

func (r *TransferRequest) validate() map[string]string {
	f := map[string]string{}
	checkIDs(f, map[string]int64{
		"product_id": r.ProductID, "from_location_id": r.FromLocationID, "to_location_id": r.ToLocationID,
	})
	if r.FromLocationID != 0 && r.FromLocationID == r.ToLocationID {
		f["to_location_id"] = "must be different from from_location_id"
	}
	checkAmount(f, "quantity", r.Quantity)
	r.Reference, r.CreatedBy = checkText(f, r.Reference, r.CreatedBy)
	return f
}

type AdjustmentRequest struct {
	ProductID  int64           `json:"product_id"`
	LocationID int64           `json:"location_id"`
	Quantity   decimal.Decimal `json:"quantity"` // signed
	Reason     string          `json:"reason"`
	Note       string          `json:"note"`
	Reference  string          `json:"reference"`
	CreatedBy  string          `json:"created_by"`

	IdempotencyKey string `json:"-"`
}

func (r *AdjustmentRequest) validate() map[string]string {
	r.Reason = strings.TrimSpace(r.Reason)
	r.Note = strings.TrimSpace(r.Note)
	f := map[string]string{}
	checkIDs(f, map[string]int64{"product_id": r.ProductID, "location_id": r.LocationID})
	checkAmount(f, "quantity", r.Quantity.Abs())
	if !ValidReason(r.Reason) {
		f["reason"] = "must be one of damaged, expired, count_correction, theft, other"
	}
	if r.Reason == "other" && r.Note == "" {
		f["note"] = "is required when reason is other"
	}
	if len(r.Note) > 1000 {
		f["note"] = "max 1000 characters"
	}
	r.Reference, r.CreatedBy = checkText(f, r.Reference, r.CreatedBy)
	return f
}

type ReverseRequest struct {
	Note      string `json:"note"`
	CreatedBy string `json:"created_by"`

	IdempotencyKey string `json:"-"`
}

func (r *ReverseRequest) validate() map[string]string {
	r.Note = strings.TrimSpace(r.Note)
	f := map[string]string{}
	_, r.CreatedBy = checkText(f, "", r.CreatedBy)
	if len(r.Note) > 1000 {
		f["note"] = "max 1000 characters"
	}
	return f
}

// Handlers: read the key and JSON, validate, post, respond.

func (h *MovementHandler) postIn(w http.ResponseWriter, r *http.Request) {
	var req InRequest
	key, ok := readKey(w, r)
	if !ok || !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	req.IdempotencyKey = key
	m, err := h.ledger.PostIn(r.Context(), req)
	h.respond(w, r, key, m, err, func(o sqlc.Movement) bool {
		return o.MovementType == "IN" && o.ProductID == req.ProductID
	})
}

func (h *MovementHandler) postOut(w http.ResponseWriter, r *http.Request) {
	var req OutRequest
	key, ok := readKey(w, r)
	if !ok || !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	req.IdempotencyKey = key
	m, err := h.ledger.PostOut(r.Context(), req)
	h.respond(w, r, key, m, err, func(o sqlc.Movement) bool {
		return o.MovementType == "OUT" && o.ProductID == req.ProductID
	})
}

func (h *MovementHandler) postTransfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	key, ok := readKey(w, r)
	if !ok || !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	req.IdempotencyKey = key
	m, err := h.ledger.PostTransfer(r.Context(), req)
	h.respond(w, r, key, m, err, func(o sqlc.Movement) bool {
		return o.MovementType == "TRANSFER" && o.ProductID == req.ProductID
	})
}

func (h *MovementHandler) postAdjustment(w http.ResponseWriter, r *http.Request) {
	var req AdjustmentRequest
	key, ok := readKey(w, r)
	if !ok || !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	req.IdempotencyKey = key
	m, err := h.ledger.PostAdjustment(r.Context(), req)
	h.respond(w, r, key, m, err, func(o sqlc.Movement) bool {
		return o.MovementType == "ADJUSTMENT" && o.ProductID == req.ProductID
	})
}

func (h *MovementHandler) reverse(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var req ReverseRequest
	key, ok := readKey(w, r)
	if !ok || !readJSON(w, r, &req) || !valid(w, req.validate()) {
		return
	}
	req.IdempotencyKey = key
	m, err := h.ledger.PostReversal(r.Context(), id, req)
	h.respond(w, r, key, m, err, func(o sqlc.Movement) bool {
		return o.MovementType == "REVERSAL" && o.ReversalOfID.Int64 == id
	})
}

func (h *MovementHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	d, err := h.q.GetMovementDetail(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "movement not found")
		return
	}
	if err != nil {
		writeInternalError(w, err)
		return
	}
	entries, err := h.q.ListMovementEntries(r.Context(), id)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	resp := toDetailResponse(d)
	for _, e := range entries {
		resp.Entries = append(resp.Entries, entryResponse{
			LocationID: e.LocationID, QuantityDelta: toDecimal(e.QuantityDelta), UnitCost: numPtr(e.UnitCost),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// list supports filters: product_id, location_id, type, reference, from, to (dates or RFC 3339), page, page_size.
// ?format=csv downloads every matching row (up to csvMaxRows) instead of one page.
func (h *MovementHandler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fields := map[string]string{}
	f := sqlc.CountMovementHistoryParams{
		ProductID:  optionalID(fields, query, "product_id"),
		LocationID: optionalID(fields, query, "location_id"),
		FromDate:   optionalTime(fields, query, "from", false),
		ToDate:     optionalTime(fields, query, "to", true),
	}
	if t := strings.ToUpper(query.Get("type")); t != "" {
		switch t {
		case "IN", "OUT", "TRANSFER", "ADJUSTMENT", "REVERSAL":
			f.MovementType = pgtype.Text{String: t, Valid: true}
		default:
			fields["type"] = "must be one of IN, OUT, TRANSFER, ADJUSTMENT, REVERSAL"
		}
	}
	if ref := strings.TrimSpace(query.Get("reference")); ref != "" {
		f.Reference = pgtype.Text{String: ref, Valid: true}
	}
	if !valid(w, fields) {
		return
	}

	page, size := pageParams(query.Get("page"), query.Get("page_size"))
	asCSV := query.Get("format") == "csv"
	if asCSV {
		page, size = 1, csvMaxRows
	}
	rows, err := h.q.ListMovementHistory(r.Context(), sqlc.ListMovementHistoryParams{
		ProductID: f.ProductID, MovementType: f.MovementType, LocationID: f.LocationID,
		Reference: f.Reference, FromDate: f.FromDate, ToDate: f.ToDate,
		PageSize: int32(size), PageOffset: int32((page - 1) * size),
	})
	if err != nil {
		writeInternalError(w, err)
		return
	}
	data := make([]movementDetailResponse, len(rows))
	for i, row := range rows {
		data[i] = toDetailResponse(sqlc.GetMovementDetailRow(row)) // same columns, same struct shape
	}
	if asCSV {
		writeMovementsCSV(w, data)
		return
	}

	total, err := h.q.CountMovementHistory(r.Context(), f)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": page, "page_size": size, "total": total})
}

func valid(w http.ResponseWriter, fields map[string]string) bool {
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return false
	}
	return true
}

func writeMovementsCSV(w http.ResponseWriter, data []movementDetailResponse) {
	rows := make([][]string, len(data))
	for i, m := range data {
		rows[i] = []string{
			strconv.FormatInt(m.ID, 10), m.CreatedAt.Format(time.RFC3339), m.Type,
			m.SKU, csvText(m.ProductName), m.Unit, csvPtr(m.FromLocationCode), csvPtr(m.ToLocationCode),
			m.Quantity.String(), csvDecimal(m.UnitCost), csvDecimal(m.TotalValue),
			csvText(m.Reference), csvPtr(m.Reason), csvText(m.Note), csvText(m.CreatedBy),
			csvID(m.ReversesMovementID), csvID(m.ReversedByID),
		}
	}
	writeCSV(w, "movements.csv", []string{
		"id", "created_at", "type", "sku", "product_name", "unit", "from_location", "to_location",
		"quantity", "unit_cost", "total_value", "reference", "reason", "note", "created_by",
		"reverses_movement_id", "reversed_by_id",
	}, rows)
}

// readKey reads the optional Idempotency-Key header. A retry with the same key
// returns the first result instead of posting again.
func readKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(key) > 128 {
		writeValidationError(w, map[string]string{"Idempotency-Key": "max 128 characters"})
		return "", false
	}
	return key, true
}

// respond writes the posting result. When the key was already used, it returns the
// original movement with 200, provided the key was used for the same kind of request.
func (h *MovementHandler) respond(w http.ResponseWriter, r *http.Request, key string, m sqlc.Movement, err error, sameRequest func(sqlc.Movement) bool) {
	if errors.Is(err, errReplay) {
		original, err := h.q.GetMovementByIdempotencyKey(r.Context(), optText(key))
		if err != nil {
			writeInternalError(w, err)
			return
		}
		if !sameRequest(original) {
			writeError(w, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "this Idempotency-Key was already used for a different request")
			return
		}
		w.Header().Set("Idempotent-Replayed", "true")
		writeJSON(w, http.StatusOK, toMovementResponse(original))
		return
	}
	if err != nil {
		writePostingError(w, r, err)
		return
	}
	slog.Info("movement posted", "movement_id", m.ID, "type", m.MovementType, "product_id", m.ProductID)
	writeJSON(w, http.StatusCreated, toMovementResponse(m))
}

// writePostingError maps business rule failures to HTTP responses.
func writePostingError(w http.ResponseWriter, r *http.Request, err error) {
	var ve validationError
	var nf notFoundError
	var is insufficientStockError
	var nr notReversibleError
	var ns noSpaceError

	switch {
	case errors.As(err, &ve):
		writeValidationError(w, ve)
	case errors.As(err, &nf):
		writeError(w, http.StatusNotFound, "NOT_FOUND", nf.Error())
	case errors.As(err, &is):
		writeJSON(w, http.StatusConflict, map[string]errorBody{"error": {
			Code:    "INSUFFICIENT_STOCK",
			Message: is.Error(),
			Details: map[string]string{"available": is.available.String(), "requested": is.requested.String()},
		}})
	case errors.Is(err, errAlreadyReversed):
		writeError(w, http.StatusConflict, "ALREADY_REVERSED", err.Error())
	case errors.As(err, &nr):
		writeError(w, http.StatusConflict, "NOT_REVERSIBLE", nr.Error())
	case errors.As(err, &ns):
		writeJSON(w, http.StatusConflict, map[string]errorBody{"error": {
			Code:    "NO_SPACE",
			Message: ns.Error(),
			Details: map[string]string{"fit": fromUnits(ns.fit).String(), "requested": fromUnits(ns.requested).String()},
		}})
	case isNumericOverflow(err):
		writeError(w, http.StatusBadRequest, "OUT_OF_RANGE", "a resulting quantity or value is too large")
	default:
		writeInternalError(w, err)
		return
	}
	slog.Warn("movement rejected", "path", r.URL.Path, "reason", err.Error())
}

// Responses

type movementResponse struct {
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

type movementDetailResponse struct {
	movementResponse
	SKU              string          `json:"sku"`
	ProductName      string          `json:"product_name"`
	Unit             string          `json:"unit"`
	FromLocationCode *string         `json:"from_location_code"`
	ToLocationCode   *string         `json:"to_location_code"`
	IsReversed       bool            `json:"is_reversed"`
	ReversedByID     *int64          `json:"reversed_by_id"`
	Entries          []entryResponse `json:"entries,omitempty"`
}

type entryResponse struct {
	LocationID    int64            `json:"location_id"`
	QuantityDelta decimal.Decimal  `json:"quantity_delta"`
	UnitCost      *decimal.Decimal `json:"unit_cost"`
}

func toMovementResponse(m sqlc.Movement) movementResponse {
	qty := toDecimal(m.Quantity)
	resp := movementResponse{
		ID:                 m.ID,
		Type:               m.MovementType,
		ProductID:          m.ProductID,
		FromLocationID:     idPtr(m.FromLocationID),
		ToLocationID:       idPtr(m.ToLocationID),
		Quantity:           qty,
		UnitCost:           numPtr(m.UnitCost),
		Reference:          m.Reference,
		Reason:             textPtr(m.Reason),
		Note:               m.Note,
		CreatedBy:          m.CreatedBy,
		ReversesMovementID: idPtr(m.ReversalOfID),
		CreatedAt:          m.CreatedAt.Time.UTC(),
	}
	if resp.UnitCost != nil {
		total := qty.Mul(*resp.UnitCost).Round(4)
		resp.TotalValue = &total
	}
	return resp
}

func toDetailResponse(d sqlc.GetMovementDetailRow) movementDetailResponse {
	return movementDetailResponse{
		movementResponse: toMovementResponse(sqlc.Movement{
			ID: d.ID, ProductID: d.ProductID, MovementType: d.MovementType,
			FromLocationID: d.FromLocationID, ToLocationID: d.ToLocationID,
			Quantity: d.Quantity, UnitCost: d.UnitCost, Reference: d.Reference,
			Reason: d.Reason, Note: d.Note, CreatedBy: d.CreatedBy,
			ReversalOfID: d.ReversalOfID, CreatedAt: d.CreatedAt,
		}),
		SKU:              d.Sku,
		ProductName:      d.ProductName,
		Unit:             d.Unit,
		FromLocationCode: textPtr(d.FromLocationCode),
		ToLocationCode:   textPtr(d.ToLocationCode),
		IsReversed:       d.IsReversed,
		ReversedByID:     idPtr(d.ReversedByID),
	}
}

func checkIDs(fields map[string]string, ids map[string]int64) {
	for name, id := range ids {
		if id < 1 {
			fields[name] = "is required and must be a positive integer"
		}
	}
}

func checkOptionalIDs(fields map[string]string, ids map[string]int64) {
	for name, id := range ids {
		if id < 0 {
			fields[name] = "must be a positive integer"
		}
	}
}

func checkAmount(fields map[string]string, name string, d decimal.Decimal) {
	if msg := ValidateQuantity(d); msg != "" {
		fields[name] = msg
	}
}

// checkText trims and validates reference and created_by.
func checkText(fields map[string]string, reference, createdBy string) (string, string) {
	reference = strings.TrimSpace(reference)
	createdBy = strings.TrimSpace(createdBy)
	if len(reference) > 128 {
		fields["reference"] = "max 128 characters"
	}
	if createdBy == "" || len(createdBy) > 128 {
		fields["created_by"] = "is required, max 128 characters"
	}
	return reference, createdBy
}
