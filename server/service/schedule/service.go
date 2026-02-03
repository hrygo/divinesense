// Package schedule provides schedule management functionality including creation,
// querying, updating, and deleting schedules with recurring event support.
//
// Key features:
//   - Recurring schedule expansion using RRule
//   - Conflict detection and prevention (atomic via DB constraint)
//   - Timezone-aware time handling
//
// The service layer abstracts business logic from the store layer and provides
// a clean interface for upper layers.
package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	aischedule "github.com/hrygo/divinesense/ai/schedule"
	"github.com/hrygo/divinesense/internal/util"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
	postgresstore "github.com/hrygo/divinesense/store/db/postgres"
)

const (
	// DefaultConflictCheckWindow is the default time window for conflict checking.
	DefaultConflictCheckWindow = 1 * time.Hour

	// This prevents excessive computation for infinite recurrence rules.
	MaxInstancesToCheck = 100
)

// Schedule-specific errors that can be checked with errors.Is.
var (
	// ErrScheduleConflict is returned when a schedule conflicts with existing schedules.
	ErrScheduleConflict = fmt.Errorf("schedule conflicts detected")
)

// ConflictError is a structured error for schedule conflicts with i18n support.
type ConflictError struct {
	Alternatives  []TimeSlotAlternative `json:"alternatives"`
	ConflictCount int                   `json:"conflict_count"`
	OriginalStart int64                 `json:"original_start"`
}

// TimeSlotAlternative represents an available time slot alternative.
type TimeSlotAlternative struct {
	Reason  string `json:"reason,omitempty"`
	StartTs int64  `json:"start_ts"`
	EndTs   int64  `json:"end_ts"`
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("schedule conflict with %d alternatives", len(e.Alternatives))
}

// NewConflictError creates a new conflict error with structured data for i18n.
func NewConflictError(alternatives []TimeSlotAlternative, conflictCount int, originalStart int64) *ConflictError {
	return &ConflictError{
		Alternatives:  alternatives,
		ConflictCount: conflictCount,
		OriginalStart: originalStart,
	}
}

type service struct {
	store Store
}

// Store is the interface for store operations needed by the schedule service.
type Store interface {
	CreateSchedule(ctx context.Context, create *store.Schedule) (*store.Schedule, error)
	ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error)
	GetSchedule(ctx context.Context, find *store.FindSchedule) (*store.Schedule, error)
	UpdateSchedule(ctx context.Context, update *store.UpdateSchedule) error
	DeleteSchedule(ctx context.Context, delete *store.DeleteSchedule) error
}

// NewService creates a new schedule service.
func NewService(store *store.Store) Service {
	return &service{store: store}
}

// FindSchedules returns schedules between start and end time, with recurring schedules expanded.
func (s *service) FindSchedules(ctx context.Context, userID int32, start, end time.Time) ([]*ScheduleInstance, error) {
	// Convert to timestamps
	startTs := start.Unix()
	endTs := end.Unix()

	// Find schedule templates from database
	// Only fetch active (NORMAL) schedules to avoid conflicts with archived ones
	normalStatus := store.Normal
	find := &store.FindSchedule{
		CreatorID: &userID,
		RowStatus: &normalStatus,
		// For recurring schedules, we need to query without time constraints
		// to get the schedule templates, then expand instances
	}

	list, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Expand recurring schedules and convert to instances
	var instances []*ScheduleInstance
	maxTotalInstances := MaxInstances // Use package constant
	truncated := false

	for _, sched := range list {
		// Check total instance limit before processing each schedule
		if len(instances) >= maxTotalInstances {
			truncated = true
			break
		}

		// If this is a recurring schedule, expand it
		if sched.RecurrenceRule != nil && *sched.RecurrenceRule != "" {
			// Parse recurrence rule
			rule, err := aischedule.ParseRecurrenceRuleFromJSON(*sched.RecurrenceRule)
			if err != nil {
				// If parsing fails, just return the base schedule
				instances = append(instances, s.convertToInstance(sched, false, ""))
				continue
			}

			// Generate instances starting from the schedule's start time
			// This ensures we get the correct sequence from the first occurrence
			instanceTimestamps := rule.GenerateInstances(sched.StartTs, endTs)

			// For each instance, create a schedule with adjusted time
			for _, instanceTs := range instanceTimestamps {
				// Check if we've hit the total instance limit
				if len(instances) >= maxTotalInstances {
					truncated = true
					break
				}

				// Only add instances within the query window
				if instanceTs < startTs || instanceTs > endTs {
					continue
				}

				instance := s.convertToInstance(sched, true, sched.UID)
				instance.StartTs = instanceTs

				// Calculate end time for this instance
				if sched.EndTs != nil && sched.StartTs > 0 {
					duration := *sched.EndTs - sched.StartTs
					endTsValue := instanceTs + duration
					instance.EndTs = &endTsValue
				}

				instances = append(instances, instance)

				// Break if we've hit the limit
				if len(instances) >= maxTotalInstances {
					truncated = true
					break
				}
			}
		} else {
			// Non-recurring schedule, add as-is if within time range
			// Check if schedule overlaps with query window
			scheduleEnd := sched.EndTs
			if scheduleEnd == nil {
				scheduleEnd = &sched.StartTs
			}

			// Check overlap: query window [startTs, endTs] vs schedule [sched.StartTs, *scheduleEnd]
			overlaps := startTs <= *scheduleEnd && endTs >= sched.StartTs

			if overlaps {
				instances = append(instances, s.convertToInstance(sched, false, ""))
			}
		}
	}

	// Log warning if truncated
	if truncated {
		slog.Warn("schedule instance expansion truncated",
			"count", len(instances),
			"limit", maxTotalInstances,
			"user_id", userID)
	}

	return instances, nil
}

