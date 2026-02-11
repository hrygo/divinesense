package logging

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	h := slog.NewTextHandler(os.Stdout, nil)
	l := NewLogger(h)

	if l == nil {
		t.Fatal("NewLogger() returned nil")
	}
	if l.handler != h {
		t.Error("NewLogger() did not set handler")
	}
	if l.level != LevelInfo {
		t.Errorf("NewLogger() level = %v, want LevelInfo", l.level)
	}
	if l.fields == nil {
		t.Error("NewLogger() fields map is nil")
	}
}

func TestNewLogger_NilHandler(t *testing.T) {
	l := NewLogger(nil)

	if l == nil {
		t.Fatal("NewLogger(nil) returned nil")
	}
	if l.handler == nil {
		t.Error("NewLogger(nil) did not create default handler")
	}
}

func TestLogger_WithLevel(t *testing.T) {
	l := NewLogger(nil)
	newLogger := l.WithLevel(LevelDebug)

	if newLogger == nil {
		t.Fatal("WithLevel() returned nil")
	}
	if newLogger.level != LevelDebug {
		t.Errorf("WithLevel() level = %v, want LevelDebug", newLogger.level)
	}
	// Original logger should be unchanged
	if l.level != LevelInfo {
		t.Errorf("original logger level = %v, want LevelInfo", l.level)
	}
}

func TestLogger_WithField(t *testing.T) {
	l := NewLogger(nil)
	l = l.WithField("key1", "value1")

	newLogger := l.WithField("key2", "value2")

	if newLogger == nil {
		t.Fatal("WithField() returned nil")
	}
	// Check that fields are copied
	if _, ok := newLogger.fields["key1"]; !ok {
		t.Error("WithField() did not copy existing fields")
	}
	if newLogger.fields["key2"] != "value2" {
		t.Errorf("WithField() key2 = %v, want value2", newLogger.fields["key2"])
	}
	// Original logger should not have the new field
	if _, ok := l.fields["key2"]; ok {
		t.Error("WithField() modified original logger")
	}
}

func TestLogger_WithFields(t *testing.T) {
	l := NewLogger(nil)
	l = l.WithField("key1", "value1")

	newFields := map[string]interface{}{
		"key2": "value2",
		"key3": 123,
	}
	newLogger := l.WithFields(newFields)

	if newLogger == nil {
		t.Fatal("WithFields() returned nil")
	}
	// Check that both old and new fields exist
	if _, ok := newLogger.fields["key1"]; !ok {
		t.Error("WithFields() did not copy existing fields")
	}
	if newLogger.fields["key2"] != "value2" {
		t.Errorf("WithFields() key2 = %v, want value2", newLogger.fields["key2"])
	}
	if newLogger.fields["key3"] != 123 {
		t.Errorf("WithFields() key3 = %v, want 123", newLogger.fields["key3"])
	}
}

func TestLogger_ThreadSafety(t *testing.T) {
	l := NewLogger(nil)
	var wg sync.WaitGroup

	// Concurrent writes to test thread safety
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.WithField("goroutine", n).Info("test message")
		}(i)
	}

	wg.Wait()
	// If we get here without panic/deadlock, test passes
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	// Create a logger with WARN level
	l := NewLogger(nil).WithLevel(LevelWarn)

	// These should not panic (they're filtered)
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	// Logger level should still be Warn
	if l.level != LevelWarn {
		t.Errorf("level = %v, want LevelWarn", l.level)
	}
}

func TestContext_Logger(t *testing.T) {
	ctx := context.Background()
	l := NewLogger(nil).WithField("context_field", "context_value")

	// Add logger to context
	ctx = ToContext(ctx, l)

	// Extract logger from context
	extracted := FromContext(ctx)

	if extracted == nil {
		t.Fatal("FromContext() returned nil")
	}

	// Check that field was preserved
	val, ok := extracted.fields["context_field"]
	if !ok {
		t.Error("FromContext() did not preserve fields")
	}
	if val != "context_value" {
		t.Errorf("FromContext() field = %v, want context_value", val)
	}
}

func TestContext_EmptyContext(t *testing.T) {
	ctx := context.Background()
	l := FromContext(ctx)

	if l == nil {
		t.Fatal("FromContext(empty) returned nil, should return default logger")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// These should not panic
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	SetLevel(LevelDebug)
}

func TestLogger_VariadicArgs(t *testing.T) {
	l := NewLogger(nil)

	// These should not panic
	l.Info("message", "key1", "value1", "key2", 123)
	l.Warn("message", "count", 42)
	l.Error("error message", "error", "test error")
}

func TestLogger_Immutability(t *testing.T) {
	l1 := NewLogger(nil)
	l2 := l1.WithField("key", "value")

	// l1 should not have the field
	if _, ok := l1.fields["key"]; ok {
		t.Error("WithField() modified original logger")
	}

	// l2 should have the field
	if _, ok := l2.fields["key"]; !ok {
		t.Error("WithField() did not add field to new logger")
	}
}
