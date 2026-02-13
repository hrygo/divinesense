package routing

import (
	"context"
	"testing"
)

func TestConfigDrivenModelStrategy_SelectModel(t *testing.T) {
	configs := map[TaskType]ModelConfig{
		TaskIntentClassification: {
			Provider:    "local",
			Model:       "qwen2.5-0.5b",
			MaxTokens:   256,
			Temperature: 0.1,
		},
		TaskComplexReasoning: {
			Provider:    "cloud",
			Model:       "deepseek-chat",
			MaxTokens:   4096,
			Temperature: 0.5,
		},
	}
	fallback := ModelConfig{
		Provider:    "cloud",
		Model:       "gpt-3.5-turbo",
		MaxTokens:   2048,
		Temperature: 0.5,
	}

	strategy := NewConfigDrivenModelStrategy(configs, fallback)
	ctx := context.Background()

	tests := []struct {
		name      string
		task      TaskType
		wantModel string
		wantErr   bool
	}{
		{
			name:      "configured task - intent classification",
			task:      TaskIntentClassification,
			wantModel: "qwen2.5-0.5b",
			wantErr:   false,
		},
		{
			name:      "configured task - complex reasoning",
			task:      TaskComplexReasoning,
			wantModel: "deepseek-chat",
			wantErr:   false,
		},
		{
			name:      "fallback for unknown task",
			task:      TaskType("unknown_task"),
			wantModel: "gpt-3.5-turbo",
			wantErr:   false,
		},
		{
			name:      "fallback for empty task",
			task:      TaskType(""),
			wantModel: "gpt-3.5-turbo",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.SelectModel(ctx, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SelectModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Model != tt.wantModel {
				t.Errorf("SelectModel() model = %v, want %v", got.Model, tt.wantModel)
			}
		})
	}
}

func TestConfigDrivenModelStrategy_Register(t *testing.T) {
	strategy := NewConfigDrivenModelStrategy(nil, ModelConfig{
		Provider:    "cloud",
		Model:       "fallback",
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	ctx := context.Background()

	// Register a new task config
	newConfig := ModelConfig{
		Provider:    "local",
		Model:       "llama-7b",
		MaxTokens:   512,
		Temperature: 0.3,
	}
	strategy.Register(TaskType("custom_task"), newConfig)

	// Verify it returns the registered config
	got, err := strategy.SelectModel(ctx, TaskType("custom_task"))
	if err != nil {
		t.Fatalf("SelectModel() error = %v", err)
	}

	if got.Model != "llama-7b" {
		t.Errorf("SelectModel() model = %v, want llama-7b", got.Model)
	}
	if got.Provider != "local" {
		t.Errorf("SelectModel() provider = %v, want local", got.Provider)
	}
}

func TestConfigDrivenModelStrategy_SetFallback(t *testing.T) {
	strategy := NewConfigDrivenModelStrategy(nil, ModelConfig{
		Provider:    "cloud",
		Model:       "old-fallback",
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	ctx := context.Background()

	// Update fallback
	newFallback := ModelConfig{
		Provider:    "cloud",
		Model:       "new-fallback",
		MaxTokens:   2048,
		Temperature: 0.4,
	}
	strategy.SetFallback(newFallback)

	// Verify unknown task uses new fallback
	got, err := strategy.SelectModel(ctx, TaskType("unknown"))
	if err != nil {
		t.Fatalf("SelectModel() error = %v", err)
	}

	if got.Model != "new-fallback" {
		t.Errorf("SelectModel() model = %v, want new-fallback", got.Model)
	}
}

func TestConfigDrivenModelStrategy_NilConfigs(t *testing.T) {
	// Test that nil configs map is handled safely
	strategy := NewConfigDrivenModelStrategy(nil, ModelConfig{
		Provider:    "cloud",
		Model:       "fallback",
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	ctx := context.Background()

	// Should use fallback without panic
	got, err := strategy.SelectModel(ctx, TaskType("any_task"))
	if err != nil {
		t.Fatalf("SelectModel() error = %v", err)
	}

	if got.Model != "fallback" {
		t.Errorf("SelectModel() model = %v, want fallback", got.Model)
	}
}

func TestNewDefaultModelStrategy(t *testing.T) {
	strategy := NewDefaultModelStrategy()
	ctx := context.Background()

	tests := []struct {
		name          string
		task          TaskType
		wantProvider  string
		wantMinTokens int
	}{
		{
			name:          "intent classification uses local",
			task:          TaskIntentClassification,
			wantProvider:  "local",
			wantMinTokens: 100,
		},
		{
			name:          "complex reasoning uses cloud",
			task:          TaskComplexReasoning,
			wantProvider:  "cloud",
			wantMinTokens: 1000,
		},
		{
			name:          "summarization uses cloud",
			task:          TaskSummarization,
			wantProvider:  "cloud",
			wantMinTokens: 1000,
		},
		{
			name:          "unknown task uses fallback",
			task:          TaskType("unknown"),
			wantProvider:  "cloud",
			wantMinTokens: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strategy.SelectModel(ctx, tt.task)
			if err != nil {
				t.Fatalf("SelectModel() error = %v", err)
			}

			if got.Provider != tt.wantProvider {
				t.Errorf("SelectModel() provider = %v, want %v", got.Provider, tt.wantProvider)
			}

			if got.MaxTokens < tt.wantMinTokens {
				t.Errorf("SelectModel() maxTokens = %v, want at least %v", got.MaxTokens, tt.wantMinTokens)
			}
		})
	}
}

func TestDefaultModelConfigs(t *testing.T) {
	configs := DefaultModelConfigs()

	// Verify all expected task types have configs
	expectedTasks := []TaskType{
		TaskIntentClassification,
		TaskEntityExtraction,
		TaskSimpleQA,
		TaskComplexReasoning,
		TaskSummarization,
		TaskTagSuggestion,
	}

	for _, task := range expectedTasks {
		cfg, exists := configs[task]
		if !exists {
			t.Errorf("DefaultModelConfigs() missing config for task %v", task)
			continue
		}

		// Verify config is valid
		if cfg.Model == "" {
			t.Errorf("DefaultModelConfigs() task %v has empty model", task)
		}
		if cfg.MaxTokens <= 0 {
			t.Errorf("DefaultModelConfigs() task %v has invalid maxTokens %v", task, cfg.MaxTokens)
		}
		if cfg.Temperature < 0 || cfg.Temperature > 1 {
			t.Errorf("DefaultModelConfigs() task %v has invalid temperature %v", task, cfg.Temperature)
		}
	}
}

func TestConfigDrivenModelStrategy_Concurrency(t *testing.T) {
	strategy := NewConfigDrivenModelStrategy(nil, ModelConfig{
		Provider:    "cloud",
		Model:       "fallback",
		MaxTokens:   1024,
		Temperature: 0.5,
	})
	ctx := context.Background()

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			strategy.Register(TaskType("dynamic_task"), ModelConfig{
				Provider:    "local",
				Model:       "dynamic-model",
				MaxTokens:   i,
				Temperature: 0.5,
			})
		}
		done <- true
	}()

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				strategy.SelectModel(ctx, TaskType("dynamic_task"))
				strategy.SelectModel(ctx, TaskType("unknown"))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 6; i++ {
		<-done
	}
}
