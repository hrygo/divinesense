package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// MockMemoryService is a mock implementation of MemoryService for testing.
type MockMemoryService struct {
	messages    map[string][]Message
	preferences map[int32]*UserPreferences
	episodes    []EpisodicMemory
	nextID      int64
	mu          sync.RWMutex
}

// NewMockMemoryService creates a new MockMemoryService with sample data.
func NewMockMemoryService() *MockMemoryService {
	mock := &MockMemoryService{
		messages:    make(map[string][]Message),
		episodes:    make([]EpisodicMemory, 0),
		preferences: make(map[int32]*UserPreferences),
		nextID:      1,
	}
	mock.seedData()
	return mock
}

// seedData populates the mock with sample data for testing.
func (m *MockMemoryService) seedData() {
	// Sample messages
	sessionID := "session-001"
	now := time.Now()
	sampleMessages := []Message{
		{Role: "user", Content: "明天下午3点提醒我开会", Timestamp: now.Add(-10 * time.Minute)},
		{Role: "assistant", Content: "好的，我已经为您设置了明天下午3点的开会提醒。", Timestamp: now.Add(-9 * time.Minute)},
		{Role: "user", Content: "帮我查一下上周的工作笔记", Timestamp: now.Add(-8 * time.Minute)},
		{Role: "assistant", Content: "我找到了5条上周的工作笔记，最近的一条是关于项目进度的。", Timestamp: now.Add(-7 * time.Minute)},
		{Role: "user", Content: "今天天气怎么样", Timestamp: now.Add(-6 * time.Minute)},
		{Role: "assistant", Content: "今天天气晴朗，气温25度，适合户外活动。", Timestamp: now.Add(-5 * time.Minute)},
		{Role: "user", Content: "记录一下：完成了Sprint 0的接口定义", Timestamp: now.Add(-4 * time.Minute)},
		{Role: "assistant", Content: "已记录：完成了Sprint 0的接口定义。", Timestamp: now.Add(-3 * time.Minute)},
		{Role: "user", Content: "最近有什么重要的日程吗", Timestamp: now.Add(-2 * time.Minute)},
		{Role: "assistant", Content: "您最近有3个重要日程：明天的会议、周五的项目评审、下周一的团队周会。", Timestamp: now.Add(-1 * time.Minute)},
	}
	m.messages[sessionID] = sampleMessages

	// Sample episodic memories
	m.episodes = []EpisodicMemory{
		{ID: 1, UserID: 1, Timestamp: now.Add(-24 * time.Hour), AgentType: "schedule", UserInput: "设置每日站会提醒", Outcome: "success", Summary: "用户设置了工作日9:30的每日站会提醒", Importance: 0.8},
		{ID: 2, UserID: 1, Timestamp: now.Add(-20 * time.Hour), AgentType: "memo", UserInput: "记录项目灵感", Outcome: "success", Summary: "用户记录了一个关于产品改进的灵感", Importance: 0.6},
		{ID: 3, UserID: 1, Timestamp: now.Add(-16 * time.Hour), AgentType: "amazing", UserInput: "今天股市行情", Outcome: "success", Summary: "用户查询了当日股市行情", Importance: 0.4},
		{ID: 4, UserID: 1, Timestamp: now.Add(-12 * time.Hour), AgentType: "schedule", UserInput: "取消下午的会议", Outcome: "success", Summary: "用户取消了原定下午3点的产品评审会", Importance: 0.7},
		{ID: 5, UserID: 1, Timestamp: now.Add(-8 * time.Hour), AgentType: "memo", UserInput: "搜索上周的笔记", Outcome: "success", Summary: "用户搜索并查看了上周的工作笔记", Importance: 0.5},
	}
	m.nextID = 6

	// Sample preferences
	m.preferences[1] = &UserPreferences{
		Timezone:           "Asia/Shanghai",
		DefaultDuration:    60,
		PreferredTimes:     []string{"09:00", "14:00", "19:00"},
		FrequentLocations:  []string{"办公室", "会议室A", "家"},
		CommunicationStyle: "concise",
		TagPreferences:     []string{"工作", "生活", "学习"},
		CustomSettings:     map[string]any{"theme": "dark", "notifications": true},
	}
}

// GetRecentMessages retrieves recent messages from a session.
func (m *MockMemoryService) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msgs, ok := m.messages[sessionID]
	if !ok {
		return []Message{}, nil
	}

	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}

	// Return the most recent messages
	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Message, limit)
	copy(result, msgs[start:])
	return result, nil
}

// AddMessage adds a message to a session.
func (m *MockMemoryService) AddMessage(ctx context.Context, sessionID string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

// SaveEpisode saves an episodic memory.
func (m *MockMemoryService) SaveEpisode(ctx context.Context, episode EpisodicMemory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	episode.ID = m.nextID
	m.nextID++

	if episode.Timestamp.IsZero() {
		episode.Timestamp = time.Now()
	}

	m.episodes = append(m.episodes, episode)
	return nil
}

// SearchEpisodes searches episodic memories for a specific user.
func (m *MockMemoryService) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []EpisodicMemory

	for _, ep := range m.episodes {
		// Filter by user ID first (multi-tenant isolation)
		if ep.UserID != userID {
			continue
		}

		if query == "" {
			// Empty query returns all user's episodes
			results = append(results, ep)
		} else {
			// Simple keyword search within user's episodes
			queryLower := strings.ToLower(query)
			if strings.Contains(strings.ToLower(ep.UserInput), queryLower) ||
				strings.Contains(strings.ToLower(ep.Summary), queryLower) {
				results = append(results, ep)
			}
		}
	}

	// Sort by timestamp descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

// ListActiveUserIDs returns user IDs with recent activity.
func (m *MockMemoryService) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().AddDate(0, 0, -lookbackDays)
	userSet := make(map[int32]struct{})

	for _, ep := range m.episodes {
		if ep.Timestamp.After(cutoff) {
			userSet[ep.UserID] = struct{}{}
		}
	}

	userIDs := make([]int32, 0, len(userSet))
	for id := range userSet {
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}

// GetPreferences retrieves user preferences.
func (m *MockMemoryService) GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prefs, ok := m.preferences[userID]
	if !ok {
		// Return default preferences
		return &UserPreferences{
			Timezone:           "Asia/Shanghai",
			DefaultDuration:    60,
			PreferredTimes:     []string{},
			FrequentLocations:  []string{},
			CommunicationStyle: "concise",
			TagPreferences:     []string{},
			CustomSettings:     map[string]any{},
		}, nil
	}

	// Return a copy to prevent mutation
	result := *prefs
	return &result, nil
}

// UpdatePreferences updates user preferences.
// prefs must not be nil.
func (m *MockMemoryService) UpdatePreferences(ctx context.Context, userID int32, prefs *UserPreferences) error {
	if prefs == nil {
		return fmt.Errorf("prefs must not be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store a copy
	copy := *prefs
	m.preferences[userID] = &copy
	return nil
}

// Ensure MockMemoryService implements MemoryService.
var _ MemoryService = (*MockMemoryService)(nil)
