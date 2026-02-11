package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnthropicConfig tests Anthropic configuration parsing.
func TestAnthropicConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid anthropic config",
			cfg: Config{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "anthropic with custom base url",
			cfg: Config{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
				APIKey:   "test-key",
				BaseURL:  "https://custom.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "anthropic with opus model",
			cfg: Config{
				Provider: "anthropic",
				Model:    "claude-3-5-opus-20241022",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "anthropic with haiku model",
			cfg: Config{
				Provider: "anthropic",
				Model:    "claude-3-5-haiku-20241022",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "missing api key - creates service but will fail on API calls",
			cfg: Config{
				Provider: "anthropic",
				Model:    "claude-3-5-sonnet-20241022",
			},
			wantErr: false, // go-openai allows empty API key for testing; actual API calls will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService(&tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
	}
}

// TestAnthropicProviderDetection tests the provider() function for Anthropic models.
func TestAnthropicProviderDetection(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"claude sonnet", "claude-3-5-sonnet-20241022", "anthropic"},
		{"claude opus", "claude-3-5-opus-20241022", "anthropic"},
		{"claude haiku", "claude-3-5-haiku-20241022", "anthropic"},
		{"claude 3 sonnet", "claude-3-sonnet-20240229", "anthropic"},
		{"deepseek", "deepseek-chat", "deepseek"},
		{"gpt-4", "gpt-4", "openai"},
		{"qwen", "qwen-turbo", "siliconflow"},
		{"unknown", "unknown-model", "llm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service{model: tt.model}
			result := svc.provider()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAnthropicWarmup tests that Warmup doesn't panic for Anthropic.
func TestAnthropicWarmup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping warmup test in short mode")
	}

	svc, err := NewService(&Config{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
		APIKey:   "test-key",
	})
	require.NoError(t, err)

	// Warmup should not panic even with invalid API key
	// It will log a warning but not crash
	ctx := context.Background()
	assert.NotPanics(t, func() {
		svc.Warmup(ctx)
	})
}

// BenchmarkAnthropicServiceCreation benchmarks the creation of Anthropic service.
func BenchmarkAnthropicServiceCreation(b *testing.B) {
	cfg := &Config{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-20241022",
		APIKey:   "test-key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewService(cfg)
	}
}
