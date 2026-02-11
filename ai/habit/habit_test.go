package habit

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/services/memory"
)

// mockMemoryService implements memory.MemoryService for testing.
type mockMemoryService struct {
	episodes      []memory.EpisodicMemory
	preferences   *memory.UserPreferences
	activeUserIDs []int32
}

func (m *mockMemoryService) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]memory.Message, error) {
	return nil, nil
}

func (m *mockMemoryService) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	return nil
}

func (m *mockMemoryService) SaveEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	return nil
}

func (m *mockMemoryService) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]memory.EpisodicMemory, error) {
	return m.episodes, nil
}

func (m *mockMemoryService) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	if m.activeUserIDs != nil {
		return m.activeUserIDs, nil
	}
	// Extract unique user IDs from episodes
	userSet := make(map[int32]struct{})
	for _, ep := range m.episodes {
		userSet[ep.UserID] = struct{}{}
	}
	userIDs := make([]int32, 0, len(userSet))
	for id := range userSet {
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}

func (m *mockMemoryService) GetPreferences(ctx context.Context, userID int32) (*memory.UserPreferences, error) {
	return m.preferences, nil
}

func (m *mockMemoryService) UpdatePreferences(ctx context.Context, userID int32, prefs *memory.UserPreferences) error {
	m.preferences = prefs
	return nil
}

// Helper to generate mock episodes.
func generateMockEpisodes(count int, hours []int, agentType string) []memory.EpisodicMemory {
	episodes := make([]memory.EpisodicMemory, count)
	now := time.Now()
	// Truncate to start of day to avoid time drift
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for i := 0; i < count; i++ {
		hour := hours[i%len(hours)]
		// Set exact hour on the target date
		ts := today.AddDate(0, 0, -i%30).Add(time.Duration(hour) * time.Hour)
		episodes[i] = memory.EpisodicMemory{
			ID:        int64(i + 1),
			UserID:    1,
			Timestamp: ts,
			AgentType: agentType,
			UserInput: "test input " + string(rune('a'+i%26)),
			Outcome:   "success",
		}
	}
	return episodes
}

func TestDefaultUserHabits(t *testing.T) {
	habits := DefaultUserHabits(1)

	if habits.UserID != 1 {
		t.Errorf("UserID = %d, want 1", habits.UserID)
	}

	if habits.Time == nil {
		t.Error("Time habits should not be nil")
	}

	if habits.Schedule == nil {
		t.Error("Schedule habits should not be nil")
	}

	if habits.Search == nil {
		t.Error("Search habits should not be nil")
	}
}

func TestAnalyzeTimeHabits(t *testing.T) {
	// Create episodes with activity at hours 9, 10, 14, 15
	episodes := generateMockEpisodes(100, []int{9, 10, 14, 15}, "schedule")

	mockSvc := &mockMemoryService{episodes: episodes}
	analyzer := NewHabitAnalyzer(mockSvc, DefaultAnalysisConfig())

	habits, err := analyzer.Analyze(context.Background(), 1)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should identify active hours
	if len(habits.Time.ActiveHours) == 0 {
		t.Error("Expected active hours to be identified")
	}

	// Check that some of the expected hours are present
	foundExpected := false
	for _, h := range habits.Time.ActiveHours {
		if h == 9 || h == 10 || h == 14 || h == 15 {
			foundExpected = true
			break
		}
	}
	if !foundExpected {
		t.Errorf("Expected to find hours 9, 10, 14, or 15 in active hours: %v", habits.Time.ActiveHours)
	}
}

func TestAnalyzeScheduleHabits(t *testing.T) {
	episodes := generateMockEpisodes(50, []int{9, 14}, "schedule")

	mockSvc := &mockMemoryService{episodes: episodes}
	analyzer := NewHabitAnalyzer(mockSvc, DefaultAnalysisConfig())

	habits, err := analyzer.Analyze(context.Background(), 1)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should have schedule habits
	if habits.Schedule == nil {
		t.Error("Schedule habits should not be nil")
	}

	// Should have preferred slots
	if len(habits.Schedule.PreferredSlots) == 0 {
		t.Error("Expected preferred slots to be identified")
	}
}

