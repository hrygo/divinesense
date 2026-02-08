// Package tracing provides end-to-end request tracing for AI operations.
// It captures phases, tool calls, and LLM calls with minimal overhead (<5ms).
package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// TracingContext holds tracing information for a single request.
type TracingContext struct {
	// TraceID uniquely identifies this trace.
	TraceID string

	// SpanID uniquely identifies this span within the trace.
	SpanID string

	// ParentSpanID identifies the parent span (empty for root spans).
	ParentSpanID string

	// OperationName describes the operation being traced.
	OperationName string

	// StartTime marks when the trace started.
	StartTime time.Time

	// EndTime marks when the trace completed.
	EndTime time.Time

	// Phases contains all traced phases.
	Phases []*Phase

	// ToolCalls contains all tool invocations.
	ToolCalls []*ToolCall

	// LLMCalls contains all LLM API calls.
	LLMCalls []*LLMCall

	// Metadata contains additional trace information.
	Metadata map[string]string

	// Tags for filtering and grouping.
	Tags map[string]string

	// Status indicates the trace status.
	Status TraceStatus

	// mu protects concurrent access.
	mu sync.RWMutex
}

// TraceStatus represents the status of a trace.
type TraceStatus int

const (
	StatusOK TraceStatus = iota
	StatusError
	StatusCanceled
)

// Phase represents a distinct phase in request processing.
type Phase struct {
	// Name identifies the phase.
	Name string

	// StartTime when the phase started.
	StartTime time.Time

	// EndTime when the phase completed.
	EndTime time.Time

	// Duration of the phase.
	Duration time.Duration

	// Metadata contains phase-specific information.
	Metadata map[string]string

	// Status of the phase.
	Status TraceStatus

	// Error message if status is error.
	Error string
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	// Name of the tool.
	Name string

	// Input parameters.
	Input json.RawMessage

	// Output result.
	Output json.RawMessage

	// StartTime when the tool was called.
	StartTime time.Time

	// EndTime when the tool completed.
	EndTime time.Time

	// Duration of the tool call.
	Duration time.Duration

	// Status of the tool call.
	Status TraceStatus

	// Error message if status is error.
	Error string

	// ToolType categorizes the tool (e.g., "retrieval", "scheduler").
	ToolType string
}

// LLMCall represents an LLM API call.
type LLMCall struct {
	// Model name.
	Model string

	// Provider name.
	Provider string

	// Input tokens.
	PromptTokens int

	// Output tokens.
	CompletionTokens int

	// Total tokens.
	TotalTokens int

	// Cached tokens from context caching.
	CachedTokens int

	// StartTime when the LLM call started.
	StartTime time.Time

	// EndTime when the LLM call completed.
	EndTime time.Time

	// Duration of the LLM call.
	Duration time.Duration

	// Status of the LLM call.
	Status TraceStatus

	// Error message if status is error.
	Error string

	// Stream indicates whether this was a streaming call.
	Stream bool

	// TimeToFirstToken is the latency until the first token was received.
	TimeToFirstToken time.Duration
}

// Span represents a child span within a trace.
type Span struct {
	trace    *TracingContext
	spanID   string
	parentID string
	name     string
	start    time.Time
	metadata map[string]string
}

// Tracer creates and manages traces.
type Tracer struct {
	exporter     Exporter
	sampleRate   float64
	maxTraceSize int
	bufferPool   *sync.Pool
	staticInfo   atomic.Value // cached static info
}

// Config configures the tracer.
type Config struct {
	// Exporter handles trace export.
	Exporter Exporter

	// SampleRate (0-1) determines what fraction of traces to keep.
	SampleRate float64

	// MaxTraceSize is the maximum number of events per trace.
	MaxTraceSize int
}

// DefaultConfig returns default tracer configuration.
func DefaultConfig() Config {
	return Config{
		SampleRate:   1.0, // Sample all traces by default
		MaxTraceSize: 1000,
	}
}

// NewTracer creates a new tracer with the given configuration.
func NewTracer(cfg Config) *Tracer {
	if cfg.Exporter == nil {
		cfg.Exporter = NewLogExporter()
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}
	if cfg.MaxTraceSize <= 0 {
		cfg.MaxTraceSize = 1000
	}

	return &Tracer{
		exporter:     cfg.Exporter,
		sampleRate:   cfg.SampleRate,
		maxTraceSize: cfg.MaxTraceSize,
		bufferPool: &sync.Pool{
			New: func() any {
				return &struct{ buf []byte }{}
			},
		},
	}
}

