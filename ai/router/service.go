// Package router provides the LLM routing service.
package router

import (
	"context"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/ai/memory"
)

// Layer 3: LLM classification (~400ms) - fallback for remaining ~20%.
type Service struct {
	ruleMatcher    *RuleMatcher
	historyMatcher *HistoryMatcher
	llmClassifier  *LLMClassifier
	memoryService  memory.MemoryService
}

// Config contains the configuration for the router service.
type Config struct {
	MemoryService memory.MemoryService
	LLMClient     LLMClient
}

// NewService creates a new router service.
func NewService(cfg Config) *Service {
	return &Service{
		ruleMatcher:    NewRuleMatcher(),
		historyMatcher: NewHistoryMatcher(cfg.MemoryService),
		llmClassifier:  NewLLMClassifier(cfg.LLMClient),
		memoryService:  cfg.MemoryService,
	}
}

// Implementation: rule-based first (0ms) -> history match (~10ms) -> LLM fallback (~400ms).
func (s *Service) ClassifyIntent(ctx context.Context, input string) (Intent, float32, error) {
	start := time.Now()

	// Layer 1: Rule-based matching
	intent, confidence, matched := s.ruleMatcher.Match(input)
	if matched {
		slog.Debug("intent classified by rule matcher",
			"input", truncate(input, 50),
			"intent", intent,
			"confidence", confidence,
			"latency_ms", time.Since(start).Milliseconds())
		return intent, confidence, nil
	}

	// Layer 2: History matching (requires userID from context)
	userID := getUserIDFromContext(ctx)
	if userID > 0 && s.historyMatcher != nil {
		result, err := s.historyMatcher.Match(ctx, userID, input)
		if err != nil {
			// History matching errors are expected when no prior history exists
			// Use Debug level instead of Warn since this is normal operation
			slog.Debug("history matcher error", "error", err)
		} else if result.Matched {
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
	if s.llmClassifier != nil && s.llmClassifier.client != nil {
		result, err := s.llmClassifier.Classify(ctx, input)
		if err != nil {
			slog.Warn("LLM classifier error", "error", err)
			return IntentUnknown, 0, err
		}

		slog.Debug("intent classified by LLM",
			"input", truncate(input, 50),
			"intent", result.Intent,
			"confidence", result.Confidence,
			"reasoning", result.Reasoning,
			"latency_ms", time.Since(start).Milliseconds())

		// Save successful classification to history
		if userID > 0 && result.Intent != IntentUnknown && s.historyMatcher != nil {
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.historyMatcher.SaveDecision(bgCtx, userID, input, result.Intent, true); err != nil {
					slog.Warn("failed to save routing decision", "error", err)
				}
			}()
		}

		return result.Intent, result.Confidence, nil
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

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure Service implements RouterService.
var _ RouterService = (*Service)(nil)
