# Spec: Unified Block Model

> **Status**: [PRD](https://github.com/hrygo/divinesense/issues/71) | **Version**: 1.0
> **Priority**: P0 (Core) | **Sprint**: Backend Refactoring

---

## 1. 目标与背景 (Goals & Background)

### 1.1 核心问题 (Core Problem)

当前 DivineSense AI 聊天系统存在两套平行的数据结构，导致数据割裂和持久化不完整：

```
现状架构问题:
┌─────────────────────────────────────────────────────────────┐
│  普通模式 (Normal Mode)     VS     CC 模式 (Geek/Evolution)  │
├─────────────────────────────────────────────────────────────┤
│  ai_message 表                      agent_session_stats 表   │
│  - role, content                    - session_id             │
│  - type (MESSAGE/SEPARATOR)         - stats (摘要)          │
│  - metadata (JSON)                   - started_at, ended_at  │
│  - 简单持久化                        - 统计数据持久化         │
│  ❌ 无会话统计数据                  ❌ 无完整事件流          │
│  ❌ 无模式标识                       ❌ 无用户输入历史       │
│  ❌ 多轮对话状态分散                 ❌ 无法追加输入          │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 设计目标 (Design Goals)

| 目标 | 描述 | 优先级 |
|:-----|:-----|:-------|
| **统一数据模型** | Block 作为"对话回合"的一等公民持久化单元 | P0 |
| **模式独立持久化** | 每个 Block 记录创建时的 mode，不受全局状态影响 | P0 |
| **追加式输入支持** | 支持 Issue #57 的会话嵌套模型 | P1 |
| **完整事件流持久化** | 保存 thinking/tool_use/answer 完整事件流 | P1 |
| **CC 会话映射** | 与 Claude Code CLI 会话的确定性映射 | P1 |
| **向后兼容** | 渐进式迁移，旧数据可访问 | P0 |

### 1.3 用户价值 (User Value)

- **持久化完整对话历史**: 用户可回顾完整的 AI 思考过程和工具调用
- **模式切换无丢失**: 在 Normal/Geek/Evolution 模式间切换，历史保持完整
- **追加式交互**: 用户可在 AI 回复后追加追问，而非被迫新建 Block
- **成本透明**: 完整保存会话统计数据（成本、token、耗时）

### 1.4 技术价值 (Technical Value)

- **数据模型统一**: 消除普通模式和 CC 模式的数据结构差异
- **简化前端逻辑**: UnifiedMessageBlock 可直接从 Block 表获取数据
- **支持会话恢复**: 基于 Block 状态的恢复策略（pending/streaming/completed）
- **扩展性**: 为未来多轮对话、分支对话奠定基础

---

## 2. 依赖关系 (Dependencies)

### 2.1 前置依赖 (Must Complete)

- [x] **Issue #69**: Warp Block UI 实现（已完成前端组件）
- [x] **CC Runner 异步架构**: 会话统计和流式事件处理 (v1.3)

### 2.2 并行依赖 (Can Parallel)

- [ ] **Issue #57**: 会话嵌套模型（可并行设计）

### 2.3 后续依赖 (Depends on This)

- [ ] 会话分享导出功能
- [ ] 对话分支管理
- [ ] 会话分析与洞察

---

## 3. 功能设计 (Functional Design)

### 3.1 架构概览 (Architecture Overview)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Unified Block Model                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  ai_conversation (会话容器)                                      │   │
│   │  - id, uid, title, parrot_id, created_ts, updated_ts            │   │
│   │                                                                  │   │
│   │  ┌─────────────────────────────────────────────────────────────┐│   │
│   │  │  ai_block (对话回合) ← 新表                                ││   │
│   │  │  ├─ id, conversation_id, round_number (0-based)            ││   │
│   │  │  ├─ block_type: 'message' | 'context_separator'            ││   │
│   │  │  ├─ mode: 'normal' | 'geek' | 'evolution'                  ││   │
│   │  │  ├─ user_inputs: JSONB [{content, timestamp}]              ││   │
│   │  │  ├─ assistant_content: TEXT                                 ││   │
│   │  │  ├─ assistant_timestamp: BIGINT                             ││   │
│   │  │  ├─ event_stream: JSONB [{type, content, timestamp, meta}] ││   │
│   │  │  ├─ session_stats: JSONB (CC 模式统计)                      ││   │
│   │  │  ├─ cc_session_id: TEXT (UUID v5 映射)                      ││   │
│   │  │  ├─ status: 'pending' | 'streaming' | 'completed' | 'error'││   │
│   │  │  └─ metadata: JSONB                                        ││   │
│   │  └─────────────────────────────────────────────────────────────┘│   │
│   │                                                                  │   │
│   │  保留兼容: ┌───────────────────────────────────────────────────┐│   │
│   │          │  ai_message (旧表, 只读)                           ││   │
│   │          │  └─ v_ai_message VIEW (兼容视图)                    ││   │
│   │          └───────────────────────────────────────────────────┘│   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  agent_session_stats (CC 模式统计 - 保留)                        │   │
│   │  - session_id, conversation_id, total_cost_usd, ...             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 核心概念 (Core Concepts)

#### 3.2.1 Block as Conversation Turn

```
Block (对话回合) = 用户输入 + AI 响应 + 元数据

┌─────────────────────────────────────────────────────────────────────┐
│  Block #3 (mode='geek', round_number=3)                             │
│  ├─ user_inputs: [                                                 │
│  │    {content: "分析代码性能", timestamp: 1707040800000},         │
│  │    {content: "检查内存泄漏", timestamp: 1707040900000}          │
│  │  ]                                                              │
│  ├─ event_stream: [                                                │
│  │    {type: "thinking", content: "正在分析...", ...},            │
│  │    {type: "tool_use", name: "bash", input: "pprof", ...},      │
│  │    {type: "tool_result", output: "Found 3 leaks", ...},        │
│  │    {type: "answer", content: "分析完成...", ...}                │
│  │  ]                                                              │
│  ├─ session_stats: {total_cost_usd: 0.0032, ...}                  │
│  ├─ cc_session_id: "uuid-v5-123"                                   │
│  └─ status: "completed"                                            │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.2.2 用户输入判断逻辑 (User Input Routing)

```
用户输入 Q → 判断最新 Block 状态
                │
                ├─ status != 'completed' → 追加到当前 Block (user_inputs)
                │
                └─ status == 'completed'  → 创建新 Block

代码逻辑:
---------
const latestBlock = await blockStore.getLatestBlock(conversationId);
if (latestBlock && latestBlock.status !== 'completed') {
  // 追加模式: 用户在 AI 回复前追加输入
  await blockStore.appendUserInput(latestBlock.id, userInput);
} else {
  // 新回合: AI 已完成回复，创建新 Block
  await blockStore.createBlock(conversationId, userInput, currentMode);
}
```

#### 3.2.3 Block Mode 独立性 (Mode Independence)

```
页面全局 mode: normal
      ↓
┌─────────────────────────────────────────────┐
│  Conversation #123                           │
│  ├─ Block #0 (mode='geek')     → 紫色主题渲染  │
│  ├─ Block #1 (mode='normal')   → 琥珀色主题  │
│  └─ Block #2 (mode='evolution') → 翠绿主题  │
└─────────────────────────────────────────────┘

规则:
1. Block 的 mode 在创建时确定
2. mode 存储在数据库，不受页面全局 currentMode 影响
3. 前端渲染时从 Block 读取 mode，选择对应主题色
```

#### 3.2.4 CC 会话映射 (CC Session Mapping)

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

映射算法:
cc_session_id = UUID v5(
  Namespace: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",  // DNS Namespace
  Name: "divinesense:conversation:{conversation_id}:{block_id}"
)
```

### 3.3 关键决策 (Key Decisions)

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:-------|:-------|:-------|:-----|:-----|
| **数据结构** | 新建 ai_block 表 | 扩展 ai_message 表 | A | 避免破坏现有 Message 语义，保持向后兼容 |
| **用户输入存储** | 单一 content 字段 | user_inputs JSONB 数组 | B | 支持追加式输入 (Issue #57) |
| **模式存储** | 全局会话 mode | 每块独立 mode | B | 支持同一会话内模式混合 |
| **事件流存储** | 分表存储 | event_stream JSONB | B | 简化查询，完整保存时序 |
| **CC 会话映射** | 动态生成 | UUID v5 确定性映射 | B | 支持会话恢复，无需额外存储 |

---

## 4. 技术实现 (Technical Implementation)

### 4.1 数据库模型 (Database Schema)

#### 4.1.1 ai_block 表 (新增)

```sql
-- =============================================================================
-- Unified Block Model (V0.60.0)
-- =============================================================================

CREATE TABLE ai_block (
  -- 主键与外键
  id BIGSERIAL PRIMARY KEY,
  conversation_id INTEGER NOT NULL,

  -- 回合信息
  round_number INTEGER NOT NULL DEFAULT 0,  -- 会话内的第几个 Block (0-based)
  block_type TEXT NOT NULL DEFAULT 'MESSAGE',  -- 'message' | 'context_separator'
  mode TEXT NOT NULL DEFAULT 'normal',  -- 'normal' | 'geek' | 'evolution'

  -- 用户输入 (支持追加)
  user_inputs JSONB NOT NULL DEFAULT '[]',  -- [{content, timestamp}]

  -- AI 响应
  assistant_content TEXT,
  assistant_timestamp BIGINT,

  -- 事件流 (完整时序)
  event_stream JSONB NOT NULL DEFAULT '[]',  -- [{type, content, timestamp, meta}]

  -- CC 模式统计
  session_stats JSONB,  -- SessionSummary (仅 geek/evolution)
  cc_session_id TEXT,  -- UUID v5 映射到 CC CLI 会话

  -- 状态
  status TEXT NOT NULL DEFAULT 'pending',  -- 'pending' | 'streaming' | 'completed' | 'error'
  error_message TEXT,

  -- 元数据
  metadata JSONB NOT NULL DEFAULT '{}',
  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),

  -- 约束
  CONSTRAINT fk_ai_block_conversation
    FOREIGN KEY (conversation_id)
    REFERENCES ai_conversation(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_block_type
    CHECK (block_type IN ('MESSAGE', 'CONTEXT_SEPARATOR')),
  CONSTRAINT chk_ai_block_mode
    CHECK (mode IN ('normal', 'geek', 'evolution')),
  CONSTRAINT chk_ai_block_status
    CHECK (status IN ('pending', 'streaming', 'completed', 'error')),
  CONSTRAINT uq_ai_block_conversation_round
    UNIQUE (conversation_id, round_number)
);

-- 索引
CREATE INDEX idx_ai_block_conversation ON ai_block(conversation_id);
CREATE INDEX idx_ai_block_round ON ai_block(conversation_id, round_number);
CREATE INDEX idx_ai_block_mode ON ai_block(mode);
CREATE INDEX idx_ai_block_status ON ai_block(status);
CREATE INDEX idx_ai_block_cc_session ON ai_block(cc_session_id) WHERE cc_session_id IS NOT NULL;

-- JSONB 索引 (GIN)
CREATE INDEX idx_ai_block_event_stream ON ai_block USING gin(event_stream);
CREATE INDEX idx_ai_block_user_inputs ON ai_block USING gin(user_inputs);

-- 触发器: 更新 updated_ts
CREATE OR REPLACE FUNCTION update_ai_block_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_ai_block_updated_ts
  BEFORE UPDATE ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION update_ai_block_updated_ts();

-- 触发器: 更新会话 updated_ts
CREATE OR REPLACE FUNCTION update_conversation_ts_from_block()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE ai_conversation
  SET updated_ts = EXTRACT(EPOCH FROM NOW())::BIGINT
  WHERE id = NEW.conversation_id;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_conversation_from_block
  AFTER INSERT OR UPDATE ON ai_block
  FOR EACH ROW
  EXECUTE FUNCTION update_conversation_ts_from_block();
```

#### 4.1.2 兼容视图 (Compatibility View)

```sql
-- =============================================================================
-- 兼容视图: 旧代码可继续查询
-- =============================================================================

CREATE VIEW v_ai_message AS
SELECT
  id,
  uid,
  conversation_id,
  'MESSAGE' as type,
  CASE
    WHEN round_number % 2 = 0 THEN 'USER'
    ELSE 'ASSISTANT'
  END as role,
  CASE
    WHEN round_number % 2 = 0
    THEN jsonb_array_elements(user_inputs)->>'content'
    ELSE assistant_content
  END as content,
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
  WHERE block_type = 'MESSAGE'
) expanded;
```

### 4.2 接口定义 (API Definitions)

#### 4.2.1 Proto Definitions

```protobuf
// =============================================================================
// Unified Block Model Messages
// =============================================================================

