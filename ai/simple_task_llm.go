package ai

import (
	"log/slog"

	"github.com/hrygo/divinesense/internal/profile"
)

// Configuration defaults for simple LLM tasks.
const (
	SimpleTaskMaxTokens   = 1024 // Simple tasks don't need many tokens
	SimpleTaskTemperature = 0.3  // Lower temperature for deterministic output
	SimpleTaskTimeout     = 30   // Shorter timeout for simple tasks (seconds)
)

// NewSimpleTaskLLMService creates an LLM service for simple tasks.
// It uses the Intent provider configuration with fallback to main LLM.
//
// Priority:
// 1. If AIIntentAPIKey is configured, use Intent provider (siliconflow by default)
// 2. Otherwise, fallback to main LLM service
//
// Returns nil if both Intent service creation fails and mainLLM is nil.
// Callers must check for nil return value.
func NewSimpleTaskLLMService(p *profile.Profile, mainLLM LLMService) LLMService {
	// Guard against nil profile
	if p == nil {
		slog.Warn("Profile is nil, returning main LLM service for simple tasks")
		return mainLLM
	}

	// If Intent API key is configured, create dedicated service
	if p.AIIntentAPIKey != "" {
		cfg := &LLMConfig{
			Provider:    p.AIIntentProvider,
			Model:       p.AIIntentModel,
			APIKey:      p.AIIntentAPIKey,
			BaseURL:     p.AIIntentBaseURL,
			MaxTokens:   SimpleTaskMaxTokens,
			Temperature: SimpleTaskTemperature,
			Timeout:     SimpleTaskTimeout,
		}

		svc, err := NewLLMService(cfg)
		if err != nil {
			slog.Warn("Failed to create simple task LLM service, falling back to main LLM",
				"provider", cfg.Provider,
				"model", cfg.Model,
				"error", err,
			)
			return mainLLM
		}

		slog.Info("Simple task LLM service initialized",
			"provider", cfg.Provider,
			"model", cfg.Model,
		)
		return svc
	}

	// Fallback to main LLM service
	slog.Info("Using main LLM service for simple tasks (no Intent API key configured)",
		"provider", p.ALLMProvider,
		"model", p.ALLMModel,
	)
	return mainLLM
}
