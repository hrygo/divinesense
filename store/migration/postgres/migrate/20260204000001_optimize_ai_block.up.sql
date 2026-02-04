-- =============================================================================
-- Unified Block Model Optimization (V0.60.1)
-- =============================================================================

-- 1. Add trigger function to auto-calculate round_number
CREATE OR REPLACE FUNCTION ai_block_auto_round_number()
RETURNS TRIGGER AS $$
DECLARE
    next_round INTEGER;
BEGIN
    -- Calculate next round number for this conversation
    SELECT COALESCE(MAX(round_number), -1) + 1
    INTO next_round
    FROM ai_block
    WHERE conversation_id = NEW.conversation_id;

    NEW.round_number := next_round;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Drop existing trigger if exists (for idempotency)
DROP TRIGGER IF EXISTS trigger_ai_block_auto_round ON ai_block;

-- 3. Add trigger to auto-set round_number before insert
CREATE TRIGGER trigger_ai_block_auto_round
    BEFORE INSERT ON ai_block
    FOR EACH ROW
    WHEN (NEW.round_number IS NULL OR NEW.round_number = 0)
    EXECUTE FUNCTION ai_block_auto_round_number();

-- 4. Add composite index for common queries
-- Covers: GetLatestBlock, ListBlocks by conversation + status
CREATE INDEX IF NOT EXISTS idx_ai_block_conversation_status_round
    ON ai_block(conversation_id, status, round_number DESC);

-- 5. Add partial index for pending/streaming blocks (cleanup queries)
CREATE INDEX IF NOT EXISTS idx_ai_block_pending_streaming
    ON ai_block(created_ts ASC)
    WHERE status IN ('pending', 'streaming');

-- 6. Optimize JSONB GIN indexes with jsonb_path_ops
DROP INDEX IF EXISTS idx_ai_block_event_stream;
DROP INDEX IF EXISTS idx_ai_block_user_inputs;

CREATE INDEX idx_ai_block_event_stream
    ON ai_block USING gin(event_stream jsonb_path_ops)
    WHERE event_stream IS NOT NULL AND jsonb_array_length(event_stream) > 0;

CREATE INDEX idx_ai_block_user_inputs
    ON ai_block USING gin(user_inputs jsonb_path_ops)
    WHERE user_inputs IS NOT NULL AND jsonb_array_length(user_inputs) > 0;

-- 7. Add index for CC session lookups (Geek/Evolution mode)
CREATE INDEX IF NOT EXISTS idx_ai_block_cc_session_conversation
    ON ai_block(cc_session_id, conversation_id)
    WHERE cc_session_id IS NOT NULL;

-- 8. Add constraint to prevent duplicate round numbers (redundant but explicit)
-- Note: This is already enforced by the unique constraint, but we keep it for clarity
-- The trigger ensures round_number is always unique per conversation
