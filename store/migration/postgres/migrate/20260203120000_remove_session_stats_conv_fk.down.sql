-- Rollback: Restore foreign key constraint on conversation_id
ALTER TABLE agent_session_stats
ADD CONSTRAINT fk_session_stats_conv
FOREIGN KEY (conversation_id) REFERENCES ai_conversation(id) ON DELETE CASCADE;

COMMENT ON COLUMN agent_session_stats.conversation_id IS
'Conversation ID reference';
