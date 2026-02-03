// Package schedule provides schedule-related AI agent utilities.
package schedule

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/aitime"
)

// BatchCreateRequest represents a batch schedule creation request.
type BatchCreateRequest struct {
	StartTime  time.Time       `json:"start_time"`
	Recurrence *RecurrenceRule `json:"recurrence"`
	EndDate    *time.Time      `json:"end_date,omitempty"`
	Title      string          `json:"title"`
	Location   string          `json:"location,omitempty"`
	Duration   int             `json:"duration"`
	Count      int             `json:"count,omitempty"`
}

// BatchCreateResult represents the result of batch schedule parsing.
type BatchCreateResult struct {
	Request        *BatchCreateRequest `json:"request,omitempty"`
	Preview        []*ScheduleRequest  `json:"preview,omitempty"`
	MissingFields  []string            `json:"missing_fields,omitempty"`
	TotalCount     int                 `json:"total_count"`
	Confidence     float64             `json:"confidence"`
	CanBatchCreate bool                `json:"can_batch_create"`
}

// BatchScheduleParser parses user input for batch schedule creation.
type BatchScheduleParser struct {
	timeService *aitime.Service
}

// NewBatchScheduleParser creates a new BatchScheduleParser.
func NewBatchScheduleParser(timeSvc *aitime.Service) *BatchScheduleParser {
	return &BatchScheduleParser{
		timeService: timeSvc,
	}
}

// recurrence patterns for natural language.
var recurrencePatterns = []struct {
	pattern *regexp.Regexp
	handler func(matches []string) *RecurrenceRule
}{
	// 每天/每日
	{
		regexp.MustCompile(`每天|每日`),
		func(_ []string) *RecurrenceRule {
			return &RecurrenceRule{Type: RecurrenceTypeDaily, Interval: 1}
		},
	},
	// 每个工作日
	{
		regexp.MustCompile(`每个?工作日`),
		func(_ []string) *RecurrenceRule {
			return &RecurrenceRule{Type: RecurrenceTypeWeekly, Interval: 1, Weekdays: []int{1, 2, 3, 4, 5}}
		},
	},
	// 每周X和Y (multiple days like 每周一三五) - must come before single day pattern
	{
		regexp.MustCompile(`每(周|星期)([一二三四五六日天]{2,})`),
		func(matches []string) *RecurrenceRule {
			if len(matches) < 3 {
				return nil
			}
			weekdays := parseMultipleWeekdays(matches[2])
			if len(weekdays) == 0 {
				return nil
			}
			return &RecurrenceRule{Type: RecurrenceTypeWeekly, Interval: 1, Weekdays: weekdays}
		},
	},
	// 每周X (single day)
	{
		regexp.MustCompile(`每(周|星期)([一二三四五六日天])`),
		func(matches []string) *RecurrenceRule {
			if len(matches) < 3 {
				return nil
			}
			weekday := parseWeekday(matches[2])
			if weekday == 0 {
				return nil
			}
			return &RecurrenceRule{Type: RecurrenceTypeWeekly, Interval: 1, Weekdays: []int{weekday}}
		},
	},
	// 每N周
	{
		regexp.MustCompile(`每(\d+)周`),
		func(matches []string) *RecurrenceRule {
			if len(matches) < 2 {
				return nil
			}
			interval := parseInt(matches[1])
			if interval <= 0 {
				interval = 1
			}
			return &RecurrenceRule{Type: RecurrenceTypeWeekly, Interval: interval, Weekdays: []int{1, 2, 3, 4, 5}}
		},
	},
	// 每月X号
	{
		regexp.MustCompile(`每月(\d{1,2})[号日]?`),
		func(matches []string) *RecurrenceRule {
			if len(matches) < 2 {
				return nil
			}
			day := parseInt(matches[1])
			if day < 1 || day > 31 {
				return nil
			}
			return &RecurrenceRule{Type: RecurrenceTypeMonthly, Interval: 1, MonthDay: day}
		},
	},
}

// Parse parses user input for batch schedule creation.
func (p *BatchScheduleParser) Parse(ctx context.Context, input string) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		Request:       &BatchCreateRequest{},
		MissingFields: []string{},
	}

	// Step 1: Detect recurrence pattern
	recurrence := p.detectRecurrence(input)
	if recurrence == nil {
		result.CanBatchCreate = false
		result.MissingFields = append(result.MissingFields, "recurrence")
		return result, nil
	}
	result.Request.Recurrence = recurrence

	// Step 2: Extract time
	startTime, err := p.extractTime(ctx, input)
	if err != nil || startTime.IsZero() {
		result.CanBatchCreate = false
		result.MissingFields = append(result.MissingFields, "time")
		return result, nil //nolint:nilerr // Return partial result on extraction failure
	}
	result.Request.StartTime = startTime

	// Step 3: Extract title
	title := p.extractBatchTitle(input)
	if title == "" {
		result.CanBatchCreate = false
		result.MissingFields = append(result.MissingFields, "title")
		return result, nil
	}
	result.Request.Title = title

	// Step 4: Apply defaults
	p.applyBatchDefaults(result.Request)

	// Step 5: Generate preview
	preview := p.generatePreview(result.Request)
	result.Preview = preview
	result.TotalCount = len(preview)

	// Step 6: Calculate confidence
	result.Confidence = p.calculateBatchConfidence(result.Request, len(preview))
	result.CanBatchCreate = result.Confidence >= 0.7 && len(preview) > 0

	return result, nil
}