// BlockType defines the type of a block
enum BlockType {
  BLOCK_TYPE_UNSPECIFIED = 0;
  MESSAGE = 1;           // Regular message block
  CONTEXT_SEPARATOR = 2; // Context separator marker
}

// BlockMode defines the execution mode
enum BlockMode {
  BLOCK_MODE_UNSPECIFIED = 0;
  NORMAL = 1;   // Normal AI assistant mode
  GEEK = 2;     // Geek mode (Claude Code CLI)
  EVOLUTION = 3; // Evolution mode (self-improvement)
}

// BlockStatus defines the processing status
enum BlockStatus {
  BLOCK_STATUS_UNSPECIFIED = 0;
  PENDING = 1;    // Waiting to start
  STREAMING = 2;  // Currently processing
  COMPLETED = 3;  // Finished successfully
  ERROR = 4;      // Finished with error
}

// UserInput represents a single user input
message UserInput {
  string content = 1;
  int64 timestamp = 2;
}

// StreamEvent represents a single event in the event stream
message StreamEvent {
  string type = 1;      // "thinking", "tool_use", "tool_result", "answer", "error"
  string content = 2;
  int64 timestamp = 3;
  string meta = 4;      // JSON-encoded metadata
}

// AIBlock represents a single conversation block
message AIBlock {
  int64 id = 1;
  string uid = 2;
  int32 conversation_id = 3;
  int32 round_number = 4;
  BlockType block_type = 5;
  BlockMode mode = 6;

  repeated UserInput user_inputs = 7;
  string assistant_content = 8;
  int64 assistant_timestamp = 9;

  repeated StreamEvent event_stream = 10;
  string session_stats = 11;  // JSON-encoded SessionSummary
  string cc_session_id = 12;

  BlockStatus status = 13;
  string error_message = 14;

  string metadata = 15;  // JSON-encoded metadata
  int64 created_ts = 16;
  int64 updated_ts = 17;
}

