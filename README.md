# Warehouse Stock Ledger

## 1. Title and one-liner

**Warehouse Stock Ledger**, KJBN take-home, **Problem 3: Warehouse Stock Ledger**.

Stock is never edited directly. Every change is recorded as an immutable movement (IN, OUT, TRANSFER, ADJUSTMENT, REVERSAL), and every balance can be explained from those movements.

## 2. Live links

**Not deployed yet.** Hosting (milestone M5) is still to be done, so there is no live UI, API or docs URL at the moment. Section 5 shows how to run everything locally in a few minutes. Once deployed, this section will list:

| What | URL |
|---|---|
| UI | not deployed |
| API base URL | not deployed |
| API docs (Swagger UI) | not deployed. Locally: `http://localhost:8080/api/docs`, also linked from the UI footer |

There is no login, so no test credentials are needed (see assumption 16).

## 3. Tech stack

| Layer | Choice |
|---|---|
| Backend | Go 1.26, standard library `net/http` (no web framework) |
| Database access | PostgreSQL 16 through `pgx/v5`; queries written in SQL and turned into Go by `sqlc` 1.31.1; migrations with `golang-migrate` |
| Money and quantities | `NUMERIC(18,4)` in the database, `shopspring/decimal` in Go. No floats |
| Frontend | TypeScript, React 19, Vite 7, Tailwind CSS 4, React Router 7, lucide-react icons |
| Tests | Go `testing` against a real PostgreSQL; Vitest for frontend helpers |
| CI | GitHub Actions (`.github/workflows/ci.yml`) |
| Database hosting (development) | Supabase (PostgreSQL) or a local PostgreSQL 16 in Docker |
| Backend and UI hosting | Planned: API on Render (Docker, free tier), UI on Vercel. Not deployed yet (see section 2) |

All choices are from the recommended stacks in the brief.

## 4. Architecture

```
 Browser (React UI)
     │  fetch /api/...            Vite dev server proxies /api to the API
     ▼
 ┌──────────────────────── backend (Go) ─────────────────────────┐
 │ httpapi   routes, JSON in/out, error → HTTP status, logging   │
 │    │                                                          │
 │    ▼                                                          │
 │ ledger    one function per stock operation, one transaction   │
 │    │       each (lock → check → write movement + entries +    │
 │    │       balance → commit)                                  │
 │    ├──► domain   pure business rules: costing, validation,    │
 │    │             error types, bin planner (no DB, no HTTP)    │
 │    ▼                                                          │
 │ postgres  connect helper, Postgres error checks,              │
 │           sqlc-generated query code                           │
 └─────────────────────────────┬─────────────────────────────────┘
                               ▼
                        PostgreSQL 16
   movements + movement_entries (the ledger, append-only)
   stock_balances (a cache of the ledger)  products  locations  warehouses
```

Read-only endpoints (lists, reports) call the generated queries directly from `httpapi`. Anything that changes stock goes through `ledger`.

**Folder structure**

| Folder | What it holds |
|---|---|
| `backend/cmd/` | Programs: `api` (the server), `seed` (development data), `loadtest` (concurrent requests against a running API) |
| `backend/db/` | `migrations/` (golang-migrate), `schema/schema.sql` (snapshot that sqlc reads), `queries/` (all SQL, by topic) |
| `backend/internal/domain/` | Business rules with no database or HTTP code: rule lists, costing, validation, errors, bin placement |
| `backend/internal/ledger/` | Stock operations (receive, ship, transfer, adjust, reverse, stock take), each one database transaction |
| `backend/internal/httpapi/` | HTTP handlers, the route list (`server.go`), error mapping (`json.go`), middleware, OpenAPI spec |
| `backend/internal/postgres/` | Database connection, Postgres error helpers, and `sqlc/` (generated, never edited by hand) |
| `backend/test/` | API-level and integration tests against a real PostgreSQL |
| `frontend/` | React UI: `src/pages/` (screens), `src/components/` (shared UI), `src/api/` (API client and types), `src/lib/` (helpers) |
| `docs/` | `backend-guide.md`: how the backend is organised, request flow, recipes for adding rules and endpoints |
| `.github/workflows/` | CI pipeline |

