package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"log/slog"

	"github.com/hrygo/divinesense/ai"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

const (
	// Validation constants.
	MaxInputLength = 500 // characters
)

// Parser handles natural language parsing for schedules.
type Parser struct {
	llmService ai.LLMService
	location   *time.Location
	validator  *TimezoneValidator // DST edge case validator
}

// NewParser creates a new schedule parser.
func NewParser(llmService ai.LLMService, timezone string) (*Parser, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Default to Asia/Shanghai if timezone is invalid
		slog.Warn("invalid timezone, falling back to Asia/Shanghai",
			"requested_timezone", timezone,
			"error", err)
		timezone = "Asia/Shanghai"
		loc, _ = time.LoadLocation("Asia/Shanghai") //nolint:errcheck // valid timezone
	}

	validator := NewTimezoneValidator(timezone)

	return &Parser{
		llmService: llmService,
		location:   loc,
		validator:  validator,
	}, nil
}

// ParseResult represents the parsed schedule information.
type ParseResult struct {
	Recurrence  *RecurrenceRule
	Title       string
	Description string
	Location    string
	Timezone    string
	Reminders   []*v1pb.Reminder
	StartTs     int64
	EndTs       int64
	AllDay      bool
}

// Parse parses natural language text and returns schedule information.
func (p *Parser) Parse(ctx context.Context, text string) (*ParseResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Validate input length
	if len(text) > MaxInputLength {
		return nil, fmt.Errorf("input too long: maximum %d characters, got %d", MaxInputLength, len(text))
	}

	// Use LLM parsing directly
	return p.parseWithLLM(ctx, text)
}

