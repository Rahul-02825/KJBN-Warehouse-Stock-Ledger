-- name: CreateLocation :one
INSERT INTO locations (code, description, warehouse_id, x, y, capacity, auto_place)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: SetLocationLayout :exec
UPDATE locations
SET x = $2, y = $3, capacity = $4, auto_place = $5
WHERE id = $1;

-- name: GetLocationByID :one
SELECT *
FROM locations
WHERE id = $1;

-- name: GetLocationByCode :one
SELECT *
FROM locations
WHERE code = $1;

-- name: ListLocations :many
SELECT
    l.*,
    COALESCE(SUM(sb.quantity) FILTER (WHERE p.unit = 'pcs'), 0)::BIGINT AS used
FROM locations l
LEFT JOIN stock_balances sb ON sb.location_id = l.id
LEFT JOIN products p ON p.id = sb.product_id
GROUP BY l.id
ORDER BY l.code;

-- name: UpdateLocationDescription :one
UPDATE locations
SET description = $2
WHERE id = $1
RETURNING *;
