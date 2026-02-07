# DivineSense 会话管理机制调研报告

> **版本**: v1.0
> **日期**: 2025-01-31
> **作者**: Claude Opus 4.5
> **状态**: 正式版

---

## 一、系统概述

DivineSense 采用**多层会话管理架构**，实现了从前端 UI 到后端持久化的完整会话生命周期管理。核心设计理念是**分离关注点**：

| 层级 | 组件 | 职责 |
|:-----|:-----|:-----|
| **前端状态层** | `AIChatContext` | React Context 管理当前会话状态 |
| **传输层** | Connect RPC (SSE) | 流式双向通信 |
| **事件层** | `EventBus` | 解耦的事件驱动架构 |
| **持久化层** | `ConversationService` | 数据库事务性存储 |
| **缓存层** | LRU Cache | 30 分钟直写缓存 |
| **清理层** | `SessionCleanupJob` | 定期清理过期会话 |

---

## 二、数据库架构

### 2.1 核心表结构

#### `ai_conversation` — 对话主表

```sql
CREATE TABLE ai_conversation (
  id SERIAL PRIMARY KEY,
  uid VARCHAR(36) NOT NULL UNIQUE,        -- 短 UUID
  creator_id INTEGER NOT NULL,            -- 用户 ID
  title VARCHAR(512) NOT NULL,            -- 对话标题
  parrot_id VARCHAR(256) NOT NULL,        -- 代理类型 (memo/schedule/amazing)
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  row_status VARCHAR(16) NOT NULL DEFAULT 'NORMAL'
);
```

**关键索引**：
- `idx_ai_conversation_creator` — 用户会话列表查询
- `idx_ai_conversation_updated` — 按更新时间排序

#### `ai_message` — 消息明细表

```sql
CREATE TABLE ai_message (
  id SERIAL PRIMARY KEY,
  uid VARCHAR(36) NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  type VARCHAR(16) NOT NULL,              -- message / separator
  role VARCHAR(16) NOT NULL,              -- user / assistant / system
  content TEXT NOT NULL,
  metadata JSONB DEFAULT '{}',
  created_ts BIGINT NOT NULL,
  FOREIGN KEY (conversation_id) REFERENCES ai_conversation(id) ON DELETE CASCADE
);
```

#### `conversation_context` — 会话上下文表（AI 特性）

```sql
CREATE TABLE conversation_context (
  id SERIAL PRIMARY KEY,
  session_id VARCHAR(64) NOT NULL UNIQUE, -- 会话 ID
  user_id INTEGER NOT NULL,
  agent_type VARCHAR(20) NOT NULL,
  context_data JSONB NOT NULL DEFAULT '{}', -- {messages: [], metadata: {}}
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL
);
```

**设计亮点**：
- `JSONB` 存储，灵活支持消息和元数据
- `updated_ts` 自动触发器更新
- 30 天自动过期清理策略

---

## 三、会话类型

### 3.1 固定会话 (Fixed Conversation)

**计算公式**：`(userID << 8) | agentTypeOffset`

| 代理 | Offset | 示例 ID (user_id=1) |
|:-----|:-------|:-------------------|
| MEMO | 2 | 258 |
| SCHEDULE | 3 | 259 |
| AMAZING | 4 | 260 |

**特性**：
- 每个用户+代理类型组合有唯一固定会话
- 持久化存储，长期保留
- 适用于常用代理的连续对话

### 3.2 临时会话 (Temporary Conversation)

**特性**：
- 每次新建对话时创建
- 标题为 `"chat.new"`（前端本地化处理编号）
- 用户可选择转为固定会话

---

## 四、后端会话管理流程

### 4.1 核心服务接口

```go
// plugin/ai/session/interface.go
type SessionService interface {
    SaveContext(ctx, sessionID, context) error
    LoadContext(ctx, sessionID) (*ConversationContext, error)
    ListSessions(ctx, userID, limit) ([]SessionSummary, error)
    DeleteSession(ctx, sessionID) error
    CleanupExpired(ctx, retentionDays) (int64, error)
}
```

### 4.2 会话生命周期

