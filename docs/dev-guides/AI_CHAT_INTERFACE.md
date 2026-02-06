# AI Chat 界面架构

> **保鲜状态**: ✅ 已验证 (2026-02-06) | **最后检查**: v0.93.0 (Unified Block Model)
> **关联规格**: [Unified Block Model](../specs/unified-block-model.md) | [P1-A006](../specs/block-design/llm-stats-collection.md)

## 概述

AI Chat 界面采用 **Unified Block Model（统一块模型）** 设计，将用户输入与 AI 回复封装为一个完整的可折叠 Block。每个 Block 包含完整的对话上下文、Token 统计、成本追踪和分支信息。

---

## 界面布局

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           DivineSense AI Chat Interface                             │
│                        (Unified Block Model - v0.93.0)                             │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────┐
│  HEADER: Mode Switcher & Session Info                                              │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  ○ Normal Mode  ● Geek Mode  ○ Evolution Mode         |  Session: #12345         │
│  ┌─────────────────────────────────────────────────────┐  │  Cost: $0.0234         │
│  │ Normal: 三层路由 → MemoParrot/ScheduleParrot       │  │  Tokens: 15.2K         │
│  │ Geek:   Claude Code CLI (零 LLM)                   │  └─────────────────────────┘
│  │ Evolution: 系统自我进化 (PR 提交)                   │
│  └─────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  INPUT AREA                                                                           │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  ┌────────────────────────────────────────────────────────────────────────────────┐ │
│  │ │ 输入消息... 发送 Ctrl+Enter 换行                                          │ │
│  └────────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────┐
│  CHAT STREAM (AIBlock[] - 按时间倒序)                                                  │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────┐
│  ╔═══════════════════════════════════════════════════════════════════════════════╗  │
│  ║  UnifiedMessageBlock #3 - Block ID: 789                               [▼]    ║  │
│  ║  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ║  │
│  ║  Block Header                                                                 ║  │
│  ║  ┌────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │ 🌿 Branch: 0/1/2  │  🤖 AUTO → MemoParrot  │  ⏱ 2.3s  │  💰 $0.0087    │   ║  │
│  ║  │ [🔄 Regenerate] [📋 Copy] [🌿 Fork] [✏️ Edit] [🗑️ Delete]              │   ║  │
│  ║  └────────────────────────────────────────────────────────────────────────┘   ║  │
│  ║                                                                               ║  │
│  ║  Block Body (Collapsible)                                                    ║  │
│  ║  ┌────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │ 👤 User: "查找关于 Neo4j 图数据库的笔记"                              │   ║  │
│  ║  │                                                                        │   ║  │
│  ║  │ 🤖 Assistant:                                                          │   ║  │
│  ║  │ 找到 3 条相关笔记：                                                   │   ║  │
│  ║  │ 1. Neo4j 基础概念 (相关度: 0.92)                                   │   ║  │
│  ║  │ 2. Cypher 查询语言入门 (相关度: 0.87)                               │   ║  │
│  ║  │ 3. 图数据库建模最佳实践 (相关度: 0.81)                              │   ║  │
│  ║  │                                                                        │   ║  │
│  ║  │ 🛠️ Tool Calls:                                                        │   ║  │
│  ║  │ ┌─ memo_search ─────────────────────────────────────────┐            │   ║  │
│  ║  │ │ Query: "Neo4j 图数据库"                                 │            │   ║  │
│  ║  │ │ Results: 3 blocks, max_score: 0.92                     │            │   ║  │
│  ║  │ │ Duration: 45ms                                           │            │   ║  │
│  ║  │ └───────────────────────────────────────────────────────────┘            │   ║  │
│  ║  │                                                                        │   ║  │
│  ║  │ 📊 Session Summary:                                                  │   ║  │
│  ║  │ ┌─ LLM Stats ─────────────────────────────────────────────────┐       │   ║  │
│  ║  │ │ 🪙 Tokens:                                                       │       │   ║  │
│  ║  │ │   Input:  856    │  Output: 423    │  Total: 1,279             │       │   ║  │
│  ║  │ │   Cache Read: 142 (绿色) │  Cache Write: 28 (蓝色)           │       │   ║  │
│  ║  │ │ ⏱ Timing:                                                       │       │   ║  │
│  ║  │ │   Thinking: 156ms  │  Generation: 412ms  │  Total: 568ms      │       │   ║  │
│  ║  │ │ 🏷 Model: deepseek-chat                                          │       │   ║  │
│  ║  │ └───────────────────────────────────────────────────────────────────┘       │   ║  │
│  ║  └────────────────────────────────────────────────────────────────────────┘   ║  │
│  ║                                                                               ║  │
│  ║  Block Footer                                                                 ║  │
│  ║  ┌────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │ 📅 2026-02-06 14:32:15  │  🔄 Regenerated: 0  │  💬 Feedback: ⭐⭐⭐⭐⭐   │   ║  │
│  ║  └────────────────────────────────────────────────────────────────────────┘   ║  │
│  ╚═══════════════════════════════════════════════════════════════════════════════╝  │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Block 数据结构

