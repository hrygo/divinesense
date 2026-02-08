# 流式工具调用技术方案

> **文档版本**: v1.0
> **创建日期**: 2026-02-08
> **关联 Issue**: #125
> **作者**: Claude Code

---

## 1. 概述

### 1.1 问题背景

当前 DivineSense AI 聊天系统在工具调用时存在用户体验问题：

- **工具调用无实时反馈**：用户发送问题后，系统长时间"静默"，工具调用开始和结束之间没有视觉反馈
- **端到端延迟高**：用户需等待完整的 LLM 响应生成完毕，才能看到工具调用决策
- **流式与非流式混用**：最终答案流式输出，但工具调用过程完全阻塞

### 1.2 目标

实现**完全流式工具调用**（Solution A），确保：

1. **工具调用准确性优先** - 不引入新的解析错误
2. **实时用户体验** - 工具调用决策即时可见
3. **向后兼容** - 保留现有同步接口
4. **渐进式交付** - 分阶段实现，降低风险

---

## 2. 架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
│  ChatMessages → useAIQueries → SSE (EventStream)                │
│    │                                                             │
│    ├─ EventBadge (工具调用徽章)                                  │
│    ├─ ToolCallCard (工具调用卡片)                                │
│    └─ StreamingIndicator (流式状态指示)                          │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC Stream
┌────────────────────────────▼────────────────────────────────────┐
│                 Backend (handler.go)                             │
│  ParrotHandler.Handle → Agent.RunWithCallback                   │
│                      │                                          │
│                      ├─ EventWithMeta {tool_use, tool_result}    │
│                      └─ stream.Send(ChatResponse)               │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│              Agent Layer (tool_adapter.go)                      │
│  Agent.RunWithCallback → NEW: StreamingToolExecutor            │
│    │                                                           │
│    ├─ ChatStreamWithTools() → chunkChan                        │
│    ├─ DetectToolCalls() → 实时检测工具调用                      │
│    ├─ ExecuteTools() → 并发执行                                 │
│    └─ callback(EventToolUse/EventToolResult)                   │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                 LLM Layer (llm.go)                              │
│  llmService.ChatStreamWithTools()                               │
│    │                                                           │
│    ├─ openai.CreateChatCompletionStream(tools)                 │
│    ├─ AccumulateToolCalls() → 按 Index 累积                    │
│    └─ StreamChunk{Content, ToolCalls[]}                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│              OpenAI-Compatible API                              │
│  DeepSeek/SiliconFlow → Streaming Tool Calls                    │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 核心组件

#### 2.2.1 StreamChunk 数据结构

```go
// StreamChunk represents a single chunk in streaming response.
type StreamChunk struct {
    // Delta content (text)
    Content string

    // Delta tool calls (incremental)
    ToolCalls []DeltaToolCall

    // IsFinal indicates this is the last chunk
    IsFinal bool

    // Usage statistics (available in final chunk)
    Usage *LLMCallStats
}

// DeltaToolCall represents a streaming tool call delta.
type DeltaToolCall struct {
    // Index in the tool_calls array (for parallel calls)
    Index int

    // ID of the tool call (available in first chunk)
    ID string

    // Type (always "function" for OpenAI)
    Type string

    // Function name (available in first chunk)
    Name string

    // Function arguments (incremental JSON string)
    Arguments string

    // Complete indicates if this tool call is fully received
    Complete bool
}
```

#### 2.2.2 新增 LLM 接口方法

```go
// LLMService is the LLM service interface.
type LLMService interface {
    // Chat performs synchronous chat.
    Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error)

    // ChatStream performs streaming chat (text only).
    ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error)

    // ChatWithTools performs synchronous chat with tools.
    ChatWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (*ChatResponse, *LLMCallStats, error)

    // ChatStreamWithTools performs streaming chat with tools (NEW).
    // Returns chunk channel, stats channel, and error channel.
    ChatStreamWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (<-chan StreamChunk, <-chan *LLMCallStats, <-chan error)
}
```

