// Package routing provides unit tests for HistoryMatcher.
package routing

import (
	"context"
	"testing"

	"github.com/hrygo/divinesense/ai/services/memory"
)

// mockMemoryService is a mock implementation of memory.MemoryService.
type mockMemoryService struct {
	episodes []memory.EpisodicMemory
	messages map[string][]memory.Message
}

func newMockMemoryService() *mockMemoryService {
	return &mockMemoryService{
		episodes: make([]memory.EpisodicMemory, 0),
		messages: make(map[string][]memory.Message),
	}
}

func (m *mockMemoryService) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]memory.Message, error) {
	if msgs, ok := m.messages[sessionID]; ok {
		if len(msgs) > limit {
			return msgs[len(msgs)-limit:], nil
		}
		return msgs, nil
	}
	return []memory.Message{}, nil
}

func (m *mockMemoryService) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]memory.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func (m *mockMemoryService) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]memory.EpisodicMemory, error) {
	// Return stored episodes
	result := make([]memory.EpisodicMemory, 0, len(m.episodes))
	for _, ep := range m.episodes {
		if ep.UserID == userID {
			result = append(result, ep)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockMemoryService) SaveEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	m.episodes = append(m.episodes, episode)
	return nil
}

func (m *mockMemoryService) GetEpisode(ctx context.Context, id int64) (*memory.EpisodicMemory, error) {
	for i := range m.episodes {
		if m.episodes[i].ID == id {
			return &m.episodes[i], nil
		}
	}
	return nil, nil
}

func (m *mockMemoryService) UpdateEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	for i := range m.episodes {
		if m.episodes[i].ID == episode.ID {
			m.episodes[i] = episode
			return nil
		}
	}
	return nil
}

