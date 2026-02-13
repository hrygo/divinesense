package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	schedsvc "github.com/hrygo/divinesense/server/service/schedule"
)

// Tool is the interface for schedule tools.
type Tool interface {
	Name() string
	Description() string
	InputType() map[string]interface{}
	Run(ctx context.Context, inputJSON string) (string, error)
}

// ScheduleQueryTool queries existing schedules.
type ScheduleQueryTool struct {
	service      schedsvc.Service
	userIDGetter func(ctx context.Context) int32
}

// NewScheduleQueryTool creates a new query tool.
func NewScheduleQueryTool(service schedsvc.Service, userIDGetter func(ctx context.Context) int32) *ScheduleQueryTool {
	return &ScheduleQueryTool{service: service, userIDGetter: userIDGetter}
}

func (t *ScheduleQueryTool) Name() string { return "schedule_query" }

func (t *ScheduleQueryTool) Description() string {
	return `Query existing schedules in a time range.

USAGE: Call BEFORE schedule_add to check conflicts.

Input: {"start_time": "ISO8601", "end_time": "ISO8601"}
Example: {"start_time": "2026-01-25T00:00:00+08:00", "end_time": "2026-01-26T00:00:00+08:00"}`
}

func (t *ScheduleQueryTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"start_time": map[string]interface{}{"type": "string", "description": "ISO8601"},
			"end_time":   map[string]interface{}{"type": "string", "description": "ISO8601"},
		},
		"required": []string{"start_time", "end_time"},
	}
}

func (t *ScheduleQueryTool) Run(ctx context.Context, inputJSON string) (string, error) {
	raw, err := parseScheduleInput(inputJSON)
	if err != nil {
		return "", err
	}

	startStr := getString(raw, "start_time")
	endStr := getString(raw, "end_time")
	if startStr == "" || endStr == "" {
		return "", fmt.Errorf("start_time and end_time are required")
	}

	startTime, err := parseISO8601(startStr)
	if err != nil {
		return "", fmt.Errorf("invalid start_time: %w", err)
	}
	endTime, err := parseISO8601(endStr)
	if err != nil {
		return "", fmt.Errorf("invalid end_time: %w", err)
	}
	if endTime.Before(startTime) {
		return "", fmt.Errorf("end_time must be after start_time")
	}

	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized")
	}

	schedules, err := t.service.FindSchedules(ctx, userID, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	if len(schedules) == 0 {
		return "No schedules found.", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d schedule(s):\n", len(schedules))
	for i, s := range schedules {
		fmt.Fprintf(&b, "%d. %s (%s", i+1, s.Title, formatTime(s.StartTs, s.Timezone))
		if s.EndTs != nil {
			fmt.Fprintf(&b, " - %s", formatTime(*s.EndTs, s.Timezone))
		}
		b.WriteString(")\n")
	}
	return b.String(), nil
}

// ScheduleAddTool creates a new schedule.
type ScheduleAddTool struct {
	service      schedsvc.Service
	userIDGetter func(ctx context.Context) int32
}

// NewScheduleAddTool creates a new add tool.
func NewScheduleAddTool(service schedsvc.Service, userIDGetter func(ctx context.Context) int32) *ScheduleAddTool {
	return &ScheduleAddTool{service: service, userIDGetter: userIDGetter}
}

func (t *ScheduleAddTool) Name() string { return "schedule_add" }

func (t *ScheduleAddTool) Description() string {
	return `Create a schedule event.

CRITICAL RULES:
1. start_time MUST be in the future (not in the past)
2. Use 08:00-22:00 for normal schedules (avoid 22:00-06:00 night hours)
3. Call schedule_query FIRST to check for conflicts

Input: {"title": "...", "start_time": "ISO8601", "end_time": "ISO8601"}
Example: {"title": "Meeting", "start_time": "2026-02-10T15:00:00+08:00", "end_time": "2026-02-10T16:00:00+08:00"}`
}

func (t *ScheduleAddTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":       map[string]interface{}{"type": "string", "description": "Event title"},
			"start_time":  map[string]interface{}{"type": "string", "description": "ISO8601"},
			"end_time":    map[string]interface{}{"type": "string", "description": "ISO8601 (optional)"},
			"description": map[string]interface{}{"type": "string", "description": "Optional"},
			"location":    map[string]interface{}{"type": "string", "description": "Optional"},
			"all_day":     map[string]interface{}{"type": "boolean", "description": "Default false"},
		},
		"required": []string{"title", "start_time"},
	}
}

func (t *ScheduleAddTool) Run(ctx context.Context, inputJSON string) (string, error) {
	raw, err := parseScheduleInput(inputJSON)
	if err != nil {
		return "", err
	}

	title := getString(raw, "title")
	startStr := getString(raw, "start_time")
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	if startStr == "" {
		return "", fmt.Errorf("start_time is required")
	}

	startTime, err := parseISO8601(startStr)
	if err != nil {
		return "", fmt.Errorf("invalid start_time: %w", err)
	}

	// Parse end time or default to 1 hour
	var endTs *int64
	if endStr := getString(raw, "end_time"); endStr != "" {
		endTime, err := parseISO8601(endStr)
		if err != nil {
			return "", fmt.Errorf("invalid end_time: %w", err)
		}
		ts := endTime.Unix()
		endTs = &ts
	} else {
		ts := startTime.Unix() + 3600 // Default 1 hour
		endTs = &ts
	}

	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized")
	}

	req := &schedsvc.CreateScheduleRequest{
		Title:       title,
		Description: getString(raw, "description"),
		Location:    getString(raw, "location"),
		StartTs:     startTime.Unix(),
		EndTs:       endTs,
		AllDay:      getBool(raw, "all_day"),
		Timezone:    DefaultTimezone,
	}

	created, err := t.service.CreateSchedule(ctx, userID, req)
	if err != nil {
		return "", fmt.Errorf("failed to create: %w", err)
	}

	result := fmt.Sprintf("Created: %s (%s", created.Title, formatTime(created.StartTs, created.Timezone))
	if created.EndTs != nil {
		result += fmt.Sprintf(" - %s", formatTime(*created.EndTs, created.Timezone))
	}
	result += ")"
	if created.Location != "" {
		result += fmt.Sprintf(" @ %s", created.Location)
	}
	return result, nil
}

