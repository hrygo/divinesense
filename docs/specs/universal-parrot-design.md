# UniversalParrot 统一架构设计文档

> **版本**: v1.0 | **日期**: 2026-02-08 | **状态**: ⚠️ 已废弃
>
> **⚠️ 重要说明**: 自 v0.99.0 起，AmazingParrot 已被 **Orchestrator-Workers 架构** 替代。
> 本文档保留作为历史参考，新开发请参考:
> - [Orchestrator-Workers 架构文档](../architecture/orchestrator-workers.md)
> - [Orchestrator 调研报告](../research/orchestrator-workers-research.md)
>
> **目标**: 三鹦鹉统一架构，差异仅在 Prompt 和工具链
>
> **关联**: [Issue #125](https://github.com/hrygo/divinesense/issues/125) | [Agent 设计调研](../research/agent-design-patterns-2026.md)（如存在）

---

## 1. 执行摘要

### 问题陈述

当前三只鹦鹉（MemoParrot、ScheduleParrotV2、AmazingParrot）存在以下问题：

- **~70% 代码重复**：ReAct 循环、缓存、统计、事件处理逻辑高度重复
- **架构不统一**：ScheduleParrotV2 使用 Native Tool Calling，其他使用 ReAct
- **扩展困难**：新增能力需要修改多处代码
- **测试分散**：没有统一的测试框架

### 解决方案

创建 **UniversalParrot** 统一基类，三只鹦鹉仅在以下方面有差异：

1. **Prompt**（人格、行为模式）
2. **工具集**（可用的工具列表）
3. **执行策略**（ReAct vs 快速通道 vs 并发）

### 预期收益

| 指标         | 改进                      |
| :----------- | :------------------------ |
| 代码重复率   | 从 ~70% → <10%            |
| 新增鹦鹉成本 | 从 ~500行 → ~50行（配置） |
| 测试覆盖     | 统一测试框架，覆盖率 >80% |
| 性能         | 快速通道优化简单 CRUD     |

---

## 2. 架构设计

### 2.1 组件图

```
┌─────────────────────────────────────────────────────────────────────┐
│                        UniversalParrot                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  统一 ReAct 循环引擎                                          │   │
│  │  - LLM 调用与解析                                            │   │
│  │  - 工具调用执行                                              │   │
│  │  - 会话管理                                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  快速通道 (Fast Path)                                       │   │
│  │  - 跳过 ReAct 循环                                            │   │
│  │  - 直接 Native Tool Calling                                │   │
│  │  - 用于简单 CRUD 操作                                        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  并发协调器 (Concurrent Orchestrator)                       │   │
│  │  - 多工具并行执行                                            │   │
│  │  - 错误隔离                                                  │   │
│  │  - 结果聚合                                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  统一服务层                                                  │   │
│  │  - LRU 缓存                                                 │   │
│  │  - SessionStats (BaseParrot)                              │   │
│  │  - 错误处理                                                 │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼─────┐    ┌─────▼──────┐    ┌────▼─────┐
   │MemoParrot│    │ScheduleParrot│   │AmazingParrot│
   │  配置    │    │   V2 配置   │    │  配置    │
   └──────────┘    └─────────────┘    └──────────┘
```

### 2.2 工具集管理（简化版：选项 C）

**工具集定义**：
```go
// ToolSetType 预定义工具集类型
type ToolSetType string

const (
    ToolSetMemo     ToolSetType = "memo_search"    // 单一工具：memo_search
    ToolSetSchedule ToolSetType = "schedule_full"  // 完整 CRUD 工具集
    ToolSetAmazing  ToolSetType = "combined"       // 全部工具
)

// ToolSetRegistry 工具集注册表
var ToolSetRegistry = map[ToolSetType][]string{
    ToolSetMemo:     {"memo_search"},
    ToolSetSchedule: {"schedule_query", "schedule_add", "schedule_update", "find_free_time"},
    ToolSetAmazing:  {"memo_search", "schedule_query", "schedule_add", "find_free_time", "schedule_update"},
}
```

### 2.3 快速通道策略（混合策略：选项 C）

**触发条件**：
1. 工具集支持快速通道（`supportsFastPath`）
2. 用户输入简单（`isSimpleQuery`）

**实现**：
```go
// supportsFastPath 判断工具集是否支持快速通道
func (t *ToolSetType) supportsFastPath() bool {
    switch t {
    case ToolSetMemo:
        return true // 单一工具查询
    case ToolSetSchedule:
        return true // CRUD 操作
    case ToolSetAmazing:
        return false // 需要协调，不支持快速通道
    default:
        return false
    }
}

// isSimpleQuery 判断是否为简单查询
func isSimpleQuery(input string) bool {
    // 无复杂结构、单意图、无歧义
    return len(input) < 200 &&
           !strings.Contains(input, "和") &&
           !strings.Contains(input, "然后") &&
           !strings.Contains(input, "同时")
}
```

### 2.4 统一错误处理

```go
// UniversalParrotError 统一错误包装
type UniversalParrotError struct {
    AgentName  string
    ToolName    string
    Operation   string
    Underlying   error
    IsRecoverable bool
    RetryAfter  time.Duration
}

func (e *UniversalParloidError) Error() string {
    return fmt.Sprintf("parrot %s: %s failed: %v", e.AgentName, e.Operation, e.Underlying)
}

func (e *UniversalParrotError) Unwrap() error {
    return e.Underlying
}
```

---

## 3. 数据结构设计

### 3.1 核心配置

```go
// ParrotConfig 鹦鹉配置
type ParrotConfig struct {
    // 基础配置
    Name           string              // "memo", "schedule", "amazing"
    AgentType      string              // 用于统计
    ToolSet        ToolSetType         // 工具集类型

    // LLM 配置
    LLM            ai.LLMService       // LLM 服务
    ModelName      string              // 模型名称（用于成本计算）

    // Prompt 配置
    PromptKey      string              // PromptRegistry 键
    PromptArgs     []any               // Prompt 模板参数

    // 行为配置
    SupportsFastPath   bool           // 是否支持快速通道
    SupportsConcurrent bool           // 是否支持并发执行
    MaxIterations      int              // 最大 ReAct 迭代次数

    // 依赖服务（按需注入）
    Retriever      *retrieval.AdaptiveRetriever
    ScheduleSvc    schedule.Service
    UserID        int32
    Timezone       string

    // 回调（可选）
    OnToolExecuted func(toolName string, result string)
    OnError       func(err error)
}

// FastPathConfig 快速通道配置
type FastPathConfig struct {
    Enabled        bool
    ToolName       string
    SimplifyInput  func(string) string  // 输入简化函数
}

// ConcurrentConfig 并发配置
type ConcurrentConfig struct {
    Enabled           bool
    MaxConcurrency    int
    Timeout          time.Duration
    ErrorStrategy    ErrorStrategy // "continue" | "abort" | "isolate"
}

type ErrorStrategy string

const (
    ErrorStrategyContinue ErrorStrategy = "continue" // 部分失败继续
    ErrorStrategyAbort    ErrorStrategy = "abort"    // 任一失败停止
    ErrorStrategyIsolate  ErrorStrategy = "isolate"  // 错误隔离
)
```

---

## 4. 实现计划

### Phase 1: UniversalParrot 基类

**文件**: `ai/agent/universal_parrot.go`

**核心方法**：

| 方法                                                 | 说明                 |
| :--------------------------------------------------- | :------------------- |
| `NewUniversalParrot(config)`                         | 构造函数，配置初始化 |
| `ExecuteWithCallback(ctx, input, history, callback)` | 主入口               |
| `executeReActLoop()`                                 | ReAct 循环执行       |
| `executeFastPath()`                                  | 快速通道执行         |
| `executeConcurrent()`                                | 并发工具执行         |
| `buildSystemPrompt()`                                | 构建 Prompt          |
| `parseToolCall()`                                    | 解析工具调用         |
| `invokeTool()`                                       | 调用工具             |
| `cacheGet/ cacheSet`                                 | 缓存管理             |

### Phase 2: 工具集注册表

**文件**: `ai/agent/toolset_registry.go`

```go
// ToolSetType 工具集类型
type ToolSetType string

const (
    ToolSetMemo     ToolSetType = "memo_search"
    ToolSetSchedule ToolSetType = "schedule_full"
    ToolSetAmazing  ToolSetType = "combined"
)

// ToolSetRegistry 工具集注册表
var ToolSetRegistry = map[ToolSetType]ToolSetConfig{
    ToolSetMemo: {
        Name: "memo_search",
        Tools: []string{"memo_search"},
        SupportsFastPath: true,
        SupportsConcurrent: false,
    },
    ToolSetSchedule: {
        Name: "schedule_full",
        Tools: []string{"schedule_query", "schedule_add", "schedule_update", "find_free_time"},
        SupportsFastPath: true,
        SupportsConcurrent: false,
    },
    ToolSetAmazing: {
        Name: "combined",
        Tools: []string{"memo_search", "schedule_query", "schedule_add", "find_free_time", "schedule_update"},
        SupportsFastPath: false,
        SupportsConcurrent: true,
    },
}

// ToolSetConfig 工具集配置
type ToolSetConfig struct {
    Name                 string
    Tools                []string
    SupportsFastPath     bool
    SupportsConcurrent   bool
    FastPathTool         string   // 快速通道默认工具
    ConcurrentTimeout    time.Duration
}
```

### Phase 3: 三鹦鹉重构

| 文件 | 变更类型 |
| :--- |::--------|
| `ai/agent/memo_parrot.go` | 重写为 UniversalParrot 配置 |
| `ai/agent/schedule_parrot_v2.go` | 重写为 UniversalParrot 配置 |
| `ai/agent/amazing_parrot.go` | 重写为 UniversalParrot 配置 |

**重构后结构**：
```go
// MemoParrot 配置化
func NewMemoParrot(...) (*ParrotAgent, error) {
    config := &ParrotConfig{
        Name:       "memo",
        AgentType:  "MEMO",
        ToolSet:     ToolSetMemo,
        LLM:         llm,
        Retriever:   retriever,
        PromptKey:    "memo",
        SupportsFastPath: true,
    }
    return NewParrotAgent(config)
}

// ScheduleParrotV2 配置化
func NewScheduleParrotV2(...) (*ParrotAgent, error) {
    config := &ParrotConfig{
        Name:       "schedule",
        AgentType:  "SCHEDULE",
        ToolSet:     ToolSetSchedule,
        LLM:         llm,
        ScheduleSvc: scheduleService,
        PromptKey:    "schedule",
        SupportsFastPath: true,
    }
    return NewParrotAgent(config)
}

// AmazingParrot 配置化
func NewAmazingParrot(...) (*ParrotAgent, error) {
    config := &ParrotConfig{
        Name:       "amazing",
        AgentType:  "AMAZING",
        ToolSet:     ToolSetAmazing,
        LLM:         llm,
        Retriever:   retriever,
        ScheduleSvc: scheduleService,
        PromptKey:    "amazing",
        SupportsFastPath: false,
        SupportsConcurrent: true,
    }
    return NewParrotAgent(config)
}
```

### Phase 4: 测试迁移

**新建文件**: `ai/agent/universal_parrot_test.go`

```go
// 表格驱动测试
func TestUniversalParrot_ReActLoop(t *testing.T) {
    tests := []struct {
        name       string
        toolSet    ToolSetType
        input      string
        expectTool string
    }{
        {"memo search", ToolSetMemo, "找笔记", "memo_search"},
        {"schedule add", ToolSetSchedule, "明天开会", "schedule_add"},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

**保留旧测试**：`*_test.go` 保留作为回归验证

### Phase 5: 调用方更新

| 文件                      | 变更         |
| :------------------------ | :----------- |
| `ai/agent/chat_router.go` | 更新创建逻辑 |
| `ai/agent/geek_parrot.go` | 如需更新     |
| 测试文件                  | 同步更新     |

---

## 5. 关键实现细节

### 5.1 ReAct 循环统一实现

```go
func (p *UniversalParrot) executeReActLoop(
    ctx context.Context,
    config *ParrotConfig,
    userInput string,
    history []string,
    callback EventCallback,
) error {
    // 1. 构建 messages
    messages := p.buildMessages(config, userInput, history)

    // 2. ReAct 迭代
    for iteration := 0; iteration < config.MaxIterations; iteration++ {
        // LLM 调用
        response, stats, err := config.LLM.Chat(ctx, messages)
        if err != nil {
            return NewUniversalParrotError(config.Name, "llm_chat", err)
        }

        // 解析工具调用
        toolName, toolInput, err := p.parseToolCall(response)
        if err != nil {
            // 无工具调用，最终答案
            return p.finalAnswer(response, callback)
        }

        // 执行工具
        result, err := p.invokeTool(ctx, config, toolName, toolInput, callback)
        if err != nil {
            // 错误处理：是否可恢复？
            if p.isRecoverable(err) {
                // 添加错误信息到 conversation，继续
            } else {
                return err
            }
        }

        // 添加到 conversation
        messages = append(messages,
            ai.Message{Role: "assistant", Content: response},
            ai.Message{Role: "user", Content: fmt.Sprintf("工具结果: %s", result)},
        )
    }

    return fmt.Errorf("exceeded max iterations")
}
```

### 5.2 快速通道实现

```go
func (p *UniversalParrot) executeFastPath(
    ctx context.Context,
    config *ParrotConfig,
    userInput string,
    callback EventCallback,
) error {
    toolSet := ToolSetRegistry[config.ToolSet]

    // 确定工具
    toolName := toolSet.FastPathTool
    if toolName == "" {
        toolName = toolSet.Tools[0]
    }

    // 简化输入
    simplifiedInput := userInput
    if config.FastPathInputSimplifier != nil {
        simplifiedInput = config.FastPathInputSimplifier(userInput)
    }

    // 直接调用工具
    result, err := p.invokeToolDirect(ctx, config, toolName, simplifiedInput, callback)
    if err != nil {
        return err
    }

    // 发送结果
    if callback != nil {
        callback(EventTypeAnswer, result)
    }

    return nil
}
```

### 5.3 并发协调器实现

```go
func (p *UniversalParrot) executeConcurrent(
    ctx context.Context,
    config *ParrotConfig,
    toolCalls []ToolCall,
    callback EventCallback,
) (map[string]string, error) {
    toolSet := ToolSetRegistry[config.ToolSet]

    results := make(map[string]string)
    var wg sync.WaitGroup
    var mu sync.Mutex
    var errorCount int32

    for _, tc := range toolCalls {
        wg.Add(1)
        go func(toolCall ToolCall) {
            defer wg.Done()

            // 发送工具使用事件
            if callback != nil {
                callback(EventTypeToolUse, toolCall.Name)
            }

            // 执行工具
            result, err := p.invokeToolDirect(ctx, config, toolCall.Name, toolCall.Input, nil)

            mu.Lock()
            defer mu.Unlock()

            if err != nil {
                atomic.AddInt32(&errorCount, 1)
                results[toolCall.Name+"_error"] = err.Error()
                // 发送错误事件
                if callback != nil {
                    callback(EventTypeError, err.Error())
                }
            } else {
                results[toolCall.Name] = result
                // 发送工具结果事件
                if callback != nil {
                    callback(EventTypeToolResult, result)
                }
            }
        }(tc)
    }

    // 等待完成或超时
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // 全部完成
    case <-ctx.Done():
        // 超时
        return nil, ctx.Err()
    case <-time.After(toolSet.ConcurrentTimeout):
        // 硬超时
        return results, fmt.Errorf("concurrent execution timeout")
    }

    return results, nil
}
```

---

## 6. Prompt 集成

### 6.1 PromptRegistry 扩展

**现有**: `ai/agent/prompts.go` 已有 PromptRegistry

**扩展**：添加 UniversalParrot prompt 支持

```go
// PromptRegistry 扩展
var PromptRegistry = struct {
    Memo     *AgentPrompts
    Schedule *AgentPrompts
    Amazing  *AgentPrompts
    Universal *AgentPrompts  // 新增
    mu       sync.RWMutex
}{
    // ...
}
```

### 6.2 Prompt 模板结构

```go
// AgentPrompts 结构（已有）
type AgentPrompts struct {
    System    *PromptConfig  // 主系统 prompt
    Planning  *PromptConfig  // 规划 prompt（Amazing）
    Synthesis *PromptConfig  // 综合 prompt（Amazing）
}

