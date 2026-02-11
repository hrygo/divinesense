// Package universal provides the core abstractions for UniversalParrot.
// It defines execution strategies that can be used interchangeably.
package universal

import (
	"context"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

// ExecutionStrategy defines how a parrot processes user input.
// Different strategies enable different execution modes:
// - ReAct: Loop-based reasoning with tool calls
// - Direct: Native tool calling (modern LLMs)
// - Planning: Two-phase planning + execution
//
// The strategy pattern allows UniversalParrot to support multiple
// execution modes without code duplication.
type ExecutionStrategy interface {
	// Name returns the strategy name for logging and debugging.
	Name() string

	// Execute runs the strategy with given inputs.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - input: User input string
	//   - history: Conversation history as AI messages
	//   - tools: Available tools for this execution
	//   - llm: LLM service for generating responses
	//   - callback: Event callback for real-time UI updates
	//   - timeContext: Structured time context for time-aware reasoning (may be nil)
	//
	// Returns:
	//   - result: Final response string
	//   - stats: Execution statistics
	//   - error: Any error that occurred
	Execute(
		ctx context.Context,
		input string,
		history []ai.Message,
		tools []agent.ToolWithSchema,
		llm ai.LLMService,
		callback agent.EventCallback,
		timeContext *TimeContext,
	) (result string, stats *ExecutionStats, err error)

	// StreamingSupported indicates if this strategy supports streaming responses.
	// If false, the strategy will use simulated streaming (chunking the final response).
	StreamingSupported() bool
}

// ExecutionStats tracks metrics for a single execution.
// These statistics are accumulated into BaseParrot's session stats.
type ExecutionStats struct {
	Strategy string

	// LLM metrics
	LLMCalls         int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CacheReadTokens  int
	CacheWriteTokens int

	// Tool metrics
	ToolCalls int

	// Timing (milliseconds)
	TotalDurationMs  int64
	ThinkingDuration int64
	ToolDurationMs   int64
}

// AccumulateLLM adds LLM call statistics to this execution stats.
// This is the single source of truth for accumulating LLM metrics.
func (s *ExecutionStats) AccumulateLLM(llmStats *ai.LLMCallStats) {
	s.LLMCalls++
	s.PromptTokens += llmStats.PromptTokens
	s.CompletionTokens += llmStats.CompletionTokens
	s.TotalTokens += llmStats.TotalTokens
	s.CacheReadTokens += llmStats.CacheReadTokens
	s.CacheWriteTokens += llmStats.CacheWriteTokens
}

// StrategyType defines the supported execution strategies.
type StrategyType string

const (
	// StrategyReAct uses loop-based reasoning with tool calls.
	// The LLM generates thoughts and tool calls in a loop until
	// a final answer is reached.
	StrategyReAct StrategyType = "react"

	// StrategyDirect uses native LLM tool calling.
	// The LLM returns structured tool calls that are executed directly.
	// This is faster and more reliable for modern LLMs.
	StrategyDirect StrategyType = "direct"

	// StrategyPlanning uses two-phase planning + execution.
	// First, the LLM plans which tools to use. Then, tools are executed
	// concurrently. Finally, the LLM synthesizes the results.
	StrategyPlanning StrategyType = "planning"

	// StrategyReflexion uses self-reflection and refinement.
	// The LLM generates an initial response, reflects on quality,
	// and refines if needed. Ideal for high-quality requirements.
	StrategyReflexion StrategyType = "reflexion"
)

// Resolver creates an ExecutionStrategy from a StrategyType.
type Resolver interface {
	Resolve(strategyType StrategyType) (ExecutionStrategy, error)
}

// DefaultResolver is the default strategy resolver.
// All strategies are created with streaming enabled by default.
type DefaultResolver struct {
	// MaxIterations is the maximum number of ReAct iterations.
	MaxIterations int
}

// NewDefaultResolver creates a new DefaultResolver.
func NewDefaultResolver(maxIterations int) *DefaultResolver {
	return &DefaultResolver{
		MaxIterations: maxIterations,
	}
}

// Resolve creates an ExecutionStrategy for the given type.
// All strategies are created with streaming enabled for better UX.
func (r *DefaultResolver) Resolve(strategyType StrategyType) (ExecutionStrategy, error) {
	switch strategyType {
	case StrategyReAct:
		return NewReActExecutor(r.MaxIterations), nil
	case StrategyDirect:
		return NewDirectExecutor(r.MaxIterations), nil
	case StrategyPlanning:
		return NewPlanningExecutor(r.MaxIterations), nil
	case StrategyReflexion:
		return NewReflexionExecutor(r.MaxIterations), nil
	default:
		return nil, &UnsupportedStrategyError{Strategy: string(strategyType)}
	}
}

// UnsupportedStrategyError is returned when an unknown strategy is requested.
type UnsupportedStrategyError struct {
	Strategy string
}

func (e *UnsupportedStrategyError) Error() string {
	return "unsupported strategy: " + e.Strategy
}

// Ensure UnsupportedStrategyError implements error at compile time.
var _ error = (*UnsupportedStrategyError)(nil) // nolint:errcheck // compile-time check
