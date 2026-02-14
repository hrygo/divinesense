-- Add episodic_memory_embedding table for semantic similarity search
-- Stores vector embeddings generated from user_input + summary

CREATE TABLE episodic_memory_embedding (
  id SERIAL PRIMARY KEY,
  episodic_memory_id INTEGER NOT NULL,
  embedding vector(1024) NOT NULL,
  model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  CONSTRAINT fk_episodic_memory_embedding_memory
    FOREIGN KEY (episodic_memory_id)
    REFERENCES episodic_memory(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_episodic_memory_embedding_memory_model
    UNIQUE (episodic_memory_id, model)
);

-- HNSW index for fast vector similarity search
CREATE INDEX idx_episodic_memory_embedding_hnsw
ON episodic_memory_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Index for foreign key lookup
CREATE INDEX idx_episodic_memory_embedding_memory_id
ON episodic_memory_embedding (episodic_memory_id);

-- Auto-update timestamp trigger
CREATE OR REPLACE FUNCTION update_episodic_memory_embedding_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_episodic_memory_embedding_updated_ts
  BEFORE UPDATE ON episodic_memory_embedding
  FOR EACH ROW
  EXECUTE FUNCTION update_episodic_memory_embedding_updated_ts();

COMMENT ON TABLE episodic_memory_embedding IS 'Stores vector embeddings for episodic memories to enable semantic similarity search';
COMMENT ON COLUMN episodic_memory_embedding.model IS 'Embedding model used (default: BAAI/bge-m3)';