### AIBlock 完整字段

```go
// AIBlock - 统一块模型
type AIBlock struct {
    // === Identity ===
    ID              int64     `json:"id"`              // 主键
    UID             string    `json:"uid"`             // 唯一标识符 (UUID)
    ConversationID  int64     `json:"conversationId"`  // 所属会话
    RoundNumber     int       `json:"roundNumber"`     // 会话内轮次 (从1开始)

    // === Mode & Type ===
    Mode         BlockMode  `json:"mode"`         // normal | geek | evolution
    BlockType    BlockType  `json:"blockType"`    // message | context_separator
    Status       BlockStatus `json:"status"`      // pending | streaming | completed | error

    // === Content ===
    UserInputs          []UserInput  `json:"userInputs"`          // 用户输入数组
    AssistantContent    string       `json:"assistantContent"`    // AI 回复内容
    AssistantTimestamp int64        `json:"assistantTimestamp"` // 回复时间戳

    // === Event Stream (流式事件历史) ===
    EventStream []Event `json:"eventStream"` // thinking | tool_use | tool_result | answer

    // === Token Usage (JSONB) ===
    TokenUsage *TokenUsage `json:"tokenUsage,omitempty"` // Token 使用统计

    // === Session Statistics (JSONB) ===
    SessionStats string `json:"sessionStats,omitempty"` // 会话统计 JSON

    // === Cost & Model ===
    CostEstimate  int64  `json:"costEstimate,omitempty"`  // 成本预估 (milli-cents)
    ModelVersion  string `json:"modelVersion,omitempty"`  // LLM 提供商/模型

    // === Branching (树分支) ===
    ParentBlockID int64  `json:"parentBlockId,omitempty"` // 父块 ID (根节点为 0)
    BranchPath    string `json:"branchPath,omitempty"`    // 分支路径 (如 "0/1/2")

    // === User Feedback ===
    UserFeedback      string `json:"userFeedback,omitempty"`      // 用户反馈
    RegenerationCount int    `json:"regenerationCount,omitempty"` // 重新生成次数
    ErrorMessage      string `json:"errorMessage,omitempty"`      // 错误信息

    // === Metadata ===
    Metadata   map[string]string `json:"metadata,omitempty"`   // 扩展元数据
    CreatedTs  int64            `json:"createdTs"`           // 创建时间
    UpdatedTs  int64            `json:"updatedTs"`           // 更新时间
    ArchivedAt int64            `json:"archivedAt,omitempty"` // 归档时间

    // === Geek Mode 专属 ===
    CCSessionID string `json:"ccSessionId,omitempty"` // CC Runner 会话 ID
}
```

### TokenUsage 结构

```go
// TokenUsage - Token 使用统计
type TokenUsage struct {
    PromptTokens     int32 `json:"promptTokens"`     // 输入 tokens
    CompletionTokens int32 `json:"completionTokens"` // 输出 tokens
    TotalTokens       int32 `json:"totalTokens"`       // 总 tokens
    CacheReadTokens   int32 `json:"cacheReadTokens"`   // 缓存命中 tokens
    CacheWriteTokens  int32 `json:"cacheWriteTokens"`  // 缓存写入 tokens
}
```

### SessionStats 结构

```go
// SessionStats - 会话统计 (JSONB)
type SessionStats struct {
    // LLM 调用记录
    LLMCalls []LLMCallStats `json:"llmCalls,omitempty"`

    // 汇总统计
    TotalTokens       int32  `json:"totalTokens,omitempty"`        // 总 tokens
    TotalCostEstimate int64  `json:"totalCostEstimate,omitempty"` // 成本预估
    TotalDurationMs   int64  `json:"totalDurationMs,omitempty"`    // 总耗时

    // 工具统计
    ToolCalls    int     `json:"toolCalls,omitempty"`    // 工具调用次数
    CacheHitRate float64 `json:"cacheHitRate,omitempty"` // 缓存命中率
}
```

---

## UI 组件层级

