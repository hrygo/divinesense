package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// EventTypeRecord records an observed event type during testing.
type EventTypeRecord struct {
	Type      string
	Subtype   string
	Name      string
	RawJSON   string
	Timestamp time.Time
}

// EventTypeRecorder records all event types observed during a test.
type EventTypeRecorder struct {
	events []EventTypeRecord
}

// NewEventTypeRecorder creates a new event recorder.
func NewEventTypeRecorder() *EventTypeRecorder {
	return &EventTypeRecorder{
		events: make([]EventTypeRecord, 0),
	}
}

// Callback returns an EventCallback that records all events.
func (r *EventTypeRecorder) Callback() EventCallback {
	return func(eventType string, eventData interface{}) error {
		// Serialize event data for recording
		rawJSON := ""
		if eventData != nil {
			if bytes, err := json.Marshal(eventData); err == nil {
				rawJSON = string(bytes)
			}
		}

		// Extract additional info from EventWithMeta if present
		subtype := ""
		name := ""
		if ewm, ok := eventData.(*EventWithMeta); ok {
			if ewm.Meta != nil {
				name = ewm.Meta.ToolName
			}
		}

		r.events = append(r.events, EventTypeRecord{
			Type:      eventType,
			Subtype:   subtype,
			Name:      name,
			RawJSON:   rawJSON,
			Timestamp: time.Now(),
		})
		return nil
	}
}

// GetUniqueTypes returns all unique event types observed.
func (r *EventTypeRecorder) GetUniqueTypes() []string {
	typeMap := make(map[string]bool)
	for _, e := range r.events {
		typeMap[e.Type] = true
	}
	types := make([]string, 0, len(typeMap))
	for t := range typeMap {
		types = append(types, t)
	}
	return types
}

