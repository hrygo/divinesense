package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// Aggregator combines multiple expert results into a unified response.
type Aggregator struct {
	llm    llm.Service
	config *OrchestratorConfig
}

// NewAggregator creates a new result aggregator.
func NewAggregator(llmService llm.Service, config *OrchestratorConfig) *Aggregator {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Aggregator{
		llm:    llmService,
		config: config,
	}
}

// Aggregate combines results from multiple tasks into a single coherent response.
func (a *Aggregator) Aggregate(ctx context.Context, result *ExecutionResult, callback EventCallback) (string, error) {
	if !result.IsAggregated || len(result.Plan.Tasks) <= 1 {
		// No aggregation needed
		return result.FinalResponse, nil
	}

	// Collect successful results
	var successfulResults []string
	for _, task := range result.Plan.Tasks {
		if task.Status == TaskStatusCompleted && task.Result != "" {
			successfulResults = append(successfulResults,
				fmt.Sprintf("【%s】\n%s", task.Agent, task.Result))
		}
	}

	if len(successfulResults) == 0 {
		return "", fmt.Errorf("no successful results to aggregate")
	}

	if len(successfulResults) == 1 {
		return successfulResults[0], nil
	}

	// Build aggregation prompt
	prompt := a.buildAggregationPrompt(result.Plan.Analysis, successfulResults)

	// Call LLM for aggregation
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	response, stats, err := a.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("aggregator: LLM call failed, falling back to concatenation", "error", err)
		// Notify frontend about fallback
		if callback != nil {
			callback("aggregation_fallback", "LLM aggregation failed, using simple concatenation")
		}
		// Fallback to simple concatenation
		return strings.Join(successfulResults, "\n\n---\n\n"), nil
	}

	// Update token usage
	if stats != nil {
		result.TokenUsage.InputTokens += int32(stats.PromptTokens)
		result.TokenUsage.OutputTokens += int32(stats.CompletionTokens)
		result.TokenUsage.CacheReadTokens += int32(stats.CacheReadTokens)
		result.TokenUsage.CacheWriteTokens += int32(stats.CacheWriteTokens)
	}

	slog.Info("aggregator: results aggregated",
		"input_tasks", len(successfulResults),
		"response_length", len(response))

	// Send aggregation event if callback provided
	if callback != nil {
		callback("aggregation", response)
	}

	return response, nil
}

// buildAggregationPrompt creates the prompt for result aggregation.
func (a *Aggregator) buildAggregationPrompt(analysis string, results []string) string {
	return fmt.Sprintf(`You are a helpful assistant that combines information from multiple sources into a coherent response.

## Original User Request
%s

## Results from Expert Agents
%s

## Your Task
Combine the above results into a single, coherent response that:
1. Addresses the user's original request completely
2. Integrates information naturally without repetition
3. Highlights the most relevant information
4. Maintains a friendly and helpful tone
5. Uses appropriate formatting (bullet points, sections, etc.) for clarity

## Response (in Chinese, matching the user's language)`,
		analysis,
		strings.Join(results, "\n\n"))
}
