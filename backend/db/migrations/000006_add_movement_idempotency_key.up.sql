-- A client-supplied key that makes a posting safe to retry: the same key never posts twice.
ALTER TABLE movements ADD COLUMN idempotency_key VARCHAR(128);

ALTER TABLE movements
    ADD CONSTRAINT movements_idempotency_key_key UNIQUE (idempotency_key);
