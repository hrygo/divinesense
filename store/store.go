package store

import (
	"context"
	"time"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/store/cache"
)

// Store provides database access to all raw objects.
type Store struct {
	profile *profile.Profile
	driver  Driver

	// Cache settings
	cacheConfig cache.Config

	// Caches
	instanceSettingCache *cache.Cache // cache for instance settings
	userCache            *cache.Cache // cache for users
	userSettingCache     *cache.Cache // cache for user settings

	// Store services
	AgentStatsStore    AgentStatsStore    // session statistics persistence
	SecurityAuditStore SecurityAuditStore // security audit logging
}

// New creates a new instance of Store.
func New(driver Driver, profile *profile.Profile) *Store {
	// Default cache settings
	cacheConfig := cache.Config{
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 5 * time.Minute,
		MaxItems:        1000,
		OnEviction:      nil,
	}

	store := &Store{
		driver:               driver,
		profile:              profile,
		cacheConfig:          cacheConfig,
		instanceSettingCache: cache.New(cacheConfig),
		userCache:            cache.New(cacheConfig),
		userSettingCache:     cache.New(cacheConfig),
		AgentStatsStore:      driver.AgentStatsStore(),
		SecurityAuditStore:   driver.SecurityAuditStore(),
	}

	return store
}

func (s *Store) GetDriver() Driver {
	return s.driver
}

func (s *Store) Close() error {
	// Stop all cache cleanup goroutines
	s.instanceSettingCache.Close()
	s.userCache.Close()
	s.userSettingCache.Close()

	return s.driver.Close()
}

func (s *Store) CreateAIConversation(ctx context.Context, create *AIConversation) (*AIConversation, error) {
	return s.driver.CreateAIConversation(ctx, create)
}

func (s *Store) ListAIConversations(ctx context.Context, find *FindAIConversation) ([]*AIConversation, error) {
	return s.driver.ListAIConversations(ctx, find)
}

func (s *Store) UpdateAIConversation(ctx context.Context, update *UpdateAIConversation) (*AIConversation, error) {
	return s.driver.UpdateAIConversation(ctx, update)
}

func (s *Store) DeleteAIConversation(ctx context.Context, delete *DeleteAIConversation) error {
	return s.driver.DeleteAIConversation(ctx, delete)
}

func (s *Store) CreateAIMessage(ctx context.Context, create *AIMessage) (*AIMessage, error) {
	return s.driver.CreateAIMessage(ctx, create)
}

func (s *Store) ListAIMessages(ctx context.Context, find *FindAIMessage) ([]*AIMessage, error) {
	return s.driver.ListAIMessages(ctx, find)
}

func (s *Store) DeleteAIMessage(ctx context.Context, delete *DeleteAIMessage) error {
	return s.driver.DeleteAIMessage(ctx, delete)
}

func (s *Store) CreateEpisodicMemory(ctx context.Context, create *EpisodicMemory) (*EpisodicMemory, error) {
	return s.driver.CreateEpisodicMemory(ctx, create)
}

func (s *Store) ListEpisodicMemories(ctx context.Context, find *FindEpisodicMemory) ([]*EpisodicMemory, error) {
	return s.driver.ListEpisodicMemories(ctx, find)
}

func (s *Store) ListActiveUserIDs(ctx context.Context, cutoff time.Time) ([]int32, error) {
	return s.driver.ListActiveUserIDs(ctx, cutoff)
}

func (s *Store) DeleteEpisodicMemory(ctx context.Context, delete *DeleteEpisodicMemory) error {
	return s.driver.DeleteEpisodicMemory(ctx, delete)
}

func (s *Store) UpsertUserPreferences(ctx context.Context, upsert *UpsertUserPreferences) (*UserPreferences, error) {
	return s.driver.UpsertUserPreferences(ctx, upsert)
}

func (s *Store) GetUserPreferences(ctx context.Context, find *FindUserPreferences) (*UserPreferences, error) {
	return s.driver.GetUserPreferences(ctx, find)
}

func (s *Store) UpsertAgentMetrics(ctx context.Context, upsert *UpsertAgentMetrics) (*AgentMetrics, error) {
	return s.driver.UpsertAgentMetrics(ctx, upsert)
}

func (s *Store) ListAgentMetrics(ctx context.Context, find *FindAgentMetrics) ([]*AgentMetrics, error) {
	return s.driver.ListAgentMetrics(ctx, find)
}

func (s *Store) DeleteAgentMetrics(ctx context.Context, delete *DeleteAgentMetrics) error {
	return s.driver.DeleteAgentMetrics(ctx, delete)
}

func (s *Store) UpsertToolMetrics(ctx context.Context, upsert *UpsertToolMetrics) (*ToolMetrics, error) {
	return s.driver.UpsertToolMetrics(ctx, upsert)
}

func (s *Store) ListToolMetrics(ctx context.Context, find *FindToolMetrics) ([]*ToolMetrics, error) {
	return s.driver.ListToolMetrics(ctx, find)
}

func (s *Store) DeleteToolMetrics(ctx context.Context, delete *DeleteToolMetrics) error {
	return s.driver.DeleteToolMetrics(ctx, delete)
}
