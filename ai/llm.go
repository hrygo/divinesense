package ai

import (
	"github.com/hrygo/divinesense/ai/core/llm"
)

// Message represents a chat message.
//
// Deprecated: Use llm.Message directly.
type Message = llm.Message

// LLMCallStats represents statistics for a single LLM call.
//
// Deprecated: Use llm.LLMCallStats directly.
type LLMCallStats = llm.LLMCallStats

// LLMService is the LLM service interface.
//
// Deprecated: Use llm.Service directly.
type LLMService = llm.Service

// ToolDescriptor represents a function/tool available to the LLM.
//
// Deprecated: Use llm.ToolDescriptor directly.
type ToolDescriptor = llm.ToolDescriptor

// ChatResponse represents the LLM response including potential tool calls.
//
// Deprecated: Use llm.ChatResponse directly.
type ChatResponse = llm.ChatResponse

// ToolCall represents a request to call a tool.
//
// Deprecated: Use llm.ToolCall directly.
type ToolCall = llm.ToolCall

// FunctionCall represents the function details.
//
// Deprecated: Use llm.FunctionCall directly.
type FunctionCall = llm.FunctionCall

// NewLLMService creates a new LLMService.
//
// Phase 1 Note: This is a bridge compatibility layer that maintains the original API.
// The actual LLM functionality has been moved to ai/core/llm/service.go.
func NewLLMService(cfg *LLMConfig) (LLMService, error) {
	return llm.NewService((*llm.Config)(cfg))
}

// SystemPrompt creates a system message.
//
// Deprecated: Use llm.SystemPrompt directly.
func SystemPrompt(content string) Message {
	return llm.SystemPrompt(content)
}

// UserMessage creates a user message.
//
// Deprecated: Use llm.UserMessage directly.
func UserMessage(content string) Message {
	return llm.UserMessage(content)
}

// AssistantMessage creates an assistant message.
//
// Deprecated: Use llm.AssistantMessage directly.
func AssistantMessage(content string) Message {
	return llm.AssistantMessage(content)
}

// FormatMessages formats messages for prompt templates.
//
// Deprecated: Use llm.FormatMessages directly.
func FormatMessages(systemPrompt string, userContent string, history []Message) []Message {
	return llm.FormatMessages(systemPrompt, userContent, history)
}
