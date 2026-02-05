# Unified Block Model: 深度分析与改进建议

> **背景**: 基于对 `unified-block-model.md` (v2.0) 的分析，结合行业标准（如 OpenAI Assistants API）及当前代码实现的审查，提出以下改进建议。
> **优先级**: **P0 (Highest)** - 必须在所有新功能开发前完成。
> **状态**: ✅ **已完成** (2026-02-05) - 时间戳和树状分支支持已实现

## 1. 架构验证与亮点

"Unified Block Model" 成功地将现代 AI Agent 工作流的复杂性封装在一个可管理的单元中。

-   **符合行业趋势**: `Block` 概念有效地融合了 OpenAI 的 `Message`（用户内容）和 `Run`（助手执行/事件）。这种"压缩"视图非常适合以用户体验为中心的聊天应用，其中"对话回合"是主要的交互单元。
-   **事件流设计**: 使用 JSONB 存储 `event_stream` 是正确的做法，避免了在关系型数据库中产生大量的细碎记录，同时保留了 AI 思考过程（Thinking）和工具调用（Tool Use）的完整可观测性。
-   **最近的代码优化**: 移除 `event_stream` 的 GIN 索引（Ref: `4d35f8a3`）是一个明智的决策，减少了写入开销，因为目前确实没有查询特定事件内容的强需求。

## 2. 发现的缺陷与风险 (Gaps & Risks)

### 2.1 [Bug] ✅ 已修复 - 时间戳单位不一致
-   **问题**: 规范中未明确定义时间戳单位，导致前后端实现不一致。
-   **状态**: ✅ **已修复** (2026-02-05)
-   **修复内容**:
    - 创建迁移 `20260205000000_timestamp_milliseconds.up.sql` 将现有秒级时间戳转换为毫秒
    - 更新后端代码使用 `time.Now().UnixMilli()` 替代 `Unix()`
    - 更新 PostgreSQL 触发器使用毫秒: `EXTRACT(EPOCH FROM NOW()) * 1000::BIGINT`
    - 前端已正确使用毫秒，无需修改
-   **影响**: 修复后前端时间戳显示正常，不再出现 1970 年问题

### 2.2 [Bug] ✅ 已修复 - 乐观更新逻辑
-   **问题**: 前端 `useBlockQueries.ts` 中的 `useCreateBlock` 试图进行乐观更新，但逻辑有误。
-   **状态**: ✅ **已验证无需修复** (2026-02-05)
-   **结论**: 经代码审查，当前实现 (v0.93.0) 已包含正确的乐观更新逻辑：
    - `onMutate` 正确创建临时 Block 并插入缓存
    - `onSuccess` 正确替换临时 Block 为服务器返回的 Block
    - `onError` 正确回滚缓存
-   **说明**: 原规格文档中的问题描述已过时，代码在 v0.93.0 中已实现

### 2.3 [架构] ✅ 已实现 - 缺乏分支与树状结构支持
-   **状态**: ✅ **已实现** (2026-02-05)
-   **实现内容**:
    - 创建迁移 `20260205000001_add_parent_block_id.up.sql`
    - 添加 `parent_block_id` (BIGINT, nullable) 和 `branch_path` (TEXT) 列
    - 更新 store 接口和 Go 结构体
    - 添加索引优化查询性能
    - 更新数据库触发器自动生成 branch_path
-   **价值**: 允许用户在对话的任意节点进行"编辑并重新提交"，系统可以创建一个指向旧父节点的新 Block，从而保留两条历史分支

## 3. 改进方案建议 (Proposals)

### ✅ 方案 A: 明确分支支持 (Schema Change) - 已实现
建议修改 `ai_block` 表结构，支持非线性历史，为未来的"对话分叉"功能预留能力。

**已实现**:
```sql
CREATE TABLE ai_block (
    id BIGSERIAL PRIMARY KEY,
    parent_block_id BIGINT,          -- ✅ 已添加
    branch_path TEXT,                -- ✅ 已添加
    round_number INTEGER NOT NULL,    -- 保留用于线性投影排序
    ...
);
```

**价值**: 允许用户在对话的任意节点进行"编辑并重新提交"，系统可以创建一个指向旧父节点的新 Block，从而保留两条历史分支。

### ✅ 方案 B: 标准化时间戳 - 已实现
更新规范，强制要求所有 `_ts` 结尾的字段均使用 **毫秒 (`int64`)**。

**已实现**:
- 迁移脚本: `20260205000000_timestamp_milliseconds.up.sql`
- Go 代码: 所有 `time.Now().Unix()` 改为 `time.Now().UnixMilli()`
- PostgreSQL 触发器: 使用 `EXTRACT(EPOCH FROM NOW()) * 1000`

### 方案 C: 客户端 ID 协议 (解决乐观更新) - 已验证无需实现
定义更健壮的前端乐观更新协议：
1.  **临时 ID**: 客户端生成 UUID (`temp_uid`)。
2.  **立即渲染**: 客户端将带有 `temp_uid` 和 `status=pending` 的 Block 立即推入 React Query 缓存。
3.  **既定事实**: 服务端在 `CreateBlock` 响应中返回这个 block (或客户端在 `onSuccess` 中用服务端 ID 替换临时 ID)。
4.  **去重**: 列表渲染时需注意去重（如果 Refetch 发生在 ID 替换之前）。

**状态**: ✅ 已在 v0.93.0 中实现，无需额外修改

## 4. 执行计划状态 (Action Items)

| 任务 | 状态 | 说明 |
|:-----|:-----|:-----|
| 修复时间戳 Bug | ✅ 已完成 | 使用 UnixMilli，迁移脚本已创建 |
| 修复缓存逻辑 | ✅ 已完成 | v0.93.0 已包含正确实现 |
| 添加 parent_block_id | ✅ 已完成 | 迁移和 Go 代码已更新 |
| 更新文档 | ✅ 已完成 | 本文档 |

### 关联兼容性 (Compatibility)

- **Session Stats**: ✅ `session_stats` 字段结构兼容 [P1-A006](./P1-A006-llm-stats-collection.md) 中定义的 `LLMCallStats`。
- **实施优先级**: ✅ 本文档所有 P0 任务已完成，可以继续开发 P1-A006 和 Tree Branching 功能。
