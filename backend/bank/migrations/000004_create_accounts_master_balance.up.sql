-- 000004_create_accounts_master_balance.up.sql
-- Create normalized account master and balance tables

CREATE TABLE IF NOT EXISTS accounts_master (
    user_id UUID PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    credit_score INTEGER NOT NULL DEFAULT 3 CHECK (credit_score BETWEEN 1 AND 10),
    current_turn INTEGER NOT NULL DEFAULT 0,
    is_frozen BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS accounts_balance (
    user_id UUID PRIMARY KEY REFERENCES accounts_master(user_id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    loan_principal BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_balance_user_id ON accounts_balance(user_id);

-- trigger to keep updated_at current for accounts_balance
CREATE OR REPLACE FUNCTION trigger_set_timestamp_accounts_balance()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp_accounts_balance ON accounts_balance;
CREATE TRIGGER set_timestamp_accounts_balance
BEFORE UPDATE ON accounts_balance
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp_accounts_balance();