// llmScheduleResponse is the intermediate JSON structure for LLM output.
type llmScheduleResponse struct {
	Recurrence  *RecurrenceRule  `json:"recurrence"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Location    string           `json:"location"`
	StartTime   string           `json:"start_time"`
	EndTime     string           `json:"end_time"`
	Reminders   []*v1pb.Reminder `json:"reminders"`
	AllDay      bool             `json:"all_day"`
}

// parseWithLLM uses LLM to parse complex natural language.
func (p *Parser) parseWithLLM(ctx context.Context, text string) (*ParseResult, error) {
	now := time.Now().In(p.location)
	nowUTC := now.UTC()

	systemPrompt := fmt.Sprintf(`You are an intelligent schedule parser. Your goal is to extract schedule details from user input into a strict JSON format.

CURRENT TIME REFERENCE:
- Current UTC Time: %s
- Current Local Time: %s
- User Timezone: %s

====================================
CRITICAL TIMEZONE RULES (MUST FOLLOW)
====================================
1. ALWAYS return times in UTC timezone (start_time, end_time)
2. NEVER include timezone suffix in the time string (e.g., "UTC", "+08:00")
3. Format requirement: YYYY-MM-DD HH:mm:ss (24-hour format, pad zeros)
   Examples: "2026-01-21 09:00:00", "2026-01-21 14:30:00", "2026-01-21 00:00:00"

====================================
RELATIVE DATE CALCULATION (STEP-BY-STEP)
====================================
When user says relative dates (tomorrow, next week, etc.):

STEP 1: Determine the reference point (User's Local Time)
- Current local time is: %s
- Today's date is: %s

STEP 2: Calculate the target date in Local Time
Examples (assuming current time is 2026-01-21 Wednesday 09:00):
- "明天" or "tomorrow" -> 2026-01-22
- "后天" or "day after tomorrow" -> 2026-01-23
- "下周一" -> 2026-01-27 (next Monday)
- "本周五" -> 2026-01-24 (if not past, otherwise next week Friday)

STEP 3: Add the time component
- If user says "明天下午2点" -> use 14:00:00
- If user says "明天上午9点" -> use 09:00:00
- If user says "明天晚上7点" -> use 19:00:00
- Common time mappings:
  * 上午/早/morning -> 08:00:00 to 11:00:00
  * 中午/noon -> 12:00:00 to 13:00:00
  * 下午/afternoon -> 14:00:00 to 18:00:00
  * 晚上/evening -> 19:00:00 to 22:00:00
  * 如果没有具体时间，使用默认时间：09:00:00

STEP 4: Convert Local Time to UTC
- Example: Local "2026-01-22 14:00:00" in Asia/Shanghai (UTC+8)
- Calculation: 14:00:00 - 8 hours = 06:00:00 UTC
- Result: "2026-01-22 06:00:00"

====================================
ALL-DAY EVENT HANDLING
====================================
Only set all_day=true in these cases:
1. User explicitly says "全天" or "all day" (e.g., "明天全天开会")
2. User only mentions date without time (e.g., "明天开会" implies daytime, use 09:00:00)
3. If all_day=true, use "00:00:00" for start_time in user's timezone, then convert to UTC

DO NOT set all_day=true if:
- User mentions a specific time ("明天下午2点")
- User mentions a time range ("明天上午9点到11点")

====================================
TIME CALCULATION RULES
====================================
1. If duration is NOT specified: end_time = start_time + 1 hour (3600 seconds)
2. If duration is specified: calculate accordingly
   - "明天下午2点到4点" -> start=14:00, end=16:00
   - "明天上午9点到下午1点" -> start=09:00, end=13:00
3. Cross-day events:
   - "今晚11点到凌晨2点" -> start today 23:00, end tomorrow 02:00 (both in UTC)
4. Past time handling:
   - If user says "今天上午9点" and it's already 15:00, still use today 09:00
   - DO NOT automatically move to tomorrow unless user says "下一个"

====================================
OUTPUT SCHEMA (JSON ONLY)
====================================
Return ONLY valid JSON, no markdown, no code blocks:

{
  "title": "clean title without time/date keywords",
  "description": "detailed description or empty string",
  "location": "location name or empty string",
  "start_time": "YYYY-MM-DD HH:mm:ss",
  "end_time": "YYYY-MM-DD HH:mm:ss",
  "all_day": boolean,
  "reminders": [
    {"type": "before", "value": 10, "unit": "minutes"}
  ] or empty array [],
  "recurrence": {
    "type": "daily|weekly|monthly",
    "interval": 1,
    "weekdays": [1,2,3,4,5],
    "month_day": 15
  } or null
}

====================================
RECURRENCE PATTERN RULES
====================================
Extract recurrence patterns:

Daily (每天):
- "每天" -> {"type": "daily", "interval": 1}
- "每3天" -> {"type": "daily", "interval": 3}
- "每天一次" -> {"type": "daily", "interval": 1}

Weekly (每周):
- "每周" -> {"type": "weekly", "interval": 1, "weekdays": [1,2,3,4,5]}
- "每周一" -> {"type": "weekly", "interval": 1, "weekdays": [1]}
- "每两周" -> {"type": "weekly", "interval": 2, "weekdays": [1,2,3,4,5]}
- "每周一到五" -> {"type": "weekly", "interval": 1, "weekdays": [1,2,3,4,5]}
- "每周三和周五" -> {"type": "weekly", "interval": 1, "weekdays": [3,5]}

Monthday numbering: Monday=1, Tuesday=2, Wednesday=3, Thursday=4, Friday=5, Saturday=6, Sunday=7

Monthly (每月):
- "每月15号" -> {"type": "monthly", "interval": 1, "month_day": 15}
- "每月1号" -> {"type": "monthly", "interval": 1, "month_day": 1}

====================================
TITLE CLEANING RULES
====================================
Remove these from title:
- Time keywords: 今天, 明天, 后天, 上午, 下午, 晚上, 今天, 本周, 下周, at, in, on
- Date keywords: 1月, 2月, ..., Jan, Feb, ..., Monday, Tuesday, ...
- Numbers that are clearly times: 9点, 2pm, 14:00

Keep in title:
- Meeting subject, event name, activity description

Examples:
- "明天下午2点开会" -> title = "开会"
- "本周五上午10点团队会议" -> title = "团队会议"
- "下周一9点面试" -> title = "面试"

====================================
REMINDER EXTRACTION
====================================
Extract reminders if mentioned:
- "提前10分钟提醒" -> {"type": "before", "value": 10, "unit": "minutes"}
- "提前1小时通知" -> {"type": "before", "value": 1, "unit": "hours"}
- "提前1天提醒" -> {"type": "before", "value": 1, "unit": "days"}

Unit options: "minutes", "hours", "days"
Type options: "before" (currently only supports before)

If no reminder mentioned, return empty array: []

====================================
VALIDATION CHECKLIST
====================================
Before returning, verify:
1. start_time and end_time are in UTC (no timezone suffix)
2. Format is exactly YYYY-MM-DD HH:mm:ss (24-hour, zero-padded)
3. end_time >= start_time
4. start_time is not more than 24 hours in the past
5. all_day only set when explicitly mentioned or time is ambiguous
6. recurrence is null if no recurring pattern mentioned
7. title is not empty
8. All JSON syntax is valid
`, nowUTC.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"), p.location.String(), now.Format("2006-01-02 15:04:05"), now.Format("2006-01-02"))

	userPrompt := fmt.Sprintf("User Input: %s", text)

	response, stats, err := p.llmService.Chat(ctx, []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM parsing failed: %w", err)
	}
	_ = stats // Stats not needed for schedule parsing

	// Clean code blocks if present
	jsonStr := strings.TrimSpace(response)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")

	// Parse JSON response
	var llmResp llmScheduleResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w. Response: %s", err, response)
	}

	// Convert string times to int64
	var startTs, endTs int64

	// Helper to parse time as UTC
	parseTime := func(timeStr string) (int64, error) {
		// Remove possible UTC suffix
		timeStr = strings.TrimSuffix(timeStr, " UTC")
		timeStr = strings.TrimSpace(timeStr)

		// Parse as UTC
		t, err := time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse time %q: %w", timeStr, err)
		}
		return t.Unix(), nil
	}

	if llmResp.StartTime != "" {
		if ts, err := parseTime(llmResp.StartTime); err == nil {
			startTs = ts
		}
	}
	if llmResp.EndTime != "" {
		if ts, err := parseTime(llmResp.EndTime); err == nil {
			endTs = ts
		}
	}

	// Validate time is not too far in the past (more than 24 hours)
	if startTs < nowUTC.Add(-24*time.Hour).Unix() {
		return nil, fmt.Errorf("parsed start time is too far in the past: %d", startTs)
	}

	// Validate end time is not before start time
	if endTs > 0 && endTs < startTs {
		return nil, fmt.Errorf("end time %d is before start time %d", endTs, startTs)
	}

	// Fallback if parsing failed (should rely on LLM correctness though)
	if startTs == 0 {
		startTs = nowUTC.Add(time.Hour).Unix()
	}
	if endTs == 0 || endTs < startTs {
		endTs = startTs + 3600
	}

	// Validate timezone edge cases (DST transitions)
	// This ensures we catch invalid/ambiguous times before creating the schedule
	if p.validator != nil {
		rangeResult := p.validator.ValidateTimeRange(startTs, endTs)
		if len(rangeResult.Warnings) > 0 {
			// Log warnings but continue - the validator has already adjusted times
			slog.Warn("timezone validation warnings",
				"warnings", rangeResult.Warnings,
				"user_timezone", p.validator.GetTimezone())
		}
	}

	return &ParseResult{
		Title:       llmResp.Title,
		Description: llmResp.Description,
		Location:    llmResp.Location,
		StartTs:     startTs,
		EndTs:       endTs,
		AllDay:      llmResp.AllDay,
		Timezone:    p.location.String(),
		Reminders:   llmResp.Reminders,
		Recurrence:  llmResp.Recurrence,
	}, nil

}

// ToSchedule converts ParseResult to v1pb.Schedule.
func (r *ParseResult) ToSchedule() *v1pb.Schedule {
	schedule := &v1pb.Schedule{
		Title:       r.Title,
		Description: r.Description,
		Location:    r.Location,
		StartTs:     r.StartTs,
		EndTs:       r.EndTs,
		AllDay:      r.AllDay,
		Timezone:    r.Timezone,
		Reminders:   r.Reminders,
		State:       "NORMAL",
	}

	// Convert recurrence rule to JSON string
	if r.Recurrence != nil {
		recurrenceJSON, err := r.Recurrence.ToJSON()
		if err != nil {
			// Failed to serialize recurrence rule - create schedule without it
			// This shouldn't happen in practice as RecurrenceRule is simple JSON
			// User can manually edit the schedule later to add recurrence
		} else {
			schedule.RecurrenceRule = recurrenceJSON
		}
	}

	return schedule
}
