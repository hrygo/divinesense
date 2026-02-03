-- Migration: 20260203_conversation_channel_type
-- Description: Add channel_type column to conversation_context table

-- Add channel_type column to track where conversations originated
ALTER TABLE conversation_context
ADD COLUMN channel_type TEXT DEFAULT 'web';

-- Add check constraint for valid channel types
ALTER TABLE conversation_context
ADD CONSTRAINT check_channel_type
CHECK (channel_type IN ('web', 'telegram', 'whatsapp', 'dingtalk'));

-- Create index for filtering by channel
CREATE INDEX idx_conversation_context_channel_type ON conversation_context(channel_type);

-- Add comment
COMMENT ON COLUMN conversation_context.channel_type IS 'Origin channel: web, telegram, whatsapp, dingtalk';
