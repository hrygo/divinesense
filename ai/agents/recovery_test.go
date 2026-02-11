package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/hrygo/divinesense/ai/aitime"
)

func TestErrorRecovery_ExecuteWithRecovery(t *testing.T) {
	ctx := context.Background()
	timeService := aitime.NewMockTimeService()
	recovery := NewErrorRecovery(timeService)

	t.Run("SuccessfulExecution_NoRecoveryNeeded", func(t *testing.T) {
		executor := func(ctx context.Context, input string) (string, error) {
			return "success result", nil
		}

		result, err := recovery.ExecuteWithRecovery(ctx, executor, "test input")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success result" {
			t.Errorf("expected 'success result', got '%s'", result)
		}
	})

	t.Run("TimeFormatError_RecoveryAttempted", func(t *testing.T) {
		callCount := 0
		executor := func(ctx context.Context, input string) (string, error) {
			callCount++
			if callCount == 1 {
				return "", ErrInvalidTimeFormat
			}
			return "recovered result", nil
		}

		result, err := recovery.ExecuteWithRecovery(ctx, executor, "明天3点开会")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("expected 2 calls (original + retry), got %d", callCount)
		}
		if result != "recovered result" {
			t.Errorf("expected 'recovered result', got '%s'", result)
		}
	})

	t.Run("ToolNotFound_RecoveryAttempted", func(t *testing.T) {
		callCount := 0
		executor := func(ctx context.Context, input string) (string, error) {
			callCount++
			if callCount == 1 {
				return "", ErrToolNotFound
			}
			return "recovered result", nil
		}

		result, err := recovery.ExecuteWithRecovery(ctx, executor, "test input")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 2 {
			t.Errorf("expected 2 calls, got %d", callCount)
		}
		if result != "recovered result" {
			t.Errorf("expected 'recovered result', got '%s'", result)
		}
	})

	t.Run("UnrecoverableError_ReturnsUserFriendlyMessage", func(t *testing.T) {
		executor := func(ctx context.Context, input string) (string, error) {
			return "", ErrNetworkError
		}

		result, err := recovery.ExecuteWithRecovery(ctx, executor, "test input")
		// Now returns original error along with user-friendly message
		if err == nil {
			t.Fatal("expected error to be returned")
		}
		if result != "网络连接出现问题，请稍后重试" {
			t.Errorf("unexpected user-friendly message: '%s'", result)
		}
	})

	t.Run("RecoveryFails_ReturnsUserFriendlyMessage", func(t *testing.T) {
		executor := func(ctx context.Context, input string) (string, error) {
			return "", ErrInvalidTimeFormat
		}

		result, err := recovery.ExecuteWithRecovery(ctx, executor, "test input")
		// Now returns original error along with user-friendly message
		if err == nil {
			t.Fatal("expected error to be returned")
		}
		// Should return user-friendly message after recovery fails
		expected := "抱歉，我没能理解时间。请尝试更明确的表达，比如\"明天下午3点\""
		if result != expected {
			t.Errorf("expected '%s', got '%s'", expected, result)
		}
	})
}

func TestErrorRecovery_ExecuteWithRecoveryDetailed(t *testing.T) {
	ctx := context.Background()
	timeService := aitime.NewMockTimeService()
	recovery := NewErrorRecovery(timeService)

	t.Run("SuccessfulExecution_DetailedResult", func(t *testing.T) {
		executor := func(ctx context.Context, input string) (string, error) {
			return "success", nil
		}

		result := recovery.ExecuteWithRecoveryDetailed(ctx, executor, "test")
		if !result.Success {
			t.Error("expected success")
		}
		if result.WasRecovered {
			t.Error("expected no recovery needed")
		}
		if result.OriginalError != nil {
			t.Error("expected no original error")
		}
	})

	t.Run("RecoveredExecution_DetailedResult", func(t *testing.T) {
		callCount := 0
		executor := func(ctx context.Context, input string) (string, error) {
			callCount++
			if callCount == 1 {
				return "", ErrToolNotFound
			}
			return "recovered", nil
		}

		result := recovery.ExecuteWithRecoveryDetailed(ctx, executor, "test")
		if !result.Success {
			t.Error("expected success after recovery")
		}
		if !result.WasRecovered {
			t.Error("expected WasRecovered to be true")
		}
		if !errors.Is(result.OriginalError, ErrToolNotFound) {
			t.Error("expected original error to be ErrToolNotFound")
		}
	})

	t.Run("FailedExecution_DetailedResult", func(t *testing.T) {
		executor := func(ctx context.Context, input string) (string, error) {
			return "", ErrServiceUnavailable
		}

		result := recovery.ExecuteWithRecoveryDetailed(ctx, executor, "test")
		if result.Success {
			t.Error("expected failure")
		}
		if result.WasRecovered {
			t.Error("expected no recovery for unrecoverable error")
		}
		if !errors.Is(result.OriginalError, ErrServiceUnavailable) {
			t.Error("expected original error to be ErrServiceUnavailable")
		}
	})
}

