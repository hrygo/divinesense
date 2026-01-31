# SPEC-001: 后端基础设施与 BaseParrot

> **状态**: 待实现
> **优先级**: P0
> **依赖**: 无
> **负责人**: 后端开发组

## 1. 概述

本规范定义了鹦鹉助手家族的共享基础设施，旨在消除代码重复，统一错误处理，并提供健壮的 ReAct (Reasoning + Acting) 循环。核心组件包括 `BaseParrot` 基类、`ParrotRouter` 路由器增强以及统一的类型系统。

## 2. 核心组件设计

### 2.1 BaseParrot 基类

**目标**: 封装所有 Agent 共用的逻辑，包括 LLM 调用、工具解析、重试机制和回调处理。

**接口定义 (`plugin/ai/agent/base_parrot.go`)**:

```go
type BaseParrot struct {
    llm          ai.LLMService
    userID       int32
    tools        map[string]*BaseTool
    failureCount map[string]int
    // ...
}

// 核心方法：通用 ReAct 循环
func (b *BaseParrot) ExecuteReActLoop(
    ctx context.Context,
    systemPrompt string,
    userInput string,
    callback func(event string, data string),
) (string, error)
```

**关键逻辑**:
1.  **Thinking 事件**: 每次 LLM 调用前发送 `thinking` 事件。
2.  **工具解析**:
    *   优先尝试 JSON 解析。
    *   回退到正则匹配 `TOOL:\s*(\w+)\s+INPUT:\s*(\{.*\})`。
    *   支持 JSON 修复 (处理 LLM 返回的非标准 JSON)。
3.  **失败重试**:
    *   单一工具连续失败 3 次以上，终止并报错。
    *   自动将错误信息反馈给 LLM 进行自我修正。
4.  **超时控制**: 整个 Loop 默认超时 2 分钟。

### 2.2 ParrotRouter 增强

**目标**: 支持所有 4 种 Agent 的注册与动态路由。

**变更 (`plugin/ai/agent/parrot_router.go`)**:
1.  **注册表**: 维护 `map[AgentType]ParrotAgent`。
2.  **AutoRoute 逻辑优化**:
    *   `schedule`: 关键词 "日程", "会议", "安排", "提醒"。
    *   `memo`: 关键词 "笔记", "搜索", "找", "总结"。
    *   `creative`: 关键词 "创意", "使得", "名字", "想法"。
    *   `amazing`: 关键词组合 (e.g., "笔记" + "日程") 或显式 `@amazing`。

### 2.3 统一类型系统

**变更 (`plugin/ai/agent/types.go`)**:
确保与 `proto/api/v1/chat.proto` 保持一致：

```go
const (
    AgentTypeDefault  = "default"
    AgentTypeMemo     = "memo"
    AgentTypeSchedule = "schedule"
    AgentTypeAmazing  = "amazing"
    AgentTypeCreative = "creative"
)

// 复用 Proto 中的 Enum
func ToProtoAgentType(t string) apiv1.AgentType
```

## 3. 验收标准 (Acceptance Criteria)

### AC-001.1: BaseParrot 正确性
- [ ] **单元测试**: `base_parrot_test.go` 覆盖率 > 80%。
- [ ] **ReAct 循环**: 能正确处理 "思考 -> 工具调用 -> 工具结果 -> 最终回答" 的完整流程。
- [ ] **JSON 修复**: 能正确解析 LLM 返回的带 Markdown 代码块的 JSON (e.g., \`\`\`json {...} \`\`\`)。
- [ ] **错误恢复**: 模拟工具执行报错后，Agent 能收到错误信息并尝试第二次调用或向用户报错。

### AC-001.2: 路由逻辑
- [ ] **显式路由**: 指定 `agent_type="schedule"` 必须路由到 `ScheduleParrot`。
- [ ] **自动路由**: 输入 "明天开会" 自动识别为 Schedule；输入 "查找笔记" 自动识别为 Memo。

### AC-001.3: 性能与稳定性
- [ ] **并发**: Router 能并发处理 50+ 请求无 Race Condition。
- [ ] **超时**: 单次请求超过 2 分钟强制返回 Timeout 错误，不挂起 Goroutine。

## 4. 实施步骤

1.  创建 `plugin/ai/agent/base_parrot.go` 实现 ReAct 循环。
2.  创建 `plugin/ai/agent/base_parrot_test.go`。
3.  更新 `types.go` 补充常量。
4.  更新 `parrot_router.go` 注册逻辑。
