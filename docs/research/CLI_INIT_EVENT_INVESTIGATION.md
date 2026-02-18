# Claude Code CLI `init` 事件调研报告

> 调研日期: 2026-02-19
> CLI 版本: 2.1.39 (Claude Code)
> 相关 Issue: GeekParrot 页面卡住问题

---

## 1. 问题背景

在 GeekParrot 模式中，我们需要等待 CLI 完全初始化后才发送用户输入。原代码使用 `init` 事件作为就绪信号，但经常超时。

## 2. 调研方法

通过多种测试场景观察 CLI 的事件输出行为：

```bash
# 测试命令模板
claude --print --verbose --output-format stream-json --input-format stream-json --session-id <UUID>
```

## 3. CLI 事件序列

### 正常启动（有 stdin 输入）

```
hook_started (x2)    ← Session hooks 开始
hook_response (x2)   ← Session hooks 完成 (outcome: success)
init                 ← CLI 完全初始化 (约 40-200ms 后)
assistant            ← AI 响应 (包含 tool_use)
user                 ← 工具执行结果 (包含 tool_result)
assistant            ← 最终响应
result               ← 任务完成
```

### 无输入启动

```
hook_started (x2)    ← Session hooks 开始
hook_response (x2)   ← Session hooks 完成
(无 init 事件)       ← CLI 等待输入，不发送 init
```

## 4. 测试结果

| 测试场景 | stdin 输入 | `init` 事件 | 备注 |
|---------|-----------|-------------|------|
| 空输入 + 15s 超时 | ❌ 无 | ❌ 不发送 | CLI 等待输入 |
| 空输入 + 30s 超时 | ❌ 无 | ❌ 不发送 | CLI 等待输入 |
| 立即输入 | ✅ 有 | ✅ 发送 | 在 hook_response 后约 40ms |
| 5s 延迟输入 | ✅ 有 | ✅ 发送 | 在输入前发送 |
| 10s 延迟输入 | ✅ 有 | ✅ 发送 | 在输入前发送 |

## 5. 关键发现

### 5.1 鸡生蛋问题

```
CLI 行为: 检测到 stdin 有数据 → 发送 init 事件
代码行为: 等待 init 事件 → 发送用户输入
结果: 互相等待 → 超时
```

### 5.2 `init` 事件触发条件

CLI **只在检测到 stdin 有数据时**才发送 `init` 事件。这是 CLI 的设计行为，不是 bug。

## 6. 解决方案

### 6.1 就绪信号选择

| 信号 | 可靠性 | 说明 |
|-----|--------|------|
| `init` 事件 | ❌ 不可靠 | 需要先有输入才发送 |
| `hook_response` (success) | ✅ 可靠 | hooks 完成即发送 |

### 6.2 实现代码

```go
// runner.go - 检测 CLI 就绪信号
if msg.Type == "system" && msg.Subtype == "hook_response" && msg.Outcome == "success" {
    r.logger.Info("CCRunner: CLI hooks ready, session ready for input",
        "session_id", session.ID, "hook_name", msg.HookName)
    select {
    case <-session.initReceived:
        // Already closed
    default:
        close(session.initReceived)
    }
}
```

## 7. 时间线对比

### 使用 `hook_response` 作为就绪信号

```
00:00.000  CLI 进程启动
00:09.000  hook_started (hooks 开始)
00:09.100  hook_response (hooks 完成) → initReceived 关闭
00:09.100  WriteInput 发送用户输入  ← 立即发送
00:09.300  init 事件 (CLI 收到输入后发送)
00:10.000  assistant (tool_use)
00:10.100  user (tool_result)
00:15.000  result (完成)
```

### 使用 `init` 作为就绪信号（旧方案）

```
00:00.000  CLI 进程启动
00:09.000  hook_started (hooks 开始)
00:09.100  hook_response (hooks 完成)
(等待 init...)
30s 超时 → 失败
```

## 8. 相关提交

- `fix(ai): use hook_response success as CLI ready signal instead of init`
- `debug(ai): add logging for tool_use callback execution`

## 9. 附录：测试命令

```bash
# 测试 1: 无输入，观察事件
timeout 15 claude --print --verbose --output-format stream-json \
  --input-format stream-json --session-id "$(uuidgen | tr '[:upper:]' '[:lower:]')" \
  2>&1 | jq -c '{type, subtype}'

# 测试 2: 有输入，观察事件
echo '{"message":{"content":"hi","role":"user"},"type":"user"}' | \
  timeout 30 claude --print --verbose --output-format stream-json \
  --input-format stream-json --session-id "$(uuidgen | tr '[:upper:]' '[:lower:]')" \
  2>&1 | jq -c '{type, subtype}'
```

---

*本报告基于 Claude Code CLI v2.1.39 的行为分析，未来版本可能有变化。*
