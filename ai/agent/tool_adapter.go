package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
)

// Regular expression to match text-based tool calls like [Tool: function_name(args)].
var textToolCallRegex = regexp.MustCompile(`\[Tool:\s*(\w+)\((.*?)\)\]`)

// ToolWithSchema extends the Tool interface to include JSON Schema definition.
// This is needed for the new Agent framework to provide tool definitions to the LLM.
type ToolWithSchema interface {
	Tool

	// Parameters returns the JSON Schema for the tool's input parameters.
	Parameters() map[string]interface{}
}

// NativeTool implements ToolWithSchema with direct function execution.
type NativeTool struct {
	execute     func(ctx context.Context, input string) (string, error)
	params      map[string]interface{}
	name        string
	description string
}

// NewNativeTool creates a new NativeTool.
func NewNativeTool(
	name string,
	description string,
	execute func(ctx context.Context, input string) (string, error),
	parameters map[string]interface{},
) ToolWithSchema {
	return &NativeTool{
		name:        name,
		description: description,
		execute:     execute,
		params:      parameters,
	}
}

// Name returns the tool name.
func (t *NativeTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *NativeTool) Description() string {
	return t.description
}

// Parameters returns the JSON Schema for parameters.
func (t *NativeTool) Parameters() map[string]interface{} {
	return t.params
}

// Run executes the tool.
func (t *NativeTool) Run(ctx context.Context, input string) (string, error) {
	return t.execute(ctx, input)
}

// ToolFromLegacy creates a ToolWithSchema from a tool that has InputType() method.
// This adapts existing tools like ScheduleQueryTool to the new framework.
func ToolFromLegacy(
	name, description string,
	runFunc func(ctx context.Context, input string) (string, error),
	inputTypeFunc func() map[string]interface{},
) ToolWithSchema {
	return &NativeTool{
		name:        name,
		description: description,
		execute:     runFunc,
		params:      inputTypeFunc(),
	}
}

// Agent is a lightweight, framework-less AI agent.
// It uses native LLM tool calling without LangChainGo.
type Agent struct {
	llm     ai.LLMService
	toolMap map[string]ToolWithSchema
	config  AgentConfig
	tools   []ToolWithSchema
}

// AgentConfig holds configuration for creating a new Agent.
type AgentConfig struct {
	// Name identifies this agent.
	Name string

	// SystemPrompt is the base system prompt for the LLM.
	SystemPrompt string

	// MaxIterations is the maximum number of tool-calling loops.
	MaxIterations int
}

// NewAgent creates a new Agent with the given configuration.
func NewAgent(llm ai.LLMService, config AgentConfig, tools []ToolWithSchema) *Agent {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}

	toolMap := make(map[string]ToolWithSchema)
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	return &Agent{
		llm:     llm,
		config:  config,
		tools:   tools,
		toolMap: toolMap,
	}
}

// Callback is called during agent execution for events.
type Callback func(event string, data string)

// Event constants for callbacks.
const (
	EventToolUse    = "tool_use"
	EventToolResult = "tool_result"
	EventAnswer     = "answer"
)

// Run executes the agent with the given input.
// Returns the final response or an error.
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	return a.RunWithCallback(ctx, input, nil)
}

