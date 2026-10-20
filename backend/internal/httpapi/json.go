package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres"
)

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

var hideErrorDetails bool

type badRequestError struct{ message string }

func (e badRequestError) Error() string { return e.message }

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeErrorBody(w http.ResponseWriter, status int, body errorBody) {
	writeJSON(w, status, map[string]errorBody{"error": body})
}

func writeError(w http.ResponseWriter, r *http.Request, err error, fields ...any) {
	var badRequest badRequestError
	var validation domain.ValidationError
	var notFound domain.NotFoundError
	var insufficient domain.InsufficientStockError
	var noSpace domain.NoSpaceError
	var notReversible domain.NotReversibleError

	switch {
	case errors.As(err, &badRequest):
		writeErrorBody(w, http.StatusBadRequest, errorBody{Code: "BAD_REQUEST", Message: badRequest.message})
		return
	case errors.As(err, &validation):
		writeErrorBody(w, http.StatusBadRequest, errorBody{Code: "VALIDATION_ERROR", Message: "validation failed", Fields: validation})
	case errors.As(err, &notFound):
		writeErrorBody(w, http.StatusNotFound, errorBody{Code: "NOT_FOUND", Message: notFound.Error()})
	case errors.As(err, &insufficient):
		writeErrorBody(w, http.StatusConflict, errorBody{
			Code:    "INSUFFICIENT_STOCK",
			Message: insufficient.Error(),
			Details: map[string]string{"available": insufficient.Available.String(), "requested": insufficient.Requested.String()},
		})
	case errors.As(err, &noSpace):
		writeErrorBody(w, http.StatusConflict, errorBody{
			Code:    "NO_SPACE",
			Message: noSpace.Error(),
			Details: map[string]string{"fit": noSpace.Fit.String(), "requested": noSpace.Requested.String()},
		})
	case errors.Is(err, domain.ErrAlreadyReversed):
		writeErrorBody(w, http.StatusConflict, errorBody{Code: "ALREADY_REVERSED", Message: err.Error()})
	case errors.As(err, &notReversible):
		writeErrorBody(w, http.StatusConflict, errorBody{Code: "NOT_REVERSIBLE", Message: notReversible.Error()})
	case errors.Is(err, domain.ErrKeyReused):
		writeErrorBody(w, http.StatusConflict, errorBody{Code: "IDEMPOTENCY_KEY_REUSED", Message: err.Error()})
	case postgres.IsNumericOverflow(err):
		writeErrorBody(w, http.StatusBadRequest, errorBody{Code: "OUT_OF_RANGE", Message: "a resulting quantity or value is too large"})
	default:
		slog.Error("internal error", internalErrorFields(r, err, fields)...)
		writeErrorBody(w, http.StatusInternalServerError, errorBody{Code: "INTERNAL", Message: "internal server error"})
		return
	}
	slog.Warn("request rejected", logFields(r, err, fields)...)
}

func logFields(r *http.Request, err error, fields []any) []any {
	all := []any{"request_id", requestID(r), "path", r.URL.Path, "reason", err.Error()}
	return append(all, fields...)
}

func internalErrorFields(r *http.Request, err error, fields []any) []any {
	if !hideErrorDetails {
		return logFields(r, err, fields)
	}
	all := []any{"request_id", requestID(r), "path", r.URL.Path}
	if code, constraint, ok := postgres.ErrorCode(err); ok {
		all = append(all, "error_kind", "postgres", "sqlstate", code, "constraint", constraint)
	} else {
		all = append(all, "error_kind", fmt.Sprintf("%T", err))
	}
	return append(all, fields...)
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return badRequestError{"invalid JSON body: " + err.Error()}
	}
	return nil
}

func readID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		return 0, badRequestError{name + " must be a positive integer"}
	}
	return id, nil
}

func readKey(r *http.Request) (string, error) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(key) > 128 {
		return "", domain.ValidationError{"Idempotency-Key": "max 128 characters"}
	}
	return key, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
