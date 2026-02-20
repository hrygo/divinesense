# Agent 开发快速开始指南

> **保鲜状态**: ✅ 已验证 (2026-02-20) | **版本**: v1.0.0
> **状态**: ✅ 完成 (v1.0.0) | **投入**: 3人天

## 概述

本指南介绍如何为 DivineSense 创建新的 AI Agent（Parrot），包括配置驱动开发、工具注册和测试方法。

DivineSense 的 Agent 系统基于 **UniversalParrot** 配置驱动架构，允许通过 YAML 配置文件创建新 Agent，无需编写代码。

---

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                     UniversalParrot 架构                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ParrotFactory                                            │  │
│  │  - 从 YAML 配置创建 Agent                                  │  │
│  │  - 从 ToolRegistry 解析工具                                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           │                                    │
│                           ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  UniversalParrot                                          │  │
│  │  - 配置驱动的通用 Agent 实现                               │  │
│  │  - 支持三种执行策略                                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           │                                    │
│                           ▼                                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ExecutionStrategy (执行策略)                             │  │
│  │  ├── ReActExecutor: 思考-行动循环                          │  │
│  │  ├── DirectExecutor: 原生工具调用                          │  │
│  │  └── PlanningExecutor: 两阶段规划 + 并发                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  ToolRegistry (工具注册表)                                │  │
│  │  - 全局工具注册                                           │  │
│  │  - 动态工具发现                                           │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 创建新 Agent

### 方式一：配置驱动（推荐）

通过创建 YAML 配置文件，无需编写代码即可定义新 Agent。

#### 2.1 创建配置文件

在 `config/parrots/` 目录下创建新配置文件：

```yaml
# config/parrots/my_agent.yaml
name: my_agent
display_name: My Custom Agent
emoji: "\U0001F5A5"  # 工具图标

# 执行策略
strategy: react  # react | direct | planning
max_iterations: 10

# 可用工具
tools:
  - memo_search
  - schedule_query

# 系统提示词
system_prompt: |
  You are a helpful assistant for managing notes and schedules.
  Be concise and helpful in your responses.

# UI 提示建议
prompt_hints:
  - "查找笔记"
  - "查看日程"

# 缓存配置
enable_cache: true
cache_ttl: 5m
cache_size: 100

# 自我描述
self_description:
  title: My Custom Agent
  name: my_agent
  emoji: "\U0001F5A5"
  capabilities:
    - memo_search
    - schedule_query
```

#### 2.2 配置字段说明

| 字段             | 类型         | 必填 | 说明                                  |
| :--------------- | :----------- | :--- | :------------------------------------ |
| `name`           | string       | ✅    | Agent 唯一标识符                      |
| `display_name`   | string       | ✅    | 显示名称                              |
| `emoji`          | string       | ❌    | 表情符号标识                          |
| `strategy`       | StrategyType | ✅    | 执行策略：`react`/`direct`/`planning` |
| `max_iterations` | int          | ❌    | 最大迭代次数（默认 10）               |
| `tools`          | []string     | ✅    | 可用工具列表                          |
| `system_prompt`  | string       | ✅    | 系统提示词                            |
| `prompt_hints`   | []string     | ❌    | UI 建议                               |
| `enable_cache`   | bool         | ❌    | 启用缓存                              |
| `cache_ttl`      | duration     | ❌    | 缓存过期时间                          |
| `cache_size`     | int          | ❌    | 缓存大小                              |

#### 2.3 执行策略选择

| 策略                 | 适用场景         | 特点                                                |
| :------------------- | :--------------- | :-------------------------------------------------- |
| **Self-Consistency** | 不确定推理任务   | 多路径采样投票，提升准确率，计算成本倍增            |
| **Direct Calling**   | 简单任务 (<3步)  | 原生 Function Calling，快速、低成本，不适合复杂任务 |
| **Hot-Multiplexing** | 长连接持久化会话 | Stdin/Stdout 流复用，零冷启动延迟，内存常驻         |
| **planning**         | 多工具协作       | 两阶段规划 + 并发执行                               |

**推荐选择**：
- 简单查询任务 → `direct`
- 需要推理的任务 → `react`
- 多工具协作 → `planning`

### 方式二：代码实现

对于需要复杂逻辑的 Agent，可以通过代码实现：

