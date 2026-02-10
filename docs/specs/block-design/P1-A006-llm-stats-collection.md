# P1-A006: LLM 层统计收集与普通模式 Session Summary 增强 (修订版)

> **状态**: 🔲 待开发
> **优先级**: P1 (重要)
> **投入**: 3人天
> **负责团队**: 团队 A (AI Core)
> **Sprint**: Sprint 未定
> **关联 Issue**: [#79](https://github.com/hrygo/divinesense/issues/79)

---

## 1. 目标与背景

### 1.1 核心目标

将 AI 会话统计收集逻辑下沉到 LLM 层，为普通模式（MemoParrot/ScheduleParrot/AmazingParrot）提供完整的 Session Summary，包括 Token 使用量、时间分解等统计数据。

### 1.2 当前问题

| 模式               | Session Summary 完整度 | 问题                                      |
| :----------------- | :--------------------- | :---------------------------------------- |
| **Geek/Evolution** | ✅ 完整                 | 通过 CC Runner 获取详细统计               |
| **Normal**         | ❌ 不完整               | 仅显示基础 duration，缺少 token/tool 统计 |

**根本原因**：
- LLM 调用层已产生 `resp.Usage` 数据（Token 统计），但未返回给 Agent
- Agent 层无法获取 LLM 统计，导致 `SessionStatsProvider` 无法实现

### 1.3 用户价值

- 普通模式用户可查看完整的 AI 调用统计（Token 使用、工具调用、时间分解）
- 与 Geek/Evolution 模式体验一致
- 帮助用户理解 AI 资源消耗（成本追踪）

### 1.4 技术价值

- **架构分层清晰**：LLM 层负责 LLM 统计，Agent 层负责组合
- **并发安全**：采用无状态 (Stateless) 设计，适应单例 LLMService 架构
- **易于扩展**：新增统计项只需修改 LLM 层返回结构

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] **[unified-block-model](./unified-block-model.md)**: `ai_block` 表已包含 `session_stats` 字段
- [x] **[unified-block-model_improvement](./unified-block-model_improvement.md)**: 确保时间戳标准 (Milliseconds) 统一 (P0)
- [x] **[cc_runner_async_arch](../../archived/specs/20260207_archive/cc_runner_async_arch.md)**: SessionStats 结构已定义

### 2.2 并行依赖

- [ ] **前端 SessionSummaryPanel 改进**: 确保普通模式正确显示统计

### 2.3 后续依赖

- 无

---

## 3. 功能设计

### 3.1 架构图 (修订)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Parrot Agents                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │ MemoParrot  │  │ScheduleParrot│  │AmazingParrot│  │BaseParrot  │    │
│  │ (Stateful)  │  │ (Stateful)  │  │ (Stateful)  │  │ (Stateful)  │    │
│  │ implements  │  │  implements  │  │  implements  │  │  implements │    │
│  │SessionStats │  │ SessionStats │  │ SessionStats │  │ SessionStats│    │
│  │   Provider  │  │   Provider   │  │   Provider   │  │   Provider  │    │
│  └──────┬──────┘  └──────┬───────┘  └──────┬───────┘  └──────┬──────┘    │
│         │                │                  │                  │           │
│         │ Call(msg)      │                  │                  │           │
│         │ <- stats       │                  │                  │           │
│         ▼                ▼                  ▼                  ▼           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  LLM Service Interface (Stateless)                              │    │
│  │    - Chat() (string, *LLMCallStats, error)                       │    │
│  │    - ChatStream() (<-chan string, <-chan *Stats, <-chan error)  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│         │                                                               │
│         ▼                                                               │
├─────────────────────────────────────────────────────────────────────────┤
│                           go-openai Library                              │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  ChatCompletionResponse { Usage: ... }                          │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 核心流程

#### 3.2.1 非流式调用 (Chat)

```
1. Agent 调用 llm.Chat(messages)
2. LLM 层记录 StartTime
3. 调用 go-openai CreateChatCompletion
4. 从 resp.Usage 提取 Token 统计
5. 计算 Duration = EndTime - StartTime
6. 返回 (content, stats, nil)
7. Agent 接收 stats 并累加到本地 sessionStats
```

