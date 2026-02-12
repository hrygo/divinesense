// Package routing provides the FastRouter service (cache -> rule).
package routing

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// Service implements FastRouter: cache -> rule.
// Complex/low-confidence requests are forwarded to Orchestrator.
type Service struct {
	ruleMatcher       *RuleMatcher
	historyMatcher    *HistoryMatcher
	cache             *RouterCache // Performance optimization: cache routing decisions
	feedbackCollector *FeedbackCollector
	weightStorage     RouterWeightStorage
}

// Config contains the configuration for the router service.
type Config struct {
	EnableCache    bool                // Enable routing result cache (default: true)
	WeightStorage  RouterWeightStorage // Storage for dynamic weights (optional)
	EnableFeedback bool                // Enable feedback-based weight adjustment (default: true)
}

// NewService creates a new router service.
func NewService(cfg Config) *Service {
	svc := &Service{
		ruleMatcher:    NewRuleMatcher(),
		historyMatcher: NewHistoryMatcher(nil), // No memory service
		weightStorage:  cfg.WeightStorage,
	}

	// Enable cache by default for performance
	if cfg.EnableCache {
		svc.cache = NewRouterCache(CacheConfig{
			Capacity:   500,
			DefaultTTL: 5 * time.Minute,
		})
	}

	// Initialize feedback collector if weight storage is provided
	if cfg.WeightStorage != nil && cfg.EnableFeedback {
		svc.feedbackCollector = NewFeedbackCollector(cfg.WeightStorage, svc.ruleMatcher)
	}

	return svc
}

// Implementation: FastRouter (cache -> rule).
// High confidence routes directly, low confidence/complex needs orchestration.
func (s *Service) ClassifyIntent(ctx context.Context, input string) (Intent, float32, bool, error) {
	start := time.Now()

	// Layer 0: Cache lookup (fastest path - ~0ms)
	if s.cache != nil {
		if intent, confidence, found := s.cache.Get(input); found {
			needsOrch := s.needsOrchestration(intent, confidence, input)
			slog.Debug("intent classified by cache",
				"input", truncate(input, 50),
				"intent", intent,
				"confidence", confidence,
				"needs_orchestration", needsOrch,
				"latency_ms", time.Since(start).Milliseconds())
			return intent, confidence, needsOrch, nil
		}
	}

	// Layer 1: Rule-based matching
	// Use MatchWithUser if userID is available to apply custom weights
	userID := getUserIDFromContext(ctx)
	var intent Intent
	var confidence float32
	var matched bool

	if userID > 0 {
		intent, confidence, matched = s.ruleMatcher.MatchWithUser(input, userID)
	} else {
		intent, confidence, matched = s.ruleMatcher.Match(input)
	}

	if matched {
		if s.cache != nil {
			s.cache.Set(input, intent, confidence, "rule")
		}
		// Async save to history (statistics only, no routing decision)
		s.saveToHistoryAsync(userID, input, intent)
		needsOrch := s.needsOrchestration(intent, confidence, input)
		slog.Debug("intent classified by rule matcher",
			"input", truncate(input, 50),
			"intent", intent,
			"confidence", confidence,
			"needs_orchestration", needsOrch,
			"latency_ms", time.Since(start).Milliseconds())
		return intent, confidence, needsOrch, nil
	}

	// Layer 2: No match → needs orchestration
	slog.Debug("no intent match found, needs orchestration",
		"input", truncate(input, 50),
		"latency_ms", time.Since(start).Milliseconds())
	return IntentUnknown, 0, true, nil
}

// needsOrchestration determines if the request needs Orchestrator handling.
// Threshold: 0.8 (stricter, more requests go to Orchestrator)
func (s *Service) needsOrchestration(intent Intent, confidence float32, input string) bool {
	// 1. Low confidence → needs Orchestrator
	if confidence < 0.8 {
		return true
	}

	// 2. Multi-intent keywords → needs Orchestrator
	multiIntentKeywords := []string{"顺便", "同时", "还有", "以及", "并且", "另外", "也"}
	for _, kw := range multiIntentKeywords {
		if strings.Contains(input, kw) {
			return true
		}
	}

	// 3. IntentUnknown → needs Orchestrator
	if intent == IntentUnknown {
		return true
	}

	return false
}

// saveToHistoryAsync saves routing decision to history (statistics only).
func (s *Service) saveToHistoryAsync(userID int32, input string, intent Intent) {
	if s.historyMatcher == nil || userID <= 0 || intent == IntentUnknown {
		return
	}
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.historyMatcher.SaveDecision(bgCtx, userID, input, intent, true); err != nil {
			slog.Debug("failed to save routing decision", "error", err)
		}
	}()
}

// Returns: model configuration (local/cloud).
func (s *Service) SelectModel(ctx context.Context, task TaskType) (ModelConfig, error) {
	// Model selection strategy based on task complexity
	switch task {
	case TaskIntentClassification:
		return ModelConfig{
			Provider:    "local",
			Model:       "qwen2.5-0.5b",
			MaxTokens:   256,
			Temperature: 0.1,
		}, nil
	case TaskEntityExtraction:
		return ModelConfig{
			Provider:    "local",
			Model:       "qwen2.5-1.5b",
			MaxTokens:   512,
			Temperature: 0.2,
		}, nil
	case TaskSimpleQA:
		return ModelConfig{
			Provider:    "local",
			Model:       "qwen2.5-3b",
			MaxTokens:   1024,
			Temperature: 0.3,
		}, nil
	case TaskComplexReasoning:
		return ModelConfig{
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   4096,
			Temperature: 0.5,
		}, nil
	case TaskSummarization:
		return ModelConfig{
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   2048,
			Temperature: 0.3,
		}, nil
	case TaskTagSuggestion:
		return ModelConfig{
			Provider:    "local",
			Model:       "qwen2.5-1.5b",
			MaxTokens:   256,
			Temperature: 0.4,
		}, nil
	default:
		return ModelConfig{
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   2048,
			Temperature: 0.5,
		}, nil
	}
}

// userIDContextKey is the context key for user ID.
type userIDContextKey struct{}

// WithUserID returns a context with user ID.
func WithUserID(ctx context.Context, userID int32) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// getUserIDFromContext extracts user ID from context.
func getUserIDFromContext(ctx context.Context) int32 {
	if v := ctx.Value(userIDContextKey{}); v != nil {
		if id, ok := v.(int32); ok {
			return id
		}
	}
	return 0
}

// Ensure Service implements RouterService.
var _ RouterService = (*Service)(nil)

// RecordFeedback records user feedback for a routing decision.
// This enables dynamic weight adjustment for improved routing accuracy.
func (s *Service) RecordFeedback(ctx context.Context, feedback *RouterFeedback) error {
	if s.feedbackCollector == nil {
		// Feedback collection not enabled, return without error
		return nil
	}

	// Set timestamp if not provided
	if feedback.Timestamp == 0 {
		feedback.Timestamp = time.Now().Unix()
	}

	// Record feedback and trigger weight adjustment
	return s.feedbackCollector.RecordFeedback(ctx, feedback)
}

// GetRouterStats retrieves routing accuracy statistics.
func (s *Service) GetRouterStats(ctx context.Context, userID int32, timeRange time.Duration) (*RouterStats, error) {
	if s.weightStorage == nil {
		// Return empty stats if weight storage is not configured
		return &RouterStats{
			ByIntent:    make(map[Intent]int64),
			BySource:    make(map[string]int64),
			LastUpdated: time.Now().Unix(),
		}, nil
	}

	return s.weightStorage.GetStats(ctx, userID, timeRange)
}
