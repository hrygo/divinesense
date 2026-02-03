package router

import (
	"context"
	"testing"
)

// TestRouterServiceContract tests the RouterService contract.
func TestRouterServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockRouterService()

	t.Run("ClassifyIntent_MemoSearch", func(t *testing.T) {
		intent, confidence, err := svc.ClassifyIntent(ctx, "帮我查找上周的笔记")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentMemoSearch {
			t.Errorf("expected IntentMemoSearch, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
	})

	t.Run("ClassifyIntent_MemoCreate", func(t *testing.T) {
		intent, confidence, err := svc.ClassifyIntent(ctx, "记录一下今天的会议内容")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentMemoCreate {
			t.Errorf("expected IntentMemoCreate, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
	})

	t.Run("ClassifyIntent_ScheduleCreate", func(t *testing.T) {
		intent, confidence, err := svc.ClassifyIntent(ctx, "提醒我明天下午3点开会")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentScheduleCreate {
			t.Errorf("expected IntentScheduleCreate, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
	})

	t.Run("ClassifyIntent_ScheduleUpdate", func(t *testing.T) {
		intent, confidence, err := svc.ClassifyIntent(ctx, "取消明天的日程")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentScheduleUpdate {
			t.Errorf("expected IntentScheduleUpdate, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
	})

	t.Run("ClassifyIntent_BatchSchedule", func(t *testing.T) {
		intent, _, err := svc.ClassifyIntent(ctx, "批量设置下周的会议提醒")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentBatchSchedule {
			t.Errorf("expected IntentBatchSchedule, got %s", intent)
		}
	})

	t.Run("ClassifyIntent_Amazing", func(t *testing.T) {
		intent, _, err := svc.ClassifyIntent(ctx, "今天天气怎么样？")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentAmazing {
			t.Errorf("expected IntentAmazing, got %s", intent)
		}
	})

	t.Run("ClassifyIntent_Unknown", func(t *testing.T) {
		intent, confidence, err := svc.ClassifyIntent(ctx, "随便说点什么")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		// Unknown or ambiguous input may return Amazing intent with moderate confidence
		_ = intent
		_ = confidence
	})

	t.Run("ClassifyIntent_Override", func(t *testing.T) {
		svc.IntentOverrides["test input"] = IntentScheduleQuery
		intent, confidence, err := svc.ClassifyIntent(ctx, "test input")
		if err != nil {
			t.Fatalf("ClassifyIntent failed: %v", err)
		}
		if intent != IntentScheduleQuery {
			t.Errorf("expected IntentScheduleQuery from override, got %s", intent)
		}
		if confidence != 1.0 {
			t.Errorf("expected confidence 1.0 for override, got %f", confidence)
		}
	})

	t.Run("SelectModel_IntentClassification", func(t *testing.T) {
		config, err := svc.SelectModel(ctx, TaskIntentClassification)
		if err != nil {
			t.Fatalf("SelectModel failed: %v", err)
		}
		if config.Provider != "local" {
			t.Errorf("expected local provider for intent classification, got %s", config.Provider)
		}
		if config.MaxTokens <= 0 {
			t.Error("expected positive max_tokens")
		}
	})

	t.Run("SelectModel_ComplexReasoning", func(t *testing.T) {
		config, err := svc.SelectModel(ctx, TaskComplexReasoning)
		if err != nil {
			t.Fatalf("SelectModel failed: %v", err)
		}
		if config.Provider != "cloud" {
			t.Errorf("expected cloud provider for complex reasoning, got %s", config.Provider)
		}
	})

	t.Run("SelectModel_Override", func(t *testing.T) {
		customConfig := ModelConfig{
			Provider:    "custom",
			Model:       "custom-model",
			MaxTokens:   100,
			Temperature: 0.5,
		}
		svc.ModelOverrides[TaskSimpleQA] = customConfig

		config, err := svc.SelectModel(ctx, TaskSimpleQA)
		if err != nil {
			t.Fatalf("SelectModel failed: %v", err)
		}
		if config.Provider != "custom" {
			t.Errorf("expected custom provider from override, got %s", config.Provider)
		}
	})

	t.Run("ModelConfig_ValidFields", func(t *testing.T) {
		tasks := []TaskType{
			TaskIntentClassification,
			TaskEntityExtraction,
			TaskSimpleQA,
			TaskComplexReasoning,
			TaskSummarization,
			TaskTagSuggestion,
		}

		for _, task := range tasks {
			config, err := svc.SelectModel(ctx, task)
			if err != nil {
				t.Errorf("SelectModel(%s) failed: %v", task, err)
				continue
			}
			if config.Provider == "" {
				t.Errorf("SelectModel(%s): empty provider", task)
			}
			if config.Model == "" {
				t.Errorf("SelectModel(%s): empty model", task)
			}
			if config.MaxTokens <= 0 {
				t.Errorf("SelectModel(%s): invalid max_tokens", task)
			}
			if config.Temperature < 0 || config.Temperature > 2 {
				t.Errorf("SelectModel(%s): invalid temperature %f", task, config.Temperature)
			}
		}
	})
}
