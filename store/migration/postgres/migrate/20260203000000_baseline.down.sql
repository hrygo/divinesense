-- =============================================================================
-- Rollback Memos 0.51.0 Baseline Migration
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Drop AI Conversation Tables
-- -----------------------------------------------------------------------------
DROP TRIGGER IF EXISTS trigger_ai_conversation_updated_ts ON ai_conversation;
DROP FUNCTION IF EXISTS update_ai_conversation_updated_ts();
DROP INDEX IF EXISTS idx_ai_message_created;
DROP INDEX IF EXISTS idx_ai_message_conversation;
DROP TABLE IF EXISTS ai_message;
DROP INDEX IF EXISTS idx_ai_conversation_updated;
DROP INDEX IF EXISTS idx_ai_conversation_creator;
DROP TABLE IF EXISTS ai_conversation;

-- -----------------------------------------------------------------------------
-- Drop Schedule Table
-- -----------------------------------------------------------------------------
DROP TRIGGER IF EXISTS trigger_schedule_updated_ts ON schedule;
DROP FUNCTION IF EXISTS update_schedule_updated_ts();
DROP INDEX IF EXISTS idx_schedule_uid;
DROP INDEX IF EXISTS idx_schedule_start_ts;
DROP INDEX IF EXISTS idx_schedule_creator_status;
DROP INDEX IF EXISTS idx_schedule_creator_start;
DROP TABLE IF EXISTS schedule;

-- -----------------------------------------------------------------------------
-- Clean Schema Version
-- -----------------------------------------------------------------------------
DELETE FROM system_setting WHERE name = 'schema_version' AND value = '0.51.0';
