# SessionSummary 定位说明

> **核心原则**: `Block.mode` 是 Mode 的**唯一真相来源**（Single Source of Truth）

---

## 三种模式对比

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DivineSense AI 聊天模式                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────────────┐  │
│  │ Normal Mode │  │  Geek Mode  │  │       Evolution Mode               │  │
│  │   (普通)     │  │   (极客)     │  │          (进化)                    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────────────────┘  │
│         │                │                          │                        │
│         ▼                ▼                          ▼                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           AI 代理                                     │   │
│  │                     AmazingParrot                                   │   │
│  │                      (综合助理)                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Normal Mode (普通模式)

```
┌─────────────────────────────────────────────────────────────────┐
│  Block.mode = "normal"                                          │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  AI 代理: AmazingParrot (综合助理)                       │    │
│  │  - LLM 驱动 (Z.AI GLM)                                   │    │
│  │  - 工具: memo_search, schedule_*                        │    │
│  │  - 用途: 日常问答、笔记搜索、日程管理                     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  SessionSummary: 空或仅有基础统计                                 │
│  - 无 sessionId                                                │
│  - 无详细 token 统计                                            │
└─────────────────────────────────────────────────────────────────┘
```

### Geek Mode (极客模式)

```
┌─────────────────────────────────────────────────────────────────┐
│  Block.mode = "geek"                                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  AI 代理: GeekParrot (Claude Code CLI 通信层)           │    │
│  │  - 零 LLM 调用 (直接透传到 CLI)                           │    │
│  │  - 工作目录: ~/.divinesense/claude/user_{id}              │    │
│  │  - 用途: 代码执行、文件操作、系统任务                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                       │                                          │
│                       ▼                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Claude Code CLI (独立 OS 进程)                          │    │
│  │  - --session-id <UUID>                                   │    │
│  │  - --output-format stream-json                           │    │
│  │  - 完整 MCP + Skills 能力                                │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  SessionSummary: 包含 CC Runner 统计                            │
│  - sessionId: Claude Code CLI session UUID                     │
│  - totalDurationMs, thinkingDurationMs, toolDurationMs          │
│  - inputTokens, outputTokens (LLM 使用)                         │
│  - toolCallCount, toolsUsed                                     │
│  - filesModified, filePaths                                     │
│  - totalCostUsd (LLM 成本)                                      │
└─────────────────────────────────────────────────────────────────┘
```

### Evolution Mode (进化模式)

```
┌─────────────────────────────────────────────────────────────────┐
│  Block.mode = "evolution"                                       │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  AI 代理: EvolutionParrot (自我进化)                     │    │
│  │  - 零 LLM 调用 (直接透传到 CLI)                           │    │
│  │  - 工作目录: DivineSense 源码根目录                      │    │
│  │  - 用途: 系统自我修改、源代码进化                         │    │
│  │  - 产出: GitHub PR (需人工审核)                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                       │                                          │
│                       ▼                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Claude Code CLI (独立 OS 进程)                          │    │
│  │  - --session-id <UUID> (Evolution 专用命名空间)          │    │
│  │  - --permission-mode bypassPermissions                  │    │
│  │  - 完整 MCP + Skills + Git 能力                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                       │                                          │
│                       ▼                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Git Workflow (强制)                                     │    │
│  │  1. 创建进化分支: evolution/{task-id}                    │    │
│  │  2. 提交修改: git commit                                  │    │
│  │  3. 创建 PR: gh pr create                                │    │
│  │  4. 等待人工审核 → 合并                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  SessionSummary: 包含 CC Runner 统计                            │
│  - sessionId: Evolution 专用 UUID (user namespace)             │
│  - 所有 Geek Mode 字段 + filesModified, filePaths             │
└─────────────────────────────────────────────────────────────────┘
```

---

## 数据结构关系

### 重构前（冗余设计）

```
┌─────────────────────────────────────────────────────────────────┐
│  ChatResponse.done = true                                       │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  SessionSummary                                            │  │
│  │  - sessionId: string                                       │  │
│  │  - mode: "geek" | "evolution" | "normal"  ← 冗余！        │  │
│  │  - totalDurationMs, tokens, tools...                      │  │
│  └───────────────────────────────────────────────────────────┘  │
│                          │                                       │
│                          ▼ (存储到)                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Block.session_stats (JSONB)                               │  │
│  │  - sessionId, durationMs, tokens...                       │  │
│  │  - mode: "geek" | "evolution" | "normal"                  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  问题: SessionSummary.mode 与 Block.mode 可能不一致              │
│  结果: 前端主题随机变化                                          │
└─────────────────────────────────────────────────────────────────┘
```

