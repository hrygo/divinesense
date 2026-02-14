package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoOpGenerator(t *testing.T) {
	gen := NewNoOpGenerator()

	t.Run("GenerateAsync does nothing", func(t *testing.T) {
		// Should not panic or block
		gen.GenerateAsync(context.Background(), MemoryRequest{
			BlockID:   1,
			UserID:    100,
			AgentType: "memo",
			UserInput: "test",
			Outcome:   "response",
		})
	})

	t.Run("GenerateSync returns nil", func(t *testing.T) {
		err := gen.GenerateSync(context.Background(), MemoryRequest{})
		assert.NoError(t, err)
	})

	t.Run("Shutdown returns nil", func(t *testing.T) {
		err := gen.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

func TestNoOpGenerator_ImplementsInterface(t *testing.T) {
	// Ensure NoOpGenerator implements Generator interface
	var _ Generator = (*NoOpGenerator)(nil)
}
