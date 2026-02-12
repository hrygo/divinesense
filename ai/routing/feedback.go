// Package routing provides feedback-based dynamic weight adjustment for routing.
package routing

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// FeedbackType represents the type of feedback for a routing decision.
type FeedbackType string

const (
	// FeedbackPositive indicates user did not correct the routing (implicit confirmation).
	FeedbackPositive FeedbackType = "positive"
	// FeedbackRephrase indicates user rephrased their query (possible routing error).
	FeedbackRephrase FeedbackType = "rephrase"
	// FeedbackSwitch indicates user explicitly switched agents (routing error).
	FeedbackSwitch FeedbackType = "switch"
)

// RouterFeedback represents a single feedback event for a routing decision.
type RouterFeedback struct {
	ID        int64        `json:"id"`
	UserID    int32        `json:"user_id"`
	Input     string       `json:"input"`
	Predicted Intent       `json:"predicted"` // What the router predicted
	Actual    Intent       `json:"actual"`    // What the user actually wanted
	Feedback  FeedbackType `json:"feedback"`
	Timestamp int64        `json:"timestamp"`
	Source    string       `json:"source"` // "rule", "history", "llm"
}

// RouterStats represents routing accuracy statistics.
type RouterStats struct {
	TotalPredictions int64            `json:"total_predictions"`
	CorrectCount     int64            `json:"correct_count"`
	IncorrectCount   int64            `json:"incorrect_count"`
	Accuracy         float64          `json:"accuracy"`
	ByIntent         map[Intent]int64 `json:"by_intent"`
	BySource         map[string]int64 `json:"by_source"`
	LastUpdated      int64            `json:"last_updated"`
}

// WeightAdjustment represents the adjustment delta for a keyword.
type WeightAdjustment struct {
	Keyword    string `json:"keyword"`
	Category   string `json:"category"` // "schedule", "memo", "amazing"
	OldWeight  int    `json:"old_weight"`
	NewWeight  int    `json:"new_weight"`
	Adjustment int    `json:"adjustment"`
	Reason     string `json:"reason"`
}

// RouterWeightStorage defines the interface for storing router weights and feedback.
type RouterWeightStorage interface {
	// GetWeights retrieves custom weights for a user.
	GetWeights(ctx context.Context, userID int32) (map[string]map[string]int, error)

	// SaveWeights saves custom weights for a user.
	SaveWeights(ctx context.Context, userID int32, weights map[string]map[string]int) error

	// RecordFeedback records a feedback event.
	RecordFeedback(ctx context.Context, feedback *RouterFeedback) error

	// GetStats retrieves routing statistics for a user.
	GetStats(ctx context.Context, userID int32, timeRange time.Duration) (*RouterStats, error)
}

// InMemoryWeightStorage provides an in-memory implementation of RouterWeightStorage.
// Used for testing and as a fallback when database is not available.
type InMemoryWeightStorage struct {
	mu       sync.RWMutex
	weights  map[int32]map[string]map[string]int // userID -> category -> keyword -> weight
	feedback map[int32][]*RouterFeedback
}

// NewInMemoryWeightStorage creates a new in-memory weight storage.
func NewInMemoryWeightStorage() *InMemoryWeightStorage {
	return &InMemoryWeightStorage{
		weights:  make(map[int32]map[string]map[string]int),
		feedback: make(map[int32][]*RouterFeedback),
	}
}

// GetWeights retrieves weights from memory.
func (s *InMemoryWeightStorage) GetWeights(ctx context.Context, userID int32) (map[string]map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if w, ok := s.weights[userID]; ok {
		// Return a copy to avoid concurrent modification
		result := make(map[string]map[string]int, len(w))
		for cat, kw := range w {
			result[cat] = make(map[string]int, len(kw))
			for k, v := range kw {
				result[cat][k] = v
			}
		}
		return result, nil
	}
	return nil, nil
}

