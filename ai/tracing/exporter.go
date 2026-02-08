// Package tracing provides trace exporters for various backends.
package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
)

// Exporter exports traces to various backends.
type Exporter interface {
	// Export exports a trace asynchronously.
	Export(trace *TracingContext)
}

// LogExporter exports traces to structured logs.
type LogExporter struct {
	logger *slog.Logger
}

// NewLogExporter creates a new log exporter.
func NewLogExporter() *LogExporter {
	return &LogExporter{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// Export logs the trace as structured JSON.
func (e *LogExporter) Export(trace *TracingContext) {
	if trace == nil {
		return
	}

	duration := trace.Duration()
	logger := e.logger

	logger.Info("ai_trace",
		"trace_id", trace.TraceID,
		"operation", trace.OperationName,
		"status", trace.Status,
		"duration_ms", duration.Milliseconds(),
		"phases", len(trace.Phases),
		"tool_calls", len(trace.ToolCalls),
		"llm_calls", len(trace.LLMCalls),
		"total_tokens", trace.TotalTokens(),
		"cached_tokens", trace.CachedTokens(),
	)

	// Log slow phases (>100ms)
	for _, phase := range trace.Phases {
		if phase.Duration.Milliseconds() > 100 {
			logger.Warn("slow_phase",
				"trace_id", trace.TraceID,
				"phase", phase.Name,
				"duration_ms", phase.Duration.Milliseconds(),
			)
		}
	}

	// Log failed tool calls
	for _, call := range trace.ToolCalls {
		if call.Status == StatusError {
			logger.Error("tool_error",
				"trace_id", trace.TraceID,
				"tool", call.Name,
				"error", call.Error,
			)
		}
	}
}

// JaegerExporter exports traces to Jaeger.
type JaegerExporter struct {
	endpoint    string
	serviceName string
	batcher     *Batcher
	httpClient  *http.Client
}

// JaegerConfig configures the Jaeger exporter.
type JaegerConfig struct {
	// Endpoint is the Jaeger HTTP endpoint.
	Endpoint string

	// ServiceName identifies the service in Jaeger.
	ServiceName string

	// BatchSize determines how many spans to batch before sending.
	BatchSize int

	// BatchTimeout is the maximum time to wait before sending a batch.
	BatchTimeout time.Duration

	// MaxQueueSize is the maximum number of pending batches.
	MaxQueueSize int
}

// DefaultJaegerConfig returns default Jaeger configuration.
func DefaultJaegerConfig() JaegerConfig {
	return JaegerConfig{
		Endpoint:     "http://localhost:14268/api/traces",
		ServiceName:  "divinesense",
		BatchSize:    100,
		BatchTimeout: 5 * time.Second,
		MaxQueueSize: 1000,
	}
}

// NewJaegerExporter creates a new Jaeger exporter.
func NewJaegerExporter(cfg JaegerConfig) *JaegerExporter {
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultJaegerConfig().Endpoint
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = DefaultJaegerConfig().ServiceName
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultJaegerConfig().BatchSize
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = DefaultJaegerConfig().BatchTimeout
	}
	if cfg.MaxQueueSize <= 0 {
		cfg.MaxQueueSize = DefaultJaegerConfig().MaxQueueSize
	}

	return &JaegerExporter{
		endpoint:    cfg.Endpoint,
		serviceName: cfg.ServiceName,
		batcher:     NewBatcher(cfg.BatchSize, cfg.BatchTimeout, cfg.MaxQueueSize),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Export exports a trace to Jaeger via the batcher.
func (e *JaegerExporter) Export(trace *TracingContext) {
	if trace == nil {
		return
	}

	span := e.convertToJaegerSpan(trace)
	e.batcher.Add(span, func(batch []*JaegerSpan) {
		e.sendBatch(batch)
	})
}

// convertToJaegerSpan converts a TracingContext to a Jaeger span.
func (e *JaegerExporter) convertToJaegerSpan(trace *TracingContext) *JaegerSpan {
	now := time.Now().UnixMicro() / 1000 // Convert to microseconds

	span := &JaegerSpan{
		TraceID:       toJaegerID(trace.TraceID),
		SpanID:        toJaegerID(trace.SpanID),
		ParentSpanID:  toJaegerID(trace.ParentSpanID),
		OperationName: trace.OperationName,
		StartTime:     now - trace.Duration().Microseconds(),
		Duration:      trace.Duration().Microseconds(),
		Tags:          make([]JaegerTag, 0, 8),
		Logs:          make([]JaegerLog, 0, 4),
	}

	// Add status tag
	span.Tags = append(span.Tags, JaegerTag{
		Key:   "status",
		VType: "string",
		Value: statusToString(trace.Status),
	})

	// Add tags from trace
	for k, v := range trace.Tags {
		span.Tags = append(span.Tags, JaegerTag{
			Key:   k,
			VType: "string",
			Value: v,
		})
	}

	// Add metadata as tags
	for k, v := range trace.Metadata {
		span.Tags = append(span.Tags, JaegerTag{
			Key:   k,
			VType: "string",
			Value: v,
		})
	}

	// Add phases as logs
	for _, phase := range trace.Phases {
		span.Logs = append(span.Logs, JaegerLog{
			Timestamp: phase.StartTime.UnixMicro() / 1000,
			Fields: []JaegerTag{
				{Key: "event", VType: "string", Value: "phase"},
				{Key: "phase_name", VType: "string", Value: phase.Name},
				{Key: "duration_ms", VType: "int64", Value: phase.Duration.Milliseconds()},
			},
		})
	}

	// Add tool calls as logs
	for _, call := range trace.ToolCalls {
		span.Logs = append(span.Logs, JaegerLog{
			Timestamp: call.StartTime.UnixMicro() / 1000,
			Fields: []JaegerTag{
				{Key: "event", VType: "string", Value: "tool_call"},
				{Key: "tool_name", VType: "string", Value: call.Name},
				{Key: "tool_type", VType: "string", Value: call.ToolType},
				{Key: "duration_ms", VType: "int64", Value: call.Duration.Milliseconds()},
			},
		})
	}

	return span
}

// sendBatch sends a batch of spans to Jaeger.
func (e *JaegerExporter) sendBatch(spans []*JaegerSpan) {
	if len(spans) == 0 {
		return
	}

	body := JaegerBatch{
		Spans: spans,
		Process: JaegerProcess{
			ServiceName: e.serviceName,
			Tags: []JaegerTag{
				{Key: "hostname", VType: "string", Value: getHostnameStr()},
				{Key: "go_version", VType: "string", Value: runtime.Version()},
			},
		},
	}

	payload := []JaegerBatch{body}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal jaeger batch", "error", err)
		return
	}

	req, err := http.NewRequest("POST", e.endpoint, bytes.NewReader(data))
	if err != nil {
		slog.Error("failed to create jaeger request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		slog.Error("failed to send jaeger batch", "error", err)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("failed to read jaeger error response", "error", err)
		} else {
			slog.Error("jaeger returned error", "status", resp.StatusCode, "body", string(body))
		}
	}
}

// Jaeger span data structures
type JaegerSpan struct {
	TraceID       string      `json:"traceID"`
	SpanID        string      `json:"spanID"`
	ParentSpanID  string      `json:"parentSpanID,omitempty"`
	OperationName string      `json:"operationName"`
	StartTime     int64       `json:"startTime"`
	Duration      int64       `json:"duration"`
	Tags          []JaegerTag `json:"tags"`
	Logs          []JaegerLog `json:"logs"`
}

type JaegerTag struct {
	Key   string `json:"key"`
	VType string `json:"type,omitempty"`
	Value any    `json:"value"`
}

type JaegerLog struct {
	Timestamp int64       `json:"timestamp"`
	Fields    []JaegerTag `json:"fields"`
}

type JaegerProcess struct {
	ServiceName string      `json:"serviceName"`
	Tags        []JaegerTag `json:"tags"`
}

type JaegerBatch struct {
	Spans   []*JaegerSpan `json:"spans"`
	Process JaegerProcess `json:"process"`
}

// toJaegerID converts a UUID to Jaeger span ID format.
func toJaegerID(id string) string {
	if id == "" {
		return ""
	}
	// Jaeger expects hex strings, our UUIDs are already hex
	// We just need to format them correctly
	uuid, err := parseUUID(id)
	if err != nil {
		return id
	}
	return fmt.Sprintf("%016x", uuid)
}

func parseUUID(id string) (uint64, error) {
	// Simple UUID parsing - take last 16 hex chars
	var result uint64
	for _, c := range id {
		switch {
		case c >= '0' && c <= '9':
			result = result<<4 | uint64(c-'0')
		case c >= 'a' && c <= 'f':
			result = result<<4 | uint64(c-'a'+10)
		case c >= 'A' && c <= 'F':
			result = result<<4 | uint64(c-'A'+10)
		}
	}
	return result, nil
}

func statusToString(status TraceStatus) string {
	switch status {
	case StatusOK:
		return "ok"
	case StatusError:
		return "error"
	case StatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

func getHostnameStr() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// Batcher batches spans before sending them to the exporter.
type Batcher struct {
	batchSize    int
	batchTimeout time.Duration
	maxQueueSize int
	queue        chan *batchItem
	once         sync.Once
	wg           sync.WaitGroup
}

type batchItem struct {
	span    *JaegerSpan
	onFlush func([]*JaegerSpan)
}

type batch struct {
	spans []*JaegerSpan
	flush func([]*JaegerSpan)
}

// NewBatcher creates a new batcher.
func NewBatcher(batchSize int, batchTimeout time.Duration, maxQueueSize int) *Batcher {
	b := &Batcher{
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		maxQueueSize: maxQueueSize,
		queue:        make(chan *batchItem, maxQueueSize),
	}

	b.start()

	return b
}

// Add adds a span to the batch.
func (b *Batcher) Add(span *JaegerSpan, onFlush func([]*JaegerSpan)) {
	select {
	case b.queue <- &batchItem{span: span, onFlush: onFlush}:
		// Queued successfully
	default:
		slog.Warn("batcher queue full, dropping span")
	}
}

// start starts the batch processor goroutine.
func (b *Batcher) start() {
	b.once.Do(func() {
		b.wg.Add(1)
		go b.process()
	})
}

// process processes batches of spans.
func (b *Batcher) process() {
	defer b.wg.Done()

	currentBatch := &batch{
		spans: make([]*JaegerSpan, 0, b.batchSize),
	}

	ticker := time.NewTicker(b.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case item, ok := <-b.queue:
			if !ok {
				// Queue closed, flush remaining
				if len(currentBatch.spans) > 0 {
					currentBatch.flush(currentBatch.spans)
				}
				return
			}

			currentBatch.spans = append(currentBatch.spans, item.span)
			currentBatch.flush = item.onFlush

			// Flush if batch is full
			if len(currentBatch.spans) >= b.batchSize {
				currentBatch.flush(currentBatch.spans)
				currentBatch = &batch{
					spans: make([]*JaegerSpan, 0, b.batchSize),
				}
			}

		case <-ticker.C:
			// Flush on timeout
			if len(currentBatch.spans) > 0 {
				currentBatch.flush(currentBatch.spans)
				currentBatch = &batch{
					spans: make([]*JaegerSpan, 0, b.batchSize),
				}
			}
		}
	}
}

// Close closes the batcher and flushes remaining spans.
func (b *Batcher) Close() {
	close(b.queue)
	b.wg.Wait()
}

// OTLPExporter exports traces in OpenTelemetry format.
type OTLPExporter struct {
	endpoint   string
	batcher    *Batcher
	httpClient *http.Client
}

// OTLPConfig configures the OTLP exporter.
type OTLPConfig struct {
	// Endpoint is the OTLP HTTP endpoint.
	Endpoint string

	// Headers to include in requests.
	Headers map[string]string

	// BatchSize determines how many spans to batch before sending.
	BatchSize int

	// BatchTimeout is the maximum time to wait before sending a batch.
	BatchTimeout time.Duration
}

// DefaultOTLPConfig returns default OTLP configuration.
func DefaultOTLPConfig() OTLPConfig {
	return OTLPConfig{
		Endpoint:     "http://localhost:4318/v1/traces",
		BatchSize:    100,
		BatchTimeout: 5 * time.Second,
	}
}

// NewOTLPExporter creates a new OTLP exporter.
func NewOTLPExporter(cfg OTLPConfig) *OTLPExporter {
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultOTLPConfig().Endpoint
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultOTLPConfig().BatchSize
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = DefaultOTLPConfig().BatchTimeout
	}

	return &OTLPExporter{
		endpoint: cfg.Endpoint,
		batcher:  NewBatcher(cfg.BatchSize, cfg.BatchTimeout, 1000),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Export exports a trace in OTLP format.
func (e *OTLPExporter) Export(trace *TracingContext) {
	if trace == nil {
		return
	}

	// TODO: Implement OTLP format conversion
	// For now, just log the trace
	slog.Debug("otlp export", "trace_id", trace.TraceID, "operation", trace.OperationName)
}

// CompositeExporter exports traces to multiple exporters.
type CompositeExporter struct {
	exporters []Exporter
}

// NewCompositeExporter creates a new composite exporter.
func NewCompositeExporter(exporters ...Exporter) *CompositeExporter {
	return &CompositeExporter{
		exporters: exporters,
	}
}

// Export exports the trace to all exporters.
func (e *CompositeExporter) Export(trace *TracingContext) {
	if trace == nil {
		return
	}

	var wg sync.WaitGroup
	for _, exporter := range e.exporters {
		wg.Add(1)
		go func(exp Exporter) {
			defer wg.Done()
			exp.Export(trace)
		}(exporter)
	}
	wg.Wait()
}

// CachedExporter wraps an exporter with cache integration.
type CachedExporter struct {
	exporter Exporter
	cache    *cache.Service
}

// NewCachedExporter creates a new cached exporter.
func NewCachedExporter(exporter Exporter, cache *cache.Service) *CachedExporter {
	return &CachedExporter{
		exporter: exporter,
		cache:    cache,
	}
}

// Export exports the trace with caching support.
func (e *CachedExporter) Export(trace *TracingContext) {
	if trace == nil {
		return
	}

	// Store trace summary in cache for quick lookup
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	summary := map[string]any{
		"trace_id":    trace.TraceID,
		"operation":   trace.OperationName,
		"duration_ms": trace.Duration().Milliseconds(),
		"status":      trace.Status,
	}

	// Cache the trace summary
	summaryBytes, err := json.Marshal(summary)
	if err != nil {
		slog.Error("failed to marshal trace summary", "error", err)
	} else if err := e.cache.Set(ctx, "trace:"+trace.TraceID, summaryBytes, 5*time.Minute); err != nil {
		slog.Error("failed to cache trace summary", "trace_id", trace.TraceID, "error", err)
	}

	// Export to underlying exporter
	e.exporter.Export(trace)
}
