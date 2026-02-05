# 三方规格联合审计报告 (Joint Spec Audit)

> **审计对象**:
> 1. `unified-block-model_improvement.md` (基座改进)
> 2. `P1-A006-llm-stats-collection.md` (统计收集)
> 3. `tree-conversation-branching.md` (树状分支)

> **审计时间**: 2026-02-05
> **总体结论**: **可行，推荐采用串行实施路径 (Improvement -> Stats -> Tree)**

## 1. 核心依赖与冲突分析

### 1.1 代码级依赖冲突 (Critical)

**发现**:
- **Stats Spec** 对 `LLMService` 接口进行了**破坏性重构**（修改了 `Chat` 和 `ChatStream` 的返回值签名，增加了 `stats` 返回）。
- **Tree Spec** 的 "Edit & Regenerate" (Fork) 功能需调用 `LLMService`。

**风险**:
如果先实施 Tree Spec，开发者会基于旧的 `LLMService` 接口编写代码。随后实施 Stats Spec 时，必须回头重构 Tree Spec 刚写好的代码。此为无效返工。

**建议**:
先实施 **Stats Spec** (完成接口重构)，再实施 **Tree Spec** (直接基于新接口开发)。

### 1.2 数据一致性风险 (High)

**发现**:
- **Improvement Spec** 指出当前系统存在“时间戳单位不一致 (秒 vs 毫秒)”的严重 Bug。
- **Tree Spec** 和 **Stats Spec** 都涉及新的数据写入（Block Creation, Stats Logging）。

**风险**:
如果不先修复 Improvement Spec 中的时间戳问题，新功能（Tree/Stats）产生的数据可能继续沿用错误的单位，导致历史数据清洗极其困难。

**建议**:
**Improvement Spec 必须作为所有工作的 P0 前置任务**。

### 1.3 数据库 Schema 版本管理

- **Tree Spec** 提议 Schema 版本 `0.65.0`。
- **Improvement Spec** 涉及的修复可能需要微调 Schema 或数据清洗脚本。
- **建议**: 统一版本规划，避免 Migration 文件冲突。

---

## 2. 推荐实施路线图 (Execution Roadmap)

为了最小化返工风险，建议严格按照以下顺序执行：

### Step 1: 地基修复 (Foundation)
> **执行文档**: `unified-block-model_improvement.md`
> **目标**: 消除技术债务，统一标准。

-   ✅ **Action**: 修复时间戳 Bug (统一为毫秒)。
-   ✅ **Action**: 优化前端乐观更新 (Optimistic UI) 逻辑。
-   ✅ **Result**: 为后续功能提供稳定、标准一致的数据底座。

### Step 2: 核心重构 (Core Refactor)
> **执行文档**: `P1-A006-llm-stats-collection.md`
> **目标**: 升级 LLM 核心能力。

-   ✅ **Action**: 重构 `LLMService` 接口 (Stateless)。
-   ✅ **Action**: 实现 Token/Duration 统计流。
-   ✅ **Result**: 确立新的后端服务接口标准。

### Step 3: 功能扩展 (Feature Expansion)
> **执行文档**: `tree-conversation-branching.md`
> **目标**: 交付用户可见的新特性。

-   ✅ **Action**: 实施 `parent_block_id` 和 `branch_path` Schema 变更。
-   ✅ **Action**: 基于 **新版 LLMService** (Step 2 产出) 实现 `ForkBlock` 的重新生成逻辑。
-   ✅ **Result**: 对话功能从线性升级为树状，且天然带有精准的统计数据。

---

## 3. 规格修改建议 (Action Items for Specs)

虽然各文档已相对完善，但在联合视角下仍需微调：

1.  **对 `tree-conversation-branching.md` 的建议**:
    -   在 "依赖关系" 章节明确添加：**依赖 P1-A006 (LLM Stats) 完成接口重构**。
    -   在代码示例中，确保时间戳字段明确使用 `int64` 毫秒 (遵循 Improvement Spec)。

2.  **对 `unified-block-model_improvement.md` 的建议**:
    -   确认 `session_stats` 字段及其内部结构与 P1-A006 定义的 `LLMCallStats` 兼容。

## 4. 总结

三份文档在逻辑上是高度互补的：
- **Improvement** 修路（平整地基）
- **Stats** 换引擎（升级核心动力）
- **Tree** 开新线（扩展业务场景）

请按 **修路 -> 换引擎 -> 开新线** 的顺序推进。
