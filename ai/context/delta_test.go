package context

import (
	"context"
	"testing"
	"time"
)

func TestDeltaBuilder(t *testing.T) {
	builder := NewDeltaBuilder()

	t.Run("ComputeDelta_FirstBuild", func(t *testing.T) {
		req := &ContextRequest{
			SessionID:    "test-session",
			CurrentQuery: "Hello",
			AgentType:    "memo",
		}

		delta := builder.ComputeDelta("test-session", req, nil)

		if delta.Strategy != FullRebuild {
			t.Errorf("expected FullRebuild strategy, got %v", delta.Strategy)
		}
	})

	t.Run("ComputeDelta_WithPrevious", func(t *testing.T) {
		now := time.Now()

		prevSnap := &ContextSnapshot{
			Query: hashString("Hello"),
			RetrievalResults: []*RetrievalItem{
				{ID: "1", Content: "Result 1", Score: 0.9},
			},
			Timestamp: now,
		}

		req := &ContextRequest{
			SessionID:    "test-session",
			CurrentQuery: "How are you?",
			AgentType:    "memo",
		}

		delta := builder.ComputeDelta("test-session", req, prevSnap)

		if delta.Strategy != ComputeDelta {
			t.Errorf("expected ComputeDelta strategy, got %v", delta.Strategy)
		}
	})

	t.Run("SelectStrategy", func(t *testing.T) {
		now := time.Now()

		t.Run("NoPrevious", func(t *testing.T) {
			req := &ContextRequest{
				SessionID: "test-session",
				AgentType: "memo",
			}

			strategy := builder.SelectStrategy(req, nil)
			if strategy != FullRebuild {
				t.Errorf("expected FullRebuild, got %v", strategy)
			}
		})

		t.Run("AppendOnly", func(t *testing.T) {
			prevSnap := &ContextSnapshot{
				Query:            hashString("Hello"),
				SystemPromptHash: hashString("memo"),
				Timestamp:        now,
			}

			req := &ContextRequest{
				SessionID:    "test-session",
				CurrentQuery: "Hello",
				AgentType:    "memo",
				RetrievalResults: []*RetrievalItem{
					{ID: "1", Content: "Result 1", Score: 0.9},
				},
			}

			strategy := builder.SelectStrategy(req, prevSnap)
			if strategy != ComputeDelta {
				t.Errorf("expected ComputeDelta, got %v", strategy)
			}
		})
	})

	t.Run("SaveAndRetrieveSnapshot", func(t *testing.T) {
		snap := &ContextSnapshot{
			Query:     hashString("test"),
			Timestamp: time.Now(),
		}

		builder.SaveSnapshot("test-session", snap)
		retrieved := builder.GetSnapshot("test-session")

		if retrieved == nil {
			t.Fatal("expected non-nil snapshot")
		}

		if retrieved.Query != snap.Query {
			t.Errorf("expected query %s, got %s", snap.Query, retrieved.Query)
		}
	})

	t.Run("CacheEviction", func(t *testing.T) {
		// Create a builder with small cache size
		smallBuilder := &DeltaBuilder{
			cache:        make(map[string]*ContextSnapshot),
			maxCacheSize: 2,
		}

		snap1 := &ContextSnapshot{Timestamp: time.Now()}
		snap2 := &ContextSnapshot{Timestamp: time.Now()}
		snap3 := &ContextSnapshot{Timestamp: time.Now()}

		smallBuilder.SaveSnapshot("session1", snap1)
		smallBuilder.SaveSnapshot("session2", snap2)
		smallBuilder.SaveSnapshot("session3", snap3) // Should evict session1

		if smallBuilder.cache["session1"] != nil {
			t.Error("expected session1 to be evicted")
		}
		if smallBuilder.cache["session2"] == nil {
			t.Error("expected session2 to remain")
		}
		if smallBuilder.cache["session3"] == nil {
			t.Error("expected session3 to be added")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		builder.SaveSnapshot("test", &ContextSnapshot{Timestamp: time.Now()})

		if len(builder.cache) == 0 {
			t.Error("expected cache to have entries")
		}

		builder.Clear()

		if len(builder.cache) != 0 {
			t.Errorf("expected empty cache after clear, got %d entries", len(builder.cache))
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := builder.GetStats()

		if stats == nil {
			t.Fatal("expected non-nil stats")
		}

		// New builder should have zero stats
		if stats.TotalDeltas != 0 {
			t.Errorf("expected 0 total deltas, got %d", stats.TotalDeltas)
		}
	})
}

func TestCreateSnapshot(t *testing.T) {

	req := &ContextRequest{
		SessionID:    "test-session",
		AgentType:    "memo",
		CurrentQuery: "Hello",
		RetrievalResults: []*RetrievalItem{
			{ID: "1", Content: "Result 1", Score: 0.9},
		},
	}

	result := &ContextResult{
		SystemPrompt:   "You are a helpful assistant.",
		TotalTokens:    1000,
		TokenBreakdown: &TokenBreakdown{},
		BuildTime:      50 * time.Millisecond,
	}

	snap := CreateSnapshot(req, result)

	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	if len(snap.RetrievalResults) != 1 {
		t.Errorf("expected 1 retrieval result, got %d", len(snap.RetrievalResults))
	}

	if snap.TokenCount != 1000 {
		t.Errorf("expected 1000 tokens, got %d", snap.TokenCount)
	}

	if snap.SystemPromptHash == "" {
		t.Error("expected non-empty system prompt hash")
	}
}

func TestDeltaToJSON(t *testing.T) {
	delta := &Delta{
		ModifiedSections:  []string{"short_term_memory"},
		RemovedSections:   []string{},
		NewRetrievalItems: []*RetrievalItem{},
		Strategy:          ComputeDelta,
		CurrentHash:       "abc123",
		PreviousHash:      "def456",
	}

	jsonStr, err := delta.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}
}

func TestSnapshotToJSON(t *testing.T) {
	snap := &ContextSnapshot{
		Query:            hashString("test"),
		Timestamp:        time.Now(),
		SystemPromptHash: "hash123",
	}

	jsonStr, err := snap.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}
}