// CreateSchedule creates a new schedule with validation and conflict checking.
func (s *service) CreateSchedule(ctx context.Context, userID int32, create *CreateScheduleRequest) (*store.Schedule, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		slog.Debug("schedule create operation",
			"user_id", userID,
			"title", create.Title,
			"duration_ms", duration.Milliseconds(),
		)
	}()

	// Validate required fields
	if create.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if create.StartTs <= 0 {
		return nil, fmt.Errorf("start_ts must be a positive timestamp")
	}
	if create.EndTs != nil && *create.EndTs < create.StartTs {
		return nil, fmt.Errorf("end_ts must be greater than or equal to start_ts")
	}

	// Set default timezone if not provided
	timezone := create.Timezone
	if timezone == "" {
		timezone = DefaultTimezone
	}

	// Marshal reminders
	var remindersStr string
	if len(create.Reminders) > 0 {
		reminders := make([]*v1pb.Reminder, len(create.Reminders))
		for i, r := range create.Reminders {
			reminders[i] = &v1pb.Reminder{
				Type:  r.Type,
				Value: r.Value,
				Unit:  r.Unit,
			}
		}
		var err error
		remindersStr, err = aischedule.MarshalReminders(reminders)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal reminders: %w", err)
		}
	} else {
		remindersStr = "[]"
	}

	// Create schedule object
	sched := &store.Schedule{
		UID:             util.GenUUID(),
		CreatorID:       userID,
		Title:           create.Title,
		Description:     create.Description,
		Location:        create.Location,
		StartTs:         create.StartTs,
		EndTs:           create.EndTs,
		AllDay:          create.AllDay,
		Timezone:        timezone,
		RecurrenceRule:  create.RecurrenceRule,
		RecurrenceEndTs: create.RecurrenceEndTs,
		Reminders:       &remindersStr,
		RowStatus:       store.Normal,
	}

	// Set default payload
	payloadStr := "{}"
	sched.Payload = &payloadStr

	// Check for conflicts before creating (provides better error messages)
	// The database EXCLUDE constraint provides the final atomic guarantee

	// Step 1: Check conflicts for the first instance (or single event)
	conflicts, err := s.CheckConflicts(ctx, userID, create.StartTs, create.EndTs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrScheduleConflict, buildConflictError(conflicts))
	}

	// Step 2: For recurring schedules, check conflicts for future instances
	if create.RecurrenceRule != nil && *create.RecurrenceRule != "" {
		recurringConflicts, err := s.checkRecurringConflicts(ctx, userID, create)
		if err != nil {
			return nil, fmt.Errorf("failed to check recurring conflicts: %w", err)
		}
		if len(recurringConflicts) > 0 {
			return nil, fmt.Errorf("%w: %s", ErrScheduleConflict, buildRecurringConflictError(recurringConflicts))
		}
	}

	// Create schedule in database
	// The database will atomically verify no conflicts exist via EXCLUDE constraint
	created, err := s.store.CreateSchedule(ctx, sched)
	if err != nil {
		// Check for database-level conflict constraint violation
		var conflictErr *postgresstore.ConflictConstraintError
		if errors.As(err, &conflictErr) {
			// Return a more user-friendly error
			return nil, fmt.Errorf("%w: %s", ErrScheduleConflict,
				"this time slot overlaps with an existing schedule (detected atomically)")
		}
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return created, nil
}

