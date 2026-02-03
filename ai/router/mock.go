package router

import (
	"context"
	"strings"
)

// MockRouterService is a mock implementation of RouterService for testing.
type MockRouterService struct {
	// IntentOverrides allows tests to override intent classification results
	IntentOverrides map[string]Intent
	// ModelOverrides allows tests to override model selection results
	ModelOverrides map[TaskType]ModelConfig
}

// NewMockRouterService creates a new MockRouterService.
func NewMockRouterService() *MockRouterService {
	return &MockRouterService{
		IntentOverrides: make(map[string]Intent),
		ModelOverrides:  make(map[TaskType]ModelConfig),
	}
}

// ClassifyIntent classifies user intent using rule-based matching.
func (m *MockRouterService) ClassifyIntent(ctx context.Context, input string) (Intent, float32, error) {
	// Check for overrides first
	if intent, ok := m.IntentOverrides[input]; ok {
		return intent, 1.0, nil
	}

	// Simple rule-based classification
	inputLower := strings.ToLower(input)

	// Memo search patterns
	searchPatterns := []string{"查找", "搜索", "找", "有什么", "哪些", "search", "find"}
	for _, p := range searchPatterns {
		if strings.Contains(inputLower, p) && containsAny(inputLower, []string{"笔记", "memo", "记录"}) {
			return IntentMemoSearch, 0.9, nil
		}
	}

	// Memo create patterns
	createPatterns := []string{"记录", "记一下", "写", "保存", "create", "write", "save"}
	for _, p := range createPatterns {
		if strings.Contains(inputLower, p) {
			return IntentMemoCreate, 0.85, nil
		}
	}

	// Schedule query patterns
	scheduleQueryPatterns := []string{"日程", "安排", "计划", "什么时候", "schedule", "appointment"}
	for _, p := range scheduleQueryPatterns {
		if strings.Contains(inputLower, p) && containsAny(inputLower, []string{"查", "看", "有", "什么", "query", "show"}) {
			return IntentScheduleQuery, 0.85, nil
		}
	}

	// Schedule create patterns
	scheduleCreatePatterns := []string{"提醒", "设置", "创建日程", "安排", "remind", "set", "schedule"}
	for _, p := range scheduleCreatePatterns {
		if strings.Contains(inputLower, p) {
			if containsAny(inputLower, []string{"批量", "多个", "batch", "multiple"}) {
				return IntentBatchSchedule, 0.8, nil
			}
			return IntentScheduleCreate, 0.85, nil
		}
	}

	// Schedule update patterns
	scheduleUpdatePatterns := []string{"修改", "更新", "取消", "改", "update", "cancel", "modify"}
	for _, p := range scheduleUpdatePatterns {
		if strings.Contains(inputLower, p) && containsAny(inputLower, []string{"日程", "提醒", "schedule", "reminder"}) {
			return IntentScheduleUpdate, 0.85, nil
		}
	}

	// Amazing (general assistant) - fallback for questions
	if strings.Contains(inputLower, "?") || strings.Contains(inputLower, "？") ||
		containsAny(inputLower, []string{"什么", "怎么", "为什么", "how", "what", "why"}) {
		return IntentAmazing, 0.7, nil
	}

	return IntentUnknown, 0.3, nil
}

// SelectModel selects an appropriate model based on task type.
func (m *MockRouterService) SelectModel(ctx context.Context, task TaskType) (ModelConfig, error) {
	// Check for overrides first
	if config, ok := m.ModelOverrides[task]; ok {
		return config, nil
	}

	// Default model configurations based on task type
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

// containsAny checks if s contains any of the patterns.
func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// Ensure MockRouterService implements RouterService.
var _ RouterService = (*MockRouterService)(nil)
