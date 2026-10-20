# Backend Guide

This guide explains how the Go backend is organised and how to change it safely. Read sections 1 and 2 first. They are enough to find your way around.

## 1. Where code lives

```
backend/
├── cmd/api/main.go          starts the server: reads env vars, connects, graceful shutdown
├── cmd/seed/main.go         development fixtures (safe to rerun)
├── db/migrations/           schema changes (golang-migrate)
├── db/queries/*.sql         every SQL query, by topic
├── internal/
│   ├── domain/              business rules only: no database, no HTTP
│   ├── ledger/              stock operations: each one is a database transaction
│   ├── httpapi/             HTTP handlers: read the request, call ledger or sqlc, write JSON
│   └── postgres/            connect helper, Postgres error checks, generated sqlc code
└── test/                    end-to-end tests against a real PostgreSQL
```

**Rule of thumb when you add code:**

| You are adding… | Put it in |
|---|---|
| A rule that doesn't need the database, such as a limit, an allowed value, or a formula | `internal/domain/` |
| Anything that changes stock or the average cost | `internal/ledger/` |
| A new URL, query-string parsing, or response shape | `internal/httpapi/` |
| A new SQL query | `db/queries/*.sql`, then run `sqlc generate` |

**The one rule never to break:** stock changes only through `ledger`. `ledger` writes the movement, its entries and the cached balance together, in one transaction.

### `domain/` at a glance

| File | Contains |
|---|---|
| `rules.go` | Every tunable rule: units, whole-number units, movement types, adjustment reasons, reasons that need a note, 4-decimal precision, max amount, stock-take limits |
| `validate.go` | Small checks built on those lists, such as `ValidateQuantity`, `ValidReason` and `CheckReversible` |
| `costing.go` | `WeightedAverage` and `BackOutAverage` |
| `errors.go` | Every business error the API can return |
| `placement.go`, `layout.go`, `units.go` | The nearest-bin planner (min-heap), the bin perimeter and capacity check, and quantity ↔ space units |

Error messages are built from the lists. Adding `"spoiled"` to `AdjustmentReasons` also updates the "must be one of …" message.

### `ledger/` at a glance

| File | Operation |
|---|---|
| `receive.go` | IN |
| `ship.go` | OUT, from one or more bins |
| `transfer.go` | TRANSFER |
| `adjust.go` | ADJUSTMENT |
| `reverse.go` | REVERSAL |
| `stocktake.go` | Stock take: one adjustment per difference |
| `save.go` | `saveMovement`: updates the balances, then writes the movement row and the ledger entries. Also `addStock`, `takeStock`, `lockProduct` |
| `bins.go` | Bin space checks, where to put a receipt, which bins to pick from |
| `preview.go` | The "Before you post" plans |
| `costcheck.go` | Replays every movement to re-check the average cost |
| `replay.go` | Idempotency-Key replays |

Each operation file holds the request struct, its `Validate` method and the operation itself.

### `httpapi/` at a glance

- `server.go` lists **every route in one place**.
- Every handler does the same four steps:

```go
func (s *Server) shipStock(w http.ResponseWriter, r *http.Request) {
	key, err := readKey(r)                         // 1. read headers / path values
	...
	var req ledger.ShipRequest
	if err := readJSON(w, r, &req); err != nil {   // 2. read the JSON body
		writeError(w, r, err)
		return
	}
	req.IdempotencyKey = key
	movement, replayed, err := s.ledger.Ship(r.Context(), req)   // 3. do the work
	writeMovement(w, r, movement, replayed, err)                 // 4. write JSON
}
```

`writeError` in `json.go` is the **only** place that turns errors into HTTP status codes:

