-- 000009_update_fn_apply_transaction.up.sql
-- Update fn_apply_transaction function to accept and enforce idempotency_key

CREATE OR REPLACE FUNCTION fn_apply_transaction(
    p_user_id uuid, 
    p_amount bigint, 
    p_type transaction_type, 
    p_desc varchar,
    p_idempotency_key uuid DEFAULT NULL
)
RETURNS void AS $$
DECLARE
    v_balance bigint;
    v_exists boolean;
BEGIN
    -- べき等性キーが指定されている場合、重複チェックを行う
    IF p_idempotency_key IS NOT NULL THEN
        SELECT EXISTS(SELECT 1 FROM transactions WHERE idempotency_key = p_idempotency_key) INTO v_exists;
        IF v_exists THEN
            RETURN; -- すでに存在する場合は何もせず正常終了
        END IF;
    END IF;

    -- acquire advisory lock per user
    PERFORM pg_advisory_xact_lock(hashtext(p_user_id::text));

    SELECT balance INTO v_balance FROM accounts_balance WHERE user_id = p_user_id FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Account not found in accounts_balance: %', p_user_id;
    END IF;

    v_balance := v_balance + p_amount;
    UPDATE accounts_balance SET balance = v_balance, updated_at = now() WHERE user_id = p_user_id;

    INSERT INTO transactions(user_id, type, amount, balance_after, description, is_printed, idempotency_key)
        VALUES (p_user_id, p_type, p_amount, v_balance, left(p_desc,15), false, p_idempotency_key);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
