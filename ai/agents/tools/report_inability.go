package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ExpertResolver defines an interface for resolving agent names dynamically.
// This enables configuration-driven expert validation instead of hardcoded whitelists.
type ExpertResolver interface {
	// IdentifyAgent resolves an agent name to its canonical ID.
	// Returns empty string if not found.
	IdentifyAgent(name string) string

	// GetAllExpertNames returns all registered expert names.
	GetAllExpertNames() []string
}

var (
	// expertResolver is the dynamic expert resolver.
	// It is set by SetExpertResolver during initialization.
	expertResolver     ExpertResolver
	expertResolverOnce sync.Once
)

// SetExpertResolver sets the expert resolver for dynamic validation.
// This should be called during application initialization.
func SetExpertResolver(resolver ExpertResolver) {
	expertResolverOnce.Do(func() {
		expertResolver = resolver
	})
}

// ResolveExpertAgent resolves an agent name to its canonical ID.
// Returns the original name if no resolver is available.
func ResolveExpertAgent(name string) string {
	if name == "" {
		return ""
	}

	// Use dynamic resolver if available
	if expertResolver != nil {
		resolved := expertResolver.IdentifyAgent(name)
		if resolved != "" {
			return resolved
		}
	}

	// Fallback: return original name
	return name
}

// ReportInabilityInput represents the input for reporting inability to handle a task.
// 当专家发现自己无法完成任务时使用此输入。
// Note: Expert only reports what it CANNOT do. Orchestrator determines the appropriate agent.
type ReportInabilityInput struct {
	Capability string `json:"capability" jsonschema_description:"缺失的能力"`
	Reason     string `json:"reason" jsonschema_description:"为什么无法完成"`
}

// Error returns a human-readable error message.
func (i *ReportInabilityInput) Error() string {
	return fmt.Sprintf("cannot handle capability %s: %s", i.Capability, i.Reason)
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

INPUT FORMAT:
{"capability": "capability_name", "reason": "why cannot handle"}

OUTPUT:
- Success: "INABILITY_REPORTED: <capability> - <reason>"
- Error: "Error: <error message>"

Note: Expert only reports what it CANNOT do. Orchestrator will use CapabilityMap to find the appropriate expert.`
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
		},
		"required": []string{"capability", "reason"},
	}
}

// MaxInputLength is the maximum allowed length for input fields.
const MaxInputLength = 1000

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

	// Validate input length to prevent DoS attacks
	if len(reportInput.Capability) > MaxInputLength {
		return "", fmt.Errorf("capability exceeds maximum length of %d", MaxInputLength)
	}
	if len(reportInput.Reason) > MaxInputLength {
		return "", fmt.Errorf("reason exceeds maximum length of %d", MaxInputLength)
	}

	// Return a special message that indicates inability
	// Orchestrator will use CapabilityMap to find the appropriate expert
	result := fmt.Sprintf("INABILITY_REPORTED: %s - %s", reportInput.Capability, reportInput.Reason)

	return result, nil
}