func TestErrorRecovery_FormatUserFriendlyError(t *testing.T) {
	recovery := NewErrorRecovery(nil)

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "InvalidTimeFormat",
			err:      ErrInvalidTimeFormat,
			expected: "抱歉，我没能理解时间。请尝试更明确的表达，比如\"明天下午3点\"",
		},
		{
			name:     "ToolNotFound",
			err:      ErrToolNotFound,
			expected: "抱歉，我暂时无法处理这个请求",
		},
		{
			name:     "NetworkError",
			err:      ErrNetworkError,
			expected: "网络连接出现问题，请稍后重试",
		},
		{
			name:     "ServiceUnavailable",
			err:      ErrServiceUnavailable,
			expected: "服务暂时不可用，请稍后重试",
		},
		{
			name:     "ScheduleConflict",
			err:      ErrScheduleConflict,
			expected: "该时间段已有安排，请选择其他时间",
		},
		{
			name:     "ParseError",
			err:      ErrParseError,
			expected: "抱歉，我没能理解您的请求。请尝试更简单的表达",
		},
		{
			name:     "InvalidInput",
			err:      ErrInvalidInput,
			expected: "输入内容有误，请检查后重试",
		},
		{
			name:     "DeadlineExceeded",
			err:      context.DeadlineExceeded,
			expected: "处理时间较长，请稍后重试",
		},
		{
			name:     "Canceled",
			err:      context.Canceled,
			expected: "请求已取消",
		},
		{
			name:     "UnknownError",
			err:      errors.New("some unknown error"),
			expected: "抱歉，处理遇到问题，请稍后重试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recovery.formatUserFriendlyError(tt.err)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestErrorRecovery_SimplifyInput(t *testing.T) {
	recovery := NewErrorRecovery(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "RemoveFillerWords",
			input:    "请帮我安排一个会议",
			expected: "安排一个会议",
		},
		{
			name:     "RemoveMultipleFillers",
			input:    "麻烦帮我看一下明天的安排",
			expected: "看一下明天的安排",
		},
		{
			name:     "CollapseSpaces",
			input:    "明天   下午   开会",
			expected: "明天 下午 开会",
		},
		{
			name:     "TrimSpaces",
			input:    "  开会  ",
			expected: "开会",
		},
		{
			name:     "NoChange",
			input:    "明天开会",
			expected: "明天开会",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := recovery.simplifyInput(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{ErrInvalidTimeFormat, "InvalidTimeFormat", true},
		{ErrToolNotFound, "ToolNotFound", true},
		{ErrParseError, "ParseError", true},
		{ErrNetworkError, "NetworkError", false},
		{ErrServiceUnavailable, "ServiceUnavailable", false},
		{ErrScheduleConflict, "ScheduleConflict", false},
		{errors.New("unknown"), "UnknownError", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoverableError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		expected bool
	}{
		{ErrNetworkError, "NetworkError", true},
		{ErrServiceUnavailable, "ServiceUnavailable", true},
		{ErrInvalidTimeFormat, "InvalidTimeFormat", false},
		{ErrToolNotFound, "ToolNotFound", false},
		{errors.New("unknown"), "UnknownError", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
