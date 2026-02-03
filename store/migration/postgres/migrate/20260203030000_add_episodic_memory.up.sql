-- Add episodic_memory table for AI memory system
-- Stores interaction history for learning user patterns and preferences

CREATE TABLE episodic_memory (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
  agent_type VARCHAR(20) NOT NULL,
  user_input TEXT NOT NULL,
  outcome VARCHAR(20) NOT NULL DEFAULT 'success',
  summary TEXT,
  importance REAL DEFAULT 0.5,
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_episodic_memory_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_episodic_memory_outcome
    CHECK (outcome IN ('success', 'failure')),
  CONSTRAINT chk_episodic_memory_agent_type
    CHECK (agent_type IN ('memo', 'schedule', 'amazing', 'assistant')),
  CONSTRAINT chk_episodic_memory_importance
    CHECK (importance >= 0 AND importance <= 1)
);

-- Index for querying user's episodes by time
CREATE INDEX idx_episodic_memory_user_time
ON episodic_memory(user_id, timestamp DESC);

-- Index for filtering by agent type
CREATE INDEX idx_episodic_memory_agent
ON episodic_memory(agent_type);

-- Index for importance-based queries
CREATE INDEX idx_episodic_memory_importance
ON episodic_memory(user_id, importance DESC);

COMMENT ON TABLE episodic_memory IS 'Stores episodic memories for AI agents to learn from past interactions';
COMMENT ON COLUMN episodic_memory.agent_type IS 'Type of agent: memo, schedule, amazing, or assistant';
COMMENT ON COLUMN episodic_memory.outcome IS 'Result of the interaction: success or failure';
COMMENT ON COLUMN episodic_memory.importance IS 'Importance score from 0 to 1, used for memory prioritization';
