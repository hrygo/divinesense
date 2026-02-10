-- ============================================================================
-- SQLite AI Support - Phase 1: Vector Search Only
-- ============================================================================
-- This migration adds vector search support to SQLite using sqlite-vec.
--
-- ⚠️ BREAKING CHANGES ⚠️:
-- This migration removes ai_conversation and ai_message tables (deprecated API).
-- - These tables used the legacy AIConversation/AIMessage API
-- - Future PRs will implement AIBlock-based conversation persistence
--
-- 🚧 FUTURE FEATURES (Planned in later PRs):
-- - AIBlock/AIConversation support (PR #132)
-- - EpisodicMemory support (PR #133)
-- - UserPreferences support (PR #134)
-- - AgentMetrics support (PR #134)
--
-- For full AI features including conversation persistence, use PostgreSQL.
-- See: https://github.com/hrygo/divinesense/issues/134
-- ============================================================================

-- Drop deprecated tables (used legacy AIConversation/AIMessage API)
DROP TABLE IF EXISTS ai_message;
DROP TABLE IF EXISTS ai_conversation;

-- 1. Vector storage table for memo embeddings
-- Vectors are stored in dual format:
-- - embedding (TEXT): JSON-encoded float32 array for fallback compatibility
-- - embedding_vec (BLOB): vec0 format for sqlite-vec O(log n) search
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

-- Note: When embedding_vec is populated, sqlite-vec will be used for
--       efficient KNN search (O(log n)). Otherwise, application-layer
--       cosine similarity (O(n)) will be used as fallback.
