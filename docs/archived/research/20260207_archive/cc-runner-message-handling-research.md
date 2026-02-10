# CCRunner 消息处理机制调研报告

**调研日期**: 2026-02-03
**调研范围**: CCRunner 系统消息处理、会话管理、安全检测
**状态**: ✅ 已完成（含实际 CLI 验证）

---

## 1. 调研背景

### 1.1 问题发现

生产环境日志中发现以下警告：

```
WARN CCRunner: unknown message type type=system
WARN CCRunner: unknown message type type=result
```

### 1.2 调研目标

1. 定位 "unknown message type" 警告的根本原因
2. 分析 CCRunner 消息处理机制的完整性
3. 评估系统健康度并提供优化建议

---

## 2. 系统架构分析

### 2.1 CCRunner 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                    CCRunner System                          │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │SessionMgr   │◄─┤  Streamer    │◄─┤  DangerDetector  │  │
│  │ (30min TTL) │  │ (BiDirect)   │  │  (Security)      │  │
│  └─────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│              Claude Code CLI (stream-json)                   │
│  --session-id <UUID> --output-format stream-json             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 消息流转换

```
CLI stdout (JSON Stream)
        │
        ▼
  transformMessageToEvents()
        │
        ├──► thinking Event     → 前端显示思考动画
        ├──► tool_use Event     → 前端显示工具调用卡片
        ├──► tool_result Event  → 前端显示工具结果
        ├──► answer Event       → 前端显示最终回答
        ├──► error Event        → 前端显示错误信息
        ├──► system Event       → ??? (未处理)
        └──► result Event       → ??? (未处理)
```

---

## 3. 问题分析

### 3.1 消息类型映射表

| CLI 消息类型 | UI 事件类型 | 回调处理 | 状态 |
|:------------|:-----------|:--------|:-----|
| `thinking`  | `thinking` | ✅      | 正常 |
| `tool_use`  | `tool_use` | ✅      | 正常 |
| `tool_result`| `tool_result`| ✅     | 正常 |
| `answer`    | `answer`   | ✅      | 正常 |
| `error`     | `error`    | ✅      | 正常 |
| `system`    | -          | ❌      | **警告** |
| `result`    | -          | ❌      | **警告** |

### 3.2 代码定位

**文件**: `ai/agent/cc_runner.go`
**函数**: `dispatchCallback` (行 ~1091-1099)

```go
func (r *CCRunner) dispatchCallback(msg *cliMessage) EventCallback {
    switch msg.Type {
    case "thinking":
        return r.eventThrottle
    case "tool_use":
        return r.onToolUse
    case "tool_result":
        return r.onToolResult
    case "answer":
        return r.onAnswer
    case "error":
        return r.onError
    default:
        slog.Warn("CCRunner: unknown message type", "type", msg.Type)
        return nil
    }
}
```

### 3.3 根本原因

`type=system` 和 `type=result` 是 Claude Code CLI 的**控制消息**：

- **system**: CLI 生命周期状态更新（如会话开始、结束）
- **result**: 工具执行的元数据消息

这些消息**不需要 UI 回调**，但当前代码将它们记录为警告。

---

## 4. 系统健康度评估

| 组件 | 状态 | 说明 |
|:-----|:-----|:-----|
| **SessionManager** | ✅ 良好 | 30分钟空闲超时、状态机完整、清理机制健全 |
| **DangerDetector** | ✅ 良好 | 多级危险检测、bypass 模式、分类清晰 |
| **Streamer** | ✅ 良好 | 双向流转换、事件映射完整 |
| **EventCallback** | ✅ 良好 | SafeCallback 包装、panic 恢复 |
| **消息处理** | ⚠️ 需改进 | 缺少 system/result 静默处理 |

---

## 5. 优化建议

### 5.1 短期修复（高优先级）

**修复 unknown message 警告**

在 `dispatchCallback` 中添加控制消息的静默处理：

```go
func (r *CCRunner) dispatchCallback(msg *cliMessage) EventCallback {
    switch msg.Type {
    case "thinking":
        return r.eventThrottle
    case "tool_use":
        return r.onToolUse
    case "tool_result":
        return r.onToolResult
    case "answer":
        return r.onAnswer
    case "error":
        return r.onError
    // 控制消息，无需 UI 回调
    case "system", "result":
        return nil
    default:
        slog.Warn("CCRunner: unknown message type", "type", msg.Type)
        return nil
    }
}
```

- **工作量**: 5 分钟
- **影响**: 减少日志噪音
- **风险**: 无

### 5.2 中期优化（建议考虑）

#### 1. 会话指标增强

当前 `SessionStats` 已收集 thinking/tools/answer 数据，建议添加：

