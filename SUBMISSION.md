# Submission: Warehouse Stock Ledger (Problem 3)

## A. What I built

A warehouse stock ledger where stock is never edited directly. Every change is a movement (IN, OUT, TRANSFER, ADJUSTMENT or REVERSAL) and every balance can be traced back to the movements behind it. All Core features are complete: products and bins, the five movement types, weighted-average costing with frozen OUT cost, no negative stock under concurrent requests, reversals, low-stock and valuation reports, and reconciliation. From the Stretch list I built stock take and the ledger CSV export, but not batch or lot tracking with expiry. On top of that I added bin capacity with automatic placement, shipping from several bins as one movement, and a pre-season stocking alert based on last year's sales.

## B. Three design decisions

**1. Cached balances with a reconciliation report**
- **Decision:** keep a `stock_balances` table updated in the same transaction as every movement.
- **Alternative:** compute every balance by summing the ledger on each read.
- **Why:** the stock checks need a row to lock and a quick `quantity >= amount` test, and the stock grid stays fast.
- **Cost:** the cache can drift from the ledger, so I had to build a reconciliation report and add a check after every movement test.

**2. Lock the product row, then update stock only if there is enough**
- **Decision:** every posting locks its product row first, then reduces stock with `UPDATE ... WHERE quantity >= amount` and checks that a row changed.
- **Alternative:** run transactions as SERIALIZABLE and retry when they conflict.
- **Why:** it is easy to explain and test, and the database can never accept an oversell.
- **Cost:** postings for the same product run one at a time, so a very busy product would slow down.

**3. A reversal is a new movement, and it can only happen once**
- **Decision:** a reversal is its own movement linked to the original with a unique `reversal_of_id`. A reversal can't be reversed, and incoming stock can't be reversed once it has been shipped.
- **Alternative:** mark the original as cancelled, or allow reversing anything.
- **Why:** history is never changed, and the unique constraint blocks two reversals of the same movement.
- **Cost:** to fix a wrong reversal, a user has to post a new movement by hand.

## C. The hardest part

Bin capacity took me the longest. My first version (`2f5f681`) placed stock automatically only for products counted in pieces, and a shipment could only take from one bin. Testing it in the UI, I had 120 litres of Dalda spread across bins but could not ship 120, because Ship only let me pick A-01. Reviewers would hit the same wall straight away. I changed capacity so every unit of any product counts toward bin space (`0e9604c`). Then I made a shipment pick from the chosen bin first and the nearest bins after it, saved as one OUT movement with one entry per bin (`586eb67`). The tricky part was keeping it a single movement while still checking and locking each bin, and making sure a reversal puts stock back in every bin it came from. `TestShip_ReversingAMultiBinShipmentRestoresEachBin` covers that.

## D. How I handle failure

| What goes wrong | What the user sees | What gets logged |
|---|---|---|
| Bad request (broken JSON, missing or invalid field) | 400 with a message for each field, shown next to the form | WARN with the request id |
| Rule broken (not enough stock, already reversed) | 409 with a clear code and message, plus the available amount for stock errors | WARN with the request id and the product and bin ids |
| Database error | 500 with a generic message. The transaction rolls back, so nothing is half saved | ERROR with the request id and the Postgres error code |
| Unexpected panic | Caught by middleware and returned as the same generic 500 | ERROR with the request id |

All four go through `writeError`, so errors share one shape and stack traces never reach the browser.

## E. Milestone log

| Milestone | Date completed | Commit |
|---|---|---|
| M0 Scaffold | 2026-10-17 | `a01998a` |
| M1 Domain model | 2026-10-17 | `99bc4e7` |
| M2 Business rules | 2026-10-17 | `2ac2748` |
| M3 API | 2026-10-19 | `0492107` |
| M4 UI | 2026-10-20 | `a3b9646` |
| M5 Deployment | 2026-10-20 | `231efe6` |
| M6 Documentation | Pending | tagged `v1.0` at submission |

## F. If I had two more days

1. **Test every failure end to end.** Stop the database in the middle of a posting, make a transaction fail halfway through a transfer, and wipe or corrupt the cached balances. Then check that nothing is half saved, the user sees a clear message, and reconciliation brings everything back in line. A ledger is only as good as its behaviour on a bad day, so this comes first.
2. **Run a longer concurrency soak test.** Fire hundreds of shipments, transfers and reversals at the same products and bins for several minutes, then confirm zero drift, no deadlocks and no negative stock. The current load test proves the race is handled. This would prove it holds under sustained pressure.
3. **Track batches and expiry dates.** Seeds, fertiliser and pesticides expire, so this is the most useful next feature for an agritech warehouse. It comes last because the two items above make what already exists more dependable.

## G. Self-assessment

| Criterion | Score | Why |
|---|---|---|
| Business logic correctness | 5 | Every Core rule is enforced in the backend and backed by the database, and each edge case in the brief has a named test, including two shipments racing for the last unit |
| Code clarity and structure | 5 | Routes, stock operations and business rules each have their own folder, and every stock operation has its own small file |
| Error and exception handling | 5 | One error shape with distinct codes for validation, rule and unexpected errors. Every operation is one transaction and no stack trace reaches the client |
| Logging and observability | 4 | Structured JSON logs with a request id on every line, domain ids on warnings, a health check and a Logs page on the hosted app |
| Data modelling and persistence | 5 | Constraints and indexes back up the key rules, migrations are in the repo, and cached balances can be checked against the ledger at any time |
| API design | 5 | Resource-based paths, correct verbs and status codes, pagination and filters on growing lists, and a full OpenAPI spec |
| UI usability | 5 | Every core flow works without instructions, with loading and error states and messages that say what went wrong |
| Testing | 4 | One command runs unit and API tests against a real database, with a committed coverage report above the 80% target |
| Deployment and operability | 5 | One-command compose, migrate and seed scripts, CI on every push, committed Render and Vercel config, and no secrets in the repo |
| Git hygiene | 5 | Milestone-tagged, conventional commits throughout. A few messages could have been more consistent |
| Documentation and write-up | 5 | The README follows the required structure and runs on a clean machine, with assumptions and limitations written down |