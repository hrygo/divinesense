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
	aischedule "github.com/hrygo/divinesense/ai/schedule"
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

	if req.UpdateMask != nil {
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "title":
				update.Title = &req.Schedule.Title
			case "description":
				update.Description = &req.Schedule.Description
			case "location":
				update.Location = &req.Schedule.Location
			case "start_ts":
				update.StartTs = &req.Schedule.StartTs
			case "end_ts":
				if req.Schedule.EndTs != 0 {
					update.EndTs = &req.Schedule.EndTs
				}
			case "all_day":
				update.AllDay = &req.Schedule.AllDay
			case "timezone":
				update.Timezone = &req.Schedule.Timezone
			case "recurrence_rule":
				update.RecurrenceRule = &req.Schedule.RecurrenceRule
			case "recurrence_end_ts":
				if req.Schedule.RecurrenceEndTs != 0 {
					update.RecurrenceEndTs = &req.Schedule.RecurrenceEndTs
				}
			case "state":
				rowStatus := store.RowStatus(req.Schedule.State)
				update.RowStatus = &rowStatus
			case "reminders":
				// Convert reminders to JSON
				if len(req.Schedule.Reminders) > 0 {
					reminders := make([]*v1pb.Reminder, 0, len(req.Schedule.Reminders))
					for _, r := range req.Schedule.Reminders {
						reminders = append(reminders, &v1pb.Reminder{
							Type:  r.Type,
							Value: r.Value,
							Unit:  r.Unit,
						})
					}
					remindersStr, err := aischedule.MarshalReminders(reminders)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
					}
					update.Reminders = &remindersStr
				}
			}
		}
	} else {
		// If no UpdateMask provided, update all non-zero/non-empty fields
		if req.Schedule.Title != "" {
			update.Title = &req.Schedule.Title
		}
		if req.Schedule.Description != "" {
			update.Description = &req.Schedule.Description
		}
		if req.Schedule.Location != "" {
			update.Location = &req.Schedule.Location
		}
		if req.Schedule.StartTs != 0 {
			update.StartTs = &req.Schedule.StartTs
		}
		if req.Schedule.EndTs != 0 {
			update.EndTs = &req.Schedule.EndTs
		}
		// Always update boolean if provided
		update.AllDay = &req.Schedule.AllDay
		if req.Schedule.Timezone != "" {
			update.Timezone = &req.Schedule.Timezone
		}
		if req.Schedule.RecurrenceRule != "" {
			update.RecurrenceRule = &req.Schedule.RecurrenceRule
		}
		if req.Schedule.RecurrenceEndTs != 0 {
			update.RecurrenceEndTs = &req.Schedule.RecurrenceEndTs
		}
		if req.Schedule.State != "" {
			rowStatus := store.RowStatus(req.Schedule.State)
			update.RowStatus = &rowStatus
		}
		// Convert reminders to JSON if provided
		if len(req.Schedule.Reminders) > 0 {
			reminders := make([]*v1pb.Reminder, 0, len(req.Schedule.Reminders))
			for _, r := range req.Schedule.Reminders {
				reminders = append(reminders, &v1pb.Reminder{
					Type:  r.Type,
					Value: r.Value,
					Unit:  r.Unit,
				})
			}
			remindersStr, err := aischedule.MarshalReminders(reminders)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to marshal reminders: %v", err)
			}
			update.Reminders = &remindersStr
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

// ParseAndCreateSchedule parses natural language and creates a schedule.
func (s *ScheduleService) ParseAndCreateSchedule(ctx context.Context, req *v1pb.ParseAndCreateScheduleRequest) (*v1pb.ParseAndCreateScheduleResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate input
	if strings.TrimSpace(req.Text) == "" {
		return nil, status.Errorf(codes.InvalidArgument, "text is required")
	}

	// TODO: Get timezone from user settings instead of hardcoding
	// For now, use Asia/Shanghai as default
	// Future enhancement: user, err := s.Store.GetUser(ctx, &store.FindUser{ID: &userID})
	// timezone := user.Timezone
	timezone := "Asia/Shanghai"

	// Create parser
	parser, err := aischedule.NewParser(s.LLMService, timezone)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create parser: %v", err)
	}

	// Parse natural language
	parsed, err := parser.Parse(ctx, req.Text)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse text: %v", err)
	}

	response := &v1pb.ParseAndCreateScheduleResponse{
		ParsedSchedule: parsed.ToSchedule(),
	}

	// If autoConfirm is true, create the schedule
	if req.AutoConfirm {

		// Create schedule
		schedule, err := scheduleToStore(parsed.ToSchedule(), userID)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schedule: %v", err)
		}

		// Generate UID
		schedule.UID = util.GenUUID()

		// Check for conflicts before creating
		conflicts, err := s.checkScheduleConflicts(ctx, userID, schedule.StartTs, schedule.EndTs, nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check conflicts: %v", err)
		}
		if len(conflicts) > 0 {
			// Return conflicts in response instead of creating
			response.Conflicts = make([]*v1pb.Schedule, len(conflicts))
			for i, c := range conflicts {
				response.Conflicts[i] = scheduleFromStore(c)
			}
			return response, nil
		}

		created, err := s.Store.CreateSchedule(ctx, schedule)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create schedule: %v", err)
		}

		response.CreatedSchedule = scheduleFromStore(created)
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

