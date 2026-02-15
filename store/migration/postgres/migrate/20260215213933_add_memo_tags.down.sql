-- Rollback memo_tags table
DROP INDEX IF EXISTS idx_memo_tags_tag;
DROP INDEX IF EXISTS idx_memo_tags_memo_id;
DROP TABLE IF EXISTS memo_tags;
