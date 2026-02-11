// Package tracing provides distributed tracing instrumentation for AI modules.
package tracing

import (
	"context"
	"time"

	"log/slog"
)

// Span represents a single operation in a trace.
type Span struct {
	name      string
	startTime time.Time
	parent    *Span
	metadata  map[string]interface{}
}

// Tracer manages trace spans and their lifecycle.
type Tracer struct {
	current *Span
	enabled bool
}

// NewTracer creates a new tracer instance.
func NewTracer(enabled bool) *Tracer {
	return &Tracer{
		enabled: enabled,
	}
}

// StartSpan begins a new trace span.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if !t.enabled {
		return ctx, &Span{name: name}
	}

	span := &Span{
		name:      name,
		startTime: time.Now(),
		metadata:  make(map[string]interface{}),
	}

	if t.current != nil {
		span.parent = t.current
	}

	t.current = span
	return context.WithValue(ctx, tracerKey{}, t), span
}

// End completes the current span.
func (t *Tracer) End(span *Span) {
	if !t.enabled || span == nil {
		return
	}

	duration := time.Since(span.startTime)
	slog.Debug("span completed",
		"name", span.name,
		"duration_ms", duration.Milliseconds(),
		"metadata", span.metadata,
	)

	// Restore parent span if exists
	if span.parent != nil {
		t.current = span.parent
	} else {
		t.current = nil
	}
}

// SetMetadata adds metadata to the current span.
func (t *Tracer) SetMetadata(span *Span, key string, value interface{}) {
	if !t.enabled || span == nil {
		return
	}
	span.metadata[key] = value
}

// RecordError records an error in the current span.
func (t *Tracer) RecordError(span *Span, err error) {
	if !t.enabled || span == nil || err == nil {
		return
	}
	span.metadata["error"] = err.Error()
}

type tracerKey struct{}

// FromContext extracts the tracer from context.
func FromContext(ctx context.Context) *Tracer {
	if t, ok := ctx.Value(tracerKey{}).(*Tracer); ok {
		return t
	}
	return &Tracer{enabled: false}
}

// WithSpan wraps a function with tracing.
func WithSpan(ctx context.Context, tracer *Tracer, name string, fn func(context.Context) error) error {
	if !tracer.enabled {
		return fn(ctx)
	}

	ctx, span := tracer.StartSpan(ctx, name)
	defer tracer.End(span)

	err := fn(ctx)
	if err != nil {
		tracer.RecordError(span, err)
	}

	return err
}