// ListBlocksRequest lists blocks in a conversation
message ListBlocksRequest {
  int32 conversation_id = 1 [(google.api.field_behavior) = REQUIRED];
  int32 limit = 2;   // Default: 50
  int32 offset = 3;  // Default: 0
}

// ListBlocksResponse returns blocks
message ListBlocksResponse {
  repeated AIBlock blocks = 1;
  int32 total_count = 2;
  bool has_more = 3;
}

// CreateBlockRequest creates a new block
message CreateBlockRequest {
  int32 conversation_id = 1 [(google.api.field_behavior) = REQUIRED];
  BlockMode mode = 2 [(google.api.field_behavior) = REQUIRED];
  UserInput user_input = 3 [(google.api.field_behavior) = REQUIRED];
}

// AppendUserInputRequest appends a user input to an existing block
message AppendUserInputRequest {
  int64 block_id = 1 [(google.api.field_behavior) = REQUIRED];
  UserInput user_input = 2 [(google.api.field_behavior) = REQUIRED];
}

// UpdateBlockRequest updates a block (streaming state)
message UpdateBlockRequest {
  int64 id = 1 [(google.api.field_behavior) = REQUIRED];

  // Updatable fields
  BlockStatus status = 2;
  string assistant_content = 3;
  repeated StreamEvent event_stream = 4;
  string session_stats = 5;
  string error_message = 6;
}
```

#### 4.2.2 Store Interface

```go
// store/block.go
package store

