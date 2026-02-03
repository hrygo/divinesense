-- =============================================================================
-- Memos 0.51.0 Baseline Migration
-- =============================================================================
--
-- Purpose: Sync databases below 0.51.0 to current schema
-- Scenario: Upgrading from legacy versions (0.19 - 0.50) to 0.51.0
--
-- Note: This script uses IF NOT EXISTS to ensure idempotency
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Schedule Table (0.26)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS schedule (
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
    CHECK (reminders ~ '^(\\[\\]|\\[\\{.*\\}\\])$')
);

CREATE INDEX IF NOT EXISTS idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX IF NOT EXISTS idx_schedule_creator_status ON schedule(creator_id, row_status);
CREATE INDEX IF NOT EXISTS idx_schedule_start_ts ON schedule(start_ts);
CREATE INDEX IF NOT EXISTS idx_schedule_uid ON schedule(uid);

CREATE OR REPLACE FUNCTION update_schedule_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
CREATE TRIGGER trigger_schedule_updated_ts
  BEFORE UPDATE ON schedule
  FOR EACH ROW
  EXECUTE FUNCTION update_schedule_updated_ts();

-- -----------------------------------------------------------------------------
-- AI Conversation Tables (0.34)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ai_conversation (
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

CREATE INDEX IF NOT EXISTS idx_ai_conversation_creator ON ai_conversation(creator_id);
CREATE INDEX IF NOT EXISTS idx_ai_conversation_updated ON ai_conversation(updated_ts DESC);

CREATE TABLE IF NOT EXISTS ai_message (
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

CREATE INDEX IF NOT EXISTS idx_ai_message_conversation ON ai_message(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_message_created ON ai_message(created_ts ASC);

CREATE OR REPLACE FUNCTION update_ai_conversation_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_ai_conversation_updated_ts ON ai_conversation;
CREATE TRIGGER trigger_ai_conversation_updated_ts
  BEFORE UPDATE ON ai_conversation
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_conversation_updated_ts();

-- -----------------------------------------------------------------------------
-- Update Schema Version
-- -----------------------------------------------------------------------------
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.51.0', 'Database schema version')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
