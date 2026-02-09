// Package universal provides tests for UniversalParrot.
package universal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agent"
)

// TestNewUniversalParrot tests parrot creation.
func TestNewUniversalParrot(t *testing.T) {
	tests := []struct {
		name        string
		config      *ParrotConfig
		llm         ai.LLMService
		tools       map[string]agent.ToolWithSchema
		userID      int32
		expectError bool
	}{
		{
			name: "valid config",
			config: &ParrotConfig{
				Name:          "test_parrot",
				Strategy:      StrategyDirect,
				MaxIterations: 5,
				SystemPrompt:  "You are helpful",
				EnableCache:   false,
			},
			llm:         &mockLLM{},
			tools:       make(map[string]agent.ToolWithSchema),
			userID:      1,
			expectError: false,
		},
		{
			name: "missing name",
			config: &ParrotConfig{
				Strategy: StrategyDirect,
			},
			llm:         &mockLLM{},
			tools:       make(map[string]agent.ToolWithSchema),
			userID:      1,
			expectError: true,
		},
		{
			name: "missing strategy",
			config: &ParrotConfig{
				Name: "test",
			},
			llm:         &mockLLM{},
			tools:       make(map[string]agent.ToolWithSchema),
			userID:      1,
			expectError: true,
		},
		{
			name: "invalid strategy",
			config: &ParrotConfig{
				Name:     "test",
				Strategy: "invalid",
			},
			llm:         &mockLLM{},
			tools:       make(map[string]agent.ToolWithSchema),
			userID:      1,
			expectError: true,
		},
		{
			name: "with cache enabled",
			config: &ParrotConfig{
				Name:        "cached_parrot",
				Strategy:    StrategyReAct,
				EnableCache: true,
				CacheSize:   50,
				CacheTTL:    10 * time.Minute,
			},
			llm:         &mockLLM{},
			tools:       make(map[string]agent.ToolWithSchema),
			userID:      1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parrot, err := NewUniversalParrot(tt.config, tt.llm, tt.tools, tt.userID)

			if tt.expectError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parrot == nil {
				t.Fatal("parrot should not be nil")
			}
			if parrot.Name() != tt.config.Name {
				t.Errorf("Name() = %q, want %q", parrot.Name(), tt.config.Name)
			}
			if parrot.userID != tt.userID {
				t.Errorf("userID = %d, want %d", parrot.userID, tt.userID)
			}
		})
	}
}

// TestUniversalParrot_Name tests the Name method.
func TestUniversalParrot_Name(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test_parrot",
		Strategy: StrategyDirect,
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	if parrot.Name() != "test_parrot" {
		t.Errorf("Name() = %q, want 'test_parrot'", parrot.Name())
	}
}

// TestUniversalParrot_SelfDescribe tests the SelfDescribe method.
func TestUniversalParrot_SelfDescribe(t *testing.T) {
	config := &ParrotConfig{
		Name:        "memo",
		DisplayName: "Memo Parrot",
		Emoji:       "📝",
		Strategy:    StrategyReAct,
		Tools:       []string{"memo_search"},
		SelfDescription: &agent.ParrotSelfCognition{
			Title:        "Memo Parrot",
			Name:         "memo",
			Emoji:        "📝",
			Capabilities: []string{"memo_search"},
		},
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	desc := parrot.SelfDescribe()

	if desc.Title != "Memo Parrot" {
		t.Errorf("Title = %q, want 'Memo Parrot'", desc.Title)
	}
	if desc.Name != "memo" {
		t.Errorf("Name = %q, want 'memo'", desc.Name)
	}
	if desc.Emoji != "📝" {
		t.Errorf("Emoji = %q, want '📝'", desc.Emoji)
	}
}

// TestUniversalParrot_SelfDescribe_Fallback tests fallback description.
func TestUniversalParrot_SelfDescribe_Fallback(t *testing.T) {
	config := &ParrotConfig{
		Name:        "test",
		DisplayName: "Test Parrot",
		Emoji:       "🧪",
		Strategy:    StrategyDirect,
		Tools:       []string{"tool1", "tool2"},
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	desc := parrot.SelfDescribe()

	// Should use config fields when SelfDescription is nil
	if desc.Title != "Test Parrot" {
		t.Errorf("Title = %q, want 'Test Parrot'", desc.Title)
	}
}

// TestUniversalParrot_ExecuteWithCache tests cache functionality.
func TestUniversalParrot_ExecuteWithCache(t *testing.T) {
	config := &ParrotConfig{
		Name:        "cached",
		Strategy:    StrategyDirect,
		EnableCache: true,
		CacheSize:   10,
		CacheTTL:    5 * time.Minute,
	}

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			return &ai.ChatResponse{Content: "Cached response"}, &ai.LLMCallStats{}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	// First call - cache miss
	err = parrot.ExecuteWithCallback(ctx, "test query", nil, callback)
	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	if llmCalls != 1 {
		t.Errorf("LLM called %d times after first call, want 1", llmCalls)
	}

	// Second call with same input - cache hit
	err = parrot.ExecuteWithCallback(ctx, "test query", nil, callback)
	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	// Should still be 1 due to cache hit
	if llmCalls != 1 {
		t.Errorf("LLM called %d times after second call (cache should have hit), want 1", llmCalls)
	}
}

// TestUniversalParrot_CacheExpiration tests cache TTL expiration.
func TestUniversalParrot_CacheExpiration(t *testing.T) {
	config := &ParrotConfig{
		Name:        "cached",
		Strategy:    StrategyDirect,
		EnableCache: true,
		CacheSize:   10,
		CacheTTL:    50 * time.Millisecond,
	}

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			return &ai.ChatResponse{Content: "Response"}, &ai.LLMCallStats{}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	// First call
	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)
	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second call - cache should have expired
	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)
	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	if llmCalls != 2 {
		t.Errorf("LLM called %d times, want 2 (cache should have expired)", llmCalls)
	}
}