```
AIChat.tsx (页面)
│
├── AIChatSidebar (侧边栏)
│   ├── ConversationList (会话列表)
│   └── NewConversationButton
│
├── ChatMessages (消息区域)
│   └── UnifiedMessageBlock[] ←─── 递归渲染分支
│       │
│       ├── BlockHeader (头部)
│       │   ├── BranchIndicator (🌿 分支指示器)
│       │   ├── ParrotAvatar (🤖 鹦鹉头像)
│       │   ├── TokenUsageBadge (🪙 Token 徽章)
│       │   ├── BlockCostBadge (💰 成本徽章)
│       │   └── BlockActions (操作按钮)
│       │
│       ├── BlockBody (主体 - 可折叠)
│       │   ├── UserInputsDisplay (用户输入显示)
│       │   ├── AssistantContent (AI 回复)
│       │   ├── ToolCallsSection (工具调用)
│       │   │   └── ToolCallCard[]
│       │   └── SessionSummaryPanel (会话统计面板)
│       │       ├── LLMStatsCard (LLM 统计)
│       │       ├── TokenUsageBreakdown (Token 明细)
│       │       ├── TimingBreakdown (时间明细)
│       │       └── CostBreakdown (成本明细)
│       │
│       └── BlockFooter (底部)
│           ├── Timestamp (时间戳)
│           ├── RegenerationCount (重新生成次数)
│           └── UserFeedback (用户反馈)
│
├── BranchSelectorDialog (分支选择器 - 模态框)
│   └── BranchTree (递归树形结构)
│       └── BranchTreeNode[]
│
├── ChatInput (输入框)
│   ├── ModeSwitcher (模式切换)
│   ├── TextArea (文本输入)
│   ├── SendButton (发送按钮)
│   └── InputHints (输入提示)
│
└── SessionSummaryBar (会话汇总栏)
    ├── TotalTokens (总 Token)
    ├── TotalCost (总成本)
    └── TotalDuration (总耗时)
```

---

## Block 状态流转

```
┌─────────────┐     create      ┌─────────────┐     stream     ┌─────────────┐
│   (不存在)    │ ──────────────► │   pending    │ ─────────────► │  streaming   │
└─────────────┘                  └─────────────┘                  └─────────────┘
     ▲                                   │                                │
     │                                   │ error/cancel                   │
     │           ┌───────────────────────┼──────────────────────┐    │
     │           ▼                       ▼                      ▼    ▼
     │     ┌─────────────┐         ┌─────────────┐       ┌─────────────┐
     └─────│  completed  │◄──────── │    error    │       │   archived  │
           └─────────────┘  fork   └─────────────┘       └─────────────┘
                  │
                  │ regenerate
                  ▼
           ┌─────────────┐
           │   pending    │ (regeneration_count++)
           └─────────────┘
```

**状态说明**:

| 状态 | 描述 | UI 表现 |
|:-----|:-----|:--------|
| `pending` | Block 已创建，等待处理 | 显示加载动画 |
| `streaming` | 流式生成中 | 实时更新内容，显示事件流 |
| `completed` | 完成 | 显示完整内容和统计 |
| `error` | 出错 | 显示错误信息，支持重试 |

---

## 分支结构

### 树形结构

```
                    Block #1 (root - branch_path: "")
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
    Block #2A         Block #2B         Block #2C
  (path: "0")        (path: "1")        (path: "2")
    [ACTIVE]           │                  │
        │         ┌─────┴─────┐             │
    Block #3A     Block #3B   Block #3C  Block #3D
  (path: "0/0")   (path: "1/0")(path: "1/1")(path: "2/0")
    [ACTIVE]
```

### 分支路径计算规则

```
branch_path 格式: "0/1/2"
  - 根节点: "" (空字符串)
  - 第一层: "0", "1", "2", ... (按创建顺序)
  - 第二层: "0/0", "0/1", "1/0", "1/1", ...
  - 第 N 层: 父路径 + "/" + 子序号

ForkBlock 操作:
  1. 查询 parent_block_id 的所有直接子节点
  2. 获取当前最大 branch_path 序号
  3. 新块路径 = parent_path + "/" + (max序号 + 1)
```

### 分支操作

| 操作 | RPC 方法 | 效果 |
|:-----|:---------|:-----|
| **Fork** | `ForkBlock` | 创建新分支，继承父块上下文 |
| **Switch** | `SwitchBranch` | 切换活动分支，归档其他分支 |
| **Delete** | `DeleteBranch` | 删除分支及其子树 (cascade) |

---

## 成本计算

### 计价规则 (DeepSeek V3)

| Token 类型 | 单价 | milli-cents/token |
|:-----------|:-----|:------------------|
| Input | ¥0.14/M tokens | 0.14 |
| Output | ¥0.28/M tokens | 0.28 |
| Cache Read | 90% off | 0.014 |
| Cache Write | 正常计费 | 0.14 |

