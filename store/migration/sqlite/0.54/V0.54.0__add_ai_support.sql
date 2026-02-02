-- ============================================================================
-- SQLite AI Support: Vector Search, Memory, Preferences, Metrics
-- ============================================================================
-- This migration adds full AI feature support to SQLite, enabling:
-- - Vector storage and similarity search (memo_embedding)
-- - Episodic memory for AI agents
-- - User preferences storage
-- - Conversation context persistence
-- - Agent and tool metrics
--
-- Note: Vector similarity search is implemented in Go application layer
-- (not via SQL extension) to maintain pure Go compatibility.
-- ============================================================================

-- 1. Vector storage table for memo embeddings
CREATE TABLE IF NOT EXISTS memo_embedding (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  memo_id INTEGER NOT NULL,
  embedding BLOB NOT NULL,  -- Vector stored as JSON-encoded float32 array
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

-- 2. Episodic memory for AI agents
CREATE TABLE IF NOT EXISTS episodic_memory (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  timestamp INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  user_input TEXT,
  outcome TEXT,
  summary TEXT,
  importance REAL,
  created_ts INTEGER NOT NULL,
  CONSTRAINT fk_episodic_memory_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_episodic_memory_user_id
  ON episodic_memory(user_id);
CREATE INDEX IF NOT EXISTS idx_episodic_memory_timestamp
  ON episodic_memory(timestamp DESC);

-- 3. User preferences (JSONB replacement)
CREATE TABLE IF NOT EXISTS user_preferences (
  user_id INTEGER PRIMARY KEY,
  preferences TEXT NOT NULL DEFAULT '{}',  -- JSON string (replaces JSONB)
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  CONSTRAINT fk_user_preferences_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

-- 4. Conversation context for session persistence
CREATE TABLE IF NOT EXISTS conversation_context (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  context_data TEXT NOT NULL DEFAULT '{}',  -- JSON string (replaces JSONB)
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  CONSTRAINT fk_conversation_context_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_conversation_context_agent_type
    CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant'))
);

CREATE INDEX IF NOT EXISTS idx_conversation_context_user
  ON conversation_context(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_context_updated
  ON conversation_context(updated_ts DESC);

-- 5. Agent metrics table
CREATE TABLE IF NOT EXISTS agent_metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hour_bucket INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  request_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms INTEGER NOT NULL DEFAULT 0,
  latency_p50_ms INTEGER,
  latency_p95_ms INTEGER,
  errors TEXT NOT NULL DEFAULT '{}',  -- JSON string (replaces JSONB)
  created_at INTEGER NOT NULL,
  CONSTRAINT uq_agent_metrics_hour_type UNIQUE (hour_bucket, agent_type)
);

CREATE INDEX IF NOT EXISTS idx_agent_metrics_hour
  ON agent_metrics(hour_bucket DESC);

-- 6. Tool metrics table
CREATE TABLE IF NOT EXISTS tool_metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hour_bucket INTEGER NOT NULL,
  tool_name TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  CONSTRAINT uq_tool_metrics_hour_name UNIQUE (hour_bucket, tool_name)
);

CREATE INDEX IF NOT EXISTS idx_tool_metrics_hour
  ON tool_metrics(hour_bucket DESC);