#### 3.2.2 流式调用 (ChatStream)

```
1. Agent 调用 llm.ChatStream(messages)
2. LLM 层返回 contentChan, statsChan, errChan
3. Agent 启动 goroutine 消费 contentChan 处理内容
4. Agent 同时监听 statsChan (通常只会有1个数据，流结束时发送)
5. LLM 层内部:
   a. 记录 FirstChunkTime
   b. 收到最后一个 chunk (含 Usage) 时，构造 stats
   c. 发送 stats 到 statsChan
   d. 关闭 channels
6. Agent 接收到 stats 后累加到本地 sessionStats
```

#### 3.2.3 Agent 组合统计

```go
func (p *BaseParrot) GetSessionStats() *SessionStats {
    // 聚合本地 cumulative stats
    return &SessionStats{
        InputTokens:          p.sessionStats.InputTokens,
        OutputTokens:         p.sessionStats.OutputTokens,
        TotalTokens:          p.sessionStats.TotalTokens,
        ThinkingDurationMs:   p.sessionStats.ThinkingDurationMs,
        GenerationDurationMs: p.sessionStats.GenerationDurationMs,
        ToolCallCount:        p.toolCallCount,
        // ...
    }
}
```

### 3.3 关键决策

| 决策点         | 方案 A (Stateful Service) | 方案 B (Stateless Service) |   选择    | 理由                                                  |
| :------------- | :------------------------ | :------------------------- | :-------: | :---------------------------------------------------- |
| **服务状态**   | Service 持有 `stats`      | Service 返回 `stats`       |   **B**   | **并发安全**。LLMService 是单例，不能持有请求级状态。 |
| **流式返回值** | `GetStats()`              | `<-chan *Stats`            |   **B**   | 配合无状态设计，通过 Channel 异步返回统计元数据。     |
| **聚合责任**   | LLM 层                    | Agent 层                   | **Agent** | LLM 层只负责单次调用的统计，Agent 负责会话级聚合。    |

---

## 4. 技术实现

### 4.1 接口定义

```go
// ai/llm.go

// LLMCallStats 表示单次 LLM 调用的统计数据 (Immutable Data)
type LLMCallStats struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    
    // 时间统计 (毫秒) - 必须与 unified-block-model_improvement 规范保持一致 (int64 ms)
    ThinkingDurationMs   int64  // 首字延迟
    GenerationDurationMs int64  // 生成时长
    TotalDurationMs      int64  // 总时长
}

// LLMService LLM 服务接口（扩展）
type LLMService interface {
    // Chat 执行非流式聊天，直接返回统计
    Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error)

    // ChatStream 执行流式聊天
    // 增加 statsChan 用于返回统计信息（在流结束时）
    ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error)
}
```

```go
// ai/agent/base_parrot.go (新建)

// BaseParrot 提供基础的 Parrot 实现，包含统计聚合逻辑
type BaseParrot struct {
    llm           ai.LLMService
    accumulatedStats *ai.LLMCallStats // 累加的统计
    toolCallCount int
    toolsUsed     []string
    lock          sync.Mutex
}

// trackLLMCall 累加单次调用统计
func (p *BaseParrot) trackLLMCall(stats *ai.LLMCallStats) {
    p.lock.Lock()
    defer p.lock.Unlock()
    
    if p.accumulatedStats == nil {
        p.accumulatedStats = &ai.LLMCallStats{}
    }
    
    p.accumulatedStats.PromptTokens += stats.PromptTokens
    p.accumulatedStats.CompletionTokens += stats.CompletionTokens
    p.accumulatedStats.TotalTokens += stats.TotalTokens
    
    // 时间统计根据场景可能需要不同的聚合策略
    // 简单起见，累加 TotalDuration
    p.accumulatedStats.TotalDurationMs += stats.TotalDurationMs
    
    // Thinking/Generation 通常取"主要回答"的那一次，或者也累加
    // 策略：如果是 ReAct 中间步骤，计入 Thinking? 
    // 简化策略：全部累加到 Total，Thinking 仅取最后一次回复的
    p.accumulatedStats.GenerationDurationMs += stats.GenerationDurationMs
}
```

