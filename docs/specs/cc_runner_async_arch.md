# CC Runner 异步架构规格说明书 (Async Architecture Spec)

**Status**: ✅ Published (Updated 2026-02-03)
**Version**: 1.3
**Context**: [Research: CC Runner Async Upgrade](../research/cc-runner-async-upgrade.md)
**Latest Research**: [CCRunner 消息处理机制调研](../research/cc-runner-message-handling-research.md)

---

## 1. 概述 (Overview)

本规格说明书详细定义了 `cc_runner` 从一次性执行（One-shot）向全双工异步持久化（Full-Duplex Persistent）架构演进的技术标准。

### 1.1 核心目标
- **持久化会话**: 保持 Claude Code CLI 进程存活，避免重复启动开销。
- **全双工交互**: 支持在执行过程中随时注入用户反馈 (Human-in-the-loop)。
- **实时流式**: 提供毫秒级的 Token 级输出和工具执行状态更新。
- **统计追踪**: 自动提取并展示会话统计数据（成本、token、耗时）。

### 1.2 更新日志 (v1.3)
- ✅ 添加 `session_stats` 事件类型用于会话完成统计
- ✅ 实现 `result` 消息的统计提取（耗时、成本、token）
- ✅ 消除 "unknown message type" 日志警告
- ✅ 前端可通过 `SessionSummary.total_cost_usd` 获取成本

---

## 2. 系统架构 (System Architecture)

```mermaid
flowchart TB
    %% Standard Shapes:
    %% [] : Component / Service
    %% [()] : Storage / State
    %% {{}} : Interface / Protocol

    subgraph ClientLayer ["Frontend Layer"]
        UI["💻 Web UI (React)<br/>[ConversationID]"]
        Stats["📊 Session Stats Panel<br/>(Cost, Tokens, Duration)"]
    end

    subgraph Transport ["Communication"]
        WS{{"📡 WebSocket Stream"}}
    end

    subgraph Backend ["DivineSense Backend (Go)"]
        direction TB
        Svc["⚙️ Agent Coordination Service"]
        Map{{"🔗 UUID v5 Generator<br/>(Hash Mapping)"}}

        subgraph SessionManager ["Session Manager"]
            Reg[("🗂️ Session Registry<br/>(map[UUID]Session)")]
            Life["⏲️ Lifecycle Controller"]
        end
    end

    subgraph AsyncCore ["Async Core (1:N)"]
        direction TB

        subgraph SessionInstance ["Session Unit (Instance n)"]
            Stream[["🔄 Bi-directional Streamer"]]
            Stats[["📊 Session Stats Collector"]]
            Pipes{{"🚇 Stdin/Stdout Pipes"}}

            subgraph Process ["Claude Code v2.x"]
                CLI["🧠 CLI Engine<br/>--session-id UUID"]
                Cache[("📝 In-Memory Context")]
                Skills["🛠️ Skills & MCP Registry"]
            end

            %% Detailed IO Flow inside Instance
            Stream <-->|"Full-Duplex"| Pipes
            Pipes <-->|"JSON Stream"| CLI
            CLI -.->|"result msg"| Stats
        end
    end

    subgraph OS ["System Environment"]
        FS[("📂 Filesystem / Persistence<br/>(~/.claude/sessions)"]
        Shell["🐚 System Shell"]
    end

    %% Connections
    UI -- "1. Sends Msg with ConversationID" --> WS
    WS --> Svc
    Svc -- "2. Hash to UUID" --> Map
    Map -- "3. Lookup/Create by UUID" --> Reg
    Reg -- "1:1 Binding" --> SessionInstance
    CLI -- "4. Resume from Disk" --> FS
    Stats --> Stats

    %% Styling (AI Native Light Theme)
    classDef client fill:#e1f5fe,stroke:#03a9f4,stroke-width:2px;
    classDef backend fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px;
    classDef core fill:#e0f2f1,stroke:#009688,stroke-width:2px;
    classDef storage fill:#fffde7,stroke:#fbc02d,stroke-width:1.5px;
    classDef os fill:#fafafa,stroke:#9e9e9e,stroke-width:1.5px,stroke-dasharray: 3 3;

    %% Container Styles (Clean Modern)
    classDef subOuter fill:#f8faff,stroke:#d1d9e6,stroke-width:1px;
    classDef subInner fill:#ffffff,stroke:#c2cfe0,stroke-width:1px;

    class UI,Stats client;
    class Svc,Map,Life backend;
    class Stream,CLI,Skills,Pipes,Stats core;
    class Reg,Cache,FS storage;
    class Shell os;

    class ClientLayer,Transport,Backend,AsyncCore,OS subOuter;
    class SessionManager,SessionInstance,Process subInner;
```

