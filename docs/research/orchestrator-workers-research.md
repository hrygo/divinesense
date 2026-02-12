# Orchestrator-Workers 多 Agent 架构调研报告

> 调研时间: 2026-02-12 | 关联 Issue: #169

---

## 1. 调研背景

### 1.1 当前系统问题

DivineSense 普通模式采用 ChatRouter → 单一 Agent 的模式：

```
用户输入 → ChatRouter → 单一 Agent → 响应
```

**痛点**：
1. 用户请求涉及多领域时，只能路由到单一 Agent
2. Agent 间缺乏协作机制，无法并行处理
3. 架构扩展性受限，新增 Agent 需要修改路由规则
4. AmazingParrot 职责模糊，与 Coordinator 重叠

### 1.2 调研目标

1. 调研业界 AI Agent 设计模式最佳实践
2. 分析如何优化协调三个 Agent（Memo、Schedule、Amazing）
3. 设计符合当前系统定位的架构方案

---

## 2. 业界最佳实践调研

### 2.1 Anthropic 多代理系统

**来源**: [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)

**核心模式**: Orchestrator-Worker

```
Lead Agent (Orchestrator)
    ├── 分析任务
    ├── 分解任务
    ├── 创建 Subagents (Workers)
    └── 聚合结果
```

**关键发现**：
- 多代理系统相比单代理 **性能提升 90.2%**
- Token 使用量解释 80% 的性能差异
- 并行执行可减少 **90%** 的研究时间

**8 大原则**：

| 原则 | 说明 |
|:-----|:-----|
| 教授协调器如何委托 | 详细任务描述，避免重复工作 |
| 根据复杂度调整努力 | 简单任务 1 agent，复杂任务 10+ subagents |
| 工具设计至关重要 | 工具描述决定 40% 效率差异 |
| 从宽到窄搜索 | 先探索全貌，再深入细节 |
| 并行工具调用 | 同时调用 3-5 个 subagents |
| 引导思考过程 | Extended thinking 提升推理质量 |
| 让 Agent 自我改进 | Agent 可优化自己的 prompt |
| 实现有效评估 | LLM-as-judge + 人类评估 |

### 2.2 Anthropic Building Effective AI Agents

