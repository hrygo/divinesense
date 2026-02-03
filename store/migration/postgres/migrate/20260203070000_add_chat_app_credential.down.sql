-- Rollback: Drop chat_app_credential table
DROP INDEX IF EXISTS idx_chat_app_credential_platform_user_id;
DROP INDEX IF EXISTS idx_chat_app_credential_platform;
DROP INDEX IF EXISTS idx_chat_app_credential_user;
DROP TABLE IF EXISTS chat_app_credential;
