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
