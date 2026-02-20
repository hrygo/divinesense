package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hrygo/divinesense/ai"
	aischedule "github.com/hrygo/divinesense/ai/services/schedule"
	"github.com/hrygo/divinesense/internal/util"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/store"
)

// ScheduleService provides schedule management APIs.
type ScheduleService struct {
	v1pb.UnimplementedScheduleServiceServer

	Store      *store.Store
	LLMService ai.LLMService
}

// scheduleFromStore converts a store.Schedule to v1pb.Schedule.
func scheduleFromStore(s *store.Schedule) *v1pb.Schedule {
	pb := &v1pb.Schedule{
		Name:      fmt.Sprintf("schedules/%s", s.UID),
		Title:     s.Title,
		StartTs:   s.StartTs,
		AllDay:    s.AllDay,
		Timezone:  s.Timezone,
		CreatedTs: s.CreatedTs,
		UpdatedTs: s.UpdatedTs,
		State:     s.RowStatus.String(),
	}

	if s.Description != "" {
		pb.Description = s.Description
	}
	if s.Location != "" {
		pb.Location = s.Location
	}
	if s.EndTs != nil {
		pb.EndTs = *s.EndTs
	}
	if s.RecurrenceRule != nil {
		pb.RecurrenceRule = *s.RecurrenceRule
	}
	if s.RecurrenceEndTs != nil {
		pb.RecurrenceEndTs = *s.RecurrenceEndTs
	}
	if s.CreatorID != 0 {
		pb.Creator = fmt.Sprintf("users/%d", s.CreatorID)
	}

	// Parse reminders from JSON
	if s.Reminders != nil && *s.Reminders != "" && *s.Reminders != "[]" {
		var reminders []map[string]interface{}
		if err := json.Unmarshal([]byte(*s.Reminders), &reminders); err == nil {
			for _, r := range reminders {
				reminder := &v1pb.Reminder{}
				if t, ok := r["type"].(string); ok {
					reminder.Type = t
				}
				if v, ok := r["value"].(float64); ok {
					reminder.Value = int32(v)
				}
				if u, ok := r["unit"].(string); ok {
					reminder.Unit = u
				}
				pb.Reminders = append(pb.Reminders, reminder)
			}
		}
	}

	return pb
}

// scheduleToStore converts a v1pb.Schedule to store.Schedule.
func scheduleToStore(pb *v1pb.Schedule, creatorID int32) (*store.Schedule, error) {
	// Parse UID from name
	uid := strings.TrimPrefix(pb.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	// Validate required fields
	if strings.TrimSpace(pb.Title) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "title is required")
	}
	if pb.StartTs <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "start_ts must be a positive timestamp")
	}
	if pb.EndTs != 0 && pb.EndTs < pb.StartTs {
		return nil, status.Errorf(codes.InvalidArgument, "end_ts must be greater than or equal to start_ts")
	}

	// Set default timezone if not provided
	timezone := pb.Timezone
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	// Validate reminders count
	const maxReminders = 10
	if len(pb.Reminders) > maxReminders {
		return nil, status.Errorf(codes.InvalidArgument, "too many reminders: maximum %d allowed, got %d", maxReminders, len(pb.Reminders))
	}

	s := &store.Schedule{
		UID:         uid,
		CreatorID:   creatorID,
		Title:       pb.Title,
		StartTs:     pb.StartTs,
		AllDay:      pb.AllDay,
		Timezone:    timezone,
		RowStatus:   store.RowStatus(pb.State),
		Description: pb.Description,
		Location:    pb.Location,
	}

	if pb.EndTs != 0 {
		s.EndTs = &pb.EndTs
	}
	if pb.RecurrenceRule != "" {
		s.RecurrenceRule = &pb.RecurrenceRule
	}
	if pb.RecurrenceEndTs != 0 {
		s.RecurrenceEndTs = &pb.RecurrenceEndTs
	}

	// Convert reminders to JSON
	var remindersStr string
	if len(pb.Reminders) > 0 {
		reminders := make([]*v1pb.Reminder, 0, len(pb.Reminders))
		for _, r := range pb.Reminders {
			// Validate reminder fields
			if r.Type == "" {
				return nil, status.Errorf(codes.InvalidArgument, "reminder type is required")
			}
			if r.Unit == "" {
				return nil, status.Errorf(codes.InvalidArgument, "reminder unit is required")
			}
			reminders = append(reminders, &v1pb.Reminder{
				Type:  r.Type,
				Value: r.Value,
				Unit:  r.Unit,
			})
		}

		// Use helper function to marshal reminders
		var err error
		remindersStr, err = aischedule.MarshalReminders(reminders)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
		}
	} else {
		// Use empty JSON array instead of empty string to satisfy NOT NULL constraint
		remindersStr = "[]"
	}
	s.Reminders = &remindersStr

	// Set default payload
	payloadStr := "{}"
	s.Payload = &payloadStr

	return s, nil
}