import (
    "context"
    "time"
)

// BlockMode defines the execution mode
type BlockMode string

const (
    BlockModeNormal    BlockMode = "normal"
    BlockModeGeek      BlockMode = "geek"
    BlockModeEvolution  BlockMode = "evolution"
)

// BlockType defines the type of block
type BlockType string

const (
    BlockTypeMessage         BlockType = "message"
    BlockTypeContextSeparator BlockType = "context_separator"
)

// BlockStatus defines the processing status
type BlockStatus string

const (
    BlockStatusPending   BlockStatus = "pending"
    BlockStatusStreaming BlockStatus = "streaming"
    BlockStatusCompleted BlockStatus = "completed"
    BlockStatusError     BlockStatus = "error"
)

// UserInput represents a single user input
type UserInput struct {
    Content   string `json:"content"`
    Timestamp int64  `json:"timestamp"`
}

// StreamEvent represents a single event in the stream
type StreamEvent struct {
    Type      string         `json:"type"`
    Content   string         `json:"content"`
    Timestamp int64          `json:"timestamp"`
    Meta      map[string]any `json:"meta,omitempty"`
}

// AIBlock represents a conversation block
type AIBlock struct {
    ID               int64
    UID              string
    ConversationID   int32
    RoundNumber      int32
    BlockType        BlockType
    Mode             BlockMode

    // User inputs (support appending)
    UserInputs       []UserInput

    // Assistant response
    AssistantContent *string
    AssistantTimestamp *int64

    // Event stream
    EventStream      []StreamEvent

    // CC mode statistics
    SessionStats     *string // JSON-encoded SessionSummary
    CCSessionID      *string

    // Status
    Status           BlockStatus
    ErrorMessage     *string

    // Metadata
    Metadata         string
    CreatedTs        int64
    UpdatedTs        int64
}

// CreateBlock creates a new block
type CreateBlock struct {
    ConversationID int32
    Mode           BlockMode
    UserInput      UserInput
    BlockType      BlockType
    Metadata       string
}

// UpdateBlock updates block fields
type UpdateBlock struct {
    ID              int64
    Status          *BlockStatus
    AssistantContent *string
    EventStream     *[]StreamEvent
    SessionStats    *string
    ErrorMessage    *string
    UpdatedTs       *int64
}

// AppendUserInput appends a user input to existing block
type AppendUserInput struct {
    ID        int64
    UserInput UserInput
}

// FindBlock filters for listing blocks
type FindBlock struct {
    ConversationID  *int32
    Mode            *BlockMode
    Status          *BlockStatus
    CCSessionID     *string
    Limit           *int
    Offset          *int
}

// BlockStore defines the interface for block operations
type BlockStore interface {
    CreateBlock(ctx context.Context, create *CreateBlock) (*AIBlock, error)
    GetBlock(ctx context.Context, id int64) (*AIBlock, error)
    ListBlocks(ctx context.Context, find *FindBlock) ([]*AIBlock, error)
    UpdateBlock(ctx context.Context, update *UpdateBlock) (*AIBlock, error)
    AppendUserInput(ctx context.Context, append *AppendUserInput) error
    DeleteBlock(ctx context.Context, id int64) error
    GetLatestBlock(ctx context.Context, conversationID int32) (*AIBlock, error)
    GetBlockByRound(ctx context.Context, conversationID int32, roundNumber int32) (*AIBlock, error)
}
```

### 4.3 关键代码路径 (Key Code Paths)

| 文件路径 | 职责 |
|:---------|:-----|
| `store/block.go` | BlockStore 接口定义 |
| `store/db/postgres/block.go` | PostgreSQL BlockStore 实现 |
| `server/service/block/block_service.go` | Block 业务逻辑层 |
| `server/router/api/v1/ai/handler.go` | Chat handler 改造，使用 Block |
| `web/src/types/block.ts` | 前端 Block 类型定义 |
| `web/src/components/AIChat/UnifiedMessageBlock.tsx` | 已有组件，适配 Block 数据 |
| `web/src/hooks/useBlockStream.ts` | Block 流式处理 Hook |

---

## 5. 前端设计 (Frontend Design)

### 5.1 类型定义 (Type Definitions)

```typescript
// web/src/types/block.ts