| Error | Status | Code |
|---|---|---|
| bad JSON or bad path id | 400 | `BAD_REQUEST` |
| `domain.ValidationError` | 400 | `VALIDATION_ERROR`, plus `fields` |
| `domain.NotFoundError` | 404 | `NOT_FOUND` |
| `domain.InsufficientStockError` | 409 | `INSUFFICIENT_STOCK`, plus `details` |
| `domain.NoSpaceError` | 409 | `NO_SPACE`, plus `details` |
| `domain.ErrAlreadyReversed` | 409 | `ALREADY_REVERSED` |
| `domain.NotReversibleError` | 409 | `NOT_REVERSIBLE` |
| `domain.ErrKeyReused` | 409 | `IDEMPOTENCY_KEY_REUSED` |
| numeric overflow from Postgres | 400 | `OUT_OF_RANGE` |
| anything else | 500 | `INTERNAL`. The real error is logged, never sent to the client |

## 2. How one request flows

`POST /api/movements/out` with `{"product_id": 1, "quantity": "90", "created_by": "Asha"}`:

1. **`httpapi.shipStock`** reads the optional `Idempotency-Key` header and the JSON body.
2. **`ledger.Ship`** calls `req.Validate()`. A bad field returns a `ValidationError`, which becomes a 400.
3. **`ledger.ship`** opens a transaction and:
   1. Locks the product row (`SELECT … FOR UPDATE`). Every posting for this product now waits its turn.
   2. Finds the warehouse and plans which bins to pick from (`planPick`). It starts at the chosen bin, or the bin holding the most stock, then moves to the nearest other bins that hold the product.
   3. Calls `saveMovement`, which decreases each bin with `UPDATE … WHERE quantity >= amount`. Zero rows updated means `InsufficientStockError`. It then writes one `movements` row and one `movement_entries` row per bin. The unit cost written is the **current average**, frozen on the movement.
   4. Commits. Any error before this point rolls everything back (`defer tx.Rollback`).
4. **`httpapi.writeMovement`** returns 201 with the movement, or 200 plus `Idempotent-Replayed: true` for a retried key.

## 3. The stock operations

| Operation | Locks | Steps | Can fail with |
|---|---|---|---|
| **Receive (IN)** | product, then warehouse | Plan bins (chosen bin, else the bin with most of the product, else the bin nearest the entrance; overflow goes to the nearest bins), compute the new average from the **old total quantity**, save, update the average | validation, not found, `NO_SPACE` |
| **Ship (OUT)** | product | Plan picks across bins, save with the average frozen as the unit cost | `INSUFFICIENT_STOCK` (with the available amount) |
| **Transfer** | product, then warehouse when space is checked | Two changes: −source, +destination. One movement, two entries. The average is unchanged | `INSUFFICIENT_STOCK`, `NO_SPACE`, same-bin validation |
| **Adjust** | product | One signed change, valued at the current average. The average is unchanged | negative stock (`INSUFFICIENT_STOCK`), bad reason, missing note |
| **Reverse** | the original movement, then product | Refuse if it's a reversal or already reversed, then flip every entry's sign and save. If the original was an IN, also back its cost out of the average | `ALREADY_REVERSED`, `NOT_REVERSIBLE`, `INSUFFICIENT_STOCK` |
| **Stock take** | product (in id order), per counted line | For each line, difference = counted − cached balance. Skip zeros, post a `count_correction` adjustment for the rest. All lines succeed or none do | anything an adjustment can fail with |

**Why concurrency is safe:**
- The product lock serialises postings for the same product.
- The conditional `UPDATE … quantity >= amount` is a second guard against negative stock.
- The warehouse lock stops two receipts overfilling the same space.
- Locks are always taken in the same order: product first, then warehouses by id, and stock-take products by id. That avoids deadlocks.
- `UNIQUE(reversal_of_id)` in the database is the last guard against double reversal.

**Costing:**
- New average = (old qty × old avg + qty × cost) ÷ (old qty + qty).
- When the old qty is 0, the receipt's own cost is used.
- Averages are rounded to 4 decimals, half away from zero.
- OUT, transfer and adjustment never change the average.
- Reversing an IN backs the receipt out. If its stock is already gone, the reversal is refused with `NOT_REVERSIBLE`.

## 4. Recipes