func TestAnalyzeInsufficientData(t *testing.T) {
	// Only 5 episodes - below minimum threshold
	episodes := generateMockEpisodes(5, []int{9}, "schedule")

	mockSvc := &mockMemoryService{episodes: episodes}
	analyzer := NewHabitAnalyzer(mockSvc, DefaultAnalysisConfig())

	habits, err := analyzer.Analyze(context.Background(), 1)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Should return defaults
	defaults := DefaultUserHabits(1)
	if len(habits.Time.PreferredTimes) != len(defaults.Time.PreferredTimes) {
		t.Error("Expected default habits when data is insufficient")
	}
}

func TestTopNHours(t *testing.T) {
	hourCounts := map[int]int{
		9:  10,
		10: 8,
		14: 15,
		15: 12,
		20: 3,
	}

	top3 := topNHours(hourCounts, 3)

	if len(top3) != 3 {
		t.Errorf("Expected 3 hours, got %d", len(top3))
	}

	// Should be sorted
	for i := 1; i < len(top3); i++ {
		if top3[i] < top3[i-1] {
			t.Error("Result should be sorted by hour")
		}
	}
}

func TestTopNStrings(t *testing.T) {
	counts := map[string]int{
		"morning":   5,
		"afternoon": 10,
		"evening":   3,
		"night":     1,
	}

	top2 := topNStrings(counts, 2)

	if len(top2) != 2 {
		t.Errorf("Expected 2 strings, got %d", len(top2))
	}

	if top2[0] != "afternoon" {
		t.Errorf("Expected 'afternoon' first, got %s", top2[0])
	}
}

func TestHourToSlot(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{8, "morning"},
		{12, "noon"},
		{15, "afternoon"},
		{19, "evening"},
		{2, "night"},
	}

	for _, tt := range tests {
		result := hourToSlot(tt.hour)
		if result != tt.expected {
			t.Errorf("hourToSlot(%d) = %s, want %s", tt.hour, result, tt.expected)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"今天有什么安排", 1}, // No spaces, single "word"
		{"查看 日程", 2},
		{"a, b, c", 3},
	}

	for _, tt := range tests {
		result := tokenize(tt.input)
		if len(result) != tt.expected {
			t.Errorf("tokenize(%q) = %d words, want %d", tt.input, len(result), tt.expected)
		}
	}
}

func TestHasExactQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`"exact"`, true},
		{`'exact'`, true},
		{"no quotes", false},
		{`"`, false},
		{`""`, false},
	}

	for _, tt := range tests {
		result := hasExactQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("hasExactQuotes(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestHasCommonChars(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		// CJK: single char match
		{"日程", "日报", true},
		{"会议", "会场", true},
		{"项目", "任务", false},
		// ASCII: 2+ consecutive chars required
		{"meeting", "meet", true},
		{"project", "pro", true},
		{"hello", "world", false},
		{"test", "best", true}, // "est" matches
		// Mixed
		{"项目meeting", "项目", true},
	}

	for _, tt := range tests {
		result := hasCommonChars(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("hasCommonChars(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestHabitApplier_ApplyToScheduleCreate(t *testing.T) {
	prefs := &memory.UserPreferences{
		DefaultDuration:   45,
		PreferredTimes:    []string{"09:00", "14:00"},
		FrequentLocations: []string{"会议室A", "咖啡厅"},
	}

	mockSvc := &mockMemoryService{preferences: prefs}
	applier := NewHabitApplier(mockSvc)

	input := &ScheduleInput{
		Title: "Test Meeting",
	}

	result := applier.ApplyToScheduleCreate(context.Background(), 1, input)

	if result.SuggestedDuration != 45 {
		t.Errorf("SuggestedDuration = %d, want 45", result.SuggestedDuration)
	}

	if len(result.SuggestedTimes) != 2 {
		t.Errorf("SuggestedTimes length = %d, want 2", len(result.SuggestedTimes))
	}

	if len(result.SuggestedLocations) != 2 {
		t.Errorf("SuggestedLocations length = %d, want 2", len(result.SuggestedLocations))
	}
}

func TestHabitApplier_InferTime(t *testing.T) {
	prefs := &memory.UserPreferences{
		PreferredTimes: []string{"09:00", "14:00", "19:00"},
	}

	mockSvc := &mockMemoryService{preferences: prefs}
	applier := NewHabitApplier(mockSvc)

	tests := []struct {
		query        string
		expectedHour int
	}{
		{"上午开会", 9},
		{"下午开会", 14},
		{"晚上开会", 19},
	}

	for _, tt := range tests {
		result := applier.InferTime(context.Background(), 1, tt.query)
		if !result.IsZero() && result.Hour() != tt.expectedHour {
			t.Errorf("InferTime(%q) hour = %d, want %d", tt.query, result.Hour(), tt.expectedHour)
		}
	}
}

func TestHabitLearner_MergeHabitsToPreferences(t *testing.T) {
	habits := &UserHabits{
		Time: &TimeHabits{
			PreferredTimes: []string{"09:00", "14:00"},
		},
		Schedule: &ScheduleHabits{
			DefaultDuration:   60,
			FrequentLocations: []string{"Office", "Home"},
		},
		Search: &SearchHabits{
			FrequentKeywords: []string{"meeting", "project"},
		},
	}

	// Test with nil existing preferences
	prefs := mergeHabitsToPreferences(habits, nil)

	if len(prefs.PreferredTimes) != 2 {
		t.Errorf("PreferredTimes length = %d, want 2", len(prefs.PreferredTimes))
	}

	if prefs.DefaultDuration != 60 {
		t.Errorf("DefaultDuration = %d, want 60", prefs.DefaultDuration)
	}

	if len(prefs.FrequentLocations) != 2 {
		t.Errorf("FrequentLocations length = %d, want 2", len(prefs.FrequentLocations))
	}

	if len(prefs.TagPreferences) != 2 {
		t.Errorf("TagPreferences length = %d, want 2", len(prefs.TagPreferences))
	}
}

func TestHabitLearner_MergePreservesExisting(t *testing.T) {
	habits := &UserHabits{
		Time: &TimeHabits{
			PreferredTimes: []string{"09:00"},
		},
		Schedule: &ScheduleHabits{
			DefaultDuration: 45,
		},
	}

	existing := &memory.UserPreferences{
		Timezone:           "Asia/Shanghai",
		CommunicationStyle: "concise",
		CustomSettings:     map[string]any{"theme": "dark"},
	}

	prefs := mergeHabitsToPreferences(habits, existing)

	// Verify existing fields are preserved
	if prefs.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %s, want Asia/Shanghai", prefs.Timezone)
	}

	if prefs.CommunicationStyle != "concise" {
		t.Errorf("CommunicationStyle = %s, want concise", prefs.CommunicationStyle)
	}

	if prefs.CustomSettings["theme"] != "dark" {
		t.Error("CustomSettings should be preserved")
	}

	// Verify habit fields are merged
	if len(prefs.PreferredTimes) != 1 || prefs.PreferredTimes[0] != "09:00" {
		t.Errorf("PreferredTimes not merged correctly: %v", prefs.PreferredTimes)
	}

	if prefs.DefaultDuration != 45 {
		t.Errorf("DefaultDuration = %d, want 45", prefs.DefaultDuration)
	}
}

func TestFilterByFrequency(t *testing.T) {
	counts := map[string]int{
		"common":   5,
		"frequent": 10,
		"rare":     1,
		"medium":   3,
	}

	// Min frequency 3
	result := filterByFrequency(counts, 3)

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}

	// Should not include "rare"
	for _, s := range result {
		if s == "rare" {
			t.Error("Should not include 'rare' (frequency 1)")
		}
	}
}

// Benchmark tests.
func BenchmarkAnalyze(b *testing.B) {
	episodes := generateMockEpisodes(500, []int{9, 10, 14, 15, 16}, "schedule")
	mockSvc := &mockMemoryService{episodes: episodes}
	analyzer := NewHabitAnalyzer(mockSvc, DefaultAnalysisConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(context.Background(), 1)
	}
}

func BenchmarkTopNHours(b *testing.B) {
	hourCounts := make(map[int]int)
	for i := 0; i < 24; i++ {
		hourCounts[i] = i * 10
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		topNHours(hourCounts, 5)
	}
}
