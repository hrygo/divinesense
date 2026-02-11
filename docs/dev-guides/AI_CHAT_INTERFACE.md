# AI Chat 界面架构

> **保鲜状态**: ✅ 已验证 (2026-02-11) | **最后检查**: v0.97.0 (Anthropic 默认 LLM)
> **关联规格**: [Unified Block Model](../specs/block-design/unified-block-model.md) | [P1-A006](../specs/block-design/P1-A006-llm-stats-collection.md)

## 概述

AI Chat 界面采用 **Unified Block Model（统一块模型）** 设计，将用户输入与 AI 回复封装为一个完整的可折叠 Block。每个 Block 包含完整的对话上下文、Token 统计、成本追踪和分支信息。

## v0.97.0 更新内容

- **BranchIndicator 友好编号**: 内部路径 "0/1/2" 转换为 "A.2.3" 显示
- **SessionStats 持久化**: 刷新后从 Block.sessionStats 自动恢复
- **SessionBar 整合**: PC 端 SessionStats 整合到 ChatHeader，移动端保留独立 SessionBar
- **Mode 差异化展示**: Normal (Tokens/Cost)、Geek (Time/Tools)、Evolution (Time/Files)

---

## 组件级架构

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                    DivineSense AI Chat 界面架构 (v0.97.0)                           │
└─────────────────────────────────────────────────────────────────────────────────────┘

AIChat.tsx (/chat)
│
├── UnifiedChatView (主视图)
│   │
│   ├── ChatHeader (桌面端 - lg:breakpoint)
│   │   ├── AnimatedAvatar (头像 + 思考动画)
│   │   ├── ActionDescription (状态文本)
│   │   ├── HeaderSessionStats (会话统计)
│   │   └── ImmersiveToggle (全屏切换)
│   │
│   ├── SessionBar (移动端 - lg:hidden)
│   │   ├── CollapseToggle (折叠/展开)
│   │   └── StatsDisplay (Cost/Tokens/Duration)
│   │
│   ├── ChatMessages (消息流 - 核心容器)
│   │   │
│   │   ├── PartnerGreeting (空状态欢迎界面)
│   │   │   ├── AnimatedAvatar (大头像)
│   │   │   ├── GreetingText (时间感知问候)
│   │   │   └── SuggestedPrompts (2x4 示例问题网格)
│   │   │
│   │   ├── UnifiedMessageBlock[] (消息块数组)
│   │   │   ├── BlockHeader (用户消息预览 + 时间 + 状态)
│   │   │   ├── BlockBody (Timeline 布局 - 可折叠)
│   │   │   │   ├── UserInputsSection (多输入展示)
│   │   │   │   ├── ThinkingSection (思考过程)
│   │   │   │   ├── ToolCallsSection (工具调用卡片)
│   │   │   │   ├── AnswerSection (Markdown 内容)
│   │   │   │   ├── ErrorSection (错误展示)
│   │   │   │   └── BlockSummarySection (会话统计)
│   │   │   └── BlockFooter (操作栏)
│   │   │
│   │   ├── AmazingInsightCard (综合模式专属)
│   │   └── BlockEditDialog (编辑弹窗)
│   │
│   └── ChatInput (输入区)
│       ├── Toolbar (新对话/清空/清上下文 + 快捷键提示)
│       ├── ModeCycleButton (模式切换)
│       └── TextArea + SendButton (自适应高度)
│
├── CapabilityPanelView (鹦鹉选择面板 - 条件渲染)
│   ├── MobileSubHeader (返回按钮)
│   └── ParrotHub (灰灰/时巧/折衷/极客/进化 卡片)
│
└── ConfirmDialog (清空聊天确认)

┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              响应式断点                                            │
└─────────────────────────────────────────────────────────────────────────────────────┘

           │   sm(640px)   │   md(768px)   │   lg(1024px)  │   xl(1280px)
───────────┼───────────────┼───────────────┼───────────────┼──────────────
ChatHeader    隐藏          隐藏           显示           显示
SessionBar    显示          显示           隐藏           隐藏
MessageWidth 100%           100%           max-w-3xl       max-w-4xl

┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Block 主题配置                                        │
└─────────────────────────────────────────────────────────────────────────────────────┘