// TestUniversalParrot_ExecuteWithCallback tests execution with callback.
func TestUniversalParrot_ExecuteWithCallback(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return &ai.ChatResponse{Content: "Test response"}, &ai.LLMCallStats{}, nil
		},
	}

	var events []string
	var mu sync.Mutex
	callback := func(eventType string, data any) error {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, eventType)
		return nil
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	err = parrot.ExecuteWithCallback(ctx, "test input", nil, callback)

	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have answer events
	if len(events) == 0 {
		t.Error("expected events to be sent")
	}
}

// TestUniversalParrot_Execute_ErrorHandling tests error handling.
func TestUniversalParrot_Execute_ErrorHandling(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			return nil, nil, errors.New("LLM error")
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)

	if err == nil {
		t.Error("expected error from ExecuteWithCallback")
	}

	// Error should be wrapped with parrot name
	var parrotErr *agent.ParrotError
	if errors.As(err, &parrotErr) {
		if parrotErr.AgentName != "test" {
			t.Errorf("ParrotError.AgentName = %q, want 'test'", parrotErr.AgentName)
		}
	}
}

// TestUniversalParrot_GetSessionStats tests session statistics.
func TestUniversalParrot_GetSessionStats(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			time.Sleep(5 * time.Millisecond) // Ensure positive duration
			return &ai.ChatResponse{Content: "Response"}, &ai.LLMCallStats{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	// Execute to accumulate stats
	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)
	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	stats := parrot.GetSessionStats()

	if stats == nil {
		t.Fatal("stats should not be nil")
	}
	if stats.AgentType != "test" {
		t.Errorf("AgentType = %q, want 'test'", stats.AgentType)
	}
	if stats.PromptTokens != 100 {
		t.Errorf("PromptTokens = %d, want 100", stats.PromptTokens)
	}
	if stats.CompletionTokens != 50 {
		t.Errorf("CompletionTokens = %d, want 50", stats.CompletionTokens)
	}
	if stats.TotalTokens != 150 {
		t.Errorf("TotalTokens = %d, want 150", stats.TotalTokens)
	}
	if stats.TotalDurationMs == 0 {
		t.Error("TotalDurationMs should be non-zero")
	}
}

// TestUniversalParrot_ResolveTools tests tool resolution.
func TestUniversalParrot_ResolveTools(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
		Tools:    []string{"tool1", "tool2", "missing_tool"},
	}

	tools := map[string]agent.ToolWithSchema{
		"tool1": &mockTool{name: "tool1"},
		"tool2": &mockTool{name: "tool2"},
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, tools, 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	resolved := parrot.resolveTools()

	// Should only return tools that exist
	if len(resolved) != 2 {
		t.Errorf("resolved %d tools, want 2", len(resolved))
	}

	names := make(map[string]bool)
	for _, t := range resolved {
		names[t.Name()] = true
	}

	if !names["tool1"] || !names["tool2"] {
		t.Error("tool1 and tool2 should be resolved")
	}
	if names["missing_tool"] {
		t.Error("missing_tool should not be resolved")
	}
}

