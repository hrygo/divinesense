package tracing

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestTracer(t *testing.T) {
	tracer := NewTracer(DefaultConfig())

	t.Run("StartTrace", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "test_operation")

		if trace == nil {
			t.Fatal("expected non-nil trace")
		}
		if trace.TraceID == "" {
			t.Error("expected non-empty trace ID")
		}
		if trace.SpanID == "" {
			t.Error("expected non-empty span ID")
		}
		if trace.OperationName != "test_operation" {
			t.Errorf("expected operation name 'test_operation', got '%s'", trace.OperationName)
		}

		// Verify context
		ctxTrace := FromContext(ctx)
		if ctxTrace != trace {
			t.Error("context should contain the same trace")
		}

		tracer.Finish(trace)
	})

	t.Run("StartSpan", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "parent")
		span := tracer.StartSpan(ctx, "child_operation")

		if span == nil {
			t.Fatal("expected non-nil span")
		}

		// Simulate work
		time.Sleep(10 * time.Millisecond)

		span.End(nil)
		tracer.Finish(trace)

		if len(trace.Phases) != 1 {
			t.Errorf("expected 1 phase, got %d", len(trace.Phases))
		}
		if trace.Phases[0].Name != "child_operation" {
			t.Errorf("expected phase name 'child_operation', got '%s'", trace.Phases[0].Name)
		}
	})

	t.Run("RecordPhase", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "phase_test")

		err := tracer.RecordPhase(trace, "test_phase", func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(trace.Phases) != 1 {
			t.Errorf("expected 1 phase, got %d", len(trace.Phases))
		}

		tracer.Finish(trace)
	})

	t.Run("RecordPhaseWithError", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "phase_error_test")

		expectedErr := errTest
		err := tracer.RecordPhase(trace, "failing_phase", func() error {
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
		if len(trace.Phases) != 1 {
			t.Errorf("expected 1 phase, got %d", len(trace.Phases))
		}
		if trace.Phases[0].Status != StatusError {
			t.Errorf("expected status StatusError, got %v", trace.Phases[0].Status)
		}

		tracer.Finish(trace)
	})

	t.Run("RecordToolCall", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "tool_test")

		input := []byte(`{"query": "test"}`)
		output := []byte(`{"result": "success"}`)

		tracer.RecordToolCall(trace, "test_tool", "retrieval", input, output, 50*time.Millisecond, nil)

		if len(trace.ToolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(trace.ToolCalls))
		}
		if trace.ToolCalls[0].Name != "test_tool" {
			t.Errorf("expected tool name 'test_tool', got '%s'", trace.ToolCalls[0].Name)
		}
		if trace.ToolCalls[0].ToolType != "retrieval" {
			t.Errorf("expected tool type 'retrieval', got '%s'", trace.ToolCalls[0].ToolType)
		}

		tracer.Finish(trace)
	})

	t.Run("RecordToolCallWithError", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "tool_error_test")

		tracer.RecordToolCall(trace, "failing_tool", "retrieval", nil, nil, 100*time.Millisecond, errTest)

		if len(trace.ToolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(trace.ToolCalls))
		}
		if trace.ToolCalls[0].Status != StatusError {
			t.Errorf("expected status StatusError, got %v", trace.ToolCalls[0].Status)
		}

		tracer.Finish(trace)
	})

	t.Run("RecordLLMCall", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "llm_test")

		tracer.RecordLLMCall(trace, "deepseek-chat", "deepseek", 100, 50, 80, 500*time.Millisecond, nil)

		if len(trace.LLMCalls) != 1 {
			t.Errorf("expected 1 LLM call, got %d", len(trace.LLMCalls))
		}
		if trace.LLMCalls[0].Model != "deepseek-chat" {
			t.Errorf("expected model 'deepseek-chat', got '%s'", trace.LLMCalls[0].Model)
		}
		if trace.LLMCalls[0].PromptTokens != 100 {
			t.Errorf("expected 100 prompt tokens, got %d", trace.LLMCalls[0].PromptTokens)
		}
		if trace.LLMCalls[0].CachedTokens != 80 {
			t.Errorf("expected 80 cached tokens, got %d", trace.LLMCalls[0].CachedTokens)
		}

		tracer.Finish(trace)
	})

	t.Run("TraceMetrics", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "metrics_test")

		tracer.RecordToolCall(trace, "tool1", "retrieval", nil, nil, 10*time.Millisecond, nil)
		tracer.RecordToolCall(trace, "tool2", "scheduler", nil, nil, 20*time.Millisecond, nil)
		tracer.RecordLLMCall(trace, "model1", "provider1", 100, 50, 0, 100*time.Millisecond, nil)
		tracer.RecordLLMCall(trace, "model2", "provider2", 200, 100, 150, 200*time.Millisecond, nil)

		tracer.Finish(trace)

		if trace.ToolCallCount() != 2 {
			t.Errorf("expected 2 tool calls, got %d", trace.ToolCallCount())
		}
		if trace.LLMCallCount() != 2 {
			t.Errorf("expected 2 LLM calls, got %d", trace.LLMCallCount())
		}
		if trace.TotalTokens() != 450 {
			t.Errorf("expected 450 total tokens, got %d", trace.TotalTokens())
		}
		if trace.CachedTokens() != 150 {
			t.Errorf("expected 150 cached tokens, got %d", trace.CachedTokens())
		}
	})

	t.Run("SetTags", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "tags_test")

		trace.SetTag("agent_type", "memo")
		trace.SetTag("user_id", "123")

		if trace.Tags["agent_type"] != "memo" {
			t.Errorf("expected tag 'agent_type' to be 'memo', got '%s'", trace.Tags["agent_type"])
		}

		tracer.Finish(trace)
	})

	t.Run("SetMetadata", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "metadata_test")

		trace.SetMetadata("custom_key", "custom_value")

		if trace.Metadata["custom_key"] != "custom_value" {
			t.Errorf("expected metadata 'custom_key' to be 'custom_value', got '%s'", trace.Metadata["custom_key"])
		}

		tracer.Finish(trace)
	})

	t.Run("FinishWithError", func(t *testing.T) {
		trace, _ := tracer.StartTrace(context.Background(), "error_test")

		tracer.FinishWithError(trace, errTest)

		if trace.Status != StatusError {
			t.Errorf("expected status StatusError, got %v", trace.Status)
		}
		if trace.Metadata["error"] == "" {
			t.Error("expected error metadata to be set")
		}
	})
}

