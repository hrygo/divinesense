# P1-A001: Unified Block Model - Phase 1 Database & Backend

> **状态**: 🔲 待开发
> **优先级**: P0 (核心)
> **投入**: 5人天
> **Sprint**: Sprint 1
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖调研**: [unified-block-model-research.md](../research/unified-block-model-research.md)

---

## 1. 目标与背景

### 1.1 核心目标

实现统一 Block 模型的数据库层和后端 Store 层，将 `Block` 作为"对话回合"的一等公民持久化单元，解决普通模式与 CC 连接模式（极客/进化）之间的数据结构割裂问题。

### 1.2 用户价值

- **完整对话历史保留**：Warp Block UI 中的所有内容（思考、工具调用、会话统计）都能持久化
- **跨模式一致性**：普通模式、Geek 模式、Evolution 模式使用统一的数据结构
- **追加式输入支持**：用户可以在 AI 回复完成前追加输入，全部记录在同一个 Block 中

### 1.3 技术价值

- **数据结构统一**：消除 `ai_message` 与 `agent_session_stats` 之间的割裂
- **前端简化**：前端可以直接读取 Block 完整状态，无需复杂配对逻辑
- **扩展性增强**：为未来的会话嵌套模型（Issue #57）奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Issue #69**: Warp Block UI 已实现前端组件
- [x] **调研报告**: unified-block-model-research.md 已完成

### 2.2 并行依赖（可同步进行）

- [ ] **P1-A002**: 前端类型定义更新（可同步开发）

### 2.3 后续依赖（依赖本 Spec）

- [ ] **P1-A003**: Chat Handler 改造
- [ ] **P1-A004**: 前端适配改造

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│  Conversation #123                                              │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Block #0 (mode='geek')                                     ││
│  │  user_inputs: [{"content": "分析代码", "timestamp": ...}]   ││
│  │  event_stream: [{type: "thinking", ...}, ...]            ││
│  │  session_stats: {total_cost_usd: 0.0123, ...}             ││
│  │  status: completed                                         ││
│  ├─────────────────────────────────────────────────────────────┤│
│  │  Block #1 (mode='normal')                                    ││
│  │  user_inputs: [{"content": "总结一下"}]                      ││
│  │  assistant_content: "今天我们分析了..."                     ││
│  │  status: completed                                         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **用户输入判断**：判断最新 Block 状态
   - `status != 'completed'` → 追加到当前 Block
   - `status == 'completed'` → 创建新 Block

2. **Block 创建**：记录用户输入、模式、初始状态

3. **事件流式写入**：AI 响应过程中的事件（thinking/tool_use/answer）写入 `event_stream`

4. **Block 完成**：AI 响应结束后，更新 `status` 为 `completed`，写入 `session_stats`

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| **兼容策略** | 立即删除 `ai_message` 表 | 保留旧表，创建兼容视图 | **B** | 平滑迁移，降低风险 |
| **用户输入存储** | 单一字段 | JSONB 数组 | **B** | 支持追加式输入 |
| **事件流存储** | 独立表 | JSONB 字段 | **B** | 简化查询，支持时间线重构 |
| **Block ID** | 自增 ID | UUID | **A** | 与现有 `ai_message` 一致 |

---

## 4. 技术实现

### 4.1 数据模型

#### 4.1.1 `ai_block` 表

