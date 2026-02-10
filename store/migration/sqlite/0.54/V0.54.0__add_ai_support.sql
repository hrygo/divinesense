-- ============================================================================
-- SQLite AI Support: Vector Search (sqlite-vec)
-- ============================================================================
-- This migration adds vector search support to SQLite using sqlite-vec:
-- - Vector storage and similarity search (memo_embedding with vec0 BLOB)
--
-- Note: AI conversation persistence (AIBlock/AIConversation) requires PostgreSQL.
--       For full AI features, use PostgreSQL with pgvector extension.
-- ============================================================================

-- 1. Vector storage table for memo embeddings
-- Vectors are stored in vec0 BLOB format for sqlite-vec compatibility
CREATE TABLE IF NOT EXISTS memo_embedding (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  memo_id INTEGER NOT NULL,
  embedding TEXT NOT NULL,       -- JSON-encoded float32 array (fallback)
  embedding_vec BLOB,            -- vec0 format BLOB for sqlite-vec (optional)
  model TEXT NOT NULL DEFAULT 'BAAI/bge-m3',
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  UNIQUE(memo_id, model),
  CONSTRAINT fk_memo_embedding_memo
    FOREIGN KEY (memo_id)
    REFERENCES memo(id)
    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_memo_embedding_memo_id
  ON memo_embedding(memo_id);
CREATE INDEX IF NOT EXISTS idx_memo_embedding_model
  ON memo_embedding(model);

-- Note: If embedding_vec is populated, sqlite-vec will be used for O(log n) search.
--       Otherwise, application-layer cosine similarity (O(n)) will be used.