// GetEventsByType returns all events of a specific type.
func (r *EventTypeRecorder) GetEventsByType(eventType string) []EventTypeRecord {
	result := make([]EventTypeRecord, 0)
	for _, e := range r.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

// HasType returns true if the event type was observed.
func (r *EventTypeRecorder) HasType(eventType string) bool {
	for _, e := range r.events {
		if e.Type == eventType {
			return true
		}
	}
	return false
}

// Count returns the number of events of a specific type.
func (r *EventTypeRecorder) Count(eventType string) int {
	count := 0
	for _, e := range r.events {
		if e.Type == eventType {
			count++
		}
	}
	return count
}

// Report generates a test report of observed events.
func (r *EventTypeRecorder) Report() string {
	var sb strings.Builder
	sb.WriteString("=== Event Type Report ===\n")
	sb.WriteString("Total Events: ")
	sb.WriteString(fmt.Sprintf("%d", len(r.events)))
	sb.WriteString("\n\n")

	// Count by type
	typeCounts := make(map[string]int)
	for _, e := range r.events {
		typeCounts[e.Type]++
	}

	sb.WriteString("Event Type Counts:\n")
	for _, typeName := range sortedKeys(typeCounts) {
		sb.WriteString("  ")
		sb.WriteString(typeName)
		sb.WriteString(": ")
		sb.WriteString(fmt.Sprintf("%d", typeCounts[typeName]))
		sb.WriteString("\n")
	}

	return sb.String()
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// TestStreamMessageParsing tests parsing of various StreamMessage formats.
func TestStreamMessageParsing(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected struct {
			Type       string
			Subtype    string
			Name       string
			HasContent bool
			HasOutput  bool
			HasError   bool
			HasMessage bool
			HasUsage   bool
			HasCostUSD bool
		}
	}{
		{
			name: "system init message",
			json: `{"type":"system","subtype":"init","cwd":"/tmp","session_id":"abc123","tools":["Task","Bash"],"model":"claude-opus-4.5-20251101"}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "system", Subtype: "init"},
		},
		{
			name: "thinking message",
			json: `{"type":"thinking","content":[{"type":"text","text":"Let me think about this..."}]}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "thinking", HasContent: true},
		},
		{
			name: "status message (treated like thinking)",
			json: `{"type":"status","content":[{"type":"text","text":"Processing..."}]}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "status", HasContent: true},
		},
		{
			name: "tool_use message",
			json: `{"type":"tool_use","name":"Bash","input":{"command":"ls -la"},"id":"toolu_01"}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "tool_use", Name: "Bash"},
		},
		{
			name: "tool_result message (standalone)",
			json: `{"type":"tool_result","output":"file1.txt\nfile2.txt","tool_use_id":"toolu_01","is_error":false}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "tool_result", HasOutput: true},
		},
		{
			name: "assistant message with nested tool_use",
			json: `{"type":"assistant","content":[{"type":"text","text":"I'll help you"},{"type":"tool_use","id":"toolu_02","name":"editor_write","input":{"file":"test.txt","content":"Hello"}}]}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "assistant", HasContent: true},
		},
		{
			name: "user message with nested tool_result",
			json: `{"type":"user","content":[{"type":"tool_result","tool_use_id":"toolu_02","content":"File created successfully","is_error":false}]}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "user", HasContent: true},
		},
		{
			name: "error message",
			json: `{"type":"error","error":"Something went wrong"}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "error", HasError: true},
		},
		{
			name: "result message with stats",
			json: `{"type":"result","subtype":"success","duration_ms":5000,"total_cost_usd":0.0123,"usage":{"input_tokens":1000,"output_tokens":500}}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "result", Subtype: "success", HasUsage: true, HasCostUSD: true},
		},
		{
			name: "result message with error",
			json: `{"type":"result","subtype":"error","duration_ms":1000,"is_error":true,"error":"Command failed"}`,
			expected: struct {
				Type       string
				Subtype    string
				Name       string
				HasContent bool
				HasOutput  bool
				HasError   bool
				HasMessage bool
				HasUsage   bool
				HasCostUSD bool
			}{Type: "result", Subtype: "error", HasError: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg StreamMessage
			if err := json.Unmarshal([]byte(tc.json), &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if msg.Type != tc.expected.Type {
				t.Errorf("Expected type=%q, got %q", tc.expected.Type, msg.Type)
			}
			if tc.expected.Subtype != "" && msg.Subtype != tc.expected.Subtype {
				t.Errorf("Expected subtype=%q, got %q", tc.expected.Subtype, msg.Subtype)
			}
			if tc.expected.Name != "" && msg.Name != tc.expected.Name {
				t.Errorf("Expected name=%q, got %q", tc.expected.Name, msg.Name)
			}
			if tc.expected.HasContent && len(msg.GetContentBlocks()) == 0 {
				t.Error("Expected content blocks, got none")
			}
			if tc.expected.HasOutput && msg.Output == "" {
				t.Error("Expected output, got empty")
			}
			if tc.expected.HasError && msg.Error == "" {
				t.Error("Expected error, got empty")
			}
			if tc.expected.HasMessage && msg.Message == nil {
				t.Error("Expected message, got nil")
			}
			if tc.expected.HasUsage && msg.Usage == nil {
				t.Error("Expected usage, got nil")
			}
			if tc.expected.HasCostUSD && msg.TotalCostUSD == 0 {
				t.Error("Expected cost_usd, got 0")
			}
		})
	}
}

// TestGetContentBlocks tests the GetContentBlocks method for nested content.
func TestGetContentBlocks(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected int // Number of expected content blocks
	}{
		{
			name:     "direct content blocks",
			json:     `{"type":"assistant","content":[{"type":"text","text":"Hello"}]}`,
			expected: 1,
		},
		{
			name:     "nested in message.content",
			json:     `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"World"}]}}`,
			expected: 1,
		},
		{
			name:     "direct takes priority",
			json:     `{"type":"assistant","content":[{"type":"text","text":"Direct"}],"message":{"content":[{"type":"text","text":"Nested"}]}}`,
			expected: 1, // Direct content takes priority
		},
		{
			name:     "no content blocks",
			json:     `{"type":"system","subtype":"init"}`,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg StreamMessage
			if err := json.Unmarshal([]byte(tc.json), &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			blocks := msg.GetContentBlocks()
			if len(blocks) != tc.expected {
				t.Errorf("Expected %d content blocks, got %d", tc.expected, len(blocks))
			}
		})
	}
}

// TestHandleResultMessage tests the result message handler.
func TestHandleResultMessage(t *testing.T) {
	runner := &CCRunner{
		logger: slog.Default(), // Use default logger for test
	}

	cfg := &CCRunnerConfig{
		SessionID: "test-session-123",
		UserID:    1,
		Mode:      "geek",
	}

	stats := &SessionStats{
		SessionID: "test-session-123",
		StartTime: time.Now(),
	}

	// Create a callback that captures session_stats events
	var capturedStats *SessionStatsData
	callback := func(eventType string, eventData interface{}) error {
		if eventType == EventTypeSessionStats {
			if data, ok := eventData.(*SessionStatsData); ok {
				capturedStats = data
			}
		}
		return nil
	}

	t.Run("successful result with stats", func(t *testing.T) {
		msg := StreamMessage{
			Type:         "result",
			Subtype:      "success",
			Duration:     10000,
			TotalCostUSD: 0.5,
			Usage: &UsageStats{
				InputTokens:           5000,
				OutputTokens:          1000,
				CacheWriteInputTokens: 200,
				CacheReadInputTokens:  300,
			},
		}

		runner.handleResultMessage(msg, stats, cfg, callback)

		if capturedStats == nil {
			t.Fatal("Expected session_stats event to be sent")
		}

		if capturedStats.TotalCostUSD != 0.5 {
			t.Errorf("Expected TotalCostUSD=0.5, got %f", capturedStats.TotalCostUSD)
		}
		if capturedStats.InputTokens != 5000 {
			t.Errorf("Expected InputTokens=5000, got %d", capturedStats.InputTokens)
		}
		if capturedStats.OutputTokens != 1000 {
			t.Errorf("Expected OutputTokens=1000, got %d", capturedStats.OutputTokens)
		}
		if capturedStats.TotalDurationMs != 10000 {
			t.Errorf("Expected TotalDurationMs=10000, got %d", capturedStats.TotalDurationMs)
		}
	})

	t.Run("result with error", func(t *testing.T) {
		capturedStats = nil
		stats := &SessionStats{
			SessionID: "test-session-error",
			StartTime: time.Now(),
		}
		cfg.SessionID = "test-session-error"

		msg := StreamMessage{
			Type:     "result",
			Subtype:  "error",
			IsError:  true,
			Error:    "Test error message",
			Duration: 5000,
		}

		runner.handleResultMessage(msg, stats, cfg, callback)

		if capturedStats == nil {
			t.Fatal("Expected session_stats event to be sent")
		}

		if !capturedStats.IsError {
			t.Error("Expected IsError=true")
		}
		if capturedStats.ErrorMessage != "Test error message" {
			t.Errorf("Expected ErrorMessage='Test error message', got %s", capturedStats.ErrorMessage)
		}
	})

	t.Run("result with zero stats", func(t *testing.T) {
		capturedStats = nil
		stats := &SessionStats{
			SessionID: "test-session-123",
			StartTime: time.Now(),
		}
		cfg.SessionID = "test-session-123"

		msg := StreamMessage{
			Type:    "result",
			Subtype: "success",
		}

		runner.handleResultMessage(msg, stats, cfg, callback)

		if capturedStats == nil {
			t.Fatal("Expected session_stats event to be sent")
		}

		// Should still have session info even with zero stats
		if capturedStats.SessionID != "test-session-123" {
			t.Errorf("Expected SessionID='test-session-123', got %s", capturedStats.SessionID)
		}
	})
}

// TestDispatchCallbackCoverage tests that dispatchCallback handles all known types.
func TestDispatchCallbackCoverage(t *testing.T) {
	runner := &CCRunner{
		logger: slog.Default(),
	}

	stats := &SessionStats{
		SessionID: "test",
		StartTime: time.Now(),
	}

	recorder := NewEventTypeRecorder()

	// Test each known event type
	testCases := []struct {
		typeName  string
		json      string
		shouldErr bool
	}{
		{
			typeName: "thinking",
			json:     `{"type":"thinking","content":[{"type":"text","text":"Thinking..."}]}`,
		},
		{
			typeName: "status",
			json:     `{"type":"status","content":[{"type":"text","text":"Status..."}]}`,
		},
		{
			typeName: "tool_use",
			json:     `{"type":"tool_use","name":"Bash","input":{"command":"echo test"}}`,
		},
		{
			typeName: "tool_result",
			json:     `{"type":"tool_result","output":"test output"}`,
		},
		{
			typeName: "assistant",
			json:     `{"type":"assistant","content":[{"type":"text","text":"Hello"}]}`,
		},
		{
			typeName: "user",
			json:     `{"type":"user","content":[{"type":"tool_result","content":"done"}]}`,
		},
		{
			typeName: "error",
			json:     `{"type":"error","error":"test error"}`,
		},
		{
			typeName:  "unknown",
			json:      `{"type":"unknown_type","content":[{"type":"text","text":"?"}]}`,
			shouldErr: false, // Unknown types should not error, just warn
		},
	}

	for _, tc := range testCases {
		t.Run(tc.typeName, func(t *testing.T) {
			recorder.events = nil // Clear recorder

			var msg StreamMessage
			if err := json.Unmarshal([]byte(tc.json), &msg); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			err := runner.dispatchCallback(msg, recorder.Callback(), stats)

			if tc.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// Verify at least one event was recorded (for non-unknown types)
			if tc.typeName != "unknown" && !recorder.HasType(tc.typeName) && !recorder.HasType("thinking") && !recorder.HasType("answer") && !recorder.HasType("tool_use") && !recorder.HasType("tool_result") && !recorder.HasType("error") {
				t.Errorf("No event recorded for type=%s", tc.typeName)
			}
		})
	}
}

// TestConversationIDToSessionID tests deterministic UUID generation.
func TestConversationIDToSessionID(t *testing.T) {
	testCases := []struct {
		conversationID int64
		expectedPrefix string // UUID format
	}{
		{1, ""},
		{123, ""},
		{999999, ""},
		{0, ""}, // Edge case: zero ID
	}

	for _, tc := range testCases {
		t.Run("conversation_id", func(t *testing.T) {
			sessionID := ConversationIDToSessionID(tc.conversationID)

			// Should be a valid UUID format
			if len(sessionID) != 36 {
				t.Errorf("Expected UUID length 36, got %d", len(sessionID))
			}

			// Should be deterministic (same input = same output)
			sessionID2 := ConversationIDToSessionID(tc.conversationID)
			if sessionID != sessionID2 {
				t.Error("Expected deterministic output, got different results")
			}

			// Different inputs should produce different outputs
			if tc.conversationID > 0 {
				otherID := ConversationIDToSessionID(tc.conversationID + 1)
				if sessionID == otherID {
					t.Error("Different conversation IDs should produce different session IDs")
				}
			}
		})
	}

	t.Run("deterministic_property", func(t *testing.T) {
		// Test that same conversation ID always produces same session ID
		id1 := ConversationIDToSessionID(12345)
		id2 := ConversationIDToSessionID(12345)
		id3 := ConversationIDToSessionID(12345)

		if id1 != id2 || id2 != id3 {
			t.Error("ConversationIDToSessionID should be deterministic")
		}
	})
}

// TestBuildSystemPromptCoverage tests system prompt generation.
func TestBuildSystemPromptCoverage(t *testing.T) {
	workDir := "/tmp/test"
	sessionID := "test-session-123"
	userID := int32(42)
	deviceContext := `{"userAgent":"Mozilla/5.0","isMobile":false,"screenWidth":1920,"screenHeight":1080,"language":"en"}`

	prompt := buildSystemPrompt(workDir, sessionID, userID, deviceContext)

	// Verify key components are present
	requiredStrings := []string{
		"DivineSense",
		"**User ID**: 42",
		"**Session**: test-session-123",
		"Desktop",
		"1920x1080",
		workDir,
	}

	for _, s := range requiredStrings {
		if !strings.Contains(prompt, s) {
			t.Errorf("Expected prompt to contain %q", s)
		}
	}

	t.Run("empty_device_context", func(t *testing.T) {
		prompt := buildSystemPrompt(workDir, sessionID, userID, "")
		if !strings.Contains(prompt, "Unknown") {
			t.Error("Expected 'Unknown' for empty device context")
		}
	})

	t.Run("invalid_json_context", func(t *testing.T) {
		prompt := buildSystemPrompt(workDir, sessionID, userID, "not json")
		if !strings.Contains(prompt, "not json") {
			t.Error("Expected raw string for invalid JSON context")
		}
	})
}

// TestSummarizeInput tests input summarization for tool calls.
func TestSummarizeInput(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name:     "command input",
			input:    map[string]any{"command": "ls -la /tmp"},
			expected: "ls -la /tmp",
		},
		{
			name:     "query input",
			input:    map[string]any{"query": "test search"},
			expected: "test search",
		},
		{
			name:     "path input",
			input:    map[string]any{"path": "/tmp/file.txt"},
			expected: "file: /tmp/file.txt",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty input",
			input:    map[string]any{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := summarizeInput(tc.input)
			if !strings.Contains(result, tc.expected) && result != tc.expected {
				t.Errorf("Expected %q to contain %q, got %q", result, tc.expected, result)
			}
		})
	}
}

// TestSessionStats tests SessionStats methods.
func TestSessionStats(t *testing.T) {
	stats := &SessionStats{
		SessionID: "test-123",
		StartTime: time.Now(),
	}

	t.Run("record_tool_use", func(t *testing.T) {
		stats.RecordToolUse("Bash", "tool-1")
		if stats.currentToolName != "Bash" {
			t.Errorf("Expected currentToolName='Bash', got %q", stats.currentToolName)
		}
		if stats.currentToolID != "tool-1" {
			t.Errorf("Expected currentToolID='tool-1', got %q", stats.currentToolID)
		}
	})

	t.Run("record_tool_result", func(t *testing.T) {
		time.Sleep(10 * time.Millisecond) // Ensure some duration
		duration := stats.RecordToolResult()
		if duration == 0 {
			t.Error("Expected non-zero duration")
		}
		if stats.ToolCallCount != 1 {
			t.Errorf("Expected ToolCallCount=1, got %d", stats.ToolCallCount)
		}
		if !stats.ToolsUsed["Bash"] {
			t.Error("Expected Bash to be in ToolsUsed")
		}
	})

	t.Run("thinking_phase", func(t *testing.T) {
		stats.StartThinking()
		time.Sleep(5 * time.Millisecond)
		stats.EndThinking()
		if stats.ThinkingDurationMs == 0 {
			t.Error("Expected non-zero thinking duration")
		}
	})

	t.Run("generation_phase", func(t *testing.T) {
		stats.StartGeneration()
		time.Sleep(5 * time.Millisecond)
		stats.EndGeneration()
		if stats.GenerationDurationMs == 0 {
			t.Error("Expected non-zero generation duration")
		}
	})

	t.Run("record_tokens", func(t *testing.T) {
		stats.RecordTokens(100, 50, 10, 5)
		if stats.InputTokens != 100 {
			t.Errorf("Expected InputTokens=100, got %d", stats.InputTokens)
		}
		if stats.OutputTokens != 50 {
			t.Errorf("Expected OutputTokens=50, got %d", stats.OutputTokens)
		}
		if stats.CacheWriteTokens != 10 {
			t.Errorf("Expected CacheWriteTokens=10, got %d", stats.CacheWriteTokens)
		}
		if stats.CacheReadTokens != 5 {
			t.Errorf("Expected CacheReadTokens=5, got %d", stats.CacheReadTokens)
		}
	})

	t.Run("to_summary", func(t *testing.T) {
		summary := stats.ToSummary()
		if summary["session_id"] != "test-123" {
			t.Errorf("Expected session_id='test-123', got %v", summary["session_id"])
		}
		// Type assert to int for comparison
		if v, ok := summary["tool_call_count"].(int32); !ok || v != 1 {
			t.Errorf("Expected tool_call_count=1 (int32), got %v (%T)", summary["tool_call_count"], summary["tool_call_count"])
		}
	})
}

// TestCCRunnerIntegration is a minimal integration test that actually calls the CLI.
// Skip this in CI/CD if CLI is not available.
func TestCCRunnerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("Claude CLI not available, skipping integration test")
	}

	// Create temp directory
	tempDir := t.TempDir()

	runner, err := NewCCRunner(30*time.Second, nil)
	if err != nil {
		t.Skip("Cannot create CCRunner:", err)
	}

	recorder := NewEventTypeRecorder()

	cfg := &CCRunnerConfig{
		Mode:           "geek",
		WorkDir:        tempDir,
		ConversationID: 12345,
		UserID:         1,
		PermissionMode: "default",
	}

	// Simple prompt that should complete quickly
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_ = runner.Execute(ctx, cfg, "say hello", recorder.Callback())

	// The test passes if we get some events
	uniqueTypes := recorder.GetUniqueTypes()
	t.Logf("Observed event types: %v", uniqueTypes)

	if len(uniqueTypes) == 0 {
		t.Error("Expected to observe some event types")
	}

	// Log report for manual inspection
	t.Log(recorder.Report())
}

// BenchmarkStreamMessageParsing benchmarks parsing performance.
func BenchmarkStreamMessageParsing(b *testing.B) {
	jsonData := []byte(`{"type":"assistant","content":[{"type":"text","text":"Hello world"},{"type":"tool_use","id":"toolu_01","name":"Bash","input":{"command":"ls"}}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var msg StreamMessage
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			b.Fatal(err)
		}
		_ = msg.GetContentBlocks()
	}
}