// detectRecurrence detects the recurrence pattern from input.
func (p *BatchScheduleParser) detectRecurrence(input string) *RecurrenceRule {
	for _, rp := range recurrencePatterns {
		matches := rp.pattern.FindStringSubmatch(input)
		if len(matches) > 0 {
			rule := rp.handler(matches)
			if rule != nil && rule.Validate() == nil {
				return rule
			}
		}
	}
	return nil
}

// extractTime extracts time from input using TimeService.
func (p *BatchScheduleParser) extractTime(ctx context.Context, input string) (time.Time, error) {
	if p.timeService == nil {
		// Fallback to simple time parsing
		return p.simpleTimeExtract(input)
	}

	tr, err := p.timeService.ParseNaturalTime(ctx, input, time.Now())
	if err != nil {
		return p.simpleTimeExtract(input)
	}

	return tr.Start, nil
}

// simpleTimeExtract extracts time using simple patterns.
func (p *BatchScheduleParser) simpleTimeExtract(input string) (time.Time, error) {
	now := time.Now()

	// Pattern: 下午X点 (must come first - more specific)
	pmPattern := regexp.MustCompile(`下午(\d{1,2})[点时](\d{1,2}分?)?`)
	if matches := pmPattern.FindStringSubmatch(input); len(matches) > 0 {
		hour := parseInt(matches[1])
		if hour < 12 {
			hour += 12
		}
		minute := 0
		if len(matches) > 2 && matches[2] != "" {
			minute = parseInt(strings.TrimSuffix(matches[2], "分"))
		}
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	// Pattern: 上午/早上X点
	amPattern := regexp.MustCompile(`(上午|早上)?(\d{1,2})[点时](\d{1,2}分?)?`)
	if matches := amPattern.FindStringSubmatch(input); len(matches) > 0 {
		hour := parseInt(matches[2])
		minute := 0
		if len(matches) > 3 && matches[3] != "" {
			minute = parseInt(strings.TrimSuffix(matches[3], "分"))
		}
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	return time.Time{}, fmt.Errorf("no time pattern found")
}

// Batch title removal patterns.
var batchTitleRemovalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`每天|每日|每个?工作日`),
	regexp.MustCompile(`每(周|星期)[一二三四五六日天]+`),
	regexp.MustCompile(`每月\d{1,2}[号日]?`),
	regexp.MustCompile(`每\d+周`),
	regexp.MustCompile(`(上午|下午|早上|中午|晚上)`),
	regexp.MustCompile(`\d{1,2}[点时](\d{1,2}分?)?`),
	regexp.MustCompile(`帮我|请|给我|安排|创建|添加|新建`),
}

// extractBatchTitle extracts title from batch schedule input.
func (p *BatchScheduleParser) extractBatchTitle(input string) string {
	cleaned := input
	for _, pattern := range batchTitleRemovalPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "，。、,. ")

	if len(cleaned) > 0 && len(cleaned) <= 50 {
		return cleaned
	}

	return ""
}

// applyBatchDefaults applies default values to the batch request.
func (p *BatchScheduleParser) applyBatchDefaults(req *BatchCreateRequest) {
	if req.Duration == 0 {
		req.Duration = 60 // Default 1 hour
	}

	if req.Count == 0 && req.EndDate == nil {
		req.Count = 12 // Default 12 instances (~3 months for weekly)
	}

	// Cap at reasonable limits
	if req.Count > 52 {
		req.Count = 52 // Max 1 year for weekly
	}
}

