-- Movements: the append-only business events. One row per posting.
--
-- quantity is always a positive magnitude. Direction comes from the type and
-- from which location columns are set:
--   IN          to_location_id only            unit_cost = receipt cost (> 0)
--   OUT         from_location_id only          unit_cost = frozen average at posting
--   TRANSFER    both, must differ              unit_cost = average at posting (informational)
--   ADJUSTMENT  exactly one: to_* = stock added, from_* = stock removed
--   REVERSAL    original's locations swapped   reversal_of_id set, unit_cost copied
--
-- reason is only present on ADJUSTMENT. 'other' requires a non-blank note.
-- Application code must never UPDATE or DELETE rows in this table.
CREATE TABLE movements (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id       BIGINT         NOT NULL REFERENCES products(id),
    movement_type    VARCHAR(16)    NOT NULL
                     CHECK (movement_type IN
                        ('IN', 'OUT', 'TRANSFER', 'ADJUSTMENT', 'REVERSAL')),
    from_location_id BIGINT         REFERENCES locations(id),
    to_location_id   BIGINT         REFERENCES locations(id),
    quantity         NUMERIC(18, 4) NOT NULL
                     CHECK (quantity > 0),
    unit_cost        NUMERIC(18, 4)
                     CHECK (unit_cost IS NULL OR unit_cost >= 0),
    reference        VARCHAR(128)   NOT NULL DEFAULT '',
    reason           VARCHAR(32)
                     CHECK (reason IS NULL OR reason IN
                        ('damaged', 'expired', 'count_correction', 'theft', 'other')),
    note             TEXT           NOT NULL DEFAULT '',
    created_by       VARCHAR(128)   NOT NULL,
    reversal_of_id   BIGINT         UNIQUE REFERENCES movements(id),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT movements_distinct_locations CHECK (
        from_location_id IS NULL
        OR to_location_id IS NULL
        OR from_location_id <> to_location_id
    ),

    -- REVERSAL rows and only REVERSAL rows point at an original.
    CONSTRAINT movements_reversal_link CHECK (
        (movement_type = 'REVERSAL') = (reversal_of_id IS NOT NULL)
    ),

    -- ADJUSTMENT rows and only ADJUSTMENT rows carry a reason.
    CONSTRAINT movements_reason_only_on_adjustment CHECK (
        (movement_type = 'ADJUSTMENT') = (reason IS NOT NULL)
    ),

    CONSTRAINT movements_other_reason_needs_note CHECK (
        reason IS DISTINCT FROM 'other' OR btrim(note) <> ''
    ),

    CONSTRAINT movements_type_shape CHECK (
        (movement_type = 'IN'
            AND from_location_id IS NULL
            AND to_location_id IS NOT NULL
            AND unit_cost IS NOT NULL
            AND unit_cost > 0)
        OR
        (movement_type = 'OUT'
            AND from_location_id IS NOT NULL
            AND to_location_id IS NULL
            AND unit_cost IS NOT NULL)
        OR
        (movement_type = 'TRANSFER'
            AND from_location_id IS NOT NULL
            AND to_location_id IS NOT NULL)
        OR
        (movement_type = 'ADJUSTMENT'
            AND ((from_location_id IS NULL) <> (to_location_id IS NULL)))
        OR
        (movement_type = 'REVERSAL'
            AND (from_location_id IS NOT NULL OR to_location_id IS NOT NULL))
    )
);

CREATE INDEX idx_movements_product_created
    ON movements (product_id, created_at DESC, id DESC);

CREATE INDEX idx_movements_type_created
    ON movements (movement_type, created_at DESC);

CREATE INDEX idx_movements_from_location
    ON movements (from_location_id);

CREATE INDEX idx_movements_to_location
    ON movements (to_location_id);

CREATE INDEX idx_movements_reference
    ON movements (reference);
