-- Add user_preferences table for AI memory system
-- Stores user preferences learned from interactions

CREATE TABLE user_preferences (
  user_id INTEGER PRIMARY KEY,
  preferences JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  CONSTRAINT fk_user_preferences_user
    FOREIGN KEY (user_id)
    REFERENCES "user"(id)
    ON DELETE CASCADE
);

-- Trigger to auto-update updated_ts
CREATE OR REPLACE FUNCTION update_user_preferences_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_user_preferences_updated_ts
  BEFORE UPDATE ON user_preferences
  FOR EACH ROW
  EXECUTE FUNCTION update_user_preferences_updated_ts();

-- GIN index for efficient JSONB queries
CREATE INDEX idx_user_preferences_gin
ON user_preferences USING gin(preferences);

COMMENT ON TABLE user_preferences IS 'Stores user preferences for AI personalization';
COMMENT ON COLUMN user_preferences.preferences IS 'JSONB containing timezone, default_duration, preferred_times, frequent_locations, communication_style, tag_preferences, and custom_settings';
