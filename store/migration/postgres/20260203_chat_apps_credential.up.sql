-- Migration: 20260203_chat_apps_credential
-- Description: Add chat_app_credential table for multi-platform chat app integrations

-- Create chat_app_credential table
CREATE TABLE chat_app_credential (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  platform_user_id TEXT NOT NULL,
  platform_chat_id TEXT,
  access_token TEXT,
  webhook_url TEXT,
  enabled BOOLEAN DEFAULT true,
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  UNIQUE(user_id, platform)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_chat_app_credential_user ON chat_app_credential(user_id);
CREATE INDEX idx_chat_app_credential_platform ON chat_app_credential(platform);
CREATE INDEX idx_chat_app_credential_platform_user_id ON chat_app_credential(platform, platform_user_id);

-- Add comment
COMMENT ON TABLE chat_app_credential IS 'Stores credentials for chat app integrations (Telegram, WhatsApp, DingTalk)';
COMMENT ON COLUMN chat_app_credential.platform IS 'Platform name: telegram, whatsapp, dingtalk';
COMMENT ON COLUMN chat_app_credential.platform_user_id IS 'Platform-specific user identifier';
COMMENT ON COLUMN chat_app_credential.platform_chat_id IS 'Platform-specific chat ID for direct messaging';
COMMENT ON COLUMN chat_app_credential.access_token IS 'Encrypted OAuth/Bot token';
COMMENT ON COLUMN chat_app_credential.webhook_url IS 'Webhook URL (DingTalk only)';
