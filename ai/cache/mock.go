package cache

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MockCacheService is a mock implementation of CacheService for testing.
type MockCacheService struct {
	store map[string]*cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	expiresAt time.Time
	value     []byte
}

// NewMockCacheService creates a new MockCacheService.
func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		store: make(map[string]*cacheEntry),
	}
}

// Get retrieves a value from cache.
func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.store[key]
	if !ok {
		return nil, false
	}

	// Check if expired
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in cache.
func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.store[key] = &cacheEntry{
		value:     value,
		expiresAt: expiresAt,
	}

	return nil
}

// Invalidate invalidates cache entries.
func (m *MockCacheService) Invalidate(ctx context.Context, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		for key := range m.store {
			if strings.HasPrefix(key, prefix) {
				delete(m.store, key)
			}
		}
	} else {
		delete(m.store, pattern)
	}

	return nil
}

// Size returns the number of items in the cache (for testing).
func (m *MockCacheService) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.store)
}

// Clear removes all items from the cache (for testing).
func (m *MockCacheService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]*cacheEntry)
}

// Ensure MockCacheService implements CacheService.
var _ CacheService = (*MockCacheService)(nil)