```sql
CREATE TABLE ai_block (
  id BIGSERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  conversation_id INTEGER NOT NULL,
  round_number INTEGER NOT NULL DEFAULT 0,

  -- Block 类型
  block_type TEXT NOT NULL DEFAULT 'message',
  -- 'message': 用户-AI 对话回合
  -- 'context_separator': 上下文分隔符

  -- AI 模式
  mode TEXT NOT NULL DEFAULT 'normal',
  -- 'normal': 普通模式（AI 助理）
  -- 'geek': 极客模式（Claude Code CLI）
  -- 'evolution': 进化模式（自我进化）

  -- 用户输入（支持追加式）
  user_inputs JSONB NOT NULL DEFAULT '[]',
  -- [{"content": "输入内容", "timestamp": 1234567890, "metadata": {...}}]

  -- AI 回复
  assistant_content TEXT,
  assistant_timestamp BIGINT,

  -- 事件流（按时间顺序）
  event_stream JSONB NOT NULL DEFAULT '[]',
  -- [{type: "thinking", content: "...", timestamp: ..., meta: {...}}, ...]

  -- 会话统计（CC 模式）
  session_stats JSONB,
  -- {session_id: "...", total_cost_usd: 0.0123, total_tokens: 1234, ...}

  -- CC 会话映射
  cc_session_id TEXT,
  -- UUID v5 映射到 Claude Code CLI 会话

  -- 状态
  status TEXT NOT NULL DEFAULT 'pending',
  -- 'pending': 等待 AI 响应
  -- 'streaming': AI 正在响应
  -- 'completed': 响应完成
  -- 'error': 发生错误

  -- 扩展字段
  metadata JSONB NOT NULL DEFAULT '{}',
  -- {error_message: "...", parrot_id: "MEMO", ...}

  created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),

  CONSTRAINT fk_ai_block_conversation
    FOREIGN KEY (conversation_id)
    REFERENCES ai_conversation(id)
    ON DELETE CASCADE,
  CONSTRAINT chk_ai_block_type
    CHECK (block_type IN ('message', 'context_separator')),
  CONSTRAINT chk_ai_block_mode
    CHECK (mode IN ('normal', 'geek', 'evolution')),
  CONSTRAINT chk_ai_block_status
    CHECK (status IN ('pending', 'streaming', 'completed', 'error'))
);

-- 索引
CREATE INDEX idx_ai_block_conversation ON ai_block(conversation_id);
CREATE INDEX idx_ai_block_created ON ai_block(created_ts ASC);
CREATE INDEX idx_ai_block_round ON ai_block(conversation_id, round_number);
CREATE INDEX idx_ai_block_status ON ai_block(status) WHERE status != 'completed';
CREATE INDEX idx_ai_block_cc_session ON ai_block(cc_session_id) WHERE cc_session_id IS NOT NULL;

-- JSONB 索引（可选，用于查询特定事件类型）
CREATE INDEX idx_ai_block_event_stream ON ai_block USING gin(event_stream);

-- 更新时间戳触发器
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
```

#### 4.1.2 兼容视图

```sql
-- 保留对旧 ai_message 表的兼容
CREATE VIEW v_ai_message AS
SELECT
  id,
  uid,
  conversation_id,
  'MESSAGE' as type,
  CASE
    WHEN block_type = 'context_separator' THEN 'SEPARATOR'
    ELSE 'MESSAGE'
  END as message_type,
  -- 从 user_inputs 提取第一个用户输入
  CASE
    WHEN jsonb_array_length(user_inputs) > 0
    THEN (user_inputs->0->>'content')
    ELSE ''
  END as user_content,
  assistant_content as content,
  metadata,
  created_ts
FROM (
  SELECT
    id,
    uid,
    conversation_id,
    block_type,
    mode,
    user_inputs,
    assistant_content,
    event_stream,
    session_stats,
    metadata,
    created_ts,
    -- 为兼容性，将 mode 和错误信息合并到 metadata
    jsonb_build_object(
      'mode', mode,
      'error', CASE WHEN status = 'error' THEN metadata->>'error_message ELSE NULL END,
      'event_stream', event_stream,
      'session_stats', session_stats
    ) || metadata as metadata_full,
    created_ts
  FROM ai_block
  WHERE block_type = 'message'
) expanded;
```

### 4.2 Store 接口定义

#### 4.2.1 Go 结构体

