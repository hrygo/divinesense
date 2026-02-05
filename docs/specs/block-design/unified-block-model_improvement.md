# Unified Block Model: 深度分析与改进建议

> **背景**: 基于对 `unified-block-model.md` (v2.0) 的分析，结合行业标准（如 OpenAI Assistants API）及当前代码实现的审查，提出以下改进建议。
> **优先级**: **P0 (Highest)** - 必须在所有新功能开发前完成。

## 1. 架构验证与亮点

"Unified Block Model" 成功地将现代 AI Agent 工作流的复杂性封装在一个可管理的单元中。

-   **符合行业趋势**: `Block` 概念有效地融合了 OpenAI 的 `Message`（用户内容）和 `Run`（助手执行/事件）。这种“压缩”视图非常适合以用户体验为中心的聊天应用，其中“对话回合”是主要的交互单元。
-   **事件流设计**: 使用 JSONB 存储 `event_stream` 是正确的做法，避免了在关系型数据库中产生大量的细碎记录，同时保留了 AI 思考过程（Thinking）和工具调用（Tool Use）的完整可观测性。
-   **最近的代码优化**: 移除 `event_stream` 的 GIN 索引（Ref: `4d35f8a3`）是一个明智的决策，减少了写入开销，因为目前确实没有查询特定事件内容的强需求。

## 2. 发现的缺陷与风险 (Gaps & Risks)

### 2.1 [Bug] 时间戳单位不一致
-   **问题**: 规范中未明确定义时间戳单位，导致前后端实现不一致。
-   **现状**:
    -   后端 (`server/.../ai_service_block.go`): 使用 **秒** (`time.Now().Unix()`)。
    -   前端 (`web/.../UnifiedMessageBlock.tsx`): 使用 `new Date(timestamp)`，在 JS 中这被解析为 **毫秒**。
-   **影响**: 前端显示的日期变成了 1970 年（例如 `1738718000` 秒被当作毫秒处理）。
-   **建议**: 统一全栈使用 **毫秒 (Milliseconds)**。这是 JS/Java 生态的标准，且对未来需要更高精度的流式事件排序更友好。

### 2.2 [Bug] 乐观更新逻辑失效
-   **问题**: 前端 `useBlockQueries.ts` 中的 `useCreateBlock` 试图进行乐观更新，但逻辑有误。
-   **现状**:
    -   `onMutate`: 并没有将预期的 Block 插入到缓存中。
    -   `onSuccess`: 试图通过 `map` 替换 `id === 0` 的 Block。由于 `onMutate` 没插入，`map` 操作什么都没做。
-   **影响**: 用户发送消息后，界面上不会立即显示“发送中”的状态，必须等待服务器返回并触发 Refetch 后才会出现，造成“卡顿感”或“消息丢失感”。

### 2.3 [架构] 缺乏分支与树状结构支持
-   **现状**: 规范使用 `round_number` (Integer) 对 Block 进行排序，这强制了**线性对话**。
-   **局限**: 目前的 `onRegenerate` 仅支持重生成“最后一条”消息。如果未来要支持“编辑历史消息并重新生成”（类似 ChatGPT/Claude 的 Edit & Fork 功能），线性模型将无法胜任。
-   **建议**: 尽早引入 `parent_block_id` 字段。

## 3. 改进方案建议 (Proposals)

### 方案 A: 明确分支支持 (Schema Change)
建议修改 `ai_block` 表结构，支持非线性历史，为未来的“对话分叉”功能预留能力。

```diff
CREATE TABLE ai_block (
    id BIGSERIAL PRIMARY KEY,
+   parent_block_id BIGINT REFERENCES ai_block(id),  -- 支持树状结构
    round_number INTEGER NOT NULL,                   -- 保留用于线性投影排序
    ...
);
```
**价值**: 允许用户在对话的任意节点进行“编辑并重新提交”，系统可以创建一个指向旧父节点的新 Block，从而保留两条历史分支。

### 方案 B: 标准化时间戳
更新规范，强制要求所有 `_ts` 结尾的字段均使用 **毫秒 (`int64`)**。
**行动点**: 更新 PostgreSQL 触发器和 Go 结构体，使用 `time.Now().UnixMilli()`。

### 方案 C: 客户端 ID 协议 (解决乐观更新)
定义更健壮的前端乐观更新协议：
1.  **临时 ID**: 客户端生成 UUID (`temp_uid`)。
2.  **立即渲染**: 客户端将带有 `temp_uid` 和 `status=pending` 的 Block 立即推入 React Query 缓存。
3.  **既定事实**: 服务端在 `CreateBlock` 响应中返回这个 block (或客户端在 `onSuccess` 中用服务端 ID 替换临时 ID)。
4.  **去重**: 列表渲染时需注意去重（如果 Refetch 发生在 ID 替换之前）。

## 4. 立即执行计划 (Action Items)

1.  **修复时间戳 Bug**: 修改后端 `CreateBlock/UpdateBlock` 逻辑，使用 `UnixMilli`；或者在前端数据转换层 (`convertAIBlocksToMessageBlocks`) 统一乘以 1000。
2.  **修复缓存逻辑**: 修改 `useCreateBlock` 的 `onMutate`，确保它真正地向缓存数组中 `push` 一个临时 Block。
3.  **更新文档**: 将上述标准写入 `docs/specs/unified-block-model.md`。

### 关联兼容性 (Compatibility)

- **Session Stats**: 确认 `session_stats` 字段结构必须兼容 [P1-A006](./P1-A006-llm-stats-collection.md) 中定义的 `LLMCallStats`。
- **实施优先级**: 本文档为 **P0 (Highest)**，必须在 P1-A006 和 Tree Branching 之前完成，以避免数据污染。