**Where the business rules live and why.**
- Rules that don't need the database are in `internal/domain`: what a valid quantity is, the weighted-average formula, which reasons need a note, and so on. They are plain functions, quick to unit-test, and easy to find.
- Rules that need locking or several writes to happen together (no negative stock, atomic transfer, single reversal) are in `internal/ledger`, inside one transaction per operation.
- The database repeats the critical rules as constraints (`CHECK`, `UNIQUE`, foreign keys) as a last line of defence.
- HTTP handlers contain no business rules.

## 5. Running locally

### Quick start: one command with Docker

Only Docker is needed (Docker Desktop, or Colima with Compose).
```bash
git clone https://github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger.git
cd KJBN-Warehouse-Stock-Ledger
docker compose up --build        # or: docker-compose up --build
```
The first build takes a minute or two; restarts take a few seconds.

| Open | URL |
|---|---|
| UI | http://localhost:3000 |
| API | http://localhost:8080 (also reachable through the UI at http://localhost:3000/api/...) |
| Swagger | http://localhost:8080/api/docs |

`docker-compose.yml` starts five services in order, each waiting for the one before it:
1. **`db`:** PostgreSQL 16, with its data in a volume.
2. **`migrate`:** applies `backend/db/migrations`, then exits.
3. **`seed`:** loads the sample data, then exits.
4. **`api`:** the Go backend.
5. **`web`:** nginx serving the built UI and forwarding `/api` to the backend.

Stopping and starting keeps the data, and the seed adds nothing twice. `docker compose down -v` deletes the data, so the next `up` starts from a freshly seeded database.

**Configuration:** compose reads `.env` from the repo root (see `.env.example`). None of it is required:
- **`DATABASE_URL`:** if empty or unset, compose uses its built-in database. If set, migrate, seed and api use that database instead.
  - Inside containers, `localhost` means the container itself. For the built-in database the host is `db` (`postgres://ledger:ledger@db:5432/ledger?sslmode=disable`); a hosted database's URL works as it is.
  - **Compose migrates and seeds whatever database this points at.**
- **`WEB_PORT`, `API_PORT`:** change the ports if 3000 or 8080 is taken.
- **A different file:** `docker compose --env-file path/to/file up --build`.

If your `.env` holds a `localhost` URL for the manual setup below, compose can't reach it. Use a separate file for compose with `--env-file`.

### Manual setup (for development)

**Prerequisites**

| Tool | Version | Needed for |
|---|---|---|
| Go | 1.26 or newer | backend |
| Node.js and npm | Node 20.19+ or 22.12+ (developed on 25.9) | frontend |
| Docker | any recent version | running PostgreSQL 16 (or install PostgreSQL 16 yourself) |
| golang-migrate CLI | v4.18 or newer (`brew install golang-migrate`) | migrations |
| make | any | the backend commands |
| sqlc | 1.31.1 | only if you change SQL in `db/queries/` |

**Environment variables.** Copy `.env.example` to `.env` in the repo root. The backend's `make` commands read it.

| Variable | Meaning |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string for the API, migrations and seed |
| `PORT` | API port (default 8080) |
| `TEST_DATABASE_URL` | A **separate, throwaway** database for `go test`. Tests empty every table. Leave it unset to skip the database tests |
| `APP_ENV` | `production` hides raw error text in internal-error logs (section 9). Default `development` |
| `CORS_ALLOWED_ORIGIN` | Browser origins allowed to call the API, comma-separated, for example the Vercel UI URL. Leave empty locally, where Vite and nginx proxy `/api` |

The frontend has one optional variable, `VITE_API_URL` (`frontend/.env.example`). Leave it empty locally: Vite forwards `/api` to `http://localhost:8080`. Set it to the API URL for a hosted build.

**Commands** (from a clean clone, without Docker Compose):
```bash
git clone https://github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger.git
cd KJBN-Warehouse-Stock-Ledger

# 1. Start PostgreSQL 16 and create the app and test databases
docker run -d --name ledger-db -e POSTGRES_USER=ledger -e POSTGRES_PASSWORD=ledger \
  -e POSTGRES_DB=ledger -p 5432:5432 postgres:16
docker exec ledger-db createdb -U ledger ledger_test

# 2. Configure
cp .env.example .env
#    then set in .env:
#    DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable
#    TEST_DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger_test?sslmode=disable

# 3. Backend: install, migrate, seed, run
cd backend
go mod download
make migrate-up
make seed          # prints: 6 new locations, 8 new products, 10 opening movements, 4 last-year movements
make run           # API on http://localhost:8080, Swagger at http://localhost:8080/api/docs

# 4. Frontend (in a second terminal)
cd frontend
npm ci
npm run dev        # UI on http://localhost:5173
```
If port 5432 is already in use, map another port (for example `-p 5433:5432`) and change the URLs to match.