// TestUniversalParrot_SetRetriever tests setting retriever dependency.
func TestUniversalParrot_SetRetriever(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	retriever := "mock_retriever"
	parrot.SetRetriever(retriever)

	if parrot.retriever != retriever {
		t.Error("retriever should be set")
	}
}

// TestUniversalParrot_SetScheduleService tests setting schedule service dependency.
func TestUniversalParrot_SetScheduleService(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	scheduleService := "mock_schedule"
	parrot.SetScheduleService(scheduleService)

	if parrot.scheduleService != scheduleService {
		t.Error("scheduleService should be set")
	}
}

// TestUniversalParrot_BuildMessages tests message building from history.
func TestUniversalParrot_BuildMessages(t *testing.T) {
	tests := []struct {
		name     string
		config   *ParrotConfig
		history  []string
		expected int
	}{
		{
			name: "empty history with system prompt",
			config: &ParrotConfig{
				Name:         "test",
				Strategy:     StrategyDirect,
				SystemPrompt: "You are helpful",
			},
			history:  nil,
			expected: 1, // Just system prompt
		},
		{
			name: "history with pairs",
			config: &ParrotConfig{
				Name:     "test",
				Strategy: StrategyDirect,
			},
			history:  []string{"user message", "assistant response"},
			expected: 2, // user + assistant pair
		},
		{
			name: "history with system prompt and pairs",
			config: &ParrotConfig{
				Name:         "test",
				Strategy:     StrategyDirect,
				SystemPrompt: "You are helpful",
			},
			history:  []string{"Hello", "Hi there", "How are you?", "Good"},
			expected: 5, // system + 2 pairs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parrot, err := NewUniversalParrot(tt.config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 1)
			if err != nil {
				t.Fatalf("NewUniversalParrot() error = %v", err)
			}

			messages := parrot.buildMessages(tt.history)

			if len(messages) != tt.expected {
				t.Errorf("message count = %d, want %d", len(messages), tt.expected)
			}

			// Check system prompt position
			if tt.config.SystemPrompt != "" && len(messages) > 0 {
				if messages[0].Role != "system" {
					t.Error("first message should be system prompt")
				}
			}
		})
	}
}

// TestUniversalParrot_GenerateCacheKey tests cache key generation.
func TestUniversalParrot_GenerateCacheKey(t *testing.T) {
	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	parrot, err := NewUniversalParrot(config, &mockLLM{}, make(map[string]agent.ToolWithSchema), 123)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	input := "test query"
	key1 := parrot.generateCacheKey(parrot.Name(), parrot.userID, input)
	key2 := parrot.generateCacheKey(parrot.Name(), parrot.userID, input)

	// Same input should produce same key
	if key1 != key2 {
		t.Error("same input should produce same cache key")
	}

	// Different input should produce different key
	key3 := parrot.generateCacheKey(parrot.Name(), parrot.userID, "different query")
	if key1 == key3 {
		t.Error("different inputs should produce different cache keys")
	}

	// Different user should produce different key
	key4 := parrot.generateCacheKey(parrot.Name(), 456, input)
	if key1 == key4 {
		t.Error("different users should produce different cache keys")
	}
}

// TestNormalSessionStats tests session statistics.
func TestNormalSessionStats(t *testing.T) {
	stats := NewNormalSessionStats("test_parrot")

	if stats.AgentType != "test_parrot" {
		t.Errorf("AgentType = %q, want 'test_parrot'", stats.AgentType)
	}
	if stats.StartTime.IsZero() {
		t.Error("StartTime should be initialized to non-zero")
	}
	if len(stats.ToolsUsed) != 0 {
		t.Error("ToolsUsed should be initialized as empty slice")
	}
}

// TestNormalSessionStats_GetStatsSnapshot tests stats snapshot.
func TestNormalSessionStats_GetStatsSnapshot(t *testing.T) {
	stats := NewNormalSessionStats("test_parrot")

	// Modify some stats
	stats.PromptTokens = 100
	stats.CompletionTokens = 50
	stats.TotalTokens = 150
	stats.ToolCallCount = 2
	stats.ToolsUsed = []string{"tool1", "tool2"}

	// Wait a bit for duration
	time.Sleep(10 * time.Millisecond)

	snapshot := stats.GetStatsSnapshot()

	// Snapshot should be a copy, not the same object
	if snapshot == stats {
		t.Error("snapshot should be a different object")
	}

	// Values should match
	if snapshot.PromptTokens != 100 {
		t.Errorf("snapshot.PromptTokens = %d, want 100", snapshot.PromptTokens)
	}
	if snapshot.CompletionTokens != 50 {
		t.Errorf("snapshot.CompletionTokens = %d, want 50", snapshot.CompletionTokens)
	}
	if snapshot.TotalTokens != 150 {
		t.Errorf("snapshot.TotalTokens = %d, want 150", snapshot.TotalTokens)
	}
	if snapshot.ToolCallCount != 2 {
		t.Errorf("snapshot.ToolCallCount = %d, want 2", snapshot.ToolCallCount)
	}
}

