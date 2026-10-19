-- Balance for one (product, location) cell. Returns a zero row when no
-- stock_balances row exists yet, as long as both product and location exist.
-- name: GetStockBalance :one
SELECT
    p.id AS product_id,
    p.sku,
    p.name AS product_name,
    p.unit,
    l.id AS location_id,
    l.code AS location_code,
    COALESCE(sb.quantity, 0)::NUMERIC(18, 4) AS quantity,
    p.average_cost
FROM products p
CROSS JOIN locations l
LEFT JOIN stock_balances sb
    ON sb.product_id = p.id
   AND sb.location_id = l.id
WHERE p.id = @product_id
  AND l.id = @location_id;

-- Full grid: every product x every location, zero where no balance row.
-- name: ListStockOverview :many
SELECT
    p.id AS product_id,
    p.sku,
    p.name AS product_name,
    p.unit,
    p.reorder_level,
    l.id AS location_id,
    l.code AS location_code,
    COALESCE(sb.quantity, 0)::NUMERIC(18, 4) AS quantity,
    p.average_cost
FROM products p
CROSS JOIN locations l
LEFT JOIN stock_balances sb
    ON sb.product_id = p.id
   AND sb.location_id = l.id
ORDER BY p.name, p.id, l.code;

-- Balances by product: one row per location that holds (or held) this product.
-- name: ListProductBalances :many
SELECT
    sb.product_id,
    sb.location_id,
    l.code AS location_code,
    sb.quantity,
    sb.updated_at
FROM stock_balances sb
JOIN locations l ON l.id = sb.location_id
WHERE sb.product_id = $1
ORDER BY l.code;

-- Total on hand for a product across all locations (cached).
-- name: GetProductTotalQuantity :one
SELECT COALESCE(SUM(quantity), 0)::NUMERIC(18, 4) AS total_quantity
FROM stock_balances
WHERE product_id = $1;

-- Balances by location: one row per product that has (or had) stock in this bin.
-- name: ListLocationBalances :many
SELECT
    sb.location_id,
    sb.product_id,
    p.sku,
    p.name AS product_name,
    p.unit,
    sb.quantity,
    p.average_cost,
    sb.updated_at
FROM stock_balances sb
JOIN products p ON p.id = sb.product_id
WHERE sb.location_id = $1
ORDER BY p.name, p.id;

-- Transaction helpers --------------------------------------------------------

-- Creates the cell if missing so later row locks have something to lock.
-- name: EnsureStockBalanceExists :exec
INSERT INTO stock_balances (product_id, location_id, quantity)
VALUES ($1, $2, 0)
ON CONFLICT (product_id, location_id) DO NOTHING;

-- name: LockStockBalance :one
SELECT product_id, location_id, quantity
FROM stock_balances
WHERE product_id = $1
  AND location_id = $2
FOR UPDATE;

-- name: IncreaseStockBalance :execrows
UPDATE stock_balances
SET quantity   = quantity + sqlc.arg('amount'),
    updated_at = NOW()
WHERE product_id = sqlc.arg('product_id')
  AND location_id = sqlc.arg('location_id');

-- Conditional decrement: zero rows affected means insufficient stock.
-- name: DecreaseStockBalanceIfAvailable :execrows
UPDATE stock_balances
SET quantity   = quantity - sqlc.arg('amount'),
    updated_at = NOW()
WHERE product_id = sqlc.arg('product_id')
  AND location_id = sqlc.arg('location_id')
  AND quantity >= sqlc.arg('amount');

-- Stock as it was at a point in time, rebuilt from the ledger.
-- Only cells with a non-zero balance at that time are returned.
-- name: ListStockAsOf :many
SELECT
    p.id AS product_id,
    p.sku,
    p.name AS product_name,
    p.unit,
    l.id AS location_id,
    l.code AS location_code,
    SUM(e.quantity_delta)::NUMERIC(18, 4) AS quantity
FROM movement_entries e
JOIN products p ON p.id = e.product_id
JOIN locations l ON l.id = e.location_id
WHERE e.created_at < sqlc.arg('as_of')
  AND (sqlc.narg('product_id')::BIGINT IS NULL
       OR e.product_id = sqlc.narg('product_id'))
  AND (sqlc.narg('location_id')::BIGINT IS NULL
       OR e.location_id = sqlc.narg('location_id'))
GROUP BY p.id, l.id
HAVING SUM(e.quantity_delta) <> 0
ORDER BY p.name, p.id, l.code;