func TestUpdateStrategyString(t *testing.T) {
	tests := []struct {
		strategy UpdateStrategy
		expected string
	}{
		{ComputeDelta, "compute_delta"},
		{AppendOnly, "append_only"},
		{UpdateConversationOnly, "update_conversation_only"},
		{FullRebuild, "full_rebuild"},
		{UpdateStrategy(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIncrementalBuilder(t *testing.T) {
	base := NewService(DefaultConfig())
	builder := NewIncrementalBuilder(base)

	t.Run("BuildIncremental", func(t *testing.T) {
		req := &ContextRequest{
			SessionID:    "test-session",
			AgentType:    "memo",
			CurrentQuery: "Hello",
		}

		result, err := builder.BuildIncremental(context.Background(), req)
		if err != nil {
			t.Fatalf("BuildIncremental failed: %v", err)
		}

		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("ClearCache", func(t *testing.T) {
		builder.ClearCache()
		stats := builder.GetDeltaStats()
		if stats == nil {
			t.Error("expected non-nil stats after clear")
		}
	})

	t.Run("GetDeltaStats", func(t *testing.T) {
		stats := builder.GetDeltaStats()
		if stats == nil {
			t.Fatal("expected non-nil stats")
		}
	})
}

func BenchmarkDeltaBuilder(b *testing.B) {
	builder := NewDeltaBuilder()

	req := &ContextRequest{
		SessionID: "test-session",
		AgentType: "memo",
	}

	b.Run("ComputeDelta", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder.ComputeDelta("test-session", req, nil)
		}
	})

	b.Run("SelectStrategy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder.SelectStrategy(req, nil)
		}
	})
}