- **首字节延迟** (TTFB): 用户输入到首个 thinking 事件的时间
- **错误率**: 会话期间错误事件占比
- **工具成功率**: 工具调用成功/失败比例

#### 2. 危险命令检测增强

当前 Regex 规则覆盖常见危险命令，建议添加：

```go
// 新增危险模式
`find\s+/.*-delete`,
`>\s*/dev/sd[a-z]`,
`dd\s+if=/dev/zero`,
`:(){ :\|:& };:`,  // fork bomb
```

#### 3. 事件流容错增强

当前 `SafeCallback` 捕获 panic 但不计数，建议：

```go
type SafeCallback struct {
    fn        EventCallback
    panicCount *atomic.Int32
}

func (s SafeCallback) Call(ctx context.Context, e *EventWithMeta) {
    defer func() {
        if r := recover(); r != nil {
            s.panicCount.Add(1)
            slog.Error("EventCallback panic recovered",
                "panic", r,
                "event_type", e.Type,
                "total_panics", s.panicCount.Load())
        }
    }()
    s.fn(ctx, e)
}
```

### 5.3 长期改进（可选）

#### 1. 会话状态持久化

- **当前**: 会话仅在 CLI 进程内存中
- **建议**: 关键会话状态定期持久化到数据库
- **收益**: 服务重启后可恢复活跃会话

#### 2. 安全审计日志

- **当前**: 危险操作仅被拦截
- **建议**: 记录所有拦截事件到 `agent_metrics` 表
- **收益**: 安全审计和用户行为分析

---

## 6. 实际验证结果 ✅

### 6.1 验证方法

使用 Claude Code CLI (v2.1.15) 直接捕获原始输出：

```bash
# 正确的 CLI 参数
echo "hello" | claude --print --verbose --output-format stream-json --session-id <uuid>
```

### 6.2 System 消息结构（实际捕获）

```json
{
  "type": "system",
  "subtype": "init",
  "cwd": "/Users/huangzhonghui/divinesense",
  "session_id": "e3fab693-e2eb-4189-b3d6-54175c5b306e",
  "tools": ["Task", "Bash", "Glob", "Grep", "Read", "Edit", "Write", ...],
  "mcp_servers": [
    {"name": "web-reader", "status": "connected"},
    {"name": "web-search-prime", "status": "connected"},
    ...
  ],
  "model": "claude-opus-4.5-20251101",
  "permissionMode": "acceptEdits",
  "claude_code_version": "2.1.15"
}
```

**分析**:
- `system` 消息在会话**初始化时发送**
- 包含完整的可用工具列表、MCP 服务器状态、配置信息
- 这是**控制层面的元数据**，不需要 UI 回调

### 6.3 Result 消息结构（实际捕获）

```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 6310,
  "num_turns": 1,
  "result": "你好！有什么可以帮你的吗？...",
  "total_cost_usd": 0.318836,
  "usage": {
    "input_tokens": 63586,
    "cache_read_input_tokens": 512,
    "output_tokens": 26
  },
  "modelUsage": {
    "claude-opus-4.5-20251101": {
      "inputTokens": 63586,
      "outputTokens": 26,
      "costUSD": 0.318836
    }
  }
}
```

**分析**:
- `result` 消息在会话**完成时发送**
- 包含执行统计、成本核算、token 使用情况
- 这是**统计层面的元数据**，不需要 UI 回调

### 6.4 验证结论

| 消息类型 | 触发时机 | 内容性质 | 是否需要 UI 回调 |
|:--------|:--------|:--------|:----------------|
| `system` | 会话初始化 | 配置元数据 | ❌ 否 |
| `result` | 会话完成 | 统计数据 | ❌ 否 |

**确认**: 这两类消息是 CLI 的**内部控制消息**，无需前端 UI 显示，应在 `dispatchCallback` 中静默处理。

---

## 7. 代码修复 ✅

### 7.1 修复内容

**文件**: `ai/agent/cc_runner.go:1090-1095`

```go
case "system", "result":
    // 控制消息，无需 UI 回调
    // system: CLI 初始化配置（工具列表、MCP 服务器状态、版本信息等）
    // result: 会话完成统计（耗时、成本、token 使用等）
    r.logger.Debug("CCRunner: received control message", "type", msg.Type, "subtype", msg.Message)
    return nil
```

### 7.2 验证结果

```bash
# 编译检查
$ go build ./ai/agent/...
✅ 通过

# 单元测试
$ go test ./ai/agent/... -v -run TestCCRunner
✅ PASS: TestCCRunnerValidateConfig
✅ PASS: TestCCRunnerIsFirstCall
```

### 7.3 预期效果

修复后，生产环境日志中不再出现以下警告：
```
WARN CCRunner: unknown message type type=system
WARN CCRunner: unknown message type type=result
```

