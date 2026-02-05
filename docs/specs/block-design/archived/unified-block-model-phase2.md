# P1-A002: Unified Block Model - Phase 2 Proto & API

> **状态**: 🔲 待开发
> **优先级**: P0 (核心)
> **投入**: 3人天
> **Sprint**: Sprint 1
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖**: Phase 1 (Database & Backend)

---

## 1. 目标与背景

### 1.1 核心目标

定义 Block 相关的 Protobuf 消息类型，并更新 AI Chat API 以支持 Block 操作。

### 1.2 用户价值

- **API 兼容性**：前端可以一次性获取完整的 Block 数据，减少网络往返
- **实时更新**：支持流式更新 Block 状态

### 1.3 技术价值

- **类型安全**：通过 Protobuf 确保前后端数据结构一致
- **版本管理**：API 变更有明确的版本控制

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Phase 1**: 数据库表和 Store 接口已定义

### 2.2 并行依赖（可同步进行）

- [ ] **P1-A003**: 前端类型定义更新

### 2.3 后续依赖（依赖本 Spec）

- [ ] **P1-A004**: Chat Handler 改造

---

## 3. 功能设计

### 3.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│  Frontend                                                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  ListBlocksRequest → ListBlocksResponse                     ││
│  │  CreateBlockRequest → CreateBlockResponse                  ││
│  │  StreamChatResponse (扩展) → 新增 Block 字段                ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Backend (Connect RPC)                                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  AIService.BlockOperations (新增 RPC)                       ││
│  │  AIService.Chat (扩展 StreamChatResponse)                   ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Store Layer                                                    │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  AIBlockStore (Phase 1 已定义)                              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **前端请求 Blocks**：调用 `ListBlocks` API 获取会话的所有 Block
2. **流式更新**：`StreamChat` 事件中包含 `block_id`，前端可实时更新 Block 状态
3. **创建 Block**：用户发送第一条消息时，自动创建新 Block

### 3.3 关键决策

| 决策点 | 方案 A | 方案 B | 选择 | 理由 |
|:---|:---|:---|:---:|:---|
| **Block RPC** | 新增独立的 BlockService | 扩展 AIService | **A** | 职责分离，便于维护 |
| **事件流格式** | JSON 字符串 | 重复字段 | **A** | 减少消息复杂度 |

---

## 4. 技术实现

### 4.1 Proto 定义

#### 4.1.1 新增消息类型

