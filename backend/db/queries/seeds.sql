-- Used only by cmd/seed to record a few fictional movements with past dates.
-- Normal postings always take created_at from NOW().

-- name: CreateBackdatedMovement :one
INSERT INTO movements (
    product_id,
    movement_type,
    from_location_id,
    to_location_id,
    quantity,
    unit_cost,
    reference,
    created_by,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: CreateBackdatedMovementEntry :exec
INSERT INTO movement_entries (
    movement_id,
    product_id,
    location_id,
    quantity_delta,
    unit_cost,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6);