---

## 3. CLI 事件类型 (CLI Event Types)

### 3.1 完整事件类型映射

基于实际 CLI 验证（v2.1.15），完整的事件类型映射如下：

| CLI 消息类型 | 后端处理 | 前端展示 | 用途 |
|:------------|:--------|:--------|:-----|
| `system` | ✅ 静默处理 | - | 会话初始化配置（工具列表、MCP 状态） |
| `thinking` | ✅ | ✅ | AI 思考过程 |
| `status` | ✅ (复用 thinking) | ✅ | 状态更新（复用 thinking 处理） |
| `tool_use` | ✅ (含嵌套) | ✅ | 工具调用（可能嵌套在 assistant 中） |
| `tool_result` | ✅ (含嵌套) | ✅ | 工具结果（可能嵌套在 user 中） |
| `assistant` | ✅ (展开嵌套) | ✅ | AI 响应（可能含嵌套 tool_use） |
| `user` | ✅ (展开嵌套) | ✅ | 用户消息（可能含嵌套 tool_result） |
| `answer` | ✅ | ✅ | 最终回答 |
| `error` | ✅ | ✅ | 系统级错误 |
| **`result`** | **✅ 提取统计** | **✅ (在 SessionSummary)** | **会话完成统计** |
| `session_stats` | ✅ | ✅ (在 SessionSummary) | 会话统计数据（前端未直接展示） |

### 3.2 特殊消息类型说明

#### System 消息
```json
{
  "type": "system",
  "subtype": "init",
  "cwd": "/path/to/workdir",
  "session_id": "uuid",
  "tools": ["Task", "Bash", ...],
  "mcp_servers": [...],
  "model": "claude-opus-4.5-20251101",
  "claude_code_version": "2.1.15"
}
```
- **处理方式**: 静默接收，记录 Debug 日志
- **无需前端显示**: 纯控制层面元数据

#### Result 消息
```json
{
  "type": "result",
  "subtype": "success",
  "duration_ms": 6310,
  "total_cost_usd": 0.318836,
  "usage": {
    "input_tokens": 63586,
    "output_tokens": 26,
    "cache_read_input_tokens": 512
  },
  "num_turns": 1
}
```
- **处理方式**: `handleResultMessage()` 提取统计，发送 `session_stats` 事件
- **前端展示**: 通过 `SessionSummary.total_cost_usd` 获取

---

## 4. 会话隔离与连续性 (Session Model)

### 4.1 隔离性 (Isolation)

- **1:N 管理模型**: 系统维护一个单例的 `Session Manager` (1)，负责协调和路由指令到多个并存的 `Session Units` (N)。
- **物理隔离**: 基于 `SessionID` 进行硬隔离。每个 Session 对应一个独立的 OS 进程 (`exec.Cmd`)，确保进程级别的安全性。
- **资源独立**: 每个进程拥有独立的内存空间（上下文）、IO 管道和文件描述符。
- **互不干扰**: Session A 的环境变更（如 `cd` 切换目录、设置环境变量）仅在其进程内生效，绝不会泄露给 Session B。并发的 Session 可以安全地并行运行。

### 4.2 连续性 (Continuity)

- **进程级保持**: 只要 Session 未被销毁（未达到 30m 空闲超时或被显式 Terminate），底层进程一直保持运行（Running/Sleep）。
- **上下文驻留**: AI 的对话历史（Conversation History）完全保留在 `claude` 进程的内存中。后端 `Session Manager` 无需在应用层序列化/反序列化聊天记录，只需通过管道透传增量数据。
- **多轮交互**: 后续的 WebSocket 消息（如用户并行的追问）直接写入对应进程的 Stdin，无缝延续上下文。

---

## 5. 会话映射模型 (Session Mapping)

前端 UI 的"对话"与后端的"进程会话"之间存在严格的 **1:1 确定性映射**。

- **标识转换**:
    - 前端使用数据库 ID (`ConversationID`) 标识聊天窗口。
    - 后端通过 `UUID v5` 定向哈希算法（以 `ConversationID` 为 Seed）生成符合 Claude Code CLI 要求的 `sessionID` (UUID)。

- **确定性映射 (Deterministic Mapping)**:
    ```
    Map(ConversationID) -> UUID v5(Namespace, "divinesense:conversation:{ID}")
    ```

- **状态恢复 (Resume)**:
    - Claude Code CLI 内部会将对话历史持久化于磁盘。
    - 由于 `sessionID` 恒定且唯一，后端启动 CLI 时带上 `--session-id <UUID>` 即可实现**自动重连与上下文恢复**，无需后端应用层干预。