/**
 * Block execution mode
 */
export type BlockMode = 'normal' | 'geek' | 'evolution';

/**
 * Block processing status
 */
export type BlockStatus = 'pending' | 'streaming' | 'completed' | 'error';

/**
 * Block type
 */
export type BlockType = 'message' | 'context_separator';

/**
 * Single user input (supports appending)
 */
export interface UserInput {
  content: string;
  timestamp: number;
}

/**
 * Stream event in the event stream
 */
export interface StreamEvent {
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
    // ... other metadata
  };
}

/**
 * Unified Block - conversation turn
 */
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
  eventStream: StreamEvent[];

  // CC mode statistics
  sessionStats?: SessionSummary;
  ccSessionId?: string;

  // Status
  status: BlockStatus;
  errorMessage?: string;

  // Metadata
  metadata: Record<string, unknown>;
  createdTs: number;
  updatedTs: number;
}

/**
 * Block creation request
 */
export interface CreateBlockRequest {
  conversationId: number;
  mode: BlockMode;
  userInput: UserInput;
  blockType?: BlockType;
  metadata?: Record<string, unknown>;
}

/**
 * Block update request (for streaming)
 */
export interface UpdateBlockRequest {
  id: string;
  status?: BlockStatus;
  assistantContent?: string;
  eventStream?: StreamEvent[];
  sessionStats?: SessionSummary;
  errorMessage?: string;
}
```

### 5.2 组件适配 (Component Adaptation)

```typescript
// web/src/components/AIChat/UnifiedMessageBlock.tsx

// 改造前:
export interface UnifiedMessageBlockProps {
  userMessage: ConversationMessage;
  assistantMessage?: ConversationMessage;
  sessionSummary?: SessionSummary;
  // ...
}

// 改造后:
export interface UnifiedMessageBlockProps {
  block: AIBlock;  // 直接接收 Block
  isStreaming?: boolean;
  streamingPhase?: "thinking" | "tools" | "answer" | null;
  // ...
}

// 组件内部逻辑简化:
// - 不再需要 groupMessagesIntoBlocks
// - mode 从 block.mode 读取，不再需要从 metadata 推断
// - userMessage 从 block.userInputs[0] 读取
// - assistantMessage 从 block.assistantContent 读取
// - eventStream 从 block.eventStream 读取
// - sessionStats 从 block.sessionStats 读取
```

### 5.3 流式处理 Hook (Streaming Hook)

```typescript
// web/src/hooks/useBlockStream.ts

import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { AIBlock, StreamEvent, BlockStatus } from '@/types/block';

interface UseBlockStreamOptions {
  conversationId: number;
  mode: BlockMode;
  onBlockComplete?: (block: AIBlock) => void;
  onError?: (error: Error) => void;
}

interface BlockStreamState {
  currentBlock: AIBlock | null;
  isStreaming: boolean;
  streamingPhase: 'thinking' | 'tools' | 'answer' | null;
  error: string | null;
}

