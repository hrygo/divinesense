// Package schedule provides schedule-related AI agent utilities.
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/store"
)

// PrecheckError codes for schedule validation errors.
const (
	ErrCodeMissingStartTime = "MISSING_START_TIME"
	ErrCodePastTime         = "PAST_TIME"
	ErrCodeTimeTooFar       = "TIME_TOO_FAR"
	ErrCodeEndBeforeStart   = "END_BEFORE_START"
	ErrCodeTimeConflict     = "TIME_CONFLICT"
)

// PrecheckWarning codes for schedule validation warnings.
const (
	WarnCodeBufferConflict      = "BUFFER_CONFLICT"
	WarnCodeLongDuration        = "LONG_DURATION"
	WarnCodeOutsideWorkHours    = "OUTSIDE_WORK_HOURS"
	WarnCodeLongTitle           = "LONG_TITLE"
	WarnCodeWeekendSchedule     = "WEEKEND_SCHEDULE"
	WarnCodeConflictCheckFailed = "CONFLICT_CHECK_FAILED"
)

// Business rule constants.
const (
	BufferMinutes      = 15  // Minimum gap between schedules (minutes)
	MaxDurationMinutes = 480 // Maximum schedule duration (8 hours)
	WorkStartHour      = 8   // Work hours start
	WorkEndHour        = 22  // Work hours end
	MaxTitleLength     = 100 // Maximum title length
	MaxFutureYears     = 1   // Maximum years in advance for scheduling
)

// PrecheckRequest represents a schedule precheck request.
type PrecheckRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Title     string    `json:"title"`
	Location  string    `json:"location,omitempty"`
	Duration  int       `json:"duration"`
}

// PrecheckResponse represents a schedule precheck result.
type PrecheckResponse struct {
	Errors      []PrecheckError      `json:"errors,omitempty"`
	Warnings    []PrecheckWarning    `json:"warnings,omitempty"`
	Suggestions []PrecheckSuggestion `json:"suggestions,omitempty"`
	Valid       bool                 `json:"valid"`
}

// PrecheckError represents a validation error.
type PrecheckError struct {
	Code    string `json:"code"`            // Error code like "TIME_CONFLICT"
	Message string `json:"message"`         // Human-readable message
	Field   string `json:"field,omitempty"` // Field that caused the error
}

// PrecheckWarning represents a validation warning.
type PrecheckWarning struct {
	Code    string `json:"code"`    // Warning code like "OUTSIDE_WORK_HOURS"
	Message string `json:"message"` // Human-readable message
}

// PrecheckSuggestion represents a suggestion for schedule correction.
type PrecheckSuggestion struct {
	Value any    `json:"value"`
	Type  string `json:"type"`
}

// AlternativeSlot represents an available time slot.
type AlternativeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Label     string    `json:"label"` // "同日稍后", "明天同一时间"
}

// ScheduleStore defines the interface for schedule storage operations.
type ScheduleStore interface {
	ListSchedules(ctx context.Context, find *store.FindSchedule) ([]*store.Schedule, error)
}

// PrecheckService provides schedule precheck functionality.
type PrecheckService struct {
	store ScheduleStore
}

// NewPrecheckService creates a new PrecheckService.
func NewPrecheckService(scheduleStore ScheduleStore) *PrecheckService {
	return &PrecheckService{
		store: scheduleStore,
	}
}

// Precheck validates a schedule creation request before actual creation.
func (s *PrecheckService) Precheck(ctx context.Context, userID int32, req *PrecheckRequest) *PrecheckResponse {
	response := &PrecheckResponse{Valid: true}

	// Normalize request - calculate EndTime from Duration if not set
	normalizedReq := s.normalizeRequest(req)

	// Step 1: Validate time format
	s.validateTimeFormat(normalizedReq, response)

	// Step 2: Detect conflicts (only if time validation passed)
	if response.Valid {
		s.detectConflicts(ctx, userID, normalizedReq, response)
	}

	// Step 3: Validate business rules
	s.validateBusinessRules(normalizedReq, response)

	// Step 4: Generate suggestions if validation failed
	if !response.Valid {
		s.generateSuggestions(ctx, userID, normalizedReq, response)
	}

	return response
}

