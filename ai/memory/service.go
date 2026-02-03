package memory

import (
	"context"
	"log/slog"

	"github.com/hrygo/divinesense/store"
)

// - Long-term: PostgreSQL for episodic memories and user preferences.
type Service struct {
	shortTerm *ShortTermMemory
	longTerm  *LongTermMemory
}

// maxShortTermMessages: maximum messages per session (default 10).
func NewService(s *store.Store, maxShortTermMessages int) *Service {
	svc := &Service{
		shortTerm: NewShortTermMemory(maxShortTermMessages),
	}
	if s != nil {
		svc.longTerm = NewLongTermMemory(s)
	} else {
		slog.Warn("memory service initialized without long-term store (episodic memory and preferences disabled)")
	}
	return svc
}

// Close releases resources held by the service.
func (s *Service) Close() {
	if s.shortTerm != nil {
		s.shortTerm.Close()
	}
}

// ========== Short-term Memory (within session) ==========

// GetRecentMessages retrieves recent messages from a session.
func (s *Service) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	return s.shortTerm.GetMessages(sessionID, limit), nil
}

// AddMessage adds a message to a session.
func (s *Service) AddMessage(ctx context.Context, sessionID string, msg Message) error {
	s.shortTerm.AddMessage(sessionID, msg)
	return nil
}

// ========== Long-term Memory (cross-session) ==========

// SaveEpisode saves an episodic memory.
// Returns ErrLongTermNotConfigured if long-term memory is not available.
func (s *Service) SaveEpisode(ctx context.Context, episode EpisodicMemory) error {
	if s.longTerm == nil {
		return ErrLongTermNotConfigured
	}
	return s.longTerm.SaveEpisode(ctx, episode)
}

// SearchEpisodes searches episodic memories for a specific user.
// Returns ErrLongTermNotConfigured if long-term memory is not available.
func (s *Service) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error) {
	if s.longTerm == nil {
		return nil, ErrLongTermNotConfigured
	}
	return s.longTerm.SearchEpisodes(ctx, userID, query, limit)
}

// ListActiveUserIDs returns user IDs with recent episodic activity.
// Returns ErrLongTermNotConfigured if long-term memory is not available.
func (s *Service) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	if s.longTerm == nil {
		return nil, ErrLongTermNotConfigured
	}
	return s.longTerm.ListActiveUserIDs(ctx, lookbackDays)
}

// ========== User Preferences ==========

// GetPreferences retrieves user preferences.
// Returns default preferences if long-term memory is not available.
func (s *Service) GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error) {
	if s.longTerm == nil {
		// For preferences, return defaults instead of error for better UX
		return DefaultPreferences(), nil
	}
	return s.longTerm.GetPreferences(ctx, userID)
}

// UpdatePreferences updates user preferences.
// Returns ErrLongTermNotConfigured if long-term memory is not available.
func (s *Service) UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error {
	if s.longTerm == nil {
		return ErrLongTermNotConfigured
	}
	return s.longTerm.UpdatePreferences(ctx, userID, prefs)
}

// ========== Session Management ==========

// ClearSession removes all short-term messages from a session.
func (s *Service) ClearSession(sessionID string) {
	s.shortTerm.ClearSession(sessionID)
}

// ActiveSessionCount returns the number of active sessions.
func (s *Service) ActiveSessionCount() int {
	return s.shortTerm.SessionCount()
}

// HasLongTermMemory returns true if long-term memory is configured.
func (s *Service) HasLongTermMemory() bool {
	return s.longTerm != nil
}

// Ensure Service implements MemoryService interface.
var _ MemoryService = (*Service)(nil)
