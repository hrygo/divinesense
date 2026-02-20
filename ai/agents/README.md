# AI Agent 智能体系统 (`ai/agents`)

`agents` 包实现了 DivineSense 的核心智能体系统，我们将其形象地称为"鹦鹉 (Parrot)"系统。

## 概览

DivineSense 使用"鹦鹉"作为 AI Agent 的隐喻——每一只鹦鹉都有独特的性格、能力和使命。系统支持两种类型的鹦鹉：

1.  **配置驱动型 (UniversalParrot)**: 通过 YAML 定义，无需写代码。
2.  **代码实现型 (GeekParrot/EvolutionParrot)**: 通过 Go 代码实现复杂逻辑。

## 架构设计

    class ParrotAgent {
        <<interface>>
        +Execute(ctx, input, history, callback)
        +Name() string
        +SelfDescribe() ParrotSelfCognition
        +GetSessionStats() NormalSessionStats
    }
    class UniversalParrot {
        -config ParrotConfig
        -strategy ExecutionStrategy
        -baseParrot BaseParrot
        +Execute()
    }
    class GeekParrot {
        -runner CCRunner (Injected Singleton)
        -mode GeekMode
        +Execute()
    }
    class EvolutionParrot {
        -runner CCRunner (Injected Singleton)
        -mode EvolutionMode
        +Execute()
    }
    class CCRunner {
        <<Global Singleton>>
        +Execute()
        +Close(Graceful Shutdown)
    }
    class Orchestrator {
        -decomposer Decomposer
        -executor Executor
        -aggregator Aggregator
        +Process()
    }
    class ChatRouter {
        +Route(input) ParrotAgent
    }

    ParrotAgent <|.. UniversalParrot
    ParrotAgent <|.. GeekParrot
    ParrotAgent <|.. EvolutionParrot
    ChatRouter ..> ParrotAgent : 路由至
    ChatRouter ..> Orchestrator : 低置信度时
    GeekParrot --> CCRunner: Hot-Multiplexing
    EvolutionParrot --> CCRunner: Hot-Multiplexing
```

## 目录结构

```
agents/
├── base_parrot.go          # 基础鹦鹉实现（统计功能）
├── chat_router.go           # 聊天路由器（路由到合适的鹦鹉）
├── intent_classifier.go     # 意图分类器
├── context.go               # Agent 上下文管理
├── cache.go                # 缓存机制
├── recovery.go             # 错误恢复
├── cc_runner.go            # 统一 Claude Code 执行器
├── universal/               # 通用鹦鹉系统（策略、工厂）
├── geek/                   # 极客鹦鹉与进化鹦鹉（代码执行 & 自我进化）
├── orchestrator/            # 多智能体编排（DAG、交接）
├── runner/                 # 统一底层执行引擎（Hot-Multiplexing、隔离、进程树强杀Graceful Shutdown）
├── tools/                  # 具体工具实现
├── events/                 # 事件定义
├── registry/               # 工具、Prompt、指标注册中心
└── *test.go                # 测试文件
```

## 鹦鹉列表 (Parrot Roster)

| 鹦鹉                | 中文名 | 角色     | 性格           | 实现方式                                |
| :------------------ | :----- | :------- | :------------- | :-------------------------------------- |
| **MemoParrot**      | 灰灰   | 笔记助手 | 好奇, 严谨     | Universal (Config)                      |
| **ScheduleParrot**  | 时巧   | 日程管家 | 有条理, 高效   | Universal (Config)                      |
| **GeneralParrot**   | 通才   | 通用助手 | 平衡, 乐于助人 | Universal (Config)                      |
| **GeekParrot**      | 极客   | 代码专家 | 技术流, 精准   | Node.js CLI (CCRunner Hot-Multiplexing) |
| **EvolutionParrot** | 进化   | 自我进化 | 分析型, 审慎   | Node.js CLI (CCRunner Hot-Multiplexing) |

**注意**: 
1. AmazingParrot 已被 Orchestrator 替代。路由层 LLM 已移除，低置信度请求直接转 Orchestrator 进行任务分解。
2. GeekParrot 和 EvolutionParrot 自身属于瞬态组装体，背后统一挂载于全局长生命周期的 **CCRunner** 引擎。这确保了无论用户发送多少请求，只要对准相同的 UUID，一律零冷启动被热加载。

## 路由机制

系统采用多级路由策略：

1.  **Layer 1 - 规则匹配**: 关键词、正则表达式匹配
2.  **Layer 2 - 语义匹配**: 基于 embedding 的向量相似度（使用 SemanticExamples）
3.  **Layer 3 - Orchestrator**: 低置信度/多意图时由 Orchestrator 任务分解

## 扩展指南

### 1. 创建新工具

实现 `ToolWithSchema` 接口并在 `registry` 中注册。

```go
type MyTool struct {}
func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "功能描述" }
func (t *MyTool) Parameters() string { return JSONSchema }
func (t *MyTool) Execute(ctx, args) (string, error) { ... }
```

### 2. 定义新鹦鹉

在 `config/parrots/` 下创建新的 YAML 配置文件：

```yaml
id: "MY_PARROT"
name: "my_parrot"
chinese_name: "我的鹦鹉"
strategy: "react"  # direct / react / planning / reflexion
tools:
  - my_tool
system_prompt: "你是..."
```

### 3. 注册工具到全局注册表

```go
// 在 init() 函数中注册
func init() {
    registry.RegisterWithMetadata("my_tool", &MyTool{}, registry.ToolMetadata{
        Category: registry.CategoryMemo,
        Tags:     []string{"semantic"},
    })
}
```
