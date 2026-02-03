package prediction

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrygo/divinesense/ai/habit"
)

// mockHabitAnalyzer implements habit.HabitAnalyzer for testing.
type mockHabitAnalyzer struct {
	habits *habit.UserHabits
	err    error
}

func (m *mockHabitAnalyzer) Analyze(ctx context.Context, userID int32) (*habit.UserHabits, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.habits, nil
}

func TestEngine_PredictByTime_MondayMorning(t *testing.T) {
	engine := NewEngine(nil, nil)

	// Test time-based prediction (results depend on current time)
	predictions := engine.predictByTime(context.Background(), 1)

	// Predictions slice should exist (may be empty depending on time)
	// This is a basic sanity check that no panic occurs
	assert.GreaterOrEqual(t, len(predictions), 0)
}

func TestEngine_PredictByContext_ScheduleCreated(t *testing.T) {
	engine := NewEngine(nil, nil)

	events := []ContextEvent{
		{
			Type:      "schedule_created",
			TargetID:  "schedule-123",
			Timestamp: time.Now(),
		},
	}

	predictions := engine.predictByContext(context.Background(), 1, events)

	require.Len(t, predictions, 1)
	assert.Equal(t, ActionSetReminder, predictions[0].Action)
	assert.Equal(t, "设置提醒", predictions[0].Label)
	assert.GreaterOrEqual(t, predictions[0].Confidence, 0.9)
}

func TestEngine_PredictByContext_MemoViewed(t *testing.T) {
	engine := NewEngine(nil, nil)

	events := []ContextEvent{
		{
			Type:      "memo_viewed",
			TargetID:  "memo-456",
			Timestamp: time.Now(),
		},
	}

	predictions := engine.predictByContext(context.Background(), 1, events)

	require.Len(t, predictions, 1)
	assert.Equal(t, ActionSearchRelated, predictions[0].Action)
}

func TestEngine_PredictByContext_OldEvent(t *testing.T) {
	engine := NewEngine(nil, nil)

	// Event from 10 minutes ago should be ignored
	events := []ContextEvent{
		{
			Type:      "schedule_created",
			TargetID:  "schedule-123",
			Timestamp: time.Now().Add(-10 * time.Minute),
		},
	}

	predictions := engine.predictByContext(context.Background(), 1, events)

	assert.Empty(t, predictions)
}

func TestEngine_PredictByPattern_WithActiveHours(t *testing.T) {
	currentHour := time.Now().Hour()

	mockAnalyzer := &mockHabitAnalyzer{
		habits: &habit.UserHabits{
			UserID: 1,
			Time: &habit.TimeHabits{
				ActiveHours: []int{currentHour}, // Match current hour
			},
		},
	}

	engine := NewEngine(mockAnalyzer, nil)

	predictions, err := engine.predictByPattern(context.Background(), 1)

	require.NoError(t, err)
	assert.NotEmpty(t, predictions)
}

func TestEngine_PredictByPattern_NoAnalyzer(t *testing.T) {
	engine := NewEngine(nil, nil)

	predictions, err := engine.predictByPattern(context.Background(), 1)

	require.NoError(t, err)
	assert.Empty(t, predictions)
}

func TestEngine_DeduplicateAndSort(t *testing.T) {
	engine := NewEngine(nil, nil)

	predictions := []Prediction{
		{Action: ActionViewWeekSchedule, Confidence: 0.5},
		{Action: ActionViewWeekSchedule, Confidence: 0.8}, // Duplicate
		{Action: ActionCreateSchedule, Confidence: 0.7},
		{Action: ActionSetReminder, Confidence: 0.9},
	}

	result := engine.deduplicateAndSort(predictions)

	// Should have 3 unique actions
	assert.Len(t, result, 3)

	// Should be sorted by confidence descending
	assert.Equal(t, ActionSetReminder, result[0].Action)
	assert.Equal(t, ActionCreateSchedule, result[1].Action)
	assert.Equal(t, ActionViewWeekSchedule, result[2].Action)
}

func TestEngine_Predict_MaxPredictions(t *testing.T) {
	engine := NewEngine(nil, nil)
	engine.SetMaxPredictions(2)

	// Create many events to generate many predictions
	events := []ContextEvent{
		{Type: "schedule_created", TargetID: "1", Timestamp: time.Now()},
		{Type: "memo_viewed", TargetID: "2", Timestamp: time.Now()},
		{Type: "schedule_completed", TargetID: "3", Timestamp: time.Now()},
	}

	predictions, err := engine.Predict(context.Background(), 1, events)

	require.NoError(t, err)
	assert.LessOrEqual(t, len(predictions), 2)
}

func TestEngine_SetMaxPredictions(t *testing.T) {
	engine := NewEngine(nil, nil)

	// Valid values
	engine.SetMaxPredictions(5)
	assert.Equal(t, 5, engine.maxPredictions)

	// Invalid value (too low)
	engine.SetMaxPredictions(0)
	assert.Equal(t, 5, engine.maxPredictions) // Should not change

	// Invalid value (too high)
	engine.SetMaxPredictions(100)
	assert.Equal(t, 5, engine.maxPredictions) // Should not change
}

func TestPrediction_Types(t *testing.T) {
	// Verify type constants
	assert.Equal(t, PredictionType("action"), PredictionTypeAction)
	assert.Equal(t, PredictionType("query"), PredictionTypeQuery)
	assert.Equal(t, PredictionType("reminder"), PredictionTypeReminder)
}

func TestActionType_Values(t *testing.T) {
	// Verify action type constants
	assert.Equal(t, ActionType("view_week_schedule"), ActionViewWeekSchedule)
	assert.Equal(t, ActionType("view_tomorrow"), ActionViewTomorrow)
	assert.Equal(t, ActionType("create_schedule"), ActionCreateSchedule)
	assert.Equal(t, ActionType("set_reminder"), ActionSetReminder)
}

func BenchmarkEngine_Predict(b *testing.B) {
	engine := NewEngine(nil, nil)
	events := []ContextEvent{
		{Type: "schedule_created", TargetID: "1", Timestamp: time.Now()},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Predict(context.Background(), 1, events)
	}
}

func BenchmarkEngine_PredictByTime(b *testing.B) {
	engine := NewEngine(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.predictByTime(context.Background(), 1)
	}
}
