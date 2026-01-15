-- 0003_create_market_batches.sql
-- MarketBatches table: records of AI-generated news and interest rates per batch

CREATE TABLE IF NOT EXISTS market_batches (
    batch_id SERIAL PRIMARY KEY,
    news_summary TEXT NOT NULL,
    interest_rate REAL NOT NULL DEFAULT 0.0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_market_batches_created_at ON market_batches(created_at DESC);
