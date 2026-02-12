-- Rollback GIN index for BM25 full-text search
DROP INDEX IF EXISTS idx_memo_content_gin;
