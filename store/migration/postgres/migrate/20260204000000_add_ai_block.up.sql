-- Unified Block Model - Phase 1
-- Add ai_block table for unified conversation block storage
-- This unifies normal mode and CC mode (Geek/Evolution) data structures

-- =============================================================================
-- ai_block Table
-- =============================================================================
CREATE TABLE ai_block (
  id BIGSERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  round_number INTEGER NOT NULL DEFAULT 0,

  -- Block type
  block_type TEXT NOT NULL DEFAULT 'message',
  -- 'message': User-AI conversation round
  -- 'context_separator': Context separator marker

  -- AI mode
  mode TEXT NOT NULL DEFAULT 'normal',
  -- 'normal': Normal AI assistant mode
  -- 'geek': Geek mode (Claude Code CLI)
  -- 'evolution': Evolution mode (self-improvement)

  -- User inputs (support append mode)
  user_inputs JSONB NOT NULL DEFAULT '[]',
  -- [{"content": "Input content", "timestamp": 1234567890, "metadata": {...}}]

  -- AI response
  assistant_content TEXT,
  assistant_timestamp BIGINT,

  -- Event stream (chronological order)
  event_stream JSONB NOT NULL DEFAULT '[]',
  -- [{type: "thinking", content: "...", timestamp: ..., meta: {...}}, ...]

  -- Session statistics (CC mode)
  session_stats JSONB,
  -- {session_id: "...", total_cost_usd: 0.0123, total_tokens: 1234, ...}

  -- CC session mapping
  cc_session_id TEXT,
  -- UUID v5 mapped to Claude Code CLI session

  -- Status
  status TEXT NOT NULL DEFAULT 'pending',
  -- 'pending': Waiting for AI response
  -- 'streaming': AI is currently responding
  -- 'completed': Response completed
  -- 'error': Error occurred

  -- Extension fields
  metadata JSONB NOT NULL DEFAULT '{}',
  -- {error_message: "...", parrot_id: "MEMO", ...}

  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),

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

-- =============================================================================
-- Compatibility View
-- =============================================================================
-- Preserve compatibility with existing ai_message table structure
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
  -- Extract first user input from user_inputs
  CASE
    WHEN jsonb_array_length(user_inputs) > 0
    THEN (user_inputs->0->>'content')
    ELSE ''
  END as user_content,
  assistant_content as content,
  metadata,
  created_ts
FROM (
  SELECT
    id,
    uid,
    conversation_id,
    block_type,
    mode,
    user_inputs,
    assistant_content,
    event_stream,
    session_stats,
    metadata,
    created_ts,
    -- For compatibility, merge mode and error info into metadata
    jsonb_build_object(
      'mode', mode,
      'error', CASE WHEN status = 'error' THEN metadata->>'error_message' ELSE NULL END,
      'event_stream', event_stream,
      'session_stats', session_stats
    ) || metadata as metadata_full,
    created_ts
  FROM ai_block
  WHERE block_type = 'message'
) expanded;

-- =============================================================================
-- Version update
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.55.0', 'Database schema version - Unified Block Model')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
