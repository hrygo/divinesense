// Package context provides context building for LLM prompts.
// It orchestrates short-term memory, long-term memory, and retrieval results
// into an optimized context window for LLM inference.
package context

import (
	"context"
	"time"
)

// ContextBuilder builds optimized context for LLM inference.
type ContextBuilder interface {
	// Build constructs the full context from various sources.
	Build(ctx context.Context, req *ContextRequest) (*ContextResult, error)

	// BuildHistory constructs history in []string format for ParrotAgent.Execute.
	// Returns alternating user/assistant messages: [user1, assistant1, user2, assistant2, ...]
	// This enables backend-driven context construction (context-engineering.md Phase 1).
	BuildHistory(ctx context.Context, req *ContextRequest) ([]string, error)

	// GetStats returns context building statistics.
	GetStats() *ContextStats
}

// ContextRequest contains parameters for context building.
type ContextRequest struct {
	SessionID        string
	CurrentQuery     string
	AgentType        string
	RetrievalResults []*RetrievalItem
	MaxTokens        int
	UserID           int32
}

// RetrievalItem represents a single retrieval result.
type RetrievalItem struct {
	ID      string
	Content string
	Source  string
	Score   float32
}

// ContextResult contains the built context.
type ContextResult struct {
	TokenBreakdown      *TokenBreakdown
	SystemPrompt        string
	ConversationContext string
	RetrievalContext    string
	UserPreferences     string
	TotalTokens         int
	BuildTime           time.Duration
}

// TokenBreakdown shows how tokens are distributed.
type TokenBreakdown struct {
	SystemPrompt    int
	ShortTermMemory int
	LongTermMemory  int
	Retrieval       int
	UserPrefs       int
}

// ContextStats tracks context building metrics.
type ContextStats struct {
	TotalBuilds      int64
	AverageTokens    float64
	CacheHits        int64
	AverageBuildTime time.Duration
}

// Message represents a conversation message.
type Message struct {
	Timestamp time.Time
	Role      string
	Content   string
}

// EpisodicMemory represents a stored episode.
type EpisodicMemory struct {
	Timestamp time.Time
	Summary   string
	AgentType string
	Outcome   string
	ID        int64
}

// UserPreferences represents user preferences.
type UserPreferences struct {
	Timezone           string
	CommunicationStyle string
	PreferredTimes     []string
	DefaultDuration    int
}
