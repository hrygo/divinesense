-- system_setting
CREATE TABLE system_setting (
  name TEXT NOT NULL PRIMARY KEY,
  value TEXT NOT NULL,
  description TEXT NOT NULL
);

-- user
CREATE TABLE "user" (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  username TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL DEFAULT 'USER',
  email TEXT NOT NULL DEFAULT '',
  nickname TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL,
  avatar_url TEXT NOT NULL,
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
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  content TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  payload JSONB NOT NULL DEFAULT '{}',
  embedding vector(1024)
);

-- Create HNSW index for fast vector similarity search
CREATE INDEX memo_embedding_idx
ON memo USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- memo_relation
CREATE TABLE memo_relation (
  memo_id INTEGER NOT NULL,
  related_memo_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  UNIQUE(memo_id, related_memo_id, type)
);

-- attachment
CREATE TABLE attachment (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  filename TEXT NOT NULL,
  blob BYTEA,
  type TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  memo_id INTEGER DEFAULT NULL,
  storage_type TEXT NOT NULL DEFAULT '',
  reference TEXT NOT NULL DEFAULT '',
  file_path TEXT,
  thumbnail_path TEXT,
  extracted_text TEXT,
  ocr_text TEXT,
  payload JSONB NOT NULL DEFAULT '{}',
  CONSTRAINT chk_attachment_row_status CHECK (row_status IN ('NORMAL', 'ARCHIVED', 'DELETED'))
);

-- Indexes for attachment table
CREATE INDEX idx_attachment_creator_status ON attachment(creator_id, row_status);
CREATE INDEX idx_attachment_type ON attachment(type);
CREATE INDEX idx_attachment_memo ON attachment(memo_id) WHERE memo_id IS NOT NULL;
CREATE INDEX idx_attachment_text_gin ON attachment USING gin(to_tsvector('simple', COALESCE(extracted_text, '') || ' ' || COALESCE(ocr_text, ''))) WHERE extracted_text IS NOT NULL OR ocr_text IS NOT NULL;

-- activity
CREATE TABLE activity (
  id SERIAL PRIMARY KEY,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  type TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT 'INFO',
  payload JSONB NOT NULL DEFAULT '{}'
);

-- idp
CREATE TABLE idp (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  identifier_filter TEXT NOT NULL DEFAULT '',
  config JSONB NOT NULL DEFAULT '{}'
);

-- inbox
CREATE TABLE inbox (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  sender_id INTEGER NOT NULL,
  receiver_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  message TEXT NOT NULL
);

-- reaction
CREATE TABLE reaction (
  id SERIAL PRIMARY KEY,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  creator_id INTEGER NOT NULL,
  content_id TEXT NOT NULL,
  reaction_type TEXT NOT NULL,
  UNIQUE(creator_id, content_id, reaction_type)
);