```protobuf
// proto/api/v1/ai_service.proto (追加)

// ============================================================================
// Block Messages (Phase 2)
// ============================================================================

// BlockType represents the type of a conversation block
enum BlockType {
  BLOCK_TYPE_UNSPECIFIED = 0;
  MESSAGE = 1;           // User-AI conversation round
  CONTEXT_SEPARATOR = 2; // Context separator marker
}

// BlockMode represents the AI mode for this block
enum BlockMode {
  BLOCK_MODE_UNSPECIFIED = 0;
  NORMAL = 1;    // Normal AI assistant mode
  GEEK = 2;      // Geek mode (Claude Code CLI)
  EVOLUTION = 3; // Evolution mode (self-improvement)
}

// BlockStatus represents the current status of a block
enum BlockStatus {
  BLOCK_STATUS_UNSPECIFIED = 0;
  PENDING = 1;   // Waiting for AI response
  STREAMING = 2; // AI is currently responding
  COMPLETED = 3; // Response completed
  ERROR = 4;     // Error occurred
}

// UserInput represents a single user input in the block
message UserInput {
  string content = 1;
  int64 timestamp = 2;
  map<string, string> metadata = 3;
}

// BlockEvent represents an event in the event stream
message BlockEvent {
  string type = 1; // "thinking", "tool_use", "tool_result", "answer", "error"
  string content = 2;
  int64 timestamp = 3;
  map<string, string> meta = 4;
}

// AIBlock represents a conversation block (round)
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

  repeated BlockEvent event_stream = 10;
  SessionSummary session_stats = 11;

  string cc_session_id = 12;
  BlockStatus status = 13;

  map<string, string> metadata = 14;

  int64 created_ts = 15;
  int64 updated_ts = 16;
}

// ============================================================================
// Block RPC Service
// ============================================================================

// Extend AIService with Block operations
service AIService {
  // ... existing methods ...

  // ListBlocks retrieves blocks for a conversation
  rpc ListBlocks(ListBlocksRequest) returns (ListBlocksResponse) {
    option (google.api.http) = {
      get: "/api/v1/ai/conversations/{conversation_id}/blocks"
    };
  }

  // GetBlock retrieves a specific block
  rpc GetBlock(GetBlockRequest) returns (AIBlock) {
    option (google.api.http) = {
      get: "/api/v1/ai/blocks/{id}"
    };
  }

  // AppendUserInput appends a user input to an existing block
  rpc AppendUserInput(AppendUserInputRequest) returns (AIBlock) {
    option (google.api.http) = {
      post: "/api/v1/ai/blocks/{id}/input"
      body: "*"
    };
  }

  // UpdateBlockStatus updates the status of a block
  rpc UpdateBlockStatus(UpdateBlockStatusRequest) returns (AIBlock) {
    option (google.api.http) = {
      patch: "/api/v1/ai/blocks/{id}/status"
      body: "*"
    };
  }
}

// ============================================================================
// Block Request/Response Messages
// ============================================================================

// ListBlocksRequest is the request for ListBlocks
message ListBlocksRequest {
  int32 conversation_id = 1 [(google.api.field_behavior) = REQUIRED];
  BlockStatus status = 2; // Filter by status (optional)
  int32 limit = 3;         // Max blocks to return (default: 100)
  string last_block_uid = 4; // For pagination
}

// ListBlocksResponse is the response for ListBlocks
message ListBlocksResponse {
  repeated AIBlock blocks = 1;
  bool has_more = 2;
  string latest_block_uid = 3;
}

// GetBlockRequest is the request for GetBlock
message GetBlockRequest {
  int64 id = 1 [(google.api.field_behavior) = REQUIRED];
}

// AppendUserInputRequest is the request for AppendUserInput
message AppendUserInputRequest {
  int64 id = 1 [(google.api.field_behavior) = REQUIRED];
  string content = 2 [(google.api.field_behavior) = REQUIRED];
  map<string, string> metadata = 3;
}

// UpdateBlockStatusRequest is the request for UpdateBlockStatus
message UpdateBlockStatusRequest {
  int64 id = 1 [(google.api.field_behavior) = REQUIRED];
  BlockStatus status = 2 [(google.api.field_behavior) = REQUIRED];
  map<string, string> metadata = 3; // Optional error message, etc.
}
```

#### 4.1.2 扩展现有消息

```protobuf
// Extend ChatResponse to include block information
message ChatResponse {
  // ... existing fields ...

  // Block information (Phase 2)
  int64 block_id = 10;           // Block ID for this response
  string block_uid = 11;         // Block UID for incremental sync
  BlockStatus block_status = 12; // Current block status
}

// Extend AIConversation to include blocks summary
message AIConversation {
  // ... existing fields ...

  // Block summary (Phase 2)
  int32 block_count = 11;        // Total number of blocks
  int64 latest_block_id = 12;    // Latest block ID
  string latest_block_uid = 13;  // Latest block UID
}
```

### 4.2 API Handler 实现

#### 4.2.1 文件结构

```
server/router/api/v1/ai/
├── handler.go          # 主处理器（扩展）
├── block_handler.go    # Block 专用处理器（新增）
└── streamer.go         # 流式响应处理器（扩展）
```

#### 4.2.2 Block Handler 接口