// SaveWeights saves weights to memory.
func (s *InMemoryWeightStorage) SaveWeights(ctx context.Context, userID int32, weights map[string]map[string]int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.weights[userID] == nil {
		s.weights[userID] = make(map[string]map[string]int)
	}

	for cat, kw := range weights {
		if s.weights[userID][cat] == nil {
			s.weights[userID][cat] = make(map[string]int)
		}
		for k, v := range kw {
			s.weights[userID][cat][k] = v
		}
	}
	return nil
}

// RecordFeedback records a feedback event in memory.
func (s *InMemoryWeightStorage) RecordFeedback(ctx context.Context, feedback *RouterFeedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.feedback[feedback.UserID] == nil {
		s.feedback[feedback.UserID] = make([]*RouterFeedback, 0, 100)
	}
	s.feedback[feedback.UserID] = append(s.feedback[feedback.UserID], feedback)
	return nil
}

// GetStats retrieves statistics from memory.
func (s *InMemoryWeightStorage) GetStats(ctx context.Context, userID int32, timeRange time.Duration) (*RouterStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	feedbacks, ok := s.feedback[userID]
	if !ok || len(feedbacks) == 0 {
		return &RouterStats{
			ByIntent:    make(map[Intent]int64),
			BySource:    make(map[string]int64),
			LastUpdated: time.Now().Unix(),
		}, nil
	}

	cutoff := time.Now().Add(-timeRange).Unix()

	stats := &RouterStats{
		ByIntent:    make(map[Intent]int64),
		BySource:    make(map[string]int64),
		LastUpdated: time.Now().Unix(),
	}

	for _, fb := range feedbacks {
		if fb.Timestamp < cutoff {
			continue
		}

		stats.TotalPredictions++
		stats.ByIntent[fb.Predicted]++
		stats.BySource[fb.Source]++

		if fb.Feedback == FeedbackPositive {
			stats.CorrectCount++
		} else {
			stats.IncorrectCount++
		}
	}

	if stats.TotalPredictions > 0 {
		stats.Accuracy = float64(stats.CorrectCount) / float64(stats.TotalPredictions)
	}

	return stats, nil
}

// FeedbackCollector collects and processes feedback for weight adjustment.
type FeedbackCollector struct {
	storage     RouterWeightStorage
	baseMatcher *RuleMatcher
	adjustments chan *WeightAdjustment
}

// NewFeedbackCollector creates a new feedback collector.
func NewFeedbackCollector(storage RouterWeightStorage, baseMatcher *RuleMatcher) *FeedbackCollector {
	fc := &FeedbackCollector{
		storage:     storage,
		baseMatcher: baseMatcher,
		adjustments: make(chan *WeightAdjustment, 100),
	}
	go fc.processAdjustments()
	return fc
}

// RecordFeedback records a feedback event and triggers weight adjustment.
func (fc *FeedbackCollector) RecordFeedback(ctx context.Context, feedback *RouterFeedback) error {
	slog.Debug("recording router feedback",
		"user_id", feedback.UserID,
		"predicted", feedback.Predicted,
		"actual", feedback.Actual,
		"feedback", feedback.Feedback)

	// Persist feedback
	if err := fc.storage.RecordFeedback(ctx, feedback); err != nil {
		slog.Warn("failed to persist router feedback", "error", err)
		// Continue with weight adjustment even if persistence fails
	}

	// Trigger weight adjustment based on feedback type
	switch feedback.Feedback {
	case FeedbackSwitch:
		// Significant weight adjustment
		fc.adjustWeightsForSwitch(ctx, feedback)
	case FeedbackRephrase:
		// Small negative adjustment
		fc.adjustWeightsForRephrase(ctx, feedback)
	case FeedbackPositive:
		// Positive reinforcement - small positive adjustment
		fc.adjustWeightsForPositive(ctx, feedback)
	}

	return nil
}

