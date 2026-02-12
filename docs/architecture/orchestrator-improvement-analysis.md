# DivineSense Agent 架构分析与改进建议

> **分析对象**：
> 1. Schedule Agent (Prompt & Mechanism)
> 2. Memo Agent (Prompt & Mechanism)
> 3. Orchestrator Configuration (Decomposer & Aggregator)
>
> **分析视角**：AI Agent 工程专家 & AI 科学家
> **基准**：2026 SOTA Multi-Agent Systems & LLM Best Practices
> **日期**：2026-02-12

---

## 1. 总体架构评估

DivineSense 的 Orchestrator-Workers 架构采用了经典的 **Router-Orchestrator-Worker** 模式，结合了 **FastRouter (L1)** 和 **LLM Orchestrator (L2)** 的两级路由机制，这是一个兼顾响应速度与处理复杂度的优秀设计。

### 核心优势
1.  **分层清晰**：L1 处理高频简单指令，L2 处理复杂长尾指令，资源利用率高。
2.  **工具即能力 (Tools-as-Capabilities)**：工具层（如 `scheduler.go`）封装了大量业务逻辑（时区、冲突解决），符合 **"Thin Agent, Fat Tools"** 的最佳实践，降低了 LLM 的幻觉风险。
3.  **结构化通信**：Agent 间通信主要依赖 JSON，且有由 Go 强类型定义的 Schema，保证了系统稳定性。

### 主要改进空间
1.  **Agent 自主性 (Autonomy) 不足**：目前的 Worker Agent (Schedule/Memo) 过度依赖 ReAct 循环，缺乏显式的 **Planning** 或 **Reflection** 步骤，导致在处理模糊指令时可能急于调用工具。
2.  **上下文传递 (Context Passing) 是静态的**：Decomposer 将任务拆解为静态字符串，缺乏 **Dynamic Variable Passing**（如将 Task A 的输出作为 Task B 的输入），限制了复杂流水线的表达能力。
3.  **缺乏反思与修正 (Self-Correction)**：在 Aggregator 阶段，如果发现某个 Agent 的输出不符合预期，缺乏自动回退或重试机制。

---

## 2. Schedule Agent 分析与改进

### 2.1 现状 (As-Is)
*   **Prompt**: `config/parrots/schedule.yaml`
*   **机制**: ReAct 循环，包含 "理解-选择-处理-确认" 四步。
*   **问题**:
    *   **过早陷入细节**: 容易直接通过工具尝试解决冲突，而不是先从更高层面思考替代方案。
    *   **缺乏主动澄清**: 当时间模糊（"改天"）或信息缺失（"约某人"但无联系方式）时，Prompt 未明确指导 Agent **Ask for Clarification**。

### 2.2 改进建议 (To-Be)

#### 2.2.1 引入 "Thought-Action-Observation" 强化
在 System Prompt 中强制开启 `<thinking>` 阶段，要求 Agent 在调用工具前先进行**可行性预演**。

**改进后的 System Prompt 片段建议：**

```markdown
## Execution Protocol (Strict Order)
1. <Analyze>: Parse user intent. Is the time/duration explicit?
   - IF NO time: Call `find_free_time` or `ask_user`.
   - IF modifying: MUST call `schedule_query` first to find the target event.
2. <Validation>: CHECK for logical conflicts (e.g., meeting at 3 AM).
3. <Execution>: Call the tool.
4. <Reflection>: Evaluate the tool specific output.
   - IF conflict: Propose alternatives politely. Don't just say "Failed".
```

#### 2.2.2 增强主动式交互 (Proactive Interaction)
增加 `clarification_triggers` 规则：
- **Trigger**: 只有动词没有时间 (e.g., "帮我安排会议") -> **Action**: Ask "请问您希望安排在具体哪天？或者通过 find_free_time 帮您查找合适的时间？"
- **Trigger**: 关键实体缺失 (e.g., "和他们开会") -> **Action**: Ask "请问是和哪个团队或具体哪位同事？"
-   **Trigger**: 只有动词没有时间 (e.g., "帮我安排会议") -> **Action**: Ask "请问您希望安排在具体哪天？或者通过 find_free_time 帮您查找合适的时间？"
-   **Trigger**: 关键实体缺失 (e.g., "和他们开会") -> **Action**: Ask "请问是和哪个团队或具体哪位同事？"

### 2.3 时间感知专项分析 (Time Awareness)

经过对 Universal Framework (`ai/agents/universal`) 和 Orchestrator (`ai/agents/orchestrator`) 的深入代码审计，发现存在严重的**时空感知断层**。

| 组件                              | 时间注入机制                 | 当前状态                                                                                                              | 风险/问题                                                                                                                                                                   |
| :-------------------------------- | :--------------------------- | :-------------------------------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Worker Agents** (Schedule/Memo) | `universal.BuildTimeContext` | ✅ **优秀**。通用框架会自动注入详细的 `<time_context>` JSON 块，包含当前时间、相对日期（Today/Tomorrow）、工作时间等。 | 无。底层框架已完美处理，Worker 具备极强的时间感知能力。                                                                                                                     |
| **Decomposer** (Orchestrator)     | `BuildDecomposerPrompt`      | ❌ **严重缺失**。Decomposer 的 Prompt 仅包含 `userInput` 和 `expertDescriptions`，完全没有时间上下文。                 | **"大脑"缺失时间概念**。Decomposer 无法理解 "下周五" 是具体哪天，导致其生成的 Task `input` 参数可能含糊不清，无法做基于时间的复杂路由决策（例如判断是本周还是下周的会议）。 |