NORMAL (MEMO/SCHEDULE/AMAZING)        GEEK (Claude Code)       EVOLUTION (系统进化)
├── border: amber-200/700            ├── border: sky-200/700       ├── border: emerald-200/700
├── headerBg: amber-50/900/20        ├── headerBg: sky-50/900/20    ├── headerBg: emerald-50/900/20
├── footerBg: amber-200/800/50       ├── footerBg: sky-200/800/50   ├── footerBg: emerald-200/800/50
└── ringColor: amber-500/20          └── ringColor: sky-500/20      └── ringColor: emerald-500/20
```

---

## UnifiedMessageBlock 详解

UnifiedMessageBlock 是 AI Chat 的核心消息容器，采用 **Warp Block** 风格设计。

### 结构图

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                         UnifiedMessageBlock 组件结构                                │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────────────────────────────────────────┐
│  ╔═══════════════════════════════════════════════════════════════════════════════╗  │
│  ║  UnifiedMessageBlock (memo) - 可折叠消息块                                    ║  │
│  ║  ┌─────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │  BlockHeader (固定 - 点击切换折叠)                                    │   ║  │
│  ║  │  ┌─────────────────────┬───────────────────────────────────────────┐   │   ║  │
│  ║  │  │ Left                 │ Right                                    │   │   ║  │
│  ║  │  ├─────────────────────┼───────────────────────────────────────────┤   │   ║  │
│  ║  │  │ 👤 UserAvatar       │ 📊 ModeSummary (Desktop/Mobile 差异)      │   │   ║  │
│  ║  │  │    [+N Badge]        │   Normal: Tokens + Time                   │   │   ║  │
│  ║  │  │ 📝 InputPreview     │   Geek: Time + Tools                      │   │   ║  │
│  ║  │  │    (视觉宽度截取)    │   Evolution: Time + Files                  │   │   ║  │
│  ║  │  │                     │ ⏰ Timestamp                             │   │   ║  │
│  ║  │  │                     │ 🏷️ ParrotBadge (NORMAL/GEEK/EVOLUTION) │   │   ║  │
│  ║  │  │                     │ 🌿 BranchIndicator ("A.2" 或 "3 branches")│   │   ║  │
│  ║  │  │                     │ ▼ CollapseToggle                          │   │   ║  │
│  ║  │  └─────────────────────┴───────────────────────────────────────────┘   │   ║  │
│  ║  └─────────────────────────────────────────────────────────────────────────┘   ║  │
│  ╠═══════════════════════════════════════════════════════════════════════════════╗  │
│  ║  ┌─────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │  BlockBody (可折叠 - Timeline 布局)                                    │   ║  │
│  ║  │  ┌─────────────────────────────────────────────────────────────────┐   │   ║  │
│  ║  │  │  Timeline Line (absolute left-[11px]) - 连接各事件节点        │   │   ║  │
│  ║  │  │  └─────────────────────────────────────────────────────────────────┘   │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  Timeline Nodes (按事件顺序排列):                                   │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  1️⃣ UserInputsSection                                                    │   ║  │
│  ║  │     ├── Node: blue-100 User Icon                                     │   ║  │
│  ║  │     ├── InputCard[] (多输入展示，带序号)                             │   ║  │
│  ║  │     └── ExpandButton (长内容自动折叠)                                │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  2️⃣ ThinkingSection (条件渲染)                                          │   ║  │
│  ║  │     ├── Node: Brain Icon / Loader2 (animated)                       │   ║  │
│  ║  │     ├── Toggle Button (foldable)                                     │   ║  │
│  ║  │     └── ReactMarkdown Content (prose-xs)                             │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  3️⃣ ToolCallsSection (条件渲染)                                         │   ║  │
│  ║  │     ┌─────────────────────────────────────────────────────────────┐   │   ║  │
│  ║  │     │  Node: Wrench Icon / Loader2 (purple, animated)            │   │   ║  │
│  ║  │     │  ToolCallCard[]                                               │   │   ║  │
│  ║  │     │  ├── ToolName + Status (running/done/error/pending)      │   │   ║  │
│  ║  │     │  ├── Duration (font-mono)                                   │   │   ║  │
│  ║  │     │  ├── InputSummary (truncate)                                │   │   ║  │
│  ║  │     │  └── OutputSummary (details - 可展开)                        │   │   ║  │
│  ║  │     │      └── isError: 红色背景                                    │   │   ║  │
│  ║  │     └─────────────────────────────────────────────────────────────┘   │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  4️⃣ AnswerSection (条件: hasAnswer)                                    │   ║  │
│  ║  │     ├── Node: Zap Icon / Loader2 (amber, animated)                  │   ║  │
│  ║  │     └── MessageBubble (themeColors.bubbleBg)                         │   ║  │
│  ║  │         └── ReactMarkdown (prose-sm + 代码高亮)                      │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  5️⃣ ErrorSection (条件: hasError)                                      │   ║  │
│  ║  │     ├── Node: AlertCircle (red-100)                                  │   ║  │
│  ║  │     └── ErrorCard (red-50 bg + error message)                         │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  6️⃣ BlockSummarySection (条件: blockSummary)                            │   ║  │
│  ║  │     ├── Node: BarChart3 (green-100)                                  │   ║  │
│  ║  │     └── ExpandedSessionSummary                                       │   ║  │
│  ║  │         ├── SessionInfo (sessionId, duration)                        │   ║  │
│  ║  │         ├── TokenStats (input/output/cache/total)                    │   ║  │
│  ║  │         ├── CostEstimate (totalCostUSD)                              │   ║  │
│  ║  │         ├── ToolUsage (toolCallCount, toolsUsed[])                    │   ║  │
│  ║  │         ├── FileChanges (filesModified, filePaths[])                  │   ║  │
│  ║  │         └── StatusIndicator (success/error)                          │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  ⏳ Pending State (isLatest && !hasAnswer)                              │   ║  │
│  ║  │     └── Loader2 + "初始化中..."                                       │   ║  │
│  ║  │                                                                     │   ║  │
│  ║  │  📍 TypingCursor (children prop - 条件渲染)                          │   ║  │
│  ║  │     └── scale-90 origin-left (流式输入指示器)                         │   ║  │
│  ║  └─────────────────────────────────────────────────────────────────────────┘   ║  │
│  ╠═══════════════════════════════════════════════════════════════════════════════╗  │
│  ║  ┌─────────────────────────────────────────────────────────────────────────┐   ║  │
│  ║  │  BlockFooter (固定显示)                                                │   ║  │
│  ║  │  ┌─────────────────────┬───────────────────────────────────────────┐   │   ║  │
│  ║  │  │ Left                 │ Right                                    │   │   ║  │
│  ║  │  ├─────────────────────┼───────────────────────────────────────────┤   │   ║  │
│  ║  │  │ ▼ CollapseToggle     │ EditButton (disabled: isStreaming)       │   │   ║  │
│  ║  │  │   "收起"/"展开"       │ RegenerateButton (条件: isLatest)         │   │   ║  │
│  ║  │  │                     │ ForgetButton (上下文遗忘)                    │   │   ║  │
│  ║  │  │                     │ CopyButton (Check/Copy Icon + 状态反馈)     │   │   ║  │
│  ║  │  │                     │ DeleteButton (red text)                    │   │   ║  │
│  ║  │  └─────────────────────┴───────────────────────────────────────────┘   │   ║  │
│  ║  └─────────────────────────────────────────────────────────────────────────┘   ║  │
│  ╚═══════════════════════════════════════════════════════════════════════════════╝  │
│                                                                               │
│  ╔═══════════════════════════════════════════════════════════════════════════════╗  │
│  ║  ACTIVE/STREAMING STATES (边框动效)                                        ║  │
│  ║  • isLatest && isStreaming → ring-2 + animate-block-pulse                   ║  │
│  ║  • isLatest && !isStreaming → ring-1                                       ║  │
│  ║  • BlockHeader Status Bubbling: streaming(blue) / error(red)               ║  │
│  ╚═══════════════════════════════════════════════════════════════════════════════╝  │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### 折叠策略

| 状态             | 默认展开 | 说明                 |
| :--------------- | :------- | :------------------- |
| **最新 Block**   | ✅        | `isLatest = true`    |
| **流式中 Block** | ✅        | `isStreaming = true` |
| **历史 Block**   | ❌        | 默认折叠             |

### 状态颜色

```
Timeline Node 颜色:
├── UserInputs:   blue-100/blue-500   (用户)
├── Thinking:     blue-100/brain        (思考中: 蓝色)
├── ToolCalling:  purple-100/purple-500 (工具调用: 紫色)
├── Answer:       amber-100/amber-500   (AI 回复: 琥珀色)
├── Error:        red-100/red-500       (错误: 红色)
└── Summary:      green-100/green-500   (统计: 绿色)
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
    BranchPath    string `json:"branchPath,omitempty"`    // 内部分支路径 (如 "0/1/2"，UI 显示为 "A.2.3")

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
// SessionStats - 会话统计 (持久化到 Block.sessionStats)
type SessionStats struct {
    // 会话标识
    SessionID       string `json:"sessionId"`       // CC Session ID (Geek/Evolution)
    ConversationID  int64  `json:"conversationId"`  // 所属会话
    AgentType       string `json:"agentType"`       // 代理类型

    // 时间统计 (bigint, 毫秒)
    TotalDurationMs      int64 `json:"totalDurationMs"`
    ThinkingDurationMs   int64 `json:"thinkingDurationMs"`
    ToolDurationMs       int64 `json:"toolDurationMs"`
    GenerationDurationMs int64 `json:"generationDurationMs"`

    // Token 统计
    InputTokens       int32 `json:"inputTokens"`
    OutputTokens      int32 `json:"outputTokens"`
    TotalTokens       int32 `json:"totalTokens"`
    CacheReadTokens   int32 `json:"cacheReadTokens"`
    CacheWriteTokens  int32 `json:"cacheWriteTokens"`

    // 成本统计
    TotalCostUsd float64 `json:"totalCostUsd"`

    // 工具统计 (Geek/Evolution)
    ToolCallCount int32    `json:"toolCallCount"`
    ToolsUsed    []string `json:"toolsUsed"`
    FilesModified int32    `json:"filesModified"`
    FilePaths    []string `json:"filePaths"`

    // 状态
    IsError      bool   `json:"isError"`
    ErrorMessage string `json:"errorMessage"`
}
```

---

## Block 状态流转

```
┌─────────────┐     create      ┌─────────────┐     stream     ┌─────────────┐
│   (不存在)    │ ──────────────► │   pending    │ ─────────────► │  streaming   │
└─────────────┘                  └─────────────┘                  └─────────────┘
     ▲                                   │                                │
     │                                   │ error/cancel                   │
     │           ┌───────────────────────┼───────────────────────┐    │
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