// PrecheckSchedule validates a schedule before creation.
func (s *ScheduleService) PrecheckSchedule(ctx context.Context, req *v1pb.PrecheckScheduleRequest) (*v1pb.PrecheckScheduleResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate required field
	if req.StartTs <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "start_ts is required")
	}

	// Create precheck service
	precheckService := aischedule.NewPrecheckService(s.Store)

	// Convert proto request to internal request
	startTime := time.Unix(req.StartTs, 0)
	var endTime time.Time
	if req.EndTs > 0 {
		endTime = time.Unix(req.EndTs, 0)
	}

	precheckReq := &aischedule.PrecheckRequest{
		Title:     req.Title,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  int(req.Duration),
		Location:  req.Location,
	}

	// Perform precheck
	result := precheckService.Precheck(ctx, userID, precheckReq)

	// Convert internal result to proto response
	response := &v1pb.PrecheckScheduleResponse{
		Valid: result.Valid,
	}

	// Convert errors
	for _, e := range result.Errors {
		response.Errors = append(response.Errors, &v1pb.PrecheckError{
			Code:    e.Code,
			Message: e.Message,
			Field:   e.Field,
		})
	}

	// Convert warnings
	for _, w := range result.Warnings {
		response.Warnings = append(response.Warnings, &v1pb.PrecheckWarning{
			Code:    w.Code,
			Message: w.Message,
		})
	}

	// Convert suggestions
	for _, sug := range result.Suggestions {
		if slot, ok := sug.Value.(aischedule.AlternativeSlot); ok {
			response.Suggestions = append(response.Suggestions, &v1pb.PrecheckSuggestion{
				Type: sug.Type,
				Slot: &v1pb.AlternativeSlot{
					StartTs: slot.StartTime.Unix(),
					EndTs:   slot.EndTime.Unix(),
					Label:   slot.Label,
				},
			})
		}
	}

	return response, nil
}

// BatchParseSchedule parses natural language for batch schedule creation.
func (s *ScheduleService) BatchParseSchedule(ctx context.Context, req *v1pb.BatchParseScheduleRequest) (*v1pb.BatchParseScheduleResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate input
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, status.Errorf(codes.InvalidArgument, "text is required")
	}

	// Create batch schedule service
	batchService := aischedule.NewBatchScheduleService(nil)

	// Parse the input
	result, err := batchService.ParseAndPreview(ctx, text)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse batch schedule: %v", err)
	}

	// Build response
	response := &v1pb.BatchParseScheduleResponse{
		CanBatchCreate: result.CanBatchCreate,
		TotalCount:     int32(result.TotalCount),
		MissingFields:  result.MissingFields,
		Confidence:     result.Confidence,
	}

	// Convert batch info
	if result.Request != nil {
		recurrenceJSON := ""
		if result.Request.Recurrence != nil {
			recurrenceJSON, _ = result.Request.Recurrence.ToJSON()
		}

		response.Info = &v1pb.BatchScheduleInfo{
			Title:          result.Request.Title,
			StartTs:        result.Request.StartTime.Unix(),
			Duration:       int32(result.Request.Duration),
			Location:       result.Request.Location,
			RecurrenceRule: recurrenceJSON,
			Count:          int32(result.Request.Count),
		}
		if result.Request.EndDate != nil {
			response.Info.EndDateTs = result.Request.EndDate.Unix()
		}
	}

	// Convert preview schedules
	previewCount := req.PreviewCount
	if previewCount <= 0 {
		previewCount = 5 // Default preview count
	}

	for i, sched := range result.Preview {
		if int32(i) >= previewCount {
			break
		}

		endTs := int64(0)
		if !sched.EndTime.IsZero() {
			endTs = sched.EndTime.Unix()
		}

		response.Preview = append(response.Preview, &v1pb.Schedule{
			Name:    fmt.Sprintf("schedules/preview-%d", i),
			Title:   sched.Title,
			StartTs: sched.StartTime.Unix(),
			EndTs:   endTs,
		})
	}

	return response, nil
}