**来源**: [Building Effective AI Agents](https://www.anthropic.com/engineering/building-effective-agents)

**核心观点**：
> "最成功的实现并不是使用复杂的框架或专门的库。**相反，它们使用简单、可组合的模式构建。**"

**Workflow vs Agent 区分**：

| 类型 | 定义 | 适用场景 |
|:-----|:-----|:---------|
| **Workflow** | LLM + 工具通过预定义代码路径编排 | 可预测的任务 |
| **Agent** | LLM 动态指导自己的过程和工具使用 | 开放式问题 |

**五种 Workflow 模式**：

| 模式 | 说明 | 适用场景 |
|:-----|:-----|:---------|
| Prompt Chaining | 顺序步骤 | 可分解为固定子任务 |
| Routing | 分类 + 路由 | 不同类别需要不同处理 |
| Parallelization | 并行子任务 | 独立子任务可并行 |
| **Orchestrator-Workers** | LLM 动态分解 + 委托 + 聚合 | 无法预测子任务的复杂任务 |
| Evaluator-Optimizer | 生成 + 评估循环 | 需要迭代优化 |

**Orchestrator-Workers 模式关键**：
- 适用于无法预测子任务的复杂任务
- 子任务不是预定义的，而是由 orchestrator 根据具体输入确定
- **灵活性**是与 Parallelization 的关键区别

### 2.3 Microsoft Azure 五大编排模式

**来源**: [AI Agent Orchestration Patterns](https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns)

| 模式 | 适用场景 |
|:-----|:---------|
| Sequential | 固定顺序流水线 |
| Concurrent | 并行多视角分析 |
| Group Chat | 协作讨论/决策 |
| Handoff | 动态任务转移 |
| Magentic | 复杂开放任务（任务账本） |

### 2.4 Google Cloud 设计模式

**来源**: [Choose a design pattern for your agentic AI system](https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system)

**多代理模式对比**：

| 模式 | 复杂度 | 延迟 | 适用场景 |
|:-----|:-------|:-----|:---------|
| Sequential | 低 | 低 | 固定流程 |
| Parallel | 低 | 最低 | 独立任务 |
| Coordinator | 中 | 中 | 动态路由 |
| Hierarchical | 高 | 高 | 复杂分解 |
| Swarm | 最高 | 最高 | 协作辩论 |

---

## 3. 当前系统分析

### 3.1 现有三 Agent 架构

| Agent | 角色 | 策略 | 工具 |
|:------|:-----|:-----|:-----|
| MemoParrot (灰灰) | 笔记搜索 | ReAct | memo_search |
| ScheduleParrot (时巧) | 日程管理 | ReAct | schedule_add/query/update, find_free_time |
| AmazingParrot (折衷) | 综合助理 | Planning | 全部工具 |

### 3.2 问题诊断

| 问题 | 说明 |
|:-----|:-----|
| Agent 间缺乏协调 | 路由到单一 Agent，无跨 Agent 协作 |
| Handoff 能力缺失 | 一旦路由确定，无法动态转移 |
| 并行能力有限 | AmazingParrot 的 Planning 策略并行调度能力有限 |
| 职责重叠 | AmazingParrot 与 Coordinator 职责高度重叠 |

### 3.3 GeekParrot / EvolutionParrot 分析

这两个 Agent 比较特殊：
- 不使用内部 LLM，直接调用外部 Claude Code CLI
- 是 **External Executor**（外部执行器）而非 **Domain Expert**（领域专家）
- 由用户显式选择模式触发，不由 Supervisor 自动路由

**结论**：不适合纳入 Expert Agent 体系，保持独立。

---

## 4. 方案设计

### 4.1 选定模式

**Orchestrator-Workers Workflow**

理由：
1. 符合 Anthropic 最佳实践
2. 适用于无法预测子任务的复杂场景
3. LLM 动态分解，自动适应新 Agent
4. 保持透明性，展示规划步骤

### 4.2 架构设计

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Normal Mode                                  │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Orchestrator Agent                          │  │
│  │                    (LLM 驱动任务分解)                          │  │
│  │                                                                │  │
│  │   Step 1: 分析 + 分解 (LLM)                                    │  │
│  │   Step 2: 显示规划 (透明性)                                    │  │
│  │   Step 3: 执行任务 (Workers)                                   │  │
│  │   Step 4: 聚合结果 (如需要)                                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                   ↑                                  │
│                        ┌─────────┴─────────┐                        │
│                        │  Expert Registry  │                        │
│                        │  (ParrotFactory)  │                        │
│                        │                   │                        │
│                        │  config/parrots/  │                        │
│                        │  ├── memo.yaml    │                        │
│                        │  └── schedule.yaml│                        │
│                        └───────────────────┘                        │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.3 核心组件

| 组件 | 职责 |
|:-----|:-----|
| **Orchestrator** | LLM 驱动的任务分解、调度、聚合 |
| **Expert Registry** | 复用 ParrotFactory，config/ 配置化扩展 |
| **Task Plan** | 结构化任务计划，支持透明性展示 |

### 4.4 任务计划结构

```json
{
  "analysis": "用户想了解今天的日程安排和相关的笔记内容",
  "tasks": [
    {"agent": "schedule", "input": "今天有什么安排？", "purpose": "查询今日日程"},
    {"agent": "memo", "input": "今天相关的笔记", "purpose": "搜索今日笔记"}
  ],
  "parallel": true,
  "aggregate": true
}
```

### 4.5 处理流程

1. **分析 + 分解**：Orchestrator 使用 LLM 分析用户输入，动态分解为子任务
2. **显示规划**：向用户展示任务计划（透明性）
3. **执行任务**：并行或顺序执行子任务
4. **聚合结果**：多结果时 LLM 聚合为统一响应

### 4.6 Agent 分类

| 类型 | 成员 | 触发方式 |
|:-----|:-----|:---------|
| **Expert Agents** | Memo, Schedule, Future... | Orchestrator 自动调度 |
| **External Executors** | Geek, Evolution | 用户显式选择模式 |

---

## 5. 实现计划

### 5.1 后端变更

| 文件 | 变更类型 | 说明 |
|:-----|:---------|:-----|
| `ai/agents/orchestrator/orchestrator.go` | **新增** | Orchestrator 主逻辑 |
| `ai/agents/orchestrator/decomposer.go` | **新增** | LLM 任务分解 |
| `ai/agents/orchestrator/planner.go` | **新增** | 规划展示 |
| `ai/agents/orchestrator/executor.go` | **新增** | 任务执行 |
| `ai/agents/orchestrator/aggregator.go` | **新增** | 结果聚合 |
| `ai/agents/chat_router.go` | **重构** | 调用 Orchestrator |
| `config/parrots/amazing.yaml` | **删除** | 不再需要 |

### 5.2 前端变更

| 文件 | 变更类型 | 说明 |
|:-----|:---------|:-----|
| `web/src/hooks/useOrchestratorEvents.ts` | **新增** | 事件处理 |
| `web/src/components/PlanDisplay.tsx` | **新增** | 规划展示组件 |

### 5.3 新增事件类型

```go
const (
    EventTypePlan      = "plan"       // 任务规划
    EventTypeTaskStart = "task_start" // 任务开始
    EventTypeTaskEnd   = "task_end"   // 任务完成
)
```

---

## 6. 参考

### 6.1 官方文档

- [Anthropic: Building Effective AI Agents](https://www.anthropic.com/engineering/building-effective-agents)
- [Anthropic: How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Microsoft Azure: AI Agent Orchestration Patterns](https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns)
- [Google Cloud: Choose a design pattern for your agentic AI system](https://docs.cloud.google.com/architecture/choose-design-pattern-agentic-ai-system)

### 6.2 框架参考

- [LangGraph: Multi-Agent Workflows](https://blog.langchain.com/langgraph-multi-agent-workflows/)
- [OpenAI Swarm - GitHub](https://github.com/openai/swarm)

---

## 7. 结论

推荐采用 **Orchestrator-Workers Workflow** 模式：

1. ✅ 符合 Anthropic 最佳实践
2. ✅ LLM 动态分解任务，非硬编码规则
3. ✅ 自动适应新 Expert Agent
4. ✅ 保持透明性，展示规划步骤
5. ✅ 支持并行执行，降低延迟
6. ✅ 架构简洁，易于维护和扩展

---

*本报告由 Idea Researcher Skill 生成*