| 状态        | 描述                   | UI 表现                  |
| :---------- | :--------------------- | :----------------------- |
| `pending`   | Block 已创建，等待处理 | 显示加载动画             |
| `streaming` | 流式生成中             | 实时更新内容，显示事件流 |
| `completed` | 完成                   | 显示完整内容和统计       |
| `error`     | 出错                   | 显示错误信息，支持重试   |

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

**内部分支路径格式** (数据库存储):
```
branch_path 格式: "0/1/2"
  - 根节点: "" (空字符串)
  - 第一层: "0", "1", "2", ... (按创建顺序)
  - 第二层: "0/0", "0/1", "1/0", "1/1", ...
  - 第 N 层: 父路径 + "/" + 子序号
```

**友好编号显示格式** (UI 显示):
```
格式: "A.1.2"
  - 第一级: 0→A, 1→B, ..., 25→Z, 26→AA, 27→AB, ...
  - 其他级别: 0→1, 1→2, 2→3, ... (0-based 转 1-based)

转换示例:
  "0"       → "A"
  "1"       → "B"
  "0/0"     → "A.1"
  "0/1"     → "A.2"
  "1/0/1"   → "B.1.2"
  "26/0"    → "AA.1"
```

### 分支操作

| 操作       | RPC 方法       | 效果                       |
| :--------- | :------------- | :------------------------- |
| **Fork**   | `ForkBlock`    | 创建新分支，继承父块上下文 |
| **Switch** | `SwitchBranch` | 切换活动分支，归档其他分支 |
| **Delete** | `DeleteBranch` | 删除分支及其子树 (cascade) |

