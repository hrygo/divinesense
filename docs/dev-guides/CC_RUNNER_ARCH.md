# CC Runner 异步架构

> **实现状态**: ✅ 完成 (v0.97.0) | **规格版本**: v1.2 | **位置**: `ai/agent/cc_runner/`

## 概述

CC Runner 是 Geek Mode 的核心异步架构，实现 Claude Code CLI 的全双工持久化会话管理。从一次性执行（One-shot）升级为持久的、双向流的会话模型。

### 架构演进

| 版本 | 特性 | 状态 |
|:-----|:-----|:-----|
| v1.0 | One-shot 执行 | 已废弃 |
| v1.1 | 持久化会话 | 已废弃 |
| v1.2 | 全双工流式 | **当前版本** |

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (React)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │  EventBadge  │  │ ToolCallCard │  │  SessionSummaryPanel │ │
│  └──────────────┘  └──────────────┘  └──────────────────────┘ │
│                              │                                  │
│                        WebSocket (SSE)                         │
└──────────────────────────────┼──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                     Backend (Go)                                │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Session Mgr │◄─┤   Streamer  │◄─┤  DangerDetector      │  │
│  │  (30min)    │  │ (Bidirect)  │  │  (rm -rf, format)    │  │
│  └─────────────┘  └──────────────┘  └──────────────────────┘  │
└──────────────────────────────┼──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                   Claude Code CLI (OS Process)                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  --session-id <UUID> --output-format stream-json          │ │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │ │
│  │  │    CLI      │  │  In-Memory   │  │  Skills & MCP    │  │ │
│  │  │   Engine    │◄─┤   Context    │  │    Registry      │  │ │
│  │  └─────────────┘  └──────────────┘  └──────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

## 核心组件

### SessionManager (`session_manager.go`)

```go
type SessionManager struct {
    sessions    map[string]*Session
    mutex       sync.RWMutex
    idleTimeout time.Duration  // 30分钟空闲超时
}

type Session struct {
    ID          string
    Process     *exec.Cmd
    Stdin       io.WriteCloser
    Stdout      io.BufferedReader
    Stderr      io.BufferedReader
    CreatedAt   time.Time
    LastActive  time.Time
    Status      SessionStatus
}

// 创建或获取会话
func (sm *SessionManager) GetOrCreate(ctx context.Context, sessionID string) (*Session, error)

// 停止会话
func (sm *SessionManager) Stop(sessionID string) error

// 清理空闲会话
func (sm *SessionManager) CleanupIdleSessions()
```

### Streamer (`streamer.go`)

```go
type Streamer struct {
    sessionMgr *SessionManager
}

// StreamInput 将用户输入流式传输到 CLI
func (s *Streamer) StreamInput(ctx context.Context, sessionID string, input <-chan string) error

// StreamOutput 将 CLI 输出流式传输到前端
func (s *Streamer) StreamOutput(ctx context.Context, sessionID string) (<-chan StreamEvent, error)

type StreamEvent struct {
    Type    EventType  // thinking, tool_use, tool_result, answer, error
    Content string
    Meta    map[string]any
}
```

### DangerDetector (`danger_detector.go`)

```go
type DangerDetector struct {
    patterns []*regexp.Regexp
}

// 危险命令模式
var dangerPatterns = []string{
    `rm\s+-rf\s+/`,           // 删除根目录
    `mkfs\.\w+`,              // 格式化磁盘
    `dd\s+if=.*of=/dev/sd`,   // 直接写磁盘
    `>\s+/dev/`,              // 覆盖设备
}

// Detect 检测危险命令
func (dd *DangerDetector) Detect(input string) (*DangerLevel, error)

type DangerLevel int
const (
    Safe DangerLevel = iota
    Warning
    Critical
)
```

### SessionStats (`session_stats.go`)

```go
type SessionStats struct {
    sessionID    string
    thinkingTime time.Duration
    tokensUsed   int
    toolCalls    int
    errors       int
    startTime    time.Time
}

// RecordThinking 记录思考时间
func (ss *SessionStats) RecordThinking(duration time.Duration)

// RecordToolCall 记录工具调用
func (ss *SessionStats) RecordToolCall(toolName string)

// GetSummary 获取统计摘要
func (ss *SessionStats) GetSummary() map[string]interface{}
```

---

## 会话映射模型

```
前端 ConversationID (数据库 ID)
         │
         ▼ UUID v5 定向哈希
         │
    SessionID (UUID)
         │
         ▼
Claude Code CLI Process
```

### UUID v5 映射实现

