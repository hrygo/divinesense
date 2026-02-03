-- Agent metrics table for hourly aggregated agent performance data
CREATE TABLE agent_metrics (
    id SERIAL PRIMARY KEY,
    hour_bucket TIMESTAMP NOT NULL,
    agent_type VARCHAR(20) NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    latency_sum_ms BIGINT NOT NULL DEFAULT 0,
    latency_p50_ms INTEGER,
    latency_p95_ms INTEGER,
    errors JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_agent_metrics_hour_type UNIQUE (hour_bucket, agent_type)
);

-- Index for time-range queries
CREATE INDEX idx_agent_metrics_hour ON agent_metrics (hour_bucket DESC);

-- Tool metrics table for hourly aggregated tool call data
CREATE TABLE tool_metrics (
    id SERIAL PRIMARY KEY,
    hour_bucket TIMESTAMP NOT NULL,
    tool_name VARCHAR(50) NOT NULL,
    call_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    latency_sum_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tool_metrics_hour_name UNIQUE (hour_bucket, tool_name)
);

-- Index for time-range queries
CREATE INDEX idx_tool_metrics_hour ON tool_metrics (hour_bucket DESC);
