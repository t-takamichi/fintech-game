-- 0001_create_accounts.sql
-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    user_id UUID PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    loan_principal BIGINT NOT NULL DEFAULT 0,
    credit_score INTEGER NOT NULL DEFAULT 3 CHECK (credit_score BETWEEN 1 AND 10),
    is_frozen BOOLEAN NOT NULL DEFAULT FALSE,
    current_turn INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_current_turn ON accounts(current_turn);

-- trigger to keep updated_at current
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_timestamp ON accounts;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
