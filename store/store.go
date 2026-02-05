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

// AIBlock methods (Unified Block Model).
// AIMessage functions removed: ALL IN Block!
func (s *Store) CreateAIBlock(ctx context.Context, create *CreateAIBlock) (*AIBlock, error) {
	return s.driver.CreateAIBlock(ctx, create)
}

func (s *Store) GetAIBlock(ctx context.Context, id int64) (*AIBlock, error) {
	return s.driver.GetAIBlock(ctx, id)
}

func (s *Store) ListAIBlocks(ctx context.Context, find *FindAIBlock) ([]*AIBlock, error) {
	return s.driver.ListAIBlocks(ctx, find)
}

func (s *Store) UpdateAIBlock(ctx context.Context, update *UpdateAIBlock) (*AIBlock, error) {
	return s.driver.UpdateAIBlock(ctx, update)
}

func (s *Store) DeleteAIBlock(ctx context.Context, id int64) error {
	return s.driver.DeleteAIBlock(ctx, id)
}

func (s *Store) AppendUserInput(ctx context.Context, blockID int64, input UserInput) error {
	return s.driver.AppendUserInput(ctx, blockID, input)
}

func (s *Store) AppendEvent(ctx context.Context, blockID int64, event BlockEvent) error {
	return s.driver.AppendEvent(ctx, blockID, event)
}

func (s *Store) AppendEventsBatch(ctx context.Context, blockID int64, events []BlockEvent) error {
	return s.driver.AppendEventsBatch(ctx, blockID, events)
}

func (s *Store) UpdateAIBlockStatus(ctx context.Context, blockID int64, status AIBlockStatus) error {
	return s.driver.UpdateAIBlockStatus(ctx, blockID, status)
}

func (s *Store) GetLatestAIBlock(ctx context.Context, conversationID int32) (*AIBlock, error) {
	return s.driver.GetLatestAIBlock(ctx, conversationID)
}

func (s *Store) GetPendingAIBlocks(ctx context.Context) ([]*AIBlock, error) {
	return s.driver.GetPendingAIBlocks(ctx)
}

func (s *Store) CreateAIBlockWithRound(ctx context.Context, create *CreateAIBlock) (*AIBlock, error) {
	return s.driver.CreateAIBlockWithRound(ctx, create)
}

func (s *Store) CompleteBlock(ctx context.Context, blockID int64, assistantContent string, sessionStats *SessionStats) error {
	return s.driver.CompleteBlock(ctx, blockID, assistantContent, sessionStats)
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
