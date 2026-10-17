package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
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
