# 分布式链路追踪系统

> **实现状态**: ✅ 完成 (v0.94.0) | **位置**: `ai/tracing/`

## 概述

分布式链路追踪系统提供 AI 代理系统的端到端可观测性，追踪请求从用户输入到 AI 响应的完整路径，帮助诊断性能瓶颈和异常。

### 设计目标

| 指标 | 目标值 |
|:-----|:------|
| 追踪开销 | <5% |
| 采样率 | 可配置（默认 10%） |
| 保留期 | 7 天 |
| 延迟精度 | ±1ms |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                     用户请求                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  TraceContext                           │
│  - TraceID: 全局追踪 ID                                  │
│  - SpanID: 当前操作 ID                                   │
│  - ParentSpanID: 父操作 ID                               │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                    路由层                                │
│  Span: /api/v1/chat/stream (HTTP)                        │
│  ├─ ParseRequest                                         │
│  ├─ Authenticate                                         │
│  └─ Route                                                │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   代理层                                │
│  Span: ChatRouter.Route()                               │
│  ├─ ClassifyIntent (LLM Call)                           │
│  ├─ SelectParrot                                        │
│  └─ Execute                                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   工具层                                │
│  Span: memo_search                                      │
│  ├─ VectorSearch                                        │
│  ├─ BM25Search                                          │
│  └─ MergeResults                                        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  LLM 层                                 │
│  Span: LLM.Chat                                         │
│  ├─ BuildPrompt                                         │
│  ├─ APICall                                             │
│  └─ ParseResponse                                       │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                 TraceExporter                           │
│  导出到 Jaeger/Zipkin/OFTL                             │
└─────────────────────────────────────────────────────────┘
```

---

## 核心组件

### TraceContext (`context.go`)

```go
type TraceContext struct {
    TraceID      string
    SpanID       string
    ParentSpanID string
    Baggage      map[string]string
    StartTime    time.Time
    Tags         map[string]string
}

// FromContext 从 context 中提取 TraceContext
func FromContext(ctx context.Context) *TraceContext

// NewContext 创建带有 TraceContext 的 context
func NewContext(parent context.Context, tc *TraceContext) context.Context

// StartSpan 创建子 Span
func (tc *TraceContext) StartSpan(name string) *Span
```

### Span (`span.go`)

```go
type Span struct {
    TraceID      string
    SpanID       string
    ParentSpanID string
    Name         string
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration
    Tags         map[string]string
    Logs         []LogEntry
    Status       SpanStatus
}

type SpanStatus int
const (
    StatusOK SpanStatus = iota
    StatusCanceled
    StatusError
)

// End 结束 Span
func (s *Span) End()

// SetTag 设置标签
func (s *Span) SetTag(key, value string)

// RecordError 记录错误
func (s *Span) RecordError(err error)
```

### Exporter (`exporter.go`)

```go
type Exporter interface {
    Export(span *Span) error
    Flush() error
    Shutdown() error
}

// OTLPExporter OpenTelemetry Protocol 导出器
type OTLPExporter struct {
    endpoint string
    client   *http.Client
    batch    []*Span
}

// ConsoleExporter 控制台导出器（开发用）
type ConsoleExporter struct {
    writer io.Writer
}
```

---

## 使用方式

### 基本使用

```go
import "divinesense/ai/tracing"

// 创建根 Span
ctx := context.Background()
tc, span := tracing.StartRoot(ctx, "handle_chat")
defer span.End()

// 添加标签
span.SetTag("user_id", "123")
span.SetTag("agent_type", "memo")

// 创建子 Span
检索Span := tc.StartSpan("memo_search")
defer 检索Span.End()

// 记录错误
if err != nil {
    检索Span.RecordError(err)
}
```

### 中间件集成

```go
func TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 从请求头提取 TraceContext
        tc := tracing.ExtractFromHTTP(r)

        // 创建 Span
        span := tc.StartSpan("http_request")
        defer span.End()

        span.SetTag("http.method", r.Method)
        span.SetTag("http.path", r.URL.Path)

        // 注入到 context
        ctx := tracing.NewContext(r.Context(), tc)

        // 调用下一个处理器
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### AI 代理集成

