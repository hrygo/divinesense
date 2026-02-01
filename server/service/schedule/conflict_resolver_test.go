package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConflictResolver_Resolve_NoConflict tests resolution when there's no conflict.
func TestConflictResolver_Resolve_NoConflict(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now().Truncate(time.Hour).Add(time.Hour) // Round to next hour

	// Empty mock store - no existing schedules
	mockStore := &MockStoreForSchedule{schedules: []*store.Schedule{}}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request a 1-hour slot
	startTime := now
	duration := time.Hour

	resolution, err := resolver.Resolve(ctx, userID, startTime, time.Time{}, duration)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Equal(t, startTime, resolution.OriginalStart)
	assert.Len(t, resolution.Conflicts, 0, "Should have no conflicts")
	assert.Len(t, resolution.Alternatives, 1, "Should have the original time as an option")
	assert.True(t, resolution.Alternatives[0].IsOriginal, "First alternative should be the original time")
	assert.Equal(t, 1000, resolution.Alternatives[0].Score, "Original slot should have highest score")
}

// TestConflictResolver_Resolve_WithConflict tests resolution when there's a conflict.
func TestConflictResolver_Resolve_WithConflict(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now().Truncate(time.Hour)

	// Create a schedule that occupies 10:00-11:00
	busyStart := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busyEnd := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "busy-uid",
				CreatorID: userID,
				Title:     "Busy Meeting",
				StartTs:   busyStart.Unix(),
				EndTs:     func() *int64 { ts := busyEnd.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request the same time slot (should conflict)
	requestedStart := busyStart
	duration := time.Hour

	resolution, err := resolver.Resolve(ctx, userID, requestedStart, time.Time{}, duration)
	require.NoError(t, err)
	assert.NotNil(t, resolution)
	assert.Len(t, resolution.Conflicts, 1, "Should detect the conflict")
	assert.Equal(t, "Busy Meeting", resolution.Conflicts[0].Title)
	assert.NotNil(t, resolution.AutoResolved, "Should provide an auto-resolved alternative")
}

// TestConflictResolver_FindAllFreeSlots tests finding all free slots in a day.
func TestConflictResolver_FindAllFreeSlots(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()
	testDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Create schedules that occupy 9:00-10:00 and 14:00-15:00
	busy1Start := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)
	busy1End := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busy2Start := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, time.Local)
	busy2End := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "busy-1",
				CreatorID: userID,
				Title:     "Morning Meeting",
				StartTs:   busy1Start.Unix(),
				EndTs:     func() *int64 { ts := busy1End.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
			{
				ID:        2,
				UID:       "busy-2",
				CreatorID: userID,
				Title:     "Afternoon Meeting",
				StartTs:   busy2Start.Unix(),
				EndTs:     func() *int64 { ts := busy2End.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Find 1-hour free slots
	freeSlots, err := resolver.FindAllFreeSlots(ctx, userID, testDate, time.Hour)
	require.NoError(t, err)

	// Should have free slots at 8:00, 11:00, 12:00, 13:00, 15:00, 16:00, etc.
	// (excluding the busy 9:00-10:00 and 14:00-15:00)
	assert.Greater(t, len(freeSlots), 0, "Should find free slots")

	// Check that the busy times are not in the free slots
	for _, slot := range freeSlots {
		assert.NotEqual(t, 9, slot.Start.Hour(), "Should not include 9:00 (busy)")
		assert.NotEqual(t, 14, slot.Start.Hour(), "Should not include 14:00 (busy)")
	}
}

// TestConflictResolver_FindAllFreeSlots_FullyBooked tests when a day is fully booked.
func TestConflictResolver_FindAllFreeSlots_FullyBooked(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()
	testDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Create schedules that occupy the entire day (8:00-22:00)
	var schedules []*store.Schedule
	for hour := 8; hour < 22; hour++ {
		start := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
		end := time.Date(now.Year(), now.Month(), now.Day(), hour+1, 0, 0, 0, time.Local)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(hour),
			UID:       "busy-" + string(rune('0'+hour)),
			CreatorID: userID,
			Title:     "Busy Slot",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Find 1-hour free slots - should be empty
	freeSlots, err := resolver.FindAllFreeSlots(ctx, userID, testDate, time.Hour)
	require.NoError(t, err)
	assert.Len(t, freeSlots, 0, "Should have no free slots when fully booked")
}

// TestConflictResolver_SameDayPriority tests that same-day alternatives get higher priority.
func TestConflictResolver_SameDayPriority(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now().Truncate(time.Hour)

	// Create a schedule at 10:00
	busyStart := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busyEnd := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				UID:       "busy-uid",
				CreatorID: userID,
				Title:     "Busy Meeting",
				StartTs:   busyStart.Unix(),
				EndTs:     func() *int64 { ts := busyEnd.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request the busy 10:00 slot
	requestedStart := busyStart
	duration := time.Hour

	resolution, err := resolver.Resolve(ctx, userID, requestedStart, time.Time{}, duration)
	require.NoError(t, err)
	require.NotNil(t, resolution.AutoResolved)

	// The auto-resolved slot should be on the same day (highest score)
	autoResolved := resolution.AutoResolved
	assert.Equal(t, now.YearDay(), autoResolved.Start.YearDay(), "Auto-resolved should be on same day")
}

// TestConflictResolver_CalculateScore tests the scoring algorithm.
func TestConflictResolver_CalculateScore(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local)
	resolver := &ConflictResolver{}

	tests := []struct {
		name           string
		expectedReason string
		alternative    TimeSlot
		minScore       int
	}{
		{
			name: "same day, same time",
			alternative: TimeSlot{
				Start: now,
				End:   now.Add(time.Hour),
			},
			minScore:       160, // 100 (same day) + 50 (same hour) + 10 (same weekday)
			expectedReason: "should have highest score for exact match",
		},
		{
			name: "same day, different time",
			alternative: TimeSlot{
				Start: time.Date(2026, 1, 15, 14, 0, 0, 0, time.Local),
				End:   time.Date(2026, 1, 15, 15, 0, 0, 0, time.Local),
			},
			minScore:       140, // 100 (same day) + 20 (same afternoon) + 10 (same weekday) + 10 (afternoon prime time)
			expectedReason: "should have high score for same day",
		},
		{
			name: "adjacent day",
			alternative: TimeSlot{
				Start:      time.Date(2026, 1, 16, 10, 0, 0, 0, time.Local),
				End:        time.Date(2026, 1, 16, 11, 0, 0, 0, time.Local),
				IsAdjacent: true,
			},
			minScore:       40, // Base score minus adjacent penalty
			expectedReason: "should have lower score for adjacent day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := resolver.calculateScore(now, tt.alternative)
			assert.GreaterOrEqual(t, score, tt.minScore, tt.expectedReason)
		})
	}
}

// TestConflictResolver_ScoreAlternativesAndSort tests that alternatives are sorted by score.
func TestConflictResolver_ScoreAlternativesAndSort(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local)
	resolver := &ConflictResolver{}

	alternatives := []TimeSlot{
		{
			Start: time.Date(2026, 1, 16, 10, 0, 0, 0, time.Local), // Next day - lower score
			End:   time.Date(2026, 1, 16, 11, 0, 0, 0, time.Local),
		},
		{
			Start: time.Date(2026, 1, 15, 9, 0, 0, 0, time.Local), // Same day - higher score
			End:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local),
		},
		{
			Start: time.Date(2026, 1, 15, 14, 0, 0, 0, time.Local), // Same day afternoon - high score
			End:   time.Date(2026, 1, 15, 15, 0, 0, 0, time.Local),
		},
	}

	sorted := resolver.scoreAlternatives(now, alternatives)

	// First element should have highest score (same day, morning prime time or afternoon)
	firstScore := sorted[0].Score
	lastScore := sorted[len(sorted)-1].Score
	assert.Greater(t, firstScore, lastScore, "Alternatives should be sorted by score (descending)")
}

