-- Unified Block Model - Phase 1 Rollback
-- Remove ai_block table and compatibility view

DROP VIEW IF EXISTS v_ai_message;

DROP INDEX IF EXISTS idx_ai_block_event_stream;
DROP INDEX IF EXISTS idx_ai_block_cc_session;
DROP INDEX IF EXISTS idx_ai_block_status;
DROP INDEX IF EXISTS idx_ai_block_round;
DROP INDEX IF EXISTS idx_ai_block_created;
DROP INDEX IF EXISTS idx_ai_block_conversation;

DROP TRIGGER IF EXISTS trigger_ai_block_updated_ts ON ai_block;
DROP FUNCTION IF EXISTS update_ai_block_updated_ts();

DROP TABLE IF EXISTS ai_block;

-- Reset version
UPDATE system_setting SET value = '0.54.3' WHERE name = 'schema_version';
