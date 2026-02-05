-- Convert timestamps from seconds to milliseconds
-- This fixes the inconsistency where backend used seconds but frontend expects milliseconds
-- Phase: UBM Improvement (P0) - unified-block-model_improvement.md

-- =============================================================================
-- Part 1: Update ai_block table timestamps
-- =============================================================================

-- Convert existing timestamps from seconds to milliseconds
-- Values before 2025-01-01 (1735689600000 ms) are assumed to be in seconds
UPDATE ai_block
SET
  created_ts = created_ts * 1000,
  updated_ts = updated_ts * 1000
WHERE created_ts < 1735689600000;  -- 2025-01-01 in milliseconds

-- Also fix event_stream timestamps (JSONB array)
UPDATE ai_block
SET event_stream = (
  SELECT jsonb_agg(
    jsonb_set(
      event,
      '{timestamp}',
      (COALESCE((event->>'timestamp')::bigint, 0) *
        CASE
          WHEN (event->>'timestamp')::bigint < 1735689600000 THEN 1000
          ELSE 1
        END
      )::text::jsonb
    )
  )
  FROM jsonb_array_elements(event_stream) AS event
)
WHERE event_stream != '[]'::jsonb;

-- Also fix user_inputs timestamps (JSONB array)
UPDATE ai_block
SET user_inputs = (
  SELECT jsonb_agg(
    jsonb_set(
      input,
      '{timestamp}',
      (COALESCE((input->>'timestamp')::bigint, 0) *
        CASE
          WHEN (input->>'timestamp')::bigint < 1735689600000 THEN 1000
          ELSE 1
        END
      )::text::jsonb
    )
  )
  FROM jsonb_array_elements(user_inputs) AS input
)
WHERE user_inputs != '[]'::jsonb;

-- =============================================================================
-- Part 2: Update other timestamp columns (for consistency)
-- =============================================================================

-- Update conversation_context table
UPDATE conversation_context
SET
  created_ts = created_ts * 1000,
  updated_ts = updated_ts * 1000
WHERE created_ts < 1735689600000;

-- Update episodic_memory table
UPDATE episodic_memory
SET
  created_ts = created_ts * 1000,
  updated_ts = updated_ts * 1000
WHERE created_ts < 1735689600000;

-- Update user_preferences table
UPDATE user_preferences
SET
  created_ts = created_ts * 1000,
  updated_ts = updated_ts * 1000
WHERE created_ts < 1735689600000;

-- =============================================================================
-- Part 3: Update database trigger to use milliseconds
-- =============================================================================

-- Drop old trigger function
DROP TRIGGER IF EXISTS trigger_ai_block_updated_ts ON ai_block;
DROP FUNCTION IF EXISTS update_ai_block_updated_ts();

-- Create new trigger function that returns milliseconds
CREATE OR REPLACE FUNCTION update_ai_block_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW()) * 1000::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_block_updated_ts
  BEFORE UPDATE ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_block_updated_ts();

-- =============================================================================
-- Part 4: Update default values for ai_block
-- =============================================================================

-- Alter table to use milliseconds in default values
ALTER TABLE ai_block
  ALTER COLUMN created_ts
    SET DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000::BIGINT,
  ALTER COLUMN updated_ts
    SET DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000::BIGINT;

-- =============================================================================
-- Version update
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.56.0', 'Database schema version - Timestamps in milliseconds')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