// normalizeRequest fills in missing fields and normalizes the request.
func (s *PrecheckService) normalizeRequest(req *PrecheckRequest) *PrecheckRequest {
	normalized := &PrecheckRequest{
		Title:     req.Title,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Duration:  req.Duration,
		Location:  req.Location,
	}

	// If EndTime is not set but Duration is provided, calculate EndTime
	if normalized.EndTime.IsZero() && normalized.Duration > 0 && !normalized.StartTime.IsZero() {
		normalized.EndTime = normalized.StartTime.Add(time.Duration(normalized.Duration) * time.Minute)
	}

	// If Duration is not set but EndTime is provided, calculate Duration
	if normalized.Duration == 0 && !normalized.EndTime.IsZero() && !normalized.StartTime.IsZero() {
		normalized.Duration = int(normalized.EndTime.Sub(normalized.StartTime).Minutes())
	}

	// Default duration if nothing is provided
	if normalized.Duration == 0 {
		normalized.Duration = 60 // Default 1 hour
	}

	// Calculate EndTime if still not set
	if normalized.EndTime.IsZero() && !normalized.StartTime.IsZero() {
		normalized.EndTime = normalized.StartTime.Add(time.Duration(normalized.Duration) * time.Minute)
	}

	return normalized
}

// validateTimeFormat validates the time format and range.
func (s *PrecheckService) validateTimeFormat(req *PrecheckRequest, resp *PrecheckResponse) {
	now := time.Now()

	// Check start time is not empty
	if req.StartTime.IsZero() {
		resp.Valid = false
		resp.Errors = append(resp.Errors, PrecheckError{
			Code:    ErrCodeMissingStartTime,
			Message: "请选择开始时间",
			Field:   "start_time",
		})
		return
	}

	// Check start time is not in the past (allow 5 minutes tolerance)
	if req.StartTime.Before(now.Add(-5 * time.Minute)) {
		resp.Valid = false
		resp.Errors = append(resp.Errors, PrecheckError{
			Code:    ErrCodePastTime,
			Message: "开始时间不能是过去",
			Field:   "start_time",
		})
	}

	// Check start time is within reasonable range (1 year)
	maxDate := now.AddDate(MaxFutureYears, 0, 0)
	if req.StartTime.After(maxDate) {
		resp.Valid = false
		resp.Errors = append(resp.Errors, PrecheckError{
			Code:    ErrCodeTimeTooFar,
			Message: "开始时间不能超过一年",
			Field:   "start_time",
		})
	}

	// Check end time is after start time
	if !req.EndTime.IsZero() && req.EndTime.Before(req.StartTime) {
		resp.Valid = false
		resp.Errors = append(resp.Errors, PrecheckError{
			Code:    ErrCodeEndBeforeStart,
			Message: "结束时间不能早于开始时间",
			Field:   "end_time",
		})
	}
}

// detectConflicts checks for scheduling conflicts with existing schedules.
func (s *PrecheckService) detectConflicts(ctx context.Context, userID int32, req *PrecheckRequest, resp *PrecheckResponse) {
	// Calculate check range with buffer
	checkStart := req.StartTime.Add(-time.Duration(BufferMinutes) * time.Minute)
	checkEnd := req.EndTime.Add(time.Duration(BufferMinutes) * time.Minute)

	// Query schedules in the time range
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   pointerTo(checkStart.Unix()),
		EndTs:     pointerTo(checkEnd.Unix()),
	}

	existingSchedules, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		slog.Warn("failed to check schedule conflicts",
			"user_id", userID,
			"error", err)
		resp.Warnings = append(resp.Warnings, PrecheckWarning{
			Code:    WarnCodeConflictCheckFailed,
			Message: "无法检查时间冲突，请自行确认",
		})
		return
	}

	reqStartTs := req.StartTime.Unix()
	reqEndTs := req.EndTime.Unix()

	for _, existing := range existingSchedules {
		existingEndTs := existing.StartTs
		if existing.EndTs != nil {
			existingEndTs = *existing.EndTs
		}

		// Check for direct overlap
		if s.hasOverlap(reqStartTs, reqEndTs, existing.StartTs, existingEndTs) {
			resp.Valid = false
			resp.Errors = append(resp.Errors, PrecheckError{
				Code:    ErrCodeTimeConflict,
				Message: fmt.Sprintf("与已有日程「%s」冲突", existing.Title),
				Field:   "start_time",
			})
		} else if s.hasBufferConflict(reqStartTs, reqEndTs, existing.StartTs, existingEndTs) {
			// Check for buffer conflict (warning only)
			resp.Warnings = append(resp.Warnings, PrecheckWarning{
				Code:    WarnCodeBufferConflict,
				Message: fmt.Sprintf("与「%s」间隔较短（少于%d分钟）", existing.Title, BufferMinutes),
			})
		}
	}
}

// hasOverlap checks if two time ranges overlap.
// Uses [start, end) convention (left-closed, right-open).
func (s *PrecheckService) hasOverlap(start1, end1, start2, end2 int64) bool {
	return start1 < end2 && end1 > start2
}

// hasBufferConflict checks if two time ranges have buffer conflict (too close).
func (s *PrecheckService) hasBufferConflict(start1, end1, start2, end2 int64) bool {
	bufferSec := int64(BufferMinutes * 60)
	// Check if ranges are within buffer distance but not overlapping
	return !s.hasOverlap(start1, end1, start2, end2) &&
		(start1 < end2+bufferSec && end1+bufferSec > start2)
}

