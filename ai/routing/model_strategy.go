// Package routing provides model selection strategy for OCP-compliant model management.
package routing

import (
	"context"
	"sync"
)

// ModelStrategy defines the interface for model selection.
// Implementations can use different strategies: config-based, cost-optimized, etc.
type ModelStrategy interface {
	// SelectModel selects an appropriate model based on task type.
	SelectModel(ctx context.Context, task TaskType) (ModelConfig, error)
}

// ConfigDrivenModelStrategy selects models based on a configuration map.
type ConfigDrivenModelStrategy struct {
	mu       sync.RWMutex
	configs  map[TaskType]ModelConfig
	fallback ModelConfig
}

// NewConfigDrivenModelStrategy creates a strategy with config-based selection.
func NewConfigDrivenModelStrategy(configs map[TaskType]ModelConfig, fallback ModelConfig) *ConfigDrivenModelStrategy {
	if configs == nil {
		configs = make(map[TaskType]ModelConfig)
	}
	return &ConfigDrivenModelStrategy{
		configs:  configs,
		fallback: fallback,
	}
}

// SelectModel implements ModelStrategy.
func (s *ConfigDrivenModelStrategy) SelectModel(ctx context.Context, task TaskType) (ModelConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cfg, ok := s.configs[task]; ok {
		return cfg, nil
	}
	return s.fallback, nil
}

// Register adds or updates a model config for a task type.
func (s *ConfigDrivenModelStrategy) Register(task TaskType, cfg ModelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[task] = cfg
}

// SetFallback updates the fallback model config.
func (s *ConfigDrivenModelStrategy) SetFallback(cfg ModelConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallback = cfg
}

// DefaultModelConfigs returns built-in model configurations.
func DefaultModelConfigs() map[TaskType]ModelConfig {
	return map[TaskType]ModelConfig{
		TaskIntentClassification: {
			Provider:    "local",
			Model:       "qwen2.5-0.5b",
			MaxTokens:   256,
			Temperature: 0.1,
		},
		TaskEntityExtraction: {
			Provider:    "local",
			Model:       "qwen2.5-1.5b",
			MaxTokens:   512,
			Temperature: 0.2,
		},
		TaskSimpleQA: {
			Provider:    "local",
			Model:       "qwen2.5-3b",
			MaxTokens:   1024,
			Temperature: 0.3,
		},
		TaskComplexReasoning: {
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   4096,
			Temperature: 0.5,
		},
		TaskSummarization: {
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   2048,
			Temperature: 0.3,
		},
		TaskTagSuggestion: {
			Provider:    "local",
			Model:       "qwen2.5-1.5b",
			MaxTokens:   256,
			Temperature: 0.4,
		},
	}
}

// DefaultFallbackModel returns the default fallback model config.
func DefaultFallbackModel() ModelConfig {
	return ModelConfig{
		Provider:    "cloud",
		Model:       "deepseek-chat",
		MaxTokens:   2048,
		Temperature: 0.5,
	}
}

// NewDefaultModelStrategy creates a strategy with built-in defaults.
func NewDefaultModelStrategy() *ConfigDrivenModelStrategy {
	return NewConfigDrivenModelStrategy(DefaultModelConfigs(), DefaultFallbackModel())
}
