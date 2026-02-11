# 流式工具执行 调研报告

> **调研日期**: 2026-02-11
> **参考项目**: OpenClaw (openclaw/openclaw)
> **目标**: 为 DivineSense 设计流式工具执行方案

---

## 执行摘要

OpenClaw 的 **Tool Streaming** 能力允许工具执行结果在生成过程中**实时推送到客户端**，显著提升用户体验。DivineSense 目前是等待工具完全执行后才返回结果。

---

## 1. OpenClaw 流式工具分析

### 1.1 架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                     AI Agent (Pi)                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Tool Call                                          ││
│  └─────────────────────┬────────────────────────────────────┘│
│                        ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │              Gateway Tool Streaming                      ││
│  │  ┌────────────────────────────────────────────────────┐  ││
│  │  │  SSE / WebSocket Event Stream             │  ││
│  └──┬────────────────────────────────────────────────────┘  ││
│     │                                                    │
└─────┼────────────────────┬──────────────────┬──────────────┘
      ▼                    ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Client App  │   │  CLI        │   │  Web UI     │
│ (Real-time   │   │(Poll       │   │(Event       │
│  Progress)   │   │ Mode)       │   │ Source)     │
└──────────────┘   └──────────────┘   └──────────────┘
```

### 1.2 核心概念

| 概念 | 说明 | DivineSense 对应 |
|:-----|:-----|:-------------|
| **Tool Streaming** | 工具执行结果通过 WebSocket/SSE 实时推送 | 需实现 |
| **Block Streaming** | LLM 响应分块推送 | 部分支持 (LLM 调用流式) |
| **Coalescing** | 合并小块后发送，减少消息碎片 | 可优化 |
| **Human Delay** | 块间随机延迟，模拟人类节奏 | 可选 |

### 1.3 协议设计

OpenClaw 通过 Gateway WebSocket 事件流式推送工具进度：

```javascript
// Gateway WebSocket Event: tool_progress
{
  type: "event",
  event: "tool",
  payload: {
    status: "running",  // or "completed", "failed"
    output: "partial data...",
    error: null,
    done: false
  }
}

// 最终完成事件
{
  type: "event",
  event: "agent",
  payload: {
    runId: "uuid",
    status: "completed",  // or "timeout", "error"
    summary: { ... },
    result: { ... }
  }
}
```

---

## 2. DivineSense 当前状态分析

### 2.1 现有架构

```go
// ai/agents/tools/executor.go
func (e *ResilientToolExecutor) Execute(ctx context.Context, tool Tool, input string) (*Result, error) {
    // 同步执行，等待完整结果
    result, err := tool.Run(execCtx, input)

    // 只有成功或失败才返回
    return &Result{
        Data:    result.Data,
        Output:  result.Output,
        Success: true,
    }, nil
}
```

**问题**：
- ❌ 无流式反馈
- ❌ 长时间执行时用户无感知
- ❌ 无法显示进度

### 2.2 LLM 调用流式 (已支持)

```go
// ai/core/llm/service.go - 已有流式支持
type StreamEvent struct {
    Delta string `json:"delta,omitempty"`
    Done  bool   `json:"done"`
}

func (s *LLMService) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
    // ✅ 支持 SSE 实时推送 token delta
}
```

**差异**：LLM 调用已流式，但工具执行仍是同步的。

---

## 3. 推荐实现方案

### Phase 1: 工具进度流 (最小改动)

#### 3.1.1 定义进度接口

```go
// ai/agents/tools/progress.go
package tools

type ProgressWriter interface {
    Write(data []byte) (int, error)
    WriteProgress(percent int, message string) error
    Close() error
}

type StreamingResult struct {
    // 传统结果
    Final *Result

    // 流式通道
    ProgressChan <-chan ProgressEvent
}

