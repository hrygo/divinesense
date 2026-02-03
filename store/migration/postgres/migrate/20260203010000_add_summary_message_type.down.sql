-- Rollback: Remove SUMMARY message type
ALTER TABLE ai_message DROP CONSTRAINT IF EXISTS chk_ai_message_type;
ALTER TABLE ai_message ADD CONSTRAINT chk_ai_message_type
  CHECK (type IN ('MESSAGE', 'SEPARATOR'));
