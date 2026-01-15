-- 000007_add_subject_id.up.sql
-- Add subject_id to accounts_master for external lookups

ALTER TABLE accounts_master
  ADD COLUMN IF NOT EXISTS subject_id TEXT;

-- subject_id を使った検索が主目的なのでインデックスを作成
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_master_subject_id ON accounts_master(subject_id);
