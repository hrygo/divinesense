// Package context provides context building for LLM prompts.
package context

// Default token budget values.
const (
	DefaultMaxTokens      = 4096
	DefaultSystemPrompt   = 500
	DefaultUserPrefsRatio = 0.10
	DefaultRetrievalRatio = 0.35
	MinSegmentTokens      = 100
)

// TokenBudget represents the token allocation plan.
type TokenBudget struct {
	Total           int
	SystemPrompt    int
	ShortTermMemory int
	LongTermMemory  int
	Retrieval       int
	UserPrefs       int
}

// BudgetAllocator allocates token budgets.
type BudgetAllocator struct {
	systemPromptTokens int
	userPrefsRatio     float64
	retrievalRatio     float64
}

// NewBudgetAllocator creates a new budget allocator with defaults.
func NewBudgetAllocator() *BudgetAllocator {
	return &BudgetAllocator{
		systemPromptTokens: DefaultSystemPrompt,
		userPrefsRatio:     DefaultUserPrefsRatio,
		retrievalRatio:     DefaultRetrievalRatio,
	}
}

// Allocate allocates token budget based on total and whether retrieval is needed.
func (a *BudgetAllocator) Allocate(total int, hasRetrieval bool) *TokenBudget {
	if total <= 0 {
		total = DefaultMaxTokens
	}

	budget := &TokenBudget{
		Total:        total,
		SystemPrompt: a.systemPromptTokens,
		UserPrefs:    int(float64(total) * a.userPrefsRatio),
	}

	remaining := total - budget.SystemPrompt - budget.UserPrefs

	if hasRetrieval {
		// With retrieval: prioritize retrieval context
		// Short-term: 40%, Long-term: 15%, Retrieval: 45%
		budget.ShortTermMemory = int(float64(remaining) * 0.40)
		budget.LongTermMemory = int(float64(remaining) * 0.15)
		budget.Retrieval = int(float64(remaining) * 0.45)
	} else {
		// No retrieval: more space for memory
		// Short-term: 55%, Long-term: 30%, Retrieval: 0%
		budget.ShortTermMemory = int(float64(remaining) * 0.55)
		budget.LongTermMemory = int(float64(remaining) * 0.30)
		budget.Retrieval = 0
	}

	return budget
}

// AllocateBudget is a convenience function.
func AllocateBudget(total int, hasRetrieval bool) *TokenBudget {
	return NewBudgetAllocator().Allocate(total, hasRetrieval)
}
