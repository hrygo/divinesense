package cache

import (
	"context"
	"sync"
	"time"
)

// ServiceConfig configures the cache service.
type ServiceConfig struct {
	Capacity        int           // Maximum number of entries (default: 1000)
	DefaultTTL      time.Duration // Default TTL for entries (default: 5 minutes)
	CleanupInterval time.Duration // Interval for expired entry cleanup (default: 1 minute)
}

// DefaultServiceConfig returns default cache service configuration.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Capacity:        1000,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: time.Minute,
	}
}

// Service implements CacheService with LRU eviction.
type Service struct {
	lru *ByteLRUCache

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cleanupInterval time.Duration
}

// NewService creates a new cache service.
func NewService(cfg ServiceConfig) *Service {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 1000
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		lru:             NewByteLRUCache(cfg.Capacity, cfg.DefaultTTL),
		ctx:             ctx,
		cancel:          cancel,
		cleanupInterval: cfg.CleanupInterval,
	}

	// Start background cleanup
	s.wg.Add(1)
	go s.cleanupLoop()

	return s
}

// Close stops the cache service.
func (s *Service) Close() {
	s.cancel()
	s.wg.Wait()
}

// Get retrieves a value from cache.
func (s *Service) Get(_ context.Context, key string) ([]byte, bool) {
	return s.lru.Get(key)
}

// Set stores a value in cache.
func (s *Service) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.lru.Set(key, value, ttl)
	return nil
}

// Invalidate invalidates cache entries matching the pattern.
func (s *Service) Invalidate(_ context.Context, pattern string) error {
	s.lru.Invalidate(pattern)
	return nil
}

// Size returns the number of entries in the cache.
func (s *Service) Size() int {
	return s.lru.Size()
}

// Clear removes all entries from the cache.
func (s *Service) Clear() {
	s.lru.Clear()
}

// cleanupLoop periodically removes expired entries.
func (s *Service) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.lru.CleanupExpired()
		}
	}
}

// Ensure Service implements CacheService.
var _ CacheService = (*Service)(nil)
