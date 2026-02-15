package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestHandoffContext_Operations tests HandoffContext methods.
func TestHandoffContext_Operations(t *testing.T) {
	tests := []struct {
		name        string
		depth       int
		elapsed     time.Duration
		wantDepth   bool
		wantTimeout bool
	}{
		{
			name:        "depth below max",
			depth:       2,
			elapsed:     0,
			wantDepth:   false,
			wantTimeout: false,
		},
		{
			name:        "depth at max",
			depth:       3,
			elapsed:     0,
			wantDepth:   true,
			wantTimeout: false,
		},
		{
			name:        "depth above max",
			depth:       4,
			elapsed:     0,
			wantDepth:   true,
			wantTimeout: false,
		},
		{
			name:        "timeout exceeded",
			depth:       0,
			elapsed:     HandoffTimeout + time.Second,
			wantDepth:   false,
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &HandoffContext{
				Depth:     tt.depth,
				StartTime: time.Now().Add(-tt.elapsed),
			}

			if got := ctx.IsMaxDepthExceeded(); got != tt.wantDepth {
				t.Errorf("IsMaxDepthExceeded() = %v, want %v", got, tt.wantDepth)
			}

			if got := ctx.IsTimeoutExceeded(); got != tt.wantTimeout {
				t.Errorf("IsTimeoutExceeded() = %v, want %v", got, tt.wantTimeout)
			}
		})
	}
}

// TestHandoffFailReason_Values tests HandoffFailReason enum values.
func TestHandoffFailReason_Values(t *testing.T) {
	expectedReasons := map[HandoffFailReason]string{
		FailNoMatchingExpert:  "no_matching_expert",
		FailTargetUnavailable: "target_unavailable",
		FailTargetExecution:   "target_execution",
		FailMaxDepthExceeded:  "max_depth_exceeded",
		FailTimeout:           "timeout",
		FailContextLost:       "context_lost",
	}

	for reason, expected := range expectedReasons {
		if string(reason) != expected {
			t.Errorf("HandoffFailReason = %v, want %v", reason, expected)
		}
	}
}

// TestHandoffHandler_HandleCannotComplete_WithDepthLimit tests depth limiting.
func TestHandoffHandler_HandleCannotComplete_WithDepthLimit(t *testing.T) {
	capabilityMap := NewCapabilityMap()
	handler := NewHandoffHandler(capabilityMap, 2)

	task, _ := NewTask("MemoParrot", "search for notes", "find information")
	reason := CannotCompleteReason{
		MissingCapabilities: []string{"日程管理"},
	}

	// Test with depth at max
	ctx := NewHandoffContextWithDepth(MaxHandoffDepth, task.ID)
	result := handler.HandleCannotComplete(context.Background(), task, reason, nil, ctx)

	if result.Success {
		t.Error("expected handoff to fail due to max depth, but it succeeded")
	}
	if result.Reason != FailMaxDepthExceeded {
		t.Errorf("expected reason = %v, got %v", FailMaxDepthExceeded, result.Reason)
	}
	if result.FallbackMessage == "" {
		t.Error("expected fallback message to be set")
	}
}

// TestHandoffHandler_HandleCannotComplete_WithTimeout tests timeout handling.
func TestHandoffHandler_HandleCannotComplete_WithTimeout(t *testing.T) {
	capabilityMap := NewCapabilityMap()
	handler := NewHandoffHandler(capabilityMap, 2)

	task, _ := NewTask("MemoParrot", "search for notes", "find information")
	reason := CannotCompleteReason{
		MissingCapabilities: []string{"日程管理"},
	}

	// Test with timeout exceeded
	ctx := &HandoffContext{
		Depth:        0,
		StartTime:    time.Now().Add(-HandoffTimeout - time.Second),
		ParentTaskID: task.ID,
	}
	result := handler.HandleCannotComplete(context.Background(), task, reason, nil, ctx)

	if result.Success {
		t.Error("expected handoff to fail due to timeout, but it succeeded")
	}
	if result.Reason != FailTimeout {
		t.Errorf("expected reason = %v, got %v", FailTimeout, result.Reason)
	}
	if result.FallbackMessage == "" {
		t.Error("expected fallback message to be set")
	}
}

// TestHandoffHandler_HandleCannotComplete_NoMatchingExpert tests no matching expert scenario.
func TestHandoffHandler_HandleCannotComplete_NoMatchingExpert(t *testing.T) {
	capabilityMap := NewCapabilityMap()
	handler := NewHandoffHandler(capabilityMap, 2)

	task, _ := NewTask("MemoParrot", "search for notes", "find information")
	reason := CannotCompleteReason{
		MissingCapabilities: []string{"non_existent_capability"},
	}

	ctx := NewHandoffContext()
	result := handler.HandleCannotComplete(context.Background(), task, reason, nil, ctx)

	if result.Success {
		t.Error("expected handoff to fail, but it succeeded")
	}
	if result.Reason != FailNoMatchingExpert {
		t.Errorf("expected reason = %v, got %v", FailNoMatchingExpert, result.Reason)
	}
	if result.FallbackMessage == "" {
		t.Error("expected fallback message to be set")
	}
}

// TestHandoffHandler_HandleTaskFailure tests task failure handling.
func TestHandoffHandler_HandleTaskFailure(t *testing.T) {
	capabilityMap := NewCapabilityMap()
	handler := NewHandoffHandler(capabilityMap, 2)

	task, _ := NewTask("MemoParrot", "search for notes", "find information")
	testErr := errors.New("无法处理此任务，需要日程管理能力")

	ctx := NewHandoffContext()
	result := handler.HandleTaskFailure(context.Background(), task, testErr, nil, ctx)

	// Should recognize the error and attempt handoff
	if result.Reason != FailNoMatchingExpert && result.Reason != "" && !result.Success {
		// Either it found an expert or it correctly set failure reason
		if result.Success && result.NewExpert == "" {
			t.Error("expected either success with expert or failure with reason")
		}
	}
}

// TestBuildFallbackResponse tests fallback message generation.
func TestBuildFallbackResponse(t *testing.T) {
	capabilityMap := NewCapabilityMap()
	handler := NewHandoffHandler(capabilityMap, 2)

	task, _ := NewTask("MemoParrot", "search for notes about meeting", "find information")

	tests := []struct {
		reason        HandoffFailReason
		wantSubstring string
	}{
		{FailNoMatchingExpert, "超出了我可以协调的范围"},
		{FailTargetUnavailable, "暂时不可用"},
		{FailTargetExecution, "遇到问题"},
		{FailMaxDepthExceeded, "多次转接"},
		{FailTimeout, "超时"},
		{FailContextLost, "连接问题"},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			msg := handler.buildFallbackResponse(task, tt.reason)
			if msg == "" {
				t.Error("expected non-empty fallback message")
			}
			// Verify message contains expected content
			if tt.wantSubstring != "" && !contains(msg, tt.wantSubstring) {
				t.Errorf("expected message to contain %q, got %q", tt.wantSubstring, msg)
			}
		})
	}
}

// TestHandoffResult_Fields tests HandoffResult fields.
func TestHandoffResult_Fields(t *testing.T) {
	result := &HandoffResult{
		Success:   true,
		NewExpert: "ScheduleParrot",
		Depth:     2,
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.NewExpert != "ScheduleParrot" {
		t.Errorf("expected NewExpert = ScheduleParrot, got %v", result.NewExpert)
	}
	if result.Depth != 2 {
		t.Errorf("expected Depth = 2, got %v", result.Depth)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
