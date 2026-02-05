-- Add parent_block_id for tree branching support
-- This enables conversation forking and "edit & regenerate" functionality
-- Phase: UBM Improvement (P0) - unified-block-model_improvement.md

-- =============================================================================
-- Part 1: Add parent_block_id column to ai_block
-- =============================================================================

-- Add parent_block_id column (nullable, defaults to NULL for root blocks)
ALTER TABLE ai_block
  ADD COLUMN parent_block_id BIGINT,
  ADD COLUMN branch_path TEXT;

-- Add foreign key constraint (optional - can be removed for performance)
-- Commented out for now to allow maximum flexibility
-- ALTER TABLE ai_block
--   ADD CONSTRAINT fk_ai_block_parent
--   FOREIGN KEY (parent_block_id)
--   REFERENCES ai_block(id)
--   ON DELETE SET NULL;

-- Add index for parent lookups (useful for fetching conversation branches)
CREATE INDEX idx_ai_block_parent ON ai_block(parent_block_id) WHERE parent_block_id IS NOT NULL;

-- Add index for branch path queries (for ordering within branches)
CREATE INDEX idx_ai_block_branch_path ON ai_block(branch_path) WHERE branch_path IS NOT NULL;

-- =============================================================================
-- Part 2: Update check constraint for block_type
-- =============================================================================

-- No changes needed - existing constraints are fine

-- =============================================================================
-- Part 3: Update compatibility view
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

-- =============================================================================
-- Part 4: Update trigger to include branch path logic
-- =============================================================================

-- Update round_number trigger to also set branch_path for new blocks
DROP TRIGGER IF EXISTS trigger_ai_block_round_number ON ai_block;

-- Function to auto-generate round_number and branch_path
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

CREATE TRIGGER trigger_ai_block_round_number
  BEFORE INSERT ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION auto_generate_block_attributes();

-- =============================================================================
-- Version update
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.57.0', 'Database schema version - Tree Branching Support')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