```go
// AIBlock represents a conversation block (round)
type AIBlock struct {
    ID              int64
    UID             string
    ConversationID  int32
    RoundNumber     int32
    BlockType       AIBlockType
    Mode            AIBlockMode
    UserInputs      []UserInput
    AssistantContent string
    AssistantTimestamp int64
    EventStream     []BlockEvent
    SessionStats    *SessionStats
    CCSessionID     string
    Status          AIBlockStatus
    Metadata        map[string]any
    CreatedTs       int64
    UpdatedTs       int64
}

// AIBlockType represents the block type
type AIBlockType string

const (
    AIBlockTypeMessage          AIBlockType = "message"
    AIBlockTypeContextSeparator AIBlockType = "context_separator"
)

// AIBlockMode represents the AI mode
type AIBlockMode string

const (
    AIBlockModeNormal    AIBlockMode = "normal"
    AIBlockModeGeek      AIBlockMode = "geek"
    AIBlockModeEvolution AIBlockMode = "evolution"
)

// UserInput represents a single user input in the block
type UserInput struct {
    Content   string          `json:"content"`
    Timestamp int64           `json:"timestamp"`
    Metadata  map[string]any  `json:"metadata,omitempty"`
}

// BlockEvent represents an event in the event stream
type BlockEvent struct {
    Type      string          `json:"type"` // "thinking", "tool_use", "tool_result", "answer", "error"
    Content   string          `json:"content,omitempty"`
    Timestamp int64           `json:"timestamp"`
    Meta      map[string]any  `json:"meta,omitempty"`
}

// AIBlockStatus represents the block status
type AIBlockStatus string

const (
    AIBlockStatusPending   AIBlockStatus = "pending"
    AIBlockStatusStreaming AIBlockStatus = "streaming"
    AIBlockStatusCompleted AIBlockStatus = "completed"
    AIBlockStatusError     AIBlockStatus = "error"
)

// CreateAIBlock represents the input for creating a block
type CreateAIBlock struct {
    UID            string
    ConversationID int32
    BlockType      AIBlockType
    Mode           AIBlockMode
    UserInputs     []UserInput
    Metadata       map[string]any
    CreatedTs      int64
    UpdatedTs      int64
}

// UpdateAIBlock represents the input for updating a block
type UpdateAIBlock struct {
    ID                int64
    UserInputs        []UserInput           // 追加用户输入
    AssistantContent  *string               // 更新 AI 回复
    EventStream       []BlockEvent          // 追加事件
    SessionStats      *SessionStats         // 更新会话统计
    CCSessionID       *string               // 更新 CC 会话 ID
    Status            *AIBlockStatus        // 更新状态
    Metadata          map[string]any        // 合并元数据
    UpdatedTs         *int64
}

// FindAIBlock represents the filter for finding blocks
type FindAIBlock struct {
    ID              *int64
    UID             *string
    ConversationID  *int32
    Status          *AIBlockStatus
    Mode            *AIBlockMode
    CCSessionID     *string
}
```

#### 4.2.2 Store 接口方法

```go
type AIBlockStore interface {
    // CreateBlock creates a new block
    CreateBlock(ctx context.Context, create *CreateAIBlock) (*AIBlock, error)

    // GetBlock retrieves a block by ID
    GetBlock(ctx context.Context, id int64) (*AIBlock, error)

    // ListBlocks retrieves blocks for a conversation
    ListBlocks(ctx context.Context, find *FindAIBlock) ([]*AIBlock, error)

    // UpdateBlock updates a block
    UpdateBlock(ctx context.Context, update *UpdateAIBlock) (*AIBlock, error)

    // AppendUserInput appends a user input to an existing block
    AppendUserInput(ctx context.Context, blockID int64, input UserInput) error

    // AppendEvent appends an event to the event stream
    AppendEvent(ctx context.Context, blockID int64, event BlockEvent) error

    // UpdateStatus updates the block status
    UpdateStatus(ctx context.Context, blockID int64, status AIBlockStatus) error

    // DeleteBlock deletes a block
    DeleteBlock(ctx context.Context, id int64) error

    // GetLatestBlock retrieves the latest block for a conversation
    GetLatestBlock(ctx context.Context, conversationID int32) (*AIBlock, error)

    // GetPendingBlocks retrieves all pending/streaming blocks for cleanup
    GetPendingBlocks(ctx context.Context) ([]*AIBlock, error)
}
```

### 4.3 关键代码路径

| 文件路径 | 职责 |
|:---|:---|
| `store/ai_block.go` | AIBlockStore 接口定义 |
| `store/db/postgres/ai_block.go` | PostgreSQL 实现 |
| `store/db/sqlite/ai_block.go` | SQLite 实现（仅开发环境，功能受限） |
| `store/migration/postgres/migrate/20260204000000_add_ai_block.up.sql` | 数据库迁移 |
| `store/migration/postgres/migrate/20260204000000_add_ai_block.down.sql` | 回滚脚本 |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `store/ai_block.go` - AIBlockStore 接口定义
- [ ] `store/db/postgres/ai_block.go` - PostgreSQL 实现
- [ ] `store/db/sqlite/ai_block.go` - SQLite 实现（空实现，返回错误）
- [ ] `store/db/postgres/common.go` - 添加 AIBlock 相关的辅助函数

### 5.2 数据库变更

- [ ] `store/migration/postgres/migrate/20260204000000_add_ai_block.up.sql` - 创建 ai_block 表
- [ ] `store/migration/postgres/migrate/20260204000000_add_ai_block.down.sql` - 回滚脚本
- [ ] `store/migration/postgres/schema/LATEST.sql` - 更新 schema 定义

### 5.3 配置变更

- [ ] `store/migrator.go` - 确保 migrator 能正确执行新迁移

### 5.4 文档更新