// CreateSchedule creates a new schedule.
func (s *ScheduleService) CreateSchedule(ctx context.Context, req *v1pb.CreateScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	schedule, err := scheduleToStore(req.Schedule, userID)
	if err != nil {
		return nil, err
	}

	// Generate UID if not provided
	if schedule.UID == "" || schedule.UID == "schedules/" {
		schedule.UID = util.GenUUID()
	}

	// Check for conflicts before creating
	conflicts, err := s.checkScheduleConflicts(ctx, userID, schedule.StartTs, schedule.EndTs, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check conflicts: %v", err)
	}
	if len(conflicts) > 0 {
		// Build conflict details
		conflictDetails := buildConflictError(conflicts)
		return nil, status.Errorf(codes.AlreadyExists, "schedule conflicts detected: %s", conflictDetails)
	}

	created, err := s.Store.CreateSchedule(ctx, schedule)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create schedule: %v", err)
	}

	return scheduleFromStore(created), nil
}

// ListSchedules lists schedules with filters.
func (s *ScheduleService) ListSchedules(ctx context.Context, req *v1pb.ListSchedulesRequest) (*v1pb.ListSchedulesResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	find := &store.FindSchedule{
		Limit: pointerOf(100), // Default limit
	}

	// Parse creator from name
	if req.Creator != "" {
		creatorID := strings.TrimPrefix(req.Creator, "users/")
		if creatorID == "" {
			return nil, status.Errorf(codes.InvalidArgument, "invalid creator format")
		}
		id, err := parseInt32(creatorID)
		if err != nil {
			return nil, err
		}
		find.CreatorID = &id
	} else {
		// Default to current user
		find.CreatorID = &userID
	}

	// NOTE: For recurring schedules, we need to query without time constraints first
	// to get the schedule templates, then expand instances
	if req.StartTs != 0 {
		find.StartTs = &req.StartTs
	}
	if req.EndTs != 0 {
		find.EndTs = &req.EndTs
	}
	if req.State != "" {
		rowStatus := store.RowStatus(req.State)
		find.RowStatus = &rowStatus
	}
	if req.PageSize != 0 {
		limit := int(req.PageSize)
		find.Limit = &limit
	}

	list, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list schedules: %v", err)
	}

	// Expand recurring schedules
	var expandedSchedules []*v1pb.Schedule
	queryStartTs := req.StartTs
	queryEndTs := req.EndTs

	// Default query window: from now to 30 days later
	now := time.Now().Unix()
	if queryStartTs == 0 {
		queryStartTs = now
	}
	if queryEndTs == 0 {
		queryEndTs = now + 30*24*3600 // Default to 30 days
	}

	// Limit total instances to prevent performance issues
	// Use page size to determine limit, with a hard maximum
	maxTotalInstances := 100 // Default value
	if req.PageSize > 0 {
		maxTotalInstances = int(req.PageSize) * 2 // Double for safety margin
	}
	if maxTotalInstances > 500 {
		maxTotalInstances = 500 // Hard limit to prevent excessive data
	}

	truncated := false

	for _, schedule := range list {
		// Check total instance limit before processing each schedule
		if len(expandedSchedules) >= maxTotalInstances {
			truncated = true
			break
		}

		pbSchedule := scheduleFromStore(schedule)

		// If this is a recurring schedule, expand it
		if schedule.RecurrenceRule != nil && *schedule.RecurrenceRule != "" {
			// Parse recurrence rule
			rule, err := aischedule.ParseRecurrenceRuleFromJSON(*schedule.RecurrenceRule)
			if err != nil {
				// If parsing fails, just return the base schedule
				expandedSchedules = append(expandedSchedules, pbSchedule)
				continue
			}

			// Generate instances starting from the schedule's start time
			// This ensures we get the correct sequence from the first occurrence
			instances := rule.GenerateInstances(pbSchedule.StartTs, queryEndTs)

			// For each instance, create a schedule with adjusted time
			for _, instanceTs := range instances {
				// Check if we've hit the total instance limit
				if len(expandedSchedules) >= maxTotalInstances {
					truncated = true
					break
				}

				// Only add instances within the query window
				if instanceTs < queryStartTs || instanceTs > queryEndTs {
					continue
				}

				instance := &v1pb.Schedule{
					Name:        fmt.Sprintf("%s/instances/%d", pbSchedule.Name, instanceTs),
					Title:       pbSchedule.Title,
					Description: pbSchedule.Description,
					Location:    pbSchedule.Location,
					StartTs:     instanceTs,
					AllDay:      pbSchedule.AllDay,
					Timezone:    pbSchedule.Timezone,
					Reminders:   pbSchedule.Reminders,
					Creator:     pbSchedule.Creator,
					State:       pbSchedule.State,
				}

				// Calculate end time for this instance
				if pbSchedule.EndTs > 0 && pbSchedule.StartTs > 0 {
					duration := pbSchedule.EndTs - pbSchedule.StartTs
					instance.EndTs = instanceTs + duration
				}

				expandedSchedules = append(expandedSchedules, instance)

				// Break if we've hit the limit
				if len(expandedSchedules) >= maxTotalInstances {
					truncated = true
					break
				}
			}
		} else {
			// Non-recurring schedule, add as-is
			expandedSchedules = append(expandedSchedules, pbSchedule)
		}
	}

	// Log warning if truncated
	if truncated {
		slog.Warn("schedule instance expansion truncated",
			"count", len(expandedSchedules),
			"limit", maxTotalInstances,
			"user_id", userID)
	}

	return &v1pb.ListSchedulesResponse{
		Schedules: expandedSchedules,
		Truncated: truncated,
	}, nil
}

