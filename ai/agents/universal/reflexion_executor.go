// Package universal provides Reflexion execution strategy.
package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
)

/*
ReflexionExecutor - Self-Reflection and Refinement

⚠️ EXPERIMENTAL: This strategy is under active development and may change.
Not yet enabled in production parrot configurations.

POSITIONING:

ReflexionExecutor implements the Reflexion pattern for high-quality responses.
The LLM generates an initial response, reflects on quality, and refines if needed.

ALGORITHM:
  1. Generate initial response using ReAct strategy
  2. Reflect on response quality (accuracy, completeness, clarity)
  3. If quality below threshold, refine response
  4. Repeat until quality threshold met or max iterations reached

USE CASES:
  - High-quality requirements (reports, analysis)
  - Complex reasoning tasks
  - Content generation where accuracy is critical

TRADEOFFS:
  - Higher latency (2-3x initial response time)
  - Higher token cost (additional LLM calls)
  - Better output quality

ENABLING IN PRODUCTION:

To enable reflexion strategy for a parrot, set in config/parrots/*.yaml:
  strategy: reflexion
  max_iterations: 3

Recommended use cases:
  - amazing: For complex multi-tool queries requiring high accuracy
  - memo: For critical research tasks
  - schedule: For complex scheduling with conflicts
*/

const (
	defaultQualityThreshold = 0.8
	defaultMaxRefinements   = 2
)

// ReflectionReport represents the structured output from reflection phase.
type ReflectionReport struct {
	Accuracy        float64  `json:"accuracy"`     // Information accuracy (0.0-1.0)
	Completeness    float64  `json:"completeness"` // Content completeness (0.0-1.0)
	Clarity         float64  `json:"clarity"`      // Expression clarity (0.0-1.0)
	Issues          []string `json:"issues"`       // Identified problems
	Suggestions     []string `json:"suggestions"`  // Improvement suggestions
	NeedsRefinement bool     `json:"needs_refinement"`
}

// ReflexionExecutor implements self-reflection and refinement.
type ReflexionExecutor struct {
	maxIterations      int
	qualityThreshold   float64
	reflectionPrompt   string
	refinePrompt       string
	underlyingStrategy ExecutionStrategy // Uses ReAct for initial response
}

// NewReflexionExecutor creates a new ReflexionExecutor.
func NewReflexionExecutor(maxIterations int) *ReflexionExecutor {
	if maxIterations <= 0 {
		maxIterations = defaultMaxRefinements + 1 // 1 initial + N refinements
	}
	// Use same maxIterations for underlying ReAct strategy
	// Note: Reflexion loop controls refinement iterations (initial + N refinements)
	//       ReAct strategy controls reasoning iterations per response generation
	//       These are independent budgets - both can use up to maxIterations
	return &ReflexionExecutor{
		maxIterations:      maxIterations,
		qualityThreshold:   defaultQualityThreshold,
		reflectionPrompt:   defaultReflectionPrompt,
		refinePrompt:       defaultRefinePrompt,
		underlyingStrategy: NewReActExecutor(maxIterations),
	}
}

// Name returns the strategy name.
func (e *ReflexionExecutor) Name() string {
	return "reflexion"
}