// generatePreview generates a preview of schedules to be created.
func (p *BatchScheduleParser) generatePreview(req *BatchCreateRequest) []*ScheduleRequest {
	var schedules []*ScheduleRequest

	// Align startTime to the first matching weekday for weekly recurrence
	startTime := req.StartTime
	if req.Recurrence.Type == RecurrenceTypeWeekly && len(req.Recurrence.Weekdays) > 0 {
		startTime = p.alignToFirstWeekday(startTime, req.Recurrence.Weekdays)
	}

	// Calculate end timestamp
	var endTs int64
	if req.EndDate != nil {
		endTs = req.EndDate.Unix()
	} else {
		// Default to 1 year from start
		endTs = startTime.AddDate(1, 0, 0).Unix()
	}

	// Generate instances using recurrence rule
	instances := req.Recurrence.GenerateInstances(startTime.Unix(), endTs)

	// Limit to requested count
	maxCount := req.Count
	if maxCount == 0 {
		maxCount = 52
	}

	for i, ts := range instances {
		if i >= maxCount {
			break
		}

		schedStartTime := time.Unix(ts, 0)
		endTime := schedStartTime.Add(time.Duration(req.Duration) * time.Minute)

		schedules = append(schedules, &ScheduleRequest{
			Title:     req.Title,
			StartTime: schedStartTime,
			EndTime:   endTime,
			Duration:  req.Duration,
			Location:  req.Location,
		})
	}

	return schedules
}

// calculateBatchConfidence calculates confidence for batch creation.
func (p *BatchScheduleParser) calculateBatchConfidence(req *BatchCreateRequest, previewCount int) float64 {
	score := 1.0

	// Recurrence validity
	if req.Recurrence == nil {
		score -= 0.4
	} else if req.Recurrence.Validate() != nil {
		score -= 0.3
	}

	// Title
	if req.Title == "" {
		score -= 0.3
	}

	// Time
	if req.StartTime.IsZero() {
		score -= 0.3
	}

	// Preview count
	if previewCount == 0 {
		score -= 0.3
	}

	if score < 0 {
		return 0
	}
	return score
}

// alignToFirstWeekday aligns time to the first matching weekday from the list.
// If current day is already in weekdays, returns the same time.
// Otherwise, finds the next matching weekday.
func (p *BatchScheduleParser) alignToFirstWeekday(t time.Time, weekdays []int) time.Time {
	if len(weekdays) == 0 {
		return t
	}

	// Convert Go weekday (0=Sunday) to ISO weekday (1=Monday, 7=Sunday)
	currentWeekday := int(t.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7
	}

	// Check if current day matches any target weekday
	for _, wd := range weekdays {
		if wd == currentWeekday {
			return t // Already on a valid weekday
		}
	}

	// Find the next matching weekday
	minDaysToAdd := 8 // Start with more than a week
	for _, targetWd := range weekdays {
		daysToAdd := targetWd - currentWeekday
		if daysToAdd <= 0 {
			daysToAdd += 7
		}
		if daysToAdd < minDaysToAdd {
			minDaysToAdd = daysToAdd
		}
	}

	return t.AddDate(0, 0, minDaysToAdd)
}

// parseWeekday parses a single Chinese weekday character to int (1=Monday, 7=Sunday).
func parseWeekday(s string) int {
	weekdayMap := map[string]int{
		"一": 1, "二": 2, "三": 3, "四": 4, "五": 5,
		"六": 6, "日": 7, "天": 7,
	}
	return weekdayMap[s]
}

// parseMultipleWeekdays parses multiple Chinese weekday characters.
func parseMultipleWeekdays(s string) []int {
	var weekdays []int
	seen := make(map[int]bool)

	for _, r := range s {
		day := parseWeekday(string(r))
		if day > 0 && !seen[day] {
			weekdays = append(weekdays, day)
			seen[day] = true
		}
	}

	return weekdays
}

// BatchScheduleService provides batch schedule operations.
type BatchScheduleService struct {
	parser *BatchScheduleParser
}

// NewBatchScheduleService creates a new BatchScheduleService.
func NewBatchScheduleService(timeSvc *aitime.Service) *BatchScheduleService {
	return &BatchScheduleService{
		parser: NewBatchScheduleParser(timeSvc),
	}
}

// ParseAndPreview parses input and returns a preview of schedules.
func (s *BatchScheduleService) ParseAndPreview(ctx context.Context, input string) (*BatchCreateResult, error) {
	return s.parser.Parse(ctx, input)
}

// GenerateSchedules generates the actual schedule objects from a validated request.
func (s *BatchScheduleService) GenerateSchedules(req *BatchCreateRequest) ([]*ScheduleRequest, error) {
	if req == nil || req.Recurrence == nil {
		return nil, fmt.Errorf("invalid batch create request")
	}

	if err := req.Recurrence.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recurrence rule: %w", err)
	}

	return s.parser.generatePreview(req), nil
}

// ValidateRequest validates a batch create request.
func (s *BatchScheduleService) ValidateRequest(req *BatchCreateRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.StartTime.IsZero() {
		return fmt.Errorf("start time is required")
	}

	if req.Recurrence == nil {
		return fmt.Errorf("recurrence rule is required")
	}

	return req.Recurrence.Validate()
}
