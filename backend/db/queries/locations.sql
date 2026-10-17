-- name: CreateLocation :one
INSERT INTO locations (code, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetLocationByID :one
SELECT *
FROM locations
WHERE id = $1;

-- name: GetLocationByCode :one
SELECT *
FROM locations
WHERE code = $1;

-- name: ListLocations :many
SELECT *
FROM locations
ORDER BY code;

-- name: UpdateLocationDescription :one
UPDATE locations
SET description = $2
WHERE id = $1
RETURNING *;
