// Package router provides the LLM routing service.
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/memory"
)

// LLMClient defines the interface for LLM API calls.
// Implemented by server layer to connect to SiliconFlow + Qwen.
type LLMClient interface {
	// Complete sends a completion request and returns the intent as a string.
	Complete(ctx context.Context, prompt string, config ModelConfig) (string, error)
}

// Service implements three-layer routing: cache -> rule -> history -> LLM.
type Service struct {
	ruleMatcher       *RuleMatcher
	historyMatcher    *HistoryMatcher
	llmClient         LLMClient // Direct LLM client (no wrapper layer)
	memoryService     memory.MemoryService
	cache             *RouterCache // Performance optimization: cache routing decisions
	feedbackCollector *FeedbackCollector
	weightStorage     RouterWeightStorage
}

// Config contains the configuration for the router service.
type Config struct {
	MemoryService  memory.MemoryService
	LLMClient      LLMClient
	EnableCache    bool                // Enable routing result cache (default: true)
	WeightStorage  RouterWeightStorage // Storage for dynamic weights (optional)
	EnableFeedback bool                // Enable feedback-based weight adjustment (default: true)
}

// NewService creates a new router service.
func NewService(cfg Config) *Service {
	svc := &Service{
		ruleMatcher:    NewRuleMatcher(),
		historyMatcher: NewHistoryMatcher(cfg.MemoryService),
		llmClient:      cfg.LLMClient,
		memoryService:  cfg.MemoryService,
		weightStorage:  cfg.WeightStorage,
	}

	// Enable cache by default for performance
	if cfg.EnableCache {
		svc.cache = NewRouterCache(CacheConfig{
			Capacity:     500,
			DefaultTTL:   5 * time.Minute,
			LLMResultTTL: 30 * time.Minute,
		})
	}

	// Initialize feedback collector if weight storage is provided
	if cfg.WeightStorage != nil && cfg.EnableFeedback {
		svc.feedbackCollector = NewFeedbackCollector(cfg.WeightStorage, svc.ruleMatcher)
	}

	return svc
}

