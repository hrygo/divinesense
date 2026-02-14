-- Rollback episodic_memory_embedding table
DROP TRIGGER IF EXISTS trigger_episodic_memory_embedding_updated_ts ON episodic_memory_embedding;
DROP FUNCTION IF EXISTS update_episodic_memory_embedding_updated_ts();
DROP INDEX IF EXISTS idx_episodic_memory_embedding_memory_id;
DROP INDEX IF EXISTS idx_episodic_memory_embedding_hnsw;
DROP TABLE IF EXISTS episodic_memory_embedding;