```go
// ai/agents/my_agent.go
package agent

import (
    "context"
    "github.com/hrygo/divinesense/ai"
)

type MyCustomAgent struct {
    llm   ai.LLMService
    tools []ToolWithSchema
}

func NewMyCustomAgent(llm ai.LLMService, tools []ToolWithSchema) *MyCustomAgent {
    return &MyCustomAgent{
        llm:   llm,
        tools: tools,
    }
}

func (a *MyCustomAgent) Name() string {
    return "my_custom"
}

func (a *MyCustomAgent) ExecuteWithCallback(
    ctx context.Context,
    userInput string,
    history []string,
    callback EventCallback,
) error {
    // 💡 架构建议：对于复杂的外部 CLI 集成（如 Claude Code），
    // 推荐直接调用 `runner.CCRunner` 单例进行 Hot-Multiplexing 执行。
    // 详见：docs/architecture/cc-runner-architecture.md
    return nil
}
```

## 4. 高级：代码执行引擎 (CCRunner v2.0)

对于 `GeekParrot` 和 `EvolutionParrot` 等需要执行 OS 命令的代理，DivineSense 提供了 **CCRunner v2.0** 核心：

- **Hot-Multiplexing**: 自动管理 OS 进程生命周期，通过 Stdin/Stdout 流复用实现“零拉起延迟”。
- **PGID 隔离**: 确保代理产生的子进程能被优雅销毁，无孤儿进程。
- **UUID v5**: 确定的 SessionID 生成机制，支持跨请求会话恢复。

开发此类 Agent 时，通常在 `Handler` 层直接注入 `runner.CCRunner` 实例并调用 `Execute` 方法。

### 高级执行模式 (CCRunner v2.0)

针对代码执行类 Agent，DivineSense 引入了 **全双工热多路复用** 模式：

```
GeekParrot/EvolutionParrot
    │
    ▼
CCRunner (Global Singleton)
    │
    ├─ GetOrCreateSession(SessionID) ──▶ UUID v5 路由
    ├─ WriteInput(Stdin) ───────────▶ Hot-Multiplexing
    └─ Terminate(PGID) ─────────────▶ Graceful Shutdown
```

### 3.1 理解工具接口

所有工具必须实现 `ToolWithSchema` 接口：

```go
// ToolWithSchema 定义了带 JSON Schema 的工具
type ToolWithSchema interface {
    Tool
    Schema() map[string]interface{}
}

type Tool interface {
    Name() string
    Description() string
    Run(ctx context.Context, input string) (string, error)
}
```

### 3.2 创建新工具

在 `ai/agents/tools/` 目录下创建新工具：

```go
// ai/agents/tools/my_tool.go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
)

type MyTool struct {
    service MyService
    userIDGetter func(ctx context.Context) int32
}

func NewMyTool(service MyService, userIDGetter func(ctx context.Context) int32) *MyTool {
    return &MyTool{
        service:     service,
        userIDGetter: userIDGetter,
    }
}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return `Does something useful.

INPUT FORMAT:
{"param1": "value1", "param2": "value2"}

OUTPUT FORMAT:
Result description`
}

func (t *MyTool) InputType() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type":        "string",
                "description": "Parameter 1",
            },
            "param2": map[string]interface{}{
                "type":        "string",
                "description": "Parameter 2",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *MyTool) Run(ctx context.Context, inputJSON string) (string, error) {
    var input struct {
        Param1 string `json:"param1"`
        Param2 string `json:"param2,omitempty"`
    }

    if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
        return "", fmt.Errorf("invalid JSON: %w", err)
    }

    // 实现工具逻辑
    result := t.service.DoSomething(input.Param1, input.Param2)

    return fmt.Sprintf("Result: %s", result), nil
}
```

### 3.3 全局注册工具

在工具包的 `init()` 函数中注册：

```go
// ai/agents/tools/register.go
package tools

import (
    "github.com/hrygo/divinesense/ai/agents/registry"
)

func init() {
    // 注册工具到全局注册表
    registry.RegisterInCategory(
        registry.CategoryCustom,
        "my_tool",
        &MyTool{...},
    )
}
```

### 3.4 工具分类

使用 `ToolCategory` 对工具进行分组：

```go
const (
    CategoryMemo    ToolCategory = "memo"
    CategorySchedule ToolCategory = "schedule"
    CategorySearch  ToolCategory = "search"
    CategorySystem  ToolCategory = "system"
    CategoryAI      ToolCategory = "ai"
    CategoryCustom  ToolCategory = "custom"
)
```

