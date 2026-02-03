package ai

import (
	"context"
	"testing"
)

// TestNewRerankerService tests service creation.
func TestNewRerankerService(t *testing.T) {
	cfg := &RerankerConfig{
		Enabled:  true,
		Provider: "siliconflow",
		Model:    "BAAI/bge-reranker-v2-m3",
		APIKey:   "test-key",
		BaseURL:  "https://api.siliconflow.cn/v1",
	}

	service := NewRerankerService(cfg)
	if service == nil {
		t.Fatal("NewRerankerService() returned nil")
	}

	if !service.IsEnabled() {
		t.Error("Expected IsEnabled()=true, got false")
	}
}

// TestRerankerService_Disabled tests disabled reranker behavior.
func TestRerankerService_Disabled(t *testing.T) {
	cfg := &RerankerConfig{
		Enabled: false,
	}

	service := NewRerankerService(cfg).(*rerankerService)

	documents := []string{"doc1", "doc2", "doc3"}
	results, err := service.Rerank(context.Background(), "test query", documents, 2)

	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that scores are in descending order (original order with slight decay)
	for i := 1; i < len(results); i++ {
		if results[i].Score >= results[i-1].Score {
			t.Errorf("Scores not in descending order: [%d]=%f >= [%d]=%f",
				i-1, results[i-1].Score, i, results[i].Score)
		}
	}

	// Check that indices are in original order
	for i, r := range results {
		if r.Index != i {
			t.Errorf("Expected Index=%d, got %d", i, r.Index)
		}
	}
}

// TestRerankerService_IsEnabled tests IsEnabled method.
func TestRerankerService_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"Enabled", true},
		{"Disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RerankerConfig{
				Enabled: tt.enabled,
			}
			service := NewRerankerService(cfg)
			if service.IsEnabled() != tt.enabled {
				t.Errorf("IsEnabled() = %v, want %v", service.IsEnabled(), tt.enabled)
			}
		})
	}
}
