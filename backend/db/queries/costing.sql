-- Serialises every cost-affecting posting for a product. Must be taken
-- before reading quantities for the weighted-average calculation.
-- name: LockProductForCostUpdate :one
SELECT *
FROM products
WHERE id = $1
FOR UPDATE;

-- name: UpdateProductAverageCost :one
UPDATE products
SET average_cost = $2,
    updated_at   = NOW()
WHERE id = $1
RETURNING id, sku, average_cost, updated_at;

-- Ledger-derived total for a product. Used by costing consistency checks so
-- that the average is never silently computed from a drifted cache.
-- name: GetLedgerProductTotalQuantity :one
SELECT COALESCE(SUM(quantity_delta), 0)::NUMERIC(18, 4) AS total_quantity
FROM movement_entries
WHERE product_id = $1;

-- Every movement in posting order, for replaying the average cost.
-- net_delta is the movement's total stock effect across all bins.
-- name: ListMovementsForCostCheck :many
SELECT
    m.id,
    m.product_id,
    m.movement_type,
    m.quantity,
    m.unit_cost,
    COALESCE(o.movement_type, '')::VARCHAR AS reversed_type,
    (SELECT COALESCE(SUM(e.quantity_delta), 0)
     FROM movement_entries e
     WHERE e.movement_id = m.id)::NUMERIC(18, 4) AS net_delta
FROM movements m
LEFT JOIN movements o ON o.id = m.reversal_of_id
ORDER BY m.product_id, m.created_at, m.id;

-- name: ListProductCosts :many
SELECT id, sku, name, average_cost
FROM products
ORDER BY sku;