### 2.3 事件流协议

#### 事件类型

| 事件类型 | 方向 | 触发时机 |
|:---------|:-----|:---------|
| `stream_start` | S→C | 流开始 |
| `thinking` | S→C | AI 思考中 |
| `tool_use` | S→C | 工具调用开始（实时） |
| `tool_result` | S→C | 工具调用结果 |
| `content_delta` | S→C | 内容增量 |
| `session_stats` | S→C | 会话统计 |
| `error` | S→C | 错误发生 |
| `stream_end` | S→C | 流结束 |

#### tool_use 事件结构

```json
{
  "type": "tool_use",
  "tool_name": "schedule_add",
  "tool_id": "call_abc123",
  "status": "running",
  "input_summary": "{\"title\":\"Team meeting\",\"start_time\":\"2026-02-09T10:00:00+08:00\"}",
  "timestamp": 1707350400000
}
```

---

## 3. 实现方案

### 3.1 分阶段交付

#### Phase 1: 核心 LLM 流式接口（1 周）

**目标**：实现 `ChatStreamWithTools` 方法

| 文件 | 改动 | 优先级 |
|:-----|:-----|:-------|
| `ai/llm.go` | 新增 `StreamChunk`、`DeltaToolCall` 类型 | P0 |
| `ai/llm.go` | 新增 `ChatStreamWithTools` 方法（~150 行） | P0 |
| `ai/llm_test.go` | 新增单元测试 | P0 |

**验收标准**：
- [ ] 流式工具调用检测准确率 > 98%
- [ ] 支持 DeepSeek 和 SiliconFlow
- [ ] 单元测试覆盖率 > 80%

#### Phase 2: Agent 层适配（1 周）

**目标**：Agent 使用流式工具调用

| 文件 | 改动 | 优先级 |
|:-----|:-----|:-------|
| `ai/agent/tool_adapter.go` | 修改 `RunWithCallback` 使用流式接口 | P0 |
| `ai/agent/streaming_executor.go` | 新增流式执行器（可选） | P1 |
| `ai/agent/scheduler_v2.go` | 优先使用流式接口 | P0 |
| `ai/agent/memo_parrot.go` | 移除模拟流式，使用真实流式 | P1 |

**验收标准**：
- [ ] 工具调用延迟降低 > 50%
- [ ] 保持工具调用准确性
- [ ] 向后兼容（同步接口仍可用）

#### Phase 3: 前端体验优化（1 周）

**目标**：优化 UI 展示

| 文件 | 改动 | 优先级 |
|:-----|:-----|:-------|
| `web/src/hooks/useStreamingToolCalls.ts` | 新增流式工具调用 Hook | P0 |
| `web/src/components/AIChat/ToolCallCard.tsx` | 优化实时状态展示 | P1 |
| `web/src/components/AIChat/StreamingIndicator.tsx` | 新增流式指示器 | P1 |

**验收标准**：
- [ ] UI 响应时间 < 100ms
- [ ] 工具调用展示延迟 < 200ms
- [ ] 用户反馈正面率 > 80%

### 3.2 核心实现逻辑

#### ChatStreamWithTools 实现

