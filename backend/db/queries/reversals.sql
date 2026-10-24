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

-- name: GetLatestActiveMovementID :one
SELECT m.id
FROM movements m
WHERE m.product_id = $1
  AND m.movement_type <> 'REVERSAL'
  AND NOT EXISTS (SELECT 1 FROM movements r WHERE r.reversal_of_id = m.id)
ORDER BY m.id DESC
LIMIT 1;
