package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/agents/universal"
	ctxpkg "github.com/hrygo/divinesense/ai/context"
	"github.com/hrygo/divinesense/ai/core/llm"
)

// Decomposer uses LLM to analyze user input and decompose it into tasks.
type Decomposer struct {
	llm          llm.Service
	config       *OrchestratorConfig
	promptConfig *PromptConfig
}

// NewDecomposer creates a new task decomposer.
func NewDecomposer(llmService llm.Service, config *OrchestratorConfig) *Decomposer {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Decomposer{
		llm:          llmService,
		config:       config,
		promptConfig: GetPromptConfig(),
	}
}

// Decompose analyzes the user input and creates a task plan.
func (d *Decomposer) Decompose(ctx context.Context, userInput string, registry ExpertRegistry, traceID string) (*TaskPlan, error) {
	startTime := time.Now()

	// Log decomposition start
	slog.Info("decomposer: start decompose",
		"trace_id", traceID,
		"user_input", userInput,
		"timestamp", time.Now().UnixMilli(),
	)

	// Get available experts
	experts := registry.GetAvailableExperts()
	expertDescriptions := d.buildExpertDescriptions(experts, registry)

	// Build time context for relative date resolution (e.g., "下周五")
	timeContext := universal.BuildTimeContext(time.Now().Location())

	// Build the decomposition prompt with time context and history
	history := ctxpkg.GetHistory(ctx)
	prompt := d.buildDecompositionPrompt(userInput, expertDescriptions, timeContext, history)

	// Call LLM for decomposition
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	response, _, err := d.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("decomposer: LLM call failed",
			"trace_id", traceID,
			"error", err)
		return nil, fmt.Errorf("LLM decomposition failed: %w", err)
	}

	// Parse the response into a TaskPlan
	plan, err := d.parseTaskPlan(response, experts)
	if err != nil {
		slog.Warn("decomposer: failed to parse plan, using fallback",
			"trace_id", traceID,
			"error", err,
			"response_length", len(response))
		plan := d.fallbackPlan(userInput, experts)
		plan.Analysis = "[Fallback Mode] " + plan.Analysis
		return plan, nil
	}

	// Log decomposition complete
	duration := time.Since(startTime)
	slog.Info("decomposer: decompose complete",
		"trace_id", traceID,
		"task_count", len(plan.Tasks),
		"duration_ms", duration.Milliseconds(),
		"analysis", plan.Analysis,
		"parallel", plan.Parallel)

	return plan, nil
}

// buildExpertDescriptions creates a description string for all available experts.
func (d *Decomposer) buildExpertDescriptions(experts []string, registry ExpertRegistry) string {
	var sb strings.Builder
	for _, name := range experts {
		desc := registry.GetExpertDescription(name)
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", name, desc))
	}
	return sb.String()
}

// buildDecompositionPrompt creates the prompt for task decomposition.
func (d *Decomposer) buildDecompositionPrompt(userInput, expertDescriptions string, timeContext *universal.TimeContext, history []string) string {
	return d.promptConfig.BuildDecomposerPrompt(userInput, expertDescriptions, timeContext, history)
}

// parseTaskPlan parses the LLM response into a TaskPlan.
// validAgents is a list of valid agent names for validation.
func (d *Decomposer) parseTaskPlan(response string, validAgents []string) (*TaskPlan, error) {
	// Clean up the response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var plan TaskPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Validate and set defaults
	if len(plan.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks in plan")
	}

	// Build valid agent set for validation
	validSet := make(map[string]bool)
	for _, a := range validAgents {
		validSet[a] = true
	}

	// Validate agent names and initialize task status
	for i, task := range plan.Tasks {
		if !validSet[task.Agent] {
			return nil, fmt.Errorf("invalid agent: %s (valid: %v)", task.Agent, validAgents)
		}
		// Generate ID if empty (LLM may not always return id field)
		if task.ID == "" {
			task.ID = fmt.Sprintf("t%d", i+1)
		}
		task.SetStatus(TaskStatusPending)
	}

	return &plan, nil
}

// fallbackPlan creates a simple plan when LLM parsing fails.
func (d *Decomposer) fallbackPlan(userInput string, availableExperts []string) *TaskPlan {
	// Default to first available expert (usually "memo" or "schedule")
	expert := ""
	if len(availableExperts) > 0 {
		// Prefer memo or schedule as they are core experts
		for _, e := range availableExperts {
			if e == "memo" || e == "schedule" {
				expert = e
				break
			}
		}
		// Fall back to first available if no preferred expert found
		if expert == "" {
			expert = availableExperts[0]
		}
	}

	// If no experts available, return error plan
	if expert == "" {
		slog.Warn("decomposer: no experts available for fallback")
		return &TaskPlan{
			Analysis:  "No expert agents available",
			Tasks:     []*Task{},
			Parallel:  false,
			Aggregate: false,
		}
	}

	return &TaskPlan{
		Analysis: "Direct routing to expert agent",
		Tasks: []*Task{{
			ID:      "t1",
			Agent:   expert,
			Input:   userInput,
			Purpose: "Handle user request",
			Status:  TaskStatusPending,
		}},
		Parallel:  false,
		Aggregate: false,
	}
}
