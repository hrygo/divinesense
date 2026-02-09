-- system_setting
CREATE TABLE system_setting (
  name TEXT NOT NULL,
  value TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  UNIQUE(name)
);

-- user
CREATE TABLE user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL',
  username TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL CHECK (role IN ('HOST', 'ADMIN', 'USER')) DEFAULT 'USER',
  email TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL,
  avatar_url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT ''
);

-- user_setting
CREATE TABLE user_setting (
  user_id INTEGER NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  UNIQUE(user_id, key)
);

-- memo
CREATE TABLE memo (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL',
  content TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE')) DEFAULT 'PRIVATE',
  pinned INTEGER NOT NULL CHECK (pinned IN (0, 1)) DEFAULT 0,
  payload TEXT NOT NULL DEFAULT '{}'
);

-- memo_relation
CREATE TABLE memo_relation (
  memo_id INTEGER NOT NULL,
  related_memo_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  UNIQUE(memo_id, related_memo_id, type)
);

-- attachment
CREATE TABLE attachment (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  filename TEXT NOT NULL DEFAULT '',
  blob BLOB DEFAULT NULL,
  type TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  memo_id INTEGER,
  storage_type TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL DEFAULT '',
  payload TEXT NOT NULL DEFAULT '{}'
);

-- activity
CREATE TABLE activity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  type TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL CHECK (level IN ('INFO', 'WARN', 'ERROR')) DEFAULT 'INFO',
  payload TEXT NOT NULL DEFAULT '{}'
);

-- idp
CREATE TABLE idp (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  identifier_filter TEXT NOT NULL DEFAULT '',
  config TEXT NOT NULL DEFAULT '{}'
);

-- inbox
CREATE TABLE inbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  sender_id INTEGER NOT NULL,
  receiver_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '{}'
);

-- reaction
CREATE TABLE reaction (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  creator_id INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  reaction_type TEXT NOT NULL,
  UNIQUE(creator_id, content_id, reaction_type)
);

-- ai_conversation
CREATE TABLE ai_conversation (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  parrot_id TEXT NOT NULL DEFAULT '',
  pinned INTEGER NOT NULL CHECK (pinned IN (0, 1)) DEFAULT 0,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL'
);

-- ai_message
CREATE TABLE ai_message (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('MESSAGE', 'SEPARATOR')) DEFAULT 'MESSAGE',
  role TEXT NOT NULL CHECK (role IN ('USER', 'ASSISTANT', 'SYSTEM')) DEFAULT 'USER',
  content TEXT NOT NULL DEFAULT '',
  metadata TEXT NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now'))
);

-- memo_embedding (Vector storage using sqlite-vec)
CREATE TABLE memo_embedding (
  memo_id INTEGER PRIMARY KEY,
  model TEXT NOT NULL DEFAULT 'BAAI/bge-m3',
  embedding BLOB NOT NULL,  -- JSON-encoded float32 array for backward compatibility
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  UNIQUE(memo_id, model),
  CONSTRAINT fk_memo_embedding_memo FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

-- vec0 virtual table for efficient vector similarity search
-- Vectors are stored in vec0's optimized format for fast KNN queries
CREATE VIRTUAL TABLE IF NOT EXISTS vec0_embeddings USING vec0(
  embedding float32[1024]  -- BAAI/bge-m3 dimension
);

-- Note: vec0 virtual tables don't require manual indexes
-- The vec0 extension provides optimized KNN search internally

-- episodic_memory (AI agent long-term memory)
CREATE TABLE episodic_memory (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  timestamp INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  user_input TEXT,
  outcome TEXT,
  summary TEXT,
  importance REAL,
  created_ts INTEGER NOT NULL,
  CONSTRAINT fk_episodic_memory_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);
CREATE INDEX idx_episodic_memory_user_id ON episodic_memory(user_id);
CREATE INDEX idx_episodic_memory_timestamp ON episodic_memory(timestamp DESC);

-- user_preferences (AI personalization settings)
CREATE TABLE user_preferences (
  user_id INTEGER PRIMARY KEY,
  preferences TEXT NOT NULL DEFAULT '{}',
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  CONSTRAINT fk_user_preferences_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

-- conversation_context (AI session persistence)
CREATE TABLE conversation_context (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  context_data TEXT NOT NULL DEFAULT '{}',
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  CONSTRAINT fk_conversation_context_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
  CONSTRAINT chk_conversation_context_agent_type CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant'))
);
CREATE INDEX idx_conversation_context_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_context_updated ON conversation_context(updated_ts DESC);

-- agent_metrics (AI agent performance tracking)
CREATE TABLE agent_metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hour_bucket INTEGER NOT NULL,
  agent_type TEXT NOT NULL,
  request_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms INTEGER NOT NULL DEFAULT 0,
  latency_p50_ms INTEGER,
  latency_p95_ms INTEGER,
  errors TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  CONSTRAINT uq_agent_metrics_hour_type UNIQUE (hour_bucket, agent_type)
);
CREATE INDEX idx_agent_metrics_hour ON agent_metrics(hour_bucket DESC);

-- tool_metrics (AI tool usage tracking)
CREATE TABLE tool_metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  hour_bucket INTEGER NOT NULL,
  tool_name TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  CONSTRAINT uq_tool_metrics_hour_name UNIQUE (hour_bucket, tool_name)
);
CREATE INDEX idx_tool_metrics_hour ON tool_metrics(hour_bucket DESC);

-- schedule (AI-powered schedule assistant)
CREATE TABLE schedule (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',
  start_ts INTEGER NOT NULL,
  end_ts INTEGER,
  all_day INTEGER NOT NULL DEFAULT 0,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  recurrence_rule TEXT,
  recurrence_end_ts INTEGER,
  reminders TEXT NOT NULL DEFAULT '[]',
  payload TEXT NOT NULL DEFAULT '{}',
  CONSTRAINT fk_schedule_creator FOREIGN KEY (creator_id) REFERENCES "user"(id) ON DELETE CASCADE,
  CHECK (end_ts IS NULL OR end_ts >= start_ts)
);
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX idx_schedule_uid ON schedule(uid);
CREATE TRIGGER trigger_schedule_updated_ts AFTER UPDATE ON schedule FOR EACH ROW WHEN NEW.updated_ts <= OLD.updated_ts BEGIN UPDATE schedule SET updated_ts = strftime('%s', 'now') WHERE id = NEW.id; END;

