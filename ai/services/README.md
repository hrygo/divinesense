# AI Services (`ai/services`)

`services` 包包含了 AI 模块的具体业务逻辑实现，它们通常组合了多个底层的 AI 能力（如 LLM, Embedding, Store）来提供完整的业务功能。

## Subpackages

### `schedule`
负责日程管理相关的智能逻辑，特别是重复规则的解析和计算。
*   **Recurrence**: 处理 RRule (Recurrence Rule) 标准，支持复杂的重复日程生成。

### `session`
管理 AI 对话会话 (Session)。
*   负责会话的创建、状态维护和持久化。
*   关联上下文 (Context) 和用户意图 (Intent)。

### `stats`
统计服务，用于分析 AI 模块的使用情况和性能指标。