// GetSchedule gets a schedule by name.
func (s *ScheduleService) GetSchedule(ctx context.Context, req *v1pb.GetScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}

	schedule, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if schedule == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	return scheduleFromStore(schedule), nil
}

var scheduleFieldMappers = map[string]func(*v1pb.Schedule, *store.UpdateSchedule) error{
	"title":       func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.Title = &pb.Title; return nil },
	"description": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.Description = &pb.Description; return nil },
	"location":    func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.Location = &pb.Location; return nil },
	"start_ts":    func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.StartTs = &pb.StartTs; return nil },
	"end_ts": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error {
		if pb.EndTs != 0 {
			u.EndTs = &pb.EndTs
		}
		return nil
	},
	"all_day":  func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.AllDay = &pb.AllDay; return nil },
	"timezone": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error { u.Timezone = &pb.Timezone; return nil },
	"recurrence_rule": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error {
		u.RecurrenceRule = &pb.RecurrenceRule
		return nil
	},
	"recurrence_end_ts": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error {
		if pb.RecurrenceEndTs != 0 {
			u.RecurrenceEndTs = &pb.RecurrenceEndTs
		}
		return nil
	},
	"state": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error {
		rowStatus := store.RowStatus(pb.State)
		u.RowStatus = &rowStatus
		return nil
	},
	"reminders": func(pb *v1pb.Schedule, u *store.UpdateSchedule) error {
		if len(pb.Reminders) > 0 {
			reminders := make([]*v1pb.Reminder, 0, len(pb.Reminders))
			for _, r := range pb.Reminders {
				reminders = append(reminders, &v1pb.Reminder{
					Type:  r.Type,
					Value: r.Value,
					Unit:  r.Unit,
				})
			}
			remindersStr, err := aischedule.MarshalReminders(reminders)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
			}
			u.Reminders = &remindersStr
		}
		return nil
	},
}

