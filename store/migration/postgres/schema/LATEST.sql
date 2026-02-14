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

-- Create GIN index for BM25 full-text search (V0.98.0)
CREATE INDEX IF NOT EXISTS idx_memo_content_gin
ON memo USING gin(to_tsvector('simple', COALESCE(content, '')));

COMMENT ON INDEX idx_memo_content_gin IS 'GIN index for BM25 full-text search on memo.content';

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
  title_source TEXT NOT NULL DEFAULT 'default',
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
    CHECK (row_status IN ('NORMAL', 'ARCHIVED')),
  CONSTRAINT chk_ai_conversation_title_source
    CHECK (title_source IN ('default', 'auto', 'user'))
);

CREATE INDEX idx_ai_conversation_creator ON ai_conversation(creator_id);
CREATE INDEX idx_ai_conversation_updated ON ai_conversation(updated_ts DESC);
CREATE INDEX idx_ai_conversation_title_source ON ai_conversation(title_source);

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
-- episodic_memory (V0.93.0)
-- Stores episodic memories for AI agents to learn from past interactions
CREATE TABLE episodic_memory (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
  agent_type VARCHAR(20) NOT NULL,
  user_input TEXT NOT NULL,
  outcome VARCHAR(20) NOT NULL DEFAULT 'success',
  summary TEXT,
  importance REAL DEFAULT 0.5,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_episodic_memory_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_episodic_memory_outcome
    CHECK (outcome IN ('success', 'failure')),
  CONSTRAINT chk_episodic_memory_agent_type
    CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant')),
  CONSTRAINT chk_episodic_memory_importance
    CHECK (importance >= 0 AND importance <= 1)
);

-- Indexes for episodic_memory
CREATE INDEX idx_episodic_memory_user_time ON episodic_memory(user_id, timestamp DESC);
CREATE INDEX idx_episodic_memory_agent ON episodic_memory(agent_type);
CREATE INDEX idx_episodic_memory_importance ON episodic_memory(user_id, importance DESC);

COMMENT ON TABLE episodic_memory IS 'Stores episodic memories for AI agents to learn from past interactions';
COMMENT ON COLUMN episodic_memory.agent_type IS 'Type of agent: memo, schedule, amazing, or assistant';
COMMENT ON COLUMN episodic_memory.outcome IS 'Result of the interaction: success or failure';
COMMENT ON COLUMN episodic_memory.importance IS 'Importance score from 0 to 1, used for memory prioritization';

-- episodic_memory_embedding
-- Stores vector embeddings for episodic memories to enable semantic similarity search
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

