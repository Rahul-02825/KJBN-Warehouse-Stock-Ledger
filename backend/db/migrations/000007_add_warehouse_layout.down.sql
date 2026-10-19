DROP INDEX IF EXISTS idx_locations_warehouse;

ALTER TABLE locations
    DROP COLUMN IF EXISTS auto_place,
    DROP COLUMN IF EXISTS capacity,
    DROP COLUMN IF EXISTS y,
    DROP COLUMN IF EXISTS x,
    DROP COLUMN IF EXISTS warehouse_id;

DROP TABLE IF EXISTS warehouses;
