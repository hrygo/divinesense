package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ReportInabilityInput represents the input for reporting inability to handle a task.
// 当专家发现自己无法完成任务时使用此输入。
type ReportInabilityInput struct {
	Capability     string `json:"capability" jsonschema_description:"缺失的能力"`
	Reason         string `json:"reason" jsonschema_description:"为什么无法完成"`
	SuggestedAgent string `json:"suggested_agent,omitempty" jsonschema_description:"建议转交的专家"`
}

// Error returns a human-readable error message.
func (i *ReportInabilityInput) Error() string {
	return fmt.Sprintf("cannot handle capability %s: %s", i.Capability, i.Reason)
}

// MissingCapability is an error type that indicates the agent cannot handle a specific capability.
// MissingCapability 错误类型，表示代理无法处理特定能力。
type MissingCapability struct {
	Capability     string
	Reason         string
	SuggestedAgent string
}

// Error returns a human-readable error message.
func (m *MissingCapability) Error() string {
	return fmt.Sprintf("missing capability: %s - %s (suggested: %s)", m.Capability, m.Reason, m.SuggestedAgent)
}

// ReportInabilityTool allows an expert agent to report when it cannot handle a task.
// This enables the Handoff mechanism where the Orchestrator can route to a different expert.
// ReportInabilityTool 允许专家代理在无法处理任务时进行报告。
// 这使得 Orchestrator 可以将任务转交给其他专家。
type ReportInabilityTool struct{}

// NewReportInabilityTool creates a new ReportInability tool.
func NewReportInabilityTool() *ReportInabilityTool {
	return &ReportInabilityTool{}
}

// Name returns the name of the tool.
func (t *ReportInabilityTool) Name() string {
	return "report_inability"
}

// Description returns a description of what the tool does.
func (t *ReportInabilityTool) Description() string {
	return `Reports when the expert cannot handle a specific task capability.

This tool is used when:
- The user request is outside the expert's capabilities
- The task requires another expert's domain knowledge
- The expert needs to handoff to a more suitable agent

INPUT FORMAT:
{"capability": "capability_name", "reason": "why cannot handle", "suggested_agent": "expert_name"}

OUTPUT:
- Success: "INABILITY_REPORTED: <capability> - <reason>"
- Error: "Error: <error message>"

The Orchestrator will use this information to route to the appropriate expert.`
}

// InputType returns the JSON schema for the tool's input parameters.
func (t *ReportInabilityTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"capability": map[string]interface{}{
				"type":        "string",
				"description": "The capability that is missing or cannot be handled",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Why this capability cannot be handled",
			},
			"suggested_agent": map[string]interface{}{
				"type":        "string",
				"description": "Optional: suggested expert agent name to handle this capability",
			},
		},
		"required": []string{"capability", "reason"},
	}
}

// Run executes the report inability tool.
// Returns a confirmation message that will trigger early stopping in the agent.
func (t *ReportInabilityTool) Run(ctx context.Context, input string) (string, error) {
	// Parse input
	var reportInput ReportInabilityInput
	if err := json.Unmarshal([]byte(input), &reportInput); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate required fields
	if reportInput.Capability == "" {
		return "", fmt.Errorf("capability is required")
	}
	if reportInput.Reason == "" {
		return "", fmt.Errorf("reason is required")
	}

	// Return a special message that indicates inability
	// This message format should match the early stopping logic in the agent
	result := fmt.Sprintf("INABILITY_REPORTED: %s - %s", reportInput.Capability, reportInput.Reason)
	if reportInput.SuggestedAgent != "" {
		result += fmt.Sprintf(" (suggested_agent: %s)", reportInput.SuggestedAgent)
	}

	return result, nil
}
