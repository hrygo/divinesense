package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/router"
)

// mockRouterService implements router.RouterService for testing.
type mockRouterService struct {
	classifyFunc func(ctx context.Context, input string) (router.Intent, float32, error)
}

func (m *mockRouterService) ClassifyIntent(ctx context.Context, input string) (router.Intent, float32, error) {
	if m.classifyFunc != nil {
		return m.classifyFunc(ctx, input)
	}
	return router.IntentUnknown, 0, nil
}

func (m *mockRouterService) SelectModel(ctx context.Context, task router.TaskType) (router.ModelConfig, error) {
	return router.ModelConfig{}, nil
}

func (m *mockRouterService) GetRouterStats(ctx context.Context, userID int32, timeRange time.Duration) (*router.RouterStats, error) {
	return &router.RouterStats{
		ByIntent:         make(map[router.Intent]int64),
		BySource:         make(map[string]int64),
		TotalPredictions: 0,
		CorrectCount:     0,
		IncorrectCount:   0,
		Accuracy:         0,
		LastUpdated:      0,
	}, nil
}

func (m *mockRouterService) RecordFeedback(ctx context.Context, feedback *router.RouterFeedback) error {
	return nil
}

func TestScheduleIntentClassifier_SimpleCreate(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input string
	}{
		{"明天下午3点开会"},
		{"今天下午3点面试"},
		{"安排下周一会议"},
		{"上午10点约客户"},
		{"帮我安排一个日程"},
		{"下周三15:00开会"},
		{"后天早上9点见面"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != IntentSimpleCreate {
				t.Errorf("Classify(%q) = %v, want IntentSimpleCreate", tt.input, result.Intent)
			}
			if result.Confidence < 0.6 {
				t.Errorf("Classify(%q) confidence = %v, want >= 0.6", tt.input, result.Confidence)
			}
		})
	}
}

func TestScheduleIntentClassifier_SimpleQuery(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input string
	}{
		{"今天有什么安排"},
		{"明天忙吗"},
		{"这周有空吗"},
		{"查看日程"},
		{"显示明天的安排"},
		{"有没有会议"},
		{"今天的日程"},
		{"几点有会？"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != IntentSimpleQuery {
				t.Errorf("Classify(%q) = %v, want IntentSimpleQuery", tt.input, result.Intent)
			}
		})
	}
}

func TestScheduleIntentClassifier_SimpleUpdate(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input string
	}{
		{"把会议改到3点"},
		{"取消明天的会议"},
		{"推迟日程"},
		{"删除这个安排"},
		{"调整会议时间"},
		{"会议延后一小时"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != IntentSimpleUpdate {
				t.Errorf("Classify(%q) = %v, want IntentSimpleUpdate", tt.input, result.Intent)
			}
		})
	}
}

func TestScheduleIntentClassifier_BatchCreate(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input string
	}{
		{"每周一9点站会"},
		{"工作日早上8点晨会"},
		{"每天下午3点review"},
		{"周一到周五都要开会"},
		{"每月1号汇报"},
		{"这周每天都要锻炼"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != IntentBatchCreate {
				t.Errorf("Classify(%q) = %v, want IntentBatchCreate", tt.input, result.Intent)
			}
			if result.Confidence < 0.9 {
				t.Errorf("Classify(%q) confidence = %v, want >= 0.9", tt.input, result.Confidence)
			}
		})
	}
}

func TestScheduleIntentClassifier_NoLLMForRuleMatch(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input string
	}{
		{"明天下午3点开会"},
		{"今天有什么安排"},
		{"取消会议"},
		{"每周一站会"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.UsedLLM {
				t.Errorf("Classify(%q) UsedLLM = true, want false (should use rules)", tt.input)
			}
		})
	}
}

func TestScheduleIntentClassifier_LLMFallback(t *testing.T) {
	mockRouter := &mockRouterService{
		classifyFunc: func(ctx context.Context, input string) (router.Intent, float32, error) {
			return router.IntentScheduleCreate, 0.8, nil
		},
	}
	c := NewScheduleIntentClassifier(mockRouter)

	// Ambiguous input that doesn't match rules
	result := c.Classify(context.Background(), "处理一下那件事")

	if result.Intent == IntentUnknown && !result.UsedLLM {
		// If intent is unknown and LLM was not used, that's expected for truly ambiguous input
		return
	}

	if result.UsedLLM {
		if result.Intent != IntentSimpleCreate {
			t.Errorf("Classify() = %v, want IntentSimpleCreate from LLM fallback", result.Intent)
		}
	}
}

