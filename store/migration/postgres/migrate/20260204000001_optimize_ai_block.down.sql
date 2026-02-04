-- =============================================================================
-- Rollback Unified Block Model Optimization
-- =============================================================================

-- 1. Drop indexes
DROP INDEX IF EXISTS idx_ai_block_cc_session_conversation;
DROP INDEX IF EXISTS idx_ai_block_user_inputs;
DROP INDEX IF EXISTS idx_ai_block_event_stream;
DROP INDEX IF EXISTS idx_ai_block_pending_streaming;
DROP INDEX IF EXISTS idx_ai_block_conversation_status_round;

-- 2. Drop trigger
DROP TRIGGER IF EXISTS trigger_ai_block_auto_round ON ai_block;

-- 3. Drop trigger function
DROP FUNCTION IF EXISTS ai_block_auto_round_number();

-- 4. Restore original GIN indexes (without jsonb_path_ops)
CREATE INDEX IF NOT EXISTS idx_ai_block_event_stream
    ON ai_block USING gin(event_stream);

CREATE INDEX IF NOT EXISTS idx_ai_block_user_inputs
    ON ai_block USING gin(user_inputs);