// adjustWeightsForSwitch adjusts weights when user explicitly switches agents.
func (fc *FeedbackCollector) adjustWeightsForSwitch(ctx context.Context, feedback *RouterFeedback) {
	// Get current weights
	currentWeights, err := fc.storage.GetWeights(ctx, feedback.UserID)
	if err != nil {
		slog.Warn("failed to get weights for adjustment", "error", err)
		return
	}

	if currentWeights == nil {
		currentWeights = make(map[string]map[string]int)
	}

	// Identify keywords in the input
	input := feedback.Input
	adjustments := fc.calculateWeightAdjustments(input, feedback.Predicted, feedback.Actual, currentWeights, -2)

	// Apply adjustments
	for _, adj := range adjustments {
		fc.applyAdjustment(ctx, feedback.UserID, adj)
	}
}

// adjustWeightsForRephrase adjusts weights when user rephrases their query.
func (fc *FeedbackCollector) adjustWeightsForRephrase(ctx context.Context, feedback *RouterFeedback) {
	currentWeights, err := fc.storage.GetWeights(ctx, feedback.UserID)
	if err != nil {
		slog.Warn("failed to get weights for adjustment", "error", err)
		return
	}

	if currentWeights == nil {
		currentWeights = make(map[string]map[string]int)
	}

	// Smaller adjustment for rephrase (user might just be clarifying)
	input := feedback.Input
	adjustments := fc.calculateWeightAdjustments(input, feedback.Predicted, feedback.Actual, currentWeights, -1)

	for _, adj := range adjustments {
		fc.applyAdjustment(ctx, feedback.UserID, adj)
	}
}

// adjustWeightsForPositive reinforces correct routing with small positive adjustment.
func (fc *FeedbackCollector) adjustWeightsForPositive(ctx context.Context, feedback *RouterFeedback) {
	// Only reinforce if prediction was correct (predicted == actual)
	if feedback.Predicted != feedback.Actual {
		return
	}

	currentWeights, err := fc.storage.GetWeights(ctx, feedback.UserID)
	if err != nil {
		return
	}

	if currentWeights == nil {
		currentWeights = make(map[string]map[string]int)
	}

	// Small positive reinforcement
	input := feedback.Input
	categories := []string{"schedule", "memo", "amazing"}

	for _, category := range categories {
		// Only reinforce the correct category
		expectedCategory := fc.intentToCategory(feedback.Actual)
		if category != expectedCategory {
			continue
		}

		keywords := fc.baseMatcher.getKeywordsForCategory(category)
		for _, keyword := range keywords {
			if contains(input, keyword) {
				if currentWeights[category] == nil {
					currentWeights[category] = make(map[string]int)
				}
				// Get current weight or default weight, then increment
				currentWeight := currentWeights[category][keyword]
				if currentWeight == 0 {
					// Use default weight from base matcher + 1
					currentWeight = fc.baseMatcher.GetKeywordWeight(feedback.UserID, category, keyword)
				}
				currentWeights[category][keyword] = currentWeight + 1
			}
		}
	}

	// Save updated weights
	if err := fc.storage.SaveWeights(ctx, feedback.UserID, currentWeights); err != nil {
		slog.Warn("failed to save adjusted weights", "error", err)
	}
}

