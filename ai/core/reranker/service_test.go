package reranker

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	cfg := &Config{
		Provider: "siliconflow",
		Model:    "test-model",
		APIKey:   "test-key",
		BaseURL:  "https://api.test.com",
		Enabled:  true,
	}

	svc := NewService(cfg)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	s, ok := svc.(*service)
	if !ok {
		t.Fatal("NewService() did not return *service type")
	}

	if s.model != "test-model" {
		t.Errorf("model = %v, want %v", s.model, "test-model")
	}
	if s.apiKey != "test-key" {
		t.Errorf("apiKey = %v, want %v", s.apiKey, "test-key")
	}
	if s.baseURL != "https://api.test.com" {
		t.Errorf("baseURL = %v, want %v", s.baseURL, "https://api.test.com")
	}
	if !s.enabled {
		t.Error("enabled = false, want true")
	}
}

func TestService_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"enabled service", true, true},
		{"disabled service", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&Config{Enabled: tt.enabled})
			if got := svc.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestService_Rerank_Disabled(t *testing.T) {
	svc := NewService(&Config{Enabled: false})

	docs := []string{"doc1", "doc2", "doc3"}
	results, err := svc.Rerank(context.Background(), "query", docs, 2)

	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Rerank() returned %d results, want 2", len(results))
	}

	// Should preserve original order when disabled
	for i, r := range results {
		if r.Index != i {
			t.Errorf("results[%d].Index = %d, want %d", i, r.Index, i)
		}
	}
}

func TestService_Rerank_TopN_Limit(t *testing.T) {
	svc := NewService(&Config{Enabled: false})

	docs := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}

	tests := []struct {
		name     string
		topN     int
		expected int
	}{
		{"top 2", 2, 2},
		{"top 10", 10, 5}, // More than available
		{"top 0", 0, 5},   // No limit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := svc.Rerank(context.Background(), "query", docs, tt.topN)
			if err != nil {
				t.Fatalf("Rerank() error = %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Rerank() returned %d results, want %d", len(results), tt.expected)
			}
		})
	}
}

func TestService_Rerank_ScoreDecrement(t *testing.T) {
	svc := NewService(&Config{Enabled: false})

	docs := []string{"doc1", "doc2", "doc3"}
	results, err := svc.Rerank(context.Background(), "query", docs, 0)

	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	// Check that scores decrease
	for i := 1; i < len(results); i++ {
		if results[i].Score >= results[i-1].Score {
			t.Errorf("results[%d].Score (%f) >= results[%d].Score (%f), want decreasing",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestResult(t *testing.T) {
	r := Result{Index: 5, Score: 0.95}

	if r.Index != 5 {
		t.Errorf("Index = %d, want 5", r.Index)
	}
	if r.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", r.Score)
	}
}

func TestNewService_DefaultConfig(t *testing.T) {
	// Test with empty config (zero values)
	svc := NewService(&Config{})

	if svc == nil {
		t.Fatal("NewService(&Config{}) returned nil")
	}

	// Should return disabled service when enabled is false
	if svc.IsEnabled() {
		t.Error("NewService(&Config{Enabled: false}) should return disabled service")
	}
}

func TestService_Rerank_EmptyDocs(t *testing.T) {
	svc := NewService(&Config{Enabled: false})

	results, err := svc.Rerank(context.Background(), "query", []string{}, 0)

	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Rerank() with empty docs returned %d results, want 0", len(results))
	}
}