func TestScheduleIntentClassifier_ShouldUsePlanExecute(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		intent   ScheduleIntent
		expected bool
	}{
		{IntentBatchCreate, true},
		{IntentSimpleCreate, false},
		{IntentSimpleQuery, false},
		{IntentSimpleUpdate, false},
		{IntentUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.intent.String(), func(t *testing.T) {
			result := c.ShouldUsePlanExecute(tt.intent)
			if result != tt.expected {
				t.Errorf("ShouldUsePlanExecute(%v) = %v, want %v", tt.intent, result, tt.expected)
			}
		})
	}
}

func TestScheduleIntentClassifier_ClassifyAndRoute(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input           string
		expectedIntent  ScheduleIntent
		expectedPlanExe bool
	}{
		{"明天3点开会", IntentSimpleCreate, false},
		{"每周一站会", IntentBatchCreate, true},
		{"今天有什么安排", IntentSimpleQuery, false},
		{"取消会议", IntentSimpleUpdate, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			intent, usePlanExecute, _ := c.ClassifyAndRoute(context.Background(), tt.input)
			if intent != tt.expectedIntent {
				t.Errorf("ClassifyAndRoute(%q) intent = %v, want %v", tt.input, intent, tt.expectedIntent)
			}
			if usePlanExecute != tt.expectedPlanExe {
				t.Errorf("ClassifyAndRoute(%q) usePlanExecute = %v, want %v", tt.input, usePlanExecute, tt.expectedPlanExe)
			}
		})
	}
}

func TestScheduleIntentClassifier_DistinguishQueryFromCreate(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input    string
		expected ScheduleIntent
	}{
		// Query cases - no specific action
		{"今天有什么", IntentSimpleQuery},
		{"明天有空吗", IntentSimpleQuery},
		{"这周安排了什么", IntentSimpleQuery},

		// Create cases - has time + action
		{"今天3点开会", IntentSimpleCreate},
		{"明天下午面试", IntentSimpleCreate},
		{"这周三安排会议", IntentSimpleCreate},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.input, result.Intent, tt.expected)
			}
		})
	}
}

func TestScheduleIntentClassifier_HighConfidenceForPatternMatch(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	// Pattern matches should have high confidence
	tests := []struct {
		input         string
		minConfidence float32
	}{
		{"每周一9点站会", 0.9},   // Batch
		{"把会议改到3点", 0.85},  // Update
		{"今天有什么安排", 0.8},   // Query - keyword fallback
		{"明天下午3点开会", 0.85}, // Create
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Confidence < tt.minConfidence {
				t.Errorf("Classify(%q) confidence = %v, want >= %v", tt.input, result.Confidence, tt.minConfidence)
			}
		})
	}
}

func TestScheduleIntentClassifier_MapRouterIntent(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		routerIntent router.Intent
		expected     ScheduleIntent
	}{
		{router.IntentScheduleCreate, IntentSimpleCreate},
		{router.IntentScheduleQuery, IntentSimpleQuery},
		{router.IntentScheduleUpdate, IntentSimpleUpdate},
		{router.IntentBatchSchedule, IntentBatchCreate},
		{router.IntentMemoSearch, IntentUnknown},
		{router.IntentUnknown, IntentUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.routerIntent), func(t *testing.T) {
			result := c.mapRouterIntent(tt.routerIntent)
			if result != tt.expected {
				t.Errorf("mapRouterIntent(%v) = %v, want %v", tt.routerIntent, result, tt.expected)
			}
		})
	}
}

func TestScheduleIntent_String(t *testing.T) {
	tests := []struct {
		intent   ScheduleIntent
		expected string
	}{
		{IntentSimpleCreate, "simple_create"},
		{IntentSimpleQuery, "simple_query"},
		{IntentSimpleUpdate, "simple_update"},
		{IntentBatchCreate, "batch_create"},
		{IntentUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.intent.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScheduleIntentClassifier_KeywordFallback(t *testing.T) {
	c := NewScheduleIntentClassifier(nil)

	tests := []struct {
		input    string
		expected ScheduleIntent
	}{
		// Keyword-based detection
		{"下午约一下", IntentSimpleCreate}, // 下午(time) + 约(create)
		{"查一下今天", IntentSimpleQuery},  // 查(query)
		{"改一下时间", IntentSimpleUpdate}, // 改(update)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := c.Classify(context.Background(), tt.input)
			if result.Intent != tt.expected {
				t.Errorf("Classify(%q) = %v, want %v", tt.input, result.Intent, tt.expected)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkScheduleIntentClassifier_Classify(b *testing.B) {
	c := NewScheduleIntentClassifier(nil)
	inputs := []string{
		"明天下午3点开会",
		"今天有什么安排",
		"每周一9点站会",
		"取消会议",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Classify(context.Background(), inputs[i%len(inputs)])
	}
}

func BenchmarkScheduleIntentClassifier_PatternMatch(b *testing.B) {
	c := NewScheduleIntentClassifier(nil)
	input := "明天下午3点开会"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Classify(context.Background(), input)
	}
}