export function useBlockStream(options: UseBlockStreamOptions) {
  const [state, setState] = useState<BlockStreamState>({
    currentBlock: null,
    isStreaming: false,
    streamingPhase: null,
    error: null,
  });

  const startBlock = useCallback(async (userInput: string) => {
    // 1. 检查是否有未完成的 Block
    const latestBlock = await api.block.getLatest(options.conversationId);

    if (latestBlock && latestBlock.status !== 'completed') {
      // 追加模式
      await api.block.appendUserInput(latestBlock.id, {
        content: userInput,
        timestamp: Date.now(),
      });
      setState(prev => ({ ...prev, currentBlock: latestBlock }));
    } else {
      // 创建新 Block
      const newBlock = await api.block.create({
        conversationId: options.conversationId,
        mode: options.mode,
        userInput: { content: userInput, timestamp: Date.now() },
      });
      setState(prev => ({
        ...prev,
        currentBlock: newBlock,
        isStreaming: true,
        streamingPhase: 'thinking',
      }));
    }
  }, [options.conversationId, options.mode]);

  const handleStreamEvent = useCallback((event: StreamEvent) => {
    setState(prev => {
      if (!prev.currentBlock) return prev;

      const updatedEvents = [...prev.currentBlock.eventStream, event];
      const updatedBlock = {
        ...prev.currentBlock,
        eventStream: updatedEvents,
      };

      // Determine streaming phase
      let streamingPhase: BlockStreamState['streamingPhase'] = null;
      if (event.type === 'thinking') {
        streamingPhase = 'thinking';
      } else if (event.type === 'tool_use') {
        streamingPhase = 'tools';
      } else if (event.type === 'answer') {
        streamingPhase = 'answer';
      }

      return {
        ...prev,
        currentBlock: updatedBlock,
        streamingPhase,
      };
    });
  }, []);

  const completeBlock = useCallback(async (sessionStats?: SessionSummary) => {
    setState(prev => {
      if (!prev.currentBlock) return prev;

      const completedBlock: AIBlock = {
        ...prev.currentBlock,
        status: 'completed',
        sessionStats,
      };

      options.onBlockComplete?.(completedBlock);

      return {
        ...prev,
        currentBlock: completedBlock,
        isStreaming: false,
        streamingPhase: null,
      };
    });
  }, [options]);

  return {
    ...state,
    startBlock,
    handleStreamEvent,
    completeBlock,
  };
}
```

### 5.4 主题色更新 (Theme Colors)

根据调研文档，三种模式的新配色:

| 模式 | 颜色 | 寓意 | Tailwind 基色 |
|:-----|:-----|:-----|:-------------|
| **Normal** | 琥珀 | 闪念如琥珀般珍贵保存 | `amber` |
| **Geek** | 石板蓝 | 代码如石板般精确 | `sky` + `slate` |
| **Evolution** | 翠绿 | 系统如植物般向上生长 | `emerald` |

```typescript
// 更新 PARROT_THEMES
export const PARROT_THEMES = {
  // Normal (琥珀色) - 更新前是绿色
  NORMAL: {
    bubbleBg: "bg-amber-50 dark:bg-amber-900/20",
    bubbleBorder: "border-amber-200 dark:border-amber-700",
    text: "text-amber-800 dark:text-amber-100",
    // ...
  },
  // Geek (石板蓝) - 保持不变
  GEEK: {
    bubbleBg: "bg-sky-50 dark:bg-slate-900/20",
    bubbleBorder: "border-sky-200 dark:border-slate-700",
    text: "text-sky-800 dark:text-slate-100",
    // ...
  },
  // Evolution (翠绿) - 更新前是玫瑰色
  EVOLUTION: {
    bubbleBg: "bg-emerald-50 dark:bg-emerald-900/20",
    bubbleBorder: "border-emerald-200 dark:border-emerald-700",
    text: "text-emerald-800 dark:text-emerald-100",
    // ...
  },
} as const;
```

---

## 6. 实施计划 (Implementation Plan)

### 6.1 时间表 (Timeline)

| 阶段 | 时间 | 任务 | 优先级 |
|:-----|:-----|:-----|:-------|
| **Phase 1** | 2-3 天 | 数据库迁移 + Proto 生成 | P0 |
| **Phase 2** | 3-4 天 | 后端 BlockStore 实现 | P0 |
| **Phase 3** | 2-3 天 | Chat Handler 改造 | P0 |
| **Phase 4** | 2 天 | 前端类型定义 + 主题色更新 | P1 |
| **Phase 5** | 3-4 天 | UnifiedMessageBlock 改造 | P1 |
| **Phase 6** | 2-3 天 | ChatMessages + AIChat Context | P1 |
| **Phase 7** | 2 天 | 集成测试 | P1 |

**总计**: 约 16-22 天

### 6.2 详细任务 (Detailed Tasks)

#### Phase 1: 数据库迁移 (2-3 天)

- [ ] **1.1** 编写 `store/migration/postgres/migrate/XXXX_create_ai_block.up.sql`
- [ ] **1.2** 创建 `ai_block` 表结构
- [ ] **1.3** 创建索引 (conversation_id, round_number, mode, status)
- [ ] **1.4** 创建 JSONB GIN 索引 (event_stream, user_inputs)
- [ ] **1.5** 创建触发器 (updated_ts, conversation updated_ts)
- [ ] **1.6** 创建兼容视图 `v_ai_message`
- [ ] **1.7** 编写 down migration (回滚脚本)
- [ ] **1.8** 运行 `make migration-up` 验证

#### Phase 2: 后端 BlockStore 实现 (3-4 天)

- [ ] **2.1** 定义 `store/block.go` 接口
- [ ] **2.2** 实现 `store/db/postgres/block.go`
  - [ ] 2.2.1 `CreateBlock()`
  - [ ] 2.2.2 `GetBlock()`
  - [ ] 2.2.3 `ListBlocks()`
  - [ ] 2.2.4 `UpdateBlock()`
  - [ ] 2.2.5 `AppendUserInput()`
  - [ ] 2.2.6 `DeleteBlock()`
  - [ ] 2.2.7 `GetLatestBlock()`
  - [ ] 2.2.8 `GetBlockByRound()`
- [ ] **2.3** 实现 `store/db/sqlite/block.go` (仅接口，无 AI 功能)
- [ ] **2.4** 添加单元测试 `store/db/postgres/block_test.go`
- [ ] **2.5** 注册 BlockStore 到 Store 接口

#### Phase 3: Chat Handler 改造 (2-3 天)

- [ ] **3.1** 更新 `proto/api/v1/ai_service.proto` 添加 Block 消息
- [ ] **3.2** 运行 `make generate` 生成代码
- [ ] **3.3** 改造 `server/router/api/v1/ai/handler.go`
  - [ ] 3.3.1 `handleChat()` 使用 BlockStore
  - [ ] 3.3.2 判断逻辑: latestBlock.status != 'completed' → 追加
  - [ ] 3.3.3 流式更新 Block (event_stream, status)
  - [ ] 3.3.4 完成时保存 session_stats
- [ ] **3.4** 添加 `CreateBlock` API 端点
- [ ] **3.5** 添加 `AppendUserInput` API 端点
- [ ] **3.6** 添加 `ListBlocks` API 端点

#### Phase 4: 前端类型定义 (2 天)

- [ ] **4.1** 创建 `web/src/types/block.ts`
- [ ] **4.2** 导出 Block 相关类型
- [ ] **4.3** 更新 `PARROT_THEMES` 配色 (琥珀/石板蓝/翠绿)
- [ ] **4.4** 添加 Block API 调用 (`api/block.ts`)
- [ ] **4.5** 运行 `pnpm check-i18n` 验证

#### Phase 5: UnifiedMessageBlock 改造 (3-4 天)

- [ ] **5.1** 重构 `UnifiedMessageBlockProps` 接收 `block: AIBlock`
- [ ] **5.2** 移除 `userMessage`/`assistantMessage` 分离逻辑
- [ ] **5.3** 从 `block.userInputs` 读取用户输入
- [ ] **5.4** 从 `block.eventStream` 渲染事件流
- [ ] **5.5** 从 `block.sessionStats` 渲染统计
- [ ] **5.6** 根据 `block.mode` 选择主题色
- [ ] **5.7** 更新 `BlockHeader` 显示追加的用户输入
- [ ] **5.8** 测试三种模式的渲染效果

#### Phase 6: ChatMessages + AIChat Context (2-3 天)

- [ ] **6.1** 移除 `groupMessagesIntoBlocks()` 逻辑
- [ ] **6.2** `ChatMessages` 直接接收 `blocks: AIBlock[]`
- [ ] **6.3** 映射 `block.mode` 到 `blockParrotId`
- [ ] **6.4** 更新 `AIChatContext` 管理 Blocks
- [ ] **6.5** 实现 `useBlockStream` Hook
- [ ] **6.6** 更新流式事件处理逻辑

#### Phase 7: 集成测试 (2 天)

- [ ] **7.1** 端到端测试: 创建会话 → 发送消息 → 接收回复
- [ ] **7.2** 测试追加输入功能
- [ ] **7.3** 测试三种模式的持久化
- [ ] **7.4** 测试模式切换
- [ ] **7.5** 测试 CC 会话映射
- [ ] **7.6** 性能测试 (大量 Blocks)
- [ ] **7.7] 向后兼容测试 (旧会话数据)

### 6.3 检查点 (Checkpoints)

- [ ] **Checkpoint 1**: 数据库迁移成功，`ai_block` 表创建完成
- [ ] **Checkpoint 2**: BlockStore 实现通过单元测试
- [ ] **Checkpoint 3**: Chat Handler 可使用 Block 创建和更新
- [ ] **Checkpoint 4**: 前端可渲染 Block (静态数据)
- [ ] **Checkpoint 5**: 流式更新 Block 正常工作
- [ ] **Checkpoint 6**: 端到端流程验证通过

---

## 7. 测试验收 (Testing & Acceptance)

### 7.1 功能测试 (Functional Tests)

| 场景 | 输入 | 预期输出 |
|:-----|:-----|:---------|
| **创建 Block** | 用户发送消息 "你好" | 新 Block 创建，status='pending', user_inputs=[{content: "你好"}] |
| **追加输入** | Block status='streaming'，用户追加 "等一下" | user_inputs 追加第二个元素 |
| **新回合** | Block status='completed'，用户发送新消息 | 创建新 Block，round_number+1 |
| **流式更新** | 接收 thinking 事件 | event_stream 追加 thinking 事件，status='streaming' |
| **完成 Block** | AI 回复完成 | status='completed', session_stats 填充 |
| **模式切换** | 同一会话内切换 mode | Block.mode 独立保存，互不影响 |
| **CC 会话映射** | Geek 模式创建 Block | cc_session_id 为 UUID v5 格式 |

### 7.2 性能验收 (Performance)

| 指标 | 目标值 | 测试方法 |
|:-----|:-------|:---------|
| Block 创建延迟 | < 50ms | 单元测试 |
| Block 更新延迟 | < 20ms | 单元测试 |
| ListBlocks (100) | < 100ms | 集成测试 |
| JSONB 索引查询 | < 50ms | EXPLAIN ANALYZE |
| 流式事件追加 | < 10ms | 压测工具 |

### 7.3 集成验收 (Integration)

- [ ] 与现有 `ai_conversation` 表集成测试
- [ ] 与 `agent_session_stats` 表集成测试
- [ ] 与 CC Runner 流式事件集成测试
- [ ] 前端与后端 API 对接测试

---

## 8. 向后兼容性 (Backward Compatibility)

### 8.1 数据迁移策略 (Migration Strategy)

```
阶段 1: 双写期 (2-4 周)
┌─────────────────────────────────────────────────────────────┐
│  新会话: 同时写 ai_block 和 ai_message                        │
│  旧会话: 继续使用 ai_message                                   │
│  前端: 优先读取 ai_block，降级到 ai_message                     │
└─────────────────────────────────────────────────────────────┘

