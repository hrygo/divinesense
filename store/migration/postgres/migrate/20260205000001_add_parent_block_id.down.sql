-- Revert parent_block_id changes
-- Down migration for tree branching support

-- =============================================================================
-- Part 1: Drop trigger and function
-- =============================================================================

DROP TRIGGER IF EXISTS trigger_ai_block_round_number ON ai_block;
DROP FUNCTION IF EXISTS auto_generate_block_attributes();

-- Recreate simpler trigger (original version)
CREATE OR REPLACE FUNCTION auto_generate_block_attributes()
RETURNS TRIGGER AS $$
BEGIN
  -- Auto-increment round_number within the conversation
  SELECT COALESCE(MAX(round_number), -1) + 1
  INTO NEW.round_number
  FROM ai_block
  WHERE conversation_id = NEW.conversation_id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_block_round_number
  BEFORE INSERT ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION auto_generate_block_attributes();

-- =============================================================================
-- Part 2: Drop compatibility view
-- =============================================================================

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
-- Part 3: Drop indexes
-- =============================================================================

DROP INDEX IF EXISTS idx_ai_block_branch_path;
DROP INDEX IF EXISTS idx_ai_block_parent;

-- =============================================================================
-- Part 4: Drop columns
-- =============================================================================

ALTER TABLE ai_block
  DROP COLUMN IF EXISTS parent_block_id,
  DROP COLUMN IF EXISTS branch_path;

-- =============================================================================
-- Version revert
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.56.0', 'Database schema version - Timestamps in milliseconds')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
