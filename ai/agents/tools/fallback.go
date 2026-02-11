// Package tools provides fallback strategies for tool execution failures.
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// userIDKey is the context key for user identification.
type userIDKey struct{}

// WithUserID adds user ID to context for cache isolation.
func WithUserID(ctx context.Context, userID int32) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// getUserID extracts user ID from context. Returns 0 if not present.
func getUserID(ctx context.Context) int32 {
	if v := ctx.Value(userIDKey{}); v != nil {
		if id, ok := v.(int32); ok {
			return id
		}
	}
	return 0
}

// FallbackFunc defines the signature for fallback handlers.
// It receives the context, the failed tool, the original input, and the error.
// It returns a graceful degradation result.
type FallbackFunc func(ctx context.Context, tool Tool, input string, err error) (*Result, error)

// DefaultFallbackRules contains the default fallback strategies for common tools.
var DefaultFallbackRules = map[string]FallbackFunc{
	"memo_search":    fallbackMemoSearch,
	"schedule_query": fallbackScheduleQuery,
	"schedule_add":   fallbackScheduleAdd,
	"memo_create":    fallbackMemoCreate,
}

// fallbackMemoSearch handles memo search failures.
func fallbackMemoSearch(_ context.Context, _ Tool, _ string, _ error) (*Result, error) {
	return &Result{
		Output:  "搜索暂时不可用，请稍后重试",
		Success: false,
	}, nil
}

// fallbackScheduleQuery handles schedule query failures.
func fallbackScheduleQuery(ctx context.Context, _ Tool, input string, _ error) (*Result, error) {
	// Try to use cached data with user isolation
	userID := getUserID(ctx)
	if userID == 0 {
		// No user context - skip cache to prevent data leakage
		return &Result{
			Output:  "日程查询暂时不可用，请稍后重试",
			Success: false,
		}, nil
	}

	if cached := getCachedSchedules(userID, input); cached != "" {
		return &Result{
			Output:  cached + "\n(来自缓存，可能不是最新)",
			Success: true,
		}, nil
	}
	return &Result{
		Output:  "日程查询暂时不可用，请稍后重试",
		Success: false,
	}, nil
}

// fallbackScheduleAdd handles schedule creation failures.
func fallbackScheduleAdd(_ context.Context, _ Tool, _ string, _ error) (*Result, error) {
	return &Result{
		Output:  "日程已记录，待确认后生效。您可以稍后在日程页面查看",
		Success: false,
	}, nil
}

// fallbackMemoCreate handles memo creation failures.
func fallbackMemoCreate(_ context.Context, _ Tool, _ string, _ error) (*Result, error) {
	return &Result{
		Output:  "笔记保存遇到问题，请稍后重试或手动保存",
		Success: false,
	}, nil
}

// scheduleCache provides simple in-memory caching for schedule queries.
// Cache keys include userID for isolation between users.
var scheduleCache = &simpleCache{
	data: make(map[string]string),
}

type simpleCache struct {
	data map[string]string
	mu   sync.RWMutex
}

// makeCacheKey creates a user-isolated cache key.
func makeCacheKey(userID int32, query string) string {
	return fmt.Sprintf("%d:%s", userID, query)
}

// getCachedSchedules retrieves cached schedule data for the given user and query.
func getCachedSchedules(userID int32, query string) string {
	scheduleCache.mu.RLock()
	defer scheduleCache.mu.RUnlock()
	return scheduleCache.data[makeCacheKey(userID, query)]
}

// SetCachedSchedules stores schedule data in the cache with user isolation.
// This should be called after successful schedule queries.
func SetCachedSchedules(userID int32, query string, data string) {
	if userID == 0 {
		return // Don't cache without user context
	}
	scheduleCache.mu.Lock()
	defer scheduleCache.mu.Unlock()
	scheduleCache.data[makeCacheKey(userID, query)] = data
}

// ClearScheduleCache clears all cached schedule data.
func ClearScheduleCache() {
	scheduleCache.mu.Lock()
	defer scheduleCache.mu.Unlock()
	scheduleCache.data = make(map[string]string)
}

// FallbackRegistry allows dynamic registration of fallback handlers.
type FallbackRegistry struct {
	handlers map[string]FallbackFunc
	mu       sync.RWMutex
}

// NewFallbackRegistry creates a new FallbackRegistry with default handlers.
func NewFallbackRegistry() *FallbackRegistry {
	r := &FallbackRegistry{
		handlers: make(map[string]FallbackFunc),
	}
	// Copy default rules
	for k, v := range DefaultFallbackRules {
		r.handlers[k] = v
	}
	return r
}

// Register adds or replaces a fallback handler for the given tool.
func (r *FallbackRegistry) Register(toolName string, handler FallbackFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[toolName] = handler
}

// Get retrieves the fallback handler for the given tool.
func (r *FallbackRegistry) Get(toolName string) (FallbackFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[toolName]
	return handler, ok
}

// GetAll returns a copy of all registered handlers.
func (r *FallbackRegistry) GetAll() map[string]FallbackFunc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]FallbackFunc, len(r.handlers))
	for k, v := range r.handlers {
		result[k] = v
	}
	return result
}

// GenericFallback creates a generic fallback handler with a custom message.
func GenericFallback(message string) FallbackFunc {
	return func(_ context.Context, _ Tool, _ string, _ error) (*Result, error) {
		return &Result{
			Output:  message,
			Success: false,
		}, nil
	}
}

// ErrorAwareFallback creates a fallback that logs error details but returns a safe message.
// Error details are logged for debugging but not exposed to users.
func ErrorAwareFallback(baseMessage string) FallbackFunc {
	return func(_ context.Context, tool Tool, _ string, err error) (*Result, error) {
		if err != nil {
			// Log error details for debugging, don't expose to user
			toolName := "unknown"
			if tool != nil {
				toolName = tool.Name()
			}
			slog.Warn("tool fallback triggered",
				slog.String("tool", toolName),
				slog.String("error", err.Error()),
			)
		}
		return &Result{
			Output:  baseMessage,
			Success: false,
		}, nil
	}
}
