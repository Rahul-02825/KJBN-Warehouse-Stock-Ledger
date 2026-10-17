-- Locations: warehouse bins such as A-01 or COLD-02.
CREATE TABLE locations (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code            VARCHAR(64)  NOT NULL UNIQUE,
    description     TEXT         NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
