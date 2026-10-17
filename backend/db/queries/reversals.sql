-- Lock the original so two concurrent reversal attempts serialise; the
-- UNIQUE constraint on reversal_of_id is the final guard.
-- name: LockMovementForReversal :one
SELECT *
FROM movements
WHERE id = $1
FOR UPDATE;

-- name: GetReversalForMovement :one
SELECT *
FROM movements
WHERE reversal_of_id = $1;

-- Entries of the original, ordered by location so balance locks are always
-- acquired in the same order (deadlock avoidance).
-- name: GetEntriesForReversal :many
SELECT
    id,
    movement_id,
    product_id,
    location_id,
    quantity_delta,
    unit_cost
FROM movement_entries
WHERE movement_id = $1
ORDER BY location_id, id;

-- Writes the exact inverse of the original's entries under the new movement.
-- name: CreateReversalEntries :execrows
INSERT INTO movement_entries (
    movement_id,
    product_id,
    location_id,
    quantity_delta,
    unit_cost
)
SELECT
    sqlc.arg('reversal_movement_id'),
    product_id,
    location_id,
    -quantity_delta,
    unit_cost
FROM movement_entries
WHERE movement_id = sqlc.arg('original_movement_id');
