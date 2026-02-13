package schedule

import (
	"testing"
	"time"
)

// mockContextProvider implements ContextProvider for testing
type mockContextProvider struct {
	extensions map[string]any
}

func newMockContextProvider() *mockContextProvider {
	return &mockContextProvider{
		extensions: make(map[string]any),
	}
}

func (m *mockContextProvider) GetExtension(key string) any {
	return m.extensions[key]
}

func (m *mockContextProvider) SetExtension(key string, value any) {
	m.extensions[key] = value
}

func TestGetWorkingState_NilProvider(t *testing.T) {
	got := GetWorkingState(nil)
	if got != nil {
		t.Errorf("GetWorkingState(nil) = %v, want nil", got)
	}
}

func TestGetWorkingState_EmptyProvider(t *testing.T) {
	provider := newMockContextProvider()
	got := GetWorkingState(provider)
	if got != nil {
		t.Errorf("GetWorkingState(empty provider) = %v, want nil", got)
	}
}

func TestGetWorkingState_WrongType(t *testing.T) {
	provider := newMockContextProvider()
	provider.SetExtension(ContextKey, "not a WorkingState")

	got := GetWorkingState(provider)
	if got != nil {
		t.Errorf("GetWorkingState(wrong type) = %v, want nil", got)
	}
}

func TestGetWorkingState_Success(t *testing.T) {
	provider := newMockContextProvider()
	ws := &WorkingState{
		CurrentStep: StepParsing,
		LastIntent:  "schedule_create",
	}
	provider.SetExtension(ContextKey, ws)

	got := GetWorkingState(provider)
	if got == nil {
		t.Fatal("GetWorkingState() returned nil")
	}
	if got.CurrentStep != StepParsing {
		t.Errorf("CurrentStep = %v, want StepParsing", got.CurrentStep)
	}
	if got.LastIntent != "schedule_create" {
		t.Errorf("LastIntent = %v, want schedule_create", got.LastIntent)
	}
}

func TestSetWorkingState_NilProvider(t *testing.T) {
	// Should not panic
	SetWorkingState(nil, &WorkingState{CurrentStep: StepIdle})
}

func TestSetWorkingState_Success(t *testing.T) {
	provider := newMockContextProvider()
	ws := &WorkingState{
		CurrentStep: StepConflictCheck,
	}

	SetWorkingState(provider, ws)

	got := provider.GetExtension(ContextKey)
	if got == nil {
		t.Fatal("SetWorkingState() did not set extension")
	}

	state, ok := got.(*WorkingState)
	if !ok {
		t.Fatalf("Extension is not *WorkingState, got %T", got)
	}
	if state.CurrentStep != StepConflictCheck {
		t.Errorf("CurrentStep = %v, want StepConflictCheck", state.CurrentStep)
	}
}

func TestSetWorkingState_NilState(t *testing.T) {
	provider := newMockContextProvider()
	provider.SetExtension(ContextKey, &WorkingState{CurrentStep: StepIdle})

	// Setting nil stores nil value (not the same as not having the key)
	// The current implementation stores nil, which is valid behavior
	SetWorkingState(provider, nil)

	// GetWorkingState handles nil extension correctly by returning nil
	got := GetWorkingState(provider)
	if got != nil {
		t.Errorf("GetWorkingState after SetWorkingState(nil) = %v, want nil", got)
	}
}

func TestNewWorkingState(t *testing.T) {
	got := NewWorkingState()

	if got == nil {
		t.Fatal("NewWorkingState() returned nil")
	}
	if got.CurrentStep != StepIdle {
		t.Errorf("CurrentStep = %v, want StepIdle", got.CurrentStep)
	}
	// Verify other fields are zero values
	if got.ProposedSchedule != nil {
		t.Errorf("ProposedSchedule should be nil, got %v", got.ProposedSchedule)
	}
	if got.LastIntent != "" {
		t.Errorf("LastIntent should be empty, got %v", got.LastIntent)
	}
}

