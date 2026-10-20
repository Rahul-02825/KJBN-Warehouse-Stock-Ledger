package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/ledger"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

type Config struct {
	AllowedOrigins []string
	Production     bool
	Logs           *LogBuffer
}

type Server struct {
	pool   *pgxpool.Pool
	q      *sqlc.Queries
	ledger *ledger.Ledger
	logs   *LogBuffer
}

func NewRouter(pool *pgxpool.Pool, cfg Config) http.Handler {
	hideErrorDetails = cfg.Production
	s := &Server{pool: pool, q: sqlc.New(pool), ledger: ledger.New(pool), logs: cfg.Logs}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/openapi.yaml", s.openapiSpec)
	mux.HandleFunc("GET /api/docs", s.docsPage)

	mux.HandleFunc("POST /api/products", s.createProduct)
	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("GET /api/products/{id}", s.getProduct)
	mux.HandleFunc("PATCH /api/products/{id}", s.updateProduct)
	mux.HandleFunc("GET /api/products/{id}/balances", s.productBalances)
	mux.HandleFunc("GET /api/products/{id}/ledger", s.productLedger)

	mux.HandleFunc("POST /api/locations", s.createLocation)
	mux.HandleFunc("GET /api/locations", s.listLocations)
	mux.HandleFunc("GET /api/locations/{id}", s.getLocation)
	mux.HandleFunc("GET /api/locations/{id}/balances", s.locationBalances)

	mux.HandleFunc("GET /api/warehouses", s.listWarehouses)
	mux.HandleFunc("POST /api/placements/preview", s.previewPlacement)
	mux.HandleFunc("POST /api/picks/preview", s.previewPick)

	mux.HandleFunc("GET /api/inventory", s.stockGrid)
	mux.HandleFunc("GET /api/inventory/as-of", s.stockAsOf)
	mux.HandleFunc("GET /api/inventory/{productId}/{locationId}", s.stockCell)

	mux.HandleFunc("POST /api/movements/in", s.receiveStock)
	mux.HandleFunc("POST /api/movements/out", s.shipStock)
	mux.HandleFunc("POST /api/movements/transfer", s.transferStock)
	mux.HandleFunc("POST /api/movements/adjustment", s.adjustStock)
	mux.HandleFunc("POST /api/movements/{id}/reverse", s.reverseMovement)
	mux.HandleFunc("POST /api/stock-takes", s.stockTake)
	mux.HandleFunc("GET /api/movements", s.listMovements)
	mux.HandleFunc("GET /api/movements/{id}", s.getMovement)

	mux.HandleFunc("GET /api/reports/low-stock", s.lowStockReport)
	mux.HandleFunc("GET /api/reports/valuation", s.valuationReport)
	mux.HandleFunc("GET /api/reports/reconciliation", s.reconciliationReport)
	mux.HandleFunc("GET /api/reports/cost-check", s.costCheckReport)
	mux.HandleFunc("GET /api/reports/daily-activity", s.dailyActivityReport)
	mux.HandleFunc("GET /api/reports/top-movers", s.topMoversReport)
	mux.HandleFunc("GET /api/reports/season-forecast", s.seasonForecastReport)
	mux.HandleFunc("GET /api/logs", s.recentLogs)

	return Logger(Recover(CORS(cfg.AllowedOrigins, mux)))
}