// calculateWeightAdjustments calculates weight adjustments based on feedback.
func (fc *FeedbackCollector) calculateWeightAdjustments(input string, predicted, actual Intent, currentWeights map[string]map[string]int, delta int) []*WeightAdjustment {
	var adjustments []*WeightAdjustment

	predictedCategory := fc.intentToCategory(predicted)
	actualCategory := fc.intentToCategory(actual)
	userID := int32(0) // We don't have userID here, use 0 for default weights

	// Decrease weights for keywords in the predicted (wrong) category
	keywords := fc.baseMatcher.getKeywordsForCategory(predictedCategory)
	for _, keyword := range keywords {
		if !contains(input, keyword) {
			continue
		}
		if currentWeights[predictedCategory] == nil {
			currentWeights[predictedCategory] = make(map[string]int)
		}
		oldWeight := currentWeights[predictedCategory][keyword]
		// If no custom weight exists, start from default weight
		if oldWeight == 0 {
			oldWeight = fc.baseMatcher.GetKeywordWeight(userID, predictedCategory, keyword)
		}
		newWeight := oldWeight + delta

		// Don't let weights go below 1
		if newWeight < 1 {
			newWeight = 1
		}

		adjustments = append(adjustments, &WeightAdjustment{
			Keyword:    keyword,
			Category:   predictedCategory,
			OldWeight:  oldWeight,
			NewWeight:  newWeight,
			Adjustment: delta,
			Reason:     "feedback: " + actualCategory + " != " + predictedCategory,
		})

		currentWeights[predictedCategory][keyword] = newWeight
	}

	// Increase weights for keywords in the actual (correct) category
	if actualCategory != predictedCategory {
		keywords = fc.baseMatcher.getKeywordsForCategory(actualCategory)
		positiveDelta := -delta // Flip sign

		for _, keyword := range keywords {
			if !contains(input, keyword) {
				continue
			}
			if currentWeights[actualCategory] == nil {
				currentWeights[actualCategory] = make(map[string]int)
			}
			oldWeight := currentWeights[actualCategory][keyword]
			// If no custom weight exists, start from default weight
			if oldWeight == 0 {
				oldWeight = fc.baseMatcher.GetKeywordWeight(userID, actualCategory, keyword)
			}
			newWeight := oldWeight + positiveDelta

			// Cap max weight at 5
			if newWeight > 5 {
				newWeight = 5
			}

			adjustments = append(adjustments, &WeightAdjustment{
				Keyword:    keyword,
				Category:   actualCategory,
				OldWeight:  oldWeight,
				NewWeight:  newWeight,
				Adjustment: positiveDelta,
				Reason:     "feedback: correct is " + actualCategory,
			})

			currentWeights[actualCategory][keyword] = newWeight
		}
	}

	return adjustments
}

// applyAdjustment applies a single weight adjustment.
func (fc *FeedbackCollector) applyAdjustment(ctx context.Context, userID int32, adj *WeightAdjustment) {
	slog.Debug("applying weight adjustment",
		"user_id", userID,
		"keyword", adj.Keyword,
		"category", adj.Category,
		"old", adj.OldWeight,
		"new", adj.NewWeight,
		"reason", adj.Reason)

	currentWeights, err := fc.storage.GetWeights(ctx, userID)
	if err != nil {
		slog.Warn("failed to get weights before applying adjustment", "error", err)
		return
	}

	if currentWeights == nil {
		currentWeights = make(map[string]map[string]int)
	}

	if currentWeights[adj.Category] == nil {
		currentWeights[adj.Category] = make(map[string]int)
	}

	currentWeights[adj.Category][adj.Keyword] = adj.NewWeight

	if err := fc.storage.SaveWeights(ctx, userID, currentWeights); err != nil {
		slog.Warn("failed to save adjusted weights", "error", err)
	}
}

// processAdjustments processes weight adjustments in the background.
func (fc *FeedbackCollector) processAdjustments() {
	for adj := range fc.adjustments {
		slog.Debug("processing weight adjustment", "keyword", adj.Keyword, "adjustment", adj.Adjustment)
	}
}

// intentToCategory converts Intent to category string.
func (fc *FeedbackCollector) intentToCategory(intent Intent) string {
	switch intent {
	case IntentScheduleQuery, IntentScheduleCreate, IntentScheduleUpdate, IntentBatchSchedule:
		return "schedule"
	case IntentMemoSearch, IntentMemoCreate:
		return "memo"
	default:
		return "unknown"
	}
}

// contains checks if substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}
