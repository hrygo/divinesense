# Memos 智能助理升级改进路径规划

> 基于 AI Agent 工程最新最佳实践，从 AI Agent 工程专家、AI 科学家、AI Native 产品经理三重视角规划升级路径。
>
> **注意**: 本文档为行业通用版参考，实际采用 [assistant-roadmap.md](./assistant-roadmap.md) 私人版方案。

**文档导航**: [主路线图](./00-master-roadmap.md) | [调研报告](./assistant-research.md) | [私人版路线图](./assistant-roadmap.md)

---

## 目录

1. [行业最佳实践调研](#1-行业最佳实践调研)
2. [现状差距分析](#2-现状差距分析)
3. [升级路径规划](#3-升级路径规划)
4. [详细实施方案](#4-详细实施方案)
5. [里程碑与验收标准](#5-里程碑与验收标准)

---

## 1. 行业最佳实践调研

### 1.1 Agent 编排模式演进 (2024-2025)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Agent 架构演进路线                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Simple RAG ─► ReAct ─► Plan-Execute ─► Multi-Agent ─► Agentic AI  │
│     (2023)     (2023)    (2024)          (2024)         (2025)     │
│                                                                     │
│  单次检索      思考-行动    计划-执行      多Agent协作    自主决策    │
│  无状态        循环迭代     预规划+执行    角色分工       动态规划    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### 1.1.1 OpenAI Agents SDK 编排模式

| 模式 | 特点 | 适用场景 |
|------|------|---------|
| **LLM 编排** | LLM 驱动决策、工具选择、Handoffs | 复杂开放任务 |
| **代码编排** | 确定性流程、结构化输出、并行执行 | 可预测工作流 |
| **Handoffs** | Agent 间委托、专业化分工 | 多领域协作 |
| **Guardrails** | 输入/输出验证、安全约束 | 生产环境必备 |

#### 1.1.2 LangGraph Plan-and-Execute 模式

```
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Planner   │────►│ Executor 1 │────►│ Executor 2 │──...──► Output
│ (规划全局) │     │  (执行步骤) │     │  (执行步骤) │
└────────────┘     └────────────┘     └────────────┘
      │                                      │
      └──────────── Re-plan ◄────────────────┘
                 (失败时重规划)
```

**vs ReAct**:
- ReAct: 每步都调用 LLM → 延迟累积
- Plan-Execute: 一次规划 + 轻量执行 → 更快、更可控

#### 1.1.3 Agentic Orchestration 六角色模型

| 角色 | 职责 | 边界 |
|------|------|------|
| **Router** | 意图识别、任务分发 | 不执行具体任务 |
| **Planner** | 任务分解、步骤规划 | 不直接调用工具 |
| **Knowledge** | RAG 检索、知识查询 | 只读操作 |
| **Tool Executor** | 工具调用、副作用操作 | 受限权限 |
| **Supervisor** | 进度监控、超时处理 | 全局视角 |
| **Critic** | 结果验证、质量评估 | 后置检查 |

### 1.2 Agentic RAG 最新进展

#### 1.2.1 Self-RAG (自反思检索增强生成)

```
┌─────────────────────────────────────────────────────────────────┐
│                      Self-RAG 工作流程                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Query ──► [Retrieve?] ──► Retrieve ──► Generate ──► [Critic]  │
│               │                              │           │      │
│               │ No                           │           │      │
│               ▼                              ▼           ▼      │
│          Direct Gen                    Output ◄── Good? ──► Retry│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**核心创新**:
- **Reflection Tokens**: 自动判断是否需要检索
- **Critique Tokens**: 评估生成质量
- **按需检索**: 避免无意义检索，提升效率

#### 1.2.2 RAG 架构模式对比

| 模式 | 检索时机 | 评估机制 | 适用场景 |
|------|---------|---------|---------|
| **Naive RAG** | 总是检索 | 无 | 简单问答 |
| **Advanced RAG** | Query 改写后检索 | Reranker | 复杂查询 |
| **Self-RAG** | 按需检索 | Reflection | 混合任务 |
| **Agentic RAG** | Agent 决策检索 | Multi-step | 复杂推理 |

### 1.3 Agent 记忆系统

#### 1.3.1 三层记忆架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    Agent Memory Architecture                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │  Episodic   │  │  Semantic   │  │      Procedural         │ │
│  │  情景记忆    │  │  语义记忆    │  │       程序记忆          │ │
│  ├─────────────┤  ├─────────────┤  ├─────────────────────────┤ │
│  │ "发生了什么"│  │ "我知道什么" │  │     "我如何做"          │ │
│  │ Vector DB   │  │ Knowledge   │  │  Cached Workflows       │ │
│  │ 对话历史    │  │ Graph/RAG   │  │  Fine-tuned Policies    │ │
│  │ 用户交互    │  │ 领域知识    │  │  RL-learned Routines    │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
│                                                                 │
│  应用: 个性化     应用: 专业知识    应用: 自动化流程           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 1.3.2 记忆实现最佳实践

| 记忆类型 | 存储方式 | 检索策略 | 生命周期 |
|---------|---------|---------|---------|
| **短期** | 上下文窗口 | 滑动窗口 | 会话内 |
| **情景** | Vector DB | 语义相似 | 跨会话 |
| **语义** | Knowledge Graph | 结构化查询 | 持久化 |
| **程序** | 配置/代码 | 规则匹配 | 版本化 |

### 1.4 Agent Guardrails 框架

#### 1.4.1 多层防护架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    Guardrails Architecture                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Pre-Execution Layer                    │  │
│  │  • Input Validation    • Access Control   • Rate Limit   │  │
│  └────────────────────────────┬─────────────────────────────┘  │
│                               ▼                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Execution Layer                        │  │
│  │  • Tool Permissions   • Timeout   • Resource Limits      │  │
│  └────────────────────────────┬─────────────────────────────┘  │
│                               ▼                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Post-Execution Layer                   │  │
│  │  • Output Validation  • PII Filter  • Hallucination Check│  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.5 AI Native UX 设计范式

#### 1.5.1 Generative UI 模式

| 模式 | 描述 | 实现 |
|------|------|------|
| **Tool-Invocation UI** | 工具调用触发专用 UI | 条件渲染组件 |
| **Streaming Feedback** | 实时状态反馈 | 流式传输 + 动画 |
| **Progressive Disclosure** | 渐进式信息展示 | 折叠/展开 |
| **Contextual Actions** | 上下文快捷操作 | 动态按钮 |
| **Confirmation Patterns** | 关键操作确认 | 卡片式确认 |

#### 1.5.2 MCP (Model Context Protocol)

> "像 USB-C 一样的 AI 应用标准接口"

```
┌─────────────┐         MCP         ┌─────────────┐
│   Client    │◄───────────────────►│   Server    │
│ (Claude/GPT)│                     │ (工具/数据)  │
└─────────────┘                     └─────────────┘
     │                                    │
     │  • Resources (数据访问)             │
     │  • Tools (功能调用)                 │
     │  • Prompts (提示模板)               │
     │                                    │
```

---

## 2. 现状差距分析

### 2.1 GAP 分析矩阵

| 维度 | 当前状态 | 行业最佳实践 | 差距 | 优先级 |
|------|---------|-------------|------|--------|
| **编排模式** | 单 Agent + 简单路由 | Multi-Agent + 动态编排 | 高 | P0 |
| **记忆系统** | 会话内上下文 | 三层记忆架构 | 高 | P0 |
| **RAG 能力** | 混合检索 + Reranker | Self-RAG + Agentic RAG | 中 | P1 |
| **Guardrails** | 基础超时/限流 | 多层防护框架 | 中 | P1 |
| **评估体系** | 无系统化评估 | Evals + 可观测性 | 高 | P1 |
| **Generative UI** | 基础流式反馈 | 完整 Tool-Invocation UI | 低 | P2 |

### 2.2 关键差距详解

#### 2.2.1 编排模式差距

**当前**:
```
ChatRouter (规则+LLM) → 单一 Agent 执行 → 返回结果
```

**目标**:
```
Router → Planner → [Executor1, Executor2, ...] → Critic → 返回结果
         ↑________________ Re-plan ___________________|
```

**具体问题**:
- 无任务分解能力，复杂任务处理困难
- Agent 间无协作，"惊奇"并发但无动态调度
- 无结果验证，输出质量不可控

#### 2.2.2 记忆系统差距

**当前**:
```go
// 仅支持会话内历史
history []string  // 简单字符串拼接
```

**目标**:
```go
type MemorySystem struct {
    ShortTerm  *ContextWindow     // 当前会话
    Episodic   *VectorMemory      // 历史交互 (跨会话)
    Semantic   *KnowledgeGraph    // 用户偏好/常用模板
    Procedural *WorkflowCache     // 学习到的流程
}
```

#### 2.2.3 评估体系差距

**当前**: 无系统化评估

**目标**:

| 评估维度 | 指标 | 目标值 |
|---------|------|--------|
| 准确性 | 任务完成率 | >90% |
| 延迟 | P95 响应时间 | <3s |
| 成本 | Token/请求 | <5000 |
| 安全 | Guardrail 拦截率 | 100% |
| 用户体验 | 满意度 | >4.5/5 |

---

## 3. 升级路径规划

### 3.1 三阶段演进路线图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         升级路径总览                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  Phase 1: 基础强化          Phase 2: 能力扩展        Phase 3: 智能进化  │
│  ─────────────────         ─────────────────        ─────────────────   │
│                                                                         │
│  • Guardrails 框架         • Plan-Execute 模式      • Self-RAG          │
│  • 记忆系统 V1             • Multi-Agent 协作       • 自主学习          │
│  • 评估体系                • Generative UI 完善     • 个性化适配        │
│  • 工具标准化              • MCP 集成               • 预测性交互        │
│                                                                         │
│  ────────────►             ────────────►            ────────────►       │
│    4-6 周                    6-8 周                   8-12 周           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Phase 1: 基础强化 (P0)

#### 目标
- 建立可靠的 Agent 基础设施
- 实现可观测性和评估能力
- 引入基础记忆系统

#### 关键交付

| 交付物 | 描述 | 验收标准 |
|--------|------|---------|
| Guardrails 框架 | 输入/执行/输出三层防护 | 100% 请求覆盖 |
| 记忆系统 V1 | 情景记忆 + 用户偏好 | 跨会话记忆可用 |
| 评估体系 | 准确性/延迟/成本指标 | Dashboard 可视化 |
| 工具标准化 | 统一 Tool Schema | 所有工具迁移 |

### 3.3 Phase 2: 能力扩展 (P1)

#### 目标
- 实现复杂任务处理能力
- 建立 Multi-Agent 协作机制
- 完善 AI Native UX

#### 关键交付

| 交付物 | 描述 | 验收标准 |
|--------|------|---------|
| Plan-Execute 模式 | Planner + Executor 分离 | 复杂任务成功率 >80% |
| Agent 协作框架 | Handoff + Supervisor | 多 Agent 任务可调度 |
| Generative UI 2.0 | 完整 Tool-Invocation UI | 覆盖所有工具 |
| MCP Server | 标准化工具接口 | 支持外部集成 |

### 3.4 Phase 3: 智能进化 (P2)

#### 目标
- 实现自主决策和学习能力
- 个性化用户体验
- 预测性交互

#### 关键交付

| 交付物 | 描述 | 验收标准 |
|--------|------|---------|
| Self-RAG | 按需检索 + 自我评估 | 检索效率提升 30% |
| 程序记忆 | 学习用户流程 | 常用模式自动识别 |
| 个性化 Agent | 用户偏好适配 | 个性化推荐准确率 >70% |
| 预测性 UI | 主动建议 | 用户采纳率 >50% |

---

## 4. 详细实施方案

### 4.1 Phase 1 详细设计

#### 4.1.1 Guardrails 框架

```go
// plugin/ai/guardrails/guardrails.go

type GuardrailsConfig struct {
    InputValidation  InputGuardrails
    ExecutionLimits  ExecutionGuardrails
    OutputFilters    OutputGuardrails
}

type InputGuardrails struct {
    MaxInputLength   int           // 最大输入长度
    ContentFilter    ContentFilter // 内容过滤器
    RateLimiter      RateLimiter   // 速率限制
    AccessControl    AccessControl // 访问控制
}

type ExecutionGuardrails struct {
    MaxIterations    int           // 最大迭代次数
    Timeout          time.Duration // 执行超时
    ToolPermissions  map[string]Permission // 工具权限
    ResourceLimits   ResourceLimits // 资源限制
}

type OutputGuardrails struct {
    PIIFilter        PIIFilter     // PII 过滤
    HallucinationCheck HallucinationChecker // 幻觉检测
    OutputValidation OutputValidator // 输出验证
}
```

**架构图**:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Guardrails Pipeline                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Input ──► [Input Guardrails] ──► Agent ──► [Output Guardrails] │
│                   │                 │              │            │
│                   ▼                 ▼              ▼            │
│             Validation         Execution      Validation        │
│             Rate Limit         Timeout        PII Filter        │
│             Access Ctrl        Resource       Hallucination     │
│                                                                 │
│             ┌─────────────────────────────────────┐             │
│             │         Supervisor (监控)           │             │
│             │  • Metrics Collection               │             │
│             │  • Anomaly Detection                │             │
│             │  • Alert & Logging                  │             │
│             └─────────────────────────────────────┘             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.1.2 记忆系统 V1

```go
// plugin/ai/memory/memory.go

type MemoryManager struct {
    shortTerm  *ShortTermMemory  // 会话内上下文
    episodic   *EpisodicMemory   // 情景记忆 (跨会话)
    semantic   *SemanticMemory   // 语义记忆 (用户偏好)
}

// 情景记忆: 存储历史交互
type EpisodicMemory struct {
    store      *store.Store
    vectorDB   *pgvector.Client
    maxEntries int
    ttl        time.Duration
}

type Episode struct {
    ID          string
    UserID      int32
    Timestamp   time.Time
    UserInput   string
    AgentOutput string
    ToolCalls   []ToolCall
    Outcome     string    // success/failure/partial
    Embedding   []float32 // 向量表示
}

// 语义记忆: 存储用户偏好
type SemanticMemory struct {
    preferences map[int32]*UserPreferences
}

type UserPreferences struct {
    Timezone       string
    DefaultDuration time.Duration // 默认日程时长
    CommonLocations []string       // 常用地点
    SchedulePatterns []SchedulePattern // 日程模式
}
```

**记忆检索流程**:

```
用户输入 ──► 短期记忆 (当前会话)
              │
              ├──► 情景记忆 (相似历史)
              │     └── Vector Search
              │
              ├──► 语义记忆 (用户偏好)
              │     └── Structured Query
              │
              └──► 聚合上下文 ──► Agent
```

#### 4.1.3 评估体系

```go
// plugin/ai/eval/evaluator.go

type AgentEvaluator struct {
    metrics    *MetricsCollector
    tracer     *Tracer
    logger     *slog.Logger
}

type Metrics struct {
    // 准确性
    TaskCompletionRate float64
    ToolSuccessRate    float64
    
    // 延迟
    P50Latency time.Duration
    P95Latency time.Duration
    P99Latency time.Duration
    
    // 成本
    TokensPerRequest   int
    LLMCallsPerRequest int
    
    // 质量
    HallucinationRate float64
    RetrievalPrecision float64
}

// 追踪单次请求
type Trace struct {
    RequestID   string
    UserID      int32
    StartTime   time.Time
    EndTime     time.Time
    Steps       []TraceStep
    Outcome     string
    Error       error
}

type TraceStep struct {
    Name      string
    StartTime time.Time
    EndTime   time.Time
    Input     string
    Output    string
    Tokens    int
}
```

### 4.2 Phase 2 详细设计

#### 4.2.1 Plan-Execute 模式

```go
// plugin/ai/agent/planner.go

type PlanExecuteAgent struct {
    planner   *Planner
    executors map[string]Executor
    critic    *Critic
}

type Planner struct {
    llm     ai.LLMService
    prompt  string
}

type Plan struct {
    Goal      string
    Steps     []PlanStep
    Context   map[string]interface{}
}

type PlanStep struct {
    ID          string
    Description string
    Executor    string   // 执行器名称
    Input       string   // 输入参数
    DependsOn   []string // 依赖步骤
}

// 执行流程
func (a *PlanExecuteAgent) Execute(ctx context.Context, input string) (*Result, error) {
    // 1. 规划
    plan, err := a.planner.CreatePlan(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // 2. 执行
    results := make(map[string]*StepResult)
    for _, step := range plan.Steps {
        executor := a.executors[step.Executor]
        result, err := executor.Execute(ctx, step)
        if err != nil {
            // 3. 重规划
            plan, err = a.planner.Replan(ctx, plan, step, err)
            if err != nil {
                return nil, err
            }
            continue
        }
        results[step.ID] = result
    }
    
    // 4. 评估
    finalResult := a.critic.Evaluate(ctx, plan, results)
    
    return finalResult, nil
}
```

**Plan-Execute 流程图**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Plan-Execute Workflow                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Input ──► Planner ──► Plan                                     │
│              │          │                                       │
│              │          ▼                                       │
│              │    ┌──────────────────────────────────┐          │
│              │    │ Step 1 ──► Step 2 ──► Step 3 ... │          │
│              │    └──────────────────────────────────┘          │
│              │                    │                             │
│              │                    ▼                             │
│              │              ┌─────────┐                         │
│              │              │ Critic  │                         │
│              │              └────┬────┘                         │
│              │                   │                              │
│              │           Good? ──┼──► Output                    │
│              │                   │                              │
│              └─── Re-plan ◄──────┘                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 4.2.2 Multi-Agent 协作框架

```go
// plugin/ai/agent/orchestrator.go

type Orchestrator struct {
    router     *Router
    agents     map[string]ParrotAgent
    supervisor *Supervisor
}

// Agent 角色定义
type AgentRole string

const (
    RoleRouter    AgentRole = "router"
    RolePlanner   AgentRole = "planner"
    RoleExecutor  AgentRole = "executor"
    RoleCritic    AgentRole = "critic"
)

// Handoff 机制
type Handoff struct {
    From       string
    To         string
    Context    map[string]interface{}
    Reason     string
}

// Supervisor 监控
type Supervisor struct {
    maxDuration time.Duration
    maxSteps    int
}

func (o *Orchestrator) Execute(ctx context.Context, input string) (*Result, error) {
    // 1. 路由决策
    route := o.router.Route(ctx, input)
    
    // 2. 任务分发 (支持 Handoff)
    var result *Result
    currentAgent := o.agents[route.Agent]
    
    for {
        stepResult, handoff, err := currentAgent.ExecuteWithHandoff(ctx, input)
        if err != nil {
            return nil, err
        }
        
        if handoff == nil {
            result = stepResult
            break
        }
        
        // Handoff 到下一个 Agent
        currentAgent = o.agents[handoff.To]
        input = handoff.Context["input"].(string)
    }
    
    return result, nil
}
```

#### 4.2.3 Generative UI 2.0

```typescript
// web/src/components/GenerativeUI/types.ts

export type UIToolType = 
  | 'schedule_suggestion'
  | 'time_slot_picker'
  | 'conflict_resolution'
  | 'memo_preview'
  | 'quick_actions'
  | 'confirmation_dialog'
  | 'progress_tracker';

export interface UITool {
  id: string;
  type: UIToolType;
  data: unknown;
  timestamp: number;
  sessionId?: string;
}

// 新增组件
export interface MemoPreviewData {
  memos: MemoSummary[];
  query: string;
  highlightTerms: string[];
}

export interface ProgressTrackerData {
  steps: ProgressStep[];
  currentStep: number;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
}

export interface ProgressStep {
  id: string;
  label: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  duration?: number;
}
```

**Generative UI 组件库**:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Generative UI Components                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ScheduleSuggestion│ │ TimeSlotPicker  │  │ConflictResolution│ │
│  │ 日程确认卡片     │ │ 时间槽选择器    │  │ 冲突解决面板    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  MemoPreview    │  │  QuickActions   │  │ProgressTracker  │ │
│  │  笔记预览卡片   │  │  快捷操作按钮   │  │  进度追踪器     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐                      │
│  │ConfirmationDialog│ │ StreamingFeedback│                     │
│  │  确认对话框      │  │  流式状态反馈   │                     │
│  └─────────────────┘  └─────────────────┘                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 Phase 3 详细设计

#### 4.3.1 Self-RAG 实现

```go
// plugin/ai/rag/self_rag.go

type SelfRAG struct {
    retriever     *AdaptiveRetriever
    llm           ai.LLMService
    reflector     *Reflector
}

type Reflector struct {
    llm ai.LLMService
}

// Reflection Token 类型
type ReflectionType string

const (
    ReflectRetrieve  ReflectionType = "retrieve"   // 是否需要检索
    ReflectRelevance ReflectionType = "relevance"  // 检索相关性
    ReflectSupport   ReflectionType = "support"    // 生成是否有依据
    ReflectUtility   ReflectionType = "utility"    // 回答是否有用
)

func (s *SelfRAG) Generate(ctx context.Context, query string) (string, error) {
    // 1. 判断是否需要检索
    needRetrieval := s.reflector.ShouldRetrieve(ctx, query)
    
    var context string
    if needRetrieval {
        // 2. 检索
        results, _ := s.retriever.Retrieve(ctx, &RetrievalOptions{
            Query: query,
            Strategy: "hybrid_standard",
        })
        
        // 3. 评估相关性
        relevantResults := s.reflector.FilterRelevant(ctx, query, results)
        context = formatContext(relevantResults)
    }
    
    // 4. 生成
    response, _ := s.llm.Chat(ctx, []ai.Message{
        {Role: "system", Content: s.buildPrompt(context)},
        {Role: "user", Content: query},
    })
    
    // 5. 自我评估
    critique := s.reflector.Critique(ctx, query, response)
    if critique.Score < 0.7 {
        // 6. 重试 (带更多上下文)
        return s.retryWithMoreContext(ctx, query, critique.Feedback)
    }
    
    return response, nil
}
```

#### 4.3.2 程序记忆 (Procedural Memory)

```go
// plugin/ai/memory/procedural.go

type ProceduralMemory struct {
    workflows map[int32][]LearnedWorkflow
    patterns  *PatternMatcher
}

// 学习到的工作流
type LearnedWorkflow struct {
    ID          string
    UserID      int32
    Trigger     string           // 触发条件 (正则/语义)
    Steps       []WorkflowStep   // 执行步骤
    Frequency   int              // 使用频率
    LastUsed    time.Time
    SuccessRate float64
}

// 模式匹配
func (p *ProceduralMemory) MatchWorkflow(ctx context.Context, userID int32, input string) *LearnedWorkflow {
    workflows := p.workflows[userID]
    
    for _, wf := range workflows {
        if p.patterns.Match(wf.Trigger, input) {
            return &wf
        }
    }
    
    return nil
}

// 学习新流程
func (p *ProceduralMemory) Learn(ctx context.Context, userID int32, trace *Trace) {
    // 从成功的 Trace 中提取模式
    if trace.Outcome != "success" {
        return
    }
    
    // 生成触发条件
    trigger := p.patterns.GenerateTrigger(trace.Steps)
    
    // 创建工作流
    workflow := LearnedWorkflow{
        UserID:  userID,
        Trigger: trigger,
        Steps:   convertToWorkflowSteps(trace.Steps),
    }
    
    p.workflows[userID] = append(p.workflows[userID], workflow)
}
```

---

## 5. 里程碑与验收标准

### 5.1 Phase 1 里程碑

| 里程碑 | 交付物 | 验收标准 | 预计周期 |
|--------|--------|---------|---------|
| M1.1 | Guardrails 框架 | 100% 请求覆盖，P99 延迟 <100ms | 2 周 |
| M1.2 | 记忆系统 V1 | 情景记忆可检索，用户偏好可配置 | 2 周 |
| M1.3 | 评估体系 | Dashboard 可用，核心指标可视化 | 1 周 |
| M1.4 | 工具标准化 | 所有工具符合统一 Schema | 1 周 |

### 5.2 Phase 2 里程碑

| 里程碑 | 交付物 | 验收标准 | 预计周期 |
|--------|--------|---------|---------|
| M2.1 | Plan-Execute | 复杂任务分解成功率 >80% | 3 周 |
| M2.2 | Agent 协作 | Handoff 成功率 >95% | 2 周 |
| M2.3 | Generative UI 2.0 | 覆盖 6+ 工具类型 | 2 周 |
| M2.4 | MCP Server | 标准接口可用 | 1 周 |

### 5.3 Phase 3 里程碑

| 里程碑 | 交付物 | 验收标准 | 预计周期 |
|--------|--------|---------|---------|
| M3.1 | Self-RAG | 检索效率提升 30%，幻觉率降低 50% | 4 周 |
| M3.2 | 程序记忆 | 常用模式识别准确率 >70% | 3 周 |
| M3.3 | 个性化 Agent | 用户满意度 >4.5/5 | 3 周 |
| M3.4 | 预测性 UI | 建议采纳率 >50% | 2 周 |

### 5.4 核心指标仪表板

```
┌─────────────────────────────────────────────────────────────────┐
│                    AI Assistant Metrics Dashboard                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Task Success    │  │  P95 Latency    │  │  Tokens/Req     │ │
│  │     92.3%       │  │     1.8s        │  │     3,200       │ │
│  │   ▲ +2.1%       │  │   ▼ -0.3s       │  │   ▼ -800        │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Hallucination   │  │  User Satisfaction│ │  Memory Hit     │ │
│  │     2.1%        │  │     4.6/5       │  │     78%         │ │
│  │   ▼ -1.2%       │  │   ▲ +0.3        │  │   ▲ +12%        │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Daily Request Volume                    │   │
│  │  ████████████████████████████████████████████ 12,340    │   │
│  │  ████████████████████████████████████ 10,890            │   │
│  │  ████████████████████████████████████████ 11,200        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 附录

### A. 技术选型建议

| 组件 | 推荐方案 | 备选方案 |
|------|---------|---------|
| 编排框架 | 自研 (Go) | LangGraph (Python) |
| 向量数据库 | pgvector | Qdrant |
| 记忆存储 | PostgreSQL + Redis | Mem0 |
| 评估工具 | 自研 + Prometheus | LangSmith |
| Guardrails | 自研 | Guardrails AI |

### B. 参考资料

1. [OpenAI Agents SDK - Multi-Agent Orchestration](https://openai.github.io/openai-agents-python/multi_agent/)
2. [LangGraph Plan-and-Execute](https://langchain-ai.github.io/langgraph/tutorials/plan-and-execute/)
3. [Agentic Orchestration Patterns That Scale](https://a21.ai/agentic-orchestration-patterns-that-scale/)
4. [Self-RAG: Learning to Retrieve, Generate, and Critique](https://arxiv.org/abs/2310.11511)
5. [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
6. [Vercel AI SDK - Generative User Interfaces](https://v4.ai-sdk.dev/docs/ai-sdk-ui/generative-user-interfaces)
7. [AI Agent Guardrails Framework](https://galileo.ai/blog/ai-agent-guardrails-framework)

---

*文档版本: v1.0 | 更新时间: 2026-01-27*
