# AI Observability (`ai/observability`)

`observability` 包为 AI 模块提供全链路的可观测性支持，包括日志、指标监控和分布式追踪。

## 模块组成

### 1. `logging`
基于 `slog` 的结构化日志封装。
*   统一日志格式（JSON/Text）。
*   支持上下文注入（TraceID, SpanID）。

### 2. `metrics`
基于 Prometheus 的指标收集系统。
*   **核心指标**:
    *   `ai_request_total`: 请求总数计数器。
    *   `ai_request_duration_seconds`: 请求耗时直方图。
    *   `ai_token_usage`: Token 消耗计数器（Input/Output）。
    *   `ai_error_total`: 错误计数器。
*   **持久化 (`Persister`)**: 支持定期将内存中的指标快照保存到磁盘，防止重启丢失。

### 3. `tracing`
基于 OpenTelemetry 的链路追踪。
*   **`Tracer`**: 自动为 AI 请求创建 Span，记录关键步骤（如 LLM 调用、RAG 检索）的耗时。

## 架构图

```mermaid
graph TD
    App[AI Service] --> Log[Logging (slog)]
    App --> Trace[Tracing (OTEL)]
    App --> Met[Metrics (Prometheus)]
    
    Met --> Persist[Persister (Disk/DB)]
    Met --> Scrape[Prometheus Server]
    
    Trace --> Export[Jaeger/Tempo]
```
