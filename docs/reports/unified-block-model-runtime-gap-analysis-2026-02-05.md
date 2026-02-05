# Unified Block Model - 运行时差距分析报告（根因分析）

**报告日期**: 2026-02-05
**分析类型**: 运行时故障根因分析
**严重等级**: P0 - 关键生产问题
**报告人**: Loki Mode v5.9.0

---

## 执行摘要

### 问题概述

用户报告的运行时差距与静态代码分析结论**完全相反**。虽然代码实现完整，但存在**关键的并发安全和错误处理缺陷**，导致：

1. **Block Mode 未正确持久化** - 主题随机变化
2. **Event Stream 刷新后丢失** - 只保留用户输入和最终回复

### 根本原因

| 问题 | 位置 | 严重性 | 影响 |
|:-----|:-----|:-------|:-----|
| **异步事件追加（错误抑制）** | `handler.go:483-486` | 🔴 P0 | event_stream 为空 |
| **Fallback 逻辑缺陷** | `useBlockQueries.ts:627` | 🟡 P1 | 新会话强制回退到 items |
| **缺少数据库验证日志** | `block_manager.go` | 🟡 P1 | 无法诊断持久化失败 |

---

## 详细分析

### 问题 1: 异步事件追加（P0 - 关键）

#### 代码位置

**文件**: `server/router/api/v1/ai/handler.go`
**行号**: 483-486

```go
// Phase 5: Append event to Block (async, don't block streaming)
if currentBlock != nil && h.blockManager != nil {
    // Build metadata for block event
    var eventMetaForBlock map[string]any
    if eventMeta != nil {
        eventMetaForBlock = map[string]any{
            // ... metadata fields
        }
    }

    // Append event asynchronously (don't block streaming)
    go func() {
        _ = h.blockManager.AppendEvent(ctx, currentBlock.ID, eventType, dataStr, eventMetaForBlock)
    }()

    // Collect assistant content for block completion
    if eventType == "answer" || eventType == "content" {
        assistantContentMu.Lock()
        assistantContent.WriteString(dataStr)
        assistantContentMu.Unlock()
    }
}
```

#### 问题分析

1. **错误被静默忽略**: `_ = h.blockManager.AppendEvent(...)` 不检查返回值
2. **Goroutine 无同步**: 发起的 goroutine 可能仍在运行，但主函数已返回
3. **无法检测失败**: 即使 AppendEvent 失败，调用方也无从知晓

#### 影响链

```
Streaming Event
    │
    ▼
go func() { _ = blockManager.AppendEvent(...) }  ← 如果失败，错误被吞噬
    │
    ▼
Block.event_stream 保持为空 []  ← 数据库中无事件数据
    │
    ▼
页面刷新后，GetBlock 返回空 event_stream
    │
    ▼
前端 convertAIBlocksToMessageBlocks 无法提取 thinkingSteps/toolCalls
    │
    ▼
UI 只显示用户输入和最终回复，其他信息丢失
```

#### 数据验证

**数据库 schema** (`20260204000000_add_ai_block.up.sql`):
```sql
event_stream JSONB NOT NULL DEFAULT '[]',
```

**创建时初始化** (`ai_block.go:110`):
```go
[]byte("[]"), // event_stream - 初始为空 JSON 数组
```

**如果 AppendEvent 全部失败，event_stream 将保持为 `[]`**，这与用户报告一致："仅仅持久化了用户输入消息，和最终模型返回消息"。

#### 修复方案

```go
// 方案 1: 同步追加（简单，但可能阻塞流）
if currentBlock != nil && h.blockManager != nil {
    if err := h.blockManager.AppendEvent(ctx, currentBlock.ID, eventType, dataStr, eventMetaForBlock); err != nil {
        logger.Warn("Failed to append event to block",
            slog.Int64("block_id", currentBlock.ID),
            slog.String("event_type", eventType),
            slog.String("error", err.Error()),
        )
    }
}

// 方案 2: 带错误通道的异步追加（推荐）
type appendResult struct {
    blockID  int64
    eventType string
    err      error
}

resultChan := make(chan appendResult, 100) // 缓冲通道

go func() {
    err := h.blockManager.AppendEvent(ctx, currentBlock.ID, eventType, dataStr, eventMetaForBlock)
    resultChan <- appendResult{blockID: currentBlock.ID, eventType: eventType, err: err}
}()

// 在流结束时收集错误
defer func() {
    close(resultChan)
    for result := range resultChan {
        if result.err != nil {
            logger.Warn("Async event append failed",
                slog.Int64("block_id", result.blockID),
                slog.String("event_type", result.eventType),
                slog.String("error", result.err.Error()),
            )
        }
    }
}()

// 方案 3: 批量追加（最优性能）
// 收集所有事件到内存，流结束时一次性写入
```

---

### 问题 2: Fallback 逻辑缺陷（P1）

#### 代码位置

**文件**: `web/src/hooks/useBlockQueries.ts`
**行号**: 627

```typescript
const shouldFallback = query.isError || (query.isSuccess && blocks.length === 0 && conversationId > 0);
```

#### 问题分析

当 `blocks.length === 0` 时（新会话或无数据），`shouldFallback` 变为 `true`，导致：

1. `blocks` 被强制设为 `[]`（AIChat.tsx:327）
2. UI 渲染使用 `items` 而非 `blocks`
3. 新消息发送时，无法利用 Block API 的完整功能

#### 循环依赖

```
blocks.length === 0
    │
    ▼
shouldFallback = true
    │
    ▼
blocks = []  (即使后续 API 返回数据)
    │
    ▼
shouldFallback 保持 true（死锁）
```

