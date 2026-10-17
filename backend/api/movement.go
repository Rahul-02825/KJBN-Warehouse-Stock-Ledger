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

type inRequest struct {
	ProductID  int64           `json:"product_id"`
	LocationID int64           `json:"location_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	UnitCost   decimal.Decimal `json:"unit_cost"`
	Reference  string          `json:"reference"`
	CreatedBy  string          `json:"created_by"`
}

type outRequest struct {
	ProductID  int64           `json:"product_id"`
	LocationID int64           `json:"location_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Reference  string          `json:"reference"`
	CreatedBy  string          `json:"created_by"`
}

type transferRequest struct {
	ProductID      int64           `json:"product_id"`
	FromLocationID int64           `json:"from_location_id"`
	ToLocationID   int64           `json:"to_location_id"`
	Quantity       decimal.Decimal `json:"quantity"`
	Reference      string          `json:"reference"`
	CreatedBy      string          `json:"created_by"`
}

type adjustmentRequest struct {
	ProductID  int64           `json:"product_id"`
	LocationID int64           `json:"location_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Reason     string          `json:"reason"`
	Note       string          `json:"note"`
	Reference  string          `json:"reference"`
	CreatedBy  string          `json:"created_by"`
}

type reverseRequest struct {
	Note      string `json:"note"`
	CreatedBy string `json:"created_by"`
}

func (h *MovementHandler) postIn(w http.ResponseWriter, r *http.Request) {
	var req inRequest
	if !readJSON(w, r, &req) {
		return
	}
	fields := map[string]string{}
	checkIDs(fields, map[string]int64{"product_id": req.ProductID, "location_id": req.LocationID})
	checkAmount(fields, "quantity", req.Quantity)
	checkAmount(fields, "unit_cost", req.UnitCost)
	req.Reference, req.CreatedBy = checkText(fields, req.Reference, req.CreatedBy)
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	m, err := h.ledger.PostIn(r.Context(), InInput{
		ProductID: req.ProductID, LocationID: req.LocationID,
		Quantity: req.Quantity, UnitCost: req.UnitCost,
		Reference: req.Reference, CreatedBy: req.CreatedBy,
	})
	h.respondPosted(w, r, m, err)
}

func (h *MovementHandler) postOut(w http.ResponseWriter, r *http.Request) {
	var req outRequest
	if !readJSON(w, r, &req) {
		return
	}
	fields := map[string]string{}
	checkIDs(fields, map[string]int64{"product_id": req.ProductID, "location_id": req.LocationID})
	checkAmount(fields, "quantity", req.Quantity)
	req.Reference, req.CreatedBy = checkText(fields, req.Reference, req.CreatedBy)
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	m, err := h.ledger.PostOut(r.Context(), OutInput{
		ProductID: req.ProductID, LocationID: req.LocationID, Quantity: req.Quantity,
		Reference: req.Reference, CreatedBy: req.CreatedBy,
	})
	h.respondPosted(w, r, m, err)
}

func (h *MovementHandler) postTransfer(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if !readJSON(w, r, &req) {
		return
	}
	fields := map[string]string{}
	checkIDs(fields, map[string]int64{
		"product_id": req.ProductID, "from_location_id": req.FromLocationID, "to_location_id": req.ToLocationID,
	})
	if req.FromLocationID != 0 && req.FromLocationID == req.ToLocationID {
		fields["to_location_id"] = "must be different from from_location_id"
	}
	checkAmount(fields, "quantity", req.Quantity)
	req.Reference, req.CreatedBy = checkText(fields, req.Reference, req.CreatedBy)
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	m, err := h.ledger.PostTransfer(r.Context(), TransferInput{
		ProductID: req.ProductID, FromLocationID: req.FromLocationID, ToLocationID: req.ToLocationID,
		Quantity: req.Quantity, Reference: req.Reference, CreatedBy: req.CreatedBy,
	})
	h.respondPosted(w, r, m, err)
}

func (h *MovementHandler) postAdjustment(w http.ResponseWriter, r *http.Request) {
	var req adjustmentRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	req.Note = strings.TrimSpace(req.Note)

	fields := map[string]string{}
	checkIDs(fields, map[string]int64{"product_id": req.ProductID, "location_id": req.LocationID})
	checkAmount(fields, "quantity", req.Quantity.Abs())
	if !ValidReason(req.Reason) {
		fields["reason"] = "must be one of damaged, expired, count_correction, theft, other"
	}
	if req.Reason == "other" && req.Note == "" {
		fields["note"] = "is required when reason is other"
	}
	if len(req.Note) > 1000 {
		fields["note"] = "max 1000 characters"
	}
	req.Reference, req.CreatedBy = checkText(fields, req.Reference, req.CreatedBy)
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	m, err := h.ledger.PostAdjustment(r.Context(), AdjustmentInput{
		ProductID: req.ProductID, LocationID: req.LocationID, Quantity: req.Quantity,
		Reason: req.Reason, Note: req.Note, Reference: req.Reference, CreatedBy: req.CreatedBy,
	})
	h.respondPosted(w, r, m, err)
}

func (h *MovementHandler) reverse(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	var req reverseRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	fields := map[string]string{}
	_, req.CreatedBy = checkText(fields, "", req.CreatedBy)
	if len(req.Note) > 1000 {
		fields["note"] = "max 1000 characters"
	}
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	m, err := h.ledger.PostReversal(r.Context(), ReversalInput{MovementID: id, Note: req.Note, CreatedBy: req.CreatedBy})
	h.respondPosted(w, r, m, err)
}

func (h *MovementHandler) respondPosted(w http.ResponseWriter, r *http.Request, m sqlc.Movement, err error) {
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
	case isNumericOverflow(err):
		writeError(w, http.StatusBadRequest, "OUT_OF_RANGE", "a resulting quantity or value is too large")
	default:
		writeInternalError(w, err)
		return
	}
	slog.Warn("movement rejected", "path", r.URL.Path, "reason", err.Error())
}

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

	resp := movementDetailResponse{
		movementResponse: toMovementResponse(sqlc.Movement{
			ID: d.ID, ProductID: d.ProductID, MovementType: d.MovementType,
			FromLocationID: d.FromLocationID, ToLocationID: d.ToLocationID,
			Quantity: d.Quantity, UnitCost: d.UnitCost, Reference: d.Reference,
			Reason: d.Reason, Note: d.Note, CreatedBy: d.CreatedBy,
			ReversalOfID: d.ReversalOfID, CreatedAt: d.CreatedAt,
		}),
		SKU: d.Sku, ProductName: d.ProductName, Unit: d.Unit,
		FromLocationCode: textPtr(d.FromLocationCode), ToLocationCode: textPtr(d.ToLocationCode),
		IsReversed: d.IsReversed, ReversedByID: idPtr(d.ReversedByID),
	}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, entryResponse{
			LocationID: e.LocationID, QuantityDelta: toDecimal(e.QuantityDelta), UnitCost: numPtr(e.UnitCost),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// list supports filters: product_id, location_id, type, reference, from, to (dates or RFC 3339), page, page_size.
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
	if len(fields) > 0 {
		writeValidationError(w, fields)
		return
	}

	page, size := pageParams(query.Get("page"), query.Get("page_size"))
	rows, err := h.q.ListMovementHistory(r.Context(), sqlc.ListMovementHistoryParams{
		ProductID: f.ProductID, MovementType: f.MovementType, LocationID: f.LocationID,
		Reference: f.Reference, FromDate: f.FromDate, ToDate: f.ToDate,
		PageSize: int32(size), PageOffset: int32((page - 1) * size),
	})
	if err != nil {
		writeInternalError(w, err)
		return
	}
	total, err := h.q.CountMovementHistory(r.Context(), f)
	if err != nil {
		writeInternalError(w, err)
		return
	}

	data := make([]movementDetailResponse, len(rows))
	for i, d := range rows {
		data[i] = movementDetailResponse{
			movementResponse: toMovementResponse(sqlc.Movement{
				ID: d.ID, ProductID: d.ProductID, MovementType: d.MovementType,
				FromLocationID: d.FromLocationID, ToLocationID: d.ToLocationID,
				Quantity: d.Quantity, UnitCost: d.UnitCost, Reference: d.Reference,
				Reason: d.Reason, Note: d.Note, CreatedBy: d.CreatedBy,
				ReversalOfID: d.ReversalOfID, CreatedAt: d.CreatedAt,
			}),
			SKU: d.Sku, ProductName: d.ProductName, Unit: d.Unit,
			FromLocationCode: textPtr(d.FromLocationCode), ToLocationCode: textPtr(d.ToLocationCode),
			IsReversed: d.IsReversed, ReversedByID: idPtr(d.ReversedByID),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "page": page, "page_size": size, "total": total})
}

func checkIDs(fields map[string]string, ids map[string]int64) {
	for name, id := range ids {
		if id < 1 {
			fields[name] = "is required and must be a positive integer"
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

func optionalID(fields map[string]string, query map[string][]string, name string) pgtype.Int8 {
	raw := ""
	if v := query[name]; len(v) > 0 {
		raw = v[0]
	}
	if raw == "" {
		return pgtype.Int8{}
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		fields[name] = "must be a positive integer"
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}

// optionalTime accepts 2006-01-02 or RFC 3339. A plain date used as the upper
// bound includes that whole day.
func optionalTime(fields map[string]string, query map[string][]string, name string, upper bool) pgtype.Timestamptz {
	raw := ""
	if v := query[name]; len(v) > 0 {
		raw = v[0]
	}
	if raw == "" {
		return pgtype.Timestamptz{}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return pgtype.Timestamptz{Time: t, Valid: true}
	}
	t, err := time.ParseInLocation("2006-01-02", raw, ist)
	if err != nil {
		fields[name] = "must be a date (2006-01-02) or RFC 3339 timestamp"
		return pgtype.Timestamptz{}
	}
	if upper {
		t = t.AddDate(0, 0, 1)
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// Plain dates in filters are read as Indian Standard Time, the business's local day.
var ist = time.FixedZone("IST", 5*3600+1800)

func pageParams(pageRaw, sizeRaw string) (int, int) {
	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeRaw)
	if err != nil || size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func idPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func numPtr(v pgtype.Numeric) *decimal.Decimal {
	if !v.Valid {
		return nil
	}
	d := toDecimal(v)
	return &d
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}
