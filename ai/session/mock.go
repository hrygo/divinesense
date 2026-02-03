package session

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MockSessionService is a mock implementation of SessionService for testing.
type MockSessionService struct {
	sessions map[string]*ConversationContext
	mu       sync.RWMutex
}

// NewMockSessionService creates a new MockSessionService with sample data.
func NewMockSessionService() *MockSessionService {
	mock := &MockSessionService{
		sessions: make(map[string]*ConversationContext),
	}
	mock.seedData()
	return mock
}

// seedData populates the mock with sample data for testing.
func (m *MockSessionService) seedData() {
	now := time.Now().Unix()

	// Sample sessions
	m.sessions["session-001"] = &ConversationContext{
		SessionID: "session-001",
		UserID:    1,
		AgentType: "memo",
		Messages: []Message{
			{Role: "user", Content: "帮我查找上周的笔记"},
			{Role: "assistant", Content: "我找到了5条上周的笔记。"},
		},
		Metadata:  map[string]any{"topic": "笔记搜索"},
		CreatedAt: now - 3600,
		UpdatedAt: now - 1800,
	}

	m.sessions["session-002"] = &ConversationContext{
		SessionID: "session-002",
		UserID:    1,
		AgentType: "schedule",
		Messages: []Message{
			{Role: "user", Content: "明天下午3点提醒我开会"},
			{Role: "assistant", Content: "好的，已设置提醒。"},
		},
		Metadata:  map[string]any{"topic": "日程创建"},
		CreatedAt: now - 7200,
		UpdatedAt: now - 3600,
	}

	m.sessions["session-003"] = &ConversationContext{
		SessionID: "session-003",
		UserID:    1,
		AgentType: "amazing",
		Messages: []Message{
			{Role: "user", Content: "今天天气怎么样？"},
			{Role: "assistant", Content: "今天天气晴朗，气温25度。"},
		},
		Metadata:  map[string]any{"topic": "天气查询"},
		CreatedAt: now - 10800,
		UpdatedAt: now - 7200,
	}

	// Another user's session
	m.sessions["session-004"] = &ConversationContext{
		SessionID: "session-004",
		UserID:    2,
		AgentType: "memo",
		Messages: []Message{
			{Role: "user", Content: "记录一下项目进度"},
		},
		Metadata:  map[string]any{},
		CreatedAt: now - 1800,
		UpdatedAt: now - 900,
	}
}

// SaveContext saves the conversation context.
func (m *MockSessionService) SaveContext(ctx context.Context, sessionID string, context *ConversationContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure timestamps are set
	now := time.Now().Unix()
	if context.CreatedAt == 0 {
		context.CreatedAt = now
	}
	context.UpdatedAt = now
	context.SessionID = sessionID

	// Store a copy
	copy := *context
	copy.Messages = make([]Message, len(context.Messages))
	for i, msg := range context.Messages {
		copy.Messages[i] = msg
	}
	if context.Metadata != nil {
		copy.Metadata = make(map[string]any)
		for k, v := range context.Metadata {
			copy.Metadata[k] = v
		}
	}

	m.sessions[sessionID] = &copy
	return nil
}

// LoadContext loads the conversation context.
func (m *MockSessionService) LoadContext(ctx context.Context, sessionID string) (*ConversationContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, nil
	}

	// Return a copy
	copy := *session
	copy.Messages = make([]Message, len(session.Messages))
	for i, msg := range session.Messages {
		copy.Messages[i] = msg
	}
	if session.Metadata != nil {
		copy.Metadata = make(map[string]any)
		for k, v := range session.Metadata {
			copy.Metadata[k] = v
		}
	}

	return &copy, nil
}

// ListSessions lists user sessions.
func (m *MockSessionService) ListSessions(ctx context.Context, userID int32, limit int) ([]SessionSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []SessionSummary

	for _, session := range m.sessions {
		if session.UserID != userID {
			continue
		}

		var lastMessage string
		if len(session.Messages) > 0 {
			lastMessage = session.Messages[len(session.Messages)-1].Content
		}

		summaries = append(summaries, SessionSummary{
			SessionID:   session.SessionID,
			AgentType:   session.AgentType,
			LastMessage: lastMessage,
			UpdatedAt:   session.UpdatedAt,
		})
	}

	// Sort by UpdatedAt descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt > summaries[j].UpdatedAt
	})

	// Apply limit
	if limit > 0 && limit < len(summaries) {
		summaries = summaries[:limit]
	}

	return summaries, nil
}

// DeleteSession deletes a session.
func (m *MockSessionService) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[sessionID]; !ok {
		return nil // Not an error if not found
	}

	delete(m.sessions, sessionID)
	return nil
}

// CleanupExpired removes sessions older than retentionDays.
func (m *MockSessionService) CleanupExpired(ctx context.Context, retentionDays int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	var deleted int64

	for sessionID, session := range m.sessions {
		if session.UpdatedAt < cutoff {
			delete(m.sessions, sessionID)
			deleted++
		}
	}

	return deleted, nil
}

// Clear removes all sessions (for testing).
func (m *MockSessionService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]*ConversationContext)
}

// SetSessionDirectly sets a session directly without updating timestamps (for testing).
func (m *MockSessionService) SetSessionDirectly(sessionID string, context *ConversationContext) {
	m.mu.Lock()
	defer m.mu.Unlock()

	context.SessionID = sessionID
	m.sessions[sessionID] = context
}

// Ensure MockSessionService implements SessionService.
var _ SessionService = (*MockSessionService)(nil)
