package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetAllocator_Allocate(t *testing.T) {
	allocator := NewBudgetAllocator()

	t.Run("With retrieval", func(t *testing.T) {
		budget := allocator.Allocate(4096, true)

		assert.Equal(t, 4096, budget.Total)
		assert.Equal(t, DefaultSystemPrompt, budget.SystemPrompt)
		assert.Greater(t, budget.Retrieval, 0)
		assert.Greater(t, budget.ShortTermMemory, 0)
		assert.Greater(t, budget.LongTermMemory, 0)
		assert.Greater(t, budget.UserPrefs, 0)

		// Verify total equals sum of parts (within rounding tolerance)
		total := budget.SystemPrompt + budget.ShortTermMemory +
			budget.LongTermMemory + budget.Retrieval + budget.UserPrefs
		assert.LessOrEqual(t, total, budget.Total)
	})

	t.Run("Without retrieval", func(t *testing.T) {
		budget := allocator.Allocate(4096, false)

		assert.Equal(t, 4096, budget.Total)
		assert.Equal(t, 0, budget.Retrieval)

		// Short-term and long-term should be larger without retrieval
		assert.Greater(t, budget.ShortTermMemory, 0)
		assert.Greater(t, budget.LongTermMemory, 0)
	})

	t.Run("Zero total uses default", func(t *testing.T) {
		budget := allocator.Allocate(0, true)

		assert.Equal(t, DefaultMaxTokens, budget.Total)
	})

	t.Run("Small budget", func(t *testing.T) {
		budget := allocator.Allocate(1000, true)

		assert.Equal(t, 1000, budget.Total)
		assert.Equal(t, DefaultSystemPrompt, budget.SystemPrompt)

		// Remaining budget after system prompt
		remaining := 1000 - DefaultSystemPrompt
		assert.Greater(t, budget.Retrieval, 0)
		assert.LessOrEqual(t, budget.Retrieval, remaining)
	})
}

func TestBudgetAllocator_AllocateForAgent(t *testing.T) {
	allocator := NewBudgetAllocator()

	t.Run("GEEK agent - no budget needed", func(t *testing.T) {
		budget := allocator.AllocateForAgent(4096, false, "GEEK")

		assert.Equal(t, 4096, budget.Total)
		assert.Equal(t, 0, budget.SystemPrompt)
		assert.Equal(t, 0, budget.UserPrefs)
		assert.Equal(t, 0, budget.ShortTermMemory)
		assert.Equal(t, 0, budget.LongTermMemory)
		assert.Equal(t, 0, budget.Retrieval)
	})

	t.Run("EVOLUTION agent - no budget needed", func(t *testing.T) {
		budget := allocator.AllocateForAgent(4096, true, "EVOLUTION")

		assert.Equal(t, 4096, budget.Total)
		assert.Equal(t, 0, budget.SystemPrompt)
		assert.Equal(t, 0, budget.Retrieval)
	})

	t.Run("MEMO agent with retrieval", func(t *testing.T) {
		budget := allocator.AllocateForAgent(4096, true, "MEMO")

		assert.Equal(t, 4096, budget.Total)
		assert.Greater(t, budget.Retrieval, 0)
		assert.Greater(t, budget.ShortTermMemory, 0)
	})

	t.Run("SCHEDULE agent without retrieval", func(t *testing.T) {
		budget := allocator.AllocateForAgent(4096, false, "SCHEDULE")

		assert.Equal(t, 4096, budget.Total)
		assert.Equal(t, 0, budget.Retrieval)
		assert.Greater(t, budget.ShortTermMemory, 0)
	})

	t.Run("Unknown agent uses default profile", func(t *testing.T) {
		budget := allocator.AllocateForAgent(4096, true, "UNKNOWN")

		assert.Equal(t, 4096, budget.Total)
		assert.Greater(t, budget.SystemPrompt, 0)
	})

	t.Run("Zero total uses default", func(t *testing.T) {
		budget := allocator.AllocateForAgent(0, false, "MEMO")

		assert.Equal(t, DefaultMaxTokens, budget.Total)
	})
}