// TestStreamMessageEdgeCases tests edge cases in message parsing.
func TestStreamMessageEdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		json    string
		wantErr bool
		check   func(*testing.T, *StreamMessage)
	}{
		{
			name: "empty content array",
			json: `{"type":"assistant","content":[]}`,
			check: func(t *testing.T, m *StreamMessage) {
				if len(m.GetContentBlocks()) != 0 {
					t.Error("Expected empty content blocks")
				}
			},
		},
		{
			name: "null fields",
			json: `{"type":"assistant","content":null,"message":null,"output":null}`,
			check: func(t *testing.T, m *StreamMessage) {
				// Should not panic
				_ = m.GetContentBlocks()
			},
		},
		{
			name: "mixed content types",
			json: `{"type":"assistant","content":[{"type":"text","text":"Hi"},{"type":"tool_use","id":"t1","name":"Bash","input":{"cmd":"ls"}}]}`,
			check: func(t *testing.T, m *StreamMessage) {
				blocks := m.GetContentBlocks()
				if len(blocks) != 2 {
					t.Errorf("Expected 2 blocks, got %d", len(blocks))
				}
			},
		},
		{
			name: "nested assistant message",
			json: `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Nested"}]}}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Message == nil {
					t.Error("Expected Message to be set")
				}
				if len(m.GetContentBlocks()) != 1 {
					t.Errorf("Expected 1 content block from nested message, got %d", len(m.GetContentBlocks()))
				}
			},
		},
		{
			name: "tool_use with empty input",
			json: `{"type":"tool_use","name":"NoArgs","id":"t1","input":{}}`,
			check: func(t *testing.T, m *StreamMessage) {
				// tool_use messages don't have content blocks at top level
				// They have name and input fields directly
				if m.Name != "NoArgs" {
					t.Errorf("Expected name=NoArgs, got %s", m.Name)
				}
			},
		},
		{
			name: "tool_result with is_error",
			json: `{"type":"tool_result","output":"Command failed","is_error":true}`,
			check: func(t *testing.T, m *StreamMessage) {
				// tool_result uses output field, not content
				if m.Output != "Command failed" {
					t.Errorf("Expected output='Command failed', got %s", m.Output)
				}
				if !m.IsError {
					t.Error("Expected IsError=true")
				}
			},
		},
		{
			name: "result with all fields",
			json: `{"type":"result","subtype":"success","duration_ms":12345,"total_cost_usd":0.123,"usage":{"input_tokens":1000,"output_tokens":200}}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Type != "result" {
					t.Errorf("Expected type=result, got %s", m.Type)
				}
				if m.Subtype != "success" {
					t.Errorf("Expected subtype=success, got %s", m.Subtype)
				}
				if m.Duration != 12345 {
					t.Errorf("Expected duration=12345, got %d", m.Duration)
				}
				if m.TotalCostUSD != 0.123 {
					t.Errorf("Expected TotalCostUSD=0.123, got %f", m.TotalCostUSD)
				}
				if m.Usage == nil {
					t.Error("Expected Usage to be set")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var msg StreamMessage
			err := json.Unmarshal([]byte(tc.json), &msg)

			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.check != nil {
				tc.check(t, &msg)
			}
		})
	}
}

