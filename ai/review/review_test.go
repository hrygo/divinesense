package review

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxDailyReviews != 20 {
		t.Errorf("MaxDailyReviews = %d, want 20", config.MaxDailyReviews)
	}
	if config.NewCardsPerDay != 5 {
		t.Errorf("NewCardsPerDay = %d, want 5", config.NewCardsPerDay)
	}
	if config.DailyReviewTime != 9 {
		t.Errorf("DailyReviewTime = %d, want 9", config.DailyReviewTime)
	}
	if !config.EnableNotifications {
		t.Error("EnableNotifications should be true by default")
	}
}

func TestReviewQualityConstants(t *testing.T) {
	if QualityAgain != 0 {
		t.Errorf("QualityAgain = %d, want 0", QualityAgain)
	}
	if QualityHard != 1 {
		t.Errorf("QualityHard = %d, want 1", QualityHard)
	}
	if QualityGood != 2 {
		t.Errorf("QualityGood = %d, want 2", QualityGood)
	}
	if QualityEasy != 3 {
		t.Errorf("QualityEasy = %d, want 3", QualityEasy)
	}
}

func TestDefaultEaseFactorConstants(t *testing.T) {
	if DefaultEaseFactor != 2.5 {
		t.Errorf("DefaultEaseFactor = %f, want 2.5", DefaultEaseFactor)
	}
	if MinEaseFactor != 1.3 {
		t.Errorf("MinEaseFactor = %f, want 1.3", MinEaseFactor)
	}
}

func TestApplySM2Algorithm_QualityAgain(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 7,
		ReviewCount:  3,
	}

	result := s.applySM2Algorithm(state, QualityAgain)

	if result.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1 (reset on Again)", result.IntervalDays)
	}
	if result.ReviewCount != 4 {
		t.Errorf("ReviewCount = %d, want 4", result.ReviewCount)
	}
	if result.LastReview.IsZero() {
		t.Error("LastReview should be set")
	}
}

func TestApplySM2Algorithm_QualityHard(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 10,
		ReviewCount:  5,
	}

	result := s.applySM2Algorithm(state, QualityHard)

	// Hard multiplies by 1.2
	expectedInterval := int(10 * 1.2) // 12
	if result.IntervalDays != expectedInterval {
		t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, expectedInterval)
	}
}

func TestApplySM2Algorithm_QualityGood_FirstReview(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 0,
		ReviewCount:  0,
	}

	result := s.applySM2Algorithm(state, QualityGood)

	if result.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1 (first review)", result.IntervalDays)
	}
}

func TestApplySM2Algorithm_QualityGood_SecondReview(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 1,
		ReviewCount:  1,
	}

	result := s.applySM2Algorithm(state, QualityGood)

	if result.IntervalDays != 3 {
		t.Errorf("IntervalDays = %d, want 3 (second review)", result.IntervalDays)
	}
}

func TestApplySM2Algorithm_QualityGood_SubsequentReviews(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 3,
		ReviewCount:  2,
	}

	result := s.applySM2Algorithm(state, QualityGood)

	// Should multiply by ease factor: 3 * 2.5 = 7.5 -> 7
	expectedInterval := 7
	if result.IntervalDays != expectedInterval {
		t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, expectedInterval)
	}
}

func TestApplySM2Algorithm_QualityEasy_FirstReview(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 0,
		ReviewCount:  0,
	}

	result := s.applySM2Algorithm(state, QualityEasy)

	if result.IntervalDays != 3 {
		t.Errorf("IntervalDays = %d, want 3 (easy first review)", result.IntervalDays)
	}
}

func TestApplySM2Algorithm_QualityEasy_SubsequentReviews(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 3,
		ReviewCount:  1,
	}

	result := s.applySM2Algorithm(state, QualityEasy)

	// Easy: interval * ease_factor * 1.3, but EF is adjusted first
	// After Easy quality (q=3), EF increases: 2.5 + 0.1 = 2.6
	// Then: 3 * 2.6 * 1.3 = 10.14 -> 10
	expectedInterval := 10
	if result.IntervalDays != expectedInterval {
		t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, expectedInterval)
	}
}

