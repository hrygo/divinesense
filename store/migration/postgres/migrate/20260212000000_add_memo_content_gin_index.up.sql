-- Add GIN index for BM25 full-text search on memo content
-- This enables Issue #167: BM25 search performance optimization

-- Create GIN index for full-text search using simple configuration
-- The 'simple' config works well for Chinese and English mixed content
CREATE INDEX IF NOT EXISTS idx_memo_content_gin
ON memo USING gin(to_tsvector('simple', COALESCE(content, '')));

COMMENT ON INDEX idx_memo_content_gin IS 'GIN index for BM25 full-text search on memo.content';
