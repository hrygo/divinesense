package context

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
)

// MetadataManager manages block metadata for session state.
// It provides type-safe access to metadata fields and caches recent values
// for performance optimization.
type MetadataManager struct {
	blockStore store.AIBlockStore
	cache      *sync.Map // conversationID(int32) -> *SessionMetadata
	cacheTTL   time.Duration
	config     *StickyConfig
}

// SessionMetadata caches metadata for a session.
type SessionMetadata struct {
	LastAgent            string
	LastIntent           string
	LastIntentConfidence float32
	StickyUntil          time.Time
	LastUpdated          time.Time
	mu                   sync.RWMutex
}

// StickyConfig configures the sticky routing behavior.
// This implements the decay strategy from context-engineering.md.
type StickyConfig struct {
	// InitialWindow is the base sticky window duration
	InitialWindow time.Duration
	// MaxExtensions is the maximum number of times sticky can be extended
	MaxExtensions int
	// DecayFactor is applied to the window for each extension
	DecayFactor float64
	// MinConfidence is the minimum confidence required for sticky routing
	MinConfidence float64
}

// DefaultStickyConfig returns the default sticky configuration.
func DefaultStickyConfig() *StickyConfig {
	return &StickyConfig{
		InitialWindow: 5 * time.Minute,
		MaxExtensions: 2,
		DecayFactor:   0.5,
		MinConfidence: 0.7,
	}
}

// NewMetadataManager creates a new metadata manager.
func NewMetadataManager(store store.AIBlockStore, cacheTTL time.Duration) *MetadataManager {
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}
	return &MetadataManager{
		blockStore: store,
		cache:      &sync.Map{},
		cacheTTL:   cacheTTL,
		config:     DefaultStickyConfig(),
	}
}

// WithStickyConfig sets a custom sticky configuration.
func (m *MetadataManager) WithStickyConfig(cfg *StickyConfig) *MetadataManager {
	if cfg != nil {
		m.config = cfg
	}
	return m
}

// GetLastAgent retrieves the last agent for a conversation.
// Uses cache to avoid database queries for recent sessions.
func (m *MetadataManager) GetLastAgent(
	ctx context.Context,
	conversationID int32,
) (string, error) {
	// Check cache first
	if cached, ok := m.cache.Load(conversationID); ok {
		// P0 fix: use comma-ok for type assertion
		meta, ok := cached.(*SessionMetadata)
		if !ok {
			// Cache corruption, clear and continue
			m.cache.Delete(conversationID)
		} else {
			meta.mu.RLock()
			defer meta.mu.RUnlock()
			if time.Since(meta.LastUpdated) < m.cacheTTL {
				return meta.LastAgent, nil
			}
		}
	}

	// Cache miss or expired, query from store
	latestBlock, err := m.blockStore.GetLatestAIBlock(ctx, conversationID)
	if err != nil {
		return "", err
	}

	if latestBlock == nil {
		return "", nil // No previous agent
	}

	lastAgent, _ := latestBlock.GetMetadataLastAgent()

	// Update cache with full metadata
	m.updateCache(conversationID, latestBlock)

	return lastAgent, nil
}

