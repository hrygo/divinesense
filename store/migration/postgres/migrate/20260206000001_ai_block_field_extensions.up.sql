-- Block Field Extensions - P1-A006 + ai-block-fields-extension
-- Add token usage, cost tracking, user feedback, and archival support to ai_block
-- Phase: Session Summary Enhancement (Issue #79)

-- =============================================================================
-- Part 1: Add token_usage column
-- =============================================================================

-- Token usage stored as JSONB for detailed breakdown
ALTER TABLE ai_block
  ADD COLUMN token_usage JSONB DEFAULT '{
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0,
    "cache_read_tokens": 0,
    "cache_write_tokens": 0
  }';

-- =============================================================================
-- Part 2: Add cost_estimate column
-- =============================================================================

-- Cost stored in milli-cents (1/1000 of a US cent, or 1/100000 USD)
-- For example: $0.0123 = 1230 milli-cents
ALTER TABLE ai_block
  ADD COLUMN cost_estimate BIGINT DEFAULT 0;

-- =============================================================================
-- Part 3: Add model_version column
-- =============================================================================

-- Track which LLM model was used for this block
ALTER TABLE ai_block
  ADD COLUMN model_version TEXT;

-- =============================================================================
-- Part 4: Add user_feedback column
-- =============================================================================

-- User feedback: "thumbs_up", "thumbs_down", or custom text
ALTER TABLE ai_block
  ADD COLUMN user_feedback TEXT;

-- =============================================================================
-- Part 5: Add regeneration_count column
-- =============================================================================

-- Track how many times this block was regenerated
ALTER TABLE ai_block
  ADD COLUMN regeneration_count INTEGER DEFAULT 0;

-- =============================================================================
-- Part 6: Add error_message column
-- =============================================================================

-- Separate error message from metadata for easier querying
ALTER TABLE ai_block
  ADD COLUMN error_message TEXT;

-- =============================================================================
-- Part 7: Add archived_at column
-- =============================================================================

-- Track when blocks were archived (NULL if active)
ALTER TABLE ai_block
  ADD COLUMN archived_at BIGINT;

-- =============================================================================
-- Part 8: Create indexes for new fields
-- =============================================================================

-- Index for cost tracking queries
CREATE INDEX idx_ai_block_cost ON ai_block(cost_estimate) WHERE cost_estimate > 0;

-- Index for user feedback queries
CREATE INDEX idx_ai_block_feedback ON ai_block(user_feedback) WHERE user_feedback IS NOT NULL;

-- Index for archived blocks
CREATE INDEX idx_ai_block_archived ON ai_block(archived_at) WHERE archived_at IS NOT NULL;

-- Index for model version analytics
CREATE INDEX idx_ai_block_model ON ai_block(model_version) WHERE model_version IS NOT NULL;

-- =============================================================================
-- Part 9: Update compatibility view
-- =============================================================================

-- Drop and recreate view with new columns
DROP VIEW IF EXISTS v_ai_message;

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
  created_ts,
  parent_block_id,
  branch_path,
  -- New fields
  token_usage,
  cost_estimate,
  model_version,
  user_feedback,
  regeneration_count,
  error_message,
  archived_at
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
    parent_block_id,
    branch_path,
    token_usage,
    cost_estimate,
    model_version,
    user_feedback,
    regeneration_count,
    error_message,
    archived_at,
    -- For compatibility, merge mode and error info into metadata
    jsonb_build_object(
      'mode', mode,
      'error', COALESCE(error_message, CASE WHEN status = 'error' THEN metadata->>'error_message' ELSE NULL END),
      'event_stream', event_stream,
      'session_stats', session_stats,
      'parent_block_id', parent_block_id,
      'branch_path', branch_path,
      'token_usage', token_usage,
      'cost_estimate', cost_estimate,
      'model_version', model_version,
      'user_feedback', user_feedback,
      'regeneration_count', regeneration_count,
      'archived_at', archived_at
    ) || metadata as metadata_full,
    created_ts
  FROM ai_block
  WHERE block_type = 'message'
) expanded;

-- =============================================================================
-- Part 10: Update trigger to set default values
-- =============================================================================

-- Update the auto_generate_block_attributes function to set defaults
CREATE OR REPLACE FUNCTION auto_generate_block_attributes()
RETURNS TRIGGER AS $$
DECLARE
  v_round_number INTEGER;
  v_branch_path TEXT;
  v_parent_round INTEGER;
BEGIN
  -- Auto-increment round_number within the conversation
  SELECT COALESCE(MAX(round_number), -1) + 1
  INTO v_round_number
  FROM ai_block
  WHERE conversation_id = NEW.conversation_id;

  NEW.round_number = v_round_number;

  -- Generate branch_path if parent_block_id is set
  IF NEW.parent_block_id IS NOT NULL THEN
    -- Get parent's branch_path and round_number
    SELECT branch_path, round_number
    INTO v_branch_path, v_parent_round
    FROM ai_block
    WHERE id = NEW.parent_block_id;

    -- Create new branch_path: parent_path/parent_round/new_round
    -- Example: "0/1" becomes "0/1/3" for new child
    IF v_branch_path IS NOT NULL THEN
      NEW.branch_path = v_branch_path || '/' || v_parent_round;
    ELSE
      NEW.branch_path := v_parent_round::TEXT;
    END IF;
  ELSE
    -- Root block: branch_path is just the round_number
    NEW.branch_path := v_round_number::TEXT;
  END IF;

  -- Set default token_usage if not provided
  IF NEW.token_usage IS NULL THEN
    NEW.token_usage := '{
      "prompt_tokens": 0,
      "completion_tokens": 0,
      "total_tokens": 0,
      "cache_read_tokens": 0,
      "cache_write_tokens": 0
    }'::jsonb;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Version update
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.58.0', 'Database schema version - Block Field Extensions')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
