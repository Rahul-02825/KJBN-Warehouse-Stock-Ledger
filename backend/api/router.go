package api

import (
	"net/http"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

// NewRouter registers every route and wraps them in the middleware.
func NewRouter(q *sqlc.Queries) http.Handler {
	mux := http.NewServeMux()
	NewProductHandler(q).Register(mux)
	NewLocationHandler(q).Register(mux)
	return Logger(Recover(mux))
}
