-- Rollback: Drop episodic_memory table
DROP INDEX IF EXISTS idx_episodic_memory_importance;
DROP INDEX IF EXISTS idx_episodic_memory_agent;
DROP INDEX IF EXISTS idx_episodic_memory_user_time;
DROP TABLE IF EXISTS episodic_memory;
