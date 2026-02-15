// Package context provides context building for LLM prompts.
package context

import "log/slog"

// Default token budget values.
//
// Token allocation strategy (with retrieval):
// - System Prompt: 500 tokens (fixed)
// - User Preferences: 10%
// - Short-term Memory: 40%
// - Long-term Memory: 15%
// - Retrieval Results: 45%
//
// Reference: docs/agent-engineering/AGENT_PROMPT_BEST_PRACTICES.md
const (
	DefaultMaxTokens      = 4096
	DefaultSystemPrompt   = 500
	DefaultUserPrefsRatio = 0.10
	DefaultRetrievalRatio = 0.45 // Updated per best practices
	MinSegmentTokens      = 100
)

// Dynamic budget adjustment thresholds (Issue #211: Phase 3)
const (
	// HistoryLengthThreshold is the number of turns after which dynamic adjustment kicks in
	// Triggers when historyLength > 20 (i.e., at 21+ turns)
	HistoryLengthThreshold = 20
	// ShortTermReductionRatio is how much to reduce ShortTerm when conversation is long
	ShortTermReductionRatio = 0.375 // Reduce from 40% to 25% (40% * 0.375 = 15% reduction)
	// LongTermIncreaseRatio is how much to increase LongTerm/Retrieval when conversation is long
	LongTermIncreaseRatio = 0.333 // Increase from 15% to 20% (15% * 0.333 ≈ 5% increase)
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
	profileRegistry    *ProfileRegistry
	intentResolver     *IntentResolver
}

// NewBudgetAllocator creates a new budget allocator with defaults.
func NewBudgetAllocator() *BudgetAllocator {
	return &BudgetAllocator{
		systemPromptTokens: DefaultSystemPrompt,
		userPrefsRatio:     DefaultUserPrefsRatio,
		retrievalRatio:     DefaultRetrievalRatio,
		profileRegistry:    NewProfileRegistry(),
		intentResolver:     NewIntentResolver(),
	}
}

// WithProfileRegistry sets a custom profile registry.
func (a *BudgetAllocator) WithProfileRegistry(registry *ProfileRegistry) *BudgetAllocator {
	a.profileRegistry = registry
	return a
}

// WithIntentResolver sets a custom intent resolver.
func (a *BudgetAllocator) WithIntentResolver(resolver *IntentResolver) *BudgetAllocator {
	a.intentResolver = resolver
	return a
}

