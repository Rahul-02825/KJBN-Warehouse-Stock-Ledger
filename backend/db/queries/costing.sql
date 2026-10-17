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