func TestFromContext(t *testing.T) {
	tracer := NewTracer(DefaultConfig())

	t.Run("ValidContext", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "test")

		retrieved := FromContext(ctx)
		if retrieved != trace {
			t.Error("retrieved trace should match original")
		}

		tracer.Finish(trace)
	})

	t.Run("NilContext", func(t *testing.T) {
		retrieved := FromContext(context.TODO())
		if retrieved != nil {
			t.Error("retrieved trace should be nil for context.TODO()")
		}
	})

	t.Run("ContextWithoutTrace", func(t *testing.T) {
		retrieved := FromContext(context.Background())
		if retrieved != nil {
			t.Error("retrieved trace should be nil for context without trace")
		}
	})
}

func TestSpan(t *testing.T) {
	tracer := NewTracer(DefaultConfig())

	t.Run("SpanEnd", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "span_test")
		span := tracer.StartSpan(ctx, "test_span")

		time.Sleep(5 * time.Millisecond)
		span.End(nil)

		if len(trace.Phases) != 1 {
			t.Errorf("expected 1 phase, got %d", len(trace.Phases))
		}

		tracer.Finish(trace)
	})

	t.Run("SpanSetMetadata", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "span_metadata_test")
		span := tracer.StartSpan(ctx, "metadata_span")

		span.SetMetadata("key", "value")
		span.End(nil)

		if len(trace.Phases) != 1 {
			t.Errorf("expected 1 phase, got %d", len(trace.Phases))
		}

		tracer.Finish(trace)
	})

	t.Run("SpanGetTraceID", func(t *testing.T) {
		trace, ctx := tracer.StartTrace(context.Background(), "trace_id_test")
		span := tracer.StartSpan(ctx, "trace_id_span")

		if span.GetTraceID() != trace.TraceID {
			t.Error("span trace ID should match trace ID")
		}

		span.End(nil)
		tracer.Finish(trace)
	})
}