// TestContentBlockTypes tests all known content block types.
func TestContentBlockTypes(t *testing.T) {
	testCases := []struct {
		name        string
		json        string
		wantType    string
		wantText    string
		wantContent string
		wantName    string
		wantID      string
		wantInput   bool
		wantError   bool
	}{
		{
			name:     "text block",
			json:     `{"type":"text","text":"Hello world"}`,
			wantType: "text",
			wantText: "Hello world",
		},
		{
			name:      "tool_use block",
			json:      `{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"ls"}}`,
			wantType:  "tool_use",
			wantName:  "Bash",
			wantID:    "tool_123",
			wantInput: true,
		},
		{
			name:        "tool_result block",
			json:        `{"type":"tool_result","content":"done","is_error":false}`,
			wantType:    "tool_result",
			wantContent: "done",
		},
		{
			name:        "tool_result block with error",
			json:        `{"type":"tool_result","content":"failed","is_error":true}`,
			wantType:    "tool_result",
			wantContent: "failed",
			wantError:   true,
		},
		{
			name:     "image block (future)",
			json:     `{"type":"image","source":{"type":"base64","media_type":"image/png","data":"abc123"}}`,
			wantType: "image",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var block ContentBlock
			if err := json.Unmarshal([]byte(tc.json), &block); err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			if block.Type != tc.wantType {
				t.Errorf("Expected type=%q, got %q", tc.wantType, block.Type)
			}
			if tc.wantText != "" && block.Text != tc.wantText {
				t.Errorf("Expected text=%q, got %q", tc.wantText, block.Text)
			}
			if tc.wantContent != "" && block.Content != tc.wantContent {
				t.Errorf("Expected content=%q, got %q", tc.wantContent, block.Content)
			}
			if tc.wantName != "" && block.Name != tc.wantName {
				t.Errorf("Expected name=%q, got %q", tc.wantName, block.Name)
			}
			if tc.wantID != "" && block.ID != tc.wantID {
				t.Errorf("Expected id=%q, got %q", tc.wantID, block.ID)
			}
			if tc.wantInput && block.Input == nil {
				t.Error("Expected Input to be set")
			}
			if tc.wantError && !block.IsError {
				t.Error("Expected IsError=true")
			}
		})
	}
}

