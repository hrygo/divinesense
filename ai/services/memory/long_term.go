package memory

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/store"
)

// ErrLongTermNotConfigured is returned when long-term memory operations are attempted
// but the store is not configured (e.g., SQLite mode).
var ErrLongTermNotConfigured = errors.New("long-term memory not configured (requires PostgreSQL)")

// LongTermMemory manages persistent episodic memories and user preferences.
type LongTermMemory struct {
	store *store.Store
}

// NewLongTermMemory creates a new long-term memory manager.
func NewLongTermMemory(s *store.Store) *LongTermMemory {
	return &LongTermMemory{store: s}
}

// SaveEpisode saves an episodic memory to persistent storage.
func (l *LongTermMemory) SaveEpisode(ctx context.Context, episode EpisodicMemory) error {
	storeEpisode := &store.EpisodicMemory{
		UserID:     episode.UserID,
		Timestamp:  episode.Timestamp,
		AgentType:  episode.AgentType,
		UserInput:  episode.UserInput,
		Outcome:    episode.Outcome,
		Summary:    episode.Summary,
		Importance: episode.Importance,
		CreatedTs:  time.Now().Unix(),
	}

	if storeEpisode.Timestamp.IsZero() {
		storeEpisode.Timestamp = time.Now()
	}

	_, err := l.store.CreateEpisodicMemory(ctx, storeEpisode)
	return err
}

// SearchEpisodes searches episodic memories by query.
func (l *LongTermMemory) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error) {
	if limit <= 0 {
		limit = 10
	}
	// Cap limit to prevent excessive data retrieval (Issue #7)
	if limit > 100 {
		limit = 100
	}

	find := &store.FindEpisodicMemory{
		UserID: &userID,
		Limit:  limit,
	}
	if query != "" {
		find.Query = &query
	}

	storeEpisodes, err := l.store.ListEpisodicMemories(ctx, find)
	if err != nil {
		return nil, err
	}

	episodes := make([]EpisodicMemory, len(storeEpisodes))
	for i, e := range storeEpisodes {
		episodes[i] = EpisodicMemory{
			ID:         e.ID,
			UserID:     e.UserID,
			Timestamp:  e.Timestamp,
			AgentType:  e.AgentType,
			UserInput:  e.UserInput,
			Outcome:    e.Outcome,
			Summary:    e.Summary,
			Importance: e.Importance,
		}
	}

	return episodes, nil
}

// ListActiveUserIDs returns user IDs with recent episodic activity.
func (l *LongTermMemory) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	return l.store.ListActiveUserIDs(ctx, cutoff)
}

// GetPreferences retrieves user preferences.
func (l *LongTermMemory) GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error) {
	prefs, err := l.store.GetUserPreferences(ctx, &store.FindUserPreferences{UserID: &userID})
	if err != nil {
		return nil, err
	}

	// Return default preferences if not found
	if prefs == nil {
		return DefaultPreferences(), nil
	}

	// Parse JSON preferences
	var userPrefs UserPreferences
	if err := json.Unmarshal([]byte(prefs.Preferences), &userPrefs); err != nil {
		// Log the error for debugging (Issue #4 fix)
		slog.Warn("failed to parse user preferences JSON, using defaults",
			"user_id", userID,
			"error", err,
		)
		return DefaultPreferences(), nil
	}

	return &userPrefs, nil
}

// UpdatePreferences updates user preferences.
func (l *LongTermMemory) UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error {
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	_, err = l.store.UpsertUserPreferences(ctx, &store.UpsertUserPreferences{
		UserID:      userID,
		Preferences: string(prefsJSON),
	})
	return err
}

// DefaultPreferences returns default user preferences.
// Exported to allow single source of truth (Issue #6 fix).
func DefaultPreferences() *UserPreferences {
	return &UserPreferences{
		Timezone:           "Asia/Shanghai",
		DefaultDuration:    60, // 1 hour
		PreferredTimes:     []string{"09:00", "14:00"},
		FrequentLocations:  []string{},
		CommunicationStyle: "concise",
		TagPreferences:     []string{},
		CustomSettings:     make(map[string]any),
	}
}
