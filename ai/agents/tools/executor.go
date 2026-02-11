// Package tools provides resilient tool execution for AI agents.
// This package implements retry logic, fallback strategies, and metrics reporting.
package tools

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/observability/metrics"
)

// Tool defines the interface for executable tools.
type Tool interface {
	// Name returns the tool's identifier.
	Name() string
	// Run executes the tool with the given input.
	Run(ctx context.Context, input string) (*Result, error)
}

// Result represents the output of a tool execution.
type Result struct {
	Data    any    `json:"data,omitempty"`
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

// ResilientToolExecutor provides retry and fallback capabilities for tool execution.
type ResilientToolExecutor struct {
	metricsService metrics.MetricsService
	fallbackRules  map[string]FallbackFunc
	maxRetries     int
	retryDelay     time.Duration
	timeout        time.Duration
}

// ExecutorOption configures a ResilientToolExecutor.
type ExecutorOption func(*ResilientToolExecutor)

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) ExecutorOption {
	return func(e *ResilientToolExecutor) {
		e.maxRetries = n
	}
}

// WithRetryDelay sets the delay between retry attempts.
func WithRetryDelay(d time.Duration) ExecutorOption {
	return func(e *ResilientToolExecutor) {
		e.retryDelay = d
	}
}

// WithTimeout sets the timeout for each execution attempt.
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *ResilientToolExecutor) {
		e.timeout = d
	}
}

// WithFallbackRules sets custom fallback rules.
// The rules map is copied to avoid concurrent modification issues.
func WithFallbackRules(rules map[string]FallbackFunc) ExecutorOption {
	return func(e *ResilientToolExecutor) {
		e.fallbackRules = copyFallbackRules(rules)
	}
}

// copyFallbackRules creates a copy of the fallback rules map.
func copyFallbackRules(src map[string]FallbackFunc) map[string]FallbackFunc {
	if src == nil {
		return nil
	}
	dst := make(map[string]FallbackFunc, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// NewResilientToolExecutor creates a new ResilientToolExecutor with the given options.
func NewResilientToolExecutor(metricsService metrics.MetricsService, opts ...ExecutorOption) *ResilientToolExecutor {
	e := &ResilientToolExecutor{
		maxRetries:     2,
		retryDelay:     500 * time.Millisecond,
		timeout:        10 * time.Second,
		metricsService: metricsService,
		fallbackRules:  copyFallbackRules(DefaultFallbackRules),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute runs the tool with retry and fallback support.
// It attempts to execute the tool, retrying on transient errors.
// If all attempts fail, it executes the fallback strategy if available.
func (e *ResilientToolExecutor) Execute(ctx context.Context, tool Tool, input string) (*Result, error) {
	start := time.Now()
	var lastErr error
	toolName := tool.Name()

attemptsLoop:
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		// Check if context is already cancelled
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break attemptsLoop
		}

		// Create timeout context for this attempt
		execCtx, cancel := context.WithTimeout(ctx, e.timeout)
		result, err := tool.Run(execCtx, input)
		cancel()

		if err == nil {
			// Success - record metrics and return
			e.recordMetrics(ctx, toolName, time.Since(start), true)
			slog.Debug("tool execution succeeded",
				slog.String("tool", toolName),
				slog.Int("attempt", attempt+1),
				slog.Duration("duration", time.Since(start)))
			return result, nil
		}

		lastErr = err
		slog.Warn("tool execution failed",
			slog.String("tool", toolName),
			slog.Int("attempt", attempt+1),
			slog.String("error", err.Error()))

		// Check if error is retryable
		if !e.isRetryable(err) {
			slog.Debug("error is not retryable, stopping attempts",
				slog.String("tool", toolName),
				slog.String("error", err.Error()))
			break attemptsLoop
		}

		// Wait before next retry (except on last attempt)
		if attempt < e.maxRetries {
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				break attemptsLoop
			case <-time.After(e.retryDelay):
				// Continue to next attempt
			}
		}
	}

	// Record failure metrics
	e.recordMetrics(ctx, toolName, time.Since(start), false)

	// Execute fallback strategy if available
	if fallback, ok := e.fallbackRules[toolName]; ok {
		slog.Info("executing fallback strategy",
			slog.String("tool", toolName))
		return fallback(ctx, tool, input, lastErr)
	}

	return nil, lastErr
}

// isRetryable determines if an error should trigger a retry.
func (e *ResilientToolExecutor) isRetryable(err error) bool {
	// Context errors are retryable (timeout, etc.)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check error message for transient patterns
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		transientPatterns := []string{
			"network",
			"timeout",
			"connection",
			"unavailable",
			"temporary",
			"retry",
			"eof",
		}
		for _, pattern := range transientPatterns {
			if strings.Contains(errMsg, pattern) {
				return true
			}
		}
	}

	return false
}

// recordMetrics records tool execution metrics.
func (e *ResilientToolExecutor) recordMetrics(ctx context.Context, toolName string, duration time.Duration, success bool) {
	if e.metricsService != nil {
		e.metricsService.RecordToolCall(ctx, toolName, duration, success)
	}
}

// ExecutionResult contains detailed information about a tool execution.
type ExecutionResult struct {
	Result        *Result
	Error         error
	FallbackError error // Error from fallback execution, if any
	Attempts      int
	TotalLatency  time.Duration
	UsedFallback  bool
}

// ExecuteDetailed runs the tool and returns detailed execution information.
func (e *ResilientToolExecutor) ExecuteDetailed(ctx context.Context, tool Tool, input string) ExecutionResult {
	start := time.Now()
	var lastErr error
	toolName := tool.Name()
	attempts := 0

attemptsLoop:
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		attempts++

		// Check if context is already cancelled
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break attemptsLoop
		}

		execCtx, cancel := context.WithTimeout(ctx, e.timeout)
		result, err := tool.Run(execCtx, input)
		cancel()

		if err == nil {
			e.recordMetrics(ctx, toolName, time.Since(start), true)
			return ExecutionResult{
				Result:       result,
				Attempts:     attempts,
				TotalLatency: time.Since(start),
				UsedFallback: false,
			}
		}

		lastErr = err

		if !e.isRetryable(err) {
			break attemptsLoop
		}

		if attempt < e.maxRetries {
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				break attemptsLoop
			case <-time.After(e.retryDelay):
			}
		}
	}

	e.recordMetrics(ctx, toolName, time.Since(start), false)

	// Try fallback
	if fallback, ok := e.fallbackRules[toolName]; ok {
		result, fbErr := fallback(ctx, tool, input, lastErr)
		return ExecutionResult{
			Result:        result,
			Error:         lastErr,
			FallbackError: fbErr,
			Attempts:      attempts,
			TotalLatency:  time.Since(start),
			UsedFallback:  true,
		}
	}

	return ExecutionResult{
		Error:        lastErr,
		Attempts:     attempts,
		TotalLatency: time.Since(start),
		UsedFallback: false,
	}
}