```go
func (p *MemoParrot) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    tc := tracing.FromContext(ctx)

    // 代理执行 Span
    span := tc.StartSpan("memo_parrot.chat")
    defer span.End()
    span.SetTag("query", req.Message)

    // 检索 Span
    检索Span := tc.StartSpan("memo_search")
    results, err := p.retriever.Search(req.Message)
    检索Span.End()
    if err != nil {
        检索Span.RecordError(err)
        return nil, err
    }

    // LLM Span
    LLMSpan := tc.StartSpan("llm.chat")
    response, err := p.llm.Chat(ctx, req)
    LLMSpan.End()

    return response, err
}
```

---

## 传播协议

### HTTP 头

| 头名称 | 格式 | 示例 |
|:-------|:-----|:-----|
| `Traceparent` | `00-traceid-spanid-flags` | `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01` |
| `Tracestate` | `key=value;...` | `rojo=00f067aa0ba902b7,congo=t61rcWkgMzE` |

### gRPC Metadata

```go
md := metadata.Pairs(
    "traceparent", fmt.Sprintf("00-%s-%s-01", traceID, spanID),
)
ctx = metadata.NewOutgoingContext(ctx, md)
```

---

## 配置选项

| 环境变量 | 默认值 | 说明 |
|:---------|:------|:-----|
| `DIVINESENSE_TRACING_ENABLED` | `true` | 是否启用追踪 |
| `DIVINESENSE_TRACING_SAMPLER` | `0.1` | 采样率（0-1） |
| `DIVINESENSE_TRACING_EXPORTER` | `console` | 导出器类型 |
| `DIVINESENSE_TRACING_ENDPOINT` | - | OTLP 端点 |
| `DIVINESENSE_TRACING_BATCH_SIZE` | `100` | 批量导出大小 |
| `DIVINESENSE_TRACING_TIMEOUT` | `30s` | 导出超时 |

---

## 导出格式

### OTLP JSON

```json
{
  "resourceSpans": [
    {
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "divinesense"}},
          {"key": "service.version", "value": {"stringValue": "v0.94.0"}}
        ]
      },
      "scopeSpans": [
        {
          "scope": {"name": "ai.agent"},
          "spans": [
            {
              "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
              "spanId": "00f067aa0ba902b7",
              "parentSpanId": "00f067aa0ba90200",
              "name": "memo_search",
              "startTimeUnixNano": 1583134921000000000,
              "endTimeUnixNano": 1583134921500000000,
              "kind": 3,
              "status": {"code": 1},
              "attributes": [
                {"key": "user_id", "value": {"stringValue": "123"}},
                {"key": "query", "value": {"stringValue": "今天的会议"}}
              ],
              "events": [
                {
                  "timeUnixNano": 1583134921200000000,
                  "name": "vector_search",
                  "attributes": [
                    {"key": "results", "value": {"intValue": 5}}
                  ]
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

---

## 可视化

### Jaeger 集成

```bash
# 启动 Jaeger
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14250:14250 \
  jaegertracing/all-in-one:latest

# 配置 DivineSense
export DIVINESENSE_TRACING_EXPORTER=otlp
export DIVINESENSE_TRACING_ENDPOINT=http://localhost:14250
```

### 查看追踪

访问 Jaeger UI: `http://localhost:16686`

搜索选项：
- **Service**: `divinesense`
- **Operation**: `handle_chat`
- **Tags**: `user_id=123`, `agent_type=memo`

---

## 监控指标

```go
type TracingMetrics struct {
    SpansStarted    int64
    SpansCompleted  int64
    SpansDropped    int64
    AvgLatency      int64
    ErrorRate       float64
    SamplingRate    float64
}
```

### Prometheus 指标

```
# Span 计数
tracing_spans_total{service="divinesense",operation="memo_search"} 1234

# Span 延迟
tracing_span_duration_ms{service="divinesense",operation="memo_search",quantile="0.99"} 45

# 错误率
tracing_span_errors_total{service="divinesense",operation="memo_search"} 12
```

---

## 性能考虑

1. **异步导出**：使用批量异步导出，避免阻塞请求
2. **采样策略**：生产环境使用 10% 采样率
3. **内存限制**：每个 Span 限制 1KB，超过则截断
4. **批量大小**：默认 100 个 Span 批量导出

---

## 测试

```bash
# 运行追踪测试
go test ./ai/tracing/ -v

# 性能基准测试
go test ./ai/tracing/ -bench=. -benchmem
```

---

## 相关文档

- [OpenTelemetry 规范](https://opentelemetry.io/docs/reference/specification/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OTLP 规范](https://opentelemetry.io/docs/reference/specification/protocol/otlp/)