-- user_preferences (V0.93.0)
-- Stores user preferences for AI personalization
CREATE TABLE user_preferences (
  user_id INTEGER PRIMARY KEY,
  preferences JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_user_preferences_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION update_user_preferences_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_user_preferences_updated_ts
  BEFORE UPDATE ON user_preferences
  FOR EACH ROW
  EXECUTE FUNCTION update_user_preferences_updated_ts();

CREATE INDEX idx_user_preferences_gin ON user_preferences USING gin(preferences);

COMMENT ON TABLE user_preferences IS 'Stores user preferences for AI personalization';
COMMENT ON COLUMN user_preferences.preferences IS 'JSONB containing timezone, default_duration, preferred_times, frequent_locations, communication_style, tag_preferences, and custom_settings';

-- router_feedback (V0.94.0)
-- Stores feedback events for router weight adjustment (Issue #95)
CREATE TABLE router_feedback (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  input TEXT NOT NULL,
  predicted_intent TEXT NOT NULL,
  actual_intent TEXT NOT NULL,
  feedback_type TEXT NOT NULL CHECK (feedback_type IN ('positive', 'rephrase', 'switch')),
  timestamp BIGINT NOT NULL,
  source TEXT NOT NULL,
  CONSTRAINT fk_router_feedback_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

CREATE INDEX idx_router_feedback_user_timestamp ON router_feedback(user_id, timestamp DESC);
CREATE INDEX idx_router_feedback_user_type ON router_feedback(user_id, feedback_type);
CREATE INDEX idx_router_feedback_intent ON router_feedback(predicted_intent);

COMMENT ON TABLE router_feedback IS 'Stores feedback events for router weight adjustment';
COMMENT ON COLUMN router_feedback.feedback_type IS 'positive: no correction, rephrase: user rephrased, switch: user switched agent';

-- router_weight (V0.94.0)
-- Stores per-user keyword weights for dynamic routing (Issue #95)
CREATE TABLE router_weight (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  category TEXT NOT NULL CHECK (category IN ('schedule', 'memo', 'amazing')),
  keyword TEXT NOT NULL,
  weight INTEGER NOT NULL CHECK (weight >= 1 AND weight <= 5),
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_router_weight_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_router_weight_user_category_keyword UNIQUE (user_id, category, keyword)
);

CREATE INDEX idx_router_weight_user ON router_weight(user_id);
CREATE INDEX idx_router_weight_user_category ON router_weight(user_id, category);

CREATE OR REPLACE FUNCTION update_router_weight_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_router_weight_updated_ts
  BEFORE UPDATE ON router_weight
  FOR EACH ROW
  EXECUTE FUNCTION update_router_weight_updated_ts();

COMMENT ON TABLE router_weight IS 'Stores per-user keyword weights for dynamic routing';
COMMENT ON COLUMN router_weight.weight IS 'Keyword weight: 1=min, 5=max (default 2)';

-- conversation_context (V0.93.0)
-- Stores conversation context for AI session persistence and recovery
CREATE TABLE conversation_context (
  id SERIAL PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  agent_type VARCHAR(20) NOT NULL,
  channel_type VARCHAR(20) NOT NULL DEFAULT 'web',
  context_data JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_conversation_context_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_conversation_context_agent_type
    CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant')),
  CONSTRAINT chk_conversation_context_channel_type
    CHECK (channel_type IN ('web', 'telegram', 'whatsapp', 'dingtalk'))
);

CREATE INDEX idx_conversation_context_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_context_updated ON conversation_context(updated_ts DESC);
CREATE INDEX idx_conversation_context_channel_type ON conversation_context(channel_type);

CREATE OR REPLACE FUNCTION update_conversation_context_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_conversation_context_updated_ts
  BEFORE UPDATE ON conversation_context
  FOR EACH ROW
  EXECUTE FUNCTION update_conversation_context_updated_ts();

COMMENT ON TABLE conversation_context IS 'Stores conversation context for AI session persistence and recovery';
COMMENT ON COLUMN conversation_context.session_id IS 'Unique session identifier';
COMMENT ON COLUMN conversation_context.context_data IS 'JSONB containing messages, metadata, and other context information';
COMMENT ON COLUMN conversation_context.channel_type IS 'Origin channel: web, telegram, whatsapp, dingtalk';

-- agent_metrics and tool_metrics (V0.93.0)
-- Hourly aggregated metrics for A/B testing and performance monitoring
CREATE TABLE agent_metrics (
  id SERIAL PRIMARY KEY,
  hour_bucket TIMESTAMP NOT NULL,
  agent_type VARCHAR(20) NOT NULL,
  request_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms BIGINT NOT NULL DEFAULT 0,
  latency_p50_ms INTEGER,
  latency_p95_ms INTEGER,
  errors JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_agent_metrics_hour_type UNIQUE (hour_bucket, agent_type)
);

CREATE INDEX idx_agent_metrics_hour ON agent_metrics (hour_bucket DESC);

CREATE TABLE tool_metrics (
  id SERIAL PRIMARY KEY,
  hour_bucket TIMESTAMP NOT NULL,
  tool_name VARCHAR(50) NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0,
  success_count INTEGER NOT NULL DEFAULT 0,
  latency_sum_ms BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_tool_metrics_hour_name UNIQUE (hour_bucket, tool_name)
);

CREATE INDEX idx_tool_metrics_hour ON tool_metrics (hour_bucket DESC);

-- chat_app_credential (V0.93.0)
-- Stores credentials for chat app integrations (Telegram, WhatsApp, DingTalk)
CREATE TABLE chat_app_credential (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  platform TEXT NOT NULL,
  platform_user_id TEXT NOT NULL,
  platform_chat_id TEXT,
  access_token TEXT,
  app_secret TEXT,
  webhook_url TEXT,
  enabled BOOLEAN DEFAULT true,
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  UNIQUE(user_id, platform),
  CONSTRAINT fk_chat_app_credential_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

CREATE INDEX idx_chat_app_credential_user ON chat_app_credential(user_id);
CREATE INDEX idx_chat_app_credential_platform ON chat_app_credential(platform);
CREATE INDEX idx_chat_app_credential_platform_user_id ON chat_app_credential(platform, platform_user_id);

COMMENT ON TABLE chat_app_credential IS 'Stores credentials for chat app integrations (Telegram, WhatsApp, DingTalk)';
COMMENT ON COLUMN chat_app_credential.platform IS 'Platform name: telegram, whatsapp, dingtalk';
COMMENT ON COLUMN chat_app_credential.platform_user_id IS 'Platform-specific user identifier';
COMMENT ON COLUMN chat_app_credential.platform_chat_id IS 'Platform-specific chat ID for direct messaging';
COMMENT ON COLUMN chat_app_credential.access_token IS 'Encrypted OAuth/Bot token';
COMMENT ON COLUMN chat_app_credential.webhook_url IS 'Webhook URL (DingTalk only)';
COMMENT ON COLUMN chat_app_credential.app_secret IS 'Additional secret for platforms that require it (e.g., DingTalk AppSecret)';

-- =============================================================================
-- Unified Block Model - ai_block (V0.93.0)
-- Core table for unified conversation block storage
-- =============================================================================
CREATE TABLE ai_block (
  id BIGSERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  round_number INTEGER NOT NULL DEFAULT 0,
  block_type TEXT NOT NULL DEFAULT 'message',
  mode TEXT NOT NULL DEFAULT 'normal',
  user_inputs JSONB NOT NULL DEFAULT '[]',
  assistant_content TEXT,
  assistant_timestamp BIGINT,
  event_stream JSONB NOT NULL DEFAULT '[]',
  session_stats JSONB,
  cc_session_id TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  parent_block_id BIGINT DEFAULT 0,
  branch_path TEXT,
  user_feedback TEXT,
  regeneration_count INTEGER DEFAULT 0,
  error_message TEXT,
  model_version TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  cost_estimate BIGINT DEFAULT 0,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  archived_at BIGINT DEFAULT 0,
  CONSTRAINT fk_ai_block_conversation
    FOREIGN KEY (conversation_id)
    REFERENCES ai_conversation(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_block_type
    CHECK (block_type IN ('message', 'context_separator')),
  CONSTRAINT chk_ai_block_mode
    CHECK (mode IN ('normal', 'geek', 'evolution')),
  CONSTRAINT chk_ai_block_status
    CHECK (status IN ('pending', 'streaming', 'completed', 'error'))
);

-- Indexes for ai_block
CREATE INDEX idx_ai_block_conversation ON ai_block(conversation_id);
CREATE INDEX idx_ai_block_created ON ai_block(created_ts ASC);
CREATE INDEX idx_ai_block_round ON ai_block(conversation_id, round_number);
CREATE INDEX idx_ai_block_status ON ai_block(status) WHERE status != 'completed';
CREATE INDEX idx_ai_block_cc_session ON ai_block(cc_session_id) WHERE cc_session_id IS NOT NULL;
CREATE INDEX idx_ai_block_conversation_status_round ON ai_block(conversation_id, status, round_number);
CREATE INDEX idx_ai_block_pending_streaming ON ai_block(conversation_id) WHERE status IN ('pending', 'streaming');
CREATE INDEX idx_ai_block_event_stream ON ai_block USING gin(event_stream);
CREATE INDEX idx_ai_block_user_inputs ON ai_block USING gin(user_inputs);
CREATE INDEX idx_ai_block_cc_session_conversation ON ai_block(cc_session_id, conversation_id) WHERE cc_session_id IS NOT NULL;
CREATE INDEX idx_ai_block_parent ON ai_block(parent_block_id) WHERE parent_block_id IS NOT NULL;
CREATE INDEX idx_ai_block_branch_path ON ai_block(branch_path) WHERE branch_path IS NOT NULL;
CREATE INDEX idx_ai_block_cost ON ai_block(cost_estimate) WHERE cost_estimate > 0;
CREATE INDEX idx_ai_block_feedback ON ai_block(user_feedback) WHERE user_feedback IS NOT NULL;
CREATE INDEX idx_ai_block_archived ON ai_block(archived_at) WHERE archived_at IS NOT NULL;
CREATE INDEX idx_ai_block_model ON ai_block(model_version) WHERE model_version IS NOT NULL;

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_ai_block_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_block_updated_ts
  BEFORE UPDATE ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_block_updated_ts();

-- Auto-increment round_number trigger
CREATE OR REPLACE FUNCTION ai_block_auto_round_number()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.round_number = 0 THEN
    SELECT COALESCE(MAX(round_number), 0) + 1
    INTO NEW.round_number
    FROM ai_block
    WHERE conversation_id = NEW.conversation_id;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_block_auto_round
  BEFORE INSERT ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION ai_block_auto_round_number();

-- Compatibility view for ai_message
CREATE VIEW v_ai_message AS
SELECT
  id,
  uid,
  conversation_id,
  'MESSAGE' as type,
  CASE
    WHEN block_type = 'context_separator' THEN 'SEPARATOR'
    ELSE 'MESSAGE'
  END as message_type,
  CASE
    WHEN jsonb_array_length(user_inputs) > 0
    THEN (user_inputs->0->>'content')
    ELSE ''
  END as user_content,
  assistant_content as content,
  metadata,
  created_ts
FROM ai_block
WHERE block_type = 'message';


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
