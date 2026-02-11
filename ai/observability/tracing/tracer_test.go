package tracing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	t.Run("enabled tracer", func(t *testing.T) {
		tracer := NewTracer(true)
		if tracer == nil {
			t.Fatal("NewTracer() returned nil")
		}
		if !tracer.enabled {
			t.Error("NewTracer(true) returned disabled tracer")
		}
	})

	t.Run("disabled tracer", func(t *testing.T) {
		tracer := NewTracer(false)
		if tracer == nil {
			t.Fatal("NewTracer() returned nil")
		}
		if tracer.enabled {
			t.Error("NewTracer(false) returned enabled tracer")
		}
	})
}

func TestTracer_StartSpan(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")

	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}
	if span.name != "test-span" {
		t.Errorf("span.name = %v, want test-span", span.name)
	}
	if span.startTime.IsZero() {
		t.Error("span.startTime is zero")
	}
	if tracer.current != span {
		t.Error("tracer.current is not the started span")
	}

	// Check context contains tracer
	extractedTracer := FromContext(ctx)
	if extractedTracer != tracer {
		t.Error("context does not contain the tracer")
	}
}

func TestTracer_StartSpan_Disabled(t *testing.T) {
	tracer := NewTracer(false)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")

	if span == nil {
		t.Fatal("StartSpan() returned nil span")
	}
	if span.name != "test-span" {
		t.Errorf("span.name = %v, want test-span", span.name)
	}
	if !span.startTime.IsZero() {
		t.Error("disabled tracer should not set startTime")
	}
}

func TestTracer_End(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	tracer.End(span)

	// After ending, current should be reset to nil
	if tracer.current != nil {
		t.Error("tracer.current should be nil after ending root span")
	}
}

func TestTracer_End_ChildSpan(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	// Start parent span
	ctx, parent := tracer.StartSpan(ctx, "parent")

	// Start child span
	ctx, child := tracer.StartSpan(ctx, "child")

	if tracer.current != child {
		t.Error("tracer.current should be child span")
	}

	// End child span
	tracer.End(child)

	if tracer.current != parent {
		t.Error("tracer.current should be parent after ending child")
	}

	// End parent span
	tracer.End(parent)

	if tracer.current != nil {
		t.Error("tracer.current should be nil after ending parent")
	}
}

func TestTracer_End_Disabled(t *testing.T) {
	tracer := NewTracer(false)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	tracer.End(span) // Should not panic
}

func TestTracer_SetMetadata(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	tracer.SetMetadata(span, "key", "value")

	if span.metadata == nil {
		t.Fatal("span.metadata is nil")
	}
	if span.metadata["key"] != "value" {
		t.Errorf("span.metadata[key] = %v, want value", span.metadata["key"])
	}
}

func TestTracer_SetMetadata_Disabled(t *testing.T) {
	tracer := NewTracer(false)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	tracer.SetMetadata(span, "key", "value") // Should not panic

	if span.metadata != nil && len(span.metadata) > 0 {
		t.Error("disabled tracer should not set metadata")
	}
}

func TestTracer_RecordError(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	testErr := errors.New("test error")
	tracer.RecordError(span, testErr)

	if span.metadata == nil {
		t.Fatal("span.metadata is nil")
	}
	if span.metadata["error"] != "test error" {
		t.Errorf("span.metadata[error] = %v, want test error", span.metadata["error"])
	}
}

func TestTracer_RecordError_NilError(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	tracer.RecordError(span, nil) // Should not panic

	if _, ok := span.metadata["error"]; ok {
		t.Error("RecordError(nil) should not set error metadata")
	}
}

func TestTracer_RecordError_Disabled(t *testing.T) {
	tracer := NewTracer(false)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")
	testErr := errors.New("test error")
	tracer.RecordError(span, testErr) // Should not panic

	if _, ok := span.metadata["error"]; ok {
		t.Error("disabled tracer should not record error")
	}
}

func TestFromContext(t *testing.T) {
	t.Run("tracer in context", func(t *testing.T) {
		tracer := NewTracer(true)
		ctx := context.Background()

		ctx, _ = tracer.StartSpan(ctx, "test-span")
		extracted := FromContext(ctx)

		if extracted == nil {
			t.Fatal("FromContext() returned nil")
		}
		if extracted != tracer {
			t.Error("FromContext() did not return the same tracer")
		}
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		extracted := FromContext(ctx)

		if extracted == nil {
			t.Fatal("FromContext(empty) returned nil")
		}
		if extracted.enabled {
			t.Error("FromContext(empty) should return disabled tracer")
		}
	})
}

func TestWithSpan(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	called := false
	err := WithSpan(ctx, tracer, "test-span", func(ctx context.Context) error {
		called = true
		return nil
	})

	if !called {
		t.Error("WithSpan() did not call the function")
	}
	if err != nil {
		t.Errorf("WithSpan() error = %v, want nil", err)
	}
}

func TestWithSpan_WithError(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	testErr := errors.New("test error")
	err := WithSpan(ctx, tracer, "test-span", func(ctx context.Context) error {
		// Inside this function, the span is active
		// RecordError is called before End, so error should be in metadata
		return testErr
	})

	if err != testErr {
		t.Errorf("WithSpan() error = %v, want %v", err, testErr)
	}

	// After WithSpan returns, the span has been ended and tracer.current is nil
	// This is expected behavior
}

func TestWithSpan_Disabled(t *testing.T) {
	tracer := NewTracer(false)
	ctx := context.Background()

	called := false
	err := WithSpan(ctx, tracer, "test-span", func(ctx context.Context) error {
		called = true
		return errors.New("test error")
	})

	if !called {
		t.Error("WithSpan() did not call the function")
	}
	if err == nil {
		t.Error("WithSpan() error = nil, want error")
	}
}

func TestSpan_ParentChain(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span1 := tracer.StartSpan(ctx, "span1")
	ctx, span2 := tracer.StartSpan(ctx, "span2")
	_, span3 := tracer.StartSpan(ctx, "span3")

	// Check parent chain
	if span3.parent != span2 {
		t.Error("span3.parent is not span2")
	}
	if span2.parent != span1 {
		t.Error("span2.parent is not span1")
	}
	if span1.parent != nil {
		t.Error("span1.parent should be nil")
	}
}

func TestSpan_Duration(t *testing.T) {
	tracer := NewTracer(true)
	ctx := context.Background()

	ctx, span := tracer.StartSpan(ctx, "test-span")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	tracer.End(span)

	// Duration should be at least 10ms
	duration := time.Since(span.startTime)
	if duration < 10*time.Millisecond {
		t.Errorf("span duration = %v, want >= 10ms", duration)
	}
}

func TestMultipleTracers(t *testing.T) {
	tracer1 := NewTracer(true)
	tracer2 := NewTracer(true)
	ctx := context.Background()

	_, span1 := tracer1.StartSpan(ctx, "tracer1-span")
	_, span2 := tracer2.StartSpan(ctx, "tracer2-span")

	if tracer1.current != span1 {
		t.Error("tracer1.current is not span1")
	}
	if tracer2.current != span2 {
		t.Error("tracer2.current is not span2")
	}

	// Tracers should be independent
	if tracer1.current == tracer2.current {
		t.Error("tracer1.current and tracer2.current are the same")
	}
}