阶段 2: 读取切换 (1-2 周)
┌─────────────────────────────────────────────────────────────┐
│  所有会话: 只读 ai_block                                       │
│  ai_message: 标记为 deprecated                                │
│  兼容视图: v_ai_message 保留供旧代码使用                        │
└─────────────────────────────────────────────────────────────┘

阶段 3: 清理期 (1-2 周后)
┌─────────────────────────────────────────────────────────────┐
│  删除 ai_message 双写逻辑                                       │
│  保留 v_ai_message 视图至少 1 个版本周期                         │
│  监控错误率，确保无兼容性问题                                     │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 兼容视图 (Compatibility View)

```sql
-- 前端可继续使用兼容视图查询
SELECT * FROM v_ai_message WHERE conversation_id = 123;

-- 或逐步迁移到新表
SELECT * FROM ai_block WHERE conversation_id = 123;
```

### 8.3 前端兼容层 (Frontend Compatibility)

```typescript
// web/src/utils/blockCompatibility.ts

/**
 * 从旧数据结构构建 Block
 */
export function legacyMessageToBlock(
  userMessage: ConversationMessage,
  assistantMessage?: ConversationMessage
): AIBlock {
  const mode = assistantMessage?.metadata?.mode || 'normal';

  return {
    id: `legacy-${userMessage.id}`,
    uid: userMessage.uid,
    conversationId: userMessage.conversationId,
    roundNumber: 0,
    blockType: 'message',
    mode,
    userInputs: [{ content: userMessage.content, timestamp: userMessage.timestamp }],
    assistantContent: assistantMessage?.content,
    assistantTimestamp: assistantMessage?.timestamp,
    eventStream: assistantMessage?.metadata?.toolCalls?.map(tc => ({
      type: 'tool_use',
      content: tc.name || '',
      timestamp: Date.now(),
      meta: { tool_id: tc.toolId, input_summary: tc.inputSummary },
    })) || [],
    status: assistantMessage ? 'completed' : 'pending',
    metadata: assistantMessage?.metadata || {},
    createdTs: userMessage.timestamp,
    updatedTs: assistantMessage?.timestamp || userMessage.timestamp,
  };
}
```

