-- Products whose total cached balance is at or below their reorder level.
-- name: ListLowStockProducts :many
SELECT
    p.id AS product_id,
    p.sku,
    p.name,
    p.unit,
    p.reorder_level,
    COALESCE(SUM(sb.quantity), 0)::NUMERIC(18, 4) AS total_quantity,
    (p.reorder_level - COALESCE(SUM(sb.quantity), 0))::NUMERIC(18, 4) AS shortfall
FROM products p
LEFT JOIN stock_balances sb ON sb.product_id = p.id
GROUP BY p.id
HAVING COALESCE(SUM(sb.quantity), 0) <= p.reorder_level
ORDER BY shortfall DESC, p.name;

-- Quantity x average cost per product.
-- name: GetInventoryValuation :many
SELECT
    p.id AS product_id,
    p.sku,
    p.name,
    p.unit,
    COALESCE(SUM(sb.quantity), 0)::NUMERIC(18, 4) AS total_quantity,
    p.average_cost,
    (COALESCE(SUM(sb.quantity), 0) * p.average_cost)::NUMERIC(24, 4) AS total_value
FROM products p
LEFT JOIN stock_balances sb ON sb.product_id = p.id
GROUP BY p.id
ORDER BY p.name, p.id;

-- name: GetTotalInventoryValue :one
SELECT COALESCE(SUM(sb.quantity * p.average_cost), 0)::NUMERIC(24, 4) AS total_value
FROM stock_balances sb
JOIN products p ON p.id = sb.product_id;

-- Reconciliation: recompute every (product, location) balance from the ledger
-- and compare with the cache. Covers cells present on either side. Read-only.
-- name: ListReconciliation :many
WITH ledger_totals AS (
    SELECT
        product_id,
        location_id,
        SUM(quantity_delta)::NUMERIC(18, 4) AS ledger_quantity
    FROM movement_entries
    GROUP BY product_id, location_id
),
cells AS (
    SELECT product_id, location_id FROM stock_balances
    UNION
    SELECT product_id, location_id FROM ledger_totals
)
SELECT
    p.id AS product_id,
    p.sku,
    p.name AS product_name,
    l.id AS location_id,
    l.code AS location_code,
    COALESCE(sb.quantity, 0)::NUMERIC(18, 4) AS cached_quantity,
    COALESCE(lt.ledger_quantity, 0)::NUMERIC(18, 4) AS ledger_quantity,
    (COALESCE(sb.quantity, 0) - COALESCE(lt.ledger_quantity, 0))::NUMERIC(18, 4) AS difference
FROM cells c
JOIN products p ON p.id = c.product_id
JOIN locations l ON l.id = c.location_id
LEFT JOIN stock_balances sb
    ON sb.product_id = c.product_id
   AND sb.location_id = c.location_id
LEFT JOIN ledger_totals lt
    ON lt.product_id = c.product_id
   AND lt.location_id = c.location_id
WHERE (sqlc.arg('only_drift')::BOOLEAN = FALSE
       OR COALESCE(sb.quantity, 0) <> COALESCE(lt.ledger_quantity, 0))
ORDER BY (COALESCE(sb.quantity, 0) <> COALESCE(lt.ledger_quantity, 0)) DESC,
         p.sku, l.code;