```go
func DeriveSessionID(conversationID int64) string {
    namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")  // DNS namespace
    data := []byte(fmt.Sprintf("divinesense:conversation:%d", conversationID))
    return uuid.NewSHA1(namespace, data).String()
}
```

**特性**：
- **确定性**：相同 conversationID 始终生成相同 sessionID
- **可恢复**：CLI 自动从 `~/.claude/sessions/` 恢复上下文
- **物理隔离**：每个会话独立 OS 进程

---

## 交互协议

### Client → Server (WebSocket Events)

| Event | Payload | 描述 |
|:-----|:--------|:-----|
| `session.start` | `{config, mode}` | 启动新会话 |
| `input.send` | `{text, sessionId}` | 发送用户输入 |
| `session.stop` | `{sessionId}` | 强制停止 |

### Server → Client (Stream Events)

| Event | Meta | 描述 |
|:-----|:-----|:-----|
| `thinking` | `{tokens, duration}` | 思考过程（增量） |
| `tool_use` | `{name, input, toolId}` | 工具调用 |
| `tool_result` | `{name, isError, output}` | 工具结果 |
| `answer` | `{delta, done}` | 最终回答（增量） |
| `error` | `{message, code}` | 系统级错误 |
| `session.stats` | `{tokens, tools, duration}` | 会话统计 |

---

## 安全与风控

### 1. Permission Bypass

```bash
claude --permission-mode bypassPermissions
```

### 2. 前端确认

```typescript
// 危险命令拦截
const DANGER_PATTERNS = [
  /rm\s+-rf\s+/,
  /mkfs\.\w+/,
  /dd\s+if=.*of=\/dev\/sd/
];

if (DANGER_PATTERNS.some(p => p.test(input))) {
  showConfirmDialog("此操作可能危险，确认继续？");
}
```

### 3. Git 恢复

```go
// 强制在 Git 仓库内运行
func (sm *SessionManager) ensureGitRepo(workDir string) error {
    if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
        return fmt.Errorf("必须在 Git 仓库内运行")
    }
    return nil
}
```

### 4. 超时保护

- **空闲超时**：30 分钟无活动自动 Kill
- **总时长限制**：2 小时强制结束
- **内存限制**：2GB 内存上限

---

## API 端点

### gRPC 服务

| RPC | 方法 | 描述 |
|:-----|:-----|:-----|
| `ChatService` | `StreamChat` | 流式聊天（SSE） |
| `ChatService` | `StopChat` | 停止会话（所有权验证） |

### HTTP/WebSocket

| 端点 | 方法 | 描述 |
|:-----|:-----|:-----|
| `/api/v1/chat/geek/stream` | WebSocket | Geek Mode 流式连接 |
| `/api/v1/chat/geek/stop` | POST | 停止会话 |

---

## 配置选项

| 环境变量 | 默认值 | 说明 |
|:---------|:------|:-----|
| `DIVINESENSE_CLAUDE_CODE_ENABLED` | `false` | 是否启用 Geek Mode |
| `DIVINESENSE_CLAUDE_CODE_WORKDIR` | `~/.divinesense/claude` | 工作目录 |
| `DIVINESENSE_CLAUDE_CODE_IDLE_TIMEOUT` | `30m` | 空闲超时 |
| `DIVINESENSE_CLAUDE_CODE_MAX_SESSIONS` | `10` | 最大并发会话 |
| `DIVINESENSE_EVOLUTION_ENABLED` | `false` | 是否启用 Evolution Mode |
| `DIVINESENSE_EVOLUTION_ADMIN_ONLY` | `true` | 仅管理员可用 Evolution |

---

## 监控指标

```go
type CCRunnerMetrics struct {
    ActiveSessions    int64
    TotalSessions     int64
    AvgSessionDuration int64
    InputChars        int64
    OutputChars       int64
    ErrorCount        int64
    DangerWarnings    int64
}
```

---

## 调试

### 查看活动会话

```bash
# 列出所有活动会话
curl http://localhost:28081/api/v1/chat/geek/sessions
```

### 查看会话日志

```bash
# 会话日志位置
~/.claude/sessions/{session-id}/logs.txt
```

### 手动清理

```bash
# 清理所有会话
killall -9 claude

# 清理特定会话
rm -rf ~/.claude/sessions/{session-id}
```

---

## 相关文档

- [架构文档 - CC Runner](../archived/specs/20260207_archive/cc_runner_async_arch.md)
- [部署指南 - Geek Mode](../deployment/BINARY_DEPLOYMENT.md#geek-mode-配置)
- [ARCHITECTURE.md - CC Runner 章节](ARCHITECTURE.md#cc-runner-异步架构-geek-mode-核心)
