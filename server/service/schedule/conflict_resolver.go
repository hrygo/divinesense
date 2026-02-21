package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/hrygo/divinesense/store"
)

// ConflictResolver provides intelligent conflict resolution for schedules.
// It can detect conflicts and suggest alternative time slots.
type ConflictResolver struct {
	service Service
}

// NewConflictResolver creates a new conflict resolver.
func NewConflictResolver(service Service) *ConflictResolver {
	return &ConflictResolver{
		service: service,
	}
}

// TimeSlot represents a time period that can be used for scheduling.
type TimeSlot struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Reason     string    `json:"reason"`      // Human-readable description
	Score      int       `json:"score"`       // Priority score (higher = better recommended)
	IsOriginal bool      `json:"is_original"` // True if this is the requested time
	IsAdjacent bool      `json:"is_adjacent"` // True if adjacent to requested time
}

// ConflictResolution represents the result of conflict resolution.
type ConflictResolution struct {
	OriginalStart time.Time           `json:"original_start"`
	OriginalEnd   time.Time           `json:"original_end"`
	AutoResolved  *TimeSlot           `json:"auto_resolved"`
	Conflicts     []*ScheduleInstance `json:"conflicts"`
	Alternatives  []TimeSlot          `json:"alternatives"`
}

// Resolve detects conflicts and provides alternative time slots.
func (r *ConflictResolver) Resolve(ctx context.Context, userID int32,
	start, end time.Time, duration time.Duration) (*ConflictResolution, error) {

	// Normalize end time if needed
	if end.IsZero() {
		end = start.Add(duration)
	}

	// Detect conflicts
	conflicts, err := r.service.CheckConflicts(ctx, userID, start.Unix(), toUnixPtr(end), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}

	resolution := &ConflictResolution{
		OriginalStart: start,
		OriginalEnd:   end,
		Conflicts:     convertToInstances(conflicts, nil),
	}

	if len(conflicts) == 0 {
		// No conflicts, mark requested time as available
		resolution.Alternatives = []TimeSlot{
			{
				Start:      start,
				End:        end,
				Reason:     start.Format("15:04"),
				Score:      1000,
				IsOriginal: true,
			},
		}
		return resolution, nil
	}

	slog.Info("conflicts detected",
		"user_id", userID,
		"requested_start", start,
		"conflict_count", len(conflicts),
	)

	// Find alternative time slots
	alternatives := r.findAlternatives(ctx, userID, start, duration)
	resolution.Alternatives = alternatives

	// Select best alternative for auto-resolution
	if len(alternatives) > 0 {
		best := r.selectBestAlternative(start, alternatives)
		resolution.AutoResolved = &best
	}

	return resolution, nil
}

// FindAllFreeSlots finds all free time slots for a specific date.
// It strictly filters out time slots that are in the past (no time travel).
func (r *ConflictResolver) FindAllFreeSlots(ctx context.Context, userID int32,
	date time.Time, duration time.Duration) ([]TimeSlot, error) {

	const (
		hourStart = 8  // 8 AM
		hourEnd   = 22 // 10 PM (last slot starts at 22:00)
	)

	loc := date.Location()
	now := time.Now().In(loc)

	// If the date is today, start from current time; otherwise start from hourStart
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), hourStart, 0, 0, 0, loc)
	if date.Year() == now.Year() && date.YearDay() == now.YearDay() {
		// Today: only consider slots from now onwards (strict no time travel)
		if now.After(startOfDay) {
			startOfDay = now
		}
	}

	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), hourEnd, 0, 0, 0, loc)

	slots, err := r.findSlotsInRange(ctx, userID, startOfDay, endOfDay, duration, loc)
	if err != nil {
		return nil, err
	}

	// Strict filter: remove any slots that start before now
	// This handles edge cases where the slot calculation returns past times
	if date.Year() == now.Year() && date.YearDay() == now.YearDay() {
		var filtered []TimeSlot
		for _, slot := range slots {
			if slot.Start.After(now) || slot.Start.Equal(now) {
				filtered = append(filtered, slot)
			}
		}
		return filtered, nil
	}

	return slots, nil
}

