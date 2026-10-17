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
