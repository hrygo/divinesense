# 服务端日程架构与交互策略 (Server-Side Scheduling Architecture & Interaction Strategy)

## 1. 问题陈述 (Problem Statement)
当前的日程管理实现存在 **逻辑不一致 (Logical Inconsistency)** 和 **交互冗余 (Interaction Redundancy)** 两大核心问题。

### 1.1 后端不一致性 (Inconsistency - Backend)
*   **双轨制入库 (Two Paths to DB)**:
    *   **API 路径 (`v1.ScheduleService`)**: 供 GUI 点击操作使用。在 Controller 层重新实现了一套松散且简陋的冲突检测逻辑。
    *   **AI 路径 (`ai.ScheduleAgent`)**: 供 Chat 对话使用。使用了健壮且智能的 `service.ConflictResolver`。
*   **风险**: GUI 可能允许创建一个 AI 会拒绝的日程，或者反之。API 层绕过了领域服务层 (Domain Service) 丰富的冲突解决能力。

### 1.2 前端交互冗余 (Redundancy - Frontend Interaction)
*   **日历里的话痨 AI (Chatty AI in Calendar)**: 日历顶部的 `ScheduleQuickInput` 组件直接调用了全功能的对话式 AI (`AgentType=SCHEDULE`)。
    *   **体验割裂**: 当用户只是想快速加个会时，却不得不忍受 AI 返回一大段 "好的，我已为您添加..." 的废话。用户只想看到结果。
    *   **冲突地狱**: 如果在快速输入框里遇到了时间冲突，用户被迫在一个小小的输入框里跟 AI 进行多轮文本谈判，而不是直接点击一个 "解决冲突" 的按钮。

## 2. 解决方案策略：统一大脑，自适应嘴巴 ("Unified Brain, Adaptive Mouth")

### 2.1 后端：统一逻辑到领域服务 (Backend: Unify Logic in Domain Service)
**目标**: API 操作和 AI 操作必须使用完全相同的大脑。

*   **重构 API 层 (Refactor API Layer)**:
    *   停止在 `server/router/api/v1/schedule_service.go` 中直接操作 `Store`。
    *   在 Controller 中注入 `service.ScheduleService` 接口。
    *   将 `Create`, `Update`, `CheckConflict` 等操作全部委托给 Domain Service。
    *   **收益**: GUI 将获得与 AI 同等强大的冲突检测和 RRule 展开能力。

### 2.2 交互：模式自适应 Agent (Interaction: Mode-Adaptive Agent)
**目标**: Agent 应根据上下文（Sidebar Chat vs. Calendar Quick Input）表现出不同的行为。

*   **新增请求字段**: 在 `ChatRequest` 中增加 `InteractionMode` 枚举：
    *   `MODE_CONVERSATIONAL` (默认): 用于侧边栏聊天。话痨、乐于助人、详细解释。
    *   `MODE_COMMAND` (用于快速输入): 用于日历快速输入。**静默执行** (Silent Execution)。只有出错时才说话。

*   **Agent 逻辑 (ScheduleParrot)**:
    *   **在 Command 模式下**:
        *   如果 `Tool` 执行成功：返回空文本或简单的 "Done"。让生成的 `BlockEvent` (`tool_result`) 驱动前端 UI 刷新。
        *   如果 `Tool` 执行失败 (冲突)：返回一个 **结构化错误 (Structured Error)** (JSON 包含在文本中)，详细列出替代方案，而不是写一段道歉信。

### 2.3 前端：结构化降级 (Frontend: Structured Fallback)
**目标**: 当事情变得复杂时，逃离那些该死的文本框。

*   **冲突 UI (Conflict UI)**:
    *   在 `ScheduleQuickInput` 中，解析 AI 返回的 "冲突错误"。
    *   如果收到结构化错误，不再展示文本，而是直接弹出一个 **Slot Picker (冲突解决器)** 界面（利用 `service.ConflictResolver` 返回的数据）。
    *   用户只需点击一个推荐的时间槽 -> 系统发送修正指令 -> 完成。

## 3. 实施计划 (Implementation Plan)

### Phase 1: 后端大一统 (Backend Unification) - P1
1.  **重构**: 重写 `api/v1/schedule_service.go`，使其全面调用 `service/schedule/service.go`。
2.  **验证**: 确保 API 现在的行为与 Domain Service 的逻辑一致，特别是冲突检测。

### Phase 2: Agent 模式自适应 (Agent Mode) - P2
1.  **Proto**: 在 `ChatRequest` proto 中添加 `InteractionMode` 字段。
2.  **Parrot**: 更新 `ScheduleParrot` 以识别 `MODE_COMMAND`（抑制闲聊）。
3.  **Frontend**: 更新 `ScheduleQuickInput` 发送 `MODE_COMMAND`。

### Phase 3: 智能冲突 UI (Smart Conflict UI) - P3
1.  **Resolver**: 通过 `ToolOutput` 暴露 `ConflictResolver` 的替代方案。
2.  **UI**: 创建 `ConflictResolverModal` 组件。当检测到 AI 返回冲突结构体时自动触发。

## 4. 为什么这个方案能赢 (Why This Wins)
1.  **一致性 (Consistency)**: 再也不会出现 "API 让我加但 AI 不同意" 的情况。
2.  **效率 (Efficiency)**: 日历输入变得像命令行一样快，但又像 AI 一样聪明。
3.  **体验 (Experience)**: 复杂的冲突通过 **1次点击** (UI) 解决，而不是 **10次打字** (Chat)。
