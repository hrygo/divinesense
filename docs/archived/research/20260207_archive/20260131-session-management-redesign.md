
# 纯净会话管理机制重构 (Non-intrusive Session Management)

## 1. 核心目标
重构现有的 `SEP` (Separator) 和 `SUM` (Summary) 机制，使其从原始会话流（Message Stream）中剥离，转为元数据驱动的管理方式。
同时，保留用户手动添加分隔符的能力，并确保系统自动行为对原始数据的“零污染”。
**新增**：引入基于 Token 的智能截断机制，以及对超大消息（如长文本分析结果）的即时摘要处理。

## 2. 核心问题分析
| 现状                         | 问题                          | 新方案                                               |
| :--------------------------- | :---------------------------- | :--------------------------------------------------- |
| System SEP 写入 `ai_message` | 污染用户会话历史              | System SEP 移除，转存至 `ai_conversation_checkpoint` |
| Summarizer 触发机制          | 仅依赖消息条数 (20条)         | **Token 阈值 (4k) + 消息数兜底**                     |
| 超大消息处理                 | 直接塞入 Context 导致爆 Token | **单条超大消息即时摘要 (Immediate Summary)**         |
| Context 构建                 | 依赖伪消息，逻辑耦合          | **基于 Checkpoint + 动态计算**                       |

## 3. 技术方案设计

### 3.1 数据库变更
新增 `ai_conversation_checkpoint` 表：
```sql
CREATE TABLE ai_conversation_checkpoint (
    id SERIAL PRIMARY KEY,
    conversation_id INT NOT NULL REFERENCES ai_conversation(id) ON DELETE CASCADE,
    last_message_id INT NOT NULL,
    summary TEXT NOT NULL,
    token_usage INT NOT NULL DEFAULT 0,
    created_ts BIGINT NOT NULL
);
CREATE INDEX idx_ai_conversation_checkpoint_cid_lastmsg ON ai_conversation_checkpoint(conversation_id, last_message_id DESC);
```

### 3.2 逻辑重构

#### A. 触发机制 (Trigger Logic)
`ConversationSummarizer` 支持两种触发模式：

1.  **累积触发 (Accumulative Trigger)**:
    *   条件: `TotalTokens >= TokenThreshold` (默认 2,000,000) 或 `MessageCount >= MaxLimit` (默认 50)。
    *   行为: 对累积的消息进行整体摘要，生成 Checkpoint。

2.  **大消息/Gen UI 智能处理 (Smart Handling for Large Messages)**:
    *   **预摘要 (Pre-summarization)**:
        *   当大消息（如 Gen UI 代码、Tool Output > 1M Tokens）产生时，**异步**生成摘要并存储在新增字段 `summary_cache` 中（不立即写入 Context Checkpoint）。
    *   **动态替换策略 (Dynamic Replacement)**:
        *   **Hot Zone (热区)**: 消息产生后的 N 轮对话内（默认 5轮），ContextBuilder **使用原文**。保证用户对当前生成内容的追问（如"修改颜色"）有效。
        *   **Cool Zone (冷区)**: 超过 N 轮后，ContextBuilder **使用 `summary_cache` 替换原文**。释放 Context 空间，保留长期记忆。
    *   **优势**: 兼顾了短期交互的精确性和长期对话的 Token 效率，避免了"即时截断"导致的上下文断层。

#### B. 写入路径 (Summarizer)
*   **全局清理**: 累积 Token > 2M -> 生成全局 Checkpoint (Truncate).
*   **局部优化**: 检测到大消息 -> 生成局部 Summary 存入 Metadata (不截断，供读取时替换).

#### C. 读取路径 (ContextBuilder Fusion Strategy)
1.  **加载存档**: 读取最新的 **全局 Checkpoint**。
2.  **加载新消息**: 读取 Checkpoint 之后的所有消息。
3.  **动态压缩 (Dynamic Compression)**:
    *   遍历加载的消息。
    *   如果遇到 **大消息/Gen UI** 且 `CurrentTurn - MessageTurn > HotZoneRadius` (已冷却):
        *   检查是否有 `summary_cache`。
        *   若有，**替换** 原文为 Summary。
        *   若无（摘要尚未生成），这也是异步机制的妥协，暂时使用原文，等待下一次请求。
4.  **手动截断优先**: 若发现 User Manual SEP，丢弃之前的 Checkpoint 和消息。

### 3.3 兼容性策略 (Breaking Change)
*   **不考虑向前兼容**: 鉴于目前无真实用户，我们将采取彻底的清理策略，以保证代码库的整洁。
*   **数据清洗**: Database Migration 脚本将**物理删除** `ai_message` 表中所有旧的 `System SEP` 和 `Summary` 类型的消息。
*   **引导用户**: 更新部署后，建议开发人员/用户重置或清空现有会话，或者接受通过 Migration 清洗后的会话状态。

## 4. Workload & Risk
*   **复杂度提升**: 需要在写入消息时（或写入后）立即进行 Token 判断，可能影响响应延迟。建议异步处理。
*   **配置 (Configurable with Defaults)**:
    *   `session.token_threshold`: 2,000,000 (2M triggers accumulative summary)
    *   `session.large_message_threshold`: 1,000,000 (1M triggers immediate summary)
    *   `session.max_message_count`: 50 (Fallback triggers accumulative summary)
*   **工作量**: 2.5 days (增加了即时摘要逻辑和配置项)