// findAlternatives finds all available alternative time slots.
func (r *ConflictResolver) findAlternatives(ctx context.Context, userID int32,
	requested time.Time, duration time.Duration) []TimeSlot {

	const (
		hourStart = 8
		hourEnd   = 22
		dayRange  = 3 // Search 3 days before and after
	)

	var allAlternatives []TimeSlot

	// Strategy 1: Same day alternatives (highest priority)
	sameDaySlots := r.findSlotsInDay(ctx, userID, requested, duration, hourStart, hourEnd)
	for i := range sameDaySlots {
		sameDaySlots[i].Reason = sameDaySlots[i].Start.Format("15:04")
	}
	allAlternatives = append(allAlternatives, sameDaySlots...)

	// Strategy 2: Adjacent days if same day is full or limited
	if len(sameDaySlots) < 3 {
		for dayOffset := 1; dayOffset <= dayRange; dayOffset++ {
			// Check day before
			beforeDay := requested.AddDate(0, 0, -dayOffset)
			beforeSlots := r.findSlotsInDay(ctx, userID, beforeDay, duration, hourStart, hourEnd)
			for i := range beforeSlots {
				// Reason format: "days_before:N" for i18n frontend handling
				beforeSlots[i].Reason = fmt.Sprintf("days_before:%d", dayOffset)
				beforeSlots[i].IsAdjacent = true
			}
			allAlternatives = append(allAlternatives, beforeSlots...)

			// Check day after
			afterDay := requested.AddDate(0, 0, dayOffset)
			afterSlots := r.findSlotsInDay(ctx, userID, afterDay, duration, hourStart, hourEnd)
			for i := range afterSlots {
				// Reason format: "days_after:N" for i18n frontend handling
				afterSlots[i].Reason = fmt.Sprintf("days_after:%d", dayOffset)
				afterSlots[i].IsAdjacent = true
			}
			allAlternatives = append(allAlternatives, afterSlots...)

			// Stop if we have enough alternatives
			if len(allAlternatives) >= 10 {
				break
			}
		}
	}

	// Score and sort alternatives
	return r.scoreAlternatives(requested, allAlternatives)
}

// findSlotsInDay finds all available time slots on a specific day.
func (r *ConflictResolver) findSlotsInDay(ctx context.Context, userID int32,
	date time.Time, duration time.Duration, hourStart, hourEnd int) []TimeSlot {

	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), hourStart, 0, 0, 0, date.Location())
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), hourEnd, 0, 0, 0, date.Location())

	slots, err := r.findSlotsInRange(ctx, userID, startOfDay, endOfDay, duration, date.Location())
	if err != nil {
		slog.Warn("failed to find slots in day",
			"date", date,
			"error", err,
		)
		return nil
	}

	return slots
}