// UpdateSchedule updates an existing schedule.
func (s *service) UpdateSchedule(ctx context.Context, userID int32, id int32, update *UpdateScheduleRequest) (*store.Schedule, error) {
	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		ID:        &id,
		CreatorID: &userID,
	}
	existing, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("schedule not found")
	}

	// Build update request
	storeUpdate := &store.UpdateSchedule{ID: id}

	if update.Title != nil {
		storeUpdate.Title = update.Title
	}
	if update.Description != nil {
		storeUpdate.Description = update.Description
	}
	if update.Location != nil {
		storeUpdate.Location = update.Location
	}
	if update.StartTs != nil {
		storeUpdate.StartTs = update.StartTs
	}
	if update.EndTs != nil {
		storeUpdate.EndTs = update.EndTs
	}
	if update.AllDay != nil {
		storeUpdate.AllDay = update.AllDay
	}
	if update.Timezone != nil {
		storeUpdate.Timezone = update.Timezone
	}
	if update.RecurrenceRule != nil {
		storeUpdate.RecurrenceRule = update.RecurrenceRule
	}
	if update.RecurrenceEndTs != nil {
		storeUpdate.RecurrenceEndTs = update.RecurrenceEndTs
	}
	if update.RowStatus != nil {
		storeUpdate.RowStatus = update.RowStatus
	}
	if update.Reminders != nil {
		reminders := make([]*v1pb.Reminder, len(update.Reminders))
		for i, r := range update.Reminders {
			reminders[i] = &v1pb.Reminder{
				Type:  r.Type,
				Value: r.Value,
				Unit:  r.Unit,
			}
		}
		remindersStr, err := aischedule.MarshalReminders(reminders)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal reminders: %w", err)
		}
		storeUpdate.Reminders = &remindersStr
	}

	// Determine new time values for conflict checking
	newStartTs := existing.StartTs
	newEndTs := existing.EndTs

	if update.StartTs != nil {
		newStartTs = *update.StartTs
	}
	if update.EndTs != nil {
		newEndTs = update.EndTs
	}

	// Check for conflicts (excluding the current schedule itself)
	excludeIDs := []int32{id}
	conflicts, err := s.CheckConflicts(ctx, userID, newStartTs, newEndTs, excludeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrScheduleConflict, buildConflictError(conflicts))
	}

	// Update schedule in database
	if err := s.store.UpdateSchedule(ctx, storeUpdate); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	// Fetch updated schedule
	updated, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated schedule: %w", err)
	}

	return updated, nil
}

// DeleteSchedule deletes a schedule by ID.
func (s *service) DeleteSchedule(ctx context.Context, userID int32, id int32) error {
	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		ID:        &id,
		CreatorID: &userID,
	}
	existing, err := s.store.GetSchedule(ctx, find)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("schedule not found")
	}

	// Delete schedule
	if err := s.store.DeleteSchedule(ctx, &store.DeleteSchedule{ID: id}); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}

// CheckConflicts checks for schedule conflicts within a time range.
// Only checks against active (NORMAL) schedules, excluding archived ones.
func (s *service) CheckConflicts(ctx context.Context, userID int32, startTs int64, endTs *int64, excludeIDs []int32) ([]*store.Schedule, error) {
	// Determine end time for conflict check
	checkEndTs := startTs
	if endTs != nil && *endTs > startTs {
		checkEndTs = *endTs
	} else {
		// Default to 1 hour from start if not specified
		checkEndTs = startTs + int64(DefaultConflictCheckWindow.Seconds())
	}

	// Find schedules that might conflict within the time window
	// Only fetch active (NORMAL) schedules to avoid conflicts with archived ones
	normalStatus := store.Normal
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &startTs,
		EndTs:     &checkEndTs,
		RowStatus: &normalStatus,
	}

	list, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Filter out excluded schedules and check for actual conflicts
	var conflicts []*store.Schedule
	excludeSet := make(map[int32]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	for _, sched := range list {
		if !excludeSet[sched.ID] {
			// Check if time ranges actually overlap
			// 区间约定 Convention: [start, end) 左闭右开
			// Two intervals [s1, e1) and [s2, e2) overlap if: s1 < e2 AND s2 < e1
			scheduleEnd := sched.EndTs
			if scheduleEnd == nil {
				// For schedules without end time, treat as a point event at start_ts
				scheduleEnd = &sched.StartTs
			}

			// Check overlap: query window [startTs, checkEndTs) vs schedule [sched.StartTs, *scheduleEnd)
			// Using [start, end) convention: overlap when new.start < existing.end AND new.end > existing.start
			if startTs < *scheduleEnd && checkEndTs > sched.StartTs {
				conflicts = append(conflicts, sched)
			}
		}
	}

	return conflicts, nil
}