-- memo_embedding
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memo_embedding (
  id SERIAL PRIMARY KEY,
  memo_id INTEGER NOT NULL,
  embedding vector(1024) NOT NULL,
  model VARCHAR(100) NOT NULL DEFAULT 'BAAI/bge-m3',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT,
  CONSTRAINT fk_memo_embedding_memo
    FOREIGN KEY (memo_id)
    REFERENCES memo(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_memo_embedding_memo_model
    UNIQUE (memo_id, model)
);

CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

CREATE INDEX idx_memo_embedding_memo_id
ON memo_embedding (memo_id);

CREATE OR REPLACE FUNCTION update_memo_embedding_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_memo_embedding_updated_ts
  BEFORE UPDATE ON memo_embedding
  FOR EACH ROW
  EXECUTE FUNCTION update_memo_embedding_updated_ts();

-- schedule
CREATE TABLE schedule (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',
  start_ts BIGINT NOT NULL,
  end_ts BIGINT,
  all_day BOOLEAN NOT NULL DEFAULT FALSE,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  recurrence_rule TEXT,
  recurrence_end_ts BIGINT,
  reminders TEXT NOT NULL DEFAULT '[]',
  payload JSONB NOT NULL DEFAULT '{}',
  CONSTRAINT fk_schedule_creator
    FOREIGN KEY (creator_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_schedule_time_range
    CHECK (end_ts IS NULL OR end_ts >= start_ts),
  CONSTRAINT chk_schedule_reminders_json
    CHECK (reminders ~ '^(\[\]|\[\{.*\}\])$')
);

CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX idx_schedule_uid ON schedule(uid);

-- Atomic conflict detection constraint (V0.52)
-- Note: The EXCLUDE constraint requires IMMUTABLE functions and is added via incremental migration
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE INDEX IF NOT EXISTS idx_schedule_creator_time
ON schedule(creator_id, start_ts)
WHERE row_status = 'NORMAL';

CREATE OR REPLACE FUNCTION update_schedule_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_schedule_updated_ts
  BEFORE UPDATE ON schedule
  FOR EACH ROW
  EXECUTE FUNCTION update_schedule_updated_ts();

-- ai_conversation
CREATE TABLE ai_conversation (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  parrot_id TEXT NOT NULL DEFAULT '',
  pinned BOOLEAN NOT NULL DEFAULT FALSE,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  row_status TEXT NOT NULL DEFAULT 'NORMAL',
  CONSTRAINT fk_ai_conversation_creator
    FOREIGN KEY (creator_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_conversation_row_status
    CHECK (row_status IN ('NORMAL', 'ARCHIVED'))
);

CREATE INDEX idx_ai_conversation_creator ON ai_conversation(creator_id);
CREATE INDEX idx_ai_conversation_updated ON ai_conversation(updated_ts DESC);

-- ai_message
CREATE TABLE ai_message (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  type TEXT NOT NULL DEFAULT 'MESSAGE',
  role TEXT NOT NULL DEFAULT 'USER',
  content TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_ai_message_conversation
    FOREIGN KEY (conversation_id)
    REFERENCES ai_conversation(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_message_type
    CHECK (type IN ('MESSAGE', 'SEPARATOR')),
  CONSTRAINT chk_ai_message_role
    CHECK (role IN ('USER', 'ASSISTANT', 'SYSTEM'))
);

CREATE INDEX idx_ai_message_conversation ON ai_message(conversation_id);
CREATE INDEX idx_ai_message_created ON ai_message(created_ts ASC);

-- Trigger to update updated_ts on ai_conversation
CREATE OR REPLACE FUNCTION update_ai_conversation_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_conversation_updated_ts
  BEFORE UPDATE ON ai_conversation
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_conversation_updated_ts();
-- =============================================================================
-- Agent Session Stats (V0.54.0)
-- =============================================================================
CREATE TABLE agent_session_stats (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL UNIQUE,
    conversation_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    agent_type VARCHAR(20) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    total_duration_ms BIGINT NOT NULL,
    thinking_duration_ms BIGINT NOT NULL DEFAULT 0,
    tool_duration_ms BIGINT NOT NULL DEFAULT 0,
    generation_duration_ms BIGINT NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    total_cost_usd NUMERIC(10,4) NOT NULL DEFAULT 0,
    tool_call_count INTEGER NOT NULL DEFAULT 0,
    tools_used JSONB,
    files_modified INTEGER NOT NULL DEFAULT 0,
    file_paths TEXT[],
    model_used VARCHAR(100),
    is_error BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_session_stats_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    -- Note: conversation_id FK removed in V0.54.3 - Geek/Evolution modes operate independently
    CONSTRAINT chk_agent_session_stats_type CHECK (agent_type IN ('geek', 'evolution'))
);

CREATE INDEX idx_session_stats_user_date ON agent_session_stats(user_id, started_at DESC);
CREATE INDEX idx_session_stats_conv ON agent_session_stats(conversation_id);
CREATE INDEX idx_session_stats_agent ON agent_session_stats(agent_type, started_at DESC);
CREATE INDEX idx_session_stats_cost ON agent_session_stats(total_cost_usd) WHERE total_cost_usd > 0;
-- Partial index for successful sessions (is_error=false) - optimizes cost queries
CREATE INDEX idx_session_stats_user_success ON agent_session_stats(user_id, started_at DESC) WHERE is_error = false;

CREATE OR REPLACE FUNCTION update_agent_session_stats_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agent_session_stats_updated_at
  BEFORE UPDATE ON agent_session_stats
  FOR EACH ROW
  EXECUTE FUNCTION update_agent_session_stats_updated_at();

-- =============================================================================
-- User Cost Settings (V0.54.1)
-- =============================================================================
CREATE TABLE user_cost_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    daily_budget_usd NUMERIC(10,4),
    per_session_threshold_usd NUMERIC(10,4) DEFAULT 5.0,
    alert_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alert_email BOOLEAN NOT NULL DEFAULT FALSE,
    alert_in_app BOOLEAN NOT NULL DEFAULT TRUE,
    budget_reset_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cost_settings_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE INDEX idx_cost_settings_user ON user_cost_settings(user_id);

CREATE OR REPLACE FUNCTION update_user_cost_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_user_cost_settings_updated_at
  BEFORE UPDATE ON user_cost_settings
  FOR EACH ROW
  EXECUTE FUNCTION update_user_cost_settings_updated_at();

-- =============================================================================
-- Security Audit Log (V0.54.2)
-- =============================================================================
CREATE TABLE agent_security_audit (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64),
    user_id INTEGER NOT NULL,
    agent_type VARCHAR(20) NOT NULL,
    operation_type VARCHAR(50) NOT NULL,
    operation_name VARCHAR(100),
    risk_level VARCHAR(20) NOT NULL,
    command_input TEXT,
    command_matched_pattern TEXT,
    action_taken VARCHAR(50),
    reason TEXT,
    file_path TEXT,
    tool_id VARCHAR(100),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    CONSTRAINT chk_audit_risk_level CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

CREATE INDEX idx_audit_user_date ON agent_security_audit(user_id, occurred_at DESC);
CREATE INDEX idx_audit_risk ON agent_security_audit(risk_level, occurred_at DESC);
CREATE INDEX idx_audit_operation ON agent_security_audit(operation_type, occurred_at DESC);
CREATE INDEX idx_audit_session ON agent_security_audit(session_id) WHERE session_id IS NOT NULL;

-- =============================================================================
-- 版本记录
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.54.2', '数据库 schema 版本')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