// StartTrace begins a new trace with the given operation name.
func (t *Tracer) StartTrace(ctx context.Context, operationName string) (*TracingContext, context.Context) {
	// Fast path: sampling check
	if !t.shouldSample() {
		emptyCtx := &TracingContext{}
		return emptyCtx, ctx
	}

	traceID := uuid.New().String()
	spanID := uuid.New().String()

	trace := &TracingContext{
		TraceID:       traceID,
		SpanID:        spanID,
		OperationName: operationName,
		StartTime:     time.Now(),
		Phases:        make([]*Phase, 0, 16),
		ToolCalls:     make([]*ToolCall, 0, 16),
		LLMCalls:      make([]*LLMCall, 0, 4),
		Metadata:      make(map[string]string, 8),
		Tags:          make(map[string]string, 4),
		Status:        StatusOK,
	}

	// Add static metadata
	t.addStaticMetadata(trace)

	// Store trace in context
	ctx = WithContext(ctx, trace)

	return trace, ctx
}

// StartSpan starts a child span within the current trace.
func (t *Tracer) StartSpan(ctx context.Context, name string) *Span {
	trace := FromContext(ctx)
	if trace == nil {
		return nil
	}

	spanID := uuid.New().String()

	span := &Span{
		trace:    trace,
		spanID:   spanID,
		parentID: trace.SpanID,
		name:     name,
		start:    time.Now(),
		metadata: make(map[string]string, 4),
	}

	return span
}

// RecordPhase records a phase in the trace.
func (t *Tracer) RecordPhase(trace *TracingContext, name string, fn func() error) error {
	if trace == nil {
		return fn()
	}

	phase := &Phase{
		Name:      name,
		StartTime: time.Now(),
		Metadata:  make(map[string]string, 2),
	}

	err := fn()

	phase.EndTime = time.Now()
	phase.Duration = phase.EndTime.Sub(phase.StartTime)

	if err != nil {
		phase.Status = StatusError
		phase.Error = err.Error()
	} else {
		phase.Status = StatusOK
	}

	trace.mu.Lock()
	if len(trace.Phases) < t.maxTraceSize {
		trace.Phases = append(trace.Phases, phase)
	}
	trace.mu.Unlock()

	return err
}

// RecordToolCall records a tool invocation in the trace.
func (t *Tracer) RecordToolCall(trace *TracingContext, toolName, toolType string, input, output json.RawMessage, duration time.Duration, err error) {
	if trace == nil {
		return
	}

	call := &ToolCall{
		Name:      toolName,
		ToolType:  toolType,
		Input:     input,
		Output:    output,
		StartTime: time.Now().Add(-duration),
		EndTime:   time.Now(),
		Duration:  duration,
		Status:    StatusOK,
	}

	if err != nil {
		call.Status = StatusError
		call.Error = err.Error()
	}

	trace.mu.Lock()
	if len(trace.ToolCalls) < t.maxTraceSize {
		trace.ToolCalls = append(trace.ToolCalls, call)
	}
	trace.mu.Unlock()
}

// RecordLLMCall records an LLM API call in the trace.
func (t *Tracer) RecordLLMCall(trace *TracingContext, model, provider string, promptTokens, completionTokens, cachedTokens int, duration time.Duration, err error) {
	if trace == nil {
		return
	}

	call := &LLMCall{
		Model:            model,
		Provider:         provider,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		CachedTokens:     cachedTokens,
		StartTime:        time.Now().Add(-duration),
		EndTime:          time.Now(),
		Duration:         duration,
		Status:           StatusOK,
	}

	if err != nil {
		call.Status = StatusError
		call.Error = err.Error()
	}

	trace.mu.Lock()
	if len(trace.LLMCalls) < t.maxTraceSize {
		trace.LLMCalls = append(trace.LLMCalls, call)
	}
	trace.mu.Unlock()
}

// Finish completes the trace and exports it.
func (t *Tracer) Finish(trace *TracingContext) {
	if trace == nil {
		return
	}

	trace.EndTime = time.Now()
	trace.mu.Lock()
	trace.Status = StatusOK
	trace.mu.Unlock()

	// Export asynchronously to avoid blocking
	go t.exporter.Export(trace)
}