### 重构后（单一真相来源）

```
┌─────────────────────────────────────────────────────────────────┐
│  ChatResponse.done = true                                       │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  SessionSummary                                            │  │
│  │  - sessionId: string                                       │  │
│  │  - totalDurationMs, tokens, tools...                      │  │
│  │  - ~~mode~~ (已移除)                                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                          │                                       │
│                          ▼ (存储到)                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Block  (唯一真相来源)                                      │  │
│  │  - id, uid, conversation_id, round_number                 │  │
│  │  - mode: "normal" | "geek" | "evolution"  ← 唯一来源     │  │
│  │  - user_inputs[], assistant_content                        │  │
│  │  - event_stream[] (thinking, tool_use, tool_result...)    │  │
│  │  - session_stats: {                                        │  │
│  │      sessionId, durationMs, tokens, tools...              │  │
│  │    }                                                       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  前端读取: effectiveParrotId = blockModeToParrotAgentType(block.mode)│
└─────────────────────────────────────────────────────────────────┘
```

---

## SessionSummary 的正确定位

### SessionSummary 是什么？

**SessionSummary 是一次聊天会话的统计摘要**，**不是**会话模式的定义。

```
┌─────────────────────────────────────────────────────────────────┐
│                     SessionSummary 的本质                         │
│                                                                  │
│  它回答: "这次会话发生了什么？"                                    │
│  - 花了多长时间？                                                 │
│  - 消耗了多少 token？                                            │
│  - 调用了哪些工具？                                               │
│  - 修改了哪些文件？                                               │
│  - 成本是多少？                                                 │
│                                                                  │
│  它不回答: "这是什么模式？" ← 由 Block.mode 回答                  │
└─────────────────────────────────────────────────────────────────┘
```

### 何时有 SessionSummary？

| 模式 | SessionSummary | 原因 |
|:-----|:---------------|:-----|
| Normal | ❌ 无 | LLM 调用由后端内部管理，无需 CLI session 统计 |
| Geek | ✅ 有 | Claude Code CLI session，需要统计 |
| Evolution | ✅ 有 | Claude Code CLI session，需要统计 |

### 数据流向

```
Normal Mode:
  用户请求 → LLM → 响应 → Block (mode="normal", session_stats=null)

Geek Mode:
  用户请求 → GeekParrot → CC Runner → Block (mode="geek", session_stats={...})
                                              │
                                              ▼
                                     SessionSummary (流式发送)

Evolution Mode:
  用户请求 → EvolutionParrot → CC Runner → Block (mode="evolution", session_stats={...})
                                                   │
                                                   ▼
                                          SessionSummary (流式发送)
```

---

## 前端使用模式

### 主题选择（重构后）

```typescript
// ChatMessages.tsx
function useEffectiveParrotId(blocks: AIBlock[]): ParrotAgentType {
  // Block.mode 是唯一来源
  if (blocks && blocks.length > 0) {
    const lastAIBlock = blocks[blocks.length - 1];
    return blockModeToParrotAgentType(lastAIBlock.mode);
  }
  return ParrotAgentType.AMAZING;
}

// Mode → Theme 映射
const PARROT_THEMES: Record<ParrotAgentType, ThemeConfig> = {
  [ParrotAgentType.MEMO]: { border: "slate", headerBg: "slate", ... },
  [ParrotAgentType.SCHEDULE]: { border: "cyan", headerBg: "cyan", ... },
  [ParrotAgentType.AMAZING]: { border: "emerald", headerBg: "emerald", ... },
  [ParrotAgentType.GEEK]: { border: "violet", headerBg: "violet", ... },
  [ParrotAgentType.EVOLUTION]: { border: "rose", headerBg: "rose", ... },
};
```

### 会话统计显示

```typescript
// UnifiedMessageBlock.tsx
{sessionSummary && (
  <SummarySection>
    <Duration>{sessionSummary.totalDurationMs}ms</Duration>
    <Tokens>{sessionSummary.totalInputTokens + sessionSummary.totalOutputTokens}</Tokens>
    <Tools>{sessionSummary.toolsUsed.join(", ")}</Tools>
  </SummarySection>
)}
```

---

## 总结

| 概念 | 唯一来源 | 用途 |
|:-----|:---------|:-----|
| **Mode** | `Block.mode` | 决定主题、路由、行为 |
| **Session Stats** | `Block.session_stats` + 流式 `SessionSummary` | 会话统计、成本追踪 |

**关键原则**: Mode 只存在于 Block，SessionSummary 只负责统计。
