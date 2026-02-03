package tools

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/server/service/schedule"
	"github.com/hrygo/divinesense/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockScheduleService is a mock implementation of schedule.Service for testing.
type MockScheduleService struct {
	findSchedulesError   error
	createScheduleError  error
	createScheduleResult *store.Schedule
	findSchedulesResult  []*schedule.ScheduleInstance
}

func (m *MockScheduleService) FindSchedules(ctx context.Context, userID int32, start, end time.Time) ([]*schedule.ScheduleInstance, error) {
	if m.findSchedulesError != nil {
		return nil, m.findSchedulesError
	}
	return m.findSchedulesResult, nil
}

func (m *MockScheduleService) CreateSchedule(ctx context.Context, userID int32, create *schedule.CreateScheduleRequest) (*store.Schedule, error) {
	if m.createScheduleError != nil {
		return nil, m.createScheduleError
	}
	if m.createScheduleResult != nil {
		return m.createScheduleResult, nil
	}
	// Return a mock schedule
	return &store.Schedule{
		ID:      1,
		Title:   create.Title,
		StartTs: create.StartTs,
	}, nil
}

func (m *MockScheduleService) UpdateSchedule(ctx context.Context, userID int32, id int32, update *schedule.UpdateScheduleRequest) (*store.Schedule, error) {
	return nil, nil
}

func (m *MockScheduleService) DeleteSchedule(ctx context.Context, userID int32, id int32) error {
	return nil
}

func (m *MockScheduleService) CheckConflicts(ctx context.Context, userID int32, startTs int64, endTs *int64, excludeIDs []int32) ([]*store.Schedule, error) {
	return nil, nil
}

// TestScheduleQueryTool_Run tests the ScheduleQueryTool.Run method.
func TestScheduleQueryTool_Run(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	// Create mock service with test data
	now := time.Now()
	mockService := &MockScheduleService{
		findSchedulesResult: []*schedule.ScheduleInstance{
			{
				ID:          1,
				Title:       "Test Meeting",
				StartTs:     now.Add(2 * time.Hour).Unix(),
				EndTs:       func() *int64 { ts := now.Add(3 * time.Hour).Unix(); return &ts }(),
				Timezone:    "Asia/Shanghai",
				IsRecurring: false,
			},
		},
	}

	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}

	tool := NewScheduleQueryTool(mockService, userIDGetter)

	t.Run("valid query", func(t *testing.T) {
		startTime := now.Format(time.RFC3339)
		endTime := now.Add(24 * time.Hour).Format(time.RFC3339)

		input := `{"start_time": "` + startTime + `", "end_time": "` + endTime + `"}`

		result, err := tool.Run(ctx, input)
		require.NoError(t, err)
		assert.Contains(t, result, "Found")
		assert.Contains(t, result, "schedule(s)")
	})

	t.Run("no schedules found", func(t *testing.T) {
		mockService.findSchedulesResult = []*schedule.ScheduleInstance{}

		startTime := now.Format(time.RFC3339)
		endTime := now.Add(24 * time.Hour).Format(time.RFC3339)

		input := `{"start_time": "` + startTime + `", "end_time": "` + endTime + `"}`

		result, err := tool.Run(ctx, input)
		require.NoError(t, err)
		assert.Contains(t, result, "No schedules found")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		input := `invalid json`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("missing start_time", func(t *testing.T) {
		input := `{"end_time": "2026-01-21T09:00:00Z"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_time is required")
	})

	t.Run("invalid time format", func(t *testing.T) {
		input := `{"start_time": "invalid-time", "end_time": "2026-01-21T09:00:00Z"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start_time format")
	})

	t.Run("end time before start time", func(t *testing.T) {
		input := `{"start_time": "2026-01-21T10:00:00Z", "end_time": "2026-01-21T09:00:00Z"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "end_time must be after start_time")
	})
}

// TestScheduleAddTool_Run tests the ScheduleAddTool.Run method.
func TestScheduleAddTool_Run(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)

	mockService := &MockScheduleService{}

	userIDGetter := func(ctx context.Context) int32 {
		return userID
	}

	tool := NewScheduleAddTool(mockService, userIDGetter)

	t.Run("valid schedule creation", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(2 * time.Hour).Format(time.RFC3339)
		endTime := now.Add(3 * time.Hour).Format(time.RFC3339)

		input := `{
			"title": "Test Meeting",
			"start_time": "` + startTime + `",
			"end_time": "` + endTime + `",
			"description": "Test Description",
			"location": "Conference Room"
		}`

		result, err := tool.Run(ctx, input)
		require.NoError(t, err)
		assert.Contains(t, result, "✓ 已创建")
		assert.Contains(t, result, "Test Meeting")
	})

	t.Run("missing title", func(t *testing.T) {
		input := `{"start_time": "2026-01-21T09:00:00Z"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("missing start_time", func(t *testing.T) {
		input := `{"title": "Test Meeting"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_time is required")
	})

	t.Run("invalid start time format", func(t *testing.T) {
		input := `{"title": "Test", "start_time": "invalid-time"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_time must be in ISO8601 format")
	})

	t.Run("invalid end time format", func(t *testing.T) {
		input := `{"title": "Test", "start_time": "2026-01-21T09:00:00Z", "end_time": "invalid"}`

		_, err := tool.Run(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid end_time format")
	})

	t.Run("all-day event", func(t *testing.T) {
		now := time.Now()
		startTime := now.Format(time.RFC3339)

		input := `{
			"title": "All Day Event",
			"start_time": "` + startTime + `",
			"all_day": true
		}`

		result, err := tool.Run(ctx, input)
		require.NoError(t, err)
		assert.Contains(t, result, "✓ 已创建")
	})
}

// TestScheduleQueryTool_Validate tests the validation logic.
func TestScheduleQueryTool_Validate(t *testing.T) {
	ctx := context.Background()
	mockService := &MockScheduleService{}
	userIDGetter := func(ctx context.Context) int32 { return 1 }
	tool := NewScheduleQueryTool(mockService, userIDGetter)

	t.Run("valid input", func(t *testing.T) {
		input := `{"start_time": "2026-01-21T09:00:00Z", "end_time": "2026-01-21T17:00:00Z"}`
		err := tool.Validate(ctx, input)
		require.NoError(t, err)
	})

	t.Run("missing start_time", func(t *testing.T) {
		input := `{"end_time": "2026-01-21T17:00:00Z"}`
		err := tool.Validate(ctx, input)
		require.Error(t, err)
	})

	t.Run("missing end_time", func(t *testing.T) {
		input := `{"start_time": "2026-01-21T09:00:00Z"}`
		err := tool.Validate(ctx, input)
		require.Error(t, err)
	})
}

// TestScheduleAddTool_Validate tests the validation logic.
func TestScheduleAddTool_Validate(t *testing.T) {
	ctx := context.Background()
	mockService := &MockScheduleService{}
	userIDGetter := func(ctx context.Context) int32 { return 1 }
	tool := NewScheduleAddTool(mockService, userIDGetter)

	t.Run("valid input", func(t *testing.T) {
		input := `{"title": "Test", "start_time": "2026-01-21T09:00:00Z"}`
		err := tool.Validate(ctx, input)
		require.NoError(t, err)
	})

	t.Run("missing title", func(t *testing.T) {
		input := `{"start_time": "2026-01-21T09:00:00Z"}`
		err := tool.Validate(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("missing start_time", func(t *testing.T) {
		input := `{"title": "Test"}`
		err := tool.Validate(ctx, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start_time is required")
	})
}
