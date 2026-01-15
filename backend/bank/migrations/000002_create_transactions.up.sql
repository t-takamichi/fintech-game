
-- create enum type for transaction types if not exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'transaction_type') THEN
        CREATE TYPE transaction_type AS ENUM ('LOAN', 'BUY', 'SELL', 'INTEREST', 'SETTLE');
    END IF;
END$$;


-- Use IDENTITY for better portability (Aurora / Postgres)
CREATE TABLE IF NOT EXISTS transactions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id UUID NOT NULL,
    type transaction_type NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    description VARCHAR(15) NOT NULL,
    is_printed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    CONSTRAINT fk_transactions_accounts FOREIGN KEY(user_id) REFERENCES accounts(user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id_created_at ON transactions(user_id, created_at DESC);

-- The following trigger enforces insert-only semantics except toggling `is_printed`.
-- It is retained as a safety net, but recommended access control is provided below
-- (create limited `api_user` role and use SECURITY DEFINER functions for writes).
CREATE OR REPLACE FUNCTION transactions_protect_mutations()
RETURNS TRIGGER AS $$
BEGIN
        IF (TG_OP = 'DELETE') THEN
                RAISE EXCEPTION 'DELETE is not allowed on transactions (insert-only)';
        END IF;

        IF (TG_OP = 'UPDATE') THEN
                IF NEW.is_printed IS DISTINCT FROM OLD.is_printed THEN
                        IF NEW.user_id     IS DISTINCT FROM OLD.user_id OR
                             NEW.type        IS DISTINCT FROM OLD.type OR
                             NEW.amount      IS DISTINCT FROM OLD.amount OR
                             NEW.balance_after IS DISTINCT FROM OLD.balance_after OR
                             NEW.description IS DISTINCT FROM OLD.description OR
                             NEW.created_at  IS DISTINCT FROM OLD.created_at THEN
                             RAISE EXCEPTION 'Only is_printed may be updated on transactions';
                        END IF;
                        RETURN NEW;
                ELSE
                        RAISE EXCEPTION 'Updates to transactions are not allowed except is_printed';
                END IF;
        END IF;

        RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS protect_transactions_mutations ON transactions;
CREATE TRIGGER protect_transactions_mutations
BEFORE UPDATE OR DELETE ON transactions
FOR EACH ROW EXECUTE PROCEDURE transactions_protect_mutations();

-- Create SECURITY DEFINER functions for use with Aurora Data API or minimized-role DB users.
-- These functions bundle multiple statements into one call so the Data API can execute atomically.

CREATE OR REPLACE FUNCTION fn_apply_transaction(p_user_id uuid, p_amount bigint, p_type transaction_type, p_desc varchar)
RETURNS void AS $$
DECLARE
    v_balance bigint;
BEGIN
    -- acquire lightweight advisory lock per user to avoid concurrent modifications
    PERFORM pg_advisory_xact_lock(hashtext(p_user_id::text));

    SELECT balance INTO v_balance FROM accounts WHERE user_id = p_user_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found: %', p_user_id;
    END IF;

    v_balance := v_balance + p_amount;
    UPDATE accounts SET balance = v_balance WHERE user_id = p_user_id;

    INSERT INTO transactions(user_id, type, amount, balance_after, description, is_printed)
        VALUES (p_user_id, p_type, p_amount, v_balance, left(p_desc,15), false);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE OR REPLACE FUNCTION fn_mark_as_printed(p_ids bigint[])
RETURNS void AS $$
BEGIN
    UPDATE transactions SET is_printed = true WHERE id = ANY(p_ids);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create an API role with minimal privileges and grant execute on safe functions.
-- In production you may prefer IAM auth and map to this role instead of storing passwords.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'api_user') THEN
        CREATE ROLE api_user LOGIN;
    END IF;
END$$;

-- Restrict direct DML for api_user; grant SELECT and EXECUTE on functions only.
REVOKE ALL ON SCHEMA public FROM api_user;
GRANT USAGE ON SCHEMA public TO api_user;

GRANT SELECT ON accounts TO api_user;
GRANT SELECT ON transactions TO api_user;

REVOKE INSERT, UPDATE, DELETE ON transactions FROM api_user;
REVOKE INSERT, UPDATE, DELETE ON accounts FROM api_user;

GRANT EXECUTE ON FUNCTION fn_apply_transaction(uuid,bigint,transaction_type,varchar) TO api_user;
GRANT EXECUTE ON FUNCTION fn_mark_as_printed(bigint[]) TO api_user;

-- Note: Database administrators can still alter roles/triggers; include this in your runbook.