// UpdateSchedule updates a schedule.
func (s *ScheduleService) UpdateSchedule(ctx context.Context, req *v1pb.UpdateScheduleRequest) (*v1pb.Schedule, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Schedule.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	// Get existing schedule
	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}
	existing, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if existing == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	// Build update request
	update := &store.UpdateSchedule{
		ID: existing.ID,
	}

	var paths []string
	if req.UpdateMask != nil {
		paths = req.UpdateMask.Paths
	} else {
		// Infer paths from non-empty fields
		if req.Schedule.Title != "" {
			paths = append(paths, "title")
		}
		if req.Schedule.Description != "" {
			paths = append(paths, "description")
		}
		if req.Schedule.Location != "" {
			paths = append(paths, "location")
		}
		if req.Schedule.StartTs != 0 {
			paths = append(paths, "start_ts")
		}
		if req.Schedule.EndTs != 0 {
			paths = append(paths, "end_ts")
		}
		paths = append(paths, "all_day") // Always update boolean if provided
		if req.Schedule.Timezone != "" {
			paths = append(paths, "timezone")
		}
		if req.Schedule.RecurrenceRule != "" {
			paths = append(paths, "recurrence_rule")
		}
		if req.Schedule.RecurrenceEndTs != 0 {
			paths = append(paths, "recurrence_end_ts")
		}
		if req.Schedule.State != "" {
			paths = append(paths, "state")
		}
		if len(req.Schedule.Reminders) > 0 {
			paths = append(paths, "reminders")
		}
	}

	for _, path := range paths {
		if mapper, ok := scheduleFieldMappers[path]; ok {
			if err := mapper(req.Schedule, update); err != nil {
				return nil, err
			}
		}
	}

	// Check for conflicts if time fields are being updated
	// Determine the new time values (use existing if not being updated)
	newStartTs := existing.StartTs
	newEndTs := existing.EndTs

	if update.StartTs != nil {
		newStartTs = *update.StartTs
	}
	if update.EndTs != nil {
		newEndTs = update.EndTs
	}

	// Check for conflicts (excluding the current schedule itself)
	conflicts, err := s.checkScheduleConflicts(ctx, userID, newStartTs, newEndTs, []string{req.Schedule.Name})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check conflicts: %v", err)
	}
	if len(conflicts) > 0 {
		// Build conflict details
		conflictDetails := buildConflictError(conflicts)
		return nil, status.Errorf(codes.AlreadyExists, "schedule conflicts detected: %s", conflictDetails)
	}

	if err := s.Store.UpdateSchedule(ctx, update); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update schedule: %v", err)
	}

	// Fetch updated schedule
	updated, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updated schedule: %v", err)
	}

	return scheduleFromStore(updated), nil
}

