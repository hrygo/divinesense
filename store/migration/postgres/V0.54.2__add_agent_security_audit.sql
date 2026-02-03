-- Security audit log table for tracking dangerous operations
-- This table logs security-relevant events in Geek/Evolution modes
CREATE TABLE agent_security_audit (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64),

    -- User information
    user_id INTEGER NOT NULL,
    agent_type VARCHAR(20) NOT NULL,

    -- Operation information
    operation_type VARCHAR(50) NOT NULL, -- 'danger_block', 'tool_use', etc.
    operation_name VARCHAR(100), -- 'rm -rf', 'format', etc.

    -- Risk assessment
    risk_level VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'

    -- Command details
    command_input TEXT,
    command_matched_pattern TEXT,

    -- Action taken
    action_taken VARCHAR(50), -- 'blocked', 'allowed', 'logged_only'
    reason TEXT,

    -- Additional context
    file_path TEXT,
    tool_id VARCHAR(100),

    -- Timestamp
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_audit_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE,
    CONSTRAINT chk_risk_level CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

-- Indexes for security queries
CREATE INDEX idx_audit_user_date ON agent_security_audit(user_id, occurred_at DESC);
CREATE INDEX idx_audit_risk ON agent_security_audit(risk_level, occurred_at DESC);
CREATE INDEX idx_audit_operation ON agent_security_audit(operation_type, occurred_at DESC);
CREATE INDEX idx_audit_session ON agent_security_audit(session_id) WHERE session_id IS NOT NULL;