**改进建议**:
1.  **Decomposer 复用 Universal TimeContext**: Orchestrator 已依赖 `ai/agents/universal` 包，应直接在 `Decomposer.Decompose` 中调用 `universal.BuildTimeContext(loc)`。
2.  **Prompt 增强**: 在 `BuildDecomposerPrompt` 中增加 time context 参数，并将格式化的 JSON 时间块注入到 System Context 中，使其具备与 Worker 同等的时间认知。

### 2.4 AI 包死代码分析 (Dead Code Analysis)

经过对 `ai` 包的静态分析，发现存在大量"僵尸代码" (Zombie Code)，即虽然被引用但属于遗留架构，应计划移除。

| 文件/组件                                        | 状态             | 说明                                                                                                         | 建议                                                  |
| :----------------------------------------------- | :--------------- | :----------------------------------------------------------------------------------------------------------- | :---------------------------------------------------- |
| `ai/agents/prompts.go`                           | 💀 **大部分已死** | 仅 `GetScheduleSystemPrompt` 被 `scheduler_v2.go` 使用。Memo, Amazing, Registry, A/B Test 等逻辑均未被引用。 | 提取 Schedule Prompt 到独立文件或配置，移除其余部分。 |
| `ai/agents/scheduler_v2.go`                      | 🧟 **僵尸代码**   | 仅被遗留服务 `ScheduleAgentService` 引用。功能上已被 `UniversalParrot` (Schedule Mode) 取代。                | 确认前端不再调用 v1 Schedule API 后彻底移除。         |
| `server/router/api/v1/schedule_agent_service.go` | 🧟 **僵尸服务**   | 使用旧版 Agent 实现。现代对话流应走 `ParrotHandler` (`/api/v1/chat/completions`)。                           | 标记 Deprecated，计划下线。                           |
| `ai/agents/memo_v2.go`                           | 👻 **已消失**     | 代码库中未找到，但 `prompts.go` 中仍保留了 Memo 相关 Prompt 代码。                                           | 清理 `prompts.go` 中的残留代码。                      |

**负资产风险**:
- **维护认知负担**: 新手开发者可能会误修改 `prompts.go`，以为会影响线上 Agent，实际现在的 Parrot Agent 使用 `universal/parrot_config.go` 和 YAML 配置。
- **配置割裂**: `scheduler_v2.go` 硬编码了 Tool Chain，与 `config/parrots` 下的配置脱节，导致行为不一致。

---

## 3. Memo Agent 分析与改进

### 3.1 现状 (As-Is)
*   **Prompt**: `config/parrots/memo.yaml`
*   **机制**: ReAct，核心是 `memo_search`。
*   **问题**:
    *   **搜索词单一**: 直接使用用户口语作为 Keyword，召回率可能低。
    *   **结果缺乏综合**: 仅做 Listing (罗列)，缺乏 Summarization (总结)。

### 3.2 改进建议 (To-Be)

#### 3.2.1 引入 "Query Expansion" (查询扩展)
在调用检索工具前，要求 Agent 生成 **2-3 个同义或关联的查询词**。

> **Example**:
> User: "上次那个很棘手的数据库 bug"
> Agent Thoughts: Keyword is "bug", related: "error", "exception", "crash", "database", "postgres".
> Action: `memo_search(query="bug error crash database")`

#### 3.2.2 结果合成增强 (Answer Synthesis)
不仅展示笔记片段，还要回答用户问题。
- **Prompt 指令**: "如果找到多个相关笔记，请先总结它们的共同点，再列出详情。"
- **引用规范**: "每条笔记必须附带 `[UID]` 或可点击的链接。"

---

## 4. Orchestrator Configuration 改进

### 4.1 Decomposer (`decomposer.yaml`)

**SOTA 建议：引入 Dependency Graph (依赖图)**

目前的任务列表是平铺的 (`tasks: []`)。建议引入 `dependencies` 字段，支持 **DAG (有向无环图)** 编排。

**改进后的 Output JSON 结构：**

```json
{
  "analysis": "...",
  "tasks": [
    {
      "id": "t1",
      "agent": "schedule",
      "input": "查询明天下午空闲时间",
      "purpose": "获取时间窗口"
    },
    {
      "id": "t2",
      "agent": "memo",
      "input": "结合 {{t1.result}}，查找该时间段前后的相关会议记录", // 这里的变量引用是关键
      "dependencies": ["t1"],
      "purpose": "上下文增强搜索"
    }
  ]
}
```

### 4.2 Aggregator (`aggregator.yaml`)

**SOTA 建议：Structure-Aware Synthesis (结构感知合成)**

Aggregator 不应只是一段通用的 "Merge Text" 指令。它应该感知 **来源的类型**。

**Prompt 增强：**
- **Type-Specific Rules**:
  - 对于 `Schedule` 结果：使用表格或时间轴展示。
  - 对于 `Memo` 结果：使用引用卡片展示。
- **Conflict Handling**: 如果 `Schedule` 说没空，但 `Memo` 说有个重要会议，Aggregator 应高亮这种冲突。

---

## 5. 总结与实施路线图

### 短期 (Quick Wins)
1.  **优化 System Prompts**: 更新 `schedule.yaml` 和 `memo.yaml`，加入 `<thinking>` 步骤和 `clarification` 策略。
2.  **查询扩展**: 在 Memo Agent 中通过 Prompt Engineering 实现简单的 Query Expansion。

### 中期 (Architectural)
1.  **Decomposer 升级**: 支持简单的变量传递和依赖关系描述。
2.  **Aggregator 增强**: 引入基于来源类型的结构化合成模板。

### 长期 (AI-Native)
1.  **User Profile 注入**: 在 Decomposer 阶段注入用户偏好（User Preference Embedding），实现个性化任务拆解。
2.  **Self-Evolving Prompts**: 基于用户反馈（点赞/点踩），自动优化 Agent 的 Few-shot Examples。
