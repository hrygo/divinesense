package router

import (
	"context"
	"testing"
	"time"
)

// TestInMemoryWeightStorage tests the in-memory weight storage implementation.
func TestInMemoryWeightStorage(t *testing.T) {
	storage := NewInMemoryWeightStorage()
	ctx := context.Background()
	userID := int32(123)

	t.Run("GetWeights returns nil for non-existent user", func(t *testing.T) {
		weights, err := storage.GetWeights(ctx, userID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if weights != nil {
			t.Fatalf("expected nil weights for non-existent user, got %v", weights)
		}
	})

	t.Run("SaveWeights and GetWeights", func(t *testing.T) {
		weights := map[string]map[string]int{
			"schedule": {"日程": 3, "会议": 1},
			"memo":     {"笔记": 2},
		}

		err := storage.SaveWeights(ctx, userID, weights)
		if err != nil {
			t.Fatalf("failed to save weights: %v", err)
		}

		retrieved, err := storage.GetWeights(ctx, userID)
		if err != nil {
			t.Fatalf("failed to get weights: %v", err)
		}

		if retrieved["schedule"]["日程"] != 3 {
			t.Errorf("expected schedule.日程 weight 3, got %d", retrieved["schedule"]["日程"])
		}
		if retrieved["schedule"]["会议"] != 1 {
			t.Errorf("expected schedule.会议 weight 1, got %d", retrieved["schedule"]["会议"])
		}
		if retrieved["memo"]["笔记"] != 2 {
			t.Errorf("expected memo.笔记 weight 2, got %d", retrieved["memo"]["笔记"])
		}
	})

	t.Run("RecordFeedback and GetStats", func(t *testing.T) {
		now := time.Now().Unix()

		// Record positive feedback
		err := storage.RecordFeedback(ctx, &RouterFeedback{
			UserID:    userID,
			Input:     "明天有什么会议",
			Predicted: IntentScheduleQuery,
			Actual:    IntentScheduleQuery,
			Feedback:  FeedbackPositive,
			Timestamp: now,
			Source:    "rule",
		})
		if err != nil {
			t.Fatalf("failed to record positive feedback: %v", err)
		}

		// Record negative feedback
		err = storage.RecordFeedback(ctx, &RouterFeedback{
			UserID:    userID,
			Input:     "搜索笔记",
			Predicted: IntentScheduleQuery,
			Actual:    IntentMemoSearch,
			Feedback:  FeedbackSwitch,
			Timestamp: now,
			Source:    "rule",
		})
		if err != nil {
			t.Fatalf("failed to record switch feedback: %v", err)
		}

		// Get stats
		stats, err := storage.GetStats(ctx, userID, 24*time.Hour)
		if err != nil {
			t.Fatalf("failed to get stats: %v", err)
		}

		if stats.TotalPredictions != 2 {
			t.Errorf("expected 2 total predictions, got %d", stats.TotalPredictions)
		}
		if stats.CorrectCount != 1 {
			t.Errorf("expected 1 correct count, got %d", stats.CorrectCount)
		}
		if stats.IncorrectCount != 1 {
			t.Errorf("expected 1 incorrect count, got %d", stats.IncorrectCount)
		}
		if stats.Accuracy != 0.5 {
			t.Errorf("expected 0.5 accuracy, got %f", stats.Accuracy)
		}
	})
}

// TestFeedbackCollector tests the feedback collector.
func TestFeedbackCollector(t *testing.T) {
	storage := NewInMemoryWeightStorage()
	baseMatcher := NewRuleMatcher()
	collector := NewFeedbackCollector(storage, baseMatcher)
	ctx := context.Background()
	userID := int32(123)

	t.Run("RecordFeedback for positive reinforcement", func(t *testing.T) {
		err := collector.RecordFeedback(ctx, &RouterFeedback{
			UserID:    userID,
			Input:     "明天会议",
			Predicted: IntentScheduleQuery,
			Actual:    IntentScheduleQuery,
			Feedback:  FeedbackPositive,
			Timestamp: time.Now().Unix(),
			Source:    "rule",
		})
		if err != nil {
			t.Fatalf("failed to record positive feedback: %v", err)
		}

		// Check that weights were updated
		weights, err := storage.GetWeights(ctx, userID)
		if err != nil {
			t.Fatalf("failed to get weights: %v", err)
		}

		if weights == nil {
			t.Fatal("expected weights to be updated, got nil")
		}

		// The keyword "会议" should have its weight increased
		if w, ok := weights["schedule"]["会议"]; ok {
			if w <= 2 {
				t.Errorf("expected keyword weight > 2, got %d", w)
			}
		}
	})

	t.Run("RecordFeedback for switch (negative adjustment)", func(t *testing.T) {
		err := collector.RecordFeedback(ctx, &RouterFeedback{
			UserID:    userID,
			Input:     "搜索笔记",
			Predicted: IntentScheduleQuery, // Wrong prediction
			Actual:    IntentMemoSearch,    // User wanted memo
			Feedback:  FeedbackSwitch,
			Timestamp: time.Now().Unix(),
			Source:    "rule",
		})
		if err != nil {
			t.Fatalf("failed to record switch feedback: %v", err)
		}

		// The keyword "搜索" in memo category should be reinforced
		weights, err := storage.GetWeights(ctx, userID)
		if err != nil {
			t.Fatalf("failed to get weights: %v", err)
		}

		// Check memo category weights
		if w, ok := weights["memo"]["搜索"]; ok {
			if w < 3 {
				t.Errorf("expected memo.搜索 weight >= 3, got %d", w)
			}
		}
	})
}

// TestRuleMatcherWithCustomWeights tests the rule matcher with custom weights.
func TestRuleMatcherWithCustomWeights(t *testing.T) {
	matcher := NewRuleMatcher()
	userID := int32(123)

	t.Run("SetCustomWeights and GetCustomWeights", func(t *testing.T) {
		weights := map[string]map[string]int{
			"schedule": {"日程": 5, "会议": 1}, // Lower weight for "会议"
			"memo":     {"笔记": 5},          // Higher weight for "笔记"
		}

		matcher.SetCustomWeights(userID, weights)

		retrieved := matcher.GetCustomWeights(userID)
		if retrieved == nil {
			t.Fatal("expected custom weights, got nil")
		}

		if retrieved["schedule"]["日程"] != 5 {
			t.Errorf("expected schedule.日程 weight 5, got %d", retrieved["schedule"]["日程"])
		}
	})

	t.Run("MatchWithUser uses custom weights", func(t *testing.T) {
		// Set custom weights to bias toward memo
		weights := map[string]map[string]int{
			"memo": {
				"笔记": 5,
				"搜索": 5,
			},
			"schedule": {
				"日程": 1,
				"会议": 1,
			},
		}
		matcher.SetCustomWeights(userID, weights)

		// Input that could match both, but custom weights favor memo
		input := "会议笔记"

		intent, _, matched := matcher.MatchWithUser(input, userID)
		if !matched {
			t.Fatal("expected match, got no match")
		}

		if intent != IntentMemoSearch {
			t.Errorf("expected IntentMemoSearch with custom weights, got %v", intent)
		}
	})

	t.Run("MatchWithUser falls back to default for no custom weights", func(t *testing.T) {
		unknownUserID := int32(999)
		input := "明天有什么会议"

		intent, _, matched := matcher.MatchWithUser(input, unknownUserID)
		if !matched {
			t.Fatal("expected match, got no match")
		}

		if intent != IntentScheduleQuery {
			t.Errorf("expected IntentScheduleQuery with default weights, got %v", intent)
		}
	})
}

// TestGetKeywordsForCategory tests the getKeywordsForCategory method.
func TestGetKeywordsForCategory(t *testing.T) {
	matcher := NewRuleMatcher()

	t.Run("schedule keywords", func(t *testing.T) {
		keywords := matcher.getKeywordsForCategory("schedule")
		if keywords == nil {
			t.Fatal("expected schedule keywords, got nil")
		}

		// Check for expected keywords
		found := false
		for _, kw := range keywords {
			if kw == "日程" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected '日程' in schedule keywords")
		}
	})

	t.Run("memo keywords", func(t *testing.T) {
		keywords := matcher.getKeywordsForCategory("memo")
		if keywords == nil {
			t.Fatal("expected memo keywords, got nil")
		}

		// Check for expected keywords
		found := false
		for _, kw := range keywords {
			if kw == "笔记" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected '笔记' in memo keywords")
		}
	})

	t.Run("amazing keywords", func(t *testing.T) {
		keywords := matcher.getKeywordsForCategory("amazing")
		if keywords == nil {
			t.Fatal("expected amazing keywords, got nil")
		}

		// Check for expected keywords
		found := false
		for _, kw := range keywords {
			if kw == "帮我" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected '帮我' in amazing keywords")
		}
	})

	t.Run("unknown category returns nil", func(t *testing.T) {
		keywords := matcher.getKeywordsForCategory("unknown")
		if keywords != nil {
			t.Error("expected nil for unknown category, got keywords")
		}
	})
}

// TestCalculateWeightAdjustments tests the weight adjustment calculation.
func TestCalculateWeightAdjustments(t *testing.T) {
	storage := NewInMemoryWeightStorage()
	baseMatcher := NewRuleMatcher()
	collector := NewFeedbackCollector(storage, baseMatcher)

	t.Run("switch feedback decreases predicted category weight", func(t *testing.T) {
		currentWeights := map[string]map[string]int{
			"schedule": {"会议": 2, "日程": 2},
			"memo":     {"笔记": 2},
		}

		input := "会议笔记"
		predicted := IntentScheduleQuery
		actual := IntentMemoSearch

		adjustments := collector.calculateWeightAdjustments(input, predicted, actual, currentWeights, -2)

		// Should have at least one adjustment (decrease "会议" weight)
		if len(adjustments) == 0 {
			t.Fatal("expected weight adjustments, got none")
		}

		// Check that "会议" weight was decreased
		found := false
		for _, adj := range adjustments {
			if adj.Keyword == "会议" && adj.Category == "schedule" {
				found = true
				if adj.Adjustment >= 0 {
					t.Errorf("expected negative adjustment for '会议', got %d", adj.Adjustment)
				}
			}
		}
		if !found {
			t.Error("expected adjustment for '会议' keyword")
		}
	})
}

// TestFeedbackType tests feedback type constants.
func TestFeedbackType(t *testing.T) {
	if FeedbackPositive != "positive" {
		t.Errorf("expected FeedbackPositive = 'positive', got %s", FeedbackPositive)
	}
	if FeedbackRephrase != "rephrase" {
		t.Errorf("expected FeedbackRephrase = 'rephrase', got %s", FeedbackRephrase)
	}
	if FeedbackSwitch != "switch" {
		t.Errorf("expected FeedbackSwitch = 'switch', got %s", FeedbackSwitch)
	}
}