// Execute runs the Reflexion pattern.
func (e *ReflexionExecutor) Execute(
	ctx context.Context,
	input string,
	history []ai.Message,
	tools []agent.ToolWithSchema,
	llm ai.LLMService,
	callback agent.EventCallback,
	timeContext *TimeContext,
) (string, *ExecutionStats, error) {
	stats := &ExecutionStats{Strategy: "reflexion"}
	startTime := time.Now()
	defer func() {
		stats.TotalDurationMs = time.Since(startTime).Milliseconds()
	}()

	safeCallback := agent.SafeCallback(callback)

	// Phase 1: Initial Response
	safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
		EventType: agent.EventTypeThinking,
		EventData: "Generating initial response...",
		Meta: &agent.EventMeta{
			CurrentStep:     1,
			TotalSteps:      3,
			TotalDurationMs: time.Since(startTime).Milliseconds(),
		},
	})

	// Use underlying ReAct strategy for initial response
	initialAnswer, underlyingStats, err := e.underlyingStrategy.Execute(
		ctx, input, history, tools, llm, callback, timeContext,
	)
	if err != nil {
		return "", stats, fmt.Errorf("initial response: %w", err)
	}
	// Accumulate underlying stats
	stats.LLMCalls += underlyingStats.LLMCalls
	stats.PromptTokens += underlyingStats.PromptTokens
	stats.CompletionTokens += underlyingStats.CompletionTokens
	stats.TotalTokens += underlyingStats.TotalTokens
	stats.CacheReadTokens += underlyingStats.CacheReadTokens
	stats.CacheWriteTokens += underlyingStats.CacheWriteTokens
	stats.ToolCalls = underlyingStats.ToolCalls

	currentAnswer := initialAnswer
	iteration := 0

	// Reflexion Loop
	for iteration < e.maxIterations-1 {
		iteration++

		// Phase 2: Reflection
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			EventData: fmt.Sprintf("Reflecting on quality (iteration %d/%d)...", iteration, e.maxIterations-1),
			Meta: &agent.EventMeta{
				CurrentStep:     2,
				TotalSteps:      3,
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})

		reflection, err := e.reflect(ctx, input, currentAnswer, llm, stats, startTime)
		if err != nil {
			slog.Warn("Reflection failed, using current answer", "error", err)
			break
		}

		// Calculate overall quality
		overallQuality := e.calculateOverallQuality(reflection)

		// Phase 3: Quality Check
		if overallQuality >= e.qualityThreshold || !reflection.NeedsRefinement {
			slog.Info("Quality threshold met", "quality", overallQuality, "threshold", e.qualityThreshold)
			break
		}

		if iteration >= e.maxIterations-1 {
			slog.Info("Max iterations reached", "quality", overallQuality)
			break
		}

		// Phase 4: Refinement
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			EventData: fmt.Sprintf("Refining response (quality: %.0f%% → improving)...", overallQuality*100),
			Meta: &agent.EventMeta{
				CurrentStep:     3,
				TotalSteps:      3,
				TotalDurationMs: time.Since(startTime).Milliseconds(),
			},
		})

		refinedAnswer, err := e.refine(ctx, input, currentAnswer, reflection, llm, stats, startTime)
		if err != nil {
			slog.Warn("Refinement failed, using current answer", "error", err)
			break
		}

		currentAnswer = refinedAnswer
		safeCallback(agent.EventTypeThinking, &agent.EventWithMeta{
			EventType: agent.EventTypeThinking,
			EventData: fmt.Sprintf("✓ Response improved (iteration %d)", iteration),
		})
	}

	return currentAnswer, stats, nil
}

// StreamingSupported returns true - Reflexion supports streaming.
func (e *ReflexionExecutor) StreamingSupported() bool {
	return true
}

// reflect generates a reflection report on the current answer.
func (e *ReflexionExecutor) reflect(
	ctx context.Context,
	input string,
	answer string,
	llm ai.LLMService,
	stats *ExecutionStats,
	startTime time.Time,
) (*ReflectionReport, error) {
	prompt := fmt.Sprintf("%s\n\n## User Question\n%s\n\n## Response to Evaluate\n%s\n\n%s",
		e.reflectionPrompt,
		input,
		answer,
		"Output ONLY valid JSON, no other text.",
	)

	messages := []ai.Message{
		{Role: "system", Content: "You are an objective response evaluator. Output only valid JSON."},
		{Role: "user", Content: prompt},
	}

	response, llmStats, err := llm.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}
	stats.AccumulateLLM(llmStats)

	// Parse JSON response
	var report ReflectionReport
	jsonStr := e.extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
		slog.Error("Failed to parse reflection JSON", "error", err, "response", response)
		// Default to low quality to force refinement when JSON parsing fails
		return &ReflectionReport{
			Accuracy:        0.5,
			Completeness:    0.5,
			Clarity:         0.5,
			Issues:          []string{"Failed to parse reflection output"},
			Suggestions:     []string{"Please review and improve the response"},
			NeedsRefinement: true, // Force refinement on parse error
		}, nil
	}

	return &report, nil
}

