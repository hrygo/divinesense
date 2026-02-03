-- Rollback: Drop conversation_context table
DROP TRIGGER IF EXISTS trigger_conversation_context_updated_ts ON conversation_context;
DROP FUNCTION IF EXISTS update_conversation_context_updated_ts();
DROP INDEX IF EXISTS idx_conversation_context_updated;
DROP INDEX IF EXISTS idx_conversation_context_user;
DROP TABLE IF EXISTS conversation_context;