// TestUniversalParrot_ConcurrentExecution tests concurrent execution safety.
func TestUniversalParrot_ConcurrentExecution(t *testing.T) {
	config := &ParrotConfig{
		Name:        "concurrent_test",
		Strategy:    StrategyDirect,
		EnableCache: true,
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			time.Sleep(10 * time.Millisecond)
			return &ai.ChatResponse{Content: "Response"}, &ai.LLMCallStats{}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Run 10 concurrent executions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := parrot.ExecuteWithCallback(ctx, "test", nil, callback)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("Execution error: %v", err)
		}
	}

	if errorCount > 0 {
		t.Errorf("got %d execution errors", errorCount)
	}

	// Verify stats are consistent
	stats := parrot.GetSessionStats()
	if stats == nil {
		t.Error("stats should not be nil after concurrent executions")
	}
}

// TestUniversalParrot_ContextCancellation tests context cancellation during execution.
func TestUniversalParrot_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping context cancellation test in short mode - timing sensitive")
	}

	config := &ParrotConfig{
		Name:     "test",
		Strategy: StrategyDirect,
	}

	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			// Use select to check context cancellation
			select {
			case <-ctx.Done():
				return nil, &ai.LLMCallStats{}, ctx.Err()
			case <-time.After(1 * time.Hour):
				return &ai.ChatResponse{Content: "Should not reach"}, &ai.LLMCallStats{}, nil
			}
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	callback := func(eventType string, data any) error { return nil }

	err = parrot.ExecuteWithCallback(ctx, "test", nil, callback)

	if err == nil {
		t.Error("expected error due to timeout")
	}
}

// TestUniversalParrot_ExecuteWithHistory tests execution with conversation history.
func TestUniversalParrot_ExecuteWithHistory(t *testing.T) {
	config := &ParrotConfig{
		Name:         "test",
		Strategy:     StrategyDirect,
		SystemPrompt: "You are a helpful assistant",
	}

	var receivedMessages []ai.Message
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			receivedMessages = messages
			return &ai.ChatResponse{Content: "Response"}, &ai.LLMCallStats{}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	history := []string{"Hello", "Hi there!", "How are you?", "I'm good"}

	err = parrot.ExecuteWithCallback(ctx, "What's next?", history, callback)

	if err != nil {
		t.Fatalf("ExecuteWithCallback() error = %v", err)
	}

	// buildMessages creates: system prompt (1) + 2 pairs from history (4 strings -> 2 user+assistant pairs = 4 messages) = 5 messages
	// BuildMessagesWithInput then appends the current input (+1) = 6 total
	expectedCount := 1 + len(history) + 1 // system (1) + history as individual messages (4) + current input (1) = 6
	if len(receivedMessages) != expectedCount {
		t.Errorf("received %d messages, want %d", len(receivedMessages), expectedCount)
	}

	// Check system prompt is first
	if receivedMessages[0].Role != "system" {
		t.Error("first message should be system prompt")
	}
}

// TestUniversalParrot_DisabledCache tests that cache is not used when disabled.
func TestUniversalParrot_DisabledCache(t *testing.T) {
	config := &ParrotConfig{
		Name:        "no_cache",
		Strategy:    StrategyDirect,
		EnableCache: false,
	}

	llmCalls := 0
	llm := &mockLLM{
		chatWithToolsFunc: func(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
			llmCalls++
			return &ai.ChatResponse{Content: "Response"}, &ai.LLMCallStats{}, nil
		},
	}

	parrot, err := NewUniversalParrot(config, llm, make(map[string]agent.ToolWithSchema), 1)
	if err != nil {
		t.Fatalf("NewUniversalParrot() error = %v", err)
	}

	ctx := context.Background()
	callback := func(eventType string, data any) error { return nil }

	// Execute twice with same input
	parrot.ExecuteWithCallback(ctx, "test", nil, callback)
	parrot.ExecuteWithCallback(ctx, "test", nil, callback)

	// Should call LLM twice (no cache)
	if llmCalls != 2 {
		t.Errorf("LLM called %d times, want 2 (cache is disabled)", llmCalls)
	}
}
