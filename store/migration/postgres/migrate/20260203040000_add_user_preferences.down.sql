-- Rollback: Drop user_preferences table
DROP TRIGGER IF EXISTS trigger_user_preferences_updated_ts ON user_preferences;
DROP FUNCTION IF EXISTS update_user_preferences_updated_ts();
DROP INDEX IF EXISTS idx_user_preferences_gin;
DROP TABLE IF EXISTS user_preferences;
