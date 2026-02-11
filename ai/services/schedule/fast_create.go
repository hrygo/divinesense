// Package schedule provides schedule-related AI agent utilities.
package schedule

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/aitime"
	"github.com/hrygo/divinesense/ai/habit"
)

// ErrTimeServiceNotConfigured is returned when time service is not available.
var ErrTimeServiceNotConfigured = errors.New("time service not configured")

// ScheduleRequest represents a schedule creation request.
type ScheduleRequest struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Title           string    `json:"title"`
	Location        string    `json:"location,omitempty"`
	Description     string    `json:"description,omitempty"`
	Duration        int       `json:"duration"`
	ReminderMinutes int       `json:"reminder_minutes,omitempty"`
}

// FastCreateResult represents the result of fast create parsing.
type FastCreateResult struct {
	Schedule      *ScheduleRequest
	MissingFields []string
	Confidence    float64
	CanFastCreate bool
}

// FastCreateParser parses user input for fast schedule creation.
type FastCreateParser struct {
	timeService  *aitime.Service
	habitApplier *habit.HabitApplier
	classifier   *ScheduleIntentClassifier
}

// NewFastCreateParser creates a new FastCreateParser.
func NewFastCreateParser(timeSvc *aitime.Service, habitApplier *habit.HabitApplier) *FastCreateParser {
	return &FastCreateParser{
		timeService:  timeSvc,
		habitApplier: habitApplier,
		classifier:   NewScheduleIntentClassifier(nil), // Rule-based only for fast create
	}
}

// Parse parses user input and returns a FastCreateResult.
func (p *FastCreateParser) Parse(ctx context.Context, userID int32, input string) (*FastCreateResult, error) {
	result := &FastCreateResult{
		Schedule:      &ScheduleRequest{},
		MissingFields: []string{},
	}

	// Step 1: Intent identification (rule-based only for speed)
	classifyResult := p.classifier.Classify(ctx, input)
	if classifyResult.Intent != IntentSimpleCreate {
		result.CanFastCreate = false
		result.MissingFields = append(result.MissingFields, "intent_unclear")
		return result, nil
	}

	// Step 2: Time extraction
	parsedTime, err := p.extractTime(ctx, input)
	if err != nil || parsedTime.IsZero() {
		result.CanFastCreate = false
		result.MissingFields = append(result.MissingFields, "time")
		return result, nil //nolint:nilerr // Return partial result on extraction failure
	}
	result.Schedule.StartTime = parsedTime

	// Step 3: Title extraction
	title := extractTitle(input)
	if title == "" {
		result.CanFastCreate = false
		result.MissingFields = append(result.MissingFields, "title")
		return result, nil
	}
	result.Schedule.Title = title

	// Step 4: Apply defaults (user habits + system defaults)
	p.applyDefaults(ctx, userID, result.Schedule)

	// Step 5: Calculate confidence
	result.Confidence = calculateConfidence(result.Schedule)
	result.CanFastCreate = result.Confidence >= 0.8

	return result, nil
}

// extractTime extracts time from user input using TimeService.
func (p *FastCreateParser) extractTime(ctx context.Context, input string) (time.Time, error) {
	if p.timeService == nil {
		return time.Time{}, ErrTimeServiceNotConfigured
	}

	// Use ParseNaturalTime to get a time range
	tr, err := p.timeService.ParseNaturalTime(ctx, input, time.Now())
	if err != nil {
		return time.Time{}, err
	}

	return tr.Start, nil
}

// applyDefaults fills in default values based on user habits and system defaults.
func (p *FastCreateParser) applyDefaults(ctx context.Context, userID int32, schedule *ScheduleRequest) {
	// Apply user habits if available
	if p.habitApplier != nil {
		habitInput := &habit.ScheduleInput{
			Title:     schedule.Title,
			StartTime: schedule.StartTime,
			Duration:  schedule.Duration,
			Location:  schedule.Location,
		}
		enhanced := p.habitApplier.ApplyToScheduleCreate(ctx, userID, habitInput)

		// Apply suggested duration if not set
		if schedule.Duration == 0 && enhanced.SuggestedDuration > 0 {
			schedule.Duration = enhanced.SuggestedDuration
		}

		// Apply first suggested location if not set
		if schedule.Location == "" && len(enhanced.SuggestedLocations) > 0 {
			schedule.Location = enhanced.SuggestedLocations[0]
		}
	}

	// System defaults
	if schedule.Duration == 0 {
		schedule.Duration = 60 // Default 1 hour
	}

	if schedule.ReminderMinutes == 0 {
		schedule.ReminderMinutes = 15 // Default 15 minutes before
	}

	// Calculate end time
	if schedule.EndTime.IsZero() && !schedule.StartTime.IsZero() {
		schedule.EndTime = schedule.StartTime.Add(time.Duration(schedule.Duration) * time.Minute)
	}
}

// Time patterns for removal during title extraction.
var timeRemovalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`今天|明天|后天|大后天`),
	regexp.MustCompile(`周[一二三四五六日天]|下周[一二三四五六日天]`),
	regexp.MustCompile(`\d{1,2}月\d{1,2}[日号]`),
	regexp.MustCompile(`[上下]午`),
	regexp.MustCompile(`\d{1,2}[点时](\d{1,2}分)?`),
	regexp.MustCompile(`早上|中午|晚上|傍晚`),
	regexp.MustCompile(`帮我|请|给我`),
	regexp.MustCompile(`安排|创建|添加|新建`),
}

// Action word mappings for title generation.
// Keys are lowercase for case-insensitive matching.
var actionMappings = map[string]string{
	"开会":         "会议",
	"meeting":    "Meeting",
	"约":          "约会",
	"面试":         "面试",
	"汇报":         "工作汇报",
	"电话":         "电话会议",
	"讨论":         "讨论",
	"聊":          "交流",
	"见":          "会面",
	"培训":         "培训",
	"review":     "Review",
	"sync":       "Sync",
	"standup":    "Standup",
	"1on1":       "1:1 会议",
	"one on one": "1:1 会议",
}

// extractTitle extracts the title/action from user input.
func extractTitle(input string) string {
	// Remove time expressions
	cleaned := input
	for _, pattern := range timeRemovalPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}

	// Clean up whitespace and punctuation
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "，。、,. ")

	// Map common action words to titles
	lowerInput := strings.ToLower(input)
	for action, title := range actionMappings {
		if strings.Contains(lowerInput, strings.ToLower(action)) || strings.Contains(input, action) {
			return title
		}
	}

	// If cleaned content remains and is reasonable length, use it
	if len(cleaned) > 0 && len(cleaned) <= 50 {
		return cleaned
	}

	return ""
}

// calculateConfidence calculates confidence score for the parsed schedule.
func calculateConfidence(schedule *ScheduleRequest) float64 {
	score := 1.0

	// Time completeness
	if schedule.StartTime.IsZero() {
		score -= 0.4
	} else if schedule.StartTime.Before(time.Now()) {
		score -= 0.2
	}

	// Title completeness
	if schedule.Title == "" {
		score -= 0.4
	} else if len(schedule.Title) < 2 {
		score -= 0.1
	}

	// Duration reasonableness
	if schedule.Duration <= 0 || schedule.Duration > 480 {
		score -= 0.1
	}

	if score < 0 {
		return 0
	}
	return score
}
