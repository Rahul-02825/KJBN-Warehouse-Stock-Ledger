-- name: CreateProduct :one
INSERT INTO products (sku, name, unit, reorder_level)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProductByID :one
SELECT *
FROM products
WHERE id = $1;

-- name: GetProductBySKU :one
SELECT *
FROM products
WHERE sku = $1;

-- name: ListProducts :many
SELECT *
FROM products
WHERE (sqlc.narg('search')::TEXT IS NULL
       OR sku ILIKE '%' || sqlc.narg('search') || '%'
       OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY name, id
LIMIT sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- name: CountProducts :one
SELECT COUNT(*)
FROM products
WHERE (sqlc.narg('search')::TEXT IS NULL
       OR sku ILIKE '%' || sqlc.narg('search') || '%'
       OR name ILIKE '%' || sqlc.narg('search') || '%');

-- Partial update: NULL arguments leave the column unchanged.
-- sku and unit are intentionally not updatable once movements may exist.
-- name: UpdateProductDetails :one
UPDATE products
SET name          = COALESCE(sqlc.narg('name'), name),
    reorder_level = COALESCE(sqlc.narg('reorder_level'), reorder_level),
    updated_at    = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;
