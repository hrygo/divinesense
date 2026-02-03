-- Rollback: Drop user_cost_settings table
DROP TRIGGER IF EXISTS trigger_user_cost_settings_updated_at ON user_cost_settings;
DROP FUNCTION IF EXISTS update_user_cost_settings_updated_at();
DROP TABLE IF EXISTS user_cost_settings;
