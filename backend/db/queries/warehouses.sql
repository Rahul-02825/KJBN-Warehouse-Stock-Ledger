-- name: ListWarehouses :many
SELECT
    w.*,
    COALESCE((
        SELECT SUM(sb.quantity)
        FROM stock_balances sb
        JOIN locations l ON l.id = sb.location_id
        WHERE l.warehouse_id = w.id
    ), 0)::NUMERIC(18, 4) AS used
FROM warehouses w
ORDER BY w.id;

-- name: GetWarehouseByID :one
SELECT *
FROM warehouses
WHERE id = $1;

-- name: GetDefaultWarehouse :one
SELECT *
FROM warehouses
ORDER BY id
LIMIT 1;

-- name: LockWarehouse :one
SELECT *
FROM warehouses
WHERE id = $1
FOR UPDATE;

-- name: UpsertWarehouse :one
INSERT INTO warehouses (code, name, x, y, radius_m, capacity)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    x = EXCLUDED.x,
    y = EXCLUDED.y,
    radius_m = EXCLUDED.radius_m,
    capacity = EXCLUDED.capacity
RETURNING *;

-- name: ListBinUsage :many
SELECT
    l.id,
    l.code,
    l.x,
    l.y,
    l.capacity,
    l.auto_place,
    COALESCE(SUM(sb.quantity), 0)::NUMERIC(18, 4) AS used,
    COALESCE(SUM(sb.quantity) FILTER (WHERE sb.product_id = sqlc.arg('product_id')), 0)::NUMERIC(18, 4) AS product_quantity
FROM locations l
LEFT JOIN stock_balances sb ON sb.location_id = l.id
WHERE l.warehouse_id = sqlc.arg('warehouse_id')
GROUP BY l.id
ORDER BY l.id;