// refine improves the answer based on reflection feedback.
func (e *ReflexionExecutor) refine(
	ctx context.Context,
	input string,
	currentAnswer string,
	reflection *ReflectionReport,
	llm ai.LLMService,
	stats *ExecutionStats,
	startTime time.Time,
) (string, error) {
	prompt := fmt.Sprintf("%s\n\n## Original Question\n%s\n\n## Current Response\n%s\n\n## Feedback\nQuality: Accuracy=%.2f, Completeness=%.2f, Clarity=%.2f\n\n## Issues to Address\n%s\n\n## Suggestions\n%s",
		e.refinePrompt,
		input,
		currentAnswer,
		reflection.Accuracy, reflection.Completeness, reflection.Clarity,
		strings.Join(reflection.Issues, "\n- "),
		strings.Join(reflection.Suggestions, "\n- "),
	)

	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that improves responses based on feedback."},
		{Role: "user", Content: prompt},
	}

	response, llmStats, err := llm.Chat(ctx, messages)
	if err != nil {
		return "", err
	}
	stats.AccumulateLLM(llmStats)

	return response, nil
}

// calculateOverallQuality computes the weighted overall quality score.
func (e *ReflexionExecutor) calculateOverallQuality(r *ReflectionReport) float64 {
	// Weighted average: Accuracy 40%, Completeness 35%, Clarity 25%
	return r.Accuracy*0.4 + r.Completeness*0.35 + r.Clarity*0.25
}

// extractJSON extracts JSON from a response that may contain other text.
// It looks for the first complete JSON object and validates basic structure.
func (e *ReflexionExecutor) extractJSON(response string) string {
	trimmed := strings.TrimSpace(response)

	// If response is already valid JSON, return as-is
	var temp interface{}
	if json.Unmarshal([]byte(trimmed), &temp) == nil {
		return trimmed
	}

	// Find JSON object boundaries
	start := strings.Index(trimmed, "{")
	if start < 0 {
		return response // No JSON found, return original
	}

	// Count braces to find matching closing brace
	braceCount := 0
	inString := false
	escaped := false

	for i := start; i < len(trimmed); i++ {
		c := trimmed[i]

		if escaped {
			escaped = false
			continue
		}

		switch c {
		case '\\':
			escaped = true
		case '"':
			inString = !inString
		case '{':
			if !inString {
				braceCount++
			}
		case '}':
			if !inString {
				braceCount--
				if braceCount == 0 {
					// Found matching closing brace
					candidate := trimmed[start : i+1]
					// Validate it's actually valid JSON
					if json.Unmarshal([]byte(candidate), &temp) == nil {
						return candidate
					}
				}
			}
		}
	}

	// Fallback: return original response
	return response
}

// Default prompts
const (
	defaultReflectionPrompt = `
Evaluate the following response objectively and output JSON:

## Evaluation Criteria

1. **Accuracy** (0.0-1.0): Are all facts correct? Any hallucinations or errors?
2. **Completeness** (0.0-1.0): Did we address all aspects of the user's question? Is anything missing?
3. **Clarity** (0.0-1.0): Is the response well-structured and easy to understand?

## Output Format

{
  "accuracy": 0.0-1.0,
  "completeness": 0.0-1.0,
  "clarity": 0.0-1.0,
  "issues": ["specific issue 1", "specific issue 2"],
  "suggestions": ["improvement suggestion 1"],
  "needs_refinement": true/false
}

Respond with ONLY valid JSON.
`

	defaultRefinePrompt = `
Improve the following response based on the feedback provided.

## Instructions

1. Address each issue mentioned in the feedback
2. Incorporate relevant suggestions
3. Maintain the good parts of the original response
4. Ensure the improved response is clear, accurate, and complete

## Output

Output only the improved response, no explanations.
`
)