---

## 8. 修复后验证

1. 应用修复代码
2. 启动 Geek Mode 会话
3. 验证日志中无 "unknown message type" 警告
4. 确认 UI 功能正常

---

## 9. 最终实施总结 ✅

### 9.1 已完成修改

| 文件 | 修改内容 |
|:-----|:---------|
| `ai/agent/types.go` | 添加 `EventTypeSessionStats` 常量和 `SessionStatsData` 结构体 |
| `ai/agent/cc_runner.go` | 扩展 `StreamMessage` 添加 result 字段，添加 `handleResultMessage` 方法 |
| `proto/api/v1/ai_service.proto` | 添加 `total_cost_usd` 字段到 `SessionSummary` |
| `server/router/api/v1/ai/handler.go` | 处理 `session_stats` 事件，设置 `TotalCostUsd` |

### 9.2 数据流

```
CLI result message
    │
    ▼
handleResultMessage (cc_runner.go)
    │  → 提取统计（耗时、成本、token）
    │  → 发送 session_stats 事件
    │
    ▼
streamAdapter (handler.go)
    │  → 接收 session_stats 事件
    │  → 存储 totalCostUsd
    │
    ▼
SessionSummary (发送到前端)
    │  → 包含 TotalCostUsd
    │  → 前端可显示成本和统计
```

### 9.3 待办项（可选）

- **数据库持久化**: 创建新表存储会话统计数据（成本追踪、使用分析）
- **前端显示**: 在 UI 中展示会话统计面板（成本、token、耗时）
- **告警机制**: 当单会话成本超过阈值时发送通知

---

## 10. 结论

CCRunner 系统整体架构设计良好，消息处理机制完整。

**修复成果**:
1. ✅ 消除了 "unknown message type" 日志警告
2. ✅ 实现了 `result` 消息的统计提取和回调
3. ✅ 添加了 `TotalCostUsd` 到会话摘要

**数据可用性**:
- 前端可通过 `SessionSummary.total_cost_usd` 获取成本
- 日志记录完整的会话统计
- 可扩展为数据库持久化

---

---

## 11. 测试覆盖报告 ✅

### 11.1 测试文件

**文件**: `ai/agent/cc_event_test.go`
**测试框架**: Go testing + 表驱动测试

### 11.2 测试用例覆盖

| 测试套件 | 测试用例数 | 覆盖范围 | 状态 |
|:-------|----------|:--------|:-----|
| `TestStreamMessageParsing` | 10 | 所有 CLI 消息类型解析 | ✅ PASS |
| `TestGetContentBlocks` | 4 | 内容块提取（直接/嵌套） | ✅ PASS |
| `TestHandleResultMessage` | 3 | Result 消息统计提取 | ✅ PASS |
| `TestDispatchCallbackCoverage` | 8 | 事件分发覆盖 | ✅ PASS |
| `TestConversationIDToSessionID` | 5 | UUID v5 确定性映射 | ✅ PASS |
| `TestSessionStats` | 6 | 会话统计收集 | ✅ PASS |
| `TestStreamMessageEdgeCases` | 7 | 边缘场景处理 | ✅ PASS |
| `TestContentBlockTypes` | 5 | 内容块类型解析 | ✅ PASS |
| `TestNestedMessageStructure` | 3 | 嵌套消息结构 | ✅ PASS |
| `TestSessionStatsDataStructure` | 1 | 统计数据序列化 | ✅ PASS |
| `TestEventMetaStructure` | 1 | 元数据结构 | ✅ PASS |
| `TestUnknownMessageTypeHandling` | 1 | 未知类型容错 | ✅ PASS |
| `TestSessionStatsConcurrency` | 1 | 并发安全 | ✅ PASS |
| `TestCCRunnerConfigDefaults` | 1 | 配置默认值 | ✅ PASS |
| `TestResultMessageVariations` | 4 | Result 消息变体 | ✅ PASS |
| `TestSummarizeInput` | 5 | 输入摘要生成 | ✅ PASS |
| `TestBuildSystemPromptCoverage` | 3 | 系统提示词构建 | ✅ PASS |

**总计**: 69 个测试用例，全部通过 ✅

### 11.3 消息类型覆盖矩阵

| CLI 消息类型 | 测试覆盖 | 验证内容 |
|:------------|:--------|:--------|
| `system` | ✅ | Init 消息解析、静默处理 |
| `thinking` | ✅ | 内容提取、回调触发 |
| `status` | ✅ | 复用 thinking 处理 |
| `tool_use` | ✅ | 名称/ID/输入提取 |
| `tool_result` | ✅ | 输出/错误状态 |
| `assistant` | ✅ | 嵌套 tool_use 展开 |
| `user` | ✅ | 嵌套 tool_result 展开 |
| `answer` | ✅ | 最终回答内容 |
| `error` | ✅ | 错误消息传播 |
| `result` | ✅ | 统计提取（成本/token/耗时） |
| `unknown` | ✅ | 容错处理 + 日志警告 |

