package ai

import (
	"log/slog"

	"github.com/hrygo/divinesense/internal/profile"
)

// Configuration defaults for simple LLM tasks.
const (
	SimpleTaskMaxTokens   = 1024 // Simple tasks don't need many tokens
	SimpleTaskTemperature = 0.3  // Lower temperature for deterministic output
)

// NewSimpleTaskLLMService creates an LLM service for simple tasks.
// It uses the Intent provider configuration with fallback to main LLM.
//
// Priority:
// 1. If AIIntentAPIKey is configured, use Intent provider (siliconflow by default)
// 2. Otherwise, fallback to main LLM service
func NewSimpleTaskLLMService(p *profile.Profile, mainLLM LLMService) LLMService {
	// If Intent API key is configured, create dedicated service
	if p.AIIntentAPIKey != "" {
		cfg := &LLMConfig{
			Provider:    p.AIIntentProvider,
			Model:       p.AIIntentModel,
			APIKey:      p.AIIntentAPIKey,
			BaseURL:     p.AIIntentBaseURL,
			MaxTokens:   SimpleTaskMaxTokens,
			Temperature: SimpleTaskTemperature,
		}

		svc, err := NewLLMService(cfg)
		if err != nil {
			slog.Warn("Failed to create simple task LLM service, falling back to main LLM",
				"provider", cfg.Provider,
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
