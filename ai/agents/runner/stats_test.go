package runner

import (
	"sync"
	"testing"
	"time"
)

// TestSessionStats_FileDeduplication tests O(1) deduplication of file paths.
// This is critical to prevent duplicate counting when the same file is modified multiple times.
func TestSessionStats_FileDeduplication(t *testing.T) {
	s := &SessionStats{}

	// Record the same file 3 times
	s.RecordFileModification("/path/to/file.txt")
	s.RecordFileModification("/path/to/file.txt")
	s.RecordFileModification("/path/to/file.txt")

	// Should only count once
	if s.FilesModified != 1 {
		t.Errorf("FilesModified = %d, want 1 (deduplication failed)", s.FilesModified)
	}
	if len(s.FilePaths) != 1 {
		t.Errorf("FilePaths length = %d, want 1", len(s.FilePaths))
	}

	// Record a different file
	s.RecordFileModification("/path/to/other.txt")

	if s.FilesModified != 2 {
		t.Errorf("FilesModified = %d, want 2", s.FilesModified)
	}
}

// TestSessionStats_ConcurrentTokenTracking tests thread-safety of token accumulation.
// Multiple goroutines recording tokens should not lose data.
func TestSessionStats_ConcurrentTokenTracking(t *testing.T) {
	s := &SessionStats{}
	var wg sync.WaitGroup

	// Simulate concurrent token updates
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.RecordTokens(10, 5, 2, 1)
		}()
	}

	wg.Wait()

	// All tokens should be accumulated
	wantInput := int32(100 * 10)
	wantOutput := int32(100 * 5)
	if s.InputTokens != wantInput {
		t.Errorf("InputTokens = %d, want %d", s.InputTokens, wantInput)
	}
	if s.OutputTokens != wantOutput {
		t.Errorf("OutputTokens = %d, want %d", s.OutputTokens, wantOutput)
	}
}

// TestSessionStats_ToolCallTracking tests sequential tool call tracking.
func TestSessionStats_ToolCallTracking(t *testing.T) {
	s := &SessionStats{}

	// Simulate sequential tool calls (actual usage pattern)
	s.RecordToolUse("tool1", "id1")
	time.Sleep(1 * time.Millisecond)
	duration1 := s.RecordToolResult()

	s.RecordToolUse("tool2", "id2")
	time.Sleep(1 * time.Millisecond)
	duration2 := s.RecordToolResult()

	if s.ToolCallCount != 2 {
		t.Errorf("ToolCallCount = %d, want 2", s.ToolCallCount)
	}
	if duration1 < 1 {
		t.Errorf("duration1 = %d, want >= 1", duration1)
	}
	if duration2 < 1 {
		t.Errorf("duration2 = %d, want >= 1", duration2)
	}
	if !s.ToolsUsed["tool1"] || !s.ToolsUsed["tool2"] {
		t.Error("Both tools should be in ToolsUsed")
	}
}

// TestSessionStats_DurationTracking tests that thinking and generation durations are correctly accumulated.
func TestSessionStats_DurationTracking(t *testing.T) {
	s := &SessionStats{}

	// First thinking phase
	s.StartThinking()
	time.Sleep(10 * time.Millisecond)
	s.EndThinking()

	// Second thinking phase (should accumulate)
	s.StartThinking()
	time.Sleep(5 * time.Millisecond)
	s.EndThinking()

	// Total should be at least 15ms (with some tolerance for scheduler overhead)
	if s.ThinkingDurationMs < 15 {
		t.Errorf("ThinkingDurationMs = %d, want >= 15", s.ThinkingDurationMs)
	}

	// Generation phase
	s.StartGeneration()
	time.Sleep(8 * time.Millisecond)
	s.EndGeneration()

	if s.GenerationDurationMs < 8 {
		t.Errorf("GenerationDurationMs = %d, want >= 8", s.GenerationDurationMs)
	}
}

// TestSessionStats_FinalizeDuration handles ongoing phases at finalization.
func TestSessionStats_FinalizeDuration(t *testing.T) {
	s := &SessionStats{}

	s.StartThinking()
	time.Sleep(5 * time.Millisecond)
	// Don't call EndThinking - simulate incomplete session

	final := s.FinalizeDuration()

	// Should include the ongoing thinking phase
	if final.ThinkingDurationMs < 5 {
		t.Errorf("FinalizeDuration() ThinkingDurationMs = %d, want >= 5", final.ThinkingDurationMs)
	}
}

// TestSessionStats_ToolTrackingWithoutStart handles RecordToolResult without RecordToolUse.
func TestSessionStats_ToolTrackingWithoutStart(t *testing.T) {
	s := &SessionStats{}

	// Call RecordToolResult without prior RecordToolUse
	duration := s.RecordToolResult()

	// Should return 0 and not panic
	if duration != 0 {
		t.Errorf("RecordToolResult() without start = %d, want 0", duration)
	}
}