### Add or change a business rule

1. Find the rule in `internal/domain/rules.go` or `validate.go`. Most rules are a list or a constant.
2. Change it.
3. Add a case to the matching `_test.go` in the same folder, then run `go test ./internal/domain`. It takes under a second and needs no database.
4. If the rule needs the database, for example "bin X can't receive", put the check inside the ledger operation after the row is loaded, and return a domain error.

### Add a new error

1. Add the type or variable to `internal/domain/errors.go`.
2. Add one `case` to `writeError` in `internal/httpapi/json.go` with its status and code.

### Add a read-only endpoint

1. Write the query in `db/queries/<topic>.sql` with a `-- name: MyQuery :many` line.
2. Run `make sqlc` (or `sqlc generate`). This creates `s.q.MyQuery(...)`.
3. Write a handler in `internal/httpapi/<topic>.go` following the 4 steps.
4. Register it in `server.go`: `mux.HandleFunc("GET /api/…", s.myHandler)`.
5. Add a test in `backend/test/` and document it in `internal/httpapi/openapi.yaml`.

### Add an endpoint that changes stock

As above, except step 3 calls a new `ledger` function. Copy `adjust.go` as a template:
- request struct and `Validate`
- open a transaction, `defer tx.Rollback`
- `lockProduct`
- load what you need
- build the changes
- `saveMovement`
- commit

## 5. Practice extensions

These are worked on paper. None of them is in the code.

**A. A new adjustment reason, `spoiled`, that needs a note.**
- In `domain/rules.go`, add `"spoiled"` to `AdjustmentReasons` and to `ReasonsNeedingNote`.
- The database also checks reasons, so add a migration that updates the CHECK constraint on `movements.reason`. If you want the database to enforce the note too, extend `movements_other_reason_needs_note` in the same migration.
- Test it by posting an adjustment with reason `spoiled` and no note. The response should be 400 with `fields.note`.

**B. No single shipment above 500 units.**
- In `domain/rules.go`: `var MaxShipment = decimal.NewFromInt(500)`.
- In `ledger/ship.go` `Validate`:

```go
if r.Quantity.GreaterThan(domain.MaxShipment) {
	fields["quantity"] = "must not exceed " + domain.MaxShipment.String() + " per shipment"
}
```

**C. Nothing may be transferred into DOCK.**
- In `ledger/transfer.go`, right after `to, err := getLocation(...)`:

```go
if to.Code == "DOCK" {
	return sqlc.Movement{}, domain.ValidationError{"to_location_id": "stock cannot be transferred into DOCK"}
}
```

**D. Allow reversing a reversal.**
- Remove the check in `domain.CheckReversible`.
- The catch is cost. Reversing a "reversal of an IN" puts the stock back, but `reverse.go` only adjusts the average when the original type is `IN`. You would also have to re-apply the receipt with `WeightedAverage`, or the cost check would report a mismatch.
- That is why the rule is "a reversal cannot be reversed; post a new movement instead".

## 6. The few spots that aren't plain Go

| Where | What it is |
|---|---|
| `httpapi/middleware.go` | Standard `net/http` middleware: functions that wrap a handler. `Recover` turns a panic into a 500 |
| `postgres/connect.go` | `AfterConnect` registers the decimal type on every new database connection, so numbers arrive as `decimal.Decimal` |
| `domain/placement.go` | `binHeap` implements Go's `container/heap` interface (`Len`, `Less`, `Swap`, `Push`, `Pop`). It always pops the nearest bin, with ties broken by bin code |
| `postgres/sqlc/` | Generated by `sqlc generate`. Never edit by hand |

## 7. Commands

```bash
cd backend
make migrate-up                      # apply migrations (uses ../.env)
make seed                            # fixtures; safe to rerun
make run                             # API on :8080
sqlc generate                        # after editing db/queries/*.sql
go test ./internal/domain            # rules only, no database
make test                            # everything; needs TEST_DATABASE_URL
gofmt -l . && go vet ./...
```
