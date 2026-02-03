-- Rollback: Remove channel_type column from conversation_context
DROP INDEX IF EXISTS idx_conversation_context_channel_type;
ALTER TABLE conversation_context DROP CONSTRAINT IF EXISTS check_channel_type;
ALTER TABLE conversation_context DROP COLUMN IF EXISTS channel_type;
