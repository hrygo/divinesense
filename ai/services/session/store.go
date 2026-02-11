package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/cache"
)

const (
	cachePrefix = "session:"
	cacheTTL    = 30 * time.Minute
)

// sessionStore implements SessionService with PostgreSQL persistence and caching.
type sessionStore struct {
	db    *sql.DB
	cache cache.CacheService
}

// NewSessionStore creates a new session store with database and cache.
func NewSessionStore(db *sql.DB, cache cache.CacheService) SessionService {
	return &sessionStore{
		db:    db,
		cache: cache,
	}
}

// SaveContext saves the conversation context.
func (s *sessionStore) SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error {
	// Set timestamps
	now := time.Now().Unix()
	if context.CreatedAt == 0 {
		context.CreatedAt = now
	}
	context.UpdatedAt = now
	context.SessionID = sessionID

	// Serialize context data (messages + metadata)
	contextData := struct {
		Metadata map[string]any `json:"metadata"`
		Messages []Message      `json:"messages"`
	}{
		Messages: context.Messages,
		Metadata: context.Metadata,
	}

	data, err := json.Marshal(contextData)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Upsert to database
	query := `
		INSERT INTO conversation_context (session_id, user_id, agent_type, context_data, created_ts, updated_ts)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (session_id) 
		DO UPDATE SET 
			agent_type = EXCLUDED.agent_type,
			context_data = EXCLUDED.context_data,
			updated_ts = EXCLUDED.updated_ts
	`

	_, err = s.db.ExecContext(ctx, query,
		sessionID, context.UserID, context.AgentType, data, context.CreatedAt, context.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// Update cache
	s.updateCache(ctx, sessionID, context)

	return nil
}

// LoadContext loads the conversation context.
func (s *sessionStore) LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error) {
	// Check cache first
	if cached := s.loadFromCache(ctx, sessionID); cached != nil {
		return cached, nil
	}

	// Query database
	query := `
		SELECT session_id, user_id, agent_type, context_data, created_ts, updated_ts
		FROM conversation_context 
		WHERE session_id = $1
	`

	var (
		result    ConversationContext
		data      []byte
		agentType string
	)

	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&result.SessionID, &result.UserID, &agentType, &data, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // New session
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load context: %w", err)
	}

	result.AgentType = agentType

	// Deserialize context data
	var contextData struct {
		Metadata map[string]any `json:"metadata"`
		Messages []Message      `json:"messages"`
	}
	if err := json.Unmarshal(data, &contextData); err != nil {
		slog.Warn("failed to unmarshal context data", "session_id", sessionID, "error", err)
		// Return with empty messages/metadata rather than failing
		result.Messages = []Message{}
		result.Metadata = map[string]any{}
	} else {
		result.Messages = contextData.Messages
		result.Metadata = contextData.Metadata
	}

	// Update cache
	s.updateCache(ctx, sessionID, &result)

	return &result, nil
}

// ListSessions lists user sessions.
func (s *sessionStore) ListSessions(ctx context.Context, userID int32, limit int) ([]SessionSummary, error) {
	query := `
		SELECT session_id, agent_type, context_data, updated_ts
		FROM conversation_context 
		WHERE user_id = $1
		ORDER BY updated_ts DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }() //nolint:errcheck // rows cleanup

	var summaries []SessionSummary
	for rows.Next() {
		var (
			sessionID string
			agentType string
			data      []byte
			updatedAt int64
		)

		if err := rows.Scan(&sessionID, &agentType, &data, &updatedAt); err != nil {
			slog.Warn("failed to scan session row", "error", err)
			continue
		}

		// Extract last message
		var contextData struct {
			Messages []Message `json:"messages"`
		}
		var lastMessage string
		if err := json.Unmarshal(data, &contextData); err == nil && len(contextData.Messages) > 0 {
			lastMessage = contextData.Messages[len(contextData.Messages)-1].Content
		}

		summaries = append(summaries, SessionSummary{
			SessionID:   sessionID,
			AgentType:   agentType,
			LastMessage: lastMessage,
			UpdatedAt:   updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate sessions: %w", err)
	}

	return summaries, nil
}

// DeleteSession deletes a session.
func (s *sessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	query := `DELETE FROM conversation_context WHERE session_id = $1`

	_, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Clear cache (idempotent - ok if not exists)
	s.invalidateCache(ctx, sessionID)

	return nil
}

// CleanupExpired removes sessions older than retentionDays.
func (s *sessionStore) CleanupExpired(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()

	query := `DELETE FROM conversation_context WHERE updated_ts < $1`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return result.RowsAffected()
}

// updateCache stores context in cache.
func (s *sessionStore) updateCache(ctx context.Context, sessionID string, context *ConversationContext) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(context)
	if err != nil {
		slog.Warn("failed to marshal context for cache", "error", err)
		return
	}

	key := cachePrefix + sessionID
	if err := s.cache.Set(ctx, key, data, cacheTTL); err != nil {
		slog.Warn("failed to update cache", "key", key, "error", err)
	}
}

// loadFromCache retrieves context from cache.
func (s *sessionStore) loadFromCache(ctx context.Context, sessionID string) *ConversationContext {
	if s.cache == nil {
		return nil
	}

	key := cachePrefix + sessionID
	data, ok := s.cache.Get(ctx, key)
	if !ok {
		return nil
	}

	var context ConversationContext
	if err := json.Unmarshal(data, &context); err != nil {
		slog.Warn("failed to unmarshal cached context", "key", key, "error", err)
		return nil
	}

	return &context
}

// invalidateCache removes context from cache.
func (s *sessionStore) invalidateCache(ctx context.Context, sessionID string) {
	if s.cache == nil {
		return
	}

	key := cachePrefix + sessionID
	if err := s.cache.Invalidate(ctx, key); err != nil {
		slog.Warn("failed to invalidate cache", "key", key, "error", err)
	}
}

// Ensure sessionStore implements SessionService.
var _ SessionService = (*sessionStore)(nil)
