package tools

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/metrics"
)

// Test errors for retry logic.
var (
	errNetwork   = errors.New("network error")
	errPermanent = errors.New("permanent error")
)

// mockTool implements Tool interface for testing.
type mockTool struct {
	runFunc   func(ctx context.Context, input string) (*Result, error)
	name      string
	callCount int32
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Run(ctx context.Context, input string) (*Result, error) {
	atomic.AddInt32(&m.callCount, 1)
	return m.runFunc(ctx, input)
}

func (m *mockTool) CallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func TestResilientToolExecutor_Execute_Success(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()
	executor := NewResilientToolExecutor(metricsService)

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (*Result, error) {
			return &Result{Output: "success", Success: true}, nil
		},
	}

	result, err := executor.Execute(ctx, tool, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "success" {
		t.Errorf("expected output 'success', got '%s'", result.Output)
	}
	if tool.CallCount() != 1 {
		t.Errorf("expected 1 call, got %d", tool.CallCount())
	}
}

func TestResilientToolExecutor_Execute_RetryOnTransientError(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()
	executor := NewResilientToolExecutor(metricsService,
		WithRetryDelay(10*time.Millisecond), // Short delay for testing
	)

	callCount := 0
	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (*Result, error) {
			callCount++
			if callCount < 3 {
				return nil, errNetwork
			}
			return &Result{Output: "success after retry", Success: true}, nil
		},
	}

	result, err := executor.Execute(ctx, tool, "test input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "success after retry" {
		t.Errorf("expected 'success after retry', got '%s'", result.Output)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 original + 2 retries), got %d", callCount)
	}
}

func TestResilientToolExecutor_Execute_NoRetryOnPermanentError(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()
	executor := NewResilientToolExecutor(metricsService)

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (*Result, error) {
			return nil, errPermanent
		},
	}

	_, err := executor.Execute(ctx, tool, "test input")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if tool.CallCount() != 1 {
		t.Errorf("expected 1 call (no retry for permanent error), got %d", tool.CallCount())
	}
}

func TestResilientToolExecutor_Execute_FallbackOnFailure(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()

	customFallback := map[string]FallbackFunc{
		"test_tool": func(ctx context.Context, tool Tool, input string, err error) (*Result, error) {
			return &Result{Output: "fallback result", Success: false}, nil
		},
	}

	executor := NewResilientToolExecutor(metricsService,
		WithFallbackRules(customFallback),
		WithMaxRetries(0), // No retries
	)

	tool := &mockTool{
		name: "test_tool",
		runFunc: func(ctx context.Context, input string) (*Result, error) {
			return nil, errors.New("permanent failure")
		},
	}

	result, err := executor.Execute(ctx, tool, "test input")
	if err != nil {
		t.Fatalf("unexpected error (should use fallback): %v", err)
	}
	if result.Output != "fallback result" {
		t.Errorf("expected 'fallback result', got '%s'", result.Output)
	}
}

func TestResilientToolExecutor_Execute_Timeout(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()
	executor := NewResilientToolExecutor(metricsService,
		WithTimeout(50*time.Millisecond),
		WithRetryDelay(10*time.Millisecond),
		WithMaxRetries(1),
	)

	tool := &mockTool{
		name: "slow_tool",
		runFunc: func(ctx context.Context, input string) (*Result, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return &Result{Output: "completed", Success: true}, nil
			}
		},
	}

	_, err := executor.Execute(ctx, tool, "test input")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	// Should have attempted original + 1 retry (timeout is retryable)
	if tool.CallCount() < 2 {
		t.Errorf("expected at least 2 calls due to retry on timeout, got %d", tool.CallCount())
	}
}

func TestResilientToolExecutor_ExecuteDetailed(t *testing.T) {
	ctx := context.Background()
	metricsService := metrics.NewMockMetricsService()
	executor := NewResilientToolExecutor(metricsService,
		WithRetryDelay(10*time.Millisecond),
	)

	t.Run("Success", func(t *testing.T) {
		tool := &mockTool{
			name: "test_tool",
			runFunc: func(ctx context.Context, input string) (*Result, error) {
				return &Result{Output: "success", Success: true}, nil
			},
		}

		result := executor.ExecuteDetailed(ctx, tool, "test")
		if result.Error != nil {
			t.Errorf("unexpected error: %v", result.Error)
		}
		if result.Attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", result.Attempts)
		}
		if result.UsedFallback {
			t.Error("should not use fallback on success")
		}
	})

	t.Run("RetryThenSuccess", func(t *testing.T) {
		callCount := 0
		tool := &mockTool{
			name: "test_tool",
			runFunc: func(ctx context.Context, input string) (*Result, error) {
				callCount++
				if callCount == 1 {
					return nil, errors.New("service unavailable")
				}
				return &Result{Output: "recovered", Success: true}, nil
			},
		}

		result := executor.ExecuteDetailed(ctx, tool, "test")
		if result.Error != nil {
			t.Errorf("unexpected error: %v", result.Error)
		}
		if result.Attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", result.Attempts)
		}
	})

	t.Run("FallbackUsed", func(t *testing.T) {
		customFallback := map[string]FallbackFunc{
			"fallback_tool": GenericFallback("fallback used"),
		}
		exec := NewResilientToolExecutor(metricsService,
			WithFallbackRules(customFallback),
			WithMaxRetries(0),
		)

		tool := &mockTool{
			name: "fallback_tool",
			runFunc: func(ctx context.Context, input string) (*Result, error) {
				return nil, errors.New("always fails")
			},
		}

		result := exec.ExecuteDetailed(ctx, tool, "test")
		if !result.UsedFallback {
			t.Error("expected fallback to be used")
		}
		if result.Result == nil || result.Result.Output != "fallback used" {
			t.Error("expected fallback result")
		}
	})
}

