-- Add conversation_context table for AI session persistence
-- Stores conversation context for session recovery and continuity

CREATE TABLE conversation_context (
  id SERIAL PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL UNIQUE,
  user_id INTEGER NOT NULL,
  agent_type VARCHAR(20) NOT NULL,
  context_data JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_conversation_context_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_conversation_context_agent_type
    CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant'))
);

-- Index for querying user's sessions
CREATE INDEX idx_conversation_context_user
ON conversation_context(user_id);

-- Index for sorting by update time
CREATE INDEX idx_conversation_context_updated
ON conversation_context(updated_ts DESC);

-- Note: session_id already has a UNIQUE constraint which creates an implicit index,
-- so no separate index is needed for session_id lookups.

-- Trigger to auto-update updated_ts
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