### 计算示例

```
Block #3 成本计算:
┌─────────────────────────────────────────────────────────────┐
│ prompt_tokens:     856   × 0.14 = 119.84 milli-cents        │
│ completion_tokens: 423   × 0.28 = 118.44 milli-cents        │
│ cache_read_tokens:  142   × 0.014 = 1.988 milli-cents (90% off)│
│ cache_write_tokens: 28   × 0.014 = 0.392 milli-cents        │
├─────────────────────────────────────────────────────────────┤
│ cost_estimate:     870n  ≈ $0.0087 (0.87 cents)           │
└─────────────────────────────────────────────────────────────┘

存储: cost_estimate = 870 (milli-cents, BigInt)
显示: "0.87¢" (cents) 或 "$0.0087" (dollars)
```

### 成本徽章颜色

| 成本范围 | 颜色 | CSS 类 |
|:---------|:-----|:--------|
| < $0.01 | 翠绿 | `emerald-100` |
| $0.01 - $0.10 | 绿色 | `green-100` |
| $0.10 - $1.00 | 琥珀 | `amber-100` |
| ≥ $1.00 | 橙色 | `orange-100` |

---

## 模式对比

| 特性 | Normal Mode | Geek Mode | Evolution Mode |
|:-----|:------------|:----------|:---------------|
| **Agent** | AUTO → 路由选择 | GeekParrot | EvolutionParrot |
| **LLM** | DeepSeek V3 | Claude Code CLI | Claude Code CLI |
| **用途** | 日常对话/搜索 | 代码执行 | 系统进化 |
| **成本** | 按 Token 计费 | 零 LLM 成本 | 零 LLM 成本 |
| **产出** | 对话回复 | 代码产物 | GitHub PR |
| **权限** | 所有用户 | 所有用户 | 仅管理员 |

---

## 相关文档

| 文档 | 描述 |
|:-----|:-----|
| [Unified Block Model 规格](../specs/unified-block-model.md) | Block 数据模型详细规格 |
| [架构文档](ARCHITECTURE.md) | 系统整体架构 |
| [前端开发指南](FRONTEND.md) | 前端组件和布局 |
| [LLM Stats Collection](../specs/block-design/llm-stats-collection.md) | Token 统计实现 |

---

## 设计考虑与待讨论问题

### 1. Block 折叠/展开状态持久化

**问题**：用户手动展开的历史 Block，在发送新消息后是否保持展开状态？

**当前行为**：
- 新/最新 Block：默认展开
- 历史 Block：默认折叠

**考虑方案**：
- 在本地存储记录 `expandedBlockIds: Set<number>`
- 或使用 `sessionStorage` 仅当前会话有效
- 或保持简单：始终基于"最新 N 个"规则

---

### 2. 分支切换 UX 流程

**问题**：切换分支时的过渡体验？

**选项**：
| 方案 | 描述 | 优缺点 |
|:-----|:-----|:-------|
| 立即刷新 | 直接切换内容 | 快速，但可能有跳跃感 |
| 平滑过渡 | 淡入淡出动画 | 体验流畅，但增加复杂度 |
| 位置保留 | 保持滚动位置 | 上下文连续，但实现复杂 |

---

### 3. TokenUsageBadge 展开方向

**问题**：展开内容在视口边缘时可能被截断

**当前实现**：绝对定位向下弹出

**改进方向**：
- 智能方向检测（Flip 技术）
- Portal 渲染到 `document.body`
- 限制同时只允许一个展开

---

### 4. 成本告警阈值

**问题**：是否需要会话预算控制和告警？

**考虑功能**：
- 会话级预算设置（如 $1.00）
- 实时阈值警告（80%、90%、100%）
- 达到预算自动停止（需确认）

**相关字段**：已在 Block 中包含 `cost_estimate`，后端支持就绪

---

### 5. 流式更新频率

**问题**：流式事件更新频率与性能平衡

**当前**：每个事件都触发 React 更新

**考虑**：
- 节流更新（如每 100ms 批处理）
- 使用 `useTransition` 标记非紧急更新
- 虚拟滚动优化长对话

---

### 6. 分支删除确认

**问题**：删除分支时是否需要确认对话框？

**考虑**：
- 有子节点的分支：强制确认
- 空分支（无子节点）：可直接删除
- 提供 `Undo` 功能

---

### 7. 用户反馈收集

**问题**：UserFeedback 字段的 UI 交互

**当前状态**：字段已存在，UI 待实现

**考虑形式**：
- 简单：👍👎 点赞/点踩
- 详细：5 星评分 + 文本评论
- 隐式：仅根据重新生成行为推断

---
