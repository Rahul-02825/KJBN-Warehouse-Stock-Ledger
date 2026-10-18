package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

// NewRouter registers every route and wraps them in the middleware.
func NewRouter(pool *pgxpool.Pool) http.Handler {
	q := sqlc.New(pool)
	mux := http.NewServeMux()
	NewProductHandler(q).Register(mux)
	NewLocationHandler(q).Register(mux)
	NewInventoryHandler(q).Register(mux)
	NewMovementHandler(q, NewLedger(pool)).Register(mux)
	NewReportHandler(q).Register(mux)
	NewHealthHandler(pool).Register(mux)
	DocsHandler{}.Register(mux)
	return Logger(Recover(mux))
}
