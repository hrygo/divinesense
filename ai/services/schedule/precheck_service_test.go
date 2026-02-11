package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrygo/divinesense/store"
)

// mockScheduleStore implements ScheduleStore for testing.
type mockScheduleStore struct {
	schedules []*store.Schedule
}

func (m *mockScheduleStore) ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error) {
	var result []*store.Schedule
	for _, s := range m.schedules {
		// Simple filtering for testing
		if find.CreatorID != nil && s.CreatorID != *find.CreatorID {
			continue
		}
		if find.StartTs != nil && find.EndTs != nil {
			// Check if schedule overlaps with the query range
			scheduleEnd := s.StartTs
			if s.EndTs != nil {
				scheduleEnd = *s.EndTs
			}
			if s.StartTs >= *find.EndTs || scheduleEnd <= *find.StartTs {
				continue
			}
		}
		result = append(result, s)
	}
	return result, nil
}

func TestPrecheckService_ValidTimeFormat(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	req := &PrecheckRequest{
		Title:     "Valid Meeting",
		StartTime: tomorrow,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.True(t, resp.Valid, "should be valid")
	assert.Empty(t, resp.Errors, "should have no errors")
}

func TestPrecheckService_MissingStartTime(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	req := &PrecheckRequest{
		Title:    "No Time Meeting",
		Duration: 60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid, "should be invalid")
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, ErrCodeMissingStartTime, resp.Errors[0].Code)
}

func TestPrecheckService_PastTime(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	yesterday := time.Now().AddDate(0, 0, -1)

	req := &PrecheckRequest{
		Title:     "Past Meeting",
		StartTime: yesterday,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid, "should be invalid")
	var codes []string
	for _, e := range resp.Errors {
		codes = append(codes, e.Code)
	}
	assert.Contains(t, codes, ErrCodePastTime)
}

func TestPrecheckService_TimeTooFar(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	twoYearsLater := time.Now().AddDate(2, 0, 0)

	req := &PrecheckRequest{
		Title:     "Future Meeting",
		StartTime: twoYearsLater,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid, "should be invalid")
	var codes []string
	for _, e := range resp.Errors {
		codes = append(codes, e.Code)
	}
	assert.Contains(t, codes, ErrCodeTimeTooFar)
}

func TestPrecheckService_EndBeforeStart(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	req := &PrecheckRequest{
		Title:     "Invalid Meeting",
		StartTime: tomorrow,
		EndTime:   tomorrow.Add(-1 * time.Hour), // End before start
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid, "should be invalid")
	var codes []string
	for _, e := range resp.Errors {
		codes = append(codes, e.Code)
	}
	assert.Contains(t, codes, ErrCodeEndBeforeStart)
}

func TestPrecheckService_TimeConflict(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	existingEnd := tomorrow.Add(time.Hour).Unix()

	mockStore := &mockScheduleStore{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: 1,
				Title:     "Existing Meeting",
				StartTs:   tomorrow.Unix(),
				EndTs:     &existingEnd,
			},
		},
	}
	service := NewPrecheckService(mockStore)

	// Try to create a meeting that overlaps
	req := &PrecheckRequest{
		Title:     "New Meeting",
		StartTime: tomorrow.Add(30 * time.Minute), // Overlaps with existing
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid, "should be invalid due to conflict")
	var codes []string
	for _, e := range resp.Errors {
		codes = append(codes, e.Code)
	}
	assert.Contains(t, codes, ErrCodeTimeConflict)
}

func TestPrecheckService_NoConflictDifferentUser(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	existingEnd := tomorrow.Add(time.Hour).Unix()

	mockStore := &mockScheduleStore{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: 2, // Different user
				Title:     "Other User's Meeting",
				StartTs:   tomorrow.Unix(),
				EndTs:     &existingEnd,
			},
		},
	}
	service := NewPrecheckService(mockStore)

	req := &PrecheckRequest{
		Title:     "My Meeting",
		StartTime: tomorrow.Add(30 * time.Minute),
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.True(t, resp.Valid, "should be valid - different user")
}

func TestPrecheckService_BufferConflictWarning(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	existingEnd := tomorrow.Add(time.Hour).Unix()

	mockStore := &mockScheduleStore{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: 1,
				Title:     "Existing Meeting",
				StartTs:   tomorrow.Unix(),
				EndTs:     &existingEnd,
			},
		},
	}
	service := NewPrecheckService(mockStore)

	// Create a meeting right after the existing one (within buffer)
	req := &PrecheckRequest{
		Title:     "Back-to-back Meeting",
		StartTime: tomorrow.Add(time.Hour + 5*time.Minute), // Only 5 min gap
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.True(t, resp.Valid, "should be valid but with warning")
	var warnCodes []string
	for _, w := range resp.Warnings {
		warnCodes = append(warnCodes, w.Code)
	}
	assert.Contains(t, warnCodes, WarnCodeBufferConflict)
}

func TestPrecheckService_LongDurationWarning(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	req := &PrecheckRequest{
		Title:     "All Day Meeting",
		StartTime: tomorrow,
		Duration:  600, // 10 hours, exceeds 8 hour max
	}

	resp := service.Precheck(context.Background(), 1, req)

	var warnCodes []string
	for _, w := range resp.Warnings {
		warnCodes = append(warnCodes, w.Code)
	}
	assert.Contains(t, warnCodes, WarnCodeLongDuration)
}

