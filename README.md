# Warehouse Stock Ledger

**Contents:** [1 Title](#1-title) · [2 Live links](#2-live-links) · [3 Tech stack](#3-tech-stack) · [4 Architecture](#4-architecture) · [5 Running locally](#5-running-locally) · [5a Deployment](#5a-deployment) · [6 Tests and coverage](#6-running-tests-and-coverage) · [7 API](#7-api-overview) · [8 Business rules](#8-business-rules-and-assumptions) · [9 Errors and logging](#9-error-handling-and-logging) · [10 Reviewer aids](#10-reviewer-aids) · [11 Limitations](#11-known-limitations) · [12 AI usage](#12-ai-and-third-party-usage)

## 1. Title

Warehouse Stock Ledger, Problem 3. Tracks stock across warehouse bins where every change is recorded as a movement (IN, OUT, TRANSFER, ADJUSTMENT, REVERSAL), so any balance can be traced back to the movements that produced it.

## 2. Live links

| | URL |
|---|---|
| UI | https://kjbn-warehouse-stock-ledger.vercel.app |
| API base URL | https://stock-ledger-api-qa1e.onrender.com |
| API docs (Swagger) | https://stock-ledger-api-qa1e.onrender.com/api/docs |

- The hosted database is already seeded.
- The API runs on Render's free tier and sleeps after about 15 minutes idle. The first request can take 30 to 60 seconds. If the UI says it can't reach the API, wait and reload.
- There is no login, so no credentials are needed.

### Screenshots

Taken from the hosted app with the seed data. They follow the order a reviewer would test the app in.

**Dashboard and stock**

| Dashboard | Stock overview |
|---|---|
| ![Dashboard with inventory value, low-stock alerts and activity](docs/screenshots/dashboard.png) | ![Every product in every bin with average cost and value](docs/screenshots/stock-overview.png) |

**Catalogue**

| Products | Bins |
|---|---|
| ![Product list](docs/screenshots/products.png) | ![Warehouse bins](docs/screenshots/bins.png) |

**Recording movements**

| Receive (IN) | Ship (OUT) |
|---|---|
| ![Receive stock form with unit cost](docs/screenshots/receive-stock.png) | ![Ship stock form](docs/screenshots/ship-stock.png) |

| Transfer | Adjustment |
|---|---|
| ![Transfer stock between bins](docs/screenshots/transfer-stock.png) | ![Adjustment with a required reason](docs/screenshots/adjust-stock.png) |

**History and audit trail**

| Movement history | Product ledger |
|---|---|
| ![Movement history with filters, reversed and reversal rows](docs/screenshots/movement-history.png) | ![Product ledger with per-bin running balances](docs/screenshots/product-ledger.png) |

**Reports**

| Low stock | Inventory valuation |
|---|---|
| ![Products at or below reorder level](docs/screenshots/low-stock.png) | ![Quantity on hand times weighted-average cost](docs/screenshots/valuation.png) |

| Reconciliation | Cost check |
|---|---|
| ![Cached balances compared with the ledger](docs/screenshots/reconciliation.png) | ![Stored average cost compared with a replay of history](docs/screenshots/cost-check.png) |

## 3. Tech stack

| Layer | Choice | Hosting |
|---|---|---|
| Backend | Go 1.26, standard library `net/http` | Render |
| Database | PostgreSQL 16, `sqlc`, `golang-migrate` | Supabase |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS | Vercel |
| CI | GitHub Actions | |

## 4. Architecture

Diagram: [Miro board](https://miro.com/app/board/uXjVEXUb1QQ=/?share_link_id=933840249205)

```
Web app (Vercel) -> API (Render) -> Database (Supabase)

API:      Routes and handlers -> Stock operations -> Business rules
Database: Ledger, stock balances, products and bins
```

| Folder | What's inside |
|---|---|
| `backend/cmd/` | API server, seed script and load test |
| `backend/db/` | Migrations and SQL queries |
| `backend/internal/httpapi/` | Routes and handlers |
| `backend/internal/ledger/` | Stock operations |
| `backend/internal/domain/` | Business rules and error types |
| `backend/internal/postgres/` | Database connection and generated queries |
| `backend/test/` | API and integration tests |
| `frontend/` | React app |

**Where the business rules live.** Simple checks like valid quantities and the average cost formula are in `domain`. Rules that need database locks like preventing negative stock are in `ledger`.

### Data model

```mermaid
erDiagram
    warehouses ||--o{ locations : contains
    products ||--o{ movements : "is moved by"
    locations |o--o{ movements : "from bin"
    locations |o--o{ movements : "to bin"
    movements ||--|{ movement_entries : "posts"
    movements |o--o| movements : "reversed by"
    products ||--o{ movement_entries : ""
    locations ||--o{ movement_entries : ""
    products ||--o{ stock_balances : ""
    locations ||--o{ stock_balances : ""

    warehouses {
        bigint id PK
        varchar code UK
        varchar name
        float radius_m
        bigint capacity
    }
    locations {
        bigint id PK
        varchar code UK
        bigint warehouse_id FK
        text description
        float x
        float y
        bigint capacity
    }
    products {
        bigint id PK
        varchar sku UK
        varchar name
        varchar unit "pcs, kg, litre"
        numeric reorder_level
        numeric average_cost
    }
    movements {
        bigint id PK
        bigint product_id FK
        varchar movement_type "IN, OUT, TRANSFER, ADJUSTMENT, REVERSAL"
        bigint from_location_id FK
        bigint to_location_id FK
        numeric quantity
        numeric unit_cost
        varchar reason
        bigint reversal_of_id FK, UK
        varchar idempotency_key UK
        varchar created_by
        timestamptz created_at
    }
    movement_entries {
        bigint id PK
        bigint movement_id FK
        bigint product_id FK
        bigint location_id FK
        numeric quantity_delta "signed"
        numeric unit_cost
    }
    stock_balances {
        bigint product_id PK, FK
        bigint location_id PK, FK
        numeric quantity "never below 0"
    }
```

- A movement is the business event. Its entries are the stock changes it causes: one for IN, OUT and ADJUSTMENT, two for TRANSFER.
- `stock_balances` is a cache. Adding up `movement_entries` for a product and bin gives the true quantity, and reconciliation compares the two.
- A reversal is a new movement that points to the original through `reversal_of_id`. That column is unique, so a movement can be reversed only once.
- Movements and entries are never updated or deleted.

## 5. Running locally

### One command with Docker

```bash
git clone https://github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger.git
cd KJBN-Warehouse-Stock-Ledger
docker compose up --build
```

UI at http://localhost:3000, API at http://localhost:8080, Swagger at http://localhost:8080/api/docs.

Compose starts PostgreSQL, runs the migrations and the seed, then starts the API and the UI. No `.env` is needed. `docker compose down -v` wipes the data.

### Manual setup

| Tool | Version |
|---|---|
| Go | 1.26+ |
| Node.js | 22.12+ |
| Docker | any recent |
| golang-migrate CLI | 4.18+ |
| make | any |

Copy `.env.example` to `.env` in the repo root.

| Variable | Meaning |
|---|---|
| `DATABASE_URL` | PostgreSQL connection for the API, migrations and seed |
| `PORT` | API port, default 8080 |
| `TEST_DATABASE_URL` | Separate database for tests, which wipe it |
| `APP_ENV` | `production` hides raw error text in logs |
| `CORS_ALLOWED_ORIGIN` | Allowed UI origin. Empty locally |
| `VITE_API_URL` | In `frontend/.env`. Empty locally |

Start PostgreSQL and create the test database:

```bash
docker run -d --name ledger-db -e POSTGRES_USER=ledger -e POSTGRES_PASSWORD=ledger -e POSTGRES_DB=ledger -p 5432:5432 postgres:16
docker exec ledger-db createdb -U ledger ledger_test
```

Set these in `.env`:

```
DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable
TEST_DATABASE_URL=postgres://ledger:ledger@localhost:5432/ledger_test?sslmode=disable
```

Backend:

```bash
cd backend
go mod download
make migrate-up
make seed
make run
```

UI, in a second terminal:

```bash
cd frontend
npm ci
npm run dev
```

The UI runs at http://localhost:5173.

## 5a. Deployment

| Part | Host | Config in repo |
|---|---|---|
| UI | Vercel | `frontend/vercel.json` |
| API | Render | `render.yaml`, `backend/Dockerfile` |
| Database | Supabase | `backend/db/migrations`, `make seed` |

How we deployed it:

1. Created a Supabase project and copied its Session pooler connection string into `.env` as `DATABASE_URL`.
2. Ran `make migrate-up` and `make seed` from `backend/` against that database.
3. Created a Blueprint on Render from this repo, which picked up `render.yaml`, and added `DATABASE_URL` in the dashboard.
4. Imported the repo on Vercel with `frontend` as the root directory and set `VITE_API_URL` to the Render URL.
5. Added the Vercel URL as `CORS_ALLOWED_ORIGIN` on Render.

Pushing to `main` redeploys both the API and the UI. Running `make seed` again re-seeds the database and only adds what is missing.

## 6. Running tests and coverage

| | Command |
|---|---|
| Backend tests | `cd backend && make test` |
| Coverage | `cd backend && make cover`, writes `backend/coverage.txt` |
| Frontend | `cd frontend && npm test` |

| Package | Coverage |
|---|---|
| Backend overall | 87.4% |
| Business rules (`domain`, `ledger`) | 89.8% |

Covered: every business rule and edge case from the problem, including concurrent shipments, a failed transfer and corrupted balances.

Not covered: generated code and the startup code in `cmd/`, a few database failure branches that need fault injection, and frontend components, which were tested by hand.

## 7. API overview

Full spec in [`backend/internal/httpapi/openapi.yaml`](backend/internal/httpapi/openapi.yaml), served at `/api/docs`. Any endpoint can also return `500 INTERNAL`.

| Method | Path | Purpose | Main errors |
|---|---|---|---|
| GET | `/healthz` | Health check including the database | 503 |
| POST | `/api/products` | Create product | 400, 409 `DUPLICATE` |
| GET | `/api/products` | List products, paginated, `search` | |
| GET | `/api/products/{id}` | One product | 404 |
| PATCH | `/api/products/{id}` | Update name or reorder level | 400, 404 |
| GET | `/api/products/{id}/balances` | Stock per bin | 404 |
| GET | `/api/products/{id}/ledger` | Entries with running balance, `format=csv` | 404 |
| POST | `/api/locations` | Create bin | 400, 409 `DUPLICATE` |
| GET | `/api/locations` | List bins | |
| GET | `/api/locations/{id}` | One bin | 404 |
| GET | `/api/locations/{id}/balances` | Stock per product in a bin | 404 |
| GET | `/api/warehouses` | Warehouse layout and capacity | |
| POST | `/api/placements/preview` | Bins incoming stock would go to, posts nothing | 409 `NO_SPACE` |
| POST | `/api/picks/preview` | Bins a shipment would take from, posts nothing | 409 `INSUFFICIENT_STOCK` |
| GET | `/api/inventory` | Stock grid, products by bins | |
| GET | `/api/inventory/{productId}/{locationId}` | One product in one bin | 404 |
| GET | `/api/inventory/as-of` | Stock rebuilt from the ledger at a past `date` | 400 |
| POST | `/api/movements/in` | Receive stock | 400, 404, 409 `NO_SPACE` |
| POST | `/api/movements/out` | Ship stock | 400, 404, 409 `INSUFFICIENT_STOCK` |
| POST | `/api/movements/transfer` | Move stock between bins | 400, 404, 409 `INSUFFICIENT_STOCK` |
| POST | `/api/movements/adjustment` | Correction with a reason | 400, 404, 409 `INSUFFICIENT_STOCK` |
| POST | `/api/movements/{id}/reverse` | Reverse a movement | 404, 409 `ALREADY_REVERSED`, `NOT_REVERSIBLE` |
| POST | `/api/stock-takes` | Post counted quantities | 400, 404 |
| GET | `/api/movements` | History with filters, paginated, `format=csv` | 400 |
| GET | `/api/movements/{id}` | One movement with entries | 404 |
| GET | `/api/reports/low-stock` | Products at or below reorder level | |
| GET | `/api/reports/valuation` | Quantity times average cost | |
| GET | `/api/reports/reconciliation` | Cached totals vs ledger | |
| GET | `/api/reports/cost-check` | Recomputed averages vs stored | |
| GET | `/api/reports/daily-activity` | Postings per day | 400 |
| GET | `/api/reports/top-movers` | Most shipped products | 400 |
| GET | `/api/reports/season-forecast` | Stock vs same weeks last year | 400 |
| GET | `/api/logs` | Last 500 log lines | 400 |

## 8. Business rules and assumptions

| Rule | Where it's enforced | Test |
|---|---|---|
| A posted movement is never edited or deleted. Mistakes are undone with a reversal, and a movement can only be reversed once. | `ledger/reverse.go` `Reverse`, plus a unique constraint on `reversal_of_id` | `TestReversal`, `TestEdge_ConcurrentReversalsOfTheSameMovement` |
| Balances come from the ledger. The cached totals can be checked against it at any time. | `ledger/save.go` `saveMovement`, `ListReconciliation` in `db/queries/reports.sql` | `TestReconcile_DetectsCorruptedCache` |
| Stock can't go negative, even when two people ship the last unit at the same time. | `ledger/save.go` `lockProduct` and `takeStock` | `TestEdge_TwoOutsRacingForTheLastUnit` |
| A transfer moves stock out of one bin and into another, or does nothing at all. | `ledger/transfer.go` `Transfer` | `TestTransfer_DestinationFailureRollsBackEverything` |
| Incoming stock updates the weighted-average cost. A shipment keeps the cost it was valued at, even if the average changes later. | `domain/costing.go` `WeightedAverage`, `ledger/ship.go` | `TestWeightedAverage_RoundingAfterManyReceipts`, `TestOut_FreezesCostAtPosting` |
| An adjustment needs a reason, and "other" needs a note. | `domain/rules.go`, `ledger/adjust.go` | `TestAdjustment`, `TestValidReason` |
| A product is low on stock when its total across all bins is at or below its reorder level. | `ListLowStockProducts` in `db/queries/reports.sql` | `TestLowStock_AtOrBelowReorderLevel` |
| Incoming stock needs a positive quantity and a positive cost. | `domain.ValidateQuantity`, plus a check constraint | `TestEdge_DatabaseRejectsZeroCostReceiptEvenWithoutTheAPI` |

Assumptions:

1. Pieces must be whole numbers. Kilograms and litres can be fractional.
2. The average cost is rounded to 4 decimal places each time stock comes in.
3. If stock is at zero, the cost of the next incoming stock becomes the new average. If stock drops to zero, the last average is kept.
4. Only incoming stock, and reversing incoming stock, change the average cost. Everything else is valued at the current average.
5. A reversal can't be reversed. To undo one, post a new movement.
6. Incoming stock can't be reversed once it has been shipped, since removing it would leave a negative stock value.
7. A shipment that one bin can't cover is picked from several bins but recorded as one movement.
8. Every unit takes one unit of bin space, whether it's a piece, a kilogram or a litre.
9. There is no login.

## 9. Error handling and logging

Every error has the same shape:

```json
{"error": {"code": "INSUFFICIENT_STOCK",
           "message": "insufficient stock at A-01: available 85, requested 120"}}
```

| Kind | HTTP |
|---|---|
| Bad input or invalid fields | 400 |
| Unknown product, bin or movement | 404 |
| Business rule broken, like not enough stock | 409 |
| Anything unexpected | 500 with a generic message |

All errors are turned into responses in one place, `writeError` in `httpapi/json.go`. A failed operation leaves no partial changes in the database.

Logs are JSON. Every request gets a request id, which is sent back in the `X-Request-ID` header and added to each log line for that request. Requests are logged at INFO, rule violations at WARN and unexpected errors at ERROR.

On the hosted app, open Reports then Logs to see recent lines and search by request id. The full history is in the Render dashboard.

## 10. Reviewer aids

- **Re-seed:** run `make seed` from `backend/`. It is safe to run again.
- **Seed data:** 6 bins and 8 grocery products with opening stock.
- **Concurrency:** run `make load-test` from `backend/`. It sends many requests at once to a running API.
- **Reconciliation:** run `UPDATE stock_balances SET quantity = quantity + 1 WHERE product_id = 1 AND location_id = 1;` then open Reports then Reconciliation to see the mismatch.

## 11. Known limitations

- The API sleeps on the free tier and takes 30 to 60 seconds to wake up. Supabase pauses after a week of no use.
- The Logs page only keeps the last 500 lines and clears on restart.
- There is no login.
- Bin space treats 1 kg, 1 litre and 1 piece as the same size.
- Reconciliation finds mismatches but does not fix them.
- Retrying a movement with the same `Idempotency-Key` but a different quantity returns the first movement instead of an error.

## 12. AI and third-party usage

Claude Code was used for boilerplate, the frontend, tests and documentation drafts. I also used it to review business logic I wrote. Every prompt is logged in [PROMPTS.md](PROMPTS.md). Libraries are listed in `backend/go.mod` and `frontend/package.json`.
