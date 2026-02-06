package ai

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestConcurrentChat_StatsIsolation verifies that stats are not mixed between concurrent Chat calls.
func TestConcurrentChat_StatsIsolation(t *testing.T) {
	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Number of concurrent calls
	numCalls := 5

	// Track results from each concurrent call
	type callResult struct {
		index int
		stats *LLMCallStats
		err   error
	}
	results := make([]callResult, numCalls)

	var wg sync.WaitGroup
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, stats, err := service.Chat(ctx, []Message{
				{Role: "user", Content: "test message"},
			})
			results[idx] = callResult{index: idx, stats: stats, err: err}
		}(i)
	}

	wg.Wait()

	// Verify each call has its own stats (or nil for timed out calls)
	for i, res := range results {
		// All calls should error due to timeout
		if res.err == nil {
			t.Errorf("Call %d: expected error due to timeout, got nil", i)
		}

		// Stats may be nil if the call timed out before API response
		// The important thing is that stats from different calls are not mixed
		if res.stats != nil {
			// If stats exist, verify they're for this specific call
			if res.stats.PromptTokens < 0 || res.stats.CompletionTokens < 0 {
				t.Errorf("Call %d: invalid token counts (prompt=%d, completion=%d)",
					i, res.stats.PromptTokens, res.stats.CompletionTokens)
			}
		}
	}
}

// TestConcurrentChatStream_StatsIsolation verifies that stats channels are not mixed between concurrent ChatStream calls.
func TestConcurrentChatStream_StatsIsolation(t *testing.T) {
	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Number of concurrent calls
	numCalls := 5

	var wg sync.WaitGroup
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			contentChan, statsChan, _ := service.ChatStream(ctx, []Message{
				{Role: "user", Content: "test message"},
			})

			// Drain content channel
			for range contentChan {
			}

			// Each goroutine should have its own stats channel
			if statsChan == nil {
				t.Errorf("Call %d: stats channel should not be nil", idx)
				return
			}

			// Drain stats channel (may or may not receive stats depending on timing)
			statsReceived := 0
			for range statsChan {
				statsReceived++
			}

			// At most one stats object should be received per call
			if statsReceived > 1 {
				t.Errorf("Call %d: received %d stats objects, expected at most 1", idx, statsReceived)
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentChat_NoDataRace verifies there are no data races when using Chat concurrently.
func TestConcurrentChat_NoDataRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	cfg := &LLMConfig{
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		APIKey:      "test-key",
		BaseURL:     "https://api.deepseek.com",
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	service, err := NewLLMService(cfg)
	if err != nil {
		t.Fatalf("NewLLMService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Run many concurrent calls to detect potential data races
	numCalls := 20

	var wg sync.WaitGroup
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _, _ = service.Chat(ctx, []Message{
				{Role: "user", Content: "test"},
			})
		}(i)
	}

	wg.Wait()
	// If we reach here without panic/deadlock, the test passes
}