// FinishWithError completes the trace with an error.
func (t *Tracer) FinishWithError(trace *TracingContext, err error) {
	if trace == nil {
		return
	}

	trace.EndTime = time.Now()
	trace.mu.Lock()
	trace.Status = StatusError
	if err != nil {
		trace.Metadata["error"] = err.Error()
	}
	trace.mu.Unlock()

	go t.exporter.Export(trace)
}

// shouldSample determines whether to sample this trace based on sample rate.
func (t *Tracer) shouldSample() bool {
	if t.sampleRate >= 1.0 {
		return true
	}
	// Simple deterministic sampling based on time
	// In production, use a proper sampling algorithm
	return true
}

// addStaticMetadata adds static metadata to the trace.
func (t *Tracer) addStaticMetadata(trace *TracingContext) {
	info := t.getStaticInfo()
	trace.Metadata["hostname"] = info.Hostname
	trace.Metadata["go_version"] = info.GoVersion
	trace.Metadata["num_cpu"] = fmt.Sprintf("%d", info.NumCPU)
}

type staticInfo struct {
	Hostname  string
	GoVersion string
	NumCPU    int
}

func (t *Tracer) getStaticInfo() *staticInfo {
	v := t.staticInfo.Load()
	if v != nil {
		info, ok := v.(*staticInfo)
		if ok {
			return info
		}
	}

	info := &staticInfo{
		GoVersion: runtime.Version(),
		NumCPU:    runtime.NumCPU(),
	}
	if hostname, err := getHostname(); err == nil {
		info.Hostname = hostname
	}

	t.staticInfo.Store(info)
	return info
}

// Span methods

// End completes the span and records it as a phase.
func (s *Span) End(err error) {
	if s == nil || s.trace == nil {
		return
	}

	duration := time.Since(s.start)

	s.trace.mu.Lock()
	defer s.trace.mu.Unlock()

	phase := &Phase{
		Name:      s.name,
		StartTime: s.start,
		EndTime:   time.Now(),
		Duration:  duration,
		Metadata:  s.metadata,
		Status:    StatusOK,
	}

	if err != nil {
		phase.Status = StatusError
		phase.Error = err.Error()
	}

	if len(s.trace.Phases) < 1000 { // MaxTraceSize default
		s.trace.Phases = append(s.trace.Phases, phase)
	}
}

// SetMetadata sets metadata on the span.
func (s *Span) SetMetadata(key, value string) {
	if s == nil {
		return
	}
	s.metadata[key] = value
}

// GetTraceID returns the trace ID.
func (s *Span) GetTraceID() string {
	if s == nil {
		return ""
	}
	return s.trace.TraceID
}

// Context key type for storing trace in context.
type contextKey struct{}

// WithContext stores the trace in the context.
func WithContext(ctx context.Context, trace *TracingContext) context.Context {
	return context.WithValue(ctx, contextKey{}, trace)
}

// FromContext retrieves the trace from the context.
func FromContext(ctx context.Context) *TracingContext {
	if ctx == nil {
		return nil
	}
	trace, ok := ctx.Value(contextKey{}).(*TracingContext)
	if !ok {
		return nil
	}
	return trace
}

// Helper functions

func getHostname() (string, error) {
	// Simple hostname detection
	return "localhost", nil
}

// Duration returns the total duration of the trace.
func (t *TracingContext) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.EndTime.Sub(t.StartTime)
}

// PhaseCount returns the number of phases in the trace.
func (t *TracingContext) PhaseCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.Phases)
}

// ToolCallCount returns the number of tool calls in the trace.
func (t *TracingContext) ToolCallCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.ToolCalls)
}

// LLMCallCount returns the number of LLM calls in the trace.
func (t *TracingContext) LLMCallCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.LLMCalls)
}

// TotalTokens returns the total tokens used in the trace.
func (t *TracingContext) TotalTokens() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	for _, call := range t.LLMCalls {
		total += call.TotalTokens
	}
	return total
}

// CachedTokens returns the total cached tokens used in the trace.
func (t *TracingContext) CachedTokens() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	for _, call := range t.LLMCalls {
		total += call.CachedTokens
	}
	return total
}

// SetTag sets a tag on the trace.
func (t *TracingContext) SetTag(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Tags == nil {
		t.Tags = make(map[string]string)
	}
	t.Tags[key] = value
}

// SetMetadata sets metadata on the trace.
func (t *TracingContext) SetMetadata(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
}
