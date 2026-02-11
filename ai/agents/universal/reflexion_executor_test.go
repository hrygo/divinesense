// Package universal provides tests for Reflexion executor.
package universal

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai"
)

// TestReflexionExecutor_Name tests the executor name.
func TestReflexionExecutor_Name(t *testing.T) {
	exec := NewReflexionExecutor(3)
	if exec.Name() != "reflexion" {
		t.Errorf("Name() = %q, want 'reflexion'", exec.Name())
	}
}

// TestReflexionExecutor_StreamingSupported tests that Reflexion executor supports streaming.
func TestReflexionExecutor_StreamingSupported(t *testing.T) {
	exec := NewReflexionExecutor(3)
	if !exec.StreamingSupported() {
		t.Error("ReflexionExecutor should support streaming")
	}
}

// TestReflexionExecutor_MaxIterations tests max iterations configuration.
func TestReflexionExecutor_MaxIterations(t *testing.T) {
	tests := []struct {
		name          string
		maxIterations int
		expected      int
	}{
		{"zero", 0, 3},      // Default to defaultMaxRefinements + 1 = 3
		{"negative", -5, 3}, // Default to 3
		{"one", 1, 1},
		{"two", 2, 2},
		{"five", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := NewReflexionExecutor(tt.maxIterations)
			if exec.maxIterations != tt.expected {
				t.Errorf("maxIterations = %d, want %d", exec.maxIterations, tt.expected)
			}
		})
	}
}

// TestReflexionExecutor_CalculateOverallQuality tests quality calculation.
func TestReflexionExecutor_CalculateOverallQuality(t *testing.T) {
	exec := NewReflexionExecutor(3)

	tests := []struct {
		name               string
		report             ReflectionReport
		expectedMinQuality float64
		expectedMaxQuality float64
	}{
		{
			name: "perfect quality",
			report: ReflectionReport{
				Accuracy:     1.0,
				Completeness: 1.0,
				Clarity:      1.0,
			},
			expectedMinQuality: 1.0,
			expectedMaxQuality: 1.0,
		},
		{
			name: "medium quality",
			report: ReflectionReport{
				Accuracy:     0.8,
				Completeness: 0.7,
				Clarity:      0.9,
			},
			expectedMinQuality: 0.78, // 0.8*0.4 + 0.7*0.35 + 0.9*0.25 = 0.785
			expectedMaxQuality: 0.80,
		},
		{
			name: "low quality",
			report: ReflectionReport{
				Accuracy:     0.5,
				Completeness: 0.4,
				Clarity:      0.6,
			},
			expectedMinQuality: 0.48, // 0.5*0.4 + 0.4*0.35 + 0.6*0.25 = 0.49
			expectedMaxQuality: 0.50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality := exec.calculateOverallQuality(&tt.report)
			if quality < tt.expectedMinQuality || quality > tt.expectedMaxQuality {
				t.Errorf("calculateOverallQuality() = %.2f, want in range [%.2f, %.2f]",
					quality, tt.expectedMinQuality, tt.expectedMaxQuality)
			}
		})
	}
}

// TestReflexionExecutor_ExtractJSON tests JSON extraction from responses.
func TestReflexionExecutor_ExtractJSON(t *testing.T) {
	exec := NewReflexionExecutor(3)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON",
			input:    `{"accuracy": 0.8, "completeness": 0.7}`,
			expected: `{"accuracy": 0.8, "completeness": 0.7}`,
		},
		{
			name:     "JSON with prefix text",
			input:    `Here's my evaluation:\n{"accuracy": 0.8}\nEnd of report`,
			expected: `{"accuracy": 0.8}`,
		},
		{
			name:     "JSON with nested braces",
			input:    `{"issues": ["problem 1", "problem 2"], "accuracy": 0.8}`,
			expected: `{"issues": ["problem 1", "problem 2"], "accuracy": 0.8}`,
		},
		{
			name:     "JSON with escaped quotes in strings",
			input:    `{"suggestions": ["Use \"clear\" language"]}`,
			expected: `{"suggestions": ["Use \"clear\" language"]}`,
		},
		{
			name:     "no JSON found",
			input:    `This is just plain text with no JSON structure`,
			expected: `This is just plain text with no JSON structure`,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("extractJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// mockReflexionLLM for testing Reflexion executor.
type mockReflexionLLM struct {
	chatFunc func(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error)
}

func (m *mockReflexionLLM) Chat(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages)
	}
	return "Mock response", &ai.LLMCallStats{}, nil
}

func (m *mockReflexionLLM) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, <-chan *ai.LLMCallStats, <-chan error) {
	// Not used in Reflexion tests
	contentChan := make(chan string, 1)
	contentChan <- "Mock response"
	close(contentChan)

	statsChan := make(chan *ai.LLMCallStats, 1)
	statsChan <- &ai.LLMCallStats{}
	close(statsChan)

	errChan := make(chan error, 1)
	close(errChan)

	return contentChan, statsChan, errChan
}

func (m *mockReflexionLLM) ChatWithTools(
	ctx context.Context,
	messages []ai.Message,
	tools []ai.ToolDescriptor,
) (*ai.ChatResponse, *ai.LLMCallStats, error) {
	// Not used in Reflexion tests, but required for interface compliance
	content, stats, err := m.Chat(ctx, messages)
	return &ai.ChatResponse{Content: content}, stats, err
}

