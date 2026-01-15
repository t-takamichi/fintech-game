-- 000005_backfill_accounts_and_update_functions.up.sql
-- Backfill data from existing `accounts` table into `accounts_master` / `accounts_balance`
-- Then update transactions foreign key and replace fn_apply_transaction to use accounts_balance.

-- Backfill master
INSERT INTO accounts_master (user_id, created_at, credit_score, current_turn, is_frozen)
SELECT user_id, created_at, credit_score, current_turn, is_frozen FROM accounts
ON CONFLICT (user_id) DO NOTHING;

-- Backfill balance
INSERT INTO accounts_balance (user_id, balance, loan_principal, updated_at)
SELECT user_id, balance, loan_principal, updated_at FROM accounts
ON CONFLICT (user_id) DO NOTHING;

-- Update transactions FK to reference accounts_master instead of accounts
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_accounts;
ALTER TABLE transactions
  ADD CONSTRAINT fk_transactions_accounts_master FOREIGN KEY(user_id) REFERENCES accounts_master(user_id) ON DELETE CASCADE;

-- Replace fn_apply_transaction to operate on accounts_balance
CREATE OR REPLACE FUNCTION fn_apply_transaction(p_user_id uuid, p_amount bigint, p_type transaction_type, p_desc varchar)
RETURNS void AS $$
DECLARE
    v_balance bigint;
BEGIN
    -- acquire advisory lock per user
    PERFORM pg_advisory_xact_lock(hashtext(p_user_id::text));

    SELECT balance INTO v_balance FROM accounts_balance WHERE user_id = p_user_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found in accounts_balance: %', p_user_id;
    END IF;

    v_balance := v_balance + p_amount;
    UPDATE accounts_balance SET balance = v_balance, updated_at = now() WHERE user_id = p_user_id;

    INSERT INTO transactions(user_id, type, amount, balance_after, description, is_printed)
        VALUES (p_user_id, p_type, p_amount, v_balance, left(p_desc,15), false);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Keep fn_mark_as_printed as-is (it operates on transactions only)
