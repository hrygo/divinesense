// Package memory provides the unified memory service interface for AI agents.
// This interface is consumed by Team B (Assistant+Schedule) and Team C (Memo Enhancement).
package memory

import (
	"context"
	"time"
)

// Consumers: Team B (Assistant+Schedule), Team C (Memo Enhancement).
type MemoryService interface {
	// ========== Short-term Memory (within session) ==========

	// GetRecentMessages retrieves recent messages from a session.
	// limit: maximum number of messages to return, recommended 10
	GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)

	// AddMessage adds a message to a session.
	AddMessage(ctx context.Context, sessionID string, msg Message) error

	// ========== Long-term Memory (cross-session) ==========

	// SaveEpisode saves an episodic memory.
	SaveEpisode(ctx context.Context, episode EpisodicMemory) error

	// SearchEpisodes searches episodic memories for a specific user.
	// userID: required, ensures multi-tenant data isolation
	// query: search keywords, empty string returns most recent records
	// limit: maximum number of results to return
	SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error)

	// ListActiveUserIDs returns user IDs with recent episodic activity.
	// lookbackDays: how many days to look back for activity
	// Returns unique user IDs that have at least one episode within the lookback period.
	ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error)

	// ========== User Preferences ==========

	// GetPreferences retrieves user preferences.
	GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error)

	// UpdatePreferences updates user preferences.
	UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error
}

// Message represents a conversation message.
type Message struct {
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
}

// EpisodicMemory represents an episodic memory record.
type EpisodicMemory struct {
	Timestamp  time.Time `json:"timestamp"`
	AgentType  string    `json:"agent_type"`
	UserInput  string    `json:"user_input"`
	Outcome    string    `json:"outcome"`
	Summary    string    `json:"summary"`
	ID         int64     `json:"id"`
	UserID     int32     `json:"user_id"`
	Importance float32   `json:"importance"`
}

// UserPreferences represents user preferences.
type UserPreferences struct {
	CustomSettings     map[string]any `json:"custom_settings"`
	Timezone           string         `json:"timezone"`
	CommunicationStyle string         `json:"communication_style"`
	PreferredTimes     []string       `json:"preferred_times"`
	FrequentLocations  []string       `json:"frequent_locations"`
	TagPreferences     []string       `json:"tag_preferences"`
	DefaultDuration    int            `json:"default_duration"`
}
