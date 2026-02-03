-- Agent session statistics table for Geek/Evolution mode tracking
-- This table stores detailed statistics about each AI agent session
-- including tokens, costs, duration, tools used, and files modified
CREATE TABLE agent_session_stats (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL UNIQUE,
    conversation_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    agent_type VARCHAR(20) NOT NULL, -- 'geek', 'evolution'

    -- Time dimensions
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL,
    total_duration_ms BIGINT NOT NULL,
    thinking_duration_ms BIGINT NOT NULL DEFAULT 0,
    tool_duration_ms BIGINT NOT NULL DEFAULT 0,
    generation_duration_ms BIGINT NOT NULL DEFAULT 0,

    -- Token usage
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_write_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,

    -- Cost tracking
    total_cost_usd NUMERIC(10,4) NOT NULL DEFAULT 0,

    -- Tool usage
    tool_call_count INTEGER NOT NULL DEFAULT 0,
    tools_used JSONB, -- ["Bash", "editor_write", ...]

    -- File operations
    files_modified INTEGER NOT NULL DEFAULT 0,
    file_paths TEXT[], -- ["path1", "path2", ...]

    -- Model information
    model_used VARCHAR(100),

    -- Status tracking
    is_error BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_session_stats_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    CONSTRAINT fk_session_stats_conv FOREIGN KEY (conversation_id) REFERENCES ai_conversation(id) ON DELETE CASCADE,
    CONSTRAINT chk_agent_session_stats_type CHECK (agent_type IN ('geek', 'evolution'))
);

-- Indexes for common queries
CREATE INDEX idx_session_stats_user_date ON agent_session_stats(user_id, started_at DESC);
CREATE INDEX idx_session_stats_conv ON agent_session_stats(conversation_id);
CREATE INDEX idx_session_stats_agent ON agent_session_stats(agent_type, started_at DESC);
CREATE INDEX idx_session_stats_cost ON agent_session_stats(total_cost_usd) WHERE total_cost_usd > 0;
-- Partial index for successful sessions (is_error=false) - optimizes cost queries
CREATE INDEX idx_session_stats_user_success ON agent_session_stats(user_id, started_at DESC) WHERE is_error = false;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_agent_session_stats_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_agent_session_stats_updated_at
  BEFORE UPDATE ON agent_session_stats
  FOR EACH ROW
  EXECUTE FUNCTION update_agent_session_stats_updated_at();