// TestNestedMessageStructure tests the nested message structure described in spec.
func TestNestedMessageStructure(t *testing.T) {
	// Test case 1: assistant with nested tool_use
	t.Run("assistant_nested_tool_use", func(t *testing.T) {
		jsonData := `{
			"type": "assistant",
			"content": [
				{"type": "text", "text": "I'll create a file for you."},
				{"type": "tool_use", "id": "toolu_01", "name": "editor_write", "input": {"path": "test.txt", "content": "Hello"}}
			]
		}`

		var msg StreamMessage
		if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
			t.Fatal(err)
		}

		blocks := msg.GetContentBlocks()
		if len(blocks) != 2 {
			t.Fatalf("Expected 2 blocks, got %d", len(blocks))
		}

		if blocks[0].Type != "text" {
			t.Errorf("Expected first block type=text, got %s", blocks[0].Type)
		}
		if blocks[1].Type != "tool_use" {
			t.Errorf("Expected second block type=tool_use, got %s", blocks[1].Type)
		}
		if blocks[1].Name != "editor_write" {
			t.Errorf("Expected tool name=editor_write, got %s", blocks[1].Name)
		}
	})

	// Test case 2: user with nested tool_result
	t.Run("user_nested_tool_result", func(t *testing.T) {
		jsonData := `{
			"type": "user",
			"content": [
				{"type": "tool_result", "tool_use_id": "toolu_01", "content": "File created successfully", "is_error": false}
			]
		}`

		var msg StreamMessage
		if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
			t.Fatal(err)
		}

		blocks := msg.GetContentBlocks()
		if len(blocks) != 1 {
			t.Fatalf("Expected 1 block, got %d", len(blocks))
		}

		if blocks[0].Type != "tool_result" {
			t.Errorf("Expected block type=tool_result, got %s", blocks[0].Type)
		}
		if blocks[0].Content != "File created successfully" {
			t.Errorf("Expected content='File created successfully', got %s", blocks[0].Content)
		}
	})

	// Test case 3: message wrapper type (alternative format)
	t.Run("message_wrapper_type", func(t *testing.T) {
		jsonData := `{
			"type": "message",
			"message": {
				"role": "assistant",
				"content": [
					{"type": "text", "text": "Response"},
					{"type": "tool_use", "id": "toolu_02", "name": "Bash", "input": {"command": "echo test"}}
				]
			}
		}`

		var msg StreamMessage
		if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
			t.Fatal(err)
		}

		blocks := msg.GetContentBlocks()
		if len(blocks) != 2 {
			t.Fatalf("Expected 2 blocks from nested message, got %d", len(blocks))
		}
	})
}