// Warmup is a no-op for the mock.
func (m *mockReflexionLLM) Warmup(ctx context.Context) {
	// No-op for mock
}

// TestReflexionExecutor_Reflect_QualityThresholdMet tests reflection when quality meets threshold.
func TestReflexionExecutor_Reflect_QualityThresholdMet(t *testing.T) {
	exec := NewReflexionExecutor(3)
	ctx := context.Background()

	// Mock LLM that returns high quality (no refinement needed)
	llm := &mockReflexionLLM{
		chatFunc: func(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
			return `{"accuracy": 0.95, "completeness": 0.9, "clarity": 0.95, "needs_refinement": false}`, &ai.LLMCallStats{}, nil
		},
	}

	stats := &ExecutionStats{}
	startTime := dummyTime()

	report, err := exec.reflect(ctx, "test input", "test answer", llm, stats, startTime)
	if err != nil {
		t.Fatalf("reflect() failed: %v", err)
	}

	if report.Accuracy != 0.95 {
		t.Errorf("Accuracy = %.2f, want 0.95", report.Accuracy)
	}
	if report.NeedsRefinement {
		t.Error("NeedsRefinement = true, want false (quality threshold met)")
	}
}

// TestReflexionExecutor_Reflect_QualityBelowThreshold tests reflection when quality is below threshold.
func TestReflexionExecutor_Reflect_QualityBelowThreshold(t *testing.T) {
	exec := NewReflexionExecutor(3)
	ctx := context.Background()

	// Mock LLM that returns low quality (refinement needed)
	llm := &mockReflexionLLM{
		chatFunc: func(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
			return `{
				"accuracy": 0.5,
				"completeness": 0.6,
				"clarity": 0.5,
				"issues": ["Missing key information", "Unclear structure"],
				"suggestions": ["Add more details", "Improve organization"],
				"needs_refinement": true
			}`, &ai.LLMCallStats{}, nil
		},
	}

	stats := &ExecutionStats{}
	startTime := dummyTime()

	report, err := exec.reflect(ctx, "test input", "test answer", llm, stats, startTime)
	if err != nil {
		t.Fatalf("reflect() failed: %v", err)
	}

	if report.Accuracy != 0.5 {
		t.Errorf("Accuracy = %.2f, want 0.5", report.Accuracy)
	}
	if !report.NeedsRefinement {
		t.Error("NeedsRefinement = false, want true (quality below threshold)")
	}
	if len(report.Issues) != 2 {
		t.Errorf("Issues length = %d, want 2", len(report.Issues))
	}
	if len(report.Suggestions) != 2 {
		t.Errorf("Suggestions length = %d, want 2", len(report.Suggestions))
	}
}

// TestReflexionExecutor_Reflect_JSONParseError tests reflection when JSON parsing fails.
func TestReflexionExecutor_Reflect_JSONParseError(t *testing.T) {
	exec := NewReflexionExecutor(3)
	ctx := context.Background()

	// Mock LLM that returns invalid JSON
	llm := &mockReflexionLLM{
		chatFunc: func(ctx context.Context, messages []ai.Message) (string, *ai.LLMCallStats, error) {
			return `This is not valid JSON at all`, &ai.LLMCallStats{}, nil
		},
	}

	stats := &ExecutionStats{}
	startTime := dummyTime()

	report, err := exec.reflect(ctx, "test input", "test answer", llm, stats, startTime)
	if err != nil {
		t.Fatalf("reflect() failed: %v", err)
	}

	// Should return default low-quality values forcing refinement
	if report.Accuracy != 0.5 {
		t.Errorf("Accuracy = %.2f, want 0.5 (default on parse error)", report.Accuracy)
	}
	if report.Completeness != 0.5 {
		t.Errorf("Completeness = %.2f, want 0.5 (default on parse error)", report.Completeness)
	}
	if report.Clarity != 0.5 {
		t.Errorf("Clarity = %.2f, want 0.5 (default on parse error)", report.Clarity)
	}
	if !report.NeedsRefinement {
		t.Error("NeedsRefinement = false, want true (force refinement on parse error)")
	}
}

// dummyTime returns a zero time for testing.
func dummyTime() time.Time {
	var t time.Time
	return t
}

// TestReflexionExecutor_Execute_StreamingSupported verifies executor interface compliance.
func TestReflexionExecutor_Execute_StreamingSupported(t *testing.T) {
	exec := NewReflexionExecutor(3)

	// Verify ExecutionStrategy interface compliance
	var _ ExecutionStrategy = exec

	// Test streaming capability
	if !exec.StreamingSupported() {
		t.Error("ReflexionExecutor should support streaming")
	}
}

// BenchmarkReflexionExecutor_ExtractJSON benchmarks JSON extraction.
func BenchmarkReflexionExecutor_ExtractJSON(b *testing.B) {
	exec := NewReflexionExecutor(3)
	input := `Here is the evaluation:\n{"accuracy": 0.85, "completeness": 0.9, "clarity": 0.8, "issues": ["minor issue"]}\nEnd`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exec.extractJSON(input)
	}
}