func TestBudgetAllocator_WithProfileRegistry(t *testing.T) {
	allocator := NewBudgetAllocator()
	registry := NewProfileRegistry()

	allocator.WithProfileRegistry(registry)

	assert.NotNil(t, allocator.profileRegistry)
	assert.Same(t, registry, allocator.profileRegistry)
}

func TestBudgetAllocator_WithIntentResolver(t *testing.T) {
	allocator := NewBudgetAllocator()
	resolver := NewIntentResolver()

	allocator.WithIntentResolver(resolver)

	assert.NotNil(t, allocator.intentResolver)
	assert.Same(t, resolver, allocator.intentResolver)
}

func TestAllocateBudget_ConvenienceFunction(t *testing.T) {
	t.Run("With retrieval", func(t *testing.T) {
		budget := AllocateBudget(8192, true)

		assert.Equal(t, 8192, budget.Total)
		assert.Greater(t, budget.Retrieval, 0)
	})

	t.Run("Without retrieval", func(t *testing.T) {
		budget := AllocateBudget(8192, false)

		assert.Equal(t, 8192, budget.Total)
		assert.Equal(t, 0, budget.Retrieval)
	})
}

func TestTokenBudget_Ratios(t *testing.T) {
	allocator := NewBudgetAllocator()

	t.Run("Verify retrieval ratios", func(t *testing.T) {
		budget := allocator.Allocate(10000, true)

		remaining := budget.Total - budget.SystemPrompt - budget.UserPrefs

		// With retrieval: Short-term 40%, Long-term 15%, Retrieval 45%
		expectedShortTerm := int(float64(remaining) * 0.40)
		expectedLongTerm := int(float64(remaining) * 0.15)
		expectedRetrieval := int(float64(remaining) * 0.45)

		assert.InDelta(t, expectedShortTerm, budget.ShortTermMemory, 1)
		assert.InDelta(t, expectedLongTerm, budget.LongTermMemory, 1)
		assert.InDelta(t, expectedRetrieval, budget.Retrieval, 1)
	})

	t.Run("Verify non-retrieval ratios", func(t *testing.T) {
		budget := allocator.Allocate(10000, false)

		remaining := budget.Total - budget.SystemPrompt - budget.UserPrefs

		// Without retrieval: Short-term 55%, Long-term 30%
		expectedShortTerm := int(float64(remaining) * 0.55)
		expectedLongTerm := int(float64(remaining) * 0.30)

		assert.InDelta(t, expectedShortTerm, budget.ShortTermMemory, 1)
		assert.InDelta(t, expectedLongTerm, budget.LongTermMemory, 1)
		assert.Equal(t, 0, budget.Retrieval)
	})
}

func TestTokenBudget_Redistribution(t *testing.T) {
	allocator := NewBudgetAllocator()

	t.Run("Retrieval budget redistributed when disabled", func(t *testing.T) {
		budgetWithRetrieval := allocator.Allocate(10000, true)
		budgetWithout := allocator.Allocate(10000, false)

		// Without retrieval, short-term and long-term should be larger
		assert.Greater(t, budgetWithout.ShortTermMemory, budgetWithRetrieval.ShortTermMemory)
		assert.Greater(t, budgetWithout.LongTermMemory, budgetWithRetrieval.LongTermMemory)
	})
}

func TestBudgetAllocator_LoadProfileFromEnv(t *testing.T) {
	allocator := NewBudgetAllocator()

	// Should not panic
	allocator.LoadProfileFromEnv()
}

// Benchmark_Allocate benchmarks budget allocation.
func Benchmark_Allocate(b *testing.B) {
	allocator := NewBudgetAllocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = allocator.Allocate(4096, true)
	}
}

// Benchmark_AllocateForAgent benchmarks agent-specific budget allocation.
func Benchmark_AllocateForAgent(b *testing.B) {
	allocator := NewBudgetAllocator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = allocator.AllocateForAgent(4096, true, "MEMO")
	}
}
