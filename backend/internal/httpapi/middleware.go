package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder remembers the status code so the logger can print it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logger gives every request an id, returns it in X-Request-ID and prints one line per request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := newRequestID(r)
		w.Header().Set("X-Request-ID", id)
		r = withRequestID(r, id)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"request_id", id,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// Recover turns a panic into a 500 response instead of crashing the server.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic", "request_id", requestID(r), "error", err, "path", r.URL.Path)
				writeErrorBody(w, http.StatusInternalServerError, errorBody{Code: "INTERNAL", Message: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
