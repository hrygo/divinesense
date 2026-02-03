-- Add SUMMARY message type for conversation summarization
-- SUMMARY type stores conversation summaries that are:
-- - Invisible to frontend (filtered in API responses)
-- - Sent to LLM as context prefix (in ContextBuilder)
-- - Preceded by a SEPARATOR message
-- - Not counted in the 100 message limit

-- Modify the type constraint to include SUMMARY
ALTER TABLE ai_message DROP CONSTRAINT IF EXISTS chk_ai_message_type;
ALTER TABLE ai_message ADD CONSTRAINT chk_ai_message_type
  CHECK (type IN ('MESSAGE', 'SEPARATOR', 'SUMMARY'));