// RunWithCallback executes the agent with callback support.
func (a *Agent) RunWithCallback(ctx context.Context, input string, callback Callback) (string, error) {
	startTime := time.Now()
	defer func() {
		slog.Debug("agent execution completed",
			"agent", a.config.Name,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
	}()

	// Build initial messages
	messages := []ai.Message{
		{Role: "system", Content: a.config.SystemPrompt},
		{Role: "user", Content: input},
	}

	// Track tool results for early stopping
	var lastToolResult string

	// Execute the agent loop
	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		iterStart := time.Now()

		// Call LLM with tools
		resp, stats, err := a.llm.ChatWithTools(ctx, messages, a.toolDescriptors())
		if err != nil {
			return "", fmt.Errorf("LLM call failed (iteration %d): %w", iteration+1, err)
		}
		_ = stats // TODO: accumulate stats in session

		// Check if LLM wants to call tools
		hasStructuredToolCalls := len(resp.ToolCalls) > 0
		hasTextToolCalls := textToolCallRegex.MatchString(resp.Content)

		// Case 1: No tool calls at all = final answer
		if !hasStructuredToolCalls && !hasTextToolCalls {
			if callback != nil && resp.Content != "" {
				callback(EventAnswer, resp.Content)
			}

			slog.Debug("agent iteration completed",
				"agent", a.config.Name,
				"iteration", iteration+1,
				"duration_ms", time.Since(iterStart).Milliseconds(),
				"final", true,
			)
			return resp.Content, nil
		}

		// Case 2: Text-based tool calls found - parse and execute them
		if !hasStructuredToolCalls && hasTextToolCalls {
			slog.Info("text-based tool calls detected, parsing",
				"agent", a.config.Name,
				"iteration", iteration+1)

			// Extract all tool calls from content
			matches := textToolCallRegex.FindAllStringSubmatch(resp.Content, -1)

			// Remove tool call syntax from content for cleaner display
			cleanContent := textToolCallRegex.ReplaceAllString(resp.Content, "")
			cleanContent = strings.TrimSpace(cleanContent)

			// Add assistant's response (without tool syntax) to history
			messages = append(messages, ai.Message{Role: "assistant", Content: cleanContent})

			// Send cleaned content to callback
			if callback != nil && cleanContent != "" {
				callback(EventAnswer, cleanContent)
			}

			// Execute each tool
			for _, match := range matches {
				if len(match) < 3 {
					continue
				}
				toolName := match[1]
				toolInput := match[2]

				// Notify callback
				if callback != nil {
					callback(EventToolUse, fmt.Sprintf("%s:%s", toolName, toolInput))
				}

				toolStart := time.Now()

				// Execute the tool
				toolResult, err := a.executeTool(ctx, toolName, toolInput)
				if err != nil {
					toolResult = fmt.Sprintf("Error: %v", err)
				}

				slog.Debug("text-based tool execution completed",
					"agent", a.config.Name,
					"tool", toolName,
					"duration_ms", time.Since(toolStart).Milliseconds(),
				)

				// Notify callback of result
				if callback != nil {
					callback(EventToolResult, toolResult)
				}

				// Add tool result as a user message
				messages = append(messages, ai.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Result from %s]: %s", toolName, toolResult),
				})

				// Check for early stopping
				if shouldEarlyStop(toolResult) {
					slog.Debug("early stopping triggered after text-based tool",
						"agent", a.config.Name,
						"iteration", iteration+1,
						"tool", toolName,
					)
					return toolResult, nil
				}
			}

			// Continue to next iteration to let LLM respond
			slog.Debug("agent iteration completed",
				"agent", a.config.Name,
				"iteration", iteration+1,
				"duration_ms", time.Since(iterStart).Milliseconds(),
				"final", false,
			)
			continue
		}

		// Case 3: Structured tool calls - existing logic
		// Send thinking/content to callback (without tool call syntax)
		// IMPORTANT: Tool call syntax is ONLY for message history, not sent to frontend
		if callback != nil && resp.Content != "" {
			callback(EventAnswer, resp.Content)
		}

		// Add assistant's response to history with tool call syntax
		assistantText := resp.Content
		if len(resp.ToolCalls) > 0 {
			for _, tc := range resp.ToolCalls {
				assistantText += fmt.Sprintf("\n[Tool: %s(%s)]", tc.Function.Name, tc.Function.Arguments)
			}
		}
		messages = append(messages, ai.Message{Role: "assistant", Content: assistantText})

		// Execute each tool and add results to history
		for _, tc := range resp.ToolCalls {
			toolName := tc.Function.Name
			toolInput := tc.Function.Arguments

			// Notify callback
			if callback != nil {
				callback(EventToolUse, fmt.Sprintf("%s:%s", toolName, toolInput))
			}

			toolStart := time.Now()

			// Execute the tool
			toolResult, err := a.executeTool(ctx, toolName, toolInput)
			if err != nil {
				toolResult = fmt.Sprintf("Error: %v", err)
			}

			slog.Debug("tool execution completed",
				"agent", a.config.Name,
				"tool", toolName,
				"duration_ms", time.Since(toolStart).Milliseconds(),
			)

			// Notify callback of result
			if callback != nil {
				callback(EventToolResult, toolResult)
			}

			// Store for early stopping check
			lastToolResult = toolResult

			// Add tool result as a user message (simulating user giving feedback)
			// This is a simplified approach; more sophisticated implementations
			// might use a dedicated "tool" message type
			messages = append(messages, ai.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Result from %s]: %s", toolName, toolResult),
			})
		}

		// Early stopping: check if task is complete after schedule_add or schedule_update
		if shouldEarlyStop(lastToolResult) {
			slog.Debug("early stopping triggered",
				"agent", a.config.Name,
				"iteration", iteration+1,
				"reason", "task_completed",
			)

			// Generate a brief final response
			finalResponse := lastToolResult
			if callback != nil {
				callback(EventAnswer, finalResponse)
			}
			return finalResponse, nil
		}

		slog.Debug("agent iteration completed",
			"agent", a.config.Name,
			"iteration", iteration+1,
			"duration_ms", time.Since(iterStart).Milliseconds(),
			"final", false,
		)
	}

	return "", fmt.Errorf("max iterations (%d) exceeded", a.config.MaxIterations)
}

// shouldEarlyStop checks if the agent should stop early based on tool results.
// Returns true if a schedule was successfully created or updated.
func shouldEarlyStop(toolResult string) bool {
	if toolResult == "" {
		return false
	}

	// Check for success indicators in Chinese and English
	successIndicators := []string{
		"✓ 已创建",
		"✓ 已更新",
		"已成功创建",
		"成功创建日程",
		"Successfully created",
		"Successfully updated",
		"schedule created",
		"schedule updated",
	}

	lowerResult := strings.ToLower(toolResult)
	for _, indicator := range successIndicators {
		if strings.Contains(toolResult, indicator) || strings.Contains(lowerResult, strings.ToLower(indicator)) {
			return true
		}
	}

	return false
}

// toolDescriptors converts the agent's tools to ai.ToolDescriptor format.
func (a *Agent) toolDescriptors() []ai.ToolDescriptor {
	descriptors := make([]ai.ToolDescriptor, len(a.tools))
	for i, tool := range a.tools {
		paramsJSON, err := json.Marshal(tool.Parameters())
		if err != nil {
			slog.Warn("failed to marshal tool parameters, using empty schema",
				"tool", tool.Name(),
				"error", err)
			paramsJSON = []byte(`{"type":"object","properties":{}}`)
		}
		descriptors[i] = ai.ToolDescriptor{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  string(paramsJSON),
		}
	}
	return descriptors
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(ctx context.Context, name, input string) (string, error) {
	tool, exists := a.toolMap[name]
	if !exists {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Run(ctx, input)
}