// TestConflictResolver_BusinessHoursPreference tests preference for business hours.
func TestConflictResolver_BusinessHoursPreference(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local)
	resolver := &ConflictResolver{}

	// Morning prime time (9-11)
	morningSlot := TimeSlot{
		Start: time.Date(2026, 1, 15, 9, 0, 0, 0, time.Local),
		End:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.Local),
	}

	// Lunch time (11-13) - less preferred
	lunchSlot := TimeSlot{
		Start: time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local),
		End:   time.Date(2026, 1, 15, 13, 0, 0, 0, time.Local),
	}

	morningScore := resolver.calculateScore(now, morningSlot)
	lunchScore := resolver.calculateScore(now, lunchSlot)

	assert.Greater(t, morningScore, lunchScore, "Morning prime time should score higher than lunch time")
}

// TestConflictResolver_AdjacentDayPreference tests that closer days are preferred over farther days.
func TestConflictResolver_AdjacentDayPreference(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	// Use a fixed date to avoid issues with time.Now() at month boundaries in UTC
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)

	// Make the current day fully booked
	var schedules []*store.Schedule
	for hour := 8; hour < 22; hour++ {
		start := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
		end := time.Date(now.Year(), now.Month(), now.Day(), hour+1, 0, 0, 0, time.UTC)
		schedules = append(schedules, &store.Schedule{
			ID:        int32(hour),
			CreatorID: userID,
			Title:     "Busy",
			StartTs:   start.Unix(),
			EndTs:     func() *int64 { ts := end.Unix(); return &ts }(),
			RowStatus: store.Normal,
		})
	}

	mockStore := &MockStoreForSchedule{schedules: schedules}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request a slot on the fully booked day
	requestedStart := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.UTC)
	duration := time.Hour

	resolution, err := resolver.Resolve(ctx, userID, requestedStart, time.Time{}, duration)
	require.NoError(t, err)
	require.NotNil(t, resolution.AutoResolved)

	// Should prefer adjacent days (tomorrow or yesterday)
	autoResolved := resolution.AutoResolved
	// Use actual day difference instead of Day() to handle month boundaries correctly
	dayDiff := int(autoResolved.Start.Sub(now).Hours() / 24)
	assert.LessOrEqual(t, abs(dayDiff), 3, "Should find a slot within 3 days")
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestConflictResolver_DurationVariations tests different duration requirements.
func TestConflictResolver_DurationVariations(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create a schedule at 10:00-11:00
	busyStart := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busyEnd := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:        1,
				CreatorID: userID,
				Title:     "Busy Meeting",
				StartTs:   busyStart.Unix(),
				EndTs:     func() *int64 { ts := busyEnd.Unix(); return &ts }(),
				RowStatus: store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	tests := []struct {
		name           string
		duration       time.Duration
		shouldFindSlot bool
	}{
		{
			name:           "30 minutes",
			duration:       30 * time.Minute,
			shouldFindSlot: true,
		},
		{
			name:           "1 hour",
			duration:       time.Hour,
			shouldFindSlot: true,
		},
		{
			name:           "2 hours",
			duration:       2 * time.Hour,
			shouldFindSlot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestedStart := busyStart
			resolution, err := resolver.Resolve(ctx, userID, requestedStart, time.Time{}, tt.duration)
			require.NoError(t, err)

			if tt.shouldFindSlot {
				assert.NotNil(t, resolution.AutoResolved, "Should find a slot for %v duration", tt.duration)
			}
		})
	}
}