// GetSessionMetadata retrieves the full session metadata for sticky routing.
func (m *MetadataManager) GetSessionMetadata(
	ctx context.Context,
	conversationID int32,
) (*SessionMetadata, error) {
	// Check cache first
	if cached, ok := m.cache.Load(conversationID); ok {
		// P0 fix: use comma-ok for type assertion
		meta, ok := cached.(*SessionMetadata)
		if !ok {
			// Cache corruption, clear and continue
			m.cache.Delete(conversationID)
		} else {
			meta.mu.RLock()
			defer meta.mu.RUnlock()
			if time.Since(meta.LastUpdated) < m.cacheTTL {
				return meta, nil
			}
		}
	}

	// Query from store
	latestBlock, err := m.blockStore.GetLatestAIBlock(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if latestBlock == nil {
		return nil, nil
	}

	// Update cache and return
	m.updateCache(conversationID, latestBlock)

	// P0 fix: use comma-ok for type assertion
	cached, ok := m.cache.Load(conversationID)
	if !ok {
		return nil, nil
	}
	meta, ok := cached.(*SessionMetadata)
	if !ok {
		return nil, nil
	}
	return meta, nil
}

// IsStickyValid checks if the sticky routing is still valid for a conversation.
func (m *MetadataManager) IsStickyValid(
	ctx context.Context,
	conversationID int32,
) (bool, *SessionMetadata) {
	meta, err := m.GetSessionMetadata(ctx, conversationID)
	if err != nil || meta == nil {
		return false, nil
	}

	meta.mu.RLock()
	defer meta.mu.RUnlock()

	// Check if within sticky window
	return time.Now().Before(meta.StickyUntil), meta
}

// CalculateStickyWindow calculates the sticky window duration based on confidence.
// Higher confidence = longer sticky window.
func (m *MetadataManager) CalculateStickyWindow(confidence float64) time.Duration {
	if confidence < m.config.MinConfidence {
		return 0 // No sticky for low confidence
	}

	// P1 fix: prevent division by zero when MinConfidence >= 1.0
	denominator := 1 - m.config.MinConfidence
	if denominator <= 0 {
		// Edge case: MinConfidence >= 1.0, just use base window
		return m.config.InitialWindow
	}

	// Linear scaling: confidence 0.7 -> 1x window, confidence 1.0 -> 1.3x window
	factor := 1 + (confidence-m.config.MinConfidence)/denominator*0.3
	return time.Duration(float64(m.config.InitialWindow) * factor)
}

// SetCurrentAgent stores the current agent for a conversation.
// This should be called after a successful routing decision.
func (m *MetadataManager) SetCurrentAgent(
	ctx context.Context,
	conversationID int32,
	blockID int64,
	agent string,
	intent string,
	confidence float32,
) error {
	// Calculate sticky window
	stickyWindow := m.CalculateStickyWindow(float64(confidence))
	stickyUntil := time.Now().Add(stickyWindow)

	// Update block metadata
	update := &store.UpdateAIBlock{
		ID: blockID,
	}
	update.SetMetadataLastAgent(agent)
	update.SetMetadataIntent(intent)
	update.SetMetadataIntentConfidence(confidence)
	update.SetMetadataStickyUntil(stickyUntil.Unix())

	_, err := m.blockStore.UpdateAIBlock(ctx, update)
	if err != nil {
		return err
	}

	// Update cache atomically
	if cached, ok := m.cache.Load(conversationID); ok {
		// P0 fix: use comma-ok for type assertion
		meta, ok := cached.(*SessionMetadata)
		if ok {
			meta.mu.Lock()
			meta.LastAgent = agent
			meta.LastIntent = intent
			meta.LastIntentConfidence = confidence
			meta.StickyUntil = stickyUntil
			meta.LastUpdated = time.Now()
			meta.mu.Unlock()
		} else {
			// Cache corruption, store new value
			m.cache.Store(conversationID, &SessionMetadata{
				LastAgent:            agent,
				LastIntent:           intent,
				LastIntentConfidence: confidence,
				StickyUntil:          stickyUntil,
				LastUpdated:          time.Now(),
			})
		}
	} else {
		m.cache.Store(conversationID, &SessionMetadata{
			LastAgent:            agent,
			LastIntent:           intent,
			LastIntentConfidence: confidence,
			StickyUntil:          stickyUntil,
			LastUpdated:          time.Now(),
		})
	}

	slog.Debug("MetadataManager.SetCurrentAgent",
		"conversation_id", conversationID,
		"block_id", blockID,
		"agent", agent,
		"intent", intent,
		"confidence", confidence,
		"sticky_until", stickyUntil.Format(time.RFC3339))

	return nil
}

// UpdateCacheOnly updates the in-memory cache without persisting to database.
// This should be called immediately after routing to enable sticky routing
// for the next request without waiting for block completion.
// Phase 2 fix: enables sticky routing across consecutive requests.
func (m *MetadataManager) UpdateCacheOnly(
	conversationID int32,
	agent string,
	intent string,
	confidence float32,
) {
	// Calculate sticky window
	stickyWindow := m.CalculateStickyWindow(float64(confidence))
	stickyUntil := time.Now().Add(stickyWindow)

	// Update cache atomically
	m.cache.Store(conversationID, &SessionMetadata{
		LastAgent:            agent,
		LastIntent:           intent,
		LastIntentConfidence: confidence,
		StickyUntil:          stickyUntil,
		LastUpdated:          time.Now(),
	})

	slog.Debug("MetadataManager.UpdateCacheOnly",
		"conversation_id", conversationID,
		"agent", agent,
		"intent", intent,
		"confidence", confidence,
		"sticky_until", stickyUntil.Format(time.RFC3339))
}

// Invalidate clears the cache for a conversation.
func (m *MetadataManager) Invalidate(conversationID int32) {
	m.cache.Delete(conversationID)
}

// updateCache updates the cache from a block.
func (m *MetadataManager) updateCache(conversationID int32, block *store.AIBlock) {
	lastAgent, _ := block.GetMetadataLastAgent()
	intent, _ := block.GetMetadataIntent()
	confidence, _ := block.GetMetadataIntentConfidence()
	stickyUntilTs, _ := block.GetMetadataStickyUntil()

	m.cache.Store(conversationID, &SessionMetadata{
		LastAgent:            lastAgent,
		LastIntent:           intent,
		LastIntentConfidence: confidence,
		StickyUntil:          time.Unix(stickyUntilTs, 0),
		LastUpdated:          time.Now(),
	})
}
