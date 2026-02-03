-- Migration: V0.53.4__add_chat_app_app_secret
-- Description: Add app_secret column to chat_app_credential for DingTalk AppSecret

-- Add app_secret column for storing additional secrets (e.g., DingTalk AppSecret)
ALTER TABLE chat_app_credential ADD COLUMN app_secret TEXT;

-- Add comment
COMMENT ON COLUMN chat_app_credential.app_secret IS 'Additional secret for platforms that require it (e.g., DingTalk AppSecret)';
