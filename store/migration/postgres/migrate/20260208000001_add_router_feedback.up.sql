-- Add router feedback and dynamic weight tracking tables
-- This enables Issue #95: Rule routing weight dynamic adjustment

-- router_feedback stores feedback events for routing decisions
CREATE TABLE router_feedback (
  id BIGSERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  input TEXT NOT NULL,
  predicted_intent TEXT NOT NULL,
  actual_intent TEXT NOT NULL,
  feedback_type TEXT NOT NULL CHECK (feedback_type IN ('positive', 'rephrase', 'switch')),
  timestamp BIGINT NOT NULL,
  source TEXT NOT NULL, -- "rule", "history", "llm"
  CONSTRAINT fk_router_feedback_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

-- Index for efficient queries
CREATE INDEX idx_router_feedback_user_timestamp ON router_feedback(user_id, timestamp DESC);
CREATE INDEX idx_router_feedback_user_type ON router_feedback(user_id, feedback_type);
CREATE INDEX idx_router_feedback_intent ON router_feedback(predicted_intent);

COMMENT ON TABLE router_feedback IS 'Stores feedback events for router weight adjustment';
COMMENT ON COLUMN router_feedback.feedback_type IS 'positive: no correction, rephrase: user rephrased, switch: user switched agent';

-- router_weight stores per-user keyword weights for routing
CREATE TABLE router_weight (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  category TEXT NOT NULL CHECK (category IN ('schedule', 'memo', 'amazing')),
  keyword TEXT NOT NULL,
  weight INTEGER NOT NULL CHECK (weight >= 1 AND weight <= 5),
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_router_weight_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE,
  CONSTRAINT uq_router_weight_user_category_keyword UNIQUE (user_id, category, keyword)
);

-- Index for efficient queries
CREATE INDEX idx_router_weight_user ON router_weight(user_id);
CREATE INDEX idx_router_weight_user_category ON router_weight(user_id, category);

-- Trigger to auto-update updated_ts
CREATE OR REPLACE FUNCTION update_router_weight_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_router_weight_updated_ts
  BEFORE UPDATE ON router_weight
  FOR EACH ROW
  EXECUTE FUNCTION update_router_weight_updated_ts();

COMMENT ON TABLE router_weight IS 'Stores per-user keyword weights for dynamic routing';
COMMENT ON COLUMN router_weight.weight IS 'Keyword weight: 1=min, 5=max (default 2)';