```go
func (s *llmService) ChatStreamWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (<-chan StreamChunk, <-chan *LLMCallStats, <-chan error) {
    chunkChan := make(chan StreamChunk, 10)
    statsChan := make(chan *LLMCallStats, 1)
    errChan := make(chan error, 1)

    go func() {
        defer close(chunkChan)
        defer close(statsChan)
        defer close(errChan)

        // Build request with tools
        openaiTools := make([]openai.Tool, len(tools))
        for i, t := range tools {
            openaiTools[i] = openai.Tool{
                Type: openai.ToolTypeFunction,
                Function: &openai.FunctionDefinition{
                    Name:       t.Name,
                    Description: t.Description,
                    Parameters: json.RawMessage(t.Parameters),
                },
            }
        }

        req := openai.ChatCompletionRequest{
            Model:         s.model,
            MaxTokens:     s.maxTokens,
            Temperature:   s.temperature,
            Messages:      convertMessages(messages),
            Tools:         openaiTools,
            StreamOptions: &openai.StreamOptions{IncludeUsage: true},
        }

        stream, err := s.client.CreateChatCompletionStream(ctx, req)
        if err != nil {
            errChan <- fmt.Errorf("create stream failed: %w", err)
            return
        }
        defer stream.Close()

        // Track accumulated tool calls by index
        type toolCallBuffer struct {
            id        string
            toolType  string
            name      string
            arguments strings.Builder
            complete  bool
        }
        toolCallBuffers := make(map[int]*toolCallBuffer)

        for {
            response, err := stream.Recv()
            if err != nil {
                if err == io.EOF || strings.Contains(err.Error(), "EOF") {
                    // Stream completed
                    statsChan <- &LLMCallStats{/* ... */}
                    return
                }
                errChan <- fmt.Errorf("stream recv failed: %w", err)
                return
            }

            if len(response.Choices) == 0 {
                continue
            }

            // Extract content delta
            contentDelta := response.Choices[0].Delta.Content

            // Extract tool call deltas
            var toolCallDeltas []DeltaToolCall
            for _, tc := range response.Choices[0].Delta.ToolCalls {
                idx := tc.Index

                // Initialize buffer if needed
                if toolCallBuffers[idx] == nil {
                    toolCallBuffers[idx] = &toolCallBuffer{}
                }

                buf := toolCallBuffers[idx]

                // Accumulate fields
                if tc.ID != "" {
                    buf.id = tc.ID
                }
                if tc.Type != "" {
                    buf.toolType = string(tc.Type)
                }
                if tc.Function.Name != "" {
                    buf.name = tc.Function.Name
                }
                if tc.Function.Arguments != "" {
                    buf.arguments.WriteString(tc.Function.Arguments)
                }

                // Check if complete (valid JSON)
                if json.Valid([]byte(buf.arguments.String())) {
                    buf.complete = true
                }

                // Send delta
                toolCallDeltas = append(toolCallDeltas, DeltaToolCall{
                    Index:     idx,
                    ID:        buf.id,
                    Type:      buf.toolType,
                    Name:      buf.name,
                    Arguments: buf.arguments.String(),
                    Complete:  buf.complete,
                })
            }

            // Send chunk
            chunkChan <- StreamChunk{
                Content:   contentDelta,
                ToolCalls: toolCallDeltas,
                IsFinal:   false,
            }

            // Check finish reason
            if response.Choices[0].FinishReason != "" {
                // Send final stats
                if response.Usage != nil {
                    statsChan <- &LLMCallStats{
                        PromptTokens:     response.Usage.PromptTokens,
                        CompletionTokens: response.Usage.CompletionTokens,
                        TotalTokens:      response.Usage.TotalTokens,
                    }
                }
                return
            }
        }
    }()

    return chunkChan, statsChan, errChan
}
```

#### Agent 流式执行