```
┌─────────────────────────────────────────────────────────────────┐
│                      会话生命周期                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. 创建阶段         2. 活跃阶段           3. 持久化阶段        │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ RecoverOr   │───▶│ AppendMessage│───▶│ SaveContext  │       │
│  │ Create      │    │              │    │ (Cache+DB)   │       │
│  └─────────────┘    └──────────────┘    └──────────────┘       │
│         │                                      │                 │
│         │                                      │                 │
│         ▼                                      ▼                 │
│  ┌─────────────┐                    ┌──────────────┐            │
│  │ NewSession  │                   │ 滑动窗口截断  │            │
│  │ Max=20条    │                   │ (保留最近20条)│            │
│  └─────────────┘                    └──────────────┘            │
│                                                                  │
│  4. 清理阶段                                                     │
│  ┌─────────────────────────────────────────────────────┐       │
│  │ SessionCleanupJob: 30天后自动删除                    │       │
│  └─────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 滑动窗口机制

```go
const MaxMessagesPerSession = 20

// plugin/ai/session/recovery.go:69-72
if len(session.Messages) > MaxMessagesPerSession {
    session.Messages = session.Messages[len(session.Messages)-MaxMessagesPerSession:]
}
```

**目的**：防止上下文无限增长，控制 LLM token 消耗

---

## 五、事件驱动架构

### 5.1 EventBus 设计

```go
// server/router/api/v1/ai/conversation_service.go
type EventBus struct {
    listeners map[ChatEventType][]ChatEventListener
    timeout   time.Duration  // 默认 5 秒
}
```

### 5.2 事件类型

| 事件类型 | 触发时机 | 监听者 |
|:---------|:---------|:-------|
| `conversation_start` | 对话开始/恢复 | `ConversationService` |
| `user_message` | 用户发送消息 | `ConversationService` |
| `assistant_response` | AI 响应完成 | `ConversationService` |
| `separator` | 工具调用分隔 | `ConversationService` |

### 5.3 并发安全机制

- 每个监听器独立 goroutine 执行
- 超时控制（默认 5 秒）
- Panic 恢复机制
- `sync.Mutex` 保护结果收集

---

## 六、上下文构建机制

### 6.1 Token 预算分配

```go
// plugin/ai/context/budget.go
func (a *BudgetAllocator) Allocate(total int, hasRetrieval bool) *TokenBudget {
    // 固定预算分配
    // SystemPrompt: 500 tokens
    // UserPrefs: 10%

    if hasRetrieval {
        // 短期记忆: 40%
        // 长期记忆: 15%
        // 检索结果: 45%
    } else {
        // 短期记忆: 55%
        // 长期记忆: 30%
    }
}
```

### 6.2 优先级排序

```go
// plugin/ai/context/priority.go
const (
    PrioritySystem      = 100  // 系统提示
    PriorityUserQuery   = 90   // 当前用户查询
    PriorityRecentTurns = 80   // 最近 3 轮
    PriorityRetrieval   = 70   // RAG 检索
    PriorityEpisodic    = 60   // 情景记忆
    PriorityPreferences = 50   // 用户偏好
    PriorityOlderTurns  = 40   // 早期对话
)
```

### 6.3 Token 估算启发式

```go
// 中文 ≈ 2 tokens/字符，ASCII ≈ 0.25 tokens/字符
func EstimateTokens(content string) int {
    chineseCount := countChinese(content)
    asciiCount := countASCII(content)
    return chineseCount*2 + asciiCount/4
}
```

---

## 七、缓存策略

### 7.1 双层缓存

```
┌───────────────────────────────────────────────────────────┐
│                       缓存层级                              │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  L1: 直写缓存 (Write-Through)                              │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ Key: "session:{session_id}"                         │ │
│  │ TTL: 30 分钟                                         │ │
│  │ 实现: plugin/ai/cache/lru_cache.go                  │ │
│  └─────────────────────────────────────────────────────┘ │
│                          │                                 │
│                          ▼                                 │
│  L2: PostgreSQL 数据库                                      │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ Table: conversation_context                         │ │
│  │ Index: session_id (UNIQUE)                          │ │
│  │ Trigger: updated_ts 自动更新                         │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

### 7.2 缓存失效策略