// TestConflictResolver_WithRecurringSchedules tests conflict resolution with recurring schedules.
func TestConflictResolver_WithRecurringSchedules(t *testing.T) {
	ctx := context.Background()
	userID := int32(1)
	now := time.Now()

	// Create a recurring daily schedule at 10:00-11:00
	rule := "FREQ=DAILY;COUNT=5"
	busyStart := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.Local)
	busyEnd := time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, time.Local)

	mockStore := &MockStoreForSchedule{
		schedules: []*store.Schedule{
			{
				ID:             1,
				CreatorID:      userID,
				Title:          "Daily Standup",
				StartTs:        busyStart.Unix(),
				EndTs:          func() *int64 { ts := busyEnd.Unix(); return &ts }(),
				RecurrenceRule: &rule,
				RowStatus:      store.Normal,
			},
		},
	}
	svc := &service{store: mockStore}
	resolver := NewConflictResolver(svc)

	// Request the same time slot - should conflict
	requestedStart := busyStart
	duration := time.Hour

	resolution, err := resolver.Resolve(ctx, userID, requestedStart, time.Time{}, duration)
	require.NoError(t, err)
	assert.Len(t, resolution.Conflicts, 1, "Should detect conflict with recurring schedule")
	assert.True(t, resolution.Conflicts[0].IsRecurring, "Conflict should be marked as recurring")
}
