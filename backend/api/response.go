package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

func optionalID(fields map[string]string, query url.Values, name string) *int64 {
	raw := query.Get(name)
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		fields[name] = "must be a positive integer"
		return nil
	}
	return &id
}

// Plain dates in filters are read as Indian Standard Time, the business's local day.
var ist = time.FixedZone("IST", 5*3600+1800)

// optionalTime accepts 2006-01-02 or RFC 3339. A plain date used as the upper
// bound includes that whole day.
func optionalTime(fields map[string]string, query url.Values, name string, upper bool) *time.Time {
	raw := query.Get(name)
	if raw == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t
	}
	t, err := time.ParseInLocation("2006-01-02", raw, ist)
	if err != nil {
		fields[name] = "must be a date (2006-01-02) or RFC 3339 timestamp"
		return nil
	}
	if upper {
		t = t.AddDate(0, 0, 1)
	}
	return &t
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// csvMaxRows caps a CSV download so one request cannot read an unbounded table.
const csvMaxRows = 100000

func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	cw := csv.NewWriter(w)
	cw.Write(header)
	cw.WriteAll(rows)
}

// csvText stops spreadsheets from running user-entered text as a formula.
func csvText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func csvPtr(s *string) string {
	if s == nil {
		return ""
	}
	return csvText(*s)
}

func csvDecimal(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func csvID(id *int64) string {
	if id == nil {
		return ""
	}
	return strconv.FormatInt(*id, 10)
}
