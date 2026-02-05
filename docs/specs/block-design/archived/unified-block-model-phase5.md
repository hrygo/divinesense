# P1-A005: Unified Block Model - Phase 5 Chat Handler Integration

> **状态**: 🔲 待开发
> **优先级**: P1 (重要)
> **投入**: 4人天
> **Sprint**: Sprint 1
> **关联 Issue**: [#71](https://github.com/hrygo/divinesense/issues/71)
> **依赖**: Phase 2 (Proto & API), Phase 3 (Frontend Types)

---

## 1. 目标与背景

### 1.1 核心目标

改造后端 Chat Handler，使其能够正确处理 Block 生命周期，包括创建、更新、完成 Block。

### 1.2 用户价值

- **完整的对话记录**：所有对话内容都被正确保存
- **追加式输入**：用户可以在 AI 回复完成前追加输入

### 1.3 技术价值

- **数据一致性**：确保 Block 状态与对话进程同步
- **代码简化**：移除对 `ai_message` 表的直接操作

---

## 2. 依赖关系

### 2.1 前置依赖（必须完成）

- [x] **Phase 1**: 数据库表和 Store 接口已定义
- [x] **Phase 2**: Proto 和 API 已定义

### 2.2 并行依赖（可同步进行）

- [ ] **P1-A004**: 前端组件改造

### 2.3 后续依赖（依赖本 Spec）

- [ ] **P1-A006**: 集成测试

---

## 3. 功能设计

### 3.1 Block 生命周期

```
┌─────────────────────────────────────────────────────────────────┐
│  Block 生命周期                                                 │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐   │
│  │ Pending │ → │Streaming │ → │Completed│ → │  Error   │   │
│  └─────────┘    └──────────┘    └──────────┘    └──────────┘   │
│       │              │               │               │           │
│       ▼              ▼               ▼               ▼           │
│  用户输入      事件流式写入      会话统计写入     错误处理      │
│  创建Block    event_stream      session_stats    metadata     │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

1. **接收用户输入**：Chat 接口收到用户消息
2. **判断 Block 状态**：
   - 如果最新 Block 状态为 `pending` 或 `streaming` → 追加输入
   - 否则 → 创建新 Block
3. **流式响应**：将事件写入 `event_stream`
4. **完成 Block**：AI 响应结束后，更新 `status` 为 `completed`

### 3.3 关键决策

| 决策点          | 方案 A                  | 方案 B       | 选择  | 理由               |
| :-------------- | :---------------------- | :----------- | :---: | :----------------- |
| **事件写入**    | 每个事件一次 DB 写入    | 批量写入     | **A** | 实时性优先         |
| **CC 会话映射** | 在 Block 创建时映射     | 在事件中映射 | **A** | 明确映射时机       |
| **向后兼容**    | 同时写 Block 和 Message | 只写 Block   | **B** | 简化代码，视图兼容 |

---

## 4. 技术实现

### 4.1 ChatHandler 改造

```go
// server/router/api/v1/ai/handler.go

package ai

import (
    "context"
    "fmt"

    "connectrpc.com/connect"
    "github.com/hrygo/divinesense/gen/api/v1/aiv1"
    "github.com/hrygo/divinesense/store"
)

type ChatHandler struct {
    db          *postgres.DB
    blockStore  store.AIBlockStore
    // ... other fields ...
}

func NewChatHandler(db *postgres.DB) *ChatHandler {
    return &ChatHandler{
        db:          db,
        blockStore:  db,
    }
}

// Chat handles streaming chat requests with Block support
func (h *ChatHandler) Chat(
    ctx context.Context,
    req *connect.Request[aiv1.ChatRequest],
    stream *connect.ServerStream[aiv1.ChatResponse],
) error {
    userID := getUserID(ctx)
    if userID == 0 {
        return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthorized"))
    }

    conversationID := req.Msg.ConversationId
    mode := determineMode(req) // "normal", "geek", or "evolution"

    // Step 1: Determine if we should append to existing block or create new one
    var block *store.AIBlock
    var err error

    if conversationID > 0 {
        // Check for existing pending/streaming block
        block, err = h.blockStore.GetLatestBlock(ctx, conversationID)
        if err == nil && block != nil && block.Status != store.AIBlockStatusCompleted {
            // Append to existing block
            h.blockStore.AppendUserInput(ctx, block.ID, store.UserInput{
                Content:   req.Msg.Message,
                Timestamp: time.Now().Unix(),
                Metadata:  nil,
            })
        } else {
            // Create new block
            block, err = h.createNewBlock(ctx, &CreateBlockParams{
                ConversationID: conversationID,
                Mode:          mode,
                UserContent:   req.Msg.Message,
                UserID:        userID,
            })
            if err != nil {
                return connect.NewError(connect.CodeInternal, err)
            }
        }
    } else {
        // New conversation - create first block
        // ... create conversation first ...
        block, err = h.createNewBlock(ctx, &CreateBlockParams{
            ConversationID: conversationID,
            Mode:          mode,
            UserContent:   req.Msg.Message,
            UserID:        userID,
        })
        if err != nil {
            return connect.NewError(connect.CodeInternal, err)
        }
    }

    // Step 2: Update block status to streaming
    h.blockStore.UpdateStatus(ctx, block.ID, store.AIBlockStatusStreaming)

    // Step 3: Send initial response with block info
    if err := stream.Send(&aiv1.ChatResponse{
        BlockId:     &block.ID,
        BlockUid:    &block.UID,
        BlockStatus: convertStatusToProto(store.AIBlockStatusStreaming),
    }); err != nil {
        return err
    }

    // Step 4: Route to appropriate parrot/agent
    switch mode {
    case BlockModeGeek:
        return h.handleGeekMode(ctx, req, stream, block)
    case BlockModeEvolution:
        return h.handleEvolutionMode(ctx, req, stream, block)
    default:
        return h.handleNormalMode(ctx, req, stream, block)
    }
}

// CreateBlockParams contains parameters for creating a new block
type CreateBlockParams struct {
    ConversationID int32
    Mode           BlockMode
    UserContent    string
    UserID         int32
    CCSessionID    string // For Geek/Evolution modes
}

func (h *ChatHandler) createNewBlock(
    ctx context.Context,
    params *CreateBlockParams,
) (*store.AIBlock, error) {
    blockMode := store.AIBlockModeNormal
    switch params.Mode {
    case BlockModeGeek:
        blockMode = store.AIBlockModeGeek
    case BlockModeEvolution:
        blockMode = store.AIBlockModeEvolution
    }

    create := &store.CreateAIBlock{
        ConversationID: params.ConversationID,
        BlockType:      store.AIBlockTypeMessage,
        Mode:           blockMode,
        UserInputs: []store.UserInput{
            {
                Content:   params.UserContent,
                Timestamp: time.Now().Unix(),
                Metadata:  nil,
            },
        },
        Status:    store.AIBlockStatusPending,
        CreatedTs: time.Now().Unix(),
        UpdatedTs: time.Now().Unix(),
    }

    // For Geek/Evolution modes, set CC session ID
    if params.CCSessionID != "" {
        create.CCSessionID = params.CCSessionID
    }

    return h.blockStore.CreateBlock(ctx, create)
}

// handleNormalMode handles normal AI chat mode
func (h *ChatHandler) handleNormalMode(
    ctx context.Context,
    req *connect.Request[aiv1.ChatRequest],
    stream *connect.ServerStream[aiv1.ChatResponse],
    block *store.AIBlock,
) error {
    // Initialize parrot agent
    parrot := h.parrotRegistry.Get(req.Msg.AgentType)

    // Stream response
    var eventStream []store.BlockEvent

    // Thinking phase
    if err := h.sendThinkingEvent(ctx, stream, block); err != nil {
        return err
    }

    // Tool calls phase
    tools, err := parrot.Execute(ctx, req.Msg.Message)
    if err != nil {
        h.blockStore.UpdateStatus(ctx, block.ID, store.AIBlockStatusError)
        return err
    }

    // Record tool events
    for _, tool := range tools {
        eventStream = append(eventStream, store.BlockEvent{
            Type:      "tool_use",
          Content:   tool.Name,
          Timestamp: time.Now().Unix(),
          Meta: map[string]any{
              "tool_name":     tool.Name,
              "input_summary": tool.Input,
          },
        })
    }

    // Answer phase
    answer := parrot.GenerateResponse(ctx, tools)

    // Update block with content
    update := &store.UpdateAIBlock{
        ID:               block.ID,
        AssistantContent: &answer,
        EventStream:      eventStream,
        Status:           store.AIBlockStatusCompleted,
        UpdatedTs:        ptr(int64(time.Now().Unix())),
    }

    _, err = h.blockStore.UpdateBlock(ctx, update)
    if err != nil {
        return connect.NewError(connect.CodeInternal, err)
    }

    // Send final response
    return stream.Send(&aiv1.ChatResponse{
        Content:     answer,
        Done:        true,
        BlockStatus: convertStatusToProto(store.AIBlockStatusCompleted),
    })
}

// handleGeekMode handles Geek mode (Claude Code CLI)
func (h *ChatHandler) handleGeekMode(
    ctx context.Context,
    req *connect.Request[aiv1.ChatRequest],
    stream *connect.ServerStream[aiv1.ChatResponse],
    block *store.AIBlock,
) error {
    ccRunner := h.ccRunnerFactory.New()

    // Generate CC session ID (UUID v5)
    ccSessionID := generateCCSessionID(block.ConversationID, block.ID)

    // Start CC session
    sessionEvents, err := ccRunner.Start(ctx, ccSessionID, req.Msg.Message)
    if err != nil {
        h.blockStore.UpdateStatus(ctx, block.ID, store.AIBlockStatusError)
        return err
    }

    var eventStream []store.BlockEvent

    // Stream CC events
    for event := range sessionEvents {
        // Convert CC event to Block event
        blockEvent := store.BlockEvent{
            Type:      event.Type,
            Content:   event.Content,
            Timestamp: event.Timestamp,
            Meta: map[string]any{
                "tool_name": event.ToolName,
                "duration":  event.Duration,
            },
        }
        eventStream = append(eventStream, blockEvent)

        // Send to client
        if err := stream.Send(&aiv1.ChatResponse{
            EventType:  event.Type,
            EventData:  event.Content,
            EventMeta:  event.Meta,
        }); err != nil {
            return err
        }
    }

    // Get session stats
    sessionStats := ccRunner.GetStats(ccSessionID)

    // Update block with session stats
    update := &store.UpdateAIBlock{
        ID:           block.ID,
        EventStream:  eventStream,
        SessionStats: sessionStats,
        Status:       store.AIBlockStatusCompleted,
        UpdatedTs:    ptr(int64(time.Now().Unix())),
    }

    _, err = h.blockStore.UpdateBlock(ctx, update)
    if err != nil {
        return connect.NewError(connect.CodeInternal, err)
    }

    return nil
}
```

### 4.2 事件写入器

```go
// server/router/api/v1/ai/event_writer.go

package ai

import (
    "context"
    "time"

    "github.com/hrygo/divinesense/store"
)

// EventWriter handles writing events to Block event stream
type EventWriter struct {
    blockStore store.AIBlockStore
    blockID    int64
    events     []store.BlockEvent
}

func NewEventWriter(blockStore store.AIBlockStore, blockID int64) *EventWriter {
    return &EventWriter{
        blockStore: blockStore,
        blockID:    blockID,
        events:     make([]store.BlockEvent, 0),
    }
}

// WriteThinking writes a thinking event
func (w *EventWriter) WriteThinking(content string) error {
    event := store.BlockEvent{
        Type:      "thinking",
        Content:   content,
        Timestamp: time.Now().Unix(),
    }
    w.events = append(w.events, event)
    return w.blockStore.AppendEvent(context.Background(), w.blockID, event)
}

// WriteToolUse writes a tool_use event
func (w *EventWriter) WriteToolUse(toolName, input string) error {
    event := store.BlockEvent{
        Type:      "tool_use",
        Content:   toolName,
        Timestamp: time.Now().Unix(),
        Meta: map[string]any{
            "tool_name":     toolName,
            "input_summary": input,
        },
    }
    w.events = append(w.events, event)
    return w.blockStore.AppendEvent(context.Background(), w.blockID, event)
}

// WriteToolResult writes a tool_result event
func (w *EventWriter) WriteToolResult(toolName, output string, duration int64) error {
    event := store.BlockEvent{
        Type:      "tool_result",
        Content:   output,
        Timestamp: time.Now().Unix(),
        Meta: map[string]any{
            "tool_name":     toolName,
            "duration_ms":   duration,
        },
    }
    w.events = append(w.events, event)
    return w.blockStore.AppendEvent(context.Background(), w.blockID, event)
}

// WriteAnswer writes an answer event (streaming)
func (w *EventWriter) WriteAnswer(content string) error {
    // For streaming answer, we accumulate content
    // This is handled separately in the main flow
    return nil
}

// Flush writes all accumulated events
func (w *EventWriter) Flush(ctx context.Context) error {
    // Batch write all events
    return nil
}
```

### 4.3 关键代码路径

| 文件路径                                       | 职责                         |
| :--------------------------------------------- | :--------------------------- |
| `server/router/api/v1/ai/handler.go`           | 主处理器，Block 生命周期管理 |
| `server/router/api/v1/ai/event_writer.go`      | 事件写入器（新增）           |
| `server/router/api/v1/ai/geek_handler.go`      | Geek 模式处理器（新增）      |
| `server/router/api/v1/ai/evolution_handler.go` | Evolution 模式处理器（新增） |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `server/router/api/v1/ai/handler.go` - 改造主处理器
- [ ] `server/router/api/v1/ai/event_writer.go` - 事件写入器（新增）
- [ ] `server/router/api/v1/ai/geek_handler.go` - Geek 模式处理器（新增）
- [ ] `server/router/api/v1/ai/evolution_handler.go` - Evolution 模式处理器（新增）

### 5.2 数据库变更

无（Phase 1 已完成）

### 5.3 配置变更

无

### 5.4 文档更新

- [ ] `docs/dev-guides/BACKEND_DB.md` - 更新 Chat Handler 说明

---

## 6. 测试验收

### 6.1 功能测试

| 场景           | 输入                    | 预期输出                            |
| :------------- | :---------------------- | :---------------------------------- |
| **创建 Block** | 用户发送第一条消息      | 新 Block 创建，status=pending       |
| **追加输入**   | 在 Block 完成前发送消息 | 追加到现有 Block                    |
| **流式响应**   | AI 回复中               | event_stream 实时更新               |
| **完成 Block** | AI 回复结束             | status=completed                    |
| **Geek 模式**  | Geek 模式请求           | cc_session_id 正确映射              |
| **错误处理**   | AI 返回错误             | status=error，metadata 包含错误信息 |

### 6.2 性能验收

| 指标            | 目标值 | 测试方法   |
| :-------------- | :----- | :--------- |
| 创建 Block 延迟 | < 20ms | 单线程压测 |
| 追加事件延迟    | < 10ms | 单线程压测 |
| 完成响应延迟    | < 50ms | 单线程压测 |

### 6.3 集成验收

- [ ] 与 Phase 1 Store 层集成成功
- [ ] 与 Phase 2 Proto 定义兼容
- [ ] 现有 Chat 功能不受影响

---

## 7. ROI 分析

| 维度     | 值                                     |
| :------- | :------------------------------------- |
| 开发投入 | 4人天                                  |
| 预期收益 | Block 生命周期正确管理，数据完整持久化 |
| 风险评估 | 中（涉及核心 Chat 逻辑）               |
| 回报周期 | 1 Sprint                               |

---

## 8. 风险与缓解

| 风险                | 概率  | 影响 | 缓解措施                 |
| :------------------ | :---: | :--- | :----------------------- |
| **向后兼容破坏**    |  中   | 高   | 保留兼容视图，渐进式迁移 |
| **性能下降**        |  低   | 中   | 批量写入优化             |
| **CC 会话映射错误** |  低   | 中   | UUID v5 确定性映射       |

---

## 9. 实施计划

### 9.1 时间表

| 阶段      | 时间  | 任务                         |
| :-------- | :---- | :--------------------------- |
| **Day 1** | 1人天 | Handler 改造，Block 创建逻辑 |
| **Day 2** | 1人天 | 事件写入器实现               |
| **Day 3** | 1人天 | Geek/Evolution 模式处理      |
| **Day 4** | 1人天 | 集成测试，问题修复           |

### 9.2 检查点

- [ ] Checkpoint 1: Block 创建/更新单元测试通过
- [ ] Checkpoint 2: 流式响应集成测试通过
- [ ] Checkpoint 3: 现有 Chat 功能回归测试通过

---

## 附录

### A. 参考资料

- [Phase 1 Spec](./unified-block-model-phase1.md)
- [Phase 2 Spec](./unified-block-model-phase2.md)
- [后端开发指南](../../dev-guides/BACKEND_DB.md)

### B. 变更记录

| 日期       | 版本 | 变更内容 | 作者   |
| :--------- | :--- | :------- | :----- |
| 2026-02-04 | v1.0 | 初始版本 | Claude |