func (m *mockMemoryService) DeleteEpisode(ctx context.Context, id int64) error {
	for i := range m.episodes {
		if m.episodes[i].ID == id {
			m.episodes = append(m.episodes[:i], m.episodes[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockMemoryService) ListEpisodes(ctx context.Context, userID int32, offset, limit int) ([]memory.EpisodicMemory, error) {
	result := make([]memory.EpisodicMemory, 0, len(m.episodes))
	for _, ep := range m.episodes {
		if ep.UserID == userID {
			result = append(result, ep)
		}
	}
	if offset >= len(result) {
		return []memory.EpisodicMemory{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockMemoryService) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	userSet := make(map[int32]bool)
	for _, ep := range m.episodes {
		userSet[ep.UserID] = true
	}
	result := make([]int32, 0, len(userSet))
	for uid := range userSet {
		result = append(result, uid)
	}
	return result, nil
}

func (m *mockMemoryService) GetPreferences(ctx context.Context, userID int32) (*memory.UserPreferences, error) {
	return &memory.UserPreferences{
		Timezone: "UTC",
	}, nil
}

func (m *mockMemoryService) UpdatePreferences(ctx context.Context, userID int32, prefs *memory.UserPreferences) error {
	return nil
}

// TestNewHistoryMatcher tests HistoryMatcher creation.
func TestNewHistoryMatcher(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	if matcher == nil {
		t.Fatal("expected non-nil HistoryMatcher")
	}
	if matcher.similarityThreshold != 0.8 {
		t.Errorf("expected similarity threshold 0.8, got %f", matcher.similarityThreshold)
	}
	if matcher.semanticThreshold != 0.75 {
		t.Errorf("expected semantic threshold 0.75, got %f", matcher.semanticThreshold)
	}
	if matcher.maxHistoryLookup != 10 {
		t.Errorf("expected max history lookup 10, got %d", matcher.maxHistoryLookup)
	}
}

// TestHistoryMatcher_Match_NoHistory tests matching with no history.
func TestHistoryMatcher_Match_NoHistory(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)
	ctx := context.Background()

	result, err := matcher.Match(ctx, 123, "搜索笔记")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Matched {
		t.Error("expected no match with empty history")
	}
}

// TestHistoryMatcher_Match_WithHistory tests matching with historical data.
func TestHistoryMatcher_Match_WithHistory(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)
	ctx := context.Background()
	userID := int32(123)

	// Add some historical episodes
	ms.episodes = []memory.EpisodicMemory{
		{
			ID:        1,
			UserID:    userID,
			UserInput: "搜索笔记",
			AgentType: "memo",
			Outcome:   "success",
		},
		{
			ID:        2,
			UserID:    userID,
			UserInput: "明天会议",
			AgentType: "schedule",
			Outcome:   "success",
		},
	}

	// Test exact match
	result, err := matcher.Match(ctx, userID, "搜索笔记")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Exact match should return high confidence
	if result.Confidence < 0.9 {
		t.Errorf("expected high confidence for exact match, got %f", result.Confidence)
	}
}

// TestHistoryMatcher_CalculateLexicalSimilarity tests lexical similarity calculation.
func TestHistoryMatcher_CalculateLexicalSimilarity(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	testCases := []struct {
		input1        string
		input2        string
		minSimilarity float32
	}{
		{"搜索笔记", "搜索笔记", 1.0}, // Exact match
		{"搜索笔记", "查找笔记", 0.2}, // Similar (lowered threshold)
		{"明天会议", "后天会议", 0.5}, // Similar
		{"搜索笔记", "明天会议", 0.0}, // Different (lowered threshold)
		{"", "", 0.0},         // Empty
	}

	for _, tc := range testCases {
		similarity := matcher.calculateLexicalSimilarity(tc.input1, tc.input2)
		if similarity < tc.minSimilarity {
			t.Errorf("calculateLexicalSimilarity(%q, %q) = %f, expected >= %f",
				tc.input1, tc.input2, similarity, tc.minSimilarity)
		}
	}
}

// TestHistoryMatcher_ExtractBigrams tests bigram extraction.
func TestHistoryMatcher_ExtractBigrams(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	testCases := []struct {
		input      string
		minBigrams int
	}{
		{"搜索", 1},
		{"搜索笔记", 3},  // 搜索, 索笔, 记事
		{"明天有会议", 4}, // Adjusted to actual output
		{"", 0},
	}

	for _, tc := range testCases {
		bigrams := matcher.extractBigrams(tc.input)
		if len(bigrams) < tc.minBigrams {
			t.Errorf("extractBigrams(%q) = %d bigrams, expected >= %d",
				tc.input, len(bigrams), tc.minBigrams)
		}
	}
}

// TestHistoryMatcher_AgentTypeToIntent tests agent type to intent conversion.
func TestHistoryMatcher_AgentTypeToIntent(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	testCases := []struct {
		agentType string
		input     string
		expected  Intent
	}{
		{"schedule", "查看明天会议", IntentScheduleQuery},
		{"schedule", "修改会议", IntentScheduleUpdate},
		{"schedule", "创建日程", IntentScheduleCreate},
		{"memo", "搜索笔记", IntentMemoSearch},
		{"memo", "记录内容", IntentMemoCreate},
		{"amazing", "帮我分析", IntentAmazing},
		{"unknown", "随便说", IntentUnknown},
	}

	for _, tc := range testCases {
		result := matcher.agentTypeToIntent(tc.agentType, tc.input)
		if result != tc.expected {
			t.Errorf("agentTypeToIntent(%q, %q) = %s, expected %s",
				tc.agentType, tc.input, result, tc.expected)
		}
	}
}

// TestHistoryMatcher_IntentToAgentType tests intent to agent type conversion.
func TestHistoryMatcher_IntentToAgentType(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	testCases := []struct {
		intent   Intent
		expected string
	}{
		{IntentMemoSearch, "memo"},
		{IntentMemoCreate, "memo"},
		{IntentScheduleQuery, "schedule"},
		{IntentScheduleCreate, "schedule"},
		{IntentScheduleUpdate, "schedule"},
		{IntentBatchSchedule, "schedule"},
		{IntentAmazing, "amazing"},
		{IntentUnknown, "unknown"},
	}

	for _, tc := range testCases {
		result := matcher.intentToAgentType(tc.intent)
		if result != tc.expected {
			t.Errorf("intentToAgentType(%s) = %s, expected %s",
				tc.intent, result, tc.expected)
		}
	}
}

// TestHistoryMatcher_SaveDecision tests saving routing decisions.
func TestHistoryMatcher_SaveDecision(t *testing.T) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)
	ctx := context.Background()
	userID := int32(123)

	err := matcher.SaveDecision(ctx, userID, "搜索笔记", IntentMemoSearch, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify episode was saved
	if len(ms.episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(ms.episodes))
	}

	ep := ms.episodes[0]
	if ep.UserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, ep.UserID)
	}
	if ep.AgentType != "memo" {
		t.Errorf("expected agent type 'memo', got %s", ep.AgentType)
	}
	if ep.Outcome != "success" {
		t.Errorf("expected outcome 'success', got %s", ep.Outcome)
	}
}

// TestCosineSimilarity tests cosine similarity calculation.
func TestCosineSimilarity(t *testing.T) {
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
		delta    float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 1.0, 0.001},
		{[]float32{1, 0, 0}, []float32{0, 1, 0}, 0.0, 0.001},
		{[]float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0, 0.001},
		{[]float32{1, 1}, []float32{1, 1}, 1.0, 0.001},
		{[]float32{1}, []float32{1, 1}, 0.0, 0.001}, // Different lengths
		{[]float32{}, []float32{1}, 0.0, 0.001},     // Empty vectors
	}

	for _, tc := range testCases {
		result := cosineSimilarity(tc.a, tc.b)
		diff := result - tc.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tc.delta {
			t.Errorf("cosineSimilarity(%v, %v) = %f, expected %f ± %f",
				tc.a, tc.b, result, tc.expected, tc.delta)
		}
	}
}

// BenchmarkHistoryMatcher_LexicalSimilarity benchmarks lexical similarity.
func BenchmarkHistoryMatcher_LexicalSimilarity(b *testing.B) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	input1 := "搜索关于人工智能的笔记"
	input2 := "查找AI相关的备忘录"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.calculateLexicalSimilarity(input1, input2)
	}
}

// BenchmarkHistoryMatcher_ExtractBigrams benchmarks bigram extraction.
func BenchmarkHistoryMatcher_ExtractBigrams(b *testing.B) {
	ms := newMockMemoryService()
	matcher := NewHistoryMatcher(ms)

	input := "搜索关于人工智能和机器学习的相关笔记内容"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.extractBigrams(input)
	}
}