```go
// server/router/api/v1/ai/block_handler.go

package ai

import (
    "context"
    "fmt"

    "connectrpc.com/connect"
    "github.com/hrygo/divinesense/gen/api/v1"
    "github.com/hrygo/divinesense/store"
    "github.com/hrygo/divinesense/store/db/postgres"
)

type BlockHandler struct {
    db *postgres.DB
}

func NewBlockHandler(db *postgres.DB) *BlockHandler {
    return &BlockHandler{db: db}
}

// ListBlocks implements AIService.ListBlocks
func (h *BlockHandler) ListBlocks(
    ctx context.Context,
    req *connect.Request[aiv1.ListBlocksRequest],
) (*connect.Response[aiv1.ListBlocksResponse], error) {
    // Validate conversation_id
    if req.Msg.ConversationId == 0 {
        return nil, connect.NewError(
            connect.CodeInvalidArgument,
            fmt.Errorf("conversation_id is required"),
        )
    }

    // Build find criteria
    find := &store.FindAIBlock{
        ConversationID: &req.Msg.ConversationId,
    }
    if req.Msg.Status != aiv1.BlockStatus_BLOCK_STATUS_UNSPECIFIED {
        status := convertBlockStatusToStore(req.Msg.Status)
        find.Status = &status
    }

    // Query blocks
    blocks, err := h.db.ListAIBlocks(ctx, find)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    // Convert to proto
    protoBlocks := make([]*aiv1.AIBlock, len(blocks))
    for i, b := range blocks {
        protoBlocks[i] = convertAIBlockToProto(b)
    }

    // Determine pagination info
    var hasMore bool
    var latestBlockUID string
    if len(blocks) > 0 {
        latestBlockUID = blocks[len(blocks)-1].UID
        hasMore = int32(len(blocks)) >= req.Msg.Limit
    }

    return connect.NewResponse(&aiv1.ListBlocksResponse{
        Blocks:          protoBlocks,
        HasMore:         hasMore,
        LatestBlockUid:  latestBlockUID,
    }), nil
}

// GetBlock implements AIService.GetBlock
func (h *BlockHandler) GetBlock(
    ctx context.Context,
    req *connect.Request[aiv1.GetBlockRequest],
) (*connect.Response[aiv1.AIBlock], error) {
    block, err := h.db.GetAIBlock(ctx, req.Msg.Id)
    if err != nil {
        return nil, connect.NewError(connect.CodeNotFound, err)
    }

    return connect.NewResponse(convertAIBlockToProto(block)), nil
}

// AppendUserInput implements AIService.AppendUserInput
func (h *BlockHandler) AppendUserInput(
    ctx context.Context,
    req *connect.Request[aiv1.AppendUserInputRequest],
) (*connect.Response[aiv1.AIBlock], error) {
    input := store.UserInput{
        Content:   req.Msg.Content,
        Timestamp: time.Now().Unix(),
        Metadata:  req.Msg.Metadata,
    }

    if err := h.db.AppendUserInput(ctx, req.Msg.Id, input); err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    block, err := h.db.GetAIBlock(ctx, req.Msg.Id)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    return connect.NewResponse(convertAIBlockToProto(block)), nil
}

// UpdateBlockStatus implements AIService.UpdateBlockStatus
func (h *BlockHandler) UpdateBlockStatus(
    ctx context.Context,
    req *connect.Request[aiv1.UpdateBlockStatusRequest],
) (*connect.Response[aiv1.AIBlock], error) {
    status := convertBlockStatusToStore(req.Msg.Status)

    if err := h.db.UpdateAIBlockStatus(ctx, req.Msg.Id, status); err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    block, err := h.db.GetAIBlock(ctx, req.Msg.Id)
    if err != nil {
        return nil, connect.NewError(connect.CodeInternal, err)
    }

    return connect.NewResponse(convertAIBlockToProto(block)), nil
}

// Helper functions

func convertAIBlockToProto(b *store.AIBlock) *aiv1.AIBlock {
    proto := &aiv1.AIBlock{
        Id:             b.ID,
        Uid:            b.UID,
        ConversationId: b.ConversationID,
        RoundNumber:    b.RoundNumber,
        BlockType:      convertBlockTypeToProto(b.BlockType),
        Mode:           convertBlockModeToProto(b.Mode),
        UserInputs:     convertUserInputsToProto(b.UserInputs),
        EventStream:    convertEventsToProto(b.EventStream),
        CcSessionId:    b.CCSessionID,
        Status:         convertBlockStatusToProto(b.Status),
        Metadata:       b.Metadata,
        CreatedTs:      b.CreatedTs,
        UpdatedTs:      b.UpdatedTs,
    }

    if b.AssistantContent != "" {
        proto.AssistantContent = &b.AssistantContent
    }
    if b.AssistantTimestamp > 0 {
        proto.AssistantTimestamp = &b.AssistantTimestamp
    }
    if b.SessionStats != nil {
        proto.SessionStats = convertSessionStatsToProto(b.SessionStats)
    }

    return proto
}

func convertBlockTypeToProto(t store.AIBlockType) aiv1.BlockType {
    switch t {
    case store.AIBlockTypeMessage:
        return aiv1.BlockType_MESSAGE
    case store.AIBlockTypeContextSeparator:
        return aiv1.BlockType_CONTEXT_SEPARATOR
    default:
        return aiv1.BlockType_BLOCK_TYPE_UNSPECIFIED
    }
}

func convertBlockModeToProto(m store.AIBlockMode) aiv1.BlockMode {
    switch m {
    case store.AIBlockModeNormal:
        return aiv1.BlockMode_NORMAL
    case store.AIBlockModeGeek:
        return aiv1.BlockMode_GEEK
    case store.AIBlockModeEvolution:
        return aiv1.BlockMode_EVOLUTION
    default:
        return aiv1.BlockMode_BLOCK_MODE_UNSPECIFIED
    }
}

func convertBlockStatusToProto(s store.AIBlockStatus) aiv1.BlockStatus {
    switch s {
    case store.AIBlockStatusPending:
        return aiv1.BlockStatus_PENDING
    case store.AIBlockStatusStreaming:
        return aiv1.BlockStatus_STREAMING
    case store.AIBlockStatusCompleted:
        return aiv1.BlockStatus_COMPLETED
    case store.AIBlockStatusError:
        return aiv1.BlockStatus_ERROR
    default:
        return aiv1.BlockStatus_BLOCK_STATUS_UNSPECIFIED
    }
}

// ... reverse conversion functions ...
```

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `proto/api/v1/ai_service.proto` - 扩展 Block 消息和 RPC
- [ ] `server/router/api/v1/ai/block_handler.go` - Block API 处理器
- [ ] `server/router/api/v1/ai/handler.go` - 扩展主处理器
- [ ] `gen/api/v1/ai_service.pb.go` - 自动生成的代码