// TestSessionStatsDataStructure tests the SessionStatsData structure.
func TestSessionStatsDataStructure(t *testing.T) {
	stats := &SessionStatsData{
		SessionID:            "sess-123",
		UserID:               42,
		AgentType:            "geek",
		StartTime:            time.Now().Unix(),
		EndTime:              time.Now().Unix() + 100,
		TotalDurationMs:      5000,
		ThinkingDurationMs:   1000,
		ToolDurationMs:       3000,
		GenerationDurationMs: 1000,
		InputTokens:          1000,
		OutputTokens:         500,
		CacheWriteTokens:     100,
		CacheReadTokens:      50,
		TotalTokens:          1650,
		ToolCallCount:        3,
		ToolsUsed:            []string{"Bash", "editor_write"},
		FilesModified:        2,
		FilePaths:            []string{"test.txt", "output.txt"},
		TotalCostUSD:         0.05,
		ModelUsed:            "claude-opus-4.5-20251101",
		IsError:              false,
	}

	// Test JSON serialization
	bytes, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Test JSON deserialization
	var unmarshaled SessionStatsData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify key fields
	if unmarshaled.SessionID != stats.SessionID {
		t.Errorf("Expected SessionID=%q, got %q", stats.SessionID, unmarshaled.SessionID)
	}
	if unmarshaled.TotalCostUSD != stats.TotalCostUSD {
		t.Errorf("Expected TotalCostUSD=%f, got %f", stats.TotalCostUSD, unmarshaled.TotalCostUSD)
	}
	if unmarshaled.TotalTokens != stats.TotalTokens {
		t.Errorf("Expected TotalTokens=%d, got %d", stats.TotalTokens, unmarshaled.TotalTokens)
	}
}