```go
func (a *Agent) ExecuteWithStreamingTools(ctx context.Context, input string, callback Callback) (string, error) {
    messages := []ai.Message{
        {Role: "system", Content: a.config.SystemPrompt},
        {Role: "user", Content: input},
    }

    for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
        // Send thinking event
        if iteration == 0 && callback != nil {
            callback(EventTypeThinking, "")
        }

        // Use streaming interface
        chunkChan, statsChan, errChan := a.llm.ChatStreamWithTools(ctx, messages, a.toolDescriptors())

        var contentBuilder strings.Builder
        var toolCallMap = make(map[int]*ToolCall)

        for {
            select {
            case chunk, ok := <-chunkChan:
                if !ok {
                    chunkChan = nil
                    continue
                }

                // Stream content
                if chunk.Content != "" {
                    contentBuilder.WriteString(chunk.Content)
                    if callback != nil {
                        callback(EventAnswer, chunk.Content)
                    }
                }

                // Accumulate tool calls
                for _, delta := range chunk.ToolCalls {
                    if existing, ok := toolCallMap[delta.Index]; ok {
                        // Append to existing
                        if delta.ID != "" {
                            existing.ID = delta.ID
                        }
                        if delta.Name != "" {
                            existing.Function.Name = delta.Name
                        }
                        if delta.Arguments != "" {
                            existing.Function.Arguments += delta.Arguments
                        }
                        existing.Complete = delta.Complete
                    } else {
                        // New tool call
                        newToolCall := &ToolCall{
                            ID:   delta.ID,
                            Type: delta.Type,
                            Function: FunctionCall{
                                Name:      delta.Name,
                                Arguments: delta.Arguments,
                            },
                            Complete: delta.Complete,
                        }
                        toolCallMap[delta.Index] = newToolCall

                        // Send tool_use event immediately
                        if callback != nil && delta.Name != "" {
                            meta := &EventMeta{
                                ToolName:     delta.Name,
                                Status:       "running",
                                InputSummary: delta.Arguments,
                            }
                            callback(EventTypeToolUse, &EventWithMeta{
                                EventType: EventTypeToolUse,
                                EventData:  delta.Arguments,
                                Meta:      meta,
                            })
                        }
                    }

                    // Execute complete tool calls
                    if delta.Complete && existing.Complete {
                        result := a.executeTool(ctx, existing.Function.Name, existing.Function.Arguments)

                        // Send tool_result event
                        if callback != nil {
                            meta := &EventMeta{
                                ToolName:      existing.Function.Name,
                                Status:        "success",
                                OutputSummary: result,
                            }
                            callback(EventTypeToolResult, &EventWithMeta{
                                EventType: EventTypeToolResult,
                                EventData:  result,
                                Meta:      meta,
                            })
                        }

                        // Add to message history
                        messages = append(messages, ai.Message{
                            Role:    "user",
                            Content: fmt.Sprintf("[Result from %s]: %s", existing.Function.Name, result),
                        })
                    }
                }

            case stats := <-statsChan:
                a.stats.RecordLLMCall(stats)

            case err := <-errChan:
                if err != nil {
                    return "", fmt.Errorf("LLM stream failed: %w", err)
                }
            }

            if chunkChan == nil && len(toolCallMap) == 0 {
                // No more chunks and no pending tool calls
                break
            }
        }

        // Check if we have a final answer
        finalContent := contentBuilder.String()
        if finalContent != "" && len(toolCallMap) == 0 {
            return finalContent, nil
        }
    }

    return "", fmt.Errorf("max iterations exceeded")
}
```

---

## 4. 风险与缓解

### 4.1 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|:-----|:-----|:-----|:---------|
| DeepSeek 流式工具调用格式不兼容 | 中 | 高 | 1. 提前验证 API 响应格式<br>2. 添加环境变量开关<br>3. 降级到同步接口 |
| 工具调用增量合并错误 | 中 | 高 | 1. 完善单元测试<br>2. 详细日志追踪<br>3. JSON 验证后再执行 |
| 并发竞争条件 | 低 | 中 | 1. Mutex 保护<br>2. 单元测试覆盖 |
| 性能下降 | 低 | 中 | 1. 基准测试<br>2. Buffer 优化 |

### 4.2 兼容性风险

| 风险 | 缓解措施 |
|:-----|:---------|
| 破坏现有功能 | 保留同步接口，Agent 优先使用流式，失败时降级 |
| 前端事件重复渲染 | 幂等性检查，状态机保护 |
| 依赖版本冲突 | 使用接口抽象，隔离第三方库变化 |

---

## 5. 测试计划

### 5.1 单元测试