- **写失效**：`SaveContext` 时更新缓存
- **删失效**：`DeleteSession` 时清除缓存
- **TTL 失效**：30 分钟自动过期

---

## 八、流式传输机制

### 8.1 协议

Connect RPC 双向流式传输：

```typescript
// web/src/hooks/useAIQueries.ts:192
const stream = aiServiceClient.chat(request);

for await (const response of stream) {
    // 处理事件类型
    if (response.eventType === "thinking") ...
    if (response.eventType === "tool_use") ...
    if (response.eventType === "answer") ...
    if (response.done === true) ...
}
```

### 8.2 事件流

```
客户端                    服务器                    Agent
  │                        │                        │
  │──── ChatRequest ──────▶│                        │
  │                        │──── Execute ──────────▶│
  │                        │                        │
  │◀──── thinking ─────────│                        │
  │◀──── tool_use (memo_search) ───────────────────│
  │                        │                        │
  │◀──── tool_result ──────│◀──── Search Result ───│
  │                        │                        │
  │◀──── answer (chunked) ─│◀──── LLM Stream ──────│
  │                        │                        │
  │◀──── done ─────────────│                        │
```

### 8.3 超时控制

- 前端：5 分钟流式超时 (`STREAM_TIMEOUT_MS`)
- 后端：Context 传播取消信号
- 事件监听器：5 秒独立超时

---

## 九、前端会话状态

### 9.1 AIChatContext

```typescript
// web/src/contexts/AIChatContext.tsx
interface AIChatContextValue {
    // 会话列表
    conversations: Conversation[]
    currentConversation: Conversation | null

    // 状态
    state: {
        currentCapability: CapabilityType
        capabilityStatus: CapabilityStatus
        currentMode: AIMode  // normal | geek | evolution
        immersiveMode: boolean
    }

    // 操作
    createConversation: () => void
    selectConversation: (id: string) => void
    addMessage: (message: ChatItem) => void
    clearMessages: () => void
}
```

### 9.2 消息类型

```typescript
type ChatItem =
  | TextMessage      // { type: 'text', role: 'user'|'assistant', content }
  | Separator        // { type: 'separator', content: '---' }
  | Thinking         // { type: 'thinking', content }
  | ToolUse          // { type: 'tool_use', name, input }
  | ToolResult       // { type: 'tool_result', content }
  | MemoQueryResult  // { type: 'memo_query_result', memos }
  | ScheduleQueryResult // { type: 'schedule_query_result', schedules }
```

---

## 十、清理与维护

### 10.1 自动清理任务

```go
// plugin/ai/session/cleanup.go
const (
    DefaultRetentionDays   = 30    // 30 天保留期
    DefaultCleanupInterval = 24h   // 24 小时清理间隔
)
```

### 10.2 清理策略

| 策略 | 描述 |
|:-----|:-----|
| 滑动窗口 | 单会话最多保留 20 条消息 |
| TTL 过期 | 缓存 30 分钟自动失效 |
| 定期清理 | 30 天未更新的会话自动删除 |

---

## 十一、特殊模式会话

### 11.1 Geek Mode

```go
// sessionID 基于 conversation_id 生成
namespace := uuid.MustParse("00000000-0000-0000-0000-000000000000")
sessionID := uuid.NewSHA1(namespace, []byte(
    fmt.Sprintf("conversation_%d", req.ConversationID)
)).String()

// 工作目录隔离
workDir := fmt.Sprintf("~/.divinesense/claude/user_%d", userID)
```

### 11.2 Evolution Mode

```go
// 用户特定命名空间隔离
namespace := uuid.MustParse(fmt.Sprintf(
    "00000000-0000-0000-0000-%012x", req.UserID
))
sessionID := uuid.NewSHA1(namespace, []byte(
    fmt.Sprintf("evolution_%d", req.ConversationID)
)).String()

// 工作目录：源代码根目录
sourceDir := os.Getenv("DIVINESENSE_SOURCE_DIR")
```

---

## 十二、关键指标与常量

