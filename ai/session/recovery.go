package session

import (
	"context"
	"fmt"
	"time"
)

const (
	// MaxMessagesPerSession is the maximum number of messages to keep in a session.
	// This implements a sliding window to prevent unbounded growth.
	MaxMessagesPerSession = 20
)

// SessionRecovery handles session recovery and message management.
type SessionRecovery struct {
	sessionSvc SessionService
}

// NewSessionRecovery creates a new session recovery handler.
func NewSessionRecovery(sessionSvc SessionService) *SessionRecovery {
	return &SessionRecovery{
		sessionSvc: sessionSvc,
	}
}

// RecoverSession recovers or creates a session for the given user.
// If the session exists, it returns the existing context.
// If not, it creates a new empty context.
func (r *SessionRecovery) RecoverSession(ctx context.Context, sessionID string, userID int32) (*ConversationContext, error) {
	// Try to load existing session
	existing, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	if existing != nil {
		// Session exists, return it
		return existing, nil
	}

	// New session: initialize empty context
	newContext := &ConversationContext{
		SessionID: sessionID,
		UserID:    userID,
		Messages:  make([]Message, 0, MaxMessagesPerSession),
		Metadata:  make(map[string]any),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	return newContext, nil
}

// AppendMessage adds a message to the session and saves it.
// Implements sliding window to keep only the last MaxMessagesPerSession messages.
func (r *SessionRecovery) AppendMessage(ctx context.Context, sessionID string, msg *Message) error {
	session, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Add message
	session.Messages = append(session.Messages, *msg)

	// Apply sliding window
	if len(session.Messages) > MaxMessagesPerSession {
		session.Messages = session.Messages[len(session.Messages)-MaxMessagesPerSession:]
	}

	// Update timestamp and agent type
	session.UpdatedAt = time.Now().Unix()

	// Save
	return r.sessionSvc.SaveContext(ctx, sessionID, session)
}

// AppendTurn adds a user-assistant turn to the session.
// This is a convenience method for the common case of adding both messages.
func (r *SessionRecovery) AppendTurn(ctx context.Context, sessionID string, userMsg, assistantMsg string, agentType string) error {
	session, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Add both messages
	session.Messages = append(session.Messages,
		Message{Role: "user", Content: userMsg},
		Message{Role: "assistant", Content: assistantMsg},
	)

	// Apply sliding window
	if len(session.Messages) > MaxMessagesPerSession {
		session.Messages = session.Messages[len(session.Messages)-MaxMessagesPerSession:]
	}

	// Update metadata
	session.UpdatedAt = time.Now().Unix()
	if agentType != "" {
		session.AgentType = agentType
	}

	// Save
	return r.sessionSvc.SaveContext(ctx, sessionID, session)
}

// UpdateMetadata updates session metadata without modifying messages.
func (r *SessionRecovery) UpdateMetadata(ctx context.Context, sessionID string, key string, value any) error {
	session, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Metadata == nil {
		session.Metadata = make(map[string]any)
	}
	session.Metadata[key] = value
	session.UpdatedAt = time.Now().Unix()

	return r.sessionSvc.SaveContext(ctx, sessionID, session)
}

// GetRecentMessages returns the last N messages from the session.
func (r *SessionRecovery) GetRecentMessages(ctx context.Context, sessionID string, n int) ([]Message, error) {
	session, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return nil, nil
	}

	messages := session.Messages
	if n > 0 && n < len(messages) {
		messages = messages[len(messages)-n:]
	}

	// Return a copy to prevent caller from modifying session state
	result := make([]Message, len(messages))
	copy(result, messages)
	return result, nil
}

// ClearMessages clears all messages from the session while preserving metadata.
func (r *SessionRecovery) ClearMessages(ctx context.Context, sessionID string) error {
	session, err := r.sessionSvc.LoadContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Messages = []Message{}
	session.UpdatedAt = time.Now().Unix()

	return r.sessionSvc.SaveContext(ctx, sessionID, session)
}