---

## 4. 测试和调试

### 4.1 单元测试

创建测试文件 `ai/agents/tools/my_tool_test.go`：

```go
package tools_test

import (
    "context"
    "testing"

    "github.com/hrygo/divinesense/ai/agents/tools"
)

func TestMyTool_Run(t *testing.T) {
    // 创建 mock service
    mockService := &MockMyService{...}

    // 创建工具
    tool := tools.NewMyTool(
        mockService,
        func(ctx context.Context) int32 { return 1 },
    )

    // 测试正常输入
    result, err := tool.Run(context.Background(), `{"param1": "test"}`)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result == "" {
        t.Error("expected non-empty result")
    }
}

func TestMyTool_InvalidJSON(t *testing.T) {
    tool := tools.NewMyTool(nil, nil)

    _, err := tool.Run(context.Background(), "invalid json")
    if err == nil {
        t.Error("expected error for invalid JSON")
    }
}
```

### 4.2 集成测试

使用测试脚本验证 Agent 行为：

```bash
# 1. 启动服务
make start

# 2. 运行测试脚本
./scripts/test_agent.sh my_agent "测试消息"
```

### 4.3 调试日志

启用调试日志查看 Agent 执行过程：

```bash
# 设置日志级别
export LOG_LEVEL=debug

# 查看实时日志
make logs-follow-backend | grep "agent\|parrot"
```

### 4.4 验证清单

- [ ] 工具实现 `ToolWithSchema` 接口
- [ ] 工具已注册到全局注册表
- [ ] 配置文件格式正确
- [ ] 系统提示词清晰描述 Agent 行为
- [ ] 单元测试覆盖核心逻辑
- [ ] 集成测试验证端到端流程

---

## 5. 参考示例

### 5.1 MemoParrot 配置

```yaml
# config/parrots/memo.yaml
name: memo
display_name: Memo Parrot
emoji: "\U0001F4DD"

strategy: react
max_iterations: 10

tools:
  - memo_search

system_prompt: |
  You are a helpful assistant for searching and retrieving notes.
  You can search through the user's notes using semantic search.

  When the user asks about their notes or wants to find information:
  1. Use the memo_search tool with relevant keywords
  2. Present the results in a clear, organized way
  3. If no relevant notes are found, suggest alternative search terms

  Be concise and helpful in your responses.
```

### 5.2 ScheduleParrot 配置

```yaml
# config/parrots/schedule.yaml
name: schedule
display_name: Schedule Parrot
emoji: "\U0001F4C5"

strategy: react
max_iterations: 10

tools:
  - schedule_add
  - schedule_query
  - schedule_update
  - find_free_time

system_prompt: |
  You are a helpful assistant for managing schedules and calendars.

  You can help users:
  - Create new schedule entries
  - Query existing schedules
  - Update existing schedule entries
  - Find free time slots

  When creating schedules:
  - Always use ISO 8601 format for times (e.g., 2026-02-09T10:00:00+08:00)
  - Detect conflicts and warn the user
  - Confirm the schedule details after creation

  Be concise and helpful in your responses.
```

---

## 6. 常见问题

### Q: 工具找不到？

**A**: 确保工具已注册到全局注册表：

```go
registry.Register("my_tool", &MyTool{...})
```

### Q: 配置文件未生效？

**A**: 检查配置文件路径和格式：

```bash
# 验证配置文件
ls -la config/parrots/
cat config/parrots/my_agent.yaml

# 检查环境变量
echo $DIVINESENSE_PARROT_CONFIG_DIR
```

### Q: Agent 响应不符合预期？

**A**: 调整系统提示词：

1. 明确描述 Agent 的职责和限制
2. 指定输出格式
3. 添加示例对话

---

## 7. 相关文档

| 文档                                                        | 描述                     |
| :---------------------------------------------------------- | :----------------------- |
| [架构文档](../../architecture/overview.md)                  | Agent 系统完整架构       |
| [UniversalParrot 设计](../specs/universal-parrot-design.md) | UniversalParrot 详细设计 |
| [Agent 测试指南](testing.md)                                | 测试方法和验证清单       |
| [执行策略实现](../../../ai/agents/universal/executor.go)    | 策略接口和实现           |

---

*维护者*: DivineSense 开发团队
*反馈渠道*: [GitHub Issues](https://github.com/hrygo/divinesense/issues)