// validateBusinessRules validates business rules (warnings only, don't block).
func (s *PrecheckService) validateBusinessRules(req *PrecheckRequest, resp *PrecheckResponse) {
	// Check duration
	if req.Duration > MaxDurationMinutes {
		resp.Warnings = append(resp.Warnings, PrecheckWarning{
			Code:    WarnCodeLongDuration,
			Message: fmt.Sprintf("日程时长超过 %d 小时，请确认", MaxDurationMinutes/60),
		})
	}

	// Check work hours
	hour := req.StartTime.Hour()
	if hour < WorkStartHour || hour >= WorkEndHour {
		resp.Warnings = append(resp.Warnings, PrecheckWarning{
			Code:    WarnCodeOutsideWorkHours,
			Message: "该时间在常规工作时间外",
		})
	}

	// Check title length
	if len(req.Title) > MaxTitleLength {
		resp.Warnings = append(resp.Warnings, PrecheckWarning{
			Code:    WarnCodeLongTitle,
			Message: "标题较长，建议精简",
		})
	}

	// Check weekend
	weekday := req.StartTime.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		resp.Warnings = append(resp.Warnings, PrecheckWarning{
			Code:    WarnCodeWeekendSchedule,
			Message: "该日程安排在周末",
		})
	}
}

// generateSuggestions generates alternative time slot suggestions.
func (s *PrecheckService) generateSuggestions(ctx context.Context, userID int32, req *PrecheckRequest, resp *PrecheckResponse) {
	// Only generate suggestions for time conflicts
	hasTimeConflict := false
	for _, err := range resp.Errors {
		if err.Code == ErrCodeTimeConflict {
			hasTimeConflict = true
			break
		}
	}

	if !hasTimeConflict {
		return
	}

	alternatives := s.findAlternativeSlots(ctx, userID, req)
	for _, alt := range alternatives {
		resp.Suggestions = append(resp.Suggestions, PrecheckSuggestion{
			Type:  "alternative_time",
			Value: alt,
		})
	}
}

// findAlternativeSlots finds available time slots as alternatives.
func (s *PrecheckService) findAlternativeSlots(ctx context.Context, userID int32, req *PrecheckRequest) []AlternativeSlot {
	var alternatives []AlternativeSlot
	duration := req.EndTime.Sub(req.StartTime)

	// Strategy 1: Same day, 2 hours later
	sameDay := req.StartTime.Add(2 * time.Hour)
	if s.isSlotAvailable(ctx, userID, sameDay, sameDay.Add(duration)) && sameDay.Hour() < 22 {
		alternatives = append(alternatives, AlternativeSlot{
			StartTime: sameDay,
			EndTime:   sameDay.Add(duration),
			Label:     "同日稍后",
		})
	}

	// Strategy 2: Next day same time
	nextDay := req.StartTime.AddDate(0, 0, 1)
	if s.isSlotAvailable(ctx, userID, nextDay, nextDay.Add(duration)) {
		alternatives = append(alternatives, AlternativeSlot{
			StartTime: nextDay,
			EndTime:   nextDay.Add(duration),
			Label:     "明天同一时间",
		})
	}

	// Strategy 3: Same day morning (if original was afternoon)
	if req.StartTime.Hour() >= 14 {
		morning := time.Date(
			req.StartTime.Year(), req.StartTime.Month(), req.StartTime.Day(),
			9, 0, 0, 0, req.StartTime.Location(),
		)
		if s.isSlotAvailable(ctx, userID, morning, morning.Add(duration)) && morning.After(time.Now()) {
			alternatives = append(alternatives, AlternativeSlot{
				StartTime: morning,
				EndTime:   morning.Add(duration),
				Label:     "今日上午",
			})
		}
	}

	return alternatives
}

// isSlotAvailable checks if a time slot is available (no conflicts).
func (s *PrecheckService) isSlotAvailable(ctx context.Context, userID int32, start, end time.Time) bool {
	find := &store.FindSchedule{
		CreatorID: &userID,
		StartTs:   pointerTo(start.Unix()),
		EndTs:     pointerTo(end.Unix()),
	}

	schedules, err := s.store.ListSchedules(ctx, find)
	if err != nil {
		return false
	}

	startTs := start.Unix()
	endTs := end.Unix()

	for _, schedule := range schedules {
		existingEndTs := schedule.StartTs
		if schedule.EndTs != nil {
			existingEndTs = *schedule.EndTs
		}
		if s.hasOverlap(startTs, endTs, schedule.StartTs, existingEndTs) {
			return false
		}
	}

	return true
}

// pointerTo returns a pointer to the given value.
func pointerTo[T any](v T) *T {
	return &v
}
