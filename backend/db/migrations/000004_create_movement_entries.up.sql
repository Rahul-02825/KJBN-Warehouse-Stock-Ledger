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
