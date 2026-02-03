-- Rollback: Drop agent and tool metrics tables
DROP INDEX IF EXISTS idx_tool_metrics_hour;
DROP TABLE IF EXISTS tool_metrics;
DROP INDEX IF EXISTS idx_agent_metrics_hour;
DROP TABLE IF EXISTS agent_metrics;
