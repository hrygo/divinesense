-- Rollback: Remove app_secret column from chat_app_credential
ALTER TABLE chat_app_credential DROP COLUMN IF EXISTS app_secret;
