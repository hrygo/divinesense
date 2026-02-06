-- Rollback Block Field Extensions
-- Remove columns added in 20260206000001_ai_block_field_extensions.up.sql

-- Drop indexes
DROP INDEX IF EXISTS idx_ai_block_model;
DROP INDEX IF EXISTS idx_ai_block_archived;
DROP INDEX IF EXISTS idx_ai_block_feedback;
DROP INDEX IF EXISTS idx_ai_block_cost;

-- Drop columns
ALTER TABLE ai_block DROP COLUMN IF EXISTS archived_at;
ALTER TABLE ai_block DROP COLUMN IF EXISTS error_message;
ALTER TABLE ai_block DROP COLUMN IF EXISTS regeneration_count;
ALTER TABLE ai_block DROP COLUMN IF EXISTS user_feedback;
ALTER TABLE ai_block DROP COLUMN IF EXISTS model_version;
ALTER TABLE ai_block DROP COLUMN IF EXISTS cost_estimate;
ALTER TABLE ai_block DROP COLUMN IF EXISTS token_usage;

-- Recreate compatibility view without new columns
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
  branch_path
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
    -- For compatibility, merge mode and error info into metadata
    jsonb_build_object(
      'mode', mode,
      'error', CASE WHEN status = 'error' THEN metadata->>'error_message' ELSE NULL END,
      'event_stream', event_stream,
      'session_stats', session_stats,
      'parent_block_id', parent_block_id,
      'branch_path', branch_path
    ) || metadata as metadata_full,
    created_ts
  FROM ai_block
  WHERE block_type = 'message'
) expanded;

-- Restore trigger function
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

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