// DeleteSchedule deletes a schedule.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, req *v1pb.DeleteScheduleRequest) (*emptypb.Empty, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	uid := strings.TrimPrefix(req.Name, "schedules/")
	if uid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule name format")
	}

	// Get existing schedule to verify ownership
	find := &store.FindSchedule{
		UID:       &uid,
		CreatorID: &userID,
	}
	existing, err := s.Store.GetSchedule(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get schedule: %v", err)
	}
	if existing == nil {
		return nil, status.Errorf(codes.NotFound, "schedule not found")
	}

	if err := s.Store.DeleteSchedule(ctx, &store.DeleteSchedule{ID: existing.ID}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete schedule: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// CheckConflict checks for schedule conflicts.
func (s *ScheduleService) CheckConflict(ctx context.Context, req *v1pb.CheckConflictRequest) (*v1pb.CheckConflictResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate time range
	if req.StartTs <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "start_ts must be positive")
	}

	endTs := req.EndTs
	if endTs == 0 {
		// Default to 1 hour from start if not specified
		endTs = req.StartTs + 3600
	}
	if endTs < req.StartTs {
		return nil, status.Errorf(codes.InvalidArgument, "end_ts must be >= start_ts")
	}

	// Find schedules that might conflict within the time window
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &req.StartTs,
		EndTs:     &endTs,
	}

	list, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check conflicts: %v", err)
	}

	// Filter out excluded schedules and check for actual conflicts
	var conflicts []*store.Schedule
	excludeSet := make(map[string]bool)
	for _, name := range req.ExcludeNames {
		excludeSet[name] = true
	}

	for _, schedule := range list {
		name := fmt.Sprintf("schedules/%s", schedule.UID)
		if !excludeSet[name] {
			if checkTimeOverlap(req.StartTs, endTs, schedule.StartTs, schedule.EndTs) {
				conflicts = append(conflicts, schedule)
			}
		}
	}

	response := &v1pb.CheckConflictResponse{
		Conflicts: make([]*v1pb.Schedule, len(conflicts)),
	}
	for i, c := range conflicts {
		response.Conflicts[i] = scheduleFromStore(c)
	}

	return response, nil
}

// Helper functions

// Two intervals [s1, e1) and [s2, e2) overlap if: s1 < e2 AND s2 < e1.
func checkTimeOverlap(start1, end1, start2 int64, end2 *int64) bool {
	scheduleEnd := end2
	if scheduleEnd == nil {
		// For schedules without end time, treat as a point event at start_ts
		scheduleEnd = &start2
	}
	// Using [start, end) convention: overlap when new.start < existing.end AND new.end > existing.start
	return start1 < *scheduleEnd && end1 > start2
}

func pointerOf[T any](v T) *T {
	return &v
}

func parseInt32(s string) (int32, error) {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, status.Errorf(codes.InvalidArgument, "invalid ID format: %s", s)
	}
	return int32(i), nil
}

// checkScheduleConflicts checks for schedule conflicts within a time range.
// Returns a list of conflicting schedules.
func (s *ScheduleService) checkScheduleConflicts(ctx context.Context, userID int32, startTs int64, endTs *int64, excludeNames []string) ([]*store.Schedule, error) {
	// Determine end time for conflict check
	checkEndTs := startTs
	if endTs != nil && *endTs > startTs {
		checkEndTs = *endTs
	} else {
		// Default to 1 hour from start if not specified
		checkEndTs = startTs + 3600
	}

	// Find schedules that might conflict within the time window
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   &startTs,
		EndTs:     &checkEndTs,
	}

	list, err := s.Store.ListSchedules(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Filter out excluded schedules and check for actual conflicts
	var conflicts []*store.Schedule
	excludeSet := make(map[string]bool)
	for _, name := range excludeNames {
		excludeSet[name] = true
	}

	for _, schedule := range list {
		name := fmt.Sprintf("schedules/%s", schedule.UID)
		if !excludeSet[name] {
			if checkTimeOverlap(startTs, checkEndTs, schedule.StartTs, schedule.EndTs) {
				conflicts = append(conflicts, schedule)
			}
		}
	}

	return conflicts, nil
}

// buildConflictError builds a human-readable error message for schedule conflicts.
func buildConflictError(conflicts []*store.Schedule) string {
	if len(conflicts) == 0 {
		return ""
	}

	if len(conflicts) == 1 {
		c := conflicts[0]
		return fmt.Sprintf("conflicts with existing schedule \"%s\" (from %d to %s)",
			c.Title,
			c.StartTs,
			formatEndTs(c.EndTs))
	}

	var titles []string
	for _, c := range conflicts {
		titles = append(titles, fmt.Sprintf("\"%s\"", c.Title))
	}

	return fmt.Sprintf("conflicts with %d existing schedules: %s",
		len(conflicts),
		strings.Join(titles, ", "))
}

// formatEndTs formats an end timestamp for display.
func formatEndTs(endTs *int64) string {
	if endTs == nil {
		return "no end time"
	}
	return fmt.Sprintf("%d", *endTs)
}
