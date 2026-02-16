# 统一块模型 (Unified Block Model) 核心规格

> **状态**: ✅ 已实现 (2026-02-10) | **版本**: v0.97.0
> **Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **优先级**: P0 (核心) | **PR**: [#78](https://github.com/hrygo/divinesense/pull/78)

---

## 1. 核心概念

### 1.1 什么是 Block？

**Block (块)** 是 AI 聊天对话的**一等公民持久化单元**，代表一个完整的对话回合：

```
Block = 用户输入 + AI 响应 + 完整事件流 + 会话统计
```

### 1.2 设计目标

| 目标                 | 描述                                           |
| :------------------- | :--------------------------------------------- |
| **统一数据模型**     | 消除普通模式和 CC 模式的数据结构差异           |
| **模式独立持久化**   | 每个 Block 记录创建时的 mode，不受全局状态影响 |
| **追加式输入支持**   | 支持在 AI 回复前追加用户输入                   |
| **完整事件流持久化** | 保存 thinking/tool_use/answer 完整事件流       |
| **CC 会话映射**      | 与 Claude Code CLI 会话的确定性映射            |

### 1.3 Block 类型

| BlockType           | 说明             |
| :------------------ | :--------------- |
| `MESSAGE`           | 用户-AI 对话回合 |
| `CONTEXT_SEPARATOR` | 上下文分隔标记   |

### 1.4 Block 模式

| BlockMode   | 说明                       | 主题色 |
| :---------- | :------------------------- | :----- |
| `NORMAL`    | 普通 AI 模式               | 琥珀色 |
| `GEEK`      | 极客模式 (Claude Code CLI) | 石板蓝 |
| `EVOLUTION` | 进化模式 (自我进化)        | 翠绿   |

---

## 2. 数据结构定义

### 2.1 数据库表结构

```sql
CREATE TABLE ai_block (
  -- 主键与外键
  id BIGSERIAL PRIMARY KEY,
  uid VARCHAR(64) UNIQUE NOT NULL,
  conversation_id INTEGER NOT NULL REFERENCES ai_conversation(id) ON DELETE CASCADE,

  -- 回合信息
  round_number INTEGER NOT NULL DEFAULT 0,
  block_type TEXT NOT NULL DEFAULT 'MESSAGE',
  mode TEXT NOT NULL DEFAULT 'normal',

  -- 用户输入 (支持追加)
  user_inputs JSONB NOT NULL DEFAULT '[]',

  -- AI 响应
  assistant_content TEXT,
  assistant_timestamp BIGINT,

  -- 事件流 (完整时序)
  event_stream JSONB NOT NULL DEFAULT '[]',

  -- CC 模式统计
  session_stats JSONB,
  cc_session_id TEXT,

  -- 状态
  status TEXT NOT NULL DEFAULT 'pending',
  error_message TEXT,

  -- 树状分支支持
  parent_block_id BIGINT,
  branch_path TEXT,

  -- Token 使用统计
  token_usage JSONB,

  -- 成本估算 (毫美分: 1/1000 美分)
  cost_estimate BIGINT DEFAULT 0,

  -- 模型信息
  model_version TEXT,

  -- 用户反馈
  user_feedback TEXT,
  regeneration_count INTEGER DEFAULT 0,

  -- 软删除
  archived_at BIGINT,

  -- 元数据
  metadata JSONB NOT NULL DEFAULT '{}',

  -- 时间戳 (毫秒)
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,

  -- 约束
  CONSTRAINT chk_ai_block_type CHECK (block_type IN ('MESSAGE', 'CONTEXT_SEPARATOR')),
  CONSTRAINT chk_ai_block_mode CHECK (mode IN ('normal', 'geek', 'evolution')),
  CONSTRAINT chk_ai_block_status CHECK (status IN ('pending', 'streaming', 'completed', 'error')),
  CONSTRAINT uq_ai_block_conversation_round UNIQUE (conversation_id, round_number)
);

-- 索引
CREATE INDEX idx_ai_block_conversation ON ai_block(conversation_id, round_number);
CREATE INDEX idx_ai_block_status ON ai_block(status);
CREATE INDEX idx_ai_block_mode ON ai_block(mode);
CREATE INDEX idx_ai_block_cc_session ON ai_block(cc_session_id) WHERE cc_session_id IS NOT NULL;
CREATE INDEX idx_ai_block_parent ON ai_block(parent_block_id) WHERE parent_block_id IS NOT NULL;
```

### 2.2 Proto 定义

```protobuf
// Block represents a single conversation block (round)
message Block {
  int64 id = 1;
  string uid = 2;
  int32 conversation_id = 3;
  int32 round_number = 4;

  // Block type and mode
  BlockType block_type = 5;
  BlockMode mode = 6;

  // User inputs (may be multiple if user adds more during AI response)
  repeated UserInput user_inputs = 7;

  // AI response
  string assistant_content = 8;
  int64 assistant_timestamp = 9;

  // Event stream (chronological order)
  repeated BlockEvent event_stream = 10;

  // Session statistics (for Geek/Evolution modes)
  SessionStats session_stats = 11;

  // CC session mapping (for Geek/Evolution modes)
  string cc_session_id = 12;

  // Block status
  BlockStatus status = 13;

  // Tree branching support
  int64 parent_block_id = 14;
  string branch_path = 15;

  // Token usage statistics
  TokenUsage token_usage = 19;

  // Cost estimation (in milli-cents: 1/1000 of a US cent)
  int64 cost_estimate = 20;

  // Model information
  string model_version = 21;

  // User feedback and regeneration
  string user_feedback = 22;
  int32 regeneration_count = 23;

  // Error tracking
  string error_message = 24;

  // Archival support
  int64 archived_at = 25;

  // Extension metadata
  string metadata = 16;

  int64 created_ts = 17;
  int64 updated_ts = 18;
}

// BlockType represents the type of block
enum BlockType {
  BLOCK_TYPE_UNSPECIFIED = 0;
  BLOCK_TYPE_MESSAGE = 1;           // User-AI conversation round
  BLOCK_TYPE_CONTEXT_SEPARATOR = 2; // Context separator marker
}

// BlockMode represents the AI mode used for this block
enum BlockMode {
  BLOCK_MODE_UNSPECIFIED = 0;
  BLOCK_MODE_NORMAL = 1;   // Normal AI assistant mode
  BLOCK_MODE_GEEK = 2;     // Geek mode (Claude Code CLI)
  BLOCK_MODE_EVOLUTION = 3; // Evolution mode (self-improvement)
}

// BlockStatus represents the current status of a block
enum BlockStatus {
  BLOCK_STATUS_UNSPECIFIED = 0;
  BLOCK_STATUS_PENDING = 1;    // Waiting for AI response
  BLOCK_STATUS_STREAMING = 2;  // AI is currently responding
  BLOCK_STATUS_COMPLETED = 3;  // Response completed
  BLOCK_STATUS_ERROR = 4;      // Error occurred
}

// UserInput represents a single user input within a block
message UserInput {
  string content = 1;
  int64 timestamp = 2;
  string metadata = 3; // JSON string
}

// BlockEvent represents an event in the event stream
message BlockEvent {
  string type = 1; // "thinking", "tool_use", "tool_result", "answer", "error"
  string content = 2;
  int64 timestamp = 3;
  string meta = 4; // JSON string with event-specific metadata
}

// TokenUsage represents detailed token usage for a single block or LLM call
message TokenUsage {
  int32 prompt_tokens = 1;        // Input tokens
  int32 completion_tokens = 2;    // Output tokens
  int32 total_tokens = 3;         // Total tokens
  int32 cache_read_tokens = 4;    // Tokens read from cache
  int32 cache_write_tokens = 5;   // Tokens written to cache
}
```

### 2.3 前端类型定义

```typescript
// web/src/types/block.ts

export type BlockType = 'message' | 'context_separator';
export type BlockMode = 'normal' | 'geek' | 'evolution';
export type BlockStatus = 'pending' | 'streaming' | 'completed' | 'error';

export interface UserInput {
  content: string;
  timestamp: number;
  metadata?: Record<string, unknown>;
}

export interface BlockEvent {
  type: 'thinking' | 'tool_use' | 'tool_result' | 'answer' | 'error';
  content: string;
  timestamp: number;
  meta?: {
    tool_name?: string;
    tool_id?: string;
    is_error?: boolean;
    file_path?: string;
    duration_ms?: number;
    input_summary?: string;
    output_summary?: string;
  };
}

export interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

export interface AIBlock {
  id: string;
  uid: string;
  conversationId: number;
  roundNumber: number;
  blockType: BlockType;
  mode: BlockMode;

  // User inputs (array for appending support)
  userInputs: UserInput[];

  // Assistant response
  assistantContent?: string;
  assistantTimestamp?: number;

  // Event stream (complete timeline)
  eventStream: BlockEvent[];

  // CC mode statistics
  sessionStats?: SessionSummary;
  ccSessionId?: string;

  // Status
  status: BlockStatus;
  errorMessage?: string;

  // Token usage
  tokenUsage?: TokenUsage;

  // Cost estimation
  costEstimate?: number;

  // Model info
  modelVersion?: string;

  // User feedback
  userFeedback?: string;
  regenerationCount?: number;

  // Metadata
  metadata: Record<string, unknown>;
  createdTs: number;
  updatedTs: number;
}
```

---

## 3. API 接口说明

### 3.1 RPC 方法

| RPC 方法          | 说明                  |
| :---------------- | :-------------------- |
| `ListBlocks`      | 列出会话的所有 Blocks |
| `GetBlock`        | 获取单个 Block 详情   |
| `CreateBlock`     | 创建新 Block          |
| `UpdateBlock`     | 更新 Block            |
| `DeleteBlock`     | 删除 Block            |
| `AppendUserInput` | 追加用户输入          |
| `AppendEvent`     | 追加事件              |
| `ForkBlock`       | 分支 Block (Phase 3)  |

### 3.2 HTTP 端点

| 操作        | 方法   | 路径                                   |
| :---------- | :----- | :------------------------------------- |
| 列出 Blocks | GET    | `/api/v1/ai/conversations/{id}/blocks` |
| 获取 Block  | GET    | `/api/v1/ai/blocks/{id}`               |
| 创建 Block  | POST   | `/api/v1/ai/conversations/{id}/blocks` |
| 更新 Block  | PATCH  | `/api/v1/ai/blocks/{id}`               |
| 删除 Block  | DELETE | `/api/v1/ai/blocks/{id}`               |
| 追加输入    | POST   | `/api/v1/ai/blocks/{id}/inputs`        |
| 追加事件    | POST   | `/api/v1/ai/blocks/{id}/events`        |
| 分支 Block  | POST   | `/api/v1/ai/blocks/{id}/fork`          |

### 3.3 请求/响应示例

#### 创建 Block

**请求**:
```json
POST /api/v1/ai/conversations/123/blocks
{
  "block_type": "MESSAGE",
  "mode": "NORMAL",
  "user_inputs": [
    {
      "content": "帮我搜索笔记",
      "timestamp": 1707520800000
    }
  ]
}
```

**响应**:
```json
{
  "id": "456",
  "uid": "blk_xxx",
  "conversation_id": 123,
  "round_number": 0,
  "block_type": "MESSAGE",
  "mode": "NORMAL",
  "status": "PENDING",
  "user_inputs": [...],
  "created_ts": 1707520800000
}
```

#### 追加事件

**请求**:
```json
POST /api/v1/ai/blocks/456/events
{
  "event": {
    "type": "thinking",
    "content": "正在搜索笔记...",
    "timestamp": 1707520801000
  }
}
```

---

## 4. 实现参考

### 4.1 后端实现

| 文件                                    | 职责                |
| :-------------------------------------- | :------------------ |
| `store/block.go`                        | BlockStore 接口定义 |
| `store/db/postgres/block.go`            | PostgreSQL 实现     |
| `server/service/block/block_service.go` | 业务逻辑层          |
| `server/router/api/v1/ai/handler.go`    | Chat Handler 集成   |

### 4.2 前端实现

| 文件                                                | 职责             |
| :-------------------------------------------------- | :--------------- |
| `web/src/types/block.ts`                            | 类型定义         |
| `web/src/hooks/useBlockQueries.ts`                  | React Query 集成 |
| `web/src/components/AIChat/UnifiedMessageBlock.tsx` | Block 渲染组件   |
| `web/src/contexts/AIChatContext.tsx`                | 状态管理         |

### 4.3 迁移文件

| 文件                                                      | 说明        |
| :-------------------------------------------------------- | :---------- |
| `store/migration/postgres/V0.60.x_create_ai_block.up.sql` | 表创建      |
| `store/migration/postgres/schema/LATEST.sql`              | 完整 Schema |

---

## 5. Block 状态流转

```
pending ──▶ streaming ──▶ completed
    │             │
    │             └──▶ error
    └──▶ error
```

| 状态        | 说明         | 触发条件          |
| :---------- | :----------- | :---------------- |
| `pending`   | 等待 AI 响应 | Block 创建时      |
| `streaming` | AI 正在响应  | 首个事件到达      |
| `completed` | 响应完成     | AI 结束或用户停止 |
| `error`     | 发生错误     | 异常或超时        |

---

## 6. 扩展功能

### 6.1 树状分支 (Phase 3)

支持对话分支和编辑重生成：

```sql
-- 扩展字段
parent_block_id BIGINT,  -- 父 Block ID
branch_path TEXT,        -- 分支路径 (如 "0/1/2")
```

### 6.2 Token 统计 (P1-A006)

普通模式的 Token 使用统计：

```sql
-- 扩展字段
token_usage JSONB,      -- Token 使用明细
cost_estimate BIGINT,   -- 成本估算 (毫美分)
model_version TEXT,     -- LLM 版本
```

---

## 7. 相关文档

| 文档                                                           | 描述               |
| :------------------------------------------------------------- | :----------------- |
| [Block Design Index](./block-design/INDEX.md)                  | Block 设计规格索引 |
| [LLM 统计收集](./block-design/P1-A006-llm-stats-collection.md) | Token 统计规格     |
| [树状分支](./block-design/tree-conversation-branching.md)      | 分支功能设计       |
| [架构文档](../architecture/overview.md)                        | 完整系统架构       |

---

*维护者*: DivineSense 开发团队
*反馈渠道*: [GitHub Issues](https://github.com/hrygo/divinesense/issues)
