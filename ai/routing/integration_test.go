// Package routing provides integration tests for the routing system.
package routing

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/services/memory"
)

// mockLLMClient is a mock implementation of LLMClient for testing.
type mockLLMClient struct {
	responses map[string]string
}

func newMockLLMClient() *mockLLMClient {
	return &mockLLMClient{
		responses: make(map[string]string),
	}
}

func (m *mockLLMClient) SetResponse(input, response string) {
	m.responses[input] = response
}

func (m *mockLLMClient) Complete(ctx context.Context, prompt string, config ModelConfig) (string, error) {
	// Check for predefined response
	if resp, ok := m.responses[prompt]; ok {
		return resp, nil
	}
	// Default JSON response for intent classification
	return `{"intent": "memo_search", "confidence": 0.9}`, nil
}

// mockMemoryService is a mock implementation of memory.MemoryService.
type mockMemoryServiceForRouter struct {
	episodes []memory.EpisodicMemory
	messages map[string][]memory.Message
}

func newMockMemoryServiceForRouter() *mockMemoryServiceForRouter {
	return &mockMemoryServiceForRouter{
		episodes: make([]memory.EpisodicMemory, 0),
		messages: make(map[string][]memory.Message),
	}
}

func (m *mockMemoryServiceForRouter) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]memory.Message, error) {
	if msgs, ok := m.messages[sessionID]; ok {
		if len(msgs) > limit {
			return msgs[len(msgs)-limit:], nil
		}
		return msgs, nil
	}
	return []memory.Message{}, nil
}

func (m *mockMemoryServiceForRouter) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]memory.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func (m *mockMemoryServiceForRouter) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]memory.EpisodicMemory, error) {
	result := make([]memory.EpisodicMemory, 0)
	for _, ep := range m.episodes {
		if ep.UserID == userID {
			result = append(result, ep)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockMemoryServiceForRouter) SaveEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	episode.ID = int64(len(m.episodes) + 1)
	m.episodes = append(m.episodes, episode)
	return nil
}

func (m *mockMemoryServiceForRouter) GetEpisode(ctx context.Context, id int64) (*memory.EpisodicMemory, error) {
	if id <= 0 || int(id) > len(m.episodes) {
		return nil, nil
	}
	return &m.episodes[id-1], nil
}

func (m *mockMemoryServiceForRouter) UpdateEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	if episode.ID <= 0 || int(episode.ID) > len(m.episodes) {
		return nil
	}
	m.episodes[episode.ID-1] = episode
	return nil
}

func (m *mockMemoryServiceForRouter) DeleteEpisode(ctx context.Context, id int64) error {
	if id <= 0 || int(id) > len(m.episodes) {
		return nil
	}
	m.episodes = append(m.episodes[:id-1], m.episodes[id:]...)
	return nil
}