Use a different env file with `make <target> ENV_FILE=../.env.local`.

**One-command compose:** see the quick start at the top of this section.

## 5a. Deployment

**Not deployed yet.** What exists today:
- **Container builds:** `backend/Dockerfile` (the `api` and `seed` programs), `frontend/Dockerfile` (the built UI served by nginx, with `frontend/nginx.conf`) and `docker-compose.yml`, which runs the full stack locally. A hosting provider can build from the same Dockerfiles.
- **CI** (`.github/workflows/ci.yml`) runs on every push and pull request.
  - **Backend job**, against a PostgreSQL 16 service container: checks that the schema snapshot and the generated sqlc code are current, runs `gofmt`, `go vet` and the OpenAPI lint, builds, applies the migrations, runs the tests with coverage, and runs the load test against a live server.
  - **Frontend job:** lint, type check, tests and production build.
- **Migrate and seed** are `make migrate-up` and `make seed` (section 5). The seed is safe to rerun: it only adds what is missing.

- **Hosting config:** `render.yaml` (Render blueprint for the API: Docker, `/healthz` health check, `APP_ENV=production`) and `frontend/vercel.json` (sends every path to the app, so deep links work).
- **Cross-origin support:** the API allows the origins in `CORS_ALLOWED_ORIGIN` and exposes `X-Request-ID`.

**Planned setup:**
- **API:** Render, from `render.yaml`.
- **UI:** Vercel, with root directory `frontend` and `VITE_API_URL` set to the API URL.
- **Database:** Supabase.

The live URLs, the exact dashboard steps and the redeploy flow will be added here once it's deployed.

**Re-seeding a database** (for example after a free-tier database is reset): run `make migrate-up` and then `make seed` against it.

## 6. Running tests and coverage

| What | Command (from `backend/`) |
|---|---|
| All backend tests | `make test`: migrates the test database, then runs every test. Needs `TEST_DATABASE_URL`; without it the database tests are skipped |
| Rules only, no database | `go test ./internal/domain` |
| Coverage report | `make cover` (writes `backend/coverage.txt`, the committed summary) |
| Frontend tests, lint, type check, build | `cd frontend && npm test && npm run lint && npm run typecheck && npm run build` |

**Coverage** (backend statements, measured with `make cover`):

| Package | Coverage |
|---|---|
| **Overall backend** (`domain` + `ledger` + `httpapi`) | **87.1%** |
| **Business rules** (`domain` + `ledger`) | **89.8%** |
| `internal/domain` | 97.3% |
| `internal/ledger` | 88.3% |
| `internal/httpapi` | 84.6% |

**What is covered** (96 backend tests and 22 frontend tests):
- **Unit tests in `internal/domain`:** weighted average including 1,000-receipt rounding drift, backing a receipt out of the average, quantity, unit, reorder-level and reason validation, and the bin planner.
- **API-level tests in `backend/test/`:** every movement type, every error code, all reports, CSV exports, idempotency, stock take, request-id logging, CORS, and production-safe error logs.
- **Every edge case named in the brief:** two OUTs racing for the last unit (25 rounds per run), a transfer whose destination write fails, concurrent and repeated reversals, IN with zero or negative cost (including the database constraint on its own), 120 real receipts checked for average drift, and reconciliation after a deliberately corrupted cache.
- **After each movement test,** a check confirms the balance cache still matches the ledger.

**What is deliberately not counted:**
- `cmd/` (wiring and the seed) and `internal/postgres/sqlc` (generated) are not in the coverage figure.
- Error branches that need the database to fail mid-request are mostly untested; the transfer-failure test is the exception.
- The frontend has unit tests for its helper functions only, not component or end-to-end tests. Its flows were checked by hand in a browser.

## 7. API overview

All endpoints are under the same base URL. The full spec is in [`backend/internal/httpapi/openapi.yaml`](backend/internal/httpapi/openapi.yaml), served as Swagger UI at `/api/docs`. Every error uses the shape in section 9.

