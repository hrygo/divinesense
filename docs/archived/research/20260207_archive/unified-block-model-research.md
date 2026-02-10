# 统一 Block 模型 - 调研报告

> **调研时间**: 2026-02-04
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **版本**: v1.0

---

## 执行摘要

本报告提出了 DivineSense AI 聊天系统的**统一 Block 模型**设计，旨在解决普通模式与 CC 连接模式（极客/进化）之间的数据结构割裂问题，同时完整支持 Warp Block UI 的持久化需求。

**核心创新**：将 `Block` 作为"对话回合"的一等公民持久化单元，替代现有的 `Message` 中心模型。

---

## 问题背景

### 当前架构的问题

```
┌─────────────────────────────────────────────────────────────┐
│  现状：两套平行的数据结构                                       │
├─────────────────────────────────────────────────────────────┤
│  普通模式                    CC 模式                       │
│  ┌─────────────────┐        ┌─────────────────────────┐   │
│  │ ai_message 表    │        │ agent_session_stats    │   │
│  │ - role           │        │ - session_id           │   │
│  │ - content        │        │ - stats (摘要)         │   │
│  └─────────────────┘        └─────────────────────────┘   │
│         │                              │                     │
│         ▼                              ▼                     │
│  前端：配对成 Block            前端：流式事件            │
│  但没有完整持久化              但没有完整持久化         │
└─────────────────────────────────────────────────────────────┘
```

### 需求来源

1. **Issue #69**（已完成）：Warp Block UI 已实现，但内容未完整持久化
2. **Issue #57**（未实施）：会话嵌套模型，支持追加式输入
3. **UI 需求**：三种模式的 Block 渲染风格应独立，不受全局 mode 影响

---

## 设计方案

### 核心概念：Block 作为对话回合

```
会话 (Conversation) = 多个 Block 的有序序列

┌─────────────────────────────────────────────────────────────────┐
│  Conversation #123                                              │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Block #0 (mode='geek')                                     ││
│  │  user_inputs: ["分析代码性能", "检查内存泄漏"]                ││
│  │  event_stream: [thinking, tool_use, tool_result, ...]      ││
│  │  status: completed                                         ││
│  ├─────────────────────────────────────────────────────────────┤│
│  │  Block #1 (mode='normal')                                    ││
│  │  user_inputs: ["总结一下"]                                  ││
│  │  assistant_content: "今天我们分析了..."                     ││
│  │  status: completed                                         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 数据模型

```
┌──────────────────────────────────────────────────────────────────────┐
│  ai_block 表                                                          │
├──────────────────────────────────────────────────────────────────────┤
│  id                  BIGSERIAL PRIMARY KEY                             │
│  conversation_id     INTEGER → ai_conversation(id)                     │
│  round_number       INTEGER -- 会话内的第几个 Block（0-based）         │
│  block_type         TEXT -- 'message' | 'context_separator'            │
│  mode                TEXT -- 'normal' | 'geek' | 'evolution'             │
│  user_inputs         JSONB -- [{"content": "...", "timestamp": ...}]      │
│  assistant_content   TEXT                                                  │
│  assistant_timestamp BIGINT                                               │
│  event_stream        JSONB -- [{type, content, timestamp, meta}]        │
│  session_stats       JSONB -- SessionSummary（CC 模式）                    │
│  cc_session_id       TEXT -- UUID v5 映射到 CC CLI 会话                     │
│  status              TEXT -- 'pending' | 'streaming' | 'completed' | 'error'│
│  metadata            JSONB                                                 │
│  created_ts          BIGINT                                               │
│  updated_ts          BIGINT                                               │
└──────────────────────────────────────────────────────────────────────┘
```

### 关键设计决策

#### 1. 用户输入判断

```
用户输入 Q → 判断最新 Block 状态
                │
                ├─ status != 'completed' → 追加到当前 Block
                │
                └─ status == 'completed'  → 创建新 Block
```

**代码逻辑**：
```typescript
const latestBlock = getLatestBlock(conversationId);
if (latestBlock && latestBlock.status !== 'completed') {
  appendToBlock(latestBlock.id, userInput);
} else {
  createNewBlock(conversationId, userInput);
}
```

#### 2. Block Mode 独立性

- Block 的 `mode` 在创建时确定
- 存储在数据库，不受页面全局 `currentMode` 影响
- 前端渲染时从 Block 读取 `mode`

```
页面全局 mode: normal
      ↓