---

## 成本计算

### 计价规则 (Anthropic Claude Opus 4.6)

| Token 类型  | 单价           | milli-cents/token |
| :---------- | :------------- | :---------------- |
| Input       | ¥15.0/M tokens | 15.0              |
| Output      | ¥75.0/M tokens | 75.0              |
| Cache Read  | 90% off        | 1.5               |
| Cache Write | 正常计费       | 15.0              |

**说明**：智谱 Z.AI 提供的 Claude 服务定价可能与官方不同，请参考 https://open.bigmodel.cn/pricing 获取最新价格。

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

| 成本范围      | 颜色 | CSS 类        |
| :------------ | :--- | :------------ |
| < $0.01       | 翠绿 | `emerald-100` |
| $0.01 - $0.10 | 绿色 | `green-100`   |
| $0.10 - $1.00 | 琥珀 | `amber-100`   |
| ≥ $1.00       | 橙色 | `orange-100`  |

---

## 模式对比

| 特性      | Normal Mode         | Geek Mode       | Evolution Mode  |
| :-------- | :------------------ | :-------------- | :-------------- |
| **Agent** | AUTO → 路由选择     | GeekParrot      | EvolutionParrot |
| **LLM**   | Anthropic Claude    | Claude Code CLI | Claude Code CLI |
| **用途**  | 日常对话/搜索       | 代码执行        | 系统进化        |
| **成本**  | 按 Token 计费       | 零 LLM 成本     | 零 LLM 成本     |
| **产出**  | 对话回复            | 代码产物        | GitHub PR       |
| **权限**  | 所有用户            | 所有用户        | 仅管理员        |

---

## 相关文档

| 文档                                                                          | 描述                   |
| :---------------------------------------------------------------------------- | :--------------------- |
| [Unified Block Model 规格](../specs/block-design/unified-block-model.md)      | Block 数据模型详细规格 |
| [架构文档](ARCHITECTURE.md)                                               | 系统整体架构           |
| [前端开发指南](FRONTEND.md)                                               | 前端组件和布局         |
| [LLM Stats Collection](../specs/block-design/P1-A006-llm-stats-collection.md) | Token 统计实现         |
