-- GENERATED FILE. Do not edit by hand.
-- Canonical schema snapshot for sqlc, built from db/migrations/*.up.sql
-- Regenerate with: sh scripts/schema-snapshot.sh

-- ---- 000001_create_products.up.sql ----
-- Products: catalogue items plus the current weighted-average cost.
-- average_cost is derived state maintained by IN (and reversal-of-IN) postings.
CREATE TABLE products (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sku             VARCHAR(64)    NOT NULL UNIQUE,
    name            VARCHAR(200)   NOT NULL,
    unit            VARCHAR(16)    NOT NULL
                    CHECK (unit IN ('kg', 'pcs', 'litre')),
    reorder_level   NUMERIC(18, 4) NOT NULL DEFAULT 0
                    CHECK (reorder_level >= 0),
    average_cost    NUMERIC(18, 4) NOT NULL DEFAULT 0
                    CHECK (average_cost >= 0),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_name ON products (name);

-- ---- 000002_create_locations.up.sql ----
-- Locations: warehouse bins such as A-01 or COLD-02.
CREATE TABLE locations (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code            VARCHAR(64)  NOT NULL UNIQUE,
    description     TEXT         NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---- 000003_create_movements.up.sql ----
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

-- ---- 000004_create_movement_entries.up.sql ----
-- Movement entries: the ledger. One signed quantity effect per location touched.
-- IN/OUT/ADJUSTMENT produce one entry, TRANSFER produces two (-source, +destination),
-- REVERSAL produces the exact inverse of the original's entries.
-- The sum of quantity_delta per (product, location) is the true stock balance.
CREATE TABLE movement_entries (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    movement_id     BIGINT         NOT NULL REFERENCES movements(id),
    product_id      BIGINT         NOT NULL REFERENCES products(id),
    location_id     BIGINT         NOT NULL REFERENCES locations(id),
    quantity_delta  NUMERIC(18, 4) NOT NULL
                    CHECK (quantity_delta <> 0),
    unit_cost       NUMERIC(18, 4)
                    CHECK (unit_cost IS NULL OR unit_cost >= 0),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    -- A movement touches a given location at most once.
    CONSTRAINT movement_entries_one_per_location UNIQUE (movement_id, location_id)
);

CREATE INDEX idx_movement_entries_product_location
    ON movement_entries (product_id, location_id, id);

-- ---- 000005_create_stock_balances.up.sql ----
-- Stock balances: cached quantity per (product, location).
-- Performance optimisation only. The ledger (movement_entries) is the source
-- of truth; the reconciliation report compares the two.
-- quantity >= 0 is a last line of defence; the application enforces it
-- with a conditional UPDATE under row locks.
CREATE TABLE stock_balances (
    product_id      BIGINT         NOT NULL REFERENCES products(id),
    location_id     BIGINT         NOT NULL REFERENCES locations(id),
    quantity        NUMERIC(18, 4) NOT NULL DEFAULT 0
                    CHECK (quantity >= 0),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, location_id)
);

CREATE INDEX idx_stock_balances_location
    ON stock_balances (location_id);

-- ---- 000006_add_movement_idempotency_key.up.sql ----
-- A client-supplied key that makes a posting safe to retry: the same key never posts twice.
ALTER TABLE movements ADD COLUMN idempotency_key VARCHAR(128);

ALTER TABLE movements
    ADD CONSTRAINT movements_idempotency_key_key UNIQUE (idempotency_key);