// BatchCreateSchedules creates multiple schedules from a batch request.
func (s *ScheduleService) BatchCreateSchedules(ctx context.Context, req *v1pb.BatchCreateSchedulesRequest) (*v1pb.BatchCreateSchedulesResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate request
	if req.Info == nil {
		return nil, status.Errorf(codes.InvalidArgument, "info is required")
	}
	if req.Info.Title == "" {
		return nil, status.Errorf(codes.InvalidArgument, "title is required")
	}
	if req.Info.StartTs <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "start_ts is required")
	}
	if req.Info.RecurrenceRule == "" {
		return nil, status.Errorf(codes.InvalidArgument, "recurrence_rule is required")
	}

	// Parse recurrence rule
	recurrence, err := aischedule.ParseRecurrence(req.Info.RecurrenceRule)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid recurrence_rule: %v", err)
	}

	// Build batch request
	batchReq := &aischedule.BatchCreateRequest{
		Title:      req.Info.Title,
		StartTime:  time.Unix(req.Info.StartTs, 0),
		Duration:   int(req.Info.Duration),
		Location:   req.Info.Location,
		Recurrence: recurrence,
		Count:      int(req.Info.Count),
	}
	if batchReq.Duration == 0 {
		batchReq.Duration = 60 // Default 1 hour
	}
	if batchReq.Count == 0 {
		batchReq.Count = 12 // Default 12 instances
	}
	if req.Info.EndDateTs > 0 {
		endDate := time.Unix(req.Info.EndDateTs, 0)
		batchReq.EndDate = &endDate
	}

	// Generate schedules
	batchService := aischedule.NewBatchScheduleService(nil)
	schedules, err := batchService.GenerateSchedules(batchReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate schedules: %v", err)
	}

	response := &v1pb.BatchCreateSchedulesResponse{}

	// Create each schedule
	for _, sched := range schedules {
		// Build store schedule
		endTs := sched.EndTime.Unix()
		storeSchedule := &store.Schedule{
			UID:       util.GenUUID(),
			CreatorID: userID,
			Title:     sched.Title,
			StartTs:   sched.StartTime.Unix(),
			EndTs:     &endTs,
			Location:  sched.Location,
			Timezone:  "Asia/Shanghai",
		}

		// Set default reminders and payload
		remindersStr := "[]"
		storeSchedule.Reminders = &remindersStr
		payloadStr := "{}"
		storeSchedule.Payload = &payloadStr

		// Check conflicts unless skipped
		if !req.SkipConflictCheck {
			conflicts, err := s.checkScheduleConflicts(ctx, userID, storeSchedule.StartTs, storeSchedule.EndTs, nil)
			if err != nil {
				slog.Warn("failed to check conflicts for batch schedule",
					"title", sched.Title,
					"error", err)
			}
			if len(conflicts) > 0 {
				// Add to conflicts list and skip creation
				response.Conflicts = append(response.Conflicts, &v1pb.Schedule{
					Name:    fmt.Sprintf("schedules/%s", storeSchedule.UID),
					Title:   storeSchedule.Title,
					StartTs: storeSchedule.StartTs,
					EndTs:   *storeSchedule.EndTs,
				})
				continue
			}
		}

		// Create the schedule
		created, err := s.Store.CreateSchedule(ctx, storeSchedule)
		if err != nil {
			slog.Warn("failed to create batch schedule",
				"title", sched.Title,
				"error", err)
			continue
		}

		response.Schedules = append(response.Schedules, scheduleFromStore(created))
	}

	response.CreatedCount = int32(len(response.Schedules))

	return response, nil
}
