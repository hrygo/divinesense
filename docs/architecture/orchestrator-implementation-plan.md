# Orchestrator 改进实施方案

> **日期**: 2026-02-12
> **状态**: Ready for Implementation
> **基于**: [Orchestrator Improvement Analysis](./orchestrator-improvement-analysis.md)
> **目标**: 提升 DivineSense Agent 的自主性、上下文感知能力、结构化处理能力及工程韧性。

---

## 第一阶段：快速赢面 (提示词工程与清理)

本阶段专注于高影响力、低风险的变更，主要涉及配置更新、提示词优化及可观测性建设。

### 1.1 Schedule Agent 增强 (`config/parrots/schedule.yaml`)

**目标**: 防止过早调用工具，提高对模糊输入的鲁棒性，减少幻觉。

**行动**: 更新 `system_prompt`，加入严格的 "思考-行动" (Thought-Action) 协议和澄清触发器。

**建议变更**:
```yaml
system_prompt: |
  ## Identity
  ... (保留现有 Identity)

  ## Execution Protocol (Strict Order)
  1. <Analyze>: 分析用户意图。时间/时长是否明确？
     - IF 没有时间: 调用 `find_free_time` 或 `ask_user`。
     - IF 修改日程: 必须先调用 `schedule_query` 找到目标事件。
  2. <Validation>: 检查逻辑冲突 (例如：凌晨 3 点开会)。
  3. <Execution>: 调用工具。
  4. <Reflection>: 评估工具输出。
     - IF 冲突: 礼貌地提出替代方案。不要只说 "失败"。

  ## Clarification Triggers (澄清触发器)
  - IF 用户说 "安排会议" (无时间): Ask "请问您希望安排在具体哪天？或者通过 find_free_time 帮您查找合适的时间？"
  - IF 用户说 "和他们开会" (无具体人): Ask "请问是和哪个团队或具体哪位同事？"

  ... (保留现有 tools/examples)
```

### 1.2 Memo Agent 增强 (`config/parrots/memo.yaml`)

**目标**: 区分"列表浏览"与"内容搜索"意图，通过查询扩展提高召回率，通过结果合成提高回答质量。

**行动**: 在 `system_prompt` 中加入详细的意图分类和查询扩展指令。

**建议变更**:
```yaml
system_prompt: |
  ## Identity
  ... (保留现有 Identity)

  ## Advanced Search Protocol
  1. **Intent Classification (意图识别)**:
     - **Listing (列表/浏览)**: 用户想要按时间浏览笔记 (e.g., "近期笔记", "今天的记录")。
       - ACTION: 将模糊时间转化为具体的时间词。
       - Example: "近期" -> query="最近7天" (系统底层支持: 今天, 昨天, 本周, 最近N天)。
       - **禁止**: 直接搜索 "近期" 这种模糊关键词。
     - **Searching (搜索)**: 用户想要查找特定内容话题 (e.g., "PostgreSQL 配置")。
       - ACTION: 进入查询扩展流程。

  2. **Query Expansion (查询扩展 - 仅搜索模式)**:
     - 不要只使用用户的原始输入。
     - 生成 2-3 个相关的关键词或同义词。
     - Example: User "DB error" -> Query "DB error database crash exception postgres"

  3. **Answer Synthesis (结果合成)**:
     - IF 找到多条笔记: 先总结共同点。
     - IF 找到具体答案: 直接引用笔记内容回答用户问题。
     - 必须始终使用 `[UID]` 或标题标注来源。

  ... (保留现有 tools)
```

### 1.2.1 Memo Agent 能力与策略分析补充

**现状确认**:
Memo Agent **不需要**拆分为多个工具 (如 `sql_search`, `vector_search`)。其底层的 `AdaptiveRetriever` (自适应检索器) 已具备智能路由能力。

**策略命名说明 (Strategy Naming)**:
代码中使用的 `_only` 后缀 (e.g., `memo_list_only`) 旨在严格区分 **原子策略** (Atomic, 单一引擎, 极快) 与 **复合策略** (`hybrid_*`, 多引擎融合, 鲁棒)。这属于内部实现细节，无需重构。