---

## 9. 风险与缓解 (Risks & Mitigation)

| 风险 | 概率 | 影响 | 缓解措施 |
|:-----|:-----|:-----|:---------|
| **数据迁移失败** | 中 | 高 | 1. 充分测试 migration 脚本 2. 保留备份 3. 双写期验证 |
| **前端性能下降** | 中 | 中 | 1. JSONB 索引优化 2. 分页加载 3. 虚拟滚动 |
| **CC 会话映射冲突** | 低 | 中 | 1. UUID v5 确定性算法 2. 唯一约束 3. 冲突检测 |
| **向后兼容性问题** | 中 | 高 | 1. 兼容视图 2. 渐进式迁移 3. 充分测试 |
| **JSONB 解析开销** | 中 | 低 | 1. 索引优化 2. 缓存热点数据 3. 监控性能 |

---

## 10. ROI 分析 (ROI Analysis)

| 维度 | 值 |
|:-----|:---:|
| **开发投入** | 16-22 人天 |
| **预期收益** | 数据模型统一，支持完整持久化和追加输入 |
| **风险评估** | 中 (主要风险在数据迁移) |
| **回报周期** | 1-2 个 Sprint |

---

## 11. 附录 (Appendix)

### A. 参考资料 (References)

- [Issue #69: Warp Block UI](https://github.com/hrygo/divinesense/issues/69)
- [Issue #71: Unified Block Model](https://github.com/hrygo/divinesense/issues/71)
- [Issue #57: 会话嵌套模型](https://github.com/hrygo/divinesense/issues/57)
- [CC Runner 异步架构](./cc_runner_async_arch.md)
- [统一 Block 模型调研](../research/unified-block-model-research.md)
- [前端开发指南](../dev-guides/FRONTEND.md)

### B. 变更记录 (Change Log)

| 日期 | 版本 | 变更内容 | 作者 |
|:-----|:-----|:---------|:-----|
| 2026-02-04 | v1.0 | 初始版本 | Claude |

### C. 术语表 (Glossary)

| 术语 | 定义 |
|:-----|:-----|
| **Block** | 对话回合的一等公民单元，包含用户输入和 AI 响应 |
| **Round Number** | 会话内的 Block 序号 (0-based) |
| **Mode** | 执行模式 (normal/geek/evolution) |
| **Event Stream** | 完整的事件时序流 (thinking/tool_use/answer) |
| **CC Session ID** | Claude Code CLI 会话的 UUID v5 映射 |
| **追加输入** | 在 Block 未完成时追加用户输入 |

---

*Spec 完成: 2026-02-04*
*关联 PR: 待创建*