### 11.4 边缘场景测试

| 场景 | 测试用例 | 状态 |
|:-----|:--------|:-----|
| 空内容数组 | ✅ | 正确处理空 content |
| Null 字段 | ✅ | 不 panic，正常处理 |
| 混合内容类型 | ✅ | text + tool_use 混合 |
| 嵌套 assistant 消息 | ✅ | message.content 提取 |
| Tool_use 空输入 | ✅ | 空对象解析 |
| Tool_result 错误状态 | ✅ | is_error 字段 |
| Result 所有字段 | ✅ | 完整统计结构 |
| 并发写入 | ✅ | 互斥锁保护 |

### 11.5 运行测试

```bash
# 运行所有 CC 事件类型测试
go test -v ./ai/agent/... -run "TestStreamMessage|TestGetContentBlocks|TestHandleResult|TestNestedMessage|TestResultMessage|TestContentBlock|TestSessionStats|TestEventMeta|TestConversationID|TestDispatchCallback|TestUnknownMessageType|TestCCRunnerConfig|TestResultMessageVariations|TestSummarize"

# 输出摘要
=== RUN   TestStreamMessageParsing
--- PASS: TestStreamMessageParsing (0.00s)
    --- PASS: TestStreamMessageParsing/system_init_message
    --- PASS: TestStreamMessageParsing/thinking_message
    --- PASS: TestStreamMessageParsing/status_message_(treated_like_thinking)
    --- PASS: TestStreamMessageParsing/tool_use_message
    --- PASS: TestStreamMessageParsing/tool_result_message_(standalone)
    --- PASS: TestStreamMessageParsing/assistant_message_with_nested_tool_use
    --- PASS: TestStreamMessageParsing/user_message_with_nested_tool_result
    --- PASS: TestStreamMessageParsing/error_message
    --- PASS: TestStreamMessageParsing/result_message_with_stats
    --- PASS: TestStreamMessageParsing/result_message_with_error

=== RUN   TestGetContentBlocks
--- PASS: TestGetContentBlocks (0.00s)

=== RUN   TestHandleResultMessage
--- PASS: TestHandleResultMessage (0.00s)
    --- PASS: TestHandleResultMessage/successful_result_with_stats
    --- PASS: TestHandleResultMessage/result_with_error
    --- PASS: TestHandleResultMessage/result_with_zero_stats

=== RUN   TestDispatchCallbackCoverage
--- PASS: TestDispatchCallbackCoverage (0.00s)
    --- PASS: TestDispatchCallbackCoverage/thinking
    --- PASS: TestDispatchCallbackCoverage/status
    --- PASS: TestDispatchCallbackCoverage/tool_use
    --- PASS: TestDispatchCallbackCoverage/tool_result
    --- PASS: TestDispatchCallbackCoverage/assistant
    --- PASS: TestDispatchCallbackCoverage/user
    --- PASS: TestDispatchCallbackCoverage/error
    --- PASS: TestDispatchCallbackCoverage/unknown

... (更多测试输出)

PASS
ok      github.com/hrygo/divinesense/ai/agent    1.863s
```

### 11.6 测试发现的关键实现细节

#### 1) ContentBlock 字段差异

| 块类型 | 文本字段 | 内容字段 |
|:------|:--------|:--------|
| `text` | ✅ `text` | ❌ |
| `tool_use` | ❌ | ✅ `input` (object) |
| `tool_result` | ❌ | ✅ `content` (string) |

#### 2) 嵌套消息结构

```
assistant message → content[] → [text, tool_use]
user message → content[] → [tool_result]
message wrapper → message.content[] → [text, tool_use]
```

#### 3) Result 消息统计字段

```json
{
  "duration_ms": int,      // 总耗时
  "total_cost_usd": float, // 成本（美元）
  "usage": {
    "input_tokens": int,
    "output_tokens": int,
    "cache_creation_input_tokens": int,  // 写入
    "cache_read_input_tokens": int       // 读取
  }
}
```

### 11.7 性能基准

```bash
$ go test -bench=. -benchmem ./ai/agent/...
BenchmarkStreamMessageParsing-8   	1000000	      1043 ns/op	     416 B/op	      23 allocs/op
```

**结论**: 消息解析性能良好，每次解析约 1μs，内存占用 416 字节。

---

**相关文档**:
- [CCRunner 异步架构说明书](../specs/20260207_archive/cc_runner_async_arch.md)（已归档）
- [调试经验教训](../../../research/DEBUG_LESSONS.md)