// LoadProfileFromEnv loads profile overrides from environment variables.
func (a *BudgetAllocator) LoadProfileFromEnv() {
	a.profileRegistry.LoadFromEnv()
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

// AllocateForAgent allocates token budget based on agent type (profile-based).
// Issue #93: 动态 Token 预算分配 - 按意图类型自适应调整
func (a *BudgetAllocator) AllocateForAgent(total int, hasRetrieval bool, agentType string) *TokenBudget {
	if total <= 0 {
		total = DefaultMaxTokens
	}

	// Special cases: GEEK and EVOLUTION use Claude Code CLI with their own context management
	if agentType == "GEEK" || agentType == "EVOLUTION" {
		// These modes use Claude Code CLI, no LLM budget needed
		return &TokenBudget{
			Total:           total,
			SystemPrompt:    0,
			UserPrefs:       0,
			ShortTermMemory: 0,
			LongTermMemory:  0,
			Retrieval:       0,
		}
	}

	// Resolve intent from agent type
	intent := a.intentResolver.Resolve(agentType)
	profile, found := a.profileRegistry.Get(intent)
	if !found {
		// Unknown profile requested - using fallback profile
		// This is expected for new agent types that don't have custom budgets
		slog.Debug("Using fallback budget profile", "intent", intent, "fallback_profile", profile.Name, "agent_type", agentType)
	}

	budget := &TokenBudget{
		Total:        total,
		SystemPrompt: a.systemPromptTokens,
		UserPrefs:    int(float64(total) * profile.UserPrefsRatio),
	}

	remaining := total - budget.SystemPrompt - budget.UserPrefs

	// Use profile ratios for remaining budget
	budget.ShortTermMemory = int(float64(remaining) * profile.ShortTermRatio)
	budget.LongTermMemory = int(float64(remaining) * profile.LongTermRatio)

	if hasRetrieval {
		budget.Retrieval = int(float64(remaining) * profile.RetrievalRatio)
	} else {
		budget.Retrieval = 0
		// Redistribute retrieval budget to short-term and long-term
		extra := int(float64(remaining) * profile.RetrievalRatio)
		totalRatio := profile.ShortTermRatio + profile.LongTermRatio
		if totalRatio > 0 {
			budget.ShortTermMemory += int(float64(extra) * (profile.ShortTermRatio / totalRatio))
			budget.LongTermMemory += int(float64(extra) * (profile.LongTermRatio / totalRatio))
		}
	}

	return budget
}

// AllocateBudget is a convenience function.
func AllocateBudget(total int, hasRetrieval bool) *TokenBudget {
	return NewBudgetAllocator().Allocate(total, hasRetrieval)
}

// AllocateForAgentWithHistory allocates token budget based on agent type and conversation length.
// Issue #211: Phase 3 - Dynamic budget adjustment based on conversation turns
// When conversation exceeds 20 turns (historyLength > 20, i.e., 21+ turns), it compresses ShortTerm
// and increases LongTerm/Retrieval to handle long conversations more efficiently.
// Note: Has a maximum adjustment cap to prevent over-compression for extremely long conversations.
func (a *BudgetAllocator) AllocateForAgentWithHistory(total int, hasRetrieval bool, agentType string, historyLength int) *TokenBudget {
	budget := a.AllocateForAgent(total, hasRetrieval, agentType)

	// Apply dynamic adjustment only if conversation exceeds threshold
	if historyLength > HistoryLengthThreshold {
		slog.Debug("Applying dynamic budget adjustment for long conversation",
			"history_length", historyLength,
			"threshold", HistoryLengthThreshold)

		// Calculate effective adjustment factor with cap
		// Cap at 100 turns to prevent over-compression for extremely long conversations
		const maxAdjustmentTurns = 100
		effectiveTurns := historyLength
		if effectiveTurns > maxAdjustmentTurns {
			effectiveTurns = maxAdjustmentTurns
		}
		adjustmentFactor := float64(effectiveTurns-HistoryLengthThreshold) / float64(maxAdjustmentTurns-HistoryLengthThreshold)
		if adjustmentFactor > 1.0 {
			adjustmentFactor = 1.0
		}

		// Calculate reduction/increase amounts with cap
		shortTermReduction := int(float64(budget.ShortTermMemory) * ShortTermReductionRatio * adjustmentFactor)
		longTermIncrease := int(float64(budget.LongTermMemory) * LongTermIncreaseRatio * adjustmentFactor)
		retrievalIncrease := int(float64(budget.Retrieval) * LongTermIncreaseRatio * adjustmentFactor)

		// Apply adjustments
		budget.ShortTermMemory -= shortTermReduction
		budget.LongTermMemory += longTermIncrease

		// For retrieval, we increase it only if retrieval is enabled
		if hasRetrieval {
			budget.Retrieval += retrievalIncrease
		} else {
			// If no retrieval, redistribute to long-term memory
			budget.LongTermMemory += retrievalIncrease
		}

		// Ensure minimum values
		if budget.ShortTermMemory < MinSegmentTokens {
			budget.ShortTermMemory = MinSegmentTokens
		}
		if budget.LongTermMemory < MinSegmentTokens {
			budget.LongTermMemory = MinSegmentTokens
		}

		slog.Debug("Dynamic budget adjustment applied",
			"history_length", historyLength,
			"effective_turns", effectiveTurns,
			"adjustment_factor", adjustmentFactor,
			"short_term_before", budget.ShortTermMemory+shortTermReduction,
			"short_term_after", budget.ShortTermMemory,
			"long_term_before", budget.LongTermMemory-longTermIncrease,
			"long_term_after", budget.LongTermMemory)
	}

	return budget
}
