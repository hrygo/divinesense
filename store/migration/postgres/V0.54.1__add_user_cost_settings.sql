-- User cost settings table for budget management and alerts
-- This table stores user-specific cost control preferences
CREATE TABLE user_cost_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,

    -- Alert thresholds
    daily_budget_usd NUMERIC(10,4),
    per_session_threshold_usd NUMERIC(10,4) DEFAULT 5.0,

    -- Notification preferences
    alert_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alert_email BOOLEAN NOT NULL DEFAULT FALSE,
    alert_in_app BOOLEAN NOT NULL DEFAULT TRUE,

    -- Budget reset tracking
    budget_reset_at DATE,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cost_settings_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

-- Note: No separate index needed on user_id - UNIQUE constraint provides automatic indexing

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_user_cost_settings_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_user_cost_settings_updated_at
  BEFORE UPDATE ON user_cost_settings
  FOR EACH ROW
  EXECUTE FUNCTION update_user_cost_settings_updated_at();
