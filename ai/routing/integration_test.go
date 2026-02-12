// Package routing provides integration tests for the routing system.
package routing

import (
	"context"
	"testing"
	"time"
)

// TestService_Integration_FullRouting tests the complete routing flow.
func TestService_Integration_FullRouting(t *testing.T) {
	ctx := context.Background()

	t.Run("rule-based routing", func(t *testing.T) {
		svc := NewService(Config{
			EnableCache: true,
		})

		intent, confidence, needsOrch, err := svc.ClassifyIntent(ctx, "明天下午3点开会")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if intent != IntentScheduleCreate {
			t.Errorf("expected IntentScheduleCreate, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
		if needsOrch {
			t.Errorf("expected needsOrchestration=false for clear schedule intent")
		}
	})

	t.Run("cache hit after first classification", func(t *testing.T) {
		svc := NewService(Config{
			EnableCache: true,
		})

		input := "搜索关于人工智能的笔记"

		intent1, conf1, needsOrch1, err1 := svc.ClassifyIntent(ctx, input)
		if err1 != nil {
			t.Fatalf("first call failed: %v", err1)
		}

		intent2, conf2, needsOrch2, err2 := svc.ClassifyIntent(ctx, input)
		if err2 != nil {
			t.Fatalf("second call failed: %v", err2)
		}

		if intent1 != intent2 {
			t.Errorf("cache returned different intent: %s vs %s", intent1, intent2)
		}
		if conf1 != conf2 {
			t.Errorf("cache returned different confidence: %f vs %f", conf1, conf2)
		}
		if needsOrch1 != needsOrch2 {
			t.Errorf("cache returned different needsOrchestration: %v vs %v", needsOrch1, needsOrch2)
		}
	})

	t.Run("needs orchestration for unknown intent", func(t *testing.T) {
		svc := NewService(Config{
			EnableCache: false,
		})

		intent, confidence, needsOrch, err := svc.ClassifyIntent(ctx, "这是一个复杂的问题")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if intent != IntentUnknown {
			t.Errorf("expected IntentUnknown, got %s", intent)
		}
		if !needsOrch {
			t.Errorf("expected needsOrchestration=true for unknown intent")
		}
		_ = confidence
	})
}

// TestService_Integration_UserContext tests user context handling.
func TestService_Integration_UserContext(t *testing.T) {
	ctx := context.Background()

	svc := NewService(Config{
		EnableCache: false,
	})

	userID := int32(456)
	ctxWithUser := WithUserID(ctx, userID)

	_, _, _, err := svc.ClassifyIntent(ctxWithUser, "搜索笔记")
	if err != nil {
		t.Fatalf("expected no error with user context, got %v", err)
	}
}

// TestService_Integration_ModelSelection tests model selection.
func TestService_Integration_ModelSelection(t *testing.T) {
	ctx := context.Background()
	svc := NewService(Config{})

	tasks := []struct {
		task             TaskType
		expectedProvider string
	}{
		{TaskIntentClassification, "local"},
		{TaskEntityExtraction, "local"},
		{TaskSimpleQA, "local"},
		{TaskComplexReasoning, "cloud"},
		{TaskSummarization, "cloud"},
		{TaskTagSuggestion, "local"},
	}

	for _, tt := range tasks {
		t.Run(string(tt.task), func(t *testing.T) {
			config, err := svc.SelectModel(ctx, tt.task)
			if err != nil {
				t.Fatalf("SelectModel failed: %v", err)
			}
			if config.Provider != tt.expectedProvider {
				t.Errorf("expected provider %s, got %s", tt.expectedProvider, config.Provider)
			}
			if config.MaxTokens <= 0 {
				t.Error("expected positive max_tokens")
			}
			if config.Temperature < 0 || config.Temperature > 2 {
				t.Errorf("invalid temperature: %f", config.Temperature)
			}
		})
	}
}

// TestService_Integration_RouterStats tests router statistics.
func TestService_Integration_RouterStats(t *testing.T) {
	ctx := context.Background()
	userID := int32(789)

	svc := NewService(Config{})

	stats, err := svc.GetRouterStats(ctx, userID, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats == nil {
		t.Fatal("expected stats to be non-nil")
	}
	if stats.ByIntent == nil {
		t.Error("expected ByIntent to be initialized")
	}
	if stats.BySource == nil {
		t.Error("expected BySource to be initialized")
	}
}

// TestService_Integration_Feedback tests feedback recording.
func TestService_Integration_Feedback(t *testing.T) {
	ctx := context.Background()
	userID := int32(999)

	storage := NewInMemoryWeightStorage()
	svc := NewService(Config{
		WeightStorage:  storage,
		EnableFeedback: true,
	})

	feedback := &RouterFeedback{
		UserID:    userID,
		Input:     "搜索笔记",
		Predicted: IntentMemoSearch,
		Actual:    IntentMemoSearch,
		Feedback:  FeedbackPositive,
		Source:    "rule",
		Timestamp: time.Now().Unix(),
	}

	err := svc.RecordFeedback(ctx, feedback)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stats, err := storage.GetStats(ctx, userID, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected no error getting stats, got %v", err)
	}

	if stats.TotalPredictions != 1 {
		t.Errorf("expected 1 prediction, got %d", stats.TotalPredictions)
	}
}

// BenchmarkService_Integration_ClassifyIntent benchmarks the full routing flow.
func BenchmarkService_Integration_ClassifyIntent(b *testing.B) {
	ctx := context.Background()
	svc := NewService(Config{
		EnableCache: true,
	})

	input := "搜索关于人工智能和机器学习的笔记"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.ClassifyIntent(ctx, input)
	}
}

// BenchmarkService_Integration_WithCache benchmarks with cache hits.
func BenchmarkService_Integration_WithCache(b *testing.B) {
	ctx := context.Background()
	svc := NewService(Config{
		EnableCache: true,
	})

	input := "搜索关于人工智能的笔记"

	svc.ClassifyIntent(ctx, input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.ClassifyIntent(ctx, input)
	}
}
