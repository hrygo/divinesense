// Package agent provides error recovery mechanisms for AI agents.
// This system automatically attempts to recover from certain error types
// by modifying inputs and retrying, improving user experience.
//
// Note: ErrorRecovery instances are designed to be created per-request or
// shared as immutable configurations. Use WithTimezone to create a new
// instance with different settings rather than modifying existing ones.
package agent

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/hrygo/divinesense/ai/aitime"
)

// Pre-compiled regex patterns for better performance.
var (
	// Time patterns for normalization.
	timePatternRegexes = []*regexp.Regexp{
		regexp.MustCompile(`明天\d{1,2}点`),
		regexp.MustCompile(`后天\d{1,2}点`),
		regexp.MustCompile(`今天\d{1,2}点`),
		regexp.MustCompile(`下午\d{1,2}点`),
		regexp.MustCompile(`上午\d{1,2}点`),
		regexp.MustCompile(`晚上\d{1,2}点`),
		regexp.MustCompile(`早上\d{1,2}点`),
		regexp.MustCompile(`\d{1,2}:\d{2}`),
		regexp.MustCompile(`\d{1,2}点\d{0,2}分?`),
	}

	// Space normalization pattern.
	spaceRegex = regexp.MustCompile(`\s+`)
)

// ErrorRecovery provides automatic error recovery for agent executions.
// It attempts to recover from certain error types by modifying inputs and retrying.
// This struct is safe for concurrent use as long as configuration is not modified
// after creation. Use WithTimezone to create new instances with different settings.
type ErrorRecovery struct {
	timeService aitime.TimeService
	timezone    string
}

// NewErrorRecovery creates a new ErrorRecovery instance.
func NewErrorRecovery(timeService aitime.TimeService) *ErrorRecovery {
	return &ErrorRecovery{
		timeService: timeService,
		timezone:    "Asia/Shanghai",
	}
}

// WithTimezone returns a new ErrorRecovery instance with the specified timezone.
// This method is safe for concurrent use as it creates a new instance.
func (r *ErrorRecovery) WithTimezone(tz string) *ErrorRecovery {
	return &ErrorRecovery{
		timeService: r.timeService,
		timezone:    tz,
	}
}

// ExecutorFunc is the function signature for agent executors.
type ExecutorFunc func(ctx context.Context, input string) (string, error)

// ExecuteWithRecovery executes the given function with automatic error recovery.
// If the execution fails with a recoverable error, it attempts to fix the input and retry once.
//
// Returns:
//   - On success: (result, nil)
//   - On failure: (user-friendly message, original error)
//
// Use ExecuteWithRecoveryDetailed if you need more detailed execution information.
func (r *ErrorRecovery) ExecuteWithRecovery(
	ctx context.Context,
	executor ExecutorFunc,
	input string,
) (string, error) {
	res := r.ExecuteWithRecoveryDetailed(ctx, executor, input)
	if !res.Success {
		return res.Result, res.OriginalError
	}
	return res.Result, nil
}

// tryRecover attempts to recover from the error by modifying the input.
// Returns (true, fixedInput) if recovery is possible, (false, "") otherwise.
func (r *ErrorRecovery) tryRecover(ctx context.Context, err error, input string) (bool, string) {
	switch {
	case errors.Is(err, ErrInvalidTimeFormat):
		// Time format error → attempt to normalize time expressions
		if normalized := r.normalizeTimeInInput(ctx, input); normalized != input {
			return true, normalized
		}

	case errors.Is(err, ErrToolNotFound):
		// Tool not found → retry with same input (let router re-select)
		return true, input

	case errors.Is(err, ErrParseError):
		// Parse error → try to simplify input
		if simplified := r.simplifyInput(input); simplified != input {
			return true, simplified
		}
	}

	return false, ""
}

// normalizeTimeInInput attempts to normalize time expressions in the input.
func (r *ErrorRecovery) normalizeTimeInInput(ctx context.Context, input string) string {
	if r.timeService == nil {
		return input
	}

	result := input
	for _, re := range timePatternRegexes {
		matches := re.FindAllString(input, -1)
		for _, match := range matches {
			// Try to normalize this time expression
			normalized, err := r.timeService.Normalize(ctx, match, r.timezone)
			if err == nil {
				// Replace with ISO format for clarity
				isoTime := normalized.Format("15:04")
				result = strings.Replace(result, match, isoTime, 1)
			}
		}
	}

	return result
}

// simplifyInput attempts to extract key information from complex input.
func (r *ErrorRecovery) simplifyInput(input string) string {
	// Remove excessive punctuation
	simplified := strings.TrimSpace(input)

	// Remove repeated spaces
	simplified = spaceRegex.ReplaceAllString(simplified, " ")

	// Remove common filler words that might confuse parsing
	fillers := []string{"请", "帮我", "麻烦", "能不能", "可以", "想"}
	for _, filler := range fillers {
		simplified = strings.ReplaceAll(simplified, filler, "")
	}

	simplified = strings.TrimSpace(simplified)

	// Only return if we actually simplified something
	if simplified != input && len(simplified) > 0 {
		return simplified
	}
	return input
}

// formatUserFriendlyError converts technical errors to user-friendly messages.
func (r *ErrorRecovery) formatUserFriendlyError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidTimeFormat):
		return "抱歉，我没能理解时间。请尝试更明确的表达，比如\"明天下午3点\""

	case errors.Is(err, ErrToolNotFound):
		return "抱歉，我暂时无法处理这个请求"

	case errors.Is(err, ErrNetworkError):
		return "网络连接出现问题，请稍后重试"

	case errors.Is(err, ErrServiceUnavailable):
		return "服务暂时不可用，请稍后重试"

	case errors.Is(err, ErrScheduleConflict):
		return "该时间段已有安排，请选择其他时间"

	case errors.Is(err, ErrParseError):
		return "抱歉，我没能理解您的请求。请尝试更简单的表达"

	case errors.Is(err, ErrInvalidInput):
		return "输入内容有误，请检查后重试"

	case errors.Is(err, context.DeadlineExceeded):
		return "处理时间较长，请稍后重试"

	case errors.Is(err, context.Canceled):
		return "请求已取消"

	default:
		return "抱歉，处理遇到问题，请稍后重试"
	}
}

// RecoveryResult contains the result of an execution with recovery.
type RecoveryResult struct {
	OriginalError error
	Result        string
	Success       bool
	WasRecovered  bool
}

// ExecuteWithRecoveryDetailed executes with recovery and returns detailed result.
// This is the recommended method when you need to programmatically distinguish
// between successful results and error messages.
func (r *ErrorRecovery) ExecuteWithRecoveryDetailed(
	ctx context.Context,
	executor ExecutorFunc,
	input string,
) RecoveryResult {
	// First attempt
	result, err := executor(ctx, input)
	if err == nil {
		return RecoveryResult{
			Success: true,
			Result:  result,
		}
	}

	originalErr := err

	// Attempt recovery (single retry)
	if recovered, fixedInput := r.tryRecover(ctx, err, input); recovered {
		result, err = executor(ctx, fixedInput)
		if err == nil {
			return RecoveryResult{
				Success:       true,
				Result:        result,
				WasRecovered:  true,
				OriginalError: originalErr,
			}
		}
	}

	// Return user-friendly error
	return RecoveryResult{
		Success:       false,
		Result:        r.formatUserFriendlyError(err),
		WasRecovered:  false,
		OriginalError: originalErr,
	}
}