func (m *mockMemoryServiceForRouter) ListEpisodes(ctx context.Context, userID int32, offset, limit int) ([]memory.EpisodicMemory, error) {
	result := make([]memory.EpisodicMemory, 0)
	for _, ep := range m.episodes {
		if ep.UserID == userID {
			result = append(result, ep)
		}
	}
	if offset >= len(result) {
		return []memory.EpisodicMemory{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockMemoryServiceForRouter) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	userSet := make(map[int32]bool)
	for _, ep := range m.episodes {
		userSet[ep.UserID] = true
	}
	result := make([]int32, 0, len(userSet))
	for uid := range userSet {
		result = append(result, uid)
	}
	return result, nil
}

func (m *mockMemoryServiceForRouter) GetPreferences(ctx context.Context, userID int32) (*memory.UserPreferences, error) {
	return &memory.UserPreferences{
		Timezone: "UTC",
	}, nil
}

func (m *mockMemoryServiceForRouter) UpdatePreferences(ctx context.Context, userID int32, prefs *memory.UserPreferences) error {
	return nil
}

// TestService_Integration_FullRouting tests the complete routing flow.
func TestService_Integration_FullRouting(t *testing.T) {
	ctx := context.Background()

	t.Run("rule-based routing", func(t *testing.T) {
		svc := NewService(Config{
			EnableCache: true,
		})

		// Clear rule-based match
		intent, confidence, err := svc.ClassifyIntent(ctx, "明天下午3点开会")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if intent != IntentScheduleCreate {
			t.Errorf("expected IntentScheduleCreate, got %s", intent)
		}
		if confidence < 0.5 {
			t.Errorf("expected confidence >= 0.5, got %f", confidence)
		}
	})

	t.Run("cache hit after first classification", func(t *testing.T) {
		svc := NewService(Config{
			EnableCache: true,
		})

		input := "搜索关于人工智能的笔记"

		// First call - should use rule matcher
		intent1, conf1, err1 := svc.ClassifyIntent(ctx, input)
		if err1 != nil {
			t.Fatalf("first call failed: %v", err1)
		}

		// Second call - should hit cache
		intent2, conf2, err2 := svc.ClassifyIntent(ctx, input)
		if err2 != nil {
			t.Fatalf("second call failed: %v", err2)
		}

		if intent1 != intent2 {
			t.Errorf("cache returned different intent: %s vs %s", intent1, intent2)
		}
		if conf1 != conf2 {
			t.Errorf("cache returned different confidence: %f vs %f", conf1, conf2)
		}
	})

	t.Run("LLM fallback when rule fails", func(t *testing.T) {
		llmClient := newMockLLMClient()
		llmClient.SetResponse("用户输入: 这是一个复杂的问题",
			`{"intent": "amazing", "confidence": 0.85}`)

		svc := NewService(Config{
			LLMClient:   llmClient,
			EnableCache: false,
		})

		intent, confidence, err := svc.ClassifyIntent(ctx, "这是一个复杂的问题")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if intent != IntentAmazing {
			t.Errorf("expected IntentAmazing, got %s", intent)
		}
		if confidence < 0.8 {
			t.Errorf("expected confidence >= 0.8, got %f", confidence)
		}
	})
}

// TestService_Integration_WithHistory tests routing with history matching.
func TestService_Integration_WithHistory(t *testing.T) {
	ctx := context.Background()
	userID := int32(123)

	memSvc := newMockMemoryServiceForRouter()
	svc := NewService(Config{
		MemoryService: memSvc,
		EnableCache:   false,
	})

	// Save some historical decisions
	svc.historyMatcher.SaveDecision(ctx, userID, "查找Go语言笔记", IntentMemoSearch, true)
	svc.historyMatcher.SaveDecision(ctx, userID, "明天会议", IntentScheduleCreate, true)

	t.Run("history match for similar input", func(t *testing.T) {
		ctxWithUser := WithUserID(ctx, userID)

		intent, confidence, err := svc.ClassifyIntent(ctxWithUser, "搜索Go语言笔记")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Should match based on history (high similarity)
		if intent != IntentMemoSearch {
			t.Errorf("expected IntentMemoSearch, got %s", intent)
		}
		if confidence < 0.8 {
			t.Errorf("expected high confidence from history match, got %f", confidence)
		}
	})
}

// TestService_Integration_CacheStats tests cache statistics.
func TestService_Integration_CacheStats(t *testing.T) {
	ctx := context.Background()
	svc := NewService(Config{
		EnableCache: true,
	})

	// Initially empty
	stats := svc.GetCacheStats()
	if stats == nil {
		t.Fatal("expected stats to be non-nil")
	}

	// Generate some activity
	svc.ClassifyIntent(ctx, "搜索笔记")
	svc.ClassifyIntent(ctx, "搜索笔记") // Cache hit

	stats = svc.GetCacheStats()
	if stats.Hits == 0 {
		t.Error("expected at least one cache hit")
	}
	if stats.Misses == 0 {
		t.Error("expected at least one cache miss")
	}
}

// TestService_Integration_ClearCache tests cache clearing.
func TestService_Integration_ClearCache(t *testing.T) {
	ctx := context.Background()
	svc := NewService(Config{
		EnableCache: true,
	})

	// Add something to cache
	svc.ClassifyIntent(ctx, "搜索笔记")

	// Clear cache
	svc.ClearCache()

	// Stats should be reset
	stats := svc.GetCacheStats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("expected stats to be reset after ClearCache")
	}
}

// TestService_Integration_UserContext tests user context handling.
func TestService_Integration_UserContext(t *testing.T) {
	ctx := context.Background()

	svc := NewService(Config{
		EnableCache: false,
	})

	userID := int32(456)
	ctxWithUser := WithUserID(ctx, userID)

	// Should not panic with user context
	_, _, err := svc.ClassifyIntent(ctxWithUser, "搜索笔记")
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

	// Get stats for a user with no history
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

	// Verify feedback was recorded
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

	// Prime the cache
	svc.ClassifyIntent(ctx, input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.ClassifyIntent(ctx, input)
	}
}