| Method | Path | Purpose | Main error codes |
|---|---|---|---|
| GET | `/healthz` | Health check including the database | 503 |
| POST | `/api/products` | Create a product | 400, 409 `DUPLICATE` |
| GET | `/api/products` | List products (`search`, `page`, `page_size`) | |
| GET | `/api/products/{id}` | One product | 400, 404 |
| PATCH | `/api/products/{id}` | Change name or reorder level (SKU and unit are fixed) | 400, 404 |
| GET | `/api/products/{id}/balances` | Stock per bin for a product | 404 |
| GET | `/api/products/{id}/ledger` | Every entry with running balances (`format=csv` for a spreadsheet) | 404 |
| POST | `/api/locations` | Create a bin (position and capacity optional) | 400, 404, 409 `DUPLICATE` |
| GET | `/api/locations` | List bins with space used | |
| GET | `/api/locations/{id}` | One bin | 404 |
| GET | `/api/locations/{id}/balances` | Stock per product in a bin | 404 |
| GET | `/api/warehouses` | Warehouse layout and capacity | |
| POST | `/api/placements/preview` | Which bins a receipt would go to (posts nothing) | 400, 404, 409 `NO_SPACE` |
| POST | `/api/picks/preview` | Which bins a shipment would take from (posts nothing) | 400, 404, 409 `INSUFFICIENT_STOCK` |
| GET | `/api/inventory` | Stock grid, products × bins | |
| GET | `/api/inventory/{productId}/{locationId}` | One product in one bin | 404 |
| GET | `/api/inventory/as-of` | Stock rebuilt from the ledger at a past `date` | 400 |
| POST | `/api/movements/in` | Receive stock | 400, 404, 409 `NO_SPACE` |
| POST | `/api/movements/out` | Ship stock (from one or several bins) | 400, 404, 409 `INSUFFICIENT_STOCK` |
| POST | `/api/movements/transfer` | Move stock between two bins | 400, 404, 409 `INSUFFICIENT_STOCK`, `NO_SPACE` |
| POST | `/api/movements/adjustment` | Signed correction with a reason | 400, 404, 409 `INSUFFICIENT_STOCK`, `NO_SPACE` |
| POST | `/api/movements/{id}/reverse` | Reverse a movement | 404, 409 `ALREADY_REVERSED`, `NOT_REVERSIBLE`, `INSUFFICIENT_STOCK`, `NO_SPACE` |
| POST | `/api/stock-takes` | Counted quantities in, one adjustment per difference | 400, 404, 409 `NO_SPACE` |
| GET | `/api/movements` | History with filters (`product_id`, `location_id`, `type`, `reference`, `from`, `to`, pages; `format=csv`) | 400 |
| GET | `/api/movements/{id}` | One movement with its entries and reversal status | 404 |
| GET | `/api/reports/low-stock` | Products at or below their reorder level (`format=csv`) | |
| GET | `/api/reports/valuation` | Quantity × average cost per product and in total (`format=csv`) | |
| GET | `/api/reports/reconciliation` | Recount every bin from the ledger and report drift (`only_drift=true`) | |
| GET | `/api/reports/cost-check` | Recalculate every average from history and compare | |
| GET | `/api/reports/daily-activity` | Postings per day (`days`) | 400 |
| GET | `/api/reports/top-movers` | Most-shipped products (`days`, `limit`) | 400 |
| GET | `/api/reports/season-forecast` | Current stock against the same weeks last year (`days`) | 400 |

All POST movement endpoints accept an optional `Idempotency-Key` header. A retry with the same key returns the original movement with `200` and `Idempotent-Replayed: true`. A key reused for a different request gets `409 IDEMPOTENCY_KEY_REUSED`. Any endpoint can also return `500 INTERNAL`.

## 8. Business rules and assumptions

**Rules from the brief**

