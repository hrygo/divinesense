// Package cache provides the cache service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule) and Team C (Memo Enhancement).
package cache

import (
	"context"
	"time"
)

// Consumers: Team B (Assistant+Schedule), Team C (Memo Enhancement).
type CacheService interface {
	// Get retrieves a value from cache.
	// Returns: value, whether it exists
	Get(ctx context.Context, key string) ([]byte, bool)

	// Set stores a value in cache.
	// ttl: expiration time
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Invalidate invalidates cache entries.
	// pattern: supports wildcards (user:123:*)
	Invalidate(ctx context.Context, pattern string) error
}