---

## 6. 消息流转换 (Message Stream Transformation)

### 6.1 CLI 输出解析

```
CLI stdout (JSON Stream)
        │
        ▼
    解析 StreamMessage
        │
        ├──► system ────► [静默处理，Debug 日志]
        ├──► result ────► [提取统计] ──► session_stats 事件
        │
        ▼
   dispatchCallback()
        │
        ├──► thinking ────► thinking 事件
        ├──► assistant ───► [展开嵌套] ──► tool_use / answer 事件
        ├──► user ───────► [展开嵌套] ──► tool_result 事件
        ├──► tool_use ────► tool_use 事件
        ├──► tool_result ─► tool_result 事件
        ├──► answer ─────► answer 事件
        ├──► error ───────► error 事件
        └──► unknown ────► Warn 日志 + 尝试提取文本
```

### 6.2 事件数据结构

```go
// 前端消费的事件
type StreamEvent struct {
    Type      string           `json:"type"`      // thinking, tool_use, tool_result, answer, error
    Content   string           `json:"content"`   // 文本内容
    Meta      *StreamEventMeta `json:"meta"`      // 强类型元数据
    Timestamp int64            `json:"timestamp"`
}

type StreamEventMeta struct {
    ToolName      string `json:"tool_name,omitempty"`
    ToolID        string `json:"tool_id,omitempty"`
    IsError       bool   `json:"is_error,omitempty"`
    FilePath      string `json:"file_path,omitempty"`
    SessionID     string `json:"session_id,omitempty"`
    DurationMs    int64  `json:"duration_ms,omitempty"`
    InputSummary  string `json:"input_summary,omitempty"`
    OutputSummary string `json:"output_summary,omitempty"`
}

// 会话统计数据（后端发送，前端通过 SessionSummary 获取）
type SessionStatsData struct {
    SessionID            string   `json:"session_id"`
    UserID               int32    `json:"user_id"`
    AgentType            string   `json:"agent_type"`      // "geek", "evolution"
    StartTime            int64    `json:"start_time"`      // Unix timestamp
    EndTime              int64    `json:"end_time"`        // Unix timestamp
    TotalDurationMs      int64    `json:"total_duration_ms"`
    ThinkingDurationMs   int64    `json:"thinking_duration_ms"`
    ToolDurationMs       int64    `json:"tool_duration_ms"`
    GenerationDurationMs int64    `json:"generation_duration_ms"`
    InputTokens          int32    `json:"input_tokens"`
    OutputTokens         int32    `json:"output_tokens"`
    CacheWriteTokens     int32    `json:"cache_write_tokens"`
    CacheReadTokens      int32    `json:"cache_read_tokens"`
    TotalTokens          int32    `json:"total_tokens"`
    ToolCallCount        int32    `json:"tool_call_count"`
    ToolsUsed            []string `json:"tools_used"`
    FilesModified        int32    `json:"files_modified"`
    FilePaths            []string `json:"file_paths"`
    TotalCostUSD         float64  `json:"total_cost_usd"`
    ModelUsed            string   `json:"model_used"`
    IsError              bool     `json:"is_error"`
    ErrorMessage         string   `json:"error_message,omitempty"`
}
```

---

## 7. 交互协议 (Interaction Protocol)

### 7.1 WebSocket 消息格式

**Client -> Server:**

| Event Type      | Payload         | Desc         |
| :-------------- | :-------------- | :----------- |
| `session.start` | `{config: ...}` | 启动新会话   |
| `input.send`    | `{text: "yes"}` | 发送用户输入 |
| `session.stop`  | `{}`            | 强制停止     |

**Server -> Client (流式事件):**

| Event Type      | Payload                                      | Desc                   |
| :------------ | :------------------------------------------- | :--------------------- |
| `thinking`      | `{content: "..."}`                           | 思考过程 (增量)        |
| `tool_use`      | `{content: "Name", meta: {name, input, id}}` | 工具调用               |
| `tool_result`   | `{content: "...", meta: {is_error}}`         | 工具结果               |
| `answer`        | `{content: "..."}`                           | 最终回答 (增量)        |
| `error`         | `{content: "..."}`                           | 系统级错误             |
| `session_stats` | *(在 SessionSummary 中)*                    | 会话统计（完成时发送） |

### 7.2 会话完成时的 SessionSummary