func TestPrecheckService_OutsideWorkHoursWarning(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	// Schedule at 6 AM (before work hours start at 8 AM)
	// Use local timezone to avoid UTC conversion issues
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 6, 0, 0, 0, now.Location())

	req := &PrecheckRequest{
		Title:     "Early Meeting",
		StartTime: tomorrow,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	var warnCodes []string
	for _, w := range resp.Warnings {
		warnCodes = append(warnCodes, w.Code)
	}
	assert.Contains(t, warnCodes, WarnCodeOutsideWorkHours)
}

func TestPrecheckService_WeekendWarning(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	// Use a fixed date (Wednesday 2026-01-28 10:00) to avoid timezone boundary issues
	now := time.Date(2026, 1, 28, 10, 0, 0, 0, time.Local)

	// Calculate days until Saturday (Wed->Thu->Fri->Sat = 3 days)
	daysUntilSaturday := (6 - int(now.Weekday()) + 7) % 7
	if daysUntilSaturday == 0 {
		daysUntilSaturday = 7
	}
	nextSaturday := now.AddDate(0, 0, daysUntilSaturday).Add(10 * time.Hour)

	req := &PrecheckRequest{
		Title:     "Weekend Meeting",
		StartTime: nextSaturday,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	var warnCodes []string
	for _, w := range resp.Warnings {
		warnCodes = append(warnCodes, w.Code)
	}
	assert.Contains(t, warnCodes, WarnCodeWeekendSchedule)
}

func TestPrecheckService_LongTitleWarning(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	// Create a very long title
	longTitle := ""
	for i := 0; i < 150; i++ {
		longTitle += "a"
	}

	req := &PrecheckRequest{
		Title:     longTitle,
		StartTime: tomorrow,
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	var warnCodes []string
	for _, w := range resp.Warnings {
		warnCodes = append(warnCodes, w.Code)
	}
	assert.Contains(t, warnCodes, WarnCodeLongTitle)
}

func TestPrecheckService_AlternativeSuggestions(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	existingEnd := tomorrow.Add(time.Hour).Unix()

	mockStore := &mockScheduleStore{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: 1,
				Title:     "Existing Meeting",
				StartTs:   tomorrow.Unix(),
				EndTs:     &existingEnd,
			},
		},
	}
	service := NewPrecheckService(mockStore)

	req := &PrecheckRequest{
		Title:     "New Meeting",
		StartTime: tomorrow.Add(30 * time.Minute), // Conflicts
		Duration:  60,
	}

	resp := service.Precheck(context.Background(), 1, req)

	assert.False(t, resp.Valid)
	// Should have alternative suggestions
	assert.NotEmpty(t, resp.Suggestions, "should have alternative suggestions")

	for _, suggestion := range resp.Suggestions {
		assert.Equal(t, "alternative_time", suggestion.Type)
		alt, ok := suggestion.Value.(AlternativeSlot)
		assert.True(t, ok, "suggestion value should be AlternativeSlot")
		assert.NotEmpty(t, alt.Label)
		assert.False(t, alt.StartTime.IsZero())
	}
}

func TestPrecheckService_NormalizeRequest(t *testing.T) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	tests := []struct {
		req           *PrecheckRequest
		name          string
		wantDuration  int
		wantEndOffset time.Duration
	}{
		{
			name: "duration only",
			req: &PrecheckRequest{
				Title:     "Test",
				StartTime: tomorrow,
				Duration:  30,
			},
			wantDuration:  30,
			wantEndOffset: 30 * time.Minute,
		},
		{
			name: "end time only",
			req: &PrecheckRequest{
				Title:     "Test",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(45 * time.Minute),
			},
			wantDuration:  45,
			wantEndOffset: 45 * time.Minute,
		},
		{
			name: "no duration or end time",
			req: &PrecheckRequest{
				Title:     "Test",
				StartTime: tomorrow,
			},
			wantDuration:  60, // Default 1 hour
			wantEndOffset: 60 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := service.normalizeRequest(tt.req)
			assert.Equal(t, tt.wantDuration, normalized.Duration)
			assert.Equal(t, tt.req.StartTime.Add(tt.wantEndOffset), normalized.EndTime)
		})
	}
}

func BenchmarkPrecheckService_Precheck(b *testing.B) {
	mockStore := &mockScheduleStore{}
	service := NewPrecheckService(mockStore)

	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)
	req := &PrecheckRequest{
		Title:     "Benchmark Meeting",
		StartTime: tomorrow,
		Duration:  60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Precheck(context.Background(), 1, req)
	}
}

func BenchmarkPrecheckService_WithConflictCheck(b *testing.B) {
	tomorrow := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(10 * time.Hour)

	// Create 10 existing schedules
	var schedules []*store.Schedule
	for i := 0; i < 10; i++ {
		startTs := tomorrow.Add(time.Duration(i*2) * time.Hour).Unix()
		endTs := tomorrow.Add(time.Duration(i*2+1) * time.Hour).Unix()
		schedules = append(schedules, &store.Schedule{
			ID:        int32(i + 1),
			CreatorID: 1,
			Title:     "Existing Meeting",
			StartTs:   startTs,
			EndTs:     &endTs,
		})
	}

	mockStore := &mockScheduleStore{schedules: schedules}
	service := NewPrecheckService(mockStore)

	req := &PrecheckRequest{
		Title:     "Benchmark Meeting",
		StartTime: tomorrow.Add(30 * time.Minute),
		Duration:  60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Precheck(context.Background(), 1, req)
	}
}