func TestApplySM2Algorithm_MaxInterval(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   2.5,
		IntervalDays: 300,
		ReviewCount:  20,
	}

	result := s.applySM2Algorithm(state, QualityEasy)

	if result.IntervalDays > 365 {
		t.Errorf("IntervalDays = %d, should be capped at 365", result.IntervalDays)
	}
}

func TestApplySM2Algorithm_EaseFactorAdjustment(t *testing.T) {
	tests := []struct {
		name           string
		quality        ReviewQuality
		initialEF      float64
		expectDecrease bool
	}{
		{"Again decreases EF", QualityAgain, 2.5, true},
		{"Hard decreases EF", QualityHard, 2.5, true},
		{"Good slightly adjusts EF", QualityGood, 2.5, false},
		{"Easy increases EF", QualityEasy, 2.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{config: DefaultConfig()}
			state := &ReviewState{
				MemoUID:      "test-memo",
				EaseFactor:   tt.initialEF,
				IntervalDays: 3,
			}

			result := s.applySM2Algorithm(state, tt.quality)

			if tt.expectDecrease && result.EaseFactor >= tt.initialEF {
				t.Errorf("EaseFactor = %f, expected decrease from %f", result.EaseFactor, tt.initialEF)
			}
			if result.EaseFactor < MinEaseFactor {
				t.Errorf("EaseFactor = %f, should not go below MinEaseFactor %f", result.EaseFactor, MinEaseFactor)
			}
		})
	}
}

func TestApplySM2Algorithm_MinEaseFactorEnforced(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	state := &ReviewState{
		MemoUID:      "test-memo",
		EaseFactor:   1.4, // Close to minimum
		IntervalDays: 3,
	}

	// Multiple "Again" responses should not push EF below minimum
	for i := 0; i < 10; i++ {
		state = s.applySM2Algorithm(state, QualityAgain)
	}

	if state.EaseFactor < MinEaseFactor {
		t.Errorf("EaseFactor = %f, should not go below %f", state.EaseFactor, MinEaseFactor)
	}
}

func TestCalculatePriority_OverdueFactor(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	// Item overdue by 5 days
	item := ReviewItem{
		MemoID:      "test",
		NextReview:  now.AddDate(0, 0, -5),
		ReviewCount: 5,
		CreatedAt:   now.AddDate(0, -1, 0),
	}

	priority := s.calculatePriority(item, now)

	// 5 days overdue * 0.1 = 0.5
	if priority < 0.4 || priority > 0.6 {
		t.Errorf("Priority = %f, expected around 0.5 for 5 days overdue", priority)
	}
}

func TestCalculatePriority_OverdueMax(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	// Item overdue by 20 days (should cap at 1.0)
	item := ReviewItem{
		MemoID:      "test",
		NextReview:  now.AddDate(0, 0, -20),
		ReviewCount: 5,
		CreatedAt:   now.AddDate(0, -1, 0),
	}

	priority := s.calculatePriority(item, now)

	// Should cap at 1.0 for overdue factor
	if priority > 1.5 { // Allow some room for other factors
		t.Errorf("Priority = %f, overdue factor should cap at 1.0", priority)
	}
}

func TestCalculatePriority_ImportantTags(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	item := ReviewItem{
		MemoID:      "test",
		NextReview:  now.AddDate(0, 0, -1), // 1 day overdue
		Tags:        []string{"重要", "work"},
		ReviewCount: 5,
		CreatedAt:   now.AddDate(0, -1, 0),
	}

	priority := s.calculatePriority(item, now)

	// Should have importance bonus
	if priority < 0.5 {
		t.Errorf("Priority = %f, expected higher due to important tag", priority)
	}
}

func TestCalculatePriority_NewItemBonus(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	newItem := ReviewItem{
		MemoID:      "test",
		NextReview:  now.AddDate(0, 0, -1),
		ReviewCount: 0, // New item
		CreatedAt:   now.AddDate(0, -1, 0),
	}

	oldItem := ReviewItem{
		MemoID:      "test",
		NextReview:  now.AddDate(0, 0, -1),
		ReviewCount: 10, // Well-reviewed
		CreatedAt:   now.AddDate(0, -1, 0),
	}

	newPriority := s.calculatePriority(newItem, now)
	oldPriority := s.calculatePriority(oldItem, now)

	if newPriority <= oldPriority {
		t.Errorf("New item priority (%f) should be higher than old item (%f)", newPriority, oldPriority)
	}
}

