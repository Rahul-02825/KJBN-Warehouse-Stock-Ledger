ALTER TABLE movements DROP CONSTRAINT IF EXISTS movements_idempotency_key_key;
ALTER TABLE movements DROP COLUMN IF EXISTS idempotency_key;
