# Claude Code CLI Stream-JSON 事件格式调研

> 调研日期：2026-02-01
> CLI 版本：2.1.15

## 概述

Claude Code CLI 的 `--output-format stream-json` 输出格式与预期不同：
- **没有**独立的 `type: "tool_use"` 或 `type: "tool_result"` 事件
- 工具事件**嵌套**在 `assistant`/`user` 消息的 `content` 数组中

## 事件类型统计

示例：调研 "AI Agent 技术进展" 问题

| 顶层 type | 数量 | 说明 |
|-----------|------|------|
| `system` | 1 | 初始化，包含可用工具列表 |
| `assistant` | 11 | 包含 tool_use 或 text 内容块 |
| `user` | 7 | 包含 tool_result 内容块 |
| `result` | 1 | 最终结果和统计 |

## 消息结构

### 1. System 消息（初始化）

```json
{
  "type": "system",
  "subtype": "init",
  "session_id": "uuid",
  "tools": ["Task", "Bash", "WebSearch", ...],
  "mcp_servers": [...]
}
```

### 2. Assistant 消息（包含 tool_use）

```json
{
  "type": "assistant",
  "message": {
    "id": "msg_xxx",
    "type": "message",
    "role": "assistant",
    "content": [
      {
        "type": "tool_use",
        "id": "call_xxx",
        "name": "WebSearch",
        "input": {"query": "..."}
      }
    ]
  },
  "session_id": "uuid"
}
```

### 3. User 消息（包含 tool_result）

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "call_xxx",
        "type": "tool_result",
        "content": "工具返回的完整内容..."
      }
    ]
  },
  "session_id": "uuid"
}
```

### 4. Assistant 消息（包含 text 回答）

```json
{
  "type": "assistant",
  "message": {
    "id": "msg_xxx",
    "type": "message",
    "role": "assistant",
    "content": [
      {
        "type": "text",
        "text": "基于搜索结果..."
      }
    ]
  },
  "session_id": "uuid"
}
```

### 5. Result 消息（完成）

```json
{
  "type": "result",
  "subtype": "success",
  "duration_ms": 12345,
  "result": "最终答案文本...",
  "usage": {...}
}
```

## 解析逻辑

### CCRunner dispatchCallback 修复

**修复前**：只处理独立的 `type: "tool_use"` 事件（不存在）
```go
case "tool_use":
    if msg.Name != "" {
        return callback(EventTypeToolUse, msg.Name)
    }
```

**修复后**：处理嵌套在 `assistant` 消息中的 `tool_use` blocks
```go
case "message", "assistant":
    for _, block := range msg.GetContentBlocks() {
        if block.Type == "tool_use" && block.Name != "" {
            callback(EventTypeToolUse, block.Name)
        }
    }
```

**新增**：处理嵌套在 `user` 消息中的 `tool_result` blocks
```go
case "user":
    for _, block := range msg.GetContentBlocks() {
        if block.Type == "tool_result" {
            callback(EventTypeToolResult, block.Content)
        }
    }
```

## ContentBlock 结构

```go
type ContentBlock struct {
    Type    string         `json:"type"`       // "text", "tool_use", "tool_result"
    Text    string         `json:"text,omitempty"`
    Name    string         `json:"name,omitempty"`     // tool_use 时的工具名
    ID      string         `json:"id,omitempty"`       // tool_use_id
    Input   map[string]any `json:"input,omitempty"`   // tool_use 参数
    Content string         `json:"content,omitempty"`  // tool_result 返回值
    IsError bool           `json:"is_error,omitempty"`
}
```

## 关键发现

1. **没有独立 tool_use/tool_result 事件** - 必须解析 content 数组
2. **user 消息用于 tool_result** - 这与直觉相反（通常 user 是输入）
3. **消息顺序**：system → assistant(tool_use) → user(tool_result) → assistant(text) → result
4. **tool_result 可能很大** - 包含完整的搜索结果或文件内容

## 前端适配建议

1. **工具名称提取**：从 `content[i].name` 获取
2. **工具结果处理**：可能需要截断或摘要 `content`（可能很大）
3. **进度显示**：assistant(tool_use) → user(tool_result) → assistant(text) 的流式展示