// FindFreeTimeTool finds available time slots.
type FindFreeTimeTool struct {
	service      schedsvc.Service
	userIDGetter func(ctx context.Context) int32
	resolver     *schedsvc.ConflictResolver
}

// NewFindFreeTimeTool creates a new find free time tool.
func NewFindFreeTimeTool(service schedsvc.Service, userIDGetter func(ctx context.Context) int32) *FindFreeTimeTool {
	return &FindFreeTimeTool{
		service:      service,
		userIDGetter: userIDGetter,
		resolver:     schedsvc.NewConflictResolver(service),
	}
}

func (t *FindFreeTimeTool) Name() string { return "find_free_time" }

func (t *FindFreeTimeTool) Description() string {
	return `Find available 1-hour time slots.

WHEN TO USE:
- User asks "when am I free"
- User doesn't specify a time for scheduling

Input: {"date": "YYYY-MM-DD"}
Output: List of available time slots in ISO8601 format`
}

func (t *FindFreeTimeTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"date": map[string]interface{}{"type": "string", "format": "date", "example": "2026-01-22"},
		},
		"required": []string{"date"},
	}
}

func (t *FindFreeTimeTool) Run(ctx context.Context, inputJSON string) (string, error) {
	var input struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	if input.Date == "" {
		return "", fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %w", err)
	}

	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized")
	}

	loc := getTimezoneLocation(DefaultTimezone)
	slots, err := t.resolver.FindAllFreeSlots(ctx, userID, date.In(loc), time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to find slots: %w", err)
	}

	if len(slots) == 0 {
		return "No available time slots found.", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Available slots for %s:\n", input.Date)
	for i, slot := range slots {
		if i >= 10 {
			break // Limit to 10 slots
		}
		fmt.Fprintf(&b, "- %s to %s\n",
			slot.Start.In(loc).Format("15:04"),
			slot.End.In(loc).Format("15:04"))
	}
	return b.String(), nil
}

// ScheduleUpdateTool updates an existing schedule.
type ScheduleUpdateTool struct {
	service      schedsvc.Service
	userIDGetter func(ctx context.Context) int32
}

// NewScheduleUpdateTool creates a new update tool.
func NewScheduleUpdateTool(service schedsvc.Service, userIDGetter func(ctx context.Context) int32) *ScheduleUpdateTool {
	return &ScheduleUpdateTool{service: service, userIDGetter: userIDGetter}
}

func (t *ScheduleUpdateTool) Name() string { return "schedule_update" }

func (t *ScheduleUpdateTool) Description() string {
	return `Update an existing schedule.

Input: {"id": 123, "title": "...", "start_time": "ISO8601", "end_time": "ISO8601"}
All fields except id are optional - only provided fields will be updated.`
}

func (t *ScheduleUpdateTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":         map[string]interface{}{"type": "integer", "description": "Schedule ID"},
			"title":      map[string]interface{}{"type": "string", "description": "New title (optional)"},
			"start_time": map[string]interface{}{"type": "string", "description": "New start time (optional)"},
			"end_time":   map[string]interface{}{"type": "string", "description": "New end time (optional)"},
			"location":   map[string]interface{}{"type": "string", "description": "New location (optional)"},
		},
		"required": []string{"id"},
	}
}

func (t *ScheduleUpdateTool) Run(ctx context.Context, inputJSON string) (string, error) {
	raw, err := parseScheduleInput(inputJSON)
	if err != nil {
		return "", err
	}

	// Get schedule ID
	var scheduleID int32
	if v, ok := raw["id"]; ok {
		switch id := v.(type) {
		case float64:
			scheduleID = int32(id)
		case int32:
			scheduleID = id
		case int:
			scheduleID = int32(id)
		}
	}
	if scheduleID == 0 {
		return "", fmt.Errorf("id is required")
	}

	userID := t.userIDGetter(ctx)
	if userID == 0 {
		return "", fmt.Errorf("unauthorized")
	}

	update := &schedsvc.UpdateScheduleRequest{}

	if title := getString(raw, "title"); title != "" {
		update.Title = &title
	}
	if desc := getString(raw, "description"); desc != "" {
		update.Description = &desc
	}
	if loc := getString(raw, "location"); loc != "" {
		update.Location = &loc
	}
	if startStr := getString(raw, "start_time"); startStr != "" {
		startTime, err := parseISO8601(startStr)
		if err != nil {
			return "", fmt.Errorf("invalid start_time: %w", err)
		}
		ts := startTime.Unix()
		update.StartTs = &ts
	}
	if endStr := getString(raw, "end_time"); endStr != "" {
		endTime, err := parseISO8601(endStr)
		if err != nil {
			return "", fmt.Errorf("invalid end_time: %w", err)
		}
		ts := endTime.Unix()
		update.EndTs = &ts
	}

	updated, err := t.service.UpdateSchedule(ctx, userID, scheduleID, update)
	if err != nil {
		return "", fmt.Errorf("failed to update: %w", err)
	}

	return fmt.Sprintf("Updated: %s", updated.Title), nil
}