| # | Rule | Enforced in | Tested by |
|---|---|---|---|
| 1 | Movements are immutable; mistakes are fixed by a reversal that references the original, which stays visible and marked reversed | No update or delete route exists (`httpapi/server.go`) and no SQL updates or deletes movements. `ledger/reverse.go` `Reverse` writes a new REVERSAL with inverse entries. `domain.CheckReversible`. `UNIQUE(reversal_of_id)` in the schema | `TestReversal`, `TestReversal_RejectsNegativeResultingStock`, `TestReversal_OfReceiptRestoresAverage`, `TestEdge_ConcurrentReversalsOfTheSameMovement`, `TestShip_ReversingAMultiBinShipmentRestoresEachBin` |
| 2 | Balances are derived from movements; a cache is allowed if a reconcile endpoint recomputes and reports drift | `ledger/save.go` `saveMovement` writes entries and the cached balance in one transaction. `ListReconciliation` (`db/queries/reports.sql`) behind `GET /api/reports/reconciliation` | `TestReconcile_DetectsCorruptedCache`, `TestStockAsOf_RebuildsPastBalanceFromLedger`, and the ledger-matches-cache check after every movement test |
| 3 | Stock never goes negative, including under concurrent requests | `ledger/save.go`: `lockProduct` (row lock) and `takeStock` (`UPDATE … WHERE quantity >= amount`, then checking the affected row count). `CHECK (quantity >= 0)` on `stock_balances` | `TestOut_RejectsInsufficientStock`, `TestOut_ConcurrentRequestsCannotGoNegative`, `TestEdge_TwoOutsRacingForTheLastUnit`, `TestShip_ConcurrentPicksAcrossBinsNeverGoNegative`, `TestTransfer_RejectsInsufficientStock`, and the CI load test |
| 4 | A TRANSFER is atomic and appears as one movement | `ledger/transfer.go` `Transfer`: one transaction, one movement row, two entries | `TestTransfer_MovesStockAsOneMovement`, `TestTransfer_DestinationFailureRollsBackEverything`, `TestEdge_FailedTransferLeavesBothBinsUnchanged` |
| 5 | Weighted-average cost; OUT valued at the average at that moment, stored on the movement | `domain/costing.go` `WeightedAverage`, applied in `ledger/receive.go`. `ledger/ship.go` stores the current average as the movement's `unit_cost` | `TestWeightedAverage`, `TestWeightedAverage_RoundingAfterManyReceipts`, `TestIn_UpdatesBalanceLedgerAndAverage`, `TestOut_FreezesCostAtPosting`, `TestEdge_AverageCostStaysAccurateAfterManyReceiptsThroughTheAPI`, `TestCostCheck_MatchesAfterEveryMovementType` |
| 6 | ADJUSTMENT needs a reason from a fixed list ("other" needs a note) and may be positive or negative | `domain/rules.go` `AdjustmentReasons` and `ReasonsNeedingNote`, checked in `ledger/adjust.go` `Validate`. The schema has a reason `CHECK` and `movements_other_reason_needs_note` | `TestAdjustment`, `TestValidReason`, `TestMovements_RejectBadInputBeforeDatabase` |
| 7 | A product at or below its reorder level (total across all bins) is in the low-stock report | `ListLowStockProducts` (`db/queries/reports.sql`): `HAVING total <= reorder_level`, including products with no stock at all | `TestLowStock_AtOrBelowReorderLevel` |
| – | IN needs a positive quantity and a positive unit cost | `domain.ValidateQuantity` via `ledger/receive.go` `Validate`; `movements_type_shape` `CHECK` (`unit_cost > 0` for IN) | `TestMovements_RejectBadInputBeforeDatabase`, `TestEdge_DatabaseRejectsZeroCostReceiptEvenWithoutTheAPI` |

**Assumptions** (decisions on points the brief left open)

1. **Units:** `pcs` must be whole numbers. `kg` and `litre` allow up to 4 decimal places.
2. **Precision:** quantities and money are stored as `NUMERIC(18,4)` and calculated with exact decimals. The API returns them as strings, for example `"12.5"`.
3. **Size limit:** a single quantity or cost is at most 1,000,000,000. Larger values are rejected (400).
4. **Rounding:** the weighted average is rounded to 4 decimal places, half away from zero, after each receipt. Over 1,000 receipts the drift measured about ₹0.00013; tests allow at most ₹0.001.
5. **Receiving into empty stock:** when stock is zero, the receipt's own cost becomes the average. When stock falls to zero, the last average is kept until the next receipt.
6. **Valuation of other movements:** OUT, TRANSFER and ADJUSTMENT are valued at the current average, which is stored on the movement. Only IN, and the reversal of an IN, change the average. A positive adjustment is valued at the current average.
7. **Reversal is its own type:** the brief lists four types plus `reverses_movement_id`. A reversal is stored as type `REVERSAL` with `reversal_of_id`, returned as `reverses_movement_id`. The original movement is never changed; "reversed" is worked out from the link.
8. **No reversing a reversal.** To undo a reversal, post a new movement.
9. **Reversing an IN** also removes that receipt's cost from the average. It is refused (`NOT_REVERSIBLE`) if the receipt's stock has already been used, because removing it would make the remaining value negative.
10. **A reversal must not drive any bin negative** (`INSUFFICIENT_STOCK`).
11. **Reason codes:** "count correction" is stored as `count_correction`. "other" needs a note that isn't blank.
12. **Opening stock** is recorded as an IN with reference "Opening balance". There is no direct edit of quantities.
13. **Concurrency:**
    - Every posting locks the product row first, so postings for one product happen one at a time.
    - Space checks lock the warehouse row.
    - Locks are always taken in the same order (product, then warehouses by id; stock-take products by id) to avoid deadlocks.