### 4.2 数据模型

#### 4.2.1 LLM 层统计结构

```go
// ai/llm.go

// 实现中不再持有 stats 字段
type llmService struct {
    client      *openai.Client
    model       string
    maxTokens   int
    temperature float32
}
```

#### 4.2.2 Agent 层组合结构

```go
type BaseParrot struct {
    // ...
    llmStats *ai.LLMCallStats // 当前会话累计
}
```

### 4.3 关键代码路径

| 文件路径                             | 职责                                | 修改类型 |
| :----------------------------------- | :---------------------------------- | :------- |
| `ai/llm.go`                          | 重构接口，返回 `LLMCallStats`       | 🔧 重构   |
| `ai/agent/base_parrot.go`            | 实现统计聚合逻辑                    | ➕ 新建   |
| `ai/agent/memo_parrot.go`            | 适配新接口，手动调用 `trackLLMCall` | 🔧 修改   |
| `ai/agent/schedule_parrot_v2.go`     | 适配新接口                          | 🔧 修改   |
| `server/router/api/v1/ai/factory.go` | 无需修改 (LLMService 保持单例)      | ✅ 无修改 |

---

## 5. 交付物清单

### 5.1 代码文件

- [ ] `ai/llm.go` - 更新接口签名，实现无状态统计返回
- [ ] `ai/agent/base_parrot.go` - 新建基础 Parrot，处理统计累加
- [ ] `ai/agent/memo_parrot.go` - 更新 Chat/ChatStream 调用处
- [ ] `ai/agent/*_parrot.go` - 更新其他 Parrot
- [ ] `ai/llm_test.go` - 单元测试：验证 statsChan 正确返回

### 5.2 文档更新

- [ ] `../../dev-guides/ARCHITECTURE.md` - 更新 LLM 层说明

---

## 6. 测试验收

### 6.1 单元测试

```go
func TestLLMService_ChatStream_Stats(t *testing.T) {
    // Mock OpenAI server returning Usage in last chunk
    // verify statsChan receives correct token counts
}
```

### 6.2 并发测试

*   启动 10 个 goroutine 并发调用同一个 `llmService` 实例。
*   验证每个调用返回的 `stats` 互不干扰，且准确。

---

## 7. ROI 分析

同原版，开发投入略有增加（由于接口重构涉及面稍广），但长期架构稳定性收益巨大。

---

## 8. 风险与缓解

| 风险               | 概率  | 影响  | 缓解措施                                                                                                              |
| :----------------- | :---: | :---: | :-------------------------------------------------------------------------------------------------------------------- |
| **接口破坏性变更** |  高   |  高   | 涉及所有调用 `Chat` 的地方。需通过编译器检查确保所有调用点都已更新。                                                  |
| **Usage 数据丢失** |  低   |  中   | 目前仅 standard library `ChatCompletion` 支持 usage，DeepSeek 等部分 provider 流式 usage 格式可能不同，需兼容性测试。 |

---

## 9. 实施计划

### 9.1 阶段划分

1.  **Phase 1: 接口重构** (Day 1)
    - 修改 `LLMService` 接口。
    - 修复所有编译错误（因前面修改签名导致）。
    - 实现 `llmService` 的内部统计构造逻辑。

2.  **Phase 2: Agent 适配** (Day 1.5)
    - 创建 `BaseParrot`。
    - 让各个 Parrot 继承/组合 `BaseParrot` 并接入统计。

3.  **Phase 3: 自测与验收** (Day 2)
    - 运行单测。
    - 手动验证普通模式 UI 显示。

## 附录

### A. 流式 Usage 补充

OpenAI 官方文档说明，流式请求中设置 `stream_options: {"include_usage": true}` 才会返回 Usage。
**注意**：`go-openai` 库已封装此逻辑，但需要在请求构造时显式开启（如果库版本较新）。若库版本较旧，可能需要升级。
需检查 `go-openai` 版本及构建参数。

```go
req.StreamOptions = &openai.StreamOptions{
    IncludeUsage: true,
}
```
这也需要加入到 `llm.go` 的实现中。