| 指标 | 值 | 说明 |
|:-----|:---|:-----|
| `MaxMessagesPerSession` | 20 | 单会话最大消息数 |
| `CacheTTL` | 30 分钟 | 会话缓存过期时间 |
| `RetentionDays` | 30 天 | 会话保留期 |
| `StreamTimeout` | 5 分钟 | 前端流式超时 |
| `ListenerTimeout` | 5 秒 | 事件监听器超时 |
| `DefaultMaxTokens` | 4096 | LLM 上下文窗口 |

---

## 十三、架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DivineSense 会话管理架构                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐                                                            │
│  │   前端 UI    │                                                            │
│  │ AIChat.tsx   │                                                            │
│  │ AIChatContext│                                                           │
│  └──────┬───────┘                                                            │
│         │ Connect RPC (SSE)                                                  │
│         ▼                                                                     │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        后端 Handler                                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────────┐ │  │
│  │  │ ParrotHand  │  │ EventBus    │  │ ConversationService              │ │  │
│  │  │ ler         │──│             │──│                                 │ │  │
│  │  └─────────────┘  └─────────────┘  │  ┌─────────────────────────────┐ │ │  │
│  │                                      │  │ 固定会话计算                │ │ │  │
│  │                                      │  │ (userID << 8) | offset    │ │ │  │
│  │                                      │  └─────────────────────────────┘ │ │  │
│  │                                      └───────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│         │                                                                     │
│         ▼                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                         会话服务层                                       │ │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐ │ │
│  │  │ SessionService   │  │ SessionRecovery  │  │ SessionCleanupJob   │ │ │
│  │  │                  │  │                  │  │ (30天自动清理)       │ │ │
│  │  │ SaveContext      │  │ RecoverSession   │  │                      │ │ │
│  │  │ LoadContext      │  │ AppendMessage    │  │ CleanupExpired       │ │ │
│  │  │ ListSessions     │  │ 滑动窗口(20条)   │  │                      │ │ │
│  │  └────────┬─────────┘  └────────┬─────────┘  └──────────────────────┘ │ │
│  │           │                     │                                      │ │
│  │           ▼                     ▼                                      │ │
│  │  ┌────────────────────────────────────────────────────────────────────┐│ │
│  │  │                      缓存层 (LRU, 30min)                           ││ │
│  │  └────────────────────────────────────────────────────────────────────┘│ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│         │                                                                     │
│         ▼                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                          持久化层 (PostgreSQL)                            │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │ │
│  │  │ ai_conversation │  │ ai_message      │  │ conversation_context    │ │ │
│  │  │ (对话主表)      │  │ (消息明细)      │  │ (JSONB上下文)           │ │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 十四、设计亮点总结

1. **关注点分离**：传输、事件、持久化、缓存各层职责明确
2. **滑动窗口**：防止上下文无限增长，控制 token 消耗
3. **事件驱动**：EventBus 解耦各组件，支持扩展
4. **缓存优化**：直写缓存 + 30 分钟 TTL
5. **固定会话**：位运算生成唯一 ID，无需数据库查询
6. **并发安全**：goroutine + 超时控制 + panic 恢复
7. **流式响应**：实时反馈，提升用户体验
8. **自动清理**：30 天过期 + 定时任务，防止数据膨胀

---

## 十五、相关文件索引

| 模块 | 文件路径 |
|:-----|:---------|
| **Session 接口** | `plugin/ai/session/interface.go` |
| **Session 存储** | `plugin/ai/session/store.go` |
| **Session 恢复** | `plugin/ai/session/recovery.go` |
| **Session 清理** | `plugin/ai/session/cleanup.go` |
| **Memory 接口** | `plugin/ai/memory/interface.go` |
| **Memory 服务** | `plugin/ai/memory/service.go` |
| **Context 构建器** | `plugin/ai/context/builder_impl.go` |
| **Token 预算** | `plugin/ai/context/budget.go` |
| **优先级排序** | `plugin/ai/context/priority.go` |
| **AI Handler** | `server/router/api/v1/ai/handler.go` |
| **Event Bus** | `server/router/api/v1/ai/conversation_service.go` |
| **前端 Hook** | `web/src/hooks/useAIQueries.ts` |
| **前端页面** | `web/src/pages/AIChat.tsx` |
| **DB 迁移** | `store/migration/postgres/V0.53.2__add_conversation_context.sql` |
