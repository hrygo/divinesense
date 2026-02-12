// Package ai provides tests for AgentFactory integration with ParrotFactory.
package ai

import (
	"context"
	"testing"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/agents/universal"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

// mockLLM is a mock LLM service for testing.
type mockLLM struct{}

func (m *mockLLM) Chat(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
	return "mock response", &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
}

func (m *mockLLM) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, <-chan *ai.LLMCallStats, <-chan error) {
	contentChan := make(chan string, 1)
	statsChan := make(chan *ai.LLMCallStats, 1)
	errChan := make(chan error, 1)

	contentChan <- "mock response"
	statsChan <- &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}
	close(contentChan)
	close(statsChan)
	close(errChan)

	return contentChan, statsChan, errChan
}

func (m *mockLLM) ChatWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolDescriptor) (*ai.ChatResponse, *ai.LLMCallStats, error) {
	return &ai.ChatResponse{
		Content: "mock response",
	}, &ai.LLMCallStats{PromptTokens: 10, CompletionTokens: 5}, nil
}

// Warmup is a no-op for the mock.
func (m *mockLLM) Warmup(ctx context.Context) {
	// No-op for mock
}

// TestBuildToolFactories verifies that buildToolFactories creates valid tool factories.
func TestBuildToolFactories(t *testing.T) {
	// Create a factory with minimal dependencies
	factory := &AgentFactory{
		llm: &mockLLM{},
		// Note: retriever and store are nil, so only nil-safe operations will work
	}

	// buildToolFactories should not panic even with nil dependencies
	factories := factory.buildToolFactories()

	if len(factories) == 0 {
		t.Log("no tool factories created (retriever and store are nil)")
		return
	}

	t.Logf("created %d tool factories", len(factories))

	// Verify each factory can be called (even if it returns error due to nil dependencies)
	for name, factoryFunc := range factories {
		t.Run(name, func(t *testing.T) {
			// Calling with test userID
			tool, err := factoryFunc(123)
			if err != nil {
				// This is expected if dependencies are nil
				t.Logf("factory %s returned error (expected with nil deps): %v", name, err)
				return
			}

			// If no error, verify the tool implements ToolWithSchema
			if tool == nil {
				t.Errorf("factory %s returned nil tool", name)
				return
			}

			// Verify tool has required methods
			if tool.Name() == "" {
				t.Errorf("tool %s has empty name", name)
			}
			if tool.Description() == "" {
				t.Errorf("tool %s has empty description", name)
			}

			// Verify Parameters() returns valid schema
			params := tool.Parameters()
			if params == nil {
				t.Errorf("tool %s returned nil parameters", name)
			}

			t.Logf("tool %s: name=%s, description=%d chars, params=%v",
				name, tool.Name(), len(tool.Description()), params)
		})
	}
}

// TestToolFactorySignature verifies that tool factory functions have correct signature.
func TestToolFactorySignature(t *testing.T) {
	// This is a compile-time test to verify ToolFactoryFunc signature matches
	// what ParrotFactory expects.

	// The signature must be: func(userID int32) (agent.ToolWithSchema, error)

	var _ universal.ToolFactoryFunc = func(userID int32) (agent.ToolWithSchema, error) {
		// Create a mock tool for testing
		return agent.NewNativeTool(
			"test_tool",
			"A test tool",
			func(ctx context.Context, input string) (string, error) {
				return "test result", nil
			},
			map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type": "string",
					},
				},
			},
		), nil
	}

	t.Log("ToolFactoryFunc signature verified")
}

// TestToolFromLegacyPattern verifies the pattern used in buildToolFactories.
func TestToolFromLegacyPattern(t *testing.T) {
	// This test verifies that the pattern used in buildToolFactories works:
	// 1. Create a tool instance
	// 2. Wrap it with ToolFromLegacy
	// 3. Verify it implements ToolWithSchema

	// Create a mock schedule tool (simplified version)
	type mockTool struct {
		name        string
		description string
	}

	mockScheduleTool := &mockTool{
		name:        "schedule_query",
		description: "Query schedules",
	}

	// Wrap using ToolFromLegacy pattern
	wrapped := agent.ToolFromLegacy(
		mockScheduleTool.name,
		mockScheduleTool.description,
		func(ctx context.Context, input string) (string, error) {
			return "mock schedule result", nil
		},
		func() map[string]interface{} {
			return map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "ISO8601 time string",
					},
				},
			}
		},
	)

	// Verify wrapped tool implements ToolWithSchema
	if wrapped.Name() != mockScheduleTool.name {
		t.Errorf("name mismatch: got %s, want %s", wrapped.Name(), mockScheduleTool.name)
	}

	if wrapped.Description() != mockScheduleTool.description {
		t.Errorf("description mismatch")
	}

	params := wrapped.Parameters()
	if params == nil {
		t.Error("parameters is nil")
	}

	t.Logf("wrapped tool: name=%s, params=%v", wrapped.Name(), params)
}

// TestAgentFactoryWithUniversalEnabled verifies AgentFactory with UniversalParrot enabled.
func TestAgentFactoryWithUniversalEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test verifies that AgentFactory can be created and initialized
	// Note: Full integration testing requires database and LLM setup

	factory := NewAgentFactory(&mockLLM{}, nil, nil)

	if factory == nil {
		t.Fatal("NewAgentFactory returned nil")
	}

	// Test Initialize with nil config (should error)
	err := factory.Initialize(nil)
	if err == nil {
		t.Error("Initialize with nil config should return error")
	}

	t.Log("AgentFactory verified")
}

// TestCreateConfig verifies AgentTypeFromProto conversion.
func TestAgentTypeFromProto(t *testing.T) {
	tests := []struct {
		name     string
		proto    v1pb.AgentType
		expected AgentType
	}{
		{"MEMO", v1pb.AgentType_AGENT_TYPE_MEMO, AgentTypeMemo},
		{"SCHEDULE", v1pb.AgentType_AGENT_TYPE_SCHEDULE, AgentTypeSchedule},
		{"AMAZING", v1pb.AgentType_AGENT_TYPE_AMAZING, AgentTypeAuto}, // AMAZING now triggers auto-routing
		{"DEFAULT", v1pb.AgentType_AGENT_TYPE_DEFAULT, AgentTypeAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AgentTypeFromProto(tt.proto)
			if result != tt.expected {
				t.Errorf("AgentTypeFromProto(%v) = %s, want %s", tt.proto, result, tt.expected)
			}
		})
	}
}

// TestAgentTypeToProto verifies AgentType.ToProto conversion.
func TestAgentTypeToProto(t *testing.T) {
	tests := []struct {
		name     string
		agent    AgentType
		expected v1pb.AgentType
	}{
		{"MEMO", AgentTypeMemo, v1pb.AgentType_AGENT_TYPE_MEMO},
		{"SCHEDULE", AgentTypeSchedule, v1pb.AgentType_AGENT_TYPE_SCHEDULE},
		{"AUTO", AgentTypeAuto, v1pb.AgentType_AGENT_TYPE_DEFAULT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.ToProto()
			if result != tt.expected {
				t.Errorf("AgentType.ToProto() = %v, want %v", result, tt.expected)
			}
		})
	}
}
