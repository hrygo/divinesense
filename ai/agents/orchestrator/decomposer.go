package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// Decomposer uses LLM to analyze user input and decompose it into tasks.
type Decomposer struct {
	llm    llm.Service
	config *OrchestratorConfig
}

// NewDecomposer creates a new task decomposer.
func NewDecomposer(llmService llm.Service, config *OrchestratorConfig) *Decomposer {
	if config == nil {
		config = DefaultOrchestratorConfig()
	}
	return &Decomposer{
		llm:    llmService,
		config: config,
	}
}

// Decompose analyzes the user input and creates a task plan.
func (d *Decomposer) Decompose(ctx context.Context, userInput string, registry ExpertRegistry) (*TaskPlan, error) {
	// Get available experts
	experts := registry.GetAvailableExperts()
	expertDescriptions := d.buildExpertDescriptions(experts, registry)

	// Build the decomposition prompt
	prompt := d.buildDecompositionPrompt(userInput, expertDescriptions)

	// Call LLM for decomposition
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	response, _, err := d.llm.Chat(ctx, messages)
	if err != nil {
		slog.Error("decomposer: LLM call failed", "error", err)
		return nil, fmt.Errorf("LLM decomposition failed: %w", err)
	}

	// Parse the response into a TaskPlan
	plan, err := d.parseTaskPlan(response)
	if err != nil {
		slog.Warn("decomposer: failed to parse plan, using fallback", "error", err, "response", response)
		return d.fallbackPlan(userInput, experts), nil
	}

	slog.Info("decomposer: task plan created",
		"analysis", plan.Analysis,
		"tasks", len(plan.Tasks),
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
func (d *Decomposer) buildDecompositionPrompt(userInput, expertDescriptions string) string {
	return fmt.Sprintf(`You are an intelligent task orchestrator. Analyze the user's request and decompose it into tasks for expert agents.

## Available Expert Agents
%s

## Your Task
1. Analyze the user's request to understand what they need
2. Determine which expert(s) should handle the request
3. Create specific tasks for each expert
4. Decide if tasks can run in parallel (independent) or must run sequentially

## Output Format
Respond with a JSON object in this exact format:
{
  "analysis": "Brief analysis of what the user wants",
  "tasks": [
    {"agent": "expert_name", "input": "specific input for this task", "purpose": "why this task is needed"}
  ],
  "parallel": true/false,
  "aggregate": true/false
}

## Rules
- Use only available expert agents
- If only one expert is needed, create one task with parallel=false
- If multiple experts are needed and tasks are independent, set parallel=true
- Set aggregate=true when multiple results need to be combined
- Keep task inputs specific and actionable

## User Request
%s

## Response (JSON only, no markdown)`, expertDescriptions, userInput)
}

// parseTaskPlan parses the LLM response into a TaskPlan.
func (d *Decomposer) parseTaskPlan(response string) (*TaskPlan, error) {
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

	// Initialize task status
	for _, task := range plan.Tasks {
		task.Status = TaskStatusPending
	}

	return &plan, nil
}

// fallbackPlan creates a simple plan when LLM parsing fails.
func (d *Decomposer) fallbackPlan(userInput string, availableExperts []string) *TaskPlan {
	// Default to first available expert (usually "memo" or "schedule")
	expert := "amazing"
	if len(availableExperts) > 0 {
		// Prefer memo or schedule over amazing if available
		for _, e := range availableExperts {
			if e == "memo" || e == "schedule" {
				expert = e
				break
			}
		}
		if expert == "amazing" {
			expert = availableExperts[0]
		}
	}

	return &TaskPlan{
		Analysis: "Direct routing to expert agent",
		Tasks: []*Task{{
			Agent:   expert,
			Input:   userInput,
			Purpose: "Handle user request",
			Status:  TaskStatusPending,
		}},
		Parallel:  false,
		Aggregate: false,
	}
}
