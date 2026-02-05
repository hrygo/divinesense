-- Revert timestamp conversion (milliseconds back to seconds)
-- Down migration for timestamp fix

-- =============================================================================
-- Part 1: Revert ai_block table timestamps
-- =============================================================================

-- Convert timestamps back from milliseconds to seconds
-- Values >= 1735689600000 (2025-01-01 in ms) are assumed to be in milliseconds
UPDATE ai_block
SET
  created_ts = created_ts / 1000,
  updated_ts = updated_ts / 1000
WHERE created_ts >= 1735689600000;

-- Revert event_stream timestamps
UPDATE ai_block
SET event_stream = (
  SELECT jsonb_agg(
    jsonb_set(
      event,
      '{timestamp}',
      (COALESCE((event->>'timestamp')::bigint, 0) /
        CASE
          WHEN (event->>'timestamp')::bigint >= 1735689600000 THEN 1000
          ELSE 1
        END
      )::text::jsonb
    )
  )
  FROM jsonb_array_elements(event_stream) AS event
)
WHERE event_stream != '[]'::jsonb;

-- Revert user_inputs timestamps
UPDATE ai_block
SET user_inputs = (
  SELECT jsonb_agg(
    jsonb_set(
      input,
      '{timestamp}',
      (COALESCE((input->>'timestamp')::bigint, 0) /
        CASE
          WHEN (input->>'timestamp')::bigint >= 1735689600000 THEN 1000
          ELSE 1
        END
      )::text::jsonb
    )
  )
  FROM jsonb_array_elements(user_inputs) AS input
)
WHERE user_inputs != '[]'::jsonb;

-- =============================================================================
-- Part 2: Revert other timestamp columns
-- =============================================================================

UPDATE conversation_context
SET
  created_ts = created_ts / 1000,
  updated_ts = updated_ts / 1000
WHERE created_ts >= 1735689600000;

UPDATE episodic_memory
SET
  created_ts = created_ts / 1000,
  updated_ts = updated_ts / 1000
WHERE created_ts >= 1735689600000;

UPDATE user_preferences
SET
  created_ts = created_ts / 1000,
  updated_ts = updated_ts / 1000
WHERE created_ts >= 1735689600000;

-- =============================================================================
-- Part 3: Revert database trigger to seconds
-- =============================================================================

DROP TRIGGER IF EXISTS trigger_ai_block_updated_ts ON ai_block;
DROP FUNCTION IF EXISTS update_ai_block_updated_ts();

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
-- Part 4: Revert default values
-- =============================================================================

ALTER TABLE ai_block
  ALTER COLUMN created_ts
    SET DEFAULT EXTRACT(EPOCH FROM NOW()),
  ALTER COLUMN updated_ts
    SET DEFAULT EXTRACT(EPOCH FROM NOW());

-- =============================================================================
-- Version revert
-- =============================================================================
INSERT INTO system_setting (name, value, description) VALUES
('schema_version', '0.55.0', 'Database schema version - Unified Block Model')
ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