### 5.2 数据库变更

无（Phase 1 已完成）

### 5.3 配置变更

无

### 5.4 文档更新

- [ ] `docs/dev-guides/BACKEND_DB.md` - 更新 API 文档

---

## 6. 测试验收

### 6.1 功能测试

| 场景 | 输入 | 预期输出 |
|:---|:---|:---|
| **ListBlocks** | conversation_id=1 | 返回该会话的所有 Blocks |
| **ListBlocks with status filter** | conversation_id=1, status=completed | 只返回已完成的 Blocks |
| **GetBlock** | id=123 | 返回指定的 Block |
| **GetBlock not found** | id=999 | 返回 404 错误 |
| **AppendUserInput** | id=123, content="补充" | UserInputs 数组增加 1 |
| **UpdateBlockStatus** | id=123, status=completed | Status 更新为 completed |

### 6.2 性能验收

| 指标 | 目标值 | 测试方法 |
|:---|:---|:---|
| ListBlocks 延迟 | < 100ms (100 blocks) | 压测工具 |
| GetBlock 延迟 | < 50ms | 压测工具 |
| AppendUserInput 延迟 | < 50ms | 压测工具 |

### 6.3 集成验收

- [ ] Proto 生成成功（make generate）
- [ ] 与 Phase 1 Store 层集成成功
- [ ] Postman/HTTP 客户端测试通过

---

## 7. ROI 分析

| 维度 | 值 |
|:---|:---|
| 开发投入 | 3人天 |
| 预期收益 | 前端可一次性获取完整 Block 数据 |
| 风险评估 | 低（纯新增，不破坏现有 API） |
| 回报周期 | 1 Sprint |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| Proto 生成失败 | 低 | 中 | 确保 buf 工具版本正确 |
| API 兼容性问题 | 低 | 中 | 新增独立 RPC，不影响现有 |

---

## 9. 实施计划

### 9.1 时间表

| 阶段 | 时间 | 任务 |
|:---|:---|:---|
| **Day 1** | 1人天 | 编写 Proto 定义 |
| **Day 2** | 1人天 | 实现 Block Handler |
| **Day 3** | 1人天 | 单元测试，集成测试 |

### 9.2 检查点

- [ ] Checkpoint 1: Proto 生成成功
- [ ] Checkpoint 2: 单元测试通过
- [ ] Checkpoint 3: 集成测试通过

---

## 附录

### A. 参考资料

- [Phase 1 Spec](./unified-block-model-phase1.md)
- [Connect RPC 文档](https://connectrpc.com/)

### B. 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|:---|:---|:---|:---|
| 2026-02-04 | v1.0 | 初始版本 | Claude |
