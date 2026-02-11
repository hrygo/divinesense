// Package schedule provides schedule-related AI agent utilities.
package schedule

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/aitime"
)

// Pre-compiled regex patterns for performance.
var (
	// Time patterns.
	hourMinutePattern = regexp.MustCompile(`(\d{1,2})[点时:](\d{0,2})`)
	periodPattern     = regexp.MustCompile(`(上午|下午|早上|晚上|中午|傍晚)`)
	iso8601Pattern    = regexp.MustCompile(`\d{4}-\d{1,2}-\d{1,2}`)

	// Chinese number patterns.
	chineseNumPattern = regexp.MustCompile(`[一二三四五六七八九十]+`)

	// Date reference patterns.
	dateRefPattern = regexp.MustCompile(`(今天|明天|后天|大后天|昨天|前天|这周|下周|本周|上周)`)

	// Chinese digit map (shared across functions).
	chineseDigitMap = map[rune]int{
		'零': 0, '〇': 0,
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9,
		'两': 2,
	}
)

// TimeHardener provides time parsing hardening for LLM outputs.
// It preprocesses various time formats before delegating to TimeService.
type TimeHardener struct {
	timeService aitime.TimeService
	timezone    *time.Location
	now         func() time.Time
}

// NewTimeHardener creates a new TimeHardener instance.
func NewTimeHardener(timeService aitime.TimeService, timezone *time.Location) *TimeHardener {
	return &TimeHardener{
		timeService: timeService,
		timezone:    timezone,
		now:         time.Now,
	}
}

// WithTimezone returns a new TimeHardener with the specified timezone.
func (h *TimeHardener) WithTimezone(tz *time.Location) *TimeHardener {
	return &TimeHardener{
		timeService: h.timeService,
		timezone:    tz,
		now:         h.now,
	}
}

// 3. Validate the result for reasonableness.
func (h *TimeHardener) HardenTime(ctx context.Context, input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, fmt.Errorf("empty time input")
	}

	// Step 1: Preprocess LLM output
	normalized := h.preprocessLLMOutput(input)

	// Step 2: Call TimeService
	t, err := h.timeService.Normalize(ctx, normalized, h.timezone.String())
	if err != nil {
		return time.Time{}, fmt.Errorf("时间解析失败: %w", err)
	}

	// Step 3: Validate reasonableness
	if err := h.validateTime(t); err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// HardenTimeWithDefaults parses time with default values for missing components.
// defaultHour is used when time has a period but no specific hour.
func (h *TimeHardener) HardenTimeWithDefaults(ctx context.Context, input string, defaultHour int) (time.Time, error) {
	if input == "" {
		return time.Time{}, fmt.Errorf("empty time input")
	}

	// Apply default hour if only period is specified
	normalized := h.applyDefaultHour(input, defaultHour)

	return h.HardenTime(ctx, normalized)
}

// preprocessLLMOutput normalizes various LLM output formats.
func (h *TimeHardener) preprocessLLMOutput(input string) string {
	result := input

	// Step 1: Convert Chinese numbers to Arabic
	result = h.convertChineseNumbers(result)

	// Step 2: Normalize Chinese time format
	result = h.normalizeChineseFormat(result)

	// Step 3: Infer date if only time is provided
	if h.hasTimeButNoDate(result) {
		result = h.inferDate(result)
	}

	return result
}

// convertChineseNumbers converts Chinese numerals to Arabic numerals.
func (h *TimeHardener) convertChineseNumbers(input string) string {
	// Handle compound numbers first (十一, 十二, etc.)
	result := input

	// Handle special compound patterns using regex replacement
	result = chineseNumPattern.ReplaceAllStringFunc(result, func(match string) string {
		return h.parseChineseNumber(match)
	})

	// Handle remaining single digits using shared map
	for ch, digit := range chineseDigitMap {
		result = strings.ReplaceAll(result, string(ch), strconv.Itoa(digit))
	}

	return result
}

// parseChineseNumber parses a Chinese number string to Arabic.
func (h *TimeHardener) parseChineseNumber(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}

	// Single digit
	if len(runes) == 1 {
		if runes[0] == '十' {
			return "10"
		}
		if v, ok := chineseDigitMap[runes[0]]; ok {
			return strconv.Itoa(v)
		}
		return s
	}

	// Handle 十X pattern (10-19)
	if runes[0] == '十' {
		if len(runes) == 1 {
			return "10"
		}
		if v, ok := chineseDigitMap[runes[1]]; ok {
			return strconv.Itoa(10 + v)
		}
	}

	// Handle X十 pattern (20, 30, etc.)
	if len(runes) >= 2 && runes[1] == '十' {
		tens, ok := chineseDigitMap[runes[0]]
		if !ok {
			return s
		}
		if len(runes) == 2 {
			return strconv.Itoa(tens * 10)
		}
		// X十Y pattern (21, 32, etc.)
		if len(runes) >= 3 {
			ones, ok := chineseDigitMap[runes[2]]
			if ok {
				return strconv.Itoa(tens*10 + ones)
			}
		}
	}

	return s
}