┌─────────────────────────────────────────────┐
│  Block #0 (mode='geek')   → 紫色主题渲染        │
│  Block #1 (mode='normal') → 琥珀色主题渲染      │
│  Block #2 (mode='evolution') → 翠绿主题渲染      │
└─────────────────────────────────────────────┘
```

#### 3. CC 会话映射

```
┌─────────────────────────────────────────────────────────────────┐
│  DivineSense 外层                                                │
│  Conversation #123                                                  │
│  ├─ Block #0 (mode='geek', cc_session_id='uuid-v5-123')         │
│  ├─ Block #1 (mode='geek', cc_session_id='uuid-v5-123')         │
│  └─ Block #2 (mode='normal', cc_session_id=null)               │
└─────────────────────────────────────────────────────────────────┘
                                │ UUID v5 映射
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  Claude Code CLI 内层                                            │
│  ~/.claude/sessions/uuid-v5-123/                               │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  CC Internal Session File (完整上下文)                      ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## 主题色设计

### 三种模式的新配色

| 模式 | 颜色 | 寓意 | Tailwind 基色 |
|:-----|:-----|:-----|:-----------|
| **Normal** | 琥珀 | 闪念如琥珀般珍贵保存 | `amber` |
| **Geek** | 石板蓝 | 代码如石板般精确 | `sky` + `slate` |
| **Evolution** | 翠绿 | 系统如植物般向上生长 | `emerald` |

### 颜色对比表（Dark Mode）

```
┌──────────────┬────────────────┬────────────────┬────────────────┐
│   组件        │   Normal (琥珀)  │   Geek (石板蓝) │ Evolution(翠绿)│
├──────────────┼────────────────┼────────────────┼────────────────┤
│ User Bubble  │  amber-500      │  sky-500        │  emerald-500    │
│ Border       │  amber-200/800  │  sky-200/800    │  emerald-200/800│
│ Header BG    │  amber-900/20   │  slate-900      │  emerald-900/20 │
│ Text Primary │  amber-100      │  sky-50         │  emerald-100    │
└──────────────┴────────────────┴────────────────┴────────────────┘
```

---

## 实现阶段规划

| 阶段 | 内容 | 优先级 |
|:-----|:-----|:-------|
| **Phase 1** | 数据库迁移 + Proto 生成 | P0 |
| **Phase 2** | 后端 BlockStore 实现 | P0 |
| **Phase 3** | Chat Handler 改造 | P0 |
| **Phase 4** | 前端类型定义 + 主题色更新 | P1 |
| **Phase 5** | UnifiedMessageBlock 改造 | P1 |
| **Phase 6** | ChatMessages + AIChat Context | P1 |
| **Phase 7** | 集成测试 | P1 |

---

## 向后兼容性

### 迁移策略

1. **保留 `ai_message` 表**（迁移期兼容）
2. **创建视图**支持旧代码查询
3. **渐进式迁移**：新会话使用 Block，旧会话保持 Message

### 兼容视图

```sql
CREATE VIEW v_ai_message AS
SELECT
  id,
  uid,
  conversation_id,
  'MESSAGE' as type,
  CASE WHEN round_number % 2 = 0 THEN 'USER' ELSE 'ASSISTANT' END as role,
  CASE WHEN round_number % 2 = 0 THEN user_content ELSE assistant_content END as content,
  metadata,
  created_ts
FROM (
  SELECT
    id,
    uid,
    conversation_id,
    user_inputs,
    assistant_content,
    metadata,
    created_ts,
    round_number * 2 as message_round
  FROM ai_block
  WHERE block_type = 'message'
) expanded;
```

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| 数据结构重构 | 高 | 保留旧表，创建兼容视图 |
| 前端组件改造 | 中 | 渐进式适配，保持 API 兼容 |
| CC 会话映射复杂性 | 中 | 明确文档，UUID v5 确定性映射 |
| 性能影响（JSON 解析） | 低 | JSONB 索引，缓存热点数据 |

---

## 参考资料

- [Issue #69: Warp Block UI](https://github.com/hrygo/divinesense/issues/69)
- [Issue #57: 会话嵌套模型](https://github.com/hrygo/divinesense/issues/57)
- [CC Runner 异步架构](../specs/20260207_archive/cc_runner_async_arch.md)（已归档）
- [前端开发指南](../../dev-guides/FRONTEND.md)

---

*调研完成时间: 2026-02-04*