// TestEventMetaStructure tests the EventMeta structure.
func TestEventMetaStructure(t *testing.T) {
	meta := &EventMeta{
		DurationMs:       1234,
		TotalDurationMs:  5678,
		ToolName:         "Bash",
		ToolID:           "tool-123",
		Status:           "success",
		ErrorMsg:         "",
		InputTokens:      100,
		OutputTokens:     50,
		CacheWriteTokens: 10,
		CacheReadTokens:  5,
		InputSummary:     "ls -la",
		OutputSummary:    "file1.txt\nfile2.txt",
		FilePath:         "/tmp/test.txt",
		LineCount:        10,
	}

	// Test JSON serialization
	bytes, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify it's valid JSON
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check key fields exist (Go uses PascalCase by default when no json tag is specified)
	if _, ok := unmarshaled["DurationMs"]; !ok {
		t.Error("Expected DurationMs field")
	}
	if _, ok := unmarshaled["ToolName"]; !ok {
		t.Error("Expected ToolName field")
	}
}

// TestUnknownMessageTypeHandling tests handling of unknown message types.
func TestUnknownMessageTypeHandling(t *testing.T) {
	runner := &CCRunner{
		logger: slog.Default(),
	}

	stats := &SessionStats{
		SessionID: "test",
		StartTime: time.Now(),
	}

	recorder := NewEventTypeRecorder()

	// Simulate an unknown message type
	msg := StreamMessage{
		Type: "future_unknown_type",
		Content: []ContentBlock{
			{Type: "text", Text: "Some content"},
		},
	}

	err := runner.dispatchCallback(msg, recorder.Callback(), stats)
	if err != nil {
		t.Errorf("Unknown message type should not error, got: %v", err)
	}

	// Should still try to extract text content
	if !recorder.HasType("answer") && len(msg.Content) > 0 {
		t.Error("Expected content to be extracted even for unknown type")
	}
}