// Implementation: cache -> rule-based first (0ms) -> history match (~10ms) -> LLM fallback (~400ms).
func (s *Service) ClassifyIntent(ctx context.Context, input string) (Intent, float32, error) {
	start := time.Now()

	// Layer 0: Cache lookup (fastest path - ~0ms)
	if s.cache != nil {
		if intent, confidence, found := s.cache.Get(input); found {
			slog.Debug("intent classified by cache",
				"input", truncate(input, 50),
				"intent", intent,
				"confidence", confidence,
				"latency_ms", time.Since(start).Milliseconds())
			return intent, confidence, nil
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
		slog.Debug("intent classified by rule matcher",
			"input", truncate(input, 50),
			"intent", intent,
			"confidence", confidence,
			"latency_ms", time.Since(start).Milliseconds())
		return intent, confidence, nil
	}

	// Layer 2: History matching (requires userID from context)
	if userID > 0 && s.historyMatcher != nil {
		result, err := s.historyMatcher.Match(ctx, userID, input)
		if err != nil {
			// History matching errors are expected when no prior history exists
			// Use Debug level instead of Warn since this is normal operation
			slog.Debug("history matcher error", "error", err)
		} else if result.Matched {
			if s.cache != nil {
				s.cache.Set(input, result.Intent, result.Confidence, "history")
			}
			slog.Debug("intent classified by history matcher",
				"input", truncate(input, 50),
				"intent", result.Intent,
				"confidence", result.Confidence,
				"source_id", result.SourceID,
				"latency_ms", time.Since(start).Milliseconds())
			return result.Intent, result.Confidence, nil
		}
	}

	// Layer 3: LLM classification (fallback)
	if s.llmClient != nil {
		intent, confidence, err := s.llmClassify(ctx, input)
		if err != nil {
			slog.Warn("LLM classifier error", "error", err)
			return IntentUnknown, 0, err
		}

		// Cache LLM results with longer TTL (expensive computation)
		if s.cache != nil && intent != IntentUnknown {
			s.cache.Set(input, intent, confidence, "llm")
		}

		slog.Debug("intent classified by LLM",
			"input", truncate(input, 50),
			"intent", intent,
			"confidence", confidence,
			"latency_ms", time.Since(start).Milliseconds())

		// Save successful classification to history
		if userID > 0 && intent != IntentUnknown && s.historyMatcher != nil {
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.historyMatcher.SaveDecision(bgCtx, userID, input, intent, true); err != nil {
					slog.Warn("failed to save routing decision", "error", err)
				}
			}()
		}

		return intent, confidence, nil
	}

	// No match found
	slog.Debug("no intent match found",
		"input", truncate(input, 50),
		"latency_ms", time.Since(start).Milliseconds())
	return IntentUnknown, 0, nil
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

// GetCacheStats returns cache statistics if cache is enabled.
func (s *Service) GetCacheStats() *Stats {
	if s.cache == nil {
		return nil
	}
	stats := s.cache.GetStats()
	return &stats
}

// ClearCache clears the routing cache.
func (s *Service) ClearCache() {
	if s.cache != nil {
		s.cache.Clear()
	}
}

// Ensure Service implements RouterService.
var _ RouterService = (*Service)(nil)

// llmClassify performs LLM-based intent classification.
// Expects LLMClient to return JSON with intent and confidence.
func (s *Service) llmClassify(ctx context.Context, input string) (Intent, float32, error) {
	// Build prompt for intent classification
	prompt := fmt.Sprintf("用户输入: %s", input)

	// Call LLM via client (server layer provides SiliconFlow + Qwen implementation)
	// Model configuration is handled by the LLMClient implementation
	config := ModelConfig{
		Provider:    "siliconflow",
		MaxTokens:   50,
		Temperature: 0,
	}

	response, err := s.llmClient.Complete(ctx, prompt, config)
	if err != nil {
		return IntentUnknown, 0, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse response - extract intent and confidence from JSON
	intent, confidence := s.parseLLMResponse(response)

	return intent, confidence, nil
}

// llmJSONResponse is the expected JSON structure from LLM.
type llmJSONResponse struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
}

// parseLLMResponse parses LLM JSON response and extracts intent with confidence.
func (s *Service) parseLLMResponse(response string) (Intent, float32) {
	response = strings.TrimSpace(response)

	// Try JSON format first
	var jsonResp llmJSONResponse
	if err := json.Unmarshal([]byte(response), &jsonResp); err == nil {
		confidence := float32(jsonResp.Confidence)
		if confidence <= 0 {
			confidence = 0.8 // Default if LLM returns invalid confidence
		}
		return s.stringToIntent(jsonResp.Intent), confidence
	}

	// Fallback: plain text matching with default confidence
	return s.stringToIntent(response), 0.8
}

// stringToIntent converts string to Intent enum.
func (s *Service) stringToIntent(str string) Intent {
	str = strings.ToLower(strings.TrimSpace(str))

	// Remove common prefixes/quotes
	str = strings.TrimPrefix(str, "\"")
	str = strings.TrimSuffix(str, "\"")
	str = strings.Trim(str, "`'")

	switch str {
	case "memo_search", "memosearch", "search":
		return IntentMemoSearch
	case "memo_create", "memocreate", "create_memo":
		return IntentMemoCreate
	case "schedule_query", "schedulequery", "query":
		return IntentScheduleQuery
	case "schedule_create", "schedulecreate", "create_schedule":
		return IntentScheduleCreate
	case "schedule_update", "scheduleupdate", "update":
		return IntentScheduleUpdate
	case "batch_schedule", "batchschedule", "batch":
		return IntentBatchSchedule
	case "amazing":
		return IntentAmazing
	default:
		// Default to amazing for ambiguous inputs
		return IntentAmazing
	}
}

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

// LoadUserWeights loads custom weights for a user into the rule matcher.
// This should be called when a user session starts.
func (s *Service) LoadUserWeights(ctx context.Context, userID int32) error {
	if s.weightStorage == nil {
		return nil
	}

	weights, err := s.weightStorage.GetWeights(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user weights: %w", err)
	}

	if len(weights) > 0 {
		s.ruleMatcher.SetCustomWeights(userID, weights)
		slog.Debug("loaded custom weights for user", "user_id", userID, "categories", len(weights))
	}

	return nil
}