// normalizeChineseFormat standardizes Chinese time expressions.
func (h *TimeHardener) normalizeChineseFormat(input string) string {
	result := input

	// Normalize period expressions to standard form
	periodNormalizations := map[string]string{
		"早上": "上午",
		"早晨": "上午",
		"傍晚": "下午",
	}

	for old, new := range periodNormalizations {
		result = strings.ReplaceAll(result, old, new)
	}

	// Normalize "点钟" to "点"
	result = strings.ReplaceAll(result, "点钟", "点")

	// Normalize "点半" to "点30分"
	result = strings.ReplaceAll(result, "点半", "点30分")

	return result
}

// hasTimeButNoDate checks if input has time but no date reference.
func (h *TimeHardener) hasTimeButNoDate(input string) bool {
	// Check for date references
	if dateRefPattern.MatchString(input) {
		return false
	}

	// Check for ISO date format
	if iso8601Pattern.MatchString(input) {
		return false
	}

	// Check for year/month/day format
	if strings.Contains(input, "年") || strings.Contains(input, "月") {
		return false
	}

	// Check if it has time indicators
	hasTime := hourMinutePattern.MatchString(input) || periodPattern.MatchString(input)

	return hasTime
}

// inferDate adds a date reference when only time is provided.
// If the time has already passed today, it assumes tomorrow.
func (h *TimeHardener) inferDate(input string) string {
	now := h.now().In(h.timezone)

	hour := h.extractHour(input)
	minute := h.extractMinute(input)
	period := h.extractPeriod(input)

	// Adjust hour for period
	if period == "下午" || period == "晚上" {
		if hour > 0 && hour < 12 {
			hour += 12
		}
	}

	// Compare with current time
	currentMinutes := now.Hour()*60 + now.Minute()
	targetMinutes := hour*60 + minute

	if targetMinutes <= currentMinutes {
		// Time has passed, assume tomorrow
		return "明天" + input
	}

	return "今天" + input
}

// extractHour extracts the hour from a time string.
func (h *TimeHardener) extractHour(input string) int {
	matches := hourMinutePattern.FindStringSubmatch(input)
	if len(matches) >= 2 {
		hour, err := strconv.Atoi(matches[1])
		if err == nil {
			return hour
		}
	}
	return 0
}

// extractMinute extracts the minute from a time string.
func (h *TimeHardener) extractMinute(input string) int {
	matches := hourMinutePattern.FindStringSubmatch(input)
	if len(matches) >= 3 && matches[2] != "" {
		minute, err := strconv.Atoi(matches[2])
		if err == nil {
			return minute
		}
	}
	return 0
}

// extractPeriod extracts the time period (上午/下午/etc.) from a time string.
func (h *TimeHardener) extractPeriod(input string) string {
	matches := periodPattern.FindStringSubmatch(input)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// applyDefaultHour adds a default hour when only a period is specified.
func (h *TimeHardener) applyDefaultHour(input string, defaultHour int) string {
	// If there's a period but no specific time
	if periodPattern.MatchString(input) && !hourMinutePattern.MatchString(input) {
		period := h.extractPeriod(input)
		hour := defaultHour

		// Adjust default based on period
		switch period {
		case "上午":
			if hour >= 12 {
				hour = 10
			}
		case "下午":
			if hour < 12 {
				hour = 14
			}
		case "晚上":
			if hour < 18 {
				hour = 19
			}
		case "中午":
			hour = 12
		}

		return strings.Replace(input, period, fmt.Sprintf("%s%d点", period, hour), 1)
	}
	return input
}

// validateTime checks if the parsed time is reasonable.
func (h *TimeHardener) validateTime(t time.Time) error {
	now := h.now().In(h.timezone)

	// Allow a small buffer (5 minutes) for "now" schedules
	buffer := 5 * time.Minute
	if t.Before(now.Add(-buffer)) {
		return fmt.Errorf("时间不能早于现在")
	}

	// Check if within 1 year
	oneYearLater := now.AddDate(1, 0, 0)
	if t.After(oneYearLater) {
		return fmt.Errorf("时间太远，请选择一年内的时间")
	}

	return nil
}

// ValidateTimeRange validates a time range.
func (h *TimeHardener) ValidateTimeRange(start, end time.Time) error {
	if err := h.validateTime(start); err != nil {
		return fmt.Errorf("开始时间无效: %w", err)
	}

	if err := h.validateTime(end); err != nil {
		return fmt.Errorf("结束时间无效: %w", err)
	}

	if end.Before(start) {
		return fmt.Errorf("结束时间不能早于开始时间")
	}

	// Check reasonable duration (max 24 hours for single event)
	if end.Sub(start) > 24*time.Hour {
		return fmt.Errorf("单个日程时长不能超过24小时")
	}

	return nil
}
