-- name: CreateMovement :one
INSERT INTO movements (
    product_id,
    movement_type,
    from_location_id,
    to_location_id,
    quantity,
    unit_cost,
    reference,
    reason,
    note,
    created_by,
    reversal_of_id
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: CreateMovementEntry :one
INSERT INTO movement_entries (
    movement_id,
    product_id,
    location_id,
    quantity_delta,
    unit_cost
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMovementByID :one
SELECT *
FROM movements
WHERE id = $1;

-- Single movement with display names and reversal status.
-- name: GetMovementDetail :one
SELECT
    m.*,
    p.sku,
    p.name AS product_name,
    p.unit,
    fl.code AS from_location_code,
    tl.code AS to_location_code,
    rev.id  AS reversed_by_id,
    (rev.id IS NOT NULL) AS is_reversed
FROM movements m
JOIN products p ON p.id = m.product_id
LEFT JOIN locations fl ON fl.id = m.from_location_id
LEFT JOIN locations tl ON tl.id = m.to_location_id
LEFT JOIN movements rev ON rev.reversal_of_id = m.id
WHERE m.id = $1;

-- name: ListMovementEntries :many
SELECT *
FROM movement_entries
WHERE movement_id = $1
ORDER BY id;

-- History with optional filters and offset pagination.
-- to_date is exclusive so callers can pass "next day 00:00" for a date range.
-- name: ListMovementHistory :many
SELECT
    m.*,
    p.sku,
    p.name AS product_name,
    p.unit,
    fl.code AS from_location_code,
    tl.code AS to_location_code,
    rev.id  AS reversed_by_id,
    (rev.id IS NOT NULL) AS is_reversed
FROM movements m
JOIN products p ON p.id = m.product_id
LEFT JOIN locations fl ON fl.id = m.from_location_id
LEFT JOIN locations tl ON tl.id = m.to_location_id
LEFT JOIN movements rev ON rev.reversal_of_id = m.id
WHERE (sqlc.narg('product_id')::BIGINT IS NULL
       OR m.product_id = sqlc.narg('product_id'))
  AND (sqlc.narg('movement_type')::VARCHAR IS NULL
       OR m.movement_type = sqlc.narg('movement_type'))
  AND (sqlc.narg('location_id')::BIGINT IS NULL
       OR m.from_location_id = sqlc.narg('location_id')
       OR m.to_location_id = sqlc.narg('location_id'))
  AND (sqlc.narg('reference')::TEXT IS NULL
       OR m.reference ILIKE '%' || sqlc.narg('reference') || '%')
  AND (sqlc.narg('from_date')::TIMESTAMPTZ IS NULL
       OR m.created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::TIMESTAMPTZ IS NULL
       OR m.created_at < sqlc.narg('to_date'))
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg('page_size')
OFFSET sqlc.arg('page_offset');

-- Same filters as ListMovementHistory, for the pagination total.
-- name: CountMovementHistory :one
SELECT COUNT(*)
FROM movements m
WHERE (sqlc.narg('product_id')::BIGINT IS NULL
       OR m.product_id = sqlc.narg('product_id'))
  AND (sqlc.narg('movement_type')::VARCHAR IS NULL
       OR m.movement_type = sqlc.narg('movement_type'))
  AND (sqlc.narg('location_id')::BIGINT IS NULL
       OR m.from_location_id = sqlc.narg('location_id')
       OR m.to_location_id = sqlc.narg('location_id'))
  AND (sqlc.narg('reference')::TEXT IS NULL
       OR m.reference ILIKE '%' || sqlc.narg('reference') || '%')
  AND (sqlc.narg('from_date')::TIMESTAMPTZ IS NULL
       OR m.created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::TIMESTAMPTZ IS NULL
       OR m.created_at < sqlc.narg('to_date'));

-- Product ledger view: every ledger entry for a product, oldest first, with
-- the running total across all locations and the running balance of the
-- entry's own location. Ordered by (created_at, movement id, entry id) so a
-- transfer's two entries stay adjacent and deterministic.
-- name: GetProductLedger :many
SELECT
    e.id AS entry_id,
    m.id AS movement_id,
    m.created_at,
    m.movement_type,
    m.reference,
    m.reason,
    m.note,
    m.created_by,
    m.reversal_of_id,
    e.location_id,
    l.code AS location_code,
    e.quantity_delta,
    e.unit_cost,
    SUM(e.quantity_delta) OVER (
        ORDER BY m.created_at, m.id, e.id
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::NUMERIC(18, 4) AS running_total,
    SUM(e.quantity_delta) OVER (
        PARTITION BY e.location_id
        ORDER BY m.created_at, m.id, e.id
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::NUMERIC(18, 4) AS running_location_balance
FROM movement_entries e
JOIN movements m ON m.id = e.movement_id
JOIN locations l ON l.id = e.location_id
WHERE e.product_id = $1
ORDER BY m.created_at, m.id, e.id;

-- Ledger restricted to one bin.
-- name: GetProductLocationLedger :many
SELECT
    e.id AS entry_id,
    m.id AS movement_id,
    m.created_at,
    m.movement_type,
    m.reference,
    m.reason,
    m.note,
    m.created_by,
    m.reversal_of_id,
    e.quantity_delta,
    e.unit_cost,
    SUM(e.quantity_delta) OVER (
        ORDER BY m.created_at, m.id, e.id
        ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::NUMERIC(18, 4) AS running_balance
FROM movement_entries e
JOIN movements m ON m.id = e.movement_id
WHERE e.product_id = $1
  AND e.location_id = $2
ORDER BY m.created_at, m.id, e.id;