func TestLogExporter(t *testing.T) {
	exporter := NewLogExporter()

	trace := &TracingContext{
		TraceID:       "test-trace-id",
		SpanID:        "test-span-id",
		OperationName: "test_operation",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(100 * time.Millisecond),
		Status:        StatusOK,
		Phases:        make([]*Phase, 0),
		ToolCalls:     make([]*ToolCall, 0),
		LLMCalls:      make([]*LLMCall, 0),
		Metadata:      make(map[string]string),
		Tags:          make(map[string]string),
	}

	// Should not panic
	exporter.Export(trace)
}

func TestJaegerExporter(t *testing.T) {
	t.Run("ConvertToJaegerSpan", func(t *testing.T) {
		exporter := NewJaegerExporter(DefaultJaegerConfig())

		trace := &TracingContext{
			TraceID:       "test-trace-id",
			SpanID:        "test-span-id",
			OperationName: "test_operation",
			StartTime:     time.Now(),
			EndTime:       time.Now().Add(100 * time.Millisecond),
			Status:        StatusOK,
			Phases: []*Phase{
				{
					Name:      "test_phase",
					StartTime: time.Now(),
					EndTime:   time.Now().Add(50 * time.Millisecond),
					Duration:  50 * time.Millisecond,
					Status:    StatusOK,
				},
			},
			ToolCalls: []*ToolCall{
				{
					Name:      "test_tool",
					ToolType:  "retrieval",
					StartTime: time.Now(),
					EndTime:   time.Now().Add(10 * time.Millisecond),
					Duration:  10 * time.Millisecond,
					Status:    StatusOK,
				},
			},
			LLMCalls: []*LLMCall{
				{
					Model:            "deepseek-chat",
					Provider:         "deepseek",
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					CachedTokens:     80,
					StartTime:        time.Now(),
					EndTime:          time.Now().Add(200 * time.Millisecond),
					Duration:         200 * time.Millisecond,
					Status:           StatusOK,
				},
			},
			Metadata: make(map[string]string),
			Tags: map[string]string{
				"agent_type": "memo",
			},
		}

		span := exporter.convertToJaegerSpan(trace)

		if span.OperationName != "test_operation" {
			t.Errorf("expected operation name 'test_operation', got '%s'", span.OperationName)
		}
		if len(span.Logs) != 2 { // 1 phase + 1 tool call
			t.Errorf("expected 2 logs, got %d", len(span.Logs))
		}
		if len(span.Tags) < 2 { // status + agent_type
			t.Errorf("expected at least 2 tags, got %d", len(span.Tags))
		}
	})
}

func TestCompositeExporter(t *testing.T) {
	logExporter := NewLogExporter()
	composite := NewCompositeExporter(logExporter)

	trace := &TracingContext{
		TraceID:       "test-trace-id",
		OperationName: "composite_test",
		StartTime:     time.Now(),
		EndTime:       time.Now().Add(50 * time.Millisecond),
		Status:        StatusOK,
		Phases:        make([]*Phase, 0),
		ToolCalls:     make([]*ToolCall, 0),
		LLMCalls:      make([]*LLMCall, 0),
		Metadata:      make(map[string]string),
		Tags:          make(map[string]string),
	}

	// Should not panic
	composite.Export(trace)
}

func BenchmarkTracer(b *testing.B) {
	tracer := NewTracer(DefaultConfig())
	ctx := context.Background()

	b.Run("StartTrace", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			trace, _ := tracer.StartTrace(ctx, "bench_operation")
			tracer.Finish(trace)
		}
	})

	b.Run("RecordPhase", func(b *testing.B) {
		trace, _ := tracer.StartTrace(ctx, "phase_bench")

		for i := 0; i < b.N; i++ {
			tracer.RecordPhase(trace, "bench_phase", func() error { return nil })
		}

		tracer.Finish(trace)
	})

	b.Run("RecordToolCall", func(b *testing.B) {
		trace, _ := tracer.StartTrace(ctx, "tool_bench")

		for i := 0; i < b.N; i++ {
			tracer.RecordToolCall(trace, "bench_tool", "retrieval", nil, nil, 10*time.Millisecond, nil)
		}

		tracer.Finish(trace)
	})
}