- [ ] `docs/dev-guides/BACKEND_DB.md` - 添加 ai_block 表说明
- [ ] `docs/specs/unified-block-model.md` - 更新实现状态

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| **创建 Block** | CreateAIBlock{ConversationID: 1, Mode: "normal"} | 返回 AIBlock，ID 分配成功 |
| **追加用户输入** | AppendUserInput(blockID, {content: "补充说明"}) | UserInputs 数组长度增加 1 |
| **追加事件** | AppendEvent(blockID, {type: "thinking"}) | EventStream 数组长度增加 1 |
| **更新状态** | UpdateStatus(blockID, "completed") | Status 字段更新为 "completed" |
| **获取最新 Block** | GetLatestBlock(conversationID) | 返回 round_number 最大的 Block |
| **查询待处理 Block** | GetPendingBlocks() | 返回 status != 'completed' 的所有 Block |
| **CC 会话映射** | CreateBlock{CCSessionID: "uuid-v5-123"} | cc_session_id 正确存储 |
| **兼容视图查询** | SELECT * FROM v_ai_message WHERE conversation_id = 1 | 返回与 ai_message 表相同的结构 |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| 创建 Block 延迟 | < 10ms | 单线程压测 |
| 追加事件延迟 | < 5ms | 单线程压测 |
| 查询会话 Blocks | < 50ms (100 blocks) | 一次性查询 |
| JSONB 解析性能 | < 1ms/event | 内存基准测试 |

### 6.3 集成验收

- [ ] 迁移脚本在 PostgreSQL 16+ 上成功执行
- [ ] 回滚脚本能正确清理 ai_block 表
- [ ] 兼容视图 v_ai_message 返回正确数据
- [ ] 与现有 ai_conversation 表的外键约束正常工作
- [ ] 触发器正确更新 updated_ts 字段

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 5人天 |
| 预期收益 | 完整对话历史持久化，支持跨模式数据统一 |
| 风险评估 | 中（数据结构重构） |
| 回报周期 | 2 Sprint |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| **数据迁移失败** | 中 | 高 | 保留旧表，创建兼容视图，逐步迁移 |
| **JSONB 性能问题** | 低 | 中 | 添加 GIN 索引，热点数据缓存 |
| **外键约束冲突** | 低 | 中 | 充分测试 FK 级联删除 |
| **SQLite 兼容性** | 中 | 低 | SQLite 使用空实现，明确文档说明 |

---

## 9. 实施计划

### 9.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| **Day 1** | 1人天 | 创建迁移脚本，在本地测试 |
| **Day 2** | 1人天 | 实现 AIBlockStore 接口 |
| **Day 3** | 1人天 | 实现 PostgreSQL AIBlockStore |
| **Day 4** | 1人天 | 编写单元测试 |
| **Day 5** | 1人天 | 集成测试，文档更新 |

### 9.2 检查点

- [ ] Checkpoint 1: 迁移脚本成功执行，表结构正确
- [ ] Checkpoint 2: 单元测试覆盖率 > 80%
- [ ] Checkpoint 3: 集成测试通过，兼容视图返回正确数据

---

## 附录

### A. 参考资料

- [调研报告](../research/unified-block-model-research.md)
- [Issue #71](https://github.com/hrygo/divinesense/issues/71)
- [后端开发指南](../dev-guides/BACKEND_DB.md)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-02-04 | v1.0 | 初始版本 | Claude |

### C. 迁移示例

从 `ai_message` 迁移到 `ai_block` 的数据转换逻辑：

```sql
-- 迁移脚本（仅示例，实际迁移在后续 Phase）
INSERT INTO ai_block (
    uid, conversation_id, round_number, block_type, mode,
    user_inputs, assistant_content, status,
    event_stream, metadata, created_ts, updated_ts
)
SELECT
    gen_random_uuid()::text as uid,
    conversation_id,
    (ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY created_ts) - 1) / 2 as round_number,
    'message' as block_type,
    'normal' as mode,
    CASE
        WHEN role = 'USER' THEN jsonb_build_array(jsonb_build_object(
            'content', content,
            'timestamp', created_ts
        ))
        ELSE '[]'::jsonb
    END as user_inputs,
    CASE WHEN role = 'ASSISTANT' THEN content ELSE NULL END as assistant_content,
    'completed' as status,
    '[]'::jsonb as event_stream,
    metadata,
    created_ts,
    updated_ts
FROM (
    SELECT *,
        LAG(role) OVER (PARTITION BY conversation_id ORDER BY created_ts) as prev_role
    FROM ai_message
    WHERE type = 'MESSAGE'
) paired
WHERE role = 'ASSISTANT' OR prev_role IS NULL;
```
