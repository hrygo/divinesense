package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToHistoryPrompt(t *testing.T) {
	ctx := NewConversationContext("test", 1, "UTC")

	// Test case 1: Empty history
	assert.Equal(t, "", ctx.ToHistoryPrompt())

	// Test case 2: Simple Turn
	ctx.AddTurn("Hello", "Hi there", nil)
	prompt := ctx.ToHistoryPrompt()
	assert.Contains(t, prompt, "User: Hello")
	assert.Contains(t, prompt, "Assistant: Hi there")

	// Test case 3: Turn with Tool Usage
	ctx.AddTurn("Find schedule", "Found it", []ToolCallRecord{
		{Tool: "schedule_query", Success: true},
	})

	prompt = ctx.ToHistoryPrompt()
	// Should see tool summary
	// "System: Agent used tools: schedule_query (OK)"
	assert.Contains(t, prompt, "System: Agent used tools: schedule_query (OK)")
	// Ensure chronological order (append)
	assert.True(t, strings.Index(prompt, "Hello") < strings.Index(prompt, "Find schedule"))
}
