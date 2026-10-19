CREATE TABLE warehouses (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code        VARCHAR(64)      NOT NULL UNIQUE,
    name        VARCHAR(200)     NOT NULL,
    x           DOUBLE PRECISION NOT NULL DEFAULT 0,
    y           DOUBLE PRECISION NOT NULL DEFAULT 0,
    radius_m    DOUBLE PRECISION NOT NULL CHECK (radius_m > 0),
    capacity    BIGINT           CHECK (capacity IS NULL OR capacity > 0),
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

INSERT INTO warehouses (code, name, radius_m) VALUES ('MAIN', 'Main warehouse', 1000);

ALTER TABLE locations
    ADD COLUMN warehouse_id BIGINT           REFERENCES warehouses(id),
    ADD COLUMN x            DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN y            DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN capacity     BIGINT           CHECK (capacity IS NULL OR capacity > 0),
    ADD COLUMN auto_place   BOOLEAN          NOT NULL DEFAULT TRUE;

UPDATE locations SET warehouse_id = (SELECT id FROM warehouses WHERE code = 'MAIN');

ALTER TABLE locations ALTER COLUMN warehouse_id SET NOT NULL;

CREATE INDEX idx_locations_warehouse ON locations (warehouse_id);