14. **Retries:** `Idempotency-Key` is optional. A replay is matched on the movement type and product (for a reversal, on the reversed movement), not on every field.
15. **Time:**
    - Timestamps are stored and returned in UTC and shown in the UI in IST (Asia/Kolkata).
    - Plain dates in filters (`from`, `to`, `date`) are read as IST days, and `to` includes the whole day.
    - No rule depends on elapsed time, so there is no test clock (section 10).
16. **No authentication.** `created_by` is free text entered by the user (the UI remembers it). Authentication was not part of the Core requirements.
17. **Bin capacity (extension):**
    - Each unit of any product (pcs, kg or litre) takes one unit of bin space. This is a deliberate simplification.
    - pcs are placed in whole pieces.
    - A bin without a capacity is unlimited.
    - DOCK is manual-only: automatic placement never chooses it.
    - Migration 000007 creates a permissive default warehouse (radius 1000 m, no capacity). The seed sets the demo layout (radius 50 m, capacity 2000).
18. **Shipping from several bins:** a shipment that one bin can't cover is picked from the nearest bins holding the product, as **one** OUT movement with one entry per bin.
19. **Stock take:** posts one `count_correction` adjustment per difference, all or nothing. Submitting the same count again posts nothing.
20. **Reports only report:** reconciliation and the cost check never change data. Quantity reconciliation can't prove the average cost is right, so the cost check exists separately.
21. **Lists:** products and movement history are paginated (default 20, maximum 100). Bin and warehouse lists aren't, as a warehouse has few bins. A CSV export returns up to 100,000 rows.
22. **Codes:** SKUs and bin codes are unique, at most 64 characters, and bin codes can't contain spaces.

## 9. Error handling and logging

**Error response shape** (always the same):
```json
{"error": {"code": "INSUFFICIENT_STOCK",
           "message": "insufficient stock at A-01: available 85, requested 120",
           "fields": {"quantity": "must be greater than zero"},
           "details": {"available": "85", "requested": "120"}}}
```
`fields` is only present on validation errors, and `details` on stock and space errors.

**How errors are told apart** (`backend/internal/httpapi/json.go` `writeError` is the only place that maps errors to HTTP codes):

| Kind | Examples | HTTP |
|---|---|---|
| Bad request | malformed JSON, unknown field, bad id in the path | 400 `BAD_REQUEST` |
| Validation | missing or invalid fields, one message per field | 400 `VALIDATION_ERROR` |
| Too large | a result too big for the database column | 400 `OUT_OF_RANGE` |
| Not found | unknown product, bin, warehouse or movement | 404 `NOT_FOUND` |
| Domain rule | `INSUFFICIENT_STOCK`, `NO_SPACE`, `ALREADY_REVERSED`, `NOT_REVERSIBLE`, `DUPLICATE`, `IDEMPOTENCY_KEY_REUSED` | 409 |
| Unexpected | database down, a bug, a panic | 500 `INTERNAL` with a generic message. The cause is logged, never sent to the client, and there are no stack traces |

Domain errors are separate Go types in `internal/domain/errors.go`. Every stock operation runs in one transaction, so a failure part-way leaves no partial writes. The UI shows the message next to the form, with the available amount for stock errors.

**Logging:**
- Logs are structured JSON from Go's `log/slog`, written to standard output. Each line has a time, a `level` and a `msg`.
- Every request gets a **request id**: generated, or taken from a safe client `X-Request-ID`. It's returned in the `X-Request-ID` response header and appears on every log line for that request.

