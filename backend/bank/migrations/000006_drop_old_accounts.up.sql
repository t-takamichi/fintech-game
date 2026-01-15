-- 000006_drop_old_accounts.up.sql
-- Safely remove legacy `accounts` table and its triggers/functions.
-- Run this only after verifying the app reads/writes to `accounts_master`/`accounts_balance` and backups exist.


-- Drop trigger that maintained updated_at on old accounts
DROP TRIGGER IF EXISTS set_timestamp ON accounts;

-- Drop trigger function if not used by others (confirm before running in production)
DROP FUNCTION IF EXISTS trigger_set_timestamp();

-- Finally drop legacy table
DROP TABLE IF EXISTS accounts;