```go
func TestChatStreamWithTools(t *testing.T) {
    tests := []struct {
        name           string
        messages       []Message
        tools          []ToolDescriptor
        wantToolCalls  []ToolCall
        wantContent    string
    }{
        {
            name: "single tool call",
            tools: []ToolDescriptor{{
                Name:        "schedule_add",
                Description: "Add a schedule",
                Parameters:  `{"type":"object","properties":{"title":{"type":"string"}}}`,
            }},
            wantToolCalls: []ToolCall{{
                Name: "schedule_add",
                Function: FunctionCall{
                    Name:      "schedule_add",
                    Arguments: `{"title":"Meeting"}`,
                },
            }},
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}

func TestToolCallAccumulation(t *testing.T) {
    // Test JSON accumulation across chunks
    chunks := []string{
        `{"ti`,
        `tle":"`,
        `"Team Meeting`,
        `"}`,
    }

    var builder strings.Builder
    for _, chunk := range chunks {
        builder.WriteString(chunk)
    }

    result := builder.String()
    if !json.Valid([]byte(result)) {
        t.Errorf("Invalid JSON: %s", result)
    }
}
```

### 5.2 集成测试

| 场景 | 测试步骤 | 预期结果 |
|:-----|:---------|:---------|
| 单一工具调用 | 发送 "明天上午10点开会" | 流式显示 tool_use → 立即显示结果 |
| 多工具串行 | 发送 "查询并添加日程" | 依次显示每个工具调用 |
| 工具调用失败 | 发送无效输入 | 显示错误但不中断流 |
| LLM 中断 | 用户取消请求 | 工具调用终止，状态清理 |

### 5.3 性能测试

```bash
# 基准测试
go test -bench=BenchmarkChatStreamWithTools -benchmem ./ai

# 对比同步 vs 流式
go test -bench=BenchmarkChatWithTools -benchmem ./ai
```

---

## 6. 监控指标

### 6.1 关键指标

| 指标 | 含义 | 告警阈值 |
|:-----|:-----|:---------|
| `streaming_tool_detection_rate` | 工具调用检测成功率 | < 95% |
| `streaming_tool_execution_latency_p50` | 工具执行延迟 (P50) | > 1s |
| `streaming_tool_execution_latency_p95` | 工具执行延迟 (P95) | > 3s |
| `streaming_ui_jank_frames` | UI 卡顿帧数 | > 5/min |
| `streaming_rollback_count` | 降级次数 | > 10/min |

### 6.2 日志示例

```json
{
  "level": "info",
  "msg": "streaming_tool_call_detected",
  "tool_name": "schedule_add",
  "detection_latency_ms": 50,
  "stream_position": 1234
}

{
  "level": "info",
  "msg": "streaming_tool_executed",
  "tool_name": "schedule_add",
  "execution_latency_ms": 450,
  "is_parallel": false
}
```

---

## 7. 交付清单

### Phase 1 交付物

- [ ] `ai/llm.go` - 新增 `ChatStreamWithTools` 方法
- [ ] `ai/llm_test.go` - 单元测试
- [ ] 单元测试通过
- [ ] 基准测试通过

### Phase 2 交付物

- [ ] `ai/agent/tool_adapter.go` - 流式执行器
- [ ] `ai/agent/scheduler_v2.go` - 使用流式接口
- [ ] 集成测试通过
- [ ] 向后兼容验证

### Phase 3 交付物

- [ ] `web/src/hooks/useStreamingToolCalls.ts` - 前端 Hook
- [ ] UI 组件优化
- [ ] 端到端测试通过
- [ ] 用户验收

---

## 8. 参考资料

### 8.1 内部文档

- [ARCHITECTURE.md](../dev-guides/ARCHITECTURE.md) - 项目架构
- [FRONTEND.md](../dev-guides/FRONTEND.md) - 前端开发指南
- [BACKEND_DB.md](../dev-guides/BACKEND_DB.md) - 后端开发指南

### 8.2 外部文档

- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI Go 客户端

---

*文档最后更新: 2026-02-08*