#### 修复方案

```typescript
// 区分"新会话"和"API 失败"
const isAPIError = query.isError;
const isNewConversation = query.isSuccess && blocks.length === 0;
const isLoaded = query.isSuccess || query.isError;

// 只在真正错误时回退
const shouldFallback = isAPIError;

// 向调用方暴露更详细的状态
return {
    blocks,
    isLoading: query.isLoading,
    error: query.error ?? null,
    shouldFallback,
    isNewConversation,  // 新增：让调用方决定如何处理
    isLoaded,
};
```

---

### 问题 3: Block Mode 持久化分析

#### 后端持久化链

```
handler.go:308-315  →  确定 blockMode
    │
    ▼
block_manager.go:41-56  →  转换为 storeMode
    │
    ▼
ai_block.go:106  →  插入 mode 列
    │
    ▼
数据库: ai_block.mode TEXT NOT NULL DEFAULT 'normal'
```

**验证**: Mode **正确持久化**到数据库。

#### 前端读取链

```
useBlocksWithFallback()  →  listBlocks API
    │
    ▼
ChatMessages.tsx:42-44  →  从 block.mode 读取
    │
    ▼
useEffectiveParrotId()  →  转换为 ParrotAgentType
    │
    ▼
PARROT_THEMES[parrotId]  →  应用主题
```

**验证**: Mode **正确读取**并应用。

#### 可能的问题

用户报告"主题随意变化"可能源于：

1. **默认值覆盖**: 如果 Mode 字段为空字符串（而非 `normal`），可能被误解析
2. **转换错误**: `blockModeToParrotAgentType` 的边界情况处理
3. **SessionSummary 优先级**: `useEffectiveParrotId` 中 SessionSummary.mode 优先级高于 Block.mode

```typescript
// ChatMessages.tsx:36-49
function useEffectiveParrotId(...): ParrotAgentType {
    return useMemo(() => {
        // Session summary has highest priority
        if (sessionSummary?.mode === "geek") return ParrotAgentType.GEEK;
        if (sessionSummary?.mode === "evolution") return ParrotAgentType.EVOLUTION;

        // Check last Block mode
        if (blocks && blocks.length > 0) {
            const lastAIBlock = blocks[blocks.length - 1];
            return blockModeToParrotAgentType(lastAIBlock.mode);
        }

        return currentParrotId ?? ParrotAgentType.AMAZING;
    }, [currentParrotId, sessionSummary?.mode, blocks]);
}
```

**如果 SessionSummary.mode 与 Block.mode 不一致**，将导致主题随机变化。

---

## 验证计划

### 数据库验证

```sql
-- 检查 Block mode 持久化
SELECT id, conversation_id, mode, status,
       jsonb_array_length(event_stream) as event_count
FROM ai_block
ORDER BY created_ts DESC
LIMIT 10;

-- 检查 event_stream 是否为空
SELECT id, mode, event_stream
FROM ai_block
WHERE jsonb_array_length(event_stream) = 0;
```

### 预期结果

| 场景 | event_count | 状态 |
|:-----|:-----------|:-----|
| 正常流式聊天 | > 0 | ✅ |
| 异步追加失败 | = 0 | ❌ 当前 bug |
| Mode 持久化 | 'geek'\|'evolution'\|'normal' | ✅ |

### 日志验证

**添加调试日志**到 `block_manager.go`:

```go
func (m *BlockManager) AppendEvent(...) error {
    slog.Info("Appending event to block",
        slog.Int64("block_id", blockID),
        slog.String("event_type", eventType),
        slog.Int("content_length", len(content)),
    )

    err := m.store.AppendEvent(ctx, blockID, event)

    if err != nil {
        slog.Error("Failed to append event",
            slog.Int64("block_id", blockID),
            slog.String("error", err.Error()),
        )
        return err
    }

    slog.Info("Event appended successfully",
        slog.Int64("block_id", blockID),
    )

    return nil
}
```

---

## 修复优先级

| 优先级 | 问题 | 预计工时 | 风险 |
|:-------|:-----|:---------|:-----|
| **P0** | 异步事件追加错误抑制 | 4h | 高（并发安全） |
| **P0** | 添加数据库验证日志 | 2h | 低 |
| **P1** | 前端 Fallback 逻辑修复 | 2h | 中 |
| **P2** | SessionSummary/Block mode 优先级问题 | 4h | 中 |

---

## 建议

### 立即行动

1. **添加错误日志**：在 `block_manager.AppendEvent` 中添加结构化日志
2. **验证生产环境**：检查数据库中 event_stream 的实际状态
3. **临时回退**：如果问题严重，考虑暂时同步追加事件

### 长期改进

1. **事件队列**：引入内存队列缓冲，批量写入数据库
2. **重试机制**：AppendEvent 失败时自动重试
3. **监控告警**：检测 event_stream 为空的 Block 数量

---

## 附录

### 相关文件

| 文件 | 行号 | 描述 |
|:-----|:-----|:-----|
| `handler.go` | 483-486 | 异步事件追加（bug 位置） |
| `useBlockQueries.ts` | 627 | Fallback 逻辑 |
| `ai_block.go` | 68-141 | CreateAIBlock 实现 |
| `block_manager.go` | 75-107 | AppendEvent 实现 |
| `ChatMessages.tsx` | 36-49 | Mode 优先级逻辑 |

### 相关 Issue

- **Issue #71**: Unified Block Model 实现
- **原始报告**: `docs/reports/unified-block-model-gap-analysis-2026-02-05.md`

---

*报告生成: Loki Mode v5.9.0*
*分析日期: 2026-02-05*
*状态: 待修复*