| Event | Level | Fields |
|---|---|---|
| Every request | INFO | `request_id`, method, path, status, duration |
| Movement posted | INFO | `request_id`, `movement_id`, type, `product_id` |
| Rule violation or bad input | WARN | `request_id`, path, reason, plus the domain ids involved (`product_id`, `location_id`, `from_location_id`, `to_location_id`, `movement_id`, quantity) |
| Unexpected error or panic | ERROR | `request_id`, path, and the real error. With `APP_ENV=production`, only the error kind (for a database error, its Postgres code and constraint name) |

Request bodies, headers and secrets are never logged. **Where to see logs:** in the terminal running the API. There's no hosted instance yet, so there's no hosted log viewer (section 11).

## 10. Reviewer aids

- **Test clock:** not applicable. No business rule depends on elapsed time (no due dates or expiry). The season forecast compares against the same calendar window last year, and the seed creates matching data dated a year back.
- **Re-seed:** `make seed` adds whatever is missing and can be rerun. For a clean start, recreate the database, then run `make migrate-up` and `make seed`.
- **Seed data:**
  - 1 warehouse and 6 bins (A-01, A-02, B-01, B-02, COLD-01 and a manual-only DOCK).
  - 8 products (rice, wheat flour, sugar, toor dal, sunflower oil, salt, tea, ghee) with opening stock.
  - RICE-25 and DAL-TOOR also get one receipt and one sale dated a year ago, so the season forecast shows a gap. That sale stays inside the forecast window for about 20 days after seeding.
- **Swagger UI** at `/api/docs` lists every endpoint, field and error code.
- **Trying the edge cases:**
  - Concurrency: `cd backend && make load-test` fires concurrent requests at a running API. It writes data, so use a scratch database.
  - Corrupting the cache for reconciliation: `UPDATE stock_balances SET quantity = quantity + 1 WHERE product_id = 1 AND location_id = 1;` then open Reports → Reconciliation.
  - Each edge case from the brief has a named test (section 8).
- **Correlating logs:** copy the `X-Request-ID` from a response (browser DevTools → Network) and search the logs for it.
- **Backend walkthrough:** [`docs/backend-guide.md`](docs/backend-guide.md) explains the folders, one request traced end to end, each stock operation's locks and failure cases, and recipes for adding a rule or an endpoint.

## 11. Known limitations

- **Not deployed yet:** there are no live URLs. The Dockerfiles, `docker-compose.yml`, `render.yaml`, `frontend/vercel.json` and CORS support are in place (section 5a).
- **No authentication;** `created_by` is free text.
- **Batch or lot tracking with expiry** (a Stretch item) is not built. Stock take and the ledger CSV export are built.
- **The Go code is checked only by `gofmt` and `go vet`,** with no stricter linter such as staticcheck. The frontend uses ESLint.
- **Frontend tests** cover helper functions only, not components or full user flows.
- **Bin capacity treats every unit as one unit of space** (1 kg = 1 litre = 1 piece). It's a simplification, not a real volume or weight model.
- **The cost check and reconciliation load all data in one pass,** which is fine at this scale. Neither can repair what it finds: a repair would have to be a separate, audited operation, which isn't built.
- **Idempotency** matches a retried key on the movement type and product only. A retry with the same key but a different quantity returns the original movement instead of an error.
- **The season forecast** is a simple comparison with last year's shipments in the same window; it doesn't forecast trends.

## 12. AI and third-party usage

- **AI assistant:** Claude Code (Anthropic) was used throughout development: planning milestones, writing and refactoring Go, SQL and React code, writing tests, reviewing edge cases, and drafting documentation. I directed the design decisions and reviewed, ran and tested the output. The full prompt log is in [PROMPTS.md](PROMPTS.md).
- **Libraries:** `pgx/v5`, `pgx-shopspring-decimal`, `shopspring/decimal`, `sqlc`, `golang-migrate` (backend); React, React Router, Vite, Tailwind CSS, lucide-react, Vitest (frontend). Swagger UI is loaded from unpkg on the docs page.
- **Design:** the UI's visual style (colours, fonts, spacing) follows the public kjbnlabs.in website. The notes are in `docs/design.md`, which is kept out of git.