// Helper functions

// convertToInstance converts a store.Schedule to a ScheduleInstance.
func (s *service) convertToInstance(sched *store.Schedule, isRecurring bool, parentUID string) *ScheduleInstance {
	return &ScheduleInstance{
		ID:          sched.ID,
		UID:         sched.UID,
		Title:       sched.Title,
		Description: sched.Description,
		Location:    sched.Location,
		StartTs:     sched.StartTs,
		EndTs:       sched.EndTs,
		AllDay:      sched.AllDay,
		Timezone:    sched.Timezone,
		IsRecurring: isRecurring,
		ParentUID:   parentUID,
	}
}

// buildConflictError builds a human-readable error message for schedule conflicts.
func buildConflictError(conflicts []*store.Schedule) string {
	if len(conflicts) == 0 {
		return ""
	}

	if len(conflicts) == 1 {
		c := conflicts[0]
		return fmt.Sprintf("conflicts with existing schedule \"%s\" (from %s to %s)",
			c.Title,
			formatTimestamp(c.StartTs, c.Timezone),
			formatEndTs(c.EndTs, c.Timezone))
	}

	var titles []string
	for _, c := range conflicts {
		titles = append(titles, fmt.Sprintf("\"%s\"", c.Title))
	}

	return fmt.Sprintf("conflicts with %d existing schedules: %s",
		len(conflicts),
		strings.Join(titles, ", "))
}

// formatEndTs formats an end timestamp for display in a human-readable format.
func formatEndTs(endTs *int64, timezone string) string {
	if endTs == nil {
		return "no end time"
	}
	return formatTimestamp(*endTs, timezone)
}

// formatTimestamp formats a Unix timestamp for display in the given timezone.
func formatTimestamp(ts int64, timezone string) string {
	t := time.Unix(ts, 0)

	// Parse timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fallback to UTC if timezone is invalid
		slog.Warn("invalid timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	return t.In(loc).Format("2006-01-02 15:04")
}

// checkRecurringConflicts checks for conflicts in recurring schedule instances.
// Uses index-based approach and iterator for improved performance with long-running recurrences.
func (s *service) checkRecurringConflicts(ctx context.Context, userID int32, create *CreateScheduleRequest) ([]*RecurringConflict, error) {
	// Parse recurrence rule
	rule, err := aischedule.ParseRecurrenceRuleFromJSON(*create.RecurrenceRule)
	if err != nil {
		return nil, fmt.Errorf("invalid recurrence rule: %w", err)
	}

	// Determine the end time for instance generation
	var endTs int64
	if create.RecurrenceEndTs != nil && *create.RecurrenceEndTs > 0 {
		endTs = *create.RecurrenceEndTs
	} else {
		// For infinite recurrence, check 1 year ahead
		endTs = create.StartTs + 365*24*3600
	}

	// Calculate duration for each instance
	duration := int64(DefaultConflictCheckWindow.Seconds())
	if create.EndTs != nil && *create.EndTs > create.StartTs {
		duration = *create.EndTs - create.StartTs
	}

	// Batch query: fetch all schedules that could potentially conflict
	// Query range covers the entire recurrence period
	normalStatus := store.Normal
	find := &store.FindSchedule{
		CreatorID: &userID,
		RowStatus: &normalStatus,
		StartTs:   &create.StartTs,
		EndTs:     &endTs,
	}

	potentialConflicts, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules for conflict check: %w", err)
	}

	// Build conflict index for O(1) lookup
	conflictIndex := s.buildConflictIndex(potentialConflicts)

	// Use iterator for lazy evaluation
	iterator := rule.Iterator(create.StartTs)

	// Check instances one by one (up to increased limit)
	const maxCheckCount = 500 // Increased from MaxInstancesToCheck (100)
	checkCount := 0

	for {
		instanceTs := iterator.Next()
		if instanceTs == 0 {
			break
		}
		if instanceTs > endTs {
			break
		}
		if checkCount >= maxCheckCount {
			slog.Warn("recurring conflict check hit limit",
				"limit", maxCheckCount,
				"user_id", userID)
			break
		}

		// Skip first instance - already checked in CreateSchedule
		if instanceTs == create.StartTs {
			checkCount++
			continue
		}

		instanceEndTs := instanceTs + duration

		// Check for conflict using index
		if s.hasConflictInIndex(conflictIndex, instanceTs, instanceEndTs) {
			// Find the conflicting schedule for detailed error
			conflictingSchedule := s.findConflictAt(conflictIndex, instanceTs, instanceEndTs)
			if conflictingSchedule != nil {
				return []*RecurringConflict{
					{
						ExistingSchedule: conflictingSchedule,
						InstanceStartTs:  instanceTs,
						InstanceEndTs:    instanceEndTs,
					},
				}, nil
			}
		}

		checkCount++
	}

	return nil, nil
}

