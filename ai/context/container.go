package context

import (
	"fmt"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
)

// Container manages dependencies for context building.
// This implements Dependency Injection following the Dependency Inversion Principle (DIP).
// High-level modules (context building) depend on abstractions (interfaces),
// not concrete implementations.
type Container struct {
	mu       sync.RWMutex
	services map[string]interface{}
	builders map[string]func() interface{}
}

// NewContainer creates a new dependency container.
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
		builders: make(map[string]func() interface{}),
	}
}

// Register registers a service instance.
func (c *Container) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = service
}

// RegisterBuilder registers a lazy service builder.
// The builder is called on first Get() and the result is cached.
func (c *Container) RegisterBuilder(name string, builder func() interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.builders[name] = builder
}

// Get retrieves a service by name.
// P0 fix: use double-checked locking to prevent race condition in lazy initialization.
func (c *Container) Get(name string) (interface{}, bool) {
	// First check with read lock (fast path)
	c.mu.RLock()
	if service, ok := c.services[name]; ok {
		c.mu.RUnlock()
		return service, true
	}
	c.mu.RUnlock()

	// Need to check builders - requires write lock for lazy init
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if service, ok := c.services[name]; ok {
		return service, true
	}

	// Check if builder exists (lazy initialization)
	if builder, ok := c.builders[name]; ok {
		service := builder()
		c.services[name] = service // Cache instance
		delete(c.builders, name)
		return service, true
	}

	return nil, false
}

// MustGet retrieves a service or panics.
func (c *Container) MustGet(name string) interface{} {
	service, ok := c.Get(name)
	if !ok {
		panic(fmt.Sprintf("service not found: %s", name))
	}
	return service
}

// Has checks if a service is registered.
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.services[name]
	if !ok {
		_, ok = c.builders[name]
	}
	return ok
}

// Remove removes a service from the container.
func (c *Container) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.services, name)
	delete(c.builders, name)
}

// Clear removes all services from the container.
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[string]interface{})
	c.builders = make(map[string]func() interface{})
}

// Service name constants for type-safe access.
const (
	// ServiceBlockStore is the AIBlock storage service
	ServiceBlockStore = "block_store"
	// ServiceVectorSearch is the vector search service
	ServiceVectorSearch = "vector_search"
	// ServiceEmbedding is the embedding generation service
	ServiceEmbedding = "embedding"
	// ServiceMetadataMgr is the metadata manager
	ServiceMetadataMgr = "metadata_manager"
	// ServiceContextBuilder is the main context builder
	ServiceContextBuilder = "context_builder"
	// ServiceMessageProvider is the message provider
	ServiceMessageProvider = "message_provider"
	// ServiceEpisodicProvider is the episodic memory provider
	ServiceEpisodicProvider = "episodic_provider"
)

// ContextEngineConfig holds configuration for the context engine.
type ContextEngineConfig struct {
	// MaxTurns is the maximum conversation turns to include
	MaxTurns int
	// MaxEpisodes is the maximum episodic memories to retrieve
	MaxEpisodes int
	// MaxTokens is the default token budget
	MaxTokens int
	// CacheTTL is the cache time-to-live
	CacheTTL int // seconds
	// MetadataCacheTTL is the metadata cache TTL in seconds
	MetadataCacheTTL int
	// StickyConfig is the sticky routing configuration
	StickyConfig *StickyConfig
}

// DefaultContextEngineConfig returns the default configuration.
func DefaultContextEngineConfig() *ContextEngineConfig {
	return &ContextEngineConfig{
		MaxTurns:         10,
		MaxEpisodes:      3,
		MaxTokens:        4096,
		CacheTTL:         300, // 5 minutes
		MetadataCacheTTL: 300,
		StickyConfig:     DefaultStickyConfig(),
	}
}

// InitializeContextEngine sets up the entire context building infrastructure.
// This is the main entry point for context engine initialization.
func InitializeContextEngine(
	blockStore store.AIBlockStore,
	cfg *ContextEngineConfig,
	userID int32,
) (*Service, *MetadataManager, *Container, error) {
	if cfg == nil {
		cfg = DefaultContextEngineConfig()
	}

	// Create container
	container := NewContainer()

	// Register core services
	container.Register(ServiceBlockStore, blockStore)

	// Create store adapter for context package
	storeAdapter := NewStoreAdapter(blockStore)
	container.Register("store_adapter", storeAdapter)

	// Create message provider
	msgProvider := NewBlockStoreMessageProvider(storeAdapter, userID)
	container.Register(ServiceMessageProvider, msgProvider)

	// Create metadata manager
	metadataMgr := NewMetadataManager(blockStore, durationFromSeconds(cfg.MetadataCacheTTL))
	if cfg.StickyConfig != nil {
		metadataMgr.WithStickyConfig(cfg.StickyConfig)
	}
	container.Register(ServiceMetadataMgr, metadataMgr)

	// Create context builder service
	ctxBuilderCfg := Config{
		MaxTurns:    cfg.MaxTurns,
		MaxEpisodes: cfg.MaxEpisodes,
		MaxTokens:   cfg.MaxTokens,
		CacheTTL:    durationFromSeconds(cfg.CacheTTL),
	}
	ctxBuilder := NewService(ctxBuilderCfg).
		WithMessageProvider(msgProvider)
	container.Register(ServiceContextBuilder, ctxBuilder)

	return ctxBuilder, metadataMgr, container, nil
}

// durationFromSeconds converts seconds to Duration.
func durationFromSeconds(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
