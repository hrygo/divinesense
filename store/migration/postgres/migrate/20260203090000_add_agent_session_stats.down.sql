-- Rollback: Drop agent_session_stats table
DROP TRIGGER IF EXISTS trigger_agent_session_stats_updated_at ON agent_session_stats;
DROP FUNCTION IF EXISTS update_agent_session_stats_updated_at();
DROP INDEX IF EXISTS idx_session_stats_user_success;
DROP INDEX IF EXISTS idx_session_stats_cost;
DROP INDEX IF EXISTS idx_session_stats_agent;
DROP INDEX IF EXISTS idx_session_stats_conv;
DROP INDEX IF EXISTS idx_session_stats_user_date;
DROP TABLE IF EXISTS agent_session_stats;