type ProgressEvent struct {
    Percent int    `json:"percent"`
    Message  string `json:"message"`
    Data     any     `json:"data,omitempty"`
    Done     bool   `json:"done"`
}
```

#### 3.1.2 修改执行器

```go
// ai/agents/tools/executor.go
func (e *ResilientToolExecutor) ExecuteStream(ctx context.Context, tool Tool, input string) (*StreamingResult, error) {
    progress := make(chan ProgressEvent, 10)

    go func() {
        // 启动 goroutine 执行工具
        result := tool.RunWithProgress(ctx, input, progress)

        // 完成后发送最终事件
        progress <- ProgressEvent{Done: true, Data: result}
    }()

    return &StreamingResult{
        ProgressChan: progress,
    }, nil
}
```

### Phase 2: WebSocket 推送 (完整实现)

#### 3.2.1 Gateway WebSocket 事件

```go
// server/handler/websocket.go
type WSEvent struct {
    Type    string `json:"type"`    // "tool_progress"
    Event   string `json:"event"`   // "progress"
    Payload any    `json:"payload"`
}

func (h *WSHandler) BroadcastToolProgress(runID string, event ProgressEvent) {
    h.broadcast <- WSEvent{
        Type:    "event",
        Event:   "tool_progress",
        Payload: map[string]any{
            "runId":  runID,
            "status": event.Percent,
            "message": event.Message,
            "done":   event.Done,
        },
    }
}
```

#### 3.2.2 前端集成

```tsx
// web/src/components/ToolExecution.tsx
function ToolProgress({ runId }: { runId: string }) {
  const [progress, setProgress] = useState<ProgressEvent>({ percent: 0, message: "" });

  useEffect(() => {
    const ws = useWebSocket();
    ws.subscribe(`tool_progress.${runId}`, (event) => {
      setProgress(event.payload);
    });
  }, [runId]);

  return (
    <div className="flex items-center gap-2">
      {progress.done ? (
        <CheckCircle className="text-green-500" />
      ) : (
        <>
          <Loader className="animate-spin" />
          <span>{progress.message}</span>
          <div className="w-32 h-2 bg-gray-200 rounded">
            <div style={{ width: `${progress.percent}%` }} />
          </div>
        </>
      )}
    </div>
  );
}
```

---

## 4. Coalescing 优化 (可选)

参考 OpenClaw 的 **EmbeddedBlockChunker**：

```go
// ai/agents/tools/coalesce.go
type ChunkerConfig struct {
    MinChars     int           `json:"minChars"`      // 最小字符数
    MaxChars     int           `json:"maxChars"`      // 最大字符数
    IdleMs       int           `json:"idleMs"`        // 空闲等待时间
    BreakPreference string        `json:"breakPreference"` // "paragraph" | "sentence"
}

type ProgressChunker struct {
    buffer   strings.Builder
    config   ChunkerConfig
    lastSend time.Time
}

func (c *ProgressChunker) Add(event ProgressEvent) []ProgressEvent {
    c.buffer.WriteString(event.Message)

    // 检查是否应该发送
    if time.Since(c.lastSend) > c.config.IdleMs {
        if c.buffer.Len() >= c.config.MinChars {
            return []ProgressEvent{ /* ... */}  // 发送并清空
        }
    }
    return nil  // 等待更多内容
}
```

---

## 5. 实现优先级

| 优先级 | 任务 | 预估工作量 |
|:-----|:-----|:----------|
| **P1** | 定义进度接口/修改核心执行器 | 2-3 天 |
| **P1** | WebSocket 事件推送 | 2-3 天 |
| **P2** | 前端进度 UI 组件 | 1-2 天 |
| **P2** | Coalescing 优化 | 1-2 天 |

---

## 6. 技术风险与缓解

| 风险 | 缓解方案 |
|:-----|:---------|
| **goroutine 泄漏** | 使用 context.WithTimeout + defer recover() |
| **通道阻塞** | 进度通道设缓冲，发送超时保护 |
| **内存占用** | 限制并发流式执行数量 |
| **向后兼容** | 新增 `ExecuteStream()` 方法，保留原 `Execute()` |

---

## 7. 总结

OpenClaw 的流式工具执行是其**用户体验优势**之一。DivineSense 建议采用 **两阶段实现**：

1. **Phase 1**: 先实现工具进度接口（不依赖 WebSocket）
2. **Phase 2**: 完整 WebSocket 推送 + 前端 UI

**关键价值**：让用户在长时间工具执行（如代码分析、大文件处理）时能看到实时进度。

---

**参考实现**:
- OpenClaw Tool Streaming: `openclaw/openclaw/docs/concepts/streaming.md`
- OpenClaw Gateway Protocol: `openclaw/openclaw/docs/concepts/architecture.md`