// 新增: FastPath prompt（可选）
type AgentPrompts struct {
    System    *PromptConfig
    Planning  *PromptConfig
    Synthesis *PromptConfig
    FastPath  *PromptConfig  // 快速通道专用 prompt
}
```

---

## 7. 向后兼容策略（一次性大重构：选项 1）

### 7.1 接口保持

```go
// ParrotAgent 接口（已有）
type ParrotAgent interface {
    Name() string
    ExecuteWithCallback(ctx context.Context, userInput string, history []string, callback EventCallback) error
    SelfDescribe() *ParrotSelfCognition
}

// UniversalParrot 实现 ParrotAgent
type ParrotAgent struct {
    config *ParrotConfig
    // ...
}

// 旧方法保持兼容
func (p *MemoParrot) Name() string { return "memo" }
func (p *MemoParrot) ExecuteWithCallback(...) { /* 配置化调用 */ }
```

### 7.2 迁移路径

```
┌─────────────────────────────────────────────────────────────┐
│ Step 1: 创建 UniversalParrot（新文件）                      │
│ ai/agent/universal_parrot.go                                │
│ ai/agent/toolset_registry.go                                 │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 2: 重写三鹦鹉（配置化）                                 │
│ ai/agent/memo_parrot.go       → ~50 行配置                │
│ ai/agent/schedule_parrot_v2.go → ~50 行配置                │
│ ai/agent/amazing_parrot.go    → ~50 行配置                │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 3: 更新调用方                                           │
│ ai/agent/chat_router.go                                   │
│ ai/agent/geek_parrot.go (如需)                            │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 4: 测试迁移                                               │
│ ai/agent/universal_parrot_test.go (新)                     │
│ 保留旧测试作为回归验证                                        │
└─────────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ Step 5: 清理死代码                                             │
│ 删除旧的 ReAct 循环代码                                     │
│ 删除重复的工具管理逻辑                                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
| :--- |::-----|:---------|
| **性能下降** | 高 | 快速通道优化简单 CRUD；基准测试验证 |
| **回归 Bug** | 高 | 保留旧测试作为回归验证；灰度发布 |
| **复杂度转移** | 中 | 工具集注册表简化配置；文档完善 |
| **测试覆盖不足** | 中 | 统一测试框架；TDD 补充 |

---

## 9. 验收标准

- [ ] 三只鹦鹉功能与重构前完全一致
- [ ] 代码重复率 < 10%
- [ ] 所有单元测试通过
- [ ] 集成测试通过
- [ ] 性能无明显下降（±5%）
- [ ] 文档更新完成（architecture/overview.md, dev-guides/frontend/overview.md）
- [ ] 代码审查通过

---

## 10. 参考资料

- [Agent 设计模式调研报告 2026](../research/agent-design-patterns-2026.md)
- [后端与数据库指南](../dev-guides/backend/database.md)
- [Issue #125 - 统一三鹦鹉架构重构](https://github.com/hrygo/divinesense/issues/125)

---

*文档版本: v1.0 | 作者: Claude Opus 4.6 | 状态: 待审核*