```protobuf
message SessionSummary {
  string session_id = 1;
  int64 total_duration_ms = 2;
  int64 thinking_duration_ms = 3;
  int64 tool_duration_ms = 4;
  int64 generation_duration_ms = 5;

  // Token usage
  int32 total_input_tokens = 6;
  int32 total_output_tokens = 7;
  int32 total_cache_write_tokens = 8;
  int32 total_cache_read_tokens = 9;

  // Tool call statistics
  int32 tool_call_count = 10;
  repeated string tools_used = 11;

  // File operations
  int32 files_modified = 12;
  repeated string file_paths = 13;

  // Cost tracking (v1.3 新增)
  double total_cost_usd = 16;  // 会话总成本（美元）

  // Status
  string status = 14;
  string error_msg = 15;
}
```

---

## 8. 关键流程 (Key Workflows)

### 8.1 启动与挂起 (Start & Park)

1. 用户发起请求，Server 检查 `Session Manager`。
2. 若无 Session，启动 `claude` 进程。
   - Args: `--print --verbose --output-format stream-json --session-id <sid>`
3. 进程启动后，收到 `system` 消息（初始化配置）。
4. 不立即关闭，保持 Stdin 打开，启动 Goroutine 持续读取 Stdout。

### 8.2 消息处理循环

```
for each line from CLI stdout:
    parse as JSON → StreamMessage

    if type == "system":
        // 静默处理，记录 Debug 日志
        continue

    if type == "result":
        // 提取统计，发送 session_stats 事件
        handleResultMessage(msg, stats, cfg, callback)
        return  // 结束扫描循环

    // 其他类型：dispatchCallback
    dispatchCallback(msg, callback, stats)
```

### 8.3 中途干预 (Interruption & Injection)

1. 用户在前端点击 "Cancel" 或输入反馈。
2. Server 收到 WebSocket 消息。
3. `Session.WriteInput()` 将消息构造为 JSON 写入 Stdin。
4. CLI 接收到 stdin event，中断当前思考或作为工具结果处理。

---

## 9. 统计数据收集 (Session Statistics)

### 9.1 SessionStats 结构

```go
type SessionStats struct {
    mu                   sync.Mutex
    SessionID            string
    StartTime            time.Time
    TotalDurationMs      int64
    ThinkingDurationMs   int64
    ToolDurationMs       int64
    GenerationDurationMs int64
    InputTokens          int32
    OutputTokens         int32
    CacheWriteTokens     int32
    CacheReadTokens      int32
    ToolCallCount        int32
    ToolsUsed            map[string]bool
    FilesModified        int32
    FilePaths            []string
}
```

### 9.2 统计数据提取流程

```
CLI result message
    │
    ├─► duration_ms      → TotalDurationMs
    ├─► total_cost_usd  → TotalCostUSD
    ├─► usage.input_tokens  → InputTokens
    ├─► usage.output_tokens → OutputTokens
    ├─► usage.cache_read... → CacheReadTokens
    └─► num_turns       → (内部计数)
```

---

## 10. 安全与风控 (Security)

> [!WARNING]
> **Permission Bypass**: 本次升级将引入 `--permission-mode bypassPermissions`。

- **风险**: AI 可能自动执行删除命令或修改关键文件。
- **缓解**:
    1. **Frontend Confirmation**: 尽管后端 bypass，但在前端对关键操作（如 `rm -rf`）进行 Regex 匹配拦截。
    2. **Git Recovery**: 强制在 Git 仓库内运行，确保所有文件变更可回滚。
    3. **Timeout**: Session 闲置 30 分钟自动 Kill，防止僵尸进程。
    4. **DangerDetector**: 多级危险命令检测（file_delete, system, network, permission, database, git）。

---

## 11. 错误处理 (Error Handling)

- **Process Crash**: 如果 CLI 异常退出，Session Manager 需从 Map 中移除并通知前端。
- **JSON Parse Error**: 对于非 JSON 的 stdout 行（如 stderr 泄漏），作为 `log` 类型原样转发，不阻塞解析。
- **Unknown Message Type**: 记录 Warn 日志，尝试提取文本内容（非关键，使用 SafeCallback）。

---

## 12. 版本历史 (Version History)

| Version | Date | Changes |
|:-------|:-----|:-------|
| 1.0 | Initial | 基础异步架构 |
| 1.1 | 2025-01-XX | 添加会话管理 |
| 1.2 | 2025-01-XX | 完善安全检测 |
| **1.3** | **2026-02-03** | **✅ 添加 session_stats 事件，result 消息统计提取，TotalCostUsd 追踪** |

---

**相关文档**:
- [CCRunner 消息处理机制调研](../research/cc-runner-message-handling-research.md)
- [Claude Stream JSON 格式调研](../research/claude-stream-json-format.md)
- [调试经验教训](../research/DEBUG_LESSONS.md)