func TestCalculatePriority_RecencyPenalty(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	recentItem := ReviewItem{
		MemoID:      "test",
		NextReview:  now,
		ReviewCount: 0,
		CreatedAt:   now.Add(-30 * time.Minute), // Created 30 minutes ago
	}

	oldItem := ReviewItem{
		MemoID:      "test",
		NextReview:  now,
		ReviewCount: 0,
		CreatedAt:   now.AddDate(0, 0, -1), // Created 1 day ago
	}

	recentPriority := s.calculatePriority(recentItem, now)
	oldPriority := s.calculatePriority(oldItem, now)

	if recentPriority >= oldPriority {
		t.Errorf("Recent item priority (%f) should be lower than old item (%f) due to recency penalty", recentPriority, oldPriority)
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"Simple title", "Hello World\nContent here", "Hello World"},
		{"Markdown heading", "# My Title\nContent", "My Title"},
		{"Multiple hashes", "### Deep Heading\nContent", "Deep Heading"},
		{"Empty content", "", ""},
		{"Long title truncated", "This is a very long title that should definitely be truncated because it exceeds fifty characters\nContent", "This is a very long title that should definitely b..."},
		{"CJK characters", "这是一个中文标题测试\n内容", "这是一个中文标题测试"},
		{"CJK long title", "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的中文标题超过五十个字\n内容", "这是一个非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常非常长的中文标题超过五十..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.content)
			if result != tt.expected {
				t.Errorf("extractTitle(%q) = %q, want %q", tt.content, result, tt.expected)
			}
		})
	}
}

func TestExtractSnippet(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLen   int
		expected string
	}{
		{"Skip title line", "Title\nThis is the content", 100, "This is the content"},
		{"Truncate long content", "Title\nThis is very long content", 10, "This is ve..."},
		{"Replace newlines", "Title\nLine1\nLine2\nLine3", 100, "Line1 Line2 Line3"},
		{"CJK content", "标题\n这是中文内容测试", 100, "这是中文内容测试"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSnippet(tt.content, tt.maxLen)
			if result != tt.expected {
				t.Errorf("extractSnippet(%q, %d) = %q, want %q", tt.content, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestReviewState_NextReviewCalculation(t *testing.T) {
	s := &Service{config: DefaultConfig()}
	now := time.Now()

	state := &ReviewState{
		MemoUID:      "test",
		EaseFactor:   2.5,
		IntervalDays: 0,
	}

	result := s.applySM2Algorithm(state, QualityGood)

	// Next review should be approximately 1 day from now
	expectedNext := now.AddDate(0, 0, 1)
	diff := result.NextReview.Sub(expectedNext)

	// Allow 1 second tolerance for test execution time
	if diff > time.Second || diff < -time.Second {
		t.Errorf("NextReview = %v, expected around %v", result.NextReview, expectedNext)
	}
}

func TestCalculateStreak(t *testing.T) {
	today := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		dates    map[string]bool
		name     string
		expected int
	}{
		{
			name:     "No reviews",
			dates:    map[string]bool{},
			expected: 0,
		},
		{
			name: "Today only",
			dates: map[string]bool{
				"2024-01-15": true,
			},
			expected: 1,
		},
		{
			name: "Yesterday only",
			dates: map[string]bool{
				"2024-01-14": true,
			},
			expected: 1,
		},
		{
			name: "Three day streak ending today",
			dates: map[string]bool{
				"2024-01-15": true,
				"2024-01-14": true,
				"2024-01-13": true,
			},
			expected: 3,
		},
		{
			name: "Broken streak",
			dates: map[string]bool{
				"2024-01-15": true,
				"2024-01-13": true, // Gap on 14th
			},
			expected: 1,
		},
		{
			name: "Old reviews only",
			dates: map[string]bool{
				"2024-01-10": true,
				"2024-01-09": true,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStreak(tt.dates, today)
			if result != tt.expected {
				t.Errorf("calculateStreak() = %d, want %d", result, tt.expected)
			}
		})
	}
}
