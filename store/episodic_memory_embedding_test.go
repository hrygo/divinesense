package store

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpisodicVectorSearchOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *EpisodicVectorSearchOptions
		wantErr bool
		errMsg  string
	}{
		{"valid defaults", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}}, false, ""},
		{"UserID <= 0", &EpisodicVectorSearchOptions{UserID: 0, Vector: []float32{0.1}}, true, "invalid UserID"},
		{"UserID negative", &EpisodicVectorSearchOptions{UserID: -1, Vector: []float32{0.1}}, true, "invalid UserID"},
		{"empty Vector", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{}}, true, "vector cannot be empty"},
		{"nil Vector", &EpisodicVectorSearchOptions{UserID: 1, Vector: nil}, true, "vector cannot be empty"},
		{"Limit negative", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: -1}, true, "limit cannot be negative"},
		{"Limit zero sets default", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: 0}, false, ""},
		{"Limit > 1000", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: 1001}, true, "limit too large"},
		{"Limit == 1000", &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: 1000}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.errMsg),
					"expected error to contain %q, got %q", tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEpisodicVectorSearchOptions_Validate_SetsDefaultLimit(t *testing.T) {
	opts := &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: 0}

	err := opts.Validate()

	require.NoError(t, err)
	assert.Equal(t, 10, opts.Limit, "Limit should be set to default value 10")
}

func TestEpisodicVectorSearchOptions_Validate_PreservesValidLimit(t *testing.T) {
	opts := &EpisodicVectorSearchOptions{UserID: 1, Vector: []float32{0.1}, Limit: 50}

	err := opts.Validate()

	require.NoError(t, err)
	assert.Equal(t, 50, opts.Limit, "Limit should remain unchanged when already set")
}