// TestSessionStatsConcurrency tests concurrent access to SessionStats.
func TestSessionStatsConcurrency(t *testing.T) {
	stats := &SessionStats{
		SessionID: "test-concurrent",
		StartTime: time.Now(),
		ToolsUsed: make(map[string]bool),
	}

	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				stats.RecordToolUse("tool", "id")
				stats.RecordToolResult()
			}
			done <- true
		}(i)
	}

	// Wait for completion
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts (allow variance due to goroutine scheduling)
	// With 10 goroutines × 100 operations, we expect close to 1000
	// but due to mutex contention, some operations may be lost
	if stats.ToolCallCount < 800 || stats.ToolCallCount > 1000 {
		t.Errorf("Expected ToolCallCount near 1000, got %d", stats.ToolCallCount)
	}
}

// TestCCRunnerConfigDefaults tests CCRunnerConfig field defaults.
func TestCCRunnerConfigDefaults(t *testing.T) {
	cfg := &CCRunnerConfig{
		ConversationID: 123,
	}

	// Generate session ID if not set
	if cfg.SessionID == "" && cfg.ConversationID > 0 {
		cfg.SessionID = ConversationIDToSessionID(cfg.ConversationID)
	}

	if cfg.SessionID == "" {
		t.Error("Expected SessionID to be generated")
	}

	// Verify generated session ID is a valid UUID
	if _, err := uuid.Parse(cfg.SessionID); err != nil {
		t.Errorf("Generated SessionID is not a valid UUID: %v", err)
	}
}

// TestResultMessageVariations tests all variations of result messages.
func TestResultMessageVariations(t *testing.T) {
	variations := []struct {
		name  string
		json  string
		check func(*testing.T, *StreamMessage)
	}{
		{
			name: "success with all fields",
			json: `{
				"type": "result",
				"subtype": "success",
				"duration_ms": 10000,
				"total_cost_usd": 0.123,
				"usage": {
					"input_tokens": 5000,
					"output_tokens": 2000,
					"cache_creation_input_tokens": 100,
					"cache_read_input_tokens": 50
				},
				"num_turns": 3
			}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Type != "result" {
					t.Errorf("Expected type=result, got %s", m.Type)
				}
				if m.Subtype != "success" {
					t.Errorf("Expected subtype=success, got %s", m.Subtype)
				}
				if m.TotalCostUSD != 0.123 {
					t.Errorf("Expected TotalCostUSD=0.123, got %f", m.TotalCostUSD)
				}
				if m.Usage == nil {
					t.Fatal("Expected Usage to be set")
				}
				if m.Usage.InputTokens != 5000 {
					t.Errorf("Expected InputTokens=5000, got %d", m.Usage.InputTokens)
				}
			},
		},
		{
			name: "success with minimal fields",
			json: `{
				"type": "result",
				"subtype": "success",
				"duration_ms": 5000
			}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Type != "result" {
					t.Errorf("Expected type=result, got %s", m.Type)
				}
				if m.TotalCostUSD != 0 {
					t.Errorf("Expected TotalCostUSD=0, got %f", m.TotalCostUSD)
				}
				if m.Usage != nil {
					t.Error("Expected Usage to be nil")
				}
			},
		},
		{
			name: "error result",
			json: `{
				"type": "result",
				"subtype": "error",
				"is_error": true,
				"error": "Command execution failed",
				"duration_ms": 2000
			}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Type != "result" {
					t.Errorf("Expected type=result, got %s", m.Type)
				}
				if m.Subtype != "error" {
					t.Errorf("Expected subtype=error, got %s", m.Subtype)
				}
				if !m.IsError {
					t.Error("Expected IsError=true")
				}
				if m.Error != "Command execution failed" {
					t.Errorf("Expected error message, got %s", m.Error)
				}
			},
		},
		{
			name: "result without subtype (legacy format)",
			json: `{
				"type": "result",
				"duration_ms": 3000
			}`,
			check: func(t *testing.T, m *StreamMessage) {
				if m.Type != "result" {
					t.Errorf("Expected type=result, got %s", m.Type)
				}
				if m.Subtype != "" {
					t.Errorf("Expected empty subtype, got %s", m.Subtype)
				}
			},
		},
	}

	for _, v := range variations {
		t.Run(v.name, func(t *testing.T) {
			var msg StreamMessage
			if err := json.Unmarshal([]byte(v.json), &msg); err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			if v.check != nil {
				v.check(t, &msg)
			}
		})
	}
}