func TestFallbackRegistry(t *testing.T) {
	registry := NewFallbackRegistry()

	t.Run("DefaultRulesLoaded", func(t *testing.T) {
		handler, ok := registry.Get("memo_search")
		if !ok {
			t.Error("expected memo_search handler to exist")
		}
		if handler == nil {
			t.Error("handler should not be nil")
		}
	})

	t.Run("RegisterCustomHandler", func(t *testing.T) {
		customHandler := GenericFallback("custom message")
		registry.Register("custom_tool", customHandler)

		handler, ok := registry.Get("custom_tool")
		if !ok {
			t.Error("expected custom_tool handler to exist")
		}

		result, _ := handler(context.Background(), nil, "", nil)
		if result.Output != "custom message" {
			t.Errorf("expected 'custom message', got '%s'", result.Output)
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		all := registry.GetAll()
		if len(all) < 4 { // Default rules + custom
			t.Errorf("expected at least 4 handlers, got %d", len(all))
		}
	})
}

func TestDefaultFallbacks(t *testing.T) {
	ctx := context.Background()
	// Add userID to context for cache isolation tests
	ctxWithUser := WithUserID(ctx, 1)

	t.Run("MemoSearch", func(t *testing.T) {
		result, err := fallbackMemoSearch(ctx, nil, "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Success {
			t.Error("fallback should not be marked as success")
		}
		if result.Output == "" {
			t.Error("fallback should have a message")
		}
	})

	t.Run("ScheduleQuery_NoCache", func(t *testing.T) {
		ClearScheduleCache()
		result, err := fallbackScheduleQuery(ctxWithUser, nil, "test query", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Success {
			t.Error("fallback without cache should not be success")
		}
	})

	t.Run("ScheduleQuery_WithCache", func(t *testing.T) {
		SetCachedSchedules(1, "cached query", "今天有3个日程")
		result, err := fallbackScheduleQuery(ctxWithUser, nil, "cached query", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("fallback with cache should be success")
		}
		if result.Output == "" {
			t.Error("should have cached data in output")
		}
		ClearScheduleCache()
	})

	t.Run("ScheduleAdd", func(t *testing.T) {
		result, err := fallbackScheduleAdd(ctx, nil, "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output == "" {
			t.Error("fallback should have a message")
		}
	})
}

func TestGenericFallback(t *testing.T) {
	handler := GenericFallback("test message")
	result, err := handler(context.Background(), nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output != "test message" {
		t.Errorf("expected 'test message', got '%s'", result.Output)
	}
	if result.Success {
		t.Error("generic fallback should not be marked as success")
	}
}

func TestErrorAwareFallback(t *testing.T) {
	handler := ErrorAwareFallback("operation failed")

	t.Run("WithError", func(t *testing.T) {
		// ErrorAwareFallback should NOT include error details in user message (security fix)
		// Error is logged internally but not exposed to users
		result, _ := handler(context.Background(), nil, "", errors.New("network timeout"))
		if result.Output != "operation failed" {
			t.Errorf("should return base message without error details, got '%s'", result.Output)
		}
	})

	t.Run("WithoutError", func(t *testing.T) {
		result, _ := handler(context.Background(), nil, "", nil)
		if result.Output != "operation failed" {
			t.Errorf("expected 'operation failed', got '%s'", result.Output)
		}
	})
}

func TestExecutorOptions(t *testing.T) {
	metricsService := metrics.NewMockMetricsService()

	t.Run("WithMaxRetries", func(t *testing.T) {
		executor := NewResilientToolExecutor(metricsService, WithMaxRetries(5))
		if executor.maxRetries != 5 {
			t.Errorf("expected maxRetries 5, got %d", executor.maxRetries)
		}
	})

	t.Run("WithRetryDelay", func(t *testing.T) {
		executor := NewResilientToolExecutor(metricsService, WithRetryDelay(time.Second))
		if executor.retryDelay != time.Second {
			t.Errorf("expected retryDelay 1s, got %v", executor.retryDelay)
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		executor := NewResilientToolExecutor(metricsService, WithTimeout(30*time.Second))
		if executor.timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", executor.timeout)
		}
	})
}
