# AIChat 普通模式体验优化调研报告

> 调研日期: 2026-02-12 | Issue: #163

---

## 背景

AIChat 普通模式（MEMO/SCHEDULE/AMAZING 代理）在多 Block 会话场景下存在体验问题，需要优化会话连贯性、建议相关性、折叠策略，并修复终止按钮 Bug。

---

## 现有架构分析

### ChatRouter 路由机制

**位置**: `ai/agents/chat_router.go`, `ai/routing/service.go`

**三层路由架构**:
```
cache → rule → history → LLM
  0ms    0ms     10ms    400ms
```

- **Layer 0 - Cache**: 内存缓存，容量 500，TTL 5-30 分钟
- **Layer 1 - Rule**: 基于关键词匹配
- **Layer 2 - History**: 基于用户历史记录语义匹配
- **Layer 3 - LLM**: 最终兜底方案

### Block 状态管理

**位置**: `web/src/components/AIChat/UnifiedMessageBlock.tsx`

**折叠逻辑**:
```typescript
function getDefaultCollapseState(isLatest: boolean, isStreaming: boolean): boolean {
  if (isStreaming || isLatest) return false;
  return true;
}
```

### 智能建议生成

**位置**: `web/src/components/AIChat/utils/quickReplyAnalyzer.ts`

**响应类型**: ScheduleCreated, MemoFound, ScheduleQuery, FreeTimeFound, Error, Generic

---

## 问题分析

### 1. 会话连贯性不足

**现象**: 用户说"OK"确认日程时，路由系统无法感知上下文

**根因**: `HistoryMatcher` 是语义匹配，不是「意图粘性」。短确认词本身没有明确的意图特征。

### 2. 建议相关性低

**现象**: AI 追问"18点可以吗？"时，建议与追问无关

**根因**: 当前分析器只检测工具调用结果，不检测追问场景

### 3. 折叠策略单一

**现象**: 只有最新 Block 展开

**根因**: `getDefaultCollapseState` 只检查 `isLatest`

### 4. 终止按钮 Bug

**现象**: 多个 Block 同时显示终止按钮

**根因**: 状态更新时序问题 - 新 Block 创建但未添加到数组时，旧 Block 的 `isLatest` 仍为 true

---

## 技术方案

详见 Issue #163

---

## 相关文件

| 需求 | 文件 |
|:-----|:-----|
| 历史路由 | `ai/agents/context.go`, `ai/agents/chat_router.go` |
| 智能建议 | `web/src/components/AIChat/utils/quickReplyAnalyzer.ts` |
| 折叠策略 | `web/src/components/AIChat/UnifiedMessageBlock.tsx`, `ChatMessages.tsx` |
| 终止按钮 | `web/src/components/AIChat/ChatMessages.tsx`, `web/src/types/block.ts` |

---

## 参考资料

- [ChatRouter 源码](../ai/agents/chat_router.go)
- [Routing Service 源码](../ai/routing/service.go)
- [UnifiedMessageBlock 源码](../web/src/components/AIChat/UnifiedMessageBlock.tsx)
- [QuickReply 分析器](../web/src/components/AIChat/utils/quickReplyAnalyzer.ts)
