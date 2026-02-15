-- Rollback memo_summary table
DROP TRIGGER IF EXISTS trigger_memo_summary_updated_ts ON memo_summary;
DROP FUNCTION IF EXISTS update_memo_summary_updated_ts();
DROP INDEX IF EXISTS idx_memo_summary_status;
DROP INDEX IF EXISTS idx_memo_summary_memo_id;
DROP TABLE IF EXISTS memo_summary;
