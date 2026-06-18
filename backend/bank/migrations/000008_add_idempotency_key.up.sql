-- 000008_add_idempotency_key.up.sql
-- Add idempotency_key to transactions to prevent double spends and ensure API idempotency

ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS idempotency_key UUID UNIQUE;
