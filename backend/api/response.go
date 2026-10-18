package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]errorBody{"error": {Code: code, Message: message}})
}

func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	writeJSON(w, http.StatusBadRequest, map[string]errorBody{
		"error": {Code: "VALIDATION_ERROR", Message: "validation failed", Fields: fields},
	})
}

// writeInternalError logs the real error and hides it from the client.
func writeInternalError(w http.ResponseWriter, err error) {
	slog.Error("internal error", "error", err.Error())
	writeError(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func toNumeric(d decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{Int: d.Coefficient(), Exp: d.Exponent(), Valid: true}
}

func toDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid || n.Int == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, n.Exp)
}

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

func optionalID(fields map[string]string, query url.Values, name string) pgtype.Int8 {
	raw := query.Get(name)
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

// Plain dates in filters are read as Indian Standard Time, the business's local day.
var ist = time.FixedZone("IST", 5*3600+1800)

// optionalTime accepts 2006-01-02 or RFC 3339. A plain date used as the upper
// bound includes that whole day.
func optionalTime(fields map[string]string, query url.Values, name string, upper bool) pgtype.Timestamptz {
	raw := query.Get(name)
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
