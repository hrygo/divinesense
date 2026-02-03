-- Remove foreign key constraint on conversation_id for agent_session_stats
-- Geek/Evolution modes operate independently and don't require ai_conversation entries
--
-- Geek/Evolution 模式独立运行，不依赖 ai_conversation 表
--
-- This allows session stats to be saved without a corresponding conversation record,
-- which is the expected behavior for Geek/Evolution modes that manage their own
-- session lifecycle independent of the conversation system.
-- 允许在没有对应 conversation 记录的情况下保存会话统计，
-- 这是 Geek/Evolution 模式的预期行为，它们独立管理自己的会话生命周期。

BEGIN;

-- Drop the foreign key constraint
-- 删除外键约束
ALTER TABLE agent_session_stats
DROP CONSTRAINT IF EXISTS fk_session_stats_conv;

-- Keep conversation_id as a field for reference and analytics, but without FK enforcement
-- 保留 conversation_id 字段用于引用和分析，但不强制外键约束

COMMENT ON COLUMN agent_session_stats.conversation_id IS
'Conversation ID reference (optional: Geek/Evolution modes may not have corresponding ai_conversation entry)';

COMMIT;