**架构关系界定**:
| 组件                       | 层级                 | 职责范围             | 示例                                |
| :------------------------- | :------------------- | :------------------- | :---------------------------------- |
| **Orchestrator**           | **L1 Global Router** | 跨域调度、意图分发   | "约个会" -> Schedule Agent          |
| **Memo AdaptiveRetriever** | **L2 Local Router**  | 领域内深搜、策略优选 | "查近期" -> SQL; "查概念" -> Vector |

### 1.3 可观测性增强 (Observability)

**目标**: 打开 Agent 决策过程的 "黑盒"，便于调试和优化。

**行动**:
- [ ] **结构化日志**: 确保 `Decomposer`, `Executor`, `Aggregator` 输出带 `trace_id` 的 JSON 日志。
- [ ] **思维链追踪**: 更新 `Executor`，如果 LLM 返回 `<thinking>` 内容，将其作为 `Thought` 事件推送到前端展示。

### 1.4 死代码清理 (Cleanup)

- [ ] **移除** `ai/agents/scheduler_v2.go` (已被 UniversalParrot 取代)
- [ ] **移除** `server/router/api/v1/schedule_agent_service.go` (遗留接口)

---

## 第二阶段：Orchestrator 核心升级

本阶段涉及修改 Orchestrator 的 Go 代码 (`decomposer` 和 `aggregator`) 以支持高级特性。

### 2.1 上下文感知 (时间注入)

**目标**: 使 Decomposer 能够理解相对时间 (例如 "下周五")，以便生成准确的 Task Input。

**实施细节**:
1.  **代码变更**: 在 `Decomposer.Decompose` 方法中调用 `universal.BuildTimeContext(time.Now())`。
2.  **Prompt 变更**: 在 System Prompt 中注入 `{{.TimeContext}}`，并指示 LLM 使用它来解析相对日期。

### 2.2 依赖图 (DAG) 支持

**目标**: 支持复杂的串行任务流，其中任务 B 依赖于任务 A 的输出。

**实施细节**:
1.  **数据模型**: `Task` 结构体增加 `Dependencies []string`。
2.  **Prompt 变更**: 指导 Decomposer 输出依赖关系，并使用 `{{task_id.result}}` 占位符引用上游结果。
3.  **Executor 变更**: 支持解析依赖关系，按拓扑序或等待依赖完成后再执行。

### 2.3 结构感知聚合器 (Structure-Aware Aggregator)

**目标**: Aggregator 应根据 *来源* Agent 的类型格式化输出。

**行动**:
- 更新 Aggregator Prompt，增加格式化规则：
  - Schedule 来源 -> Markdown 表格
  - Memo 来源 -> 引用列表

### 2.4 错误处理与韧性

**目标**: 提高系统的容错能力。

**行动**:
- **重试机制**: 对 LLM API 超时等瞬态错误实现 Backoff 重试。
- **部分失败 (Partial Failure)**: 在并行执行中，允许部分非关键任务失败而不中断整个流程，由 Aggregator 决定如何告知用户。

---

## 第三阶段：验证策略

### 3.1 单元测试
- **新增**: `ai/agents/orchestrator/decomposer_test.go`
    - `TestDecompose_TimeContext`: 验证 Prompt 中包含了当前时间。
    - `TestParseTaskPlan_DAG`: 验证 JSON 解析能正确读取 `dependencies` 字段。

### 3.2 手动验证场景 (Acceptance Criteria)

1.  **复杂日程**: "帮我安排下周二下午的团队同步。" -> 验证 Decomposer 准确识别 "下周二" 的日期。
2.  **跨域协作**: "找到昨天关于 DB Bug 的笔记，并安排明天的评审会。" -> 验证 Task 2 (Schedule) 依赖 Task 1 (Memo) 的结果。
3.  **模糊指令**: "安排个会。" -> 验证 Schedule Agent 主动询问细节 (Time/People)。
4.  **笔记浏览**: "看看最近的笔记。" -> 验证 Memo Agent 使用 `memo_list_only` 或 `memo_filter_only` 策略，而非搜索关键词 "最近"。
