-- ============================================================================
-- SQLite Schema - LATEST
-- ============================================================================
-- ⚠️ AI SUPPORT POLICY:
-- - PostgreSQL: Full AI support (AIBlock, episodic memory, etc.)
-- - SQLite: Vector search ONLY (see #134 for roadmap)
--
-- 🚧 TODO (Future PRs):
-- - PR #132: AIBlock/AIConversation SQLite support
-- - PR #133: EpisodicMemory SQLite support
-- - PR #134: UserPreferences SQLite support
-- ============================================================================

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

-- memo_embedding (Vector storage for semantic search)
-- ⚠️ NOTE: SQLite ONLY supports vector search (semantic retrieval).
--          Full AI features (AIBlock, episodic memory, etc.) require PostgreSQL.
--          See: https://github.com/hrygo/divinesense/issues/134
--
-- Vectors are stored in dual format:
-- - embedding (TEXT): JSON-encoded float32 array for fallback
-- - embedding_vec (BLOB): vec0 format for sqlite-vec O(log n) search
CREATE TABLE memo_embedding (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  memo_id INTEGER NOT NULL,
  embedding TEXT NOT NULL,       -- JSON-encoded float32 array (fallback)
  embedding_vec BLOB,            -- vec0 format BLOB for sqlite-vec (optional)
  model TEXT NOT NULL DEFAULT 'BAAI/bge-m3',
  created_ts INTEGER NOT NULL,
  updated_ts INTEGER NOT NULL,
  UNIQUE(memo_id, model),
  CONSTRAINT fk_memo_embedding_memo FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

CREATE INDEX idx_memo_embedding_memo_id ON memo_embedding(memo_id);
CREATE INDEX idx_memo_embedding_model ON memo_embedding(model);

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