func TestWorkingState_Fields(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)

	ws := &WorkingState{
		ProposedSchedule: &ScheduleDraft{
			StartTime:     &now,
			EndTime:       &later,
			Title:         "Test Meeting",
			Description:   "A test event",
			Location:      "Room 101",
			Timezone:      "Asia/Shanghai",
			OriginalInput: "test input",
			AllDay:        false,
			Confidence:    map[string]float32{"time": 0.9, "title": 0.8},
		},
		LastIntent:   "schedule_create",
		LastToolUsed: "create_schedule",
		CurrentStep:  StepConfirming,
	}

	// Verify all fields are set correctly
	if ws.ProposedSchedule.Title != "Test Meeting" {
		t.Errorf("Title = %v, want Test Meeting", ws.ProposedSchedule.Title)
	}
	if !ws.ProposedSchedule.StartTime.Equal(now) {
		t.Errorf("StartTime mismatch")
	}
	if ws.ProposedSchedule.Confidence["time"] != 0.9 {
		t.Errorf("Confidence[time] = %v, want 0.9", ws.ProposedSchedule.Confidence["time"])
	}
	if ws.CurrentStep != StepConfirming {
		t.Errorf("CurrentStep = %v, want StepConfirming", ws.CurrentStep)
	}
	if ws.LastIntent != "schedule_create" {
		t.Errorf("LastIntent = %v, want schedule_create", ws.LastIntent)
	}
	if ws.LastToolUsed != "create_schedule" {
		t.Errorf("LastToolUsed = %v, want create_schedule", ws.LastToolUsed)
	}
}

func TestWorkflowStep_Constants(t *testing.T) {
	// Verify all workflow steps are defined
	steps := []WorkflowStep{
		StepIdle,
		StepParsing,
		StepConflictCheck,
		StepConflictResolve,
		StepConfirming,
		StepCompleted,
	}

	expectedSteps := []string{
		"idle",
		"parsing",
		"conflict_check",
		"conflict_resolve",
		"confirming",
		"completed",
	}

	for i, step := range steps {
		if string(step) != expectedSteps[i] {
			t.Errorf("WorkflowStep[%d] = %v, want %v", i, step, expectedSteps[i])
		}
	}
}

func TestScheduleDraft_AllFields(t *testing.T) {
	now := time.Now()

	draft := &ScheduleDraft{
		StartTime:     &now,
		EndTime:       nil, // Optional end time
		Title:         "Event Title",
		Description:   "Event Description",
		Location:      "Event Location",
		Timezone:      "UTC",
		OriginalInput: "Original user input",
		AllDay:        true,
		Confidence: map[string]float32{
			"title":    0.95,
			"time":     0.80,
			"location": 0.60,
		},
	}

	// Verify all fields are set correctly
	if draft.Title != "Event Title" {
		t.Errorf("Title = %v, want Event Title", draft.Title)
	}
	if !draft.StartTime.Equal(now) {
		t.Errorf("StartTime mismatch")
	}
	if draft.Description != "Event Description" {
		t.Errorf("Description = %v, want Event Description", draft.Description)
	}
	if draft.Location != "Event Location" {
		t.Errorf("Location = %v, want Event Location", draft.Location)
	}
	if draft.Timezone != "UTC" {
		t.Errorf("Timezone = %v, want UTC", draft.Timezone)
	}
	if draft.OriginalInput != "Original user input" {
		t.Errorf("OriginalInput = %v, want Original user input", draft.OriginalInput)
	}
	if len(draft.Confidence) != 3 {
		t.Errorf("Confidence map should have 3 entries, got %d", len(draft.Confidence))
	}
	if !draft.AllDay {
		t.Error("AllDay should be true")
	}
	if draft.EndTime != nil {
		t.Error("EndTime should be nil for this test")
	}
}