// buildConflictIndex builds an hour-indexed map for efficient conflict lookup.
// Using hourly buckets avoids timezone issues and provides better precision than daily buckets.
func (s *service) buildConflictIndex(schedules []*store.Schedule) map[int64][]*store.Schedule {
	index := make(map[int64][]*store.Schedule)
	for _, sched := range schedules {
		// Index by hour for efficient range queries
		// Each schedule is added to all hour buckets it spans
		startHour := sched.StartTs / 3600
		var endHour int64
		if sched.EndTs != nil && *sched.EndTs > sched.StartTs {
			endHour = *sched.EndTs / 3600
		} else {
			endHour = startHour + 1 // Default 1 hour
		}

		// Add to each hour bucket this schedule spans
		for hour := startHour; hour <= endHour; hour++ {
			index[hour] = append(index[hour], sched)
		}
	}
	return index
}

// hasConflictInIndex checks if a time range conflicts with any schedule in the index.
func (s *service) hasConflictInIndex(index map[int64][]*store.Schedule, startTs, endTs int64) bool {
	startHour := startTs / 3600
	endHour := endTs / 3600

	// Check each hour bucket that the range spans
	// Add buffer of +1 hour to catch schedules that start/end within the range
	for hour := startHour; hour <= endHour+1; hour++ {
		for _, sched := range index[hour] {
			schedEnd := sched.EndTs
			if schedEnd == nil {
				schedEnd = &sched.StartTs
			}
			// Overlap check: [start, end) convention
			if startTs < *schedEnd && endTs > sched.StartTs {
				return true
			}
		}
	}
	return false
}

// findConflictAt finds the specific schedule that conflicts with the given time range.
func (s *service) findConflictAt(index map[int64][]*store.Schedule, startTs, endTs int64) *store.Schedule {
	startHour := startTs / 3600
	endHour := endTs / 3600

	for hour := startHour; hour <= endHour+1; hour++ {
		for _, sched := range index[hour] {
			schedEnd := sched.EndTs
			if schedEnd == nil {
				schedEnd = &sched.StartTs
			}
			if startTs < *schedEnd && endTs > sched.StartTs {
				return sched
			}
		}
	}
	return nil
}

// RecurringConflict represents a conflict with a recurring schedule instance.
type RecurringConflict struct {
	ExistingSchedule *store.Schedule
	InstanceStartTs  int64
	InstanceEndTs    int64
}

// buildRecurringConflictError builds a human-readable error message for recurring conflicts.
func buildRecurringConflictError(conflicts []*RecurringConflict) string {
	if len(conflicts) == 0 {
		return ""
	}

	if len(conflicts) == 1 {
		c := conflicts[0]
		return fmt.Sprintf("recurring schedule conflicts with existing schedule \"%s\" on %s",
			c.ExistingSchedule.Title,
			formatTimestamp(c.InstanceStartTs, c.ExistingSchedule.Timezone))
	}

	// Multiple conflicts - summarize them
	var parts []string
	for _, c := range conflicts {
		parts = append(parts, fmt.Sprintf("\"%s\" on %s",
			c.ExistingSchedule.Title,
			formatTimestamp(c.InstanceStartTs, c.ExistingSchedule.Timezone)))
	}

	return fmt.Sprintf("recurring schedule conflicts with %d existing schedules: %s",
		len(conflicts),
		strings.Join(parts, ", "))
}