// findSlotsInRange finds available time slots within a range.
func (r *ConflictResolver) findSlotsInRange(ctx context.Context, userID int32,
	startOfDay, endOfDay time.Time, duration time.Duration, loc *time.Location) ([]TimeSlot, error) {

	// Get existing schedules
	schedules, err := r.service.FindSchedules(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// Build busy time ranges
	var busyRanges []timeRange
	for _, sched := range schedules {
		schedStart := time.Unix(sched.StartTs, 0).In(loc)
		var schedEnd time.Time
		if sched.EndTs != nil && *sched.EndTs != 0 {
			schedEnd = time.Unix(*sched.EndTs, 0).In(loc)
		} else {
			schedEnd = schedStart.Add(time.Hour)
		}
		busyRanges = append(busyRanges, timeRange{start: schedStart, end: schedEnd})
	}

	// Sort busy ranges by start time
	sort.Slice(busyRanges, func(i, j int) bool {
		return busyRanges[i].start.Before(busyRanges[j].start)
	})

	// Find free slots between busy ranges
	var freeSlots []TimeSlot
	current := startOfDay

	for _, busy := range busyRanges {
		// Skip if busy range is before current
		if busy.end.Before(current) || busy.end.Equal(current) {
			continue
		}

		// Check if there's a gap before this busy range
		if busy.start.After(current) {
			gapDuration := busy.start.Sub(current)
			if gapDuration >= duration {
				freeSlots = append(freeSlots, TimeSlot{
					Start:  current,
					End:    current.Add(duration),
					Reason: current.Format("15:04"),
				})
			}
		}

		// Move current position to after the busy range
		if busy.end.After(current) {
			current = busy.end
		}
	}

	// Check final gap
	if endOfDay.Sub(current) >= duration {
		freeSlots = append(freeSlots, TimeSlot{
			Start:  current,
			End:    current.Add(duration),
			Reason: current.Format("15:04"),
		})
	}

	return freeSlots, nil
}

// selectBestAlternative selects the best alternative time slot.
func (r *ConflictResolver) selectBestAlternative(_ time.Time, alternatives []TimeSlot) TimeSlot {
	if len(alternatives) == 0 {
		return TimeSlot{}
	}

	// Alternatives are already sorted by score, return the best
	return alternatives[0]
}

// scoreAlternatives assigns scores to each alternative and sorts them.
func (r *ConflictResolver) scoreAlternatives(requested time.Time, alternatives []TimeSlot) []TimeSlot {
	for i := range alternatives {
		alt := &alternatives[i]
		alt.Score = r.calculateScore(requested, *alt)
	}

	// Sort by score (descending)
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Score > alternatives[j].Score
	})

	return alternatives
}

// calculateScore calculates a priority score for a time slot.
// Higher scores indicate better alternatives.
func (r *ConflictResolver) calculateScore(requested time.Time, alt TimeSlot) int {
	score := 0

	// Factor 1: Same day is best (100 points)
	if alt.Start.YearDay() == requested.YearDay() {
		score += 100
	}

	// Factor 2: Proximity to requested time (up to 50 points)
	hourDiff := alt.Start.Hour() - requested.Hour()
	if hourDiff < 0 {
		hourDiff = -hourDiff
	}
	if hourDiff == 0 {
		score += 50
	} else {
		score += (24 - hourDiff) * 2
	}

	// Factor 3: Same time of day (morning/afternoon) (20 points)
	if (alt.Start.Hour() < 12) == (requested.Hour() < 12) {
		score += 20
	}

	// Factor 4: Same weekday (10 points)
	if alt.Start.Weekday() == requested.Weekday() {
		score += 10
	}

	// Factor 5: Business hours preference
	hour := alt.Start.Hour()
	if hour >= 9 && hour <= 11 {
		score += 15 // Morning prime time
	} else if hour >= 14 && hour <= 16 {
		score += 15 // Afternoon prime time
	} else if hour >= 11 && hour <= 13 {
		score += 10 // Lunch time (less preferred)
	}

	// Factor 6: Penalty for non-adjacent days
	if alt.IsAdjacent {
		score -= 5
	}

	return score
}

// Helper types and functions

type timeRange struct {
	start, end time.Time
}

func toUnixPtr(t time.Time) *int64 {
	ts := t.Unix()
	return &ts
}

func convertToInstances(schedules []*store.Schedule, _ *time.Location) []*ScheduleInstance {
	instances := make([]*ScheduleInstance, 0, len(schedules))
	for _, sched := range schedules {
		instances = append(instances, &ScheduleInstance{
			ID:          sched.ID,
			UID:         sched.UID,
			Title:       sched.Title,
			Description: sched.Description,
			Location:    sched.Location,
			StartTs:     sched.StartTs,
			EndTs:       sched.EndTs,
			AllDay:      sched.AllDay,
			Timezone:    sched.Timezone,
			IsRecurring: sched.RecurrenceRule != nil && *sched.RecurrenceRule != "",
			ParentUID:   "",
		})
	}
	return instances
}
