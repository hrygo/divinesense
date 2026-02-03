package aitime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Patterns for time parsing.
var (
	// Arabic number patterns.
	numberPattern  = regexp.MustCompile(`(\d+)`)
	minutePattern  = regexp.MustCompile(`(\d+)\s*分`)
	hourMinPattern = regexp.MustCompile(`(\d{1,2})[:\s时点](\d{1,2})`)

	// Relative time patterns.
	relativePattern = regexp.MustCompile(`(\d+)\s*(小时|分钟|天|周|月)([后前])`)

	// Weekday patterns.
	weekdayPattern     = regexp.MustCompile(`[这本]?周([一二三四五六日天])`)
	nextWeekdayPattern = regexp.MustCompile(`下周([一二三四五六日天])`)
	lastWeekdayPattern = regexp.MustCompile(`上周([一二三四五六日天])`)
)

// relDateOffsets maps relative date keywords to day offsets.
var relDateOffsets = map[string]int{
	"今天":  0,
	"明天":  1,
	"后天":  2,
	"大后天": 3,
	"昨天":  -1,
	"前天":  -2,
}

// periodHours maps time period keywords to typical hours.
var periodHours = map[string]int{
	"早上": 7,
	"上午": 9,
	"中午": 12,
	"下午": 14,
	"傍晚": 17,
	"晚上": 19,
	"夜里": 22,
	"凌晨": 2,
}

// chineseNums maps Chinese numbers to integers (ordered by length for correct matching).
var chineseNums = []struct {
	pattern string
	value   int
}{
	{"二十四", 24},
	{"二十三", 23},
	{"二十二", 22},
	{"二十一", 21},
	{"二十", 20},
	{"十九", 19},
	{"十八", 18},
	{"十七", 17},
	{"十六", 16},
	{"十五", 15},
	{"十四", 14},
	{"十三", 13},
	{"十二", 12},
	{"十一", 11},
	{"十", 10},
	{"九", 9},
	{"八", 8},
	{"七", 7},
	{"六", 6},
	{"五", 5},
	{"四", 4},
	{"三", 3},
	{"二", 2},
	{"两", 2},
	{"一", 1},
}

// weekdayMap maps Chinese weekday names to time.Weekday offset from Monday.
var weekdayMap = map[string]int{
	"一": 0, // Monday
	"二": 1,
	"三": 2,
	"四": 3,
	"五": 4,
	"六": 5,
	"日": 6,
	"天": 6,
}

// Parser parses natural language time expressions.
type Parser struct {
	timezone *time.Location
	now      func() time.Time
}

// NewParser creates a new time parser with the given timezone.
func NewParser(timezone *time.Location) *Parser {
	if timezone == nil {
		timezone = time.Local
	}
	return &Parser{
		timezone: timezone,
		now:      time.Now,
	}
}

// WithTimezone returns a new parser with the given timezone.
func (p *Parser) WithTimezone(tz *time.Location) *Parser {
	return &Parser{
		timezone: tz,
		now:      p.now,
	}
}

// Parse parses a time expression and returns the parsed time.
func (p *Parser) Parse(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty input")
	}

	now := p.now().In(p.timezone)

	// Try standard formats first
	if t, ok := p.tryStandardFormats(input); ok {
		return t, nil
	}

	// Try relative time (e.g., "1小时后")
	if t, ok := p.tryRelativeTime(input, now); ok {
		return t, nil
	}

	// Parse Chinese expressions
	return p.parseChineseTime(input, now)
}

// tryStandardFormats attempts to parse standard date/time formats.
func (p *Parser) tryStandardFormats(input string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"2006年01月02日 15:04",
		"2006年01月02日",
		"2006年1月2日 15:04",
		"2006年1月2日",
		"01/02/2006",
		"15:04:05",
		"15:04",
	}

	now := p.now().In(p.timezone)

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, p.timezone); err == nil {
			// If only time, use today's date
			if format == "15:04:05" || format == "15:04" {
				return time.Date(now.Year(), now.Month(), now.Day(),
					t.Hour(), t.Minute(), t.Second(), 0, p.timezone), true
			}
			return t, true
		}
	}

	return time.Time{}, false
}

// tryRelativeTime parses relative time expressions like "1小时后".
func (p *Parser) tryRelativeTime(input string, now time.Time) (time.Time, bool) {
	matches := relativePattern.FindStringSubmatch(input)
	if len(matches) != 4 {
		return time.Time{}, false
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, false
	}
	unit := matches[2]
	direction := matches[3]

	var d time.Duration
	switch unit {
	case "小时":
		d = time.Duration(n) * time.Hour
	case "分钟":
		d = time.Duration(n) * time.Minute
	case "天":
		d = time.Duration(n) * 24 * time.Hour
	case "周":
		d = time.Duration(n) * 7 * 24 * time.Hour
	case "月":
		if direction == "后" {
			return now.AddDate(0, n, 0), true
		}
		return now.AddDate(0, -n, 0), true
	default:
		return time.Time{}, false
	}

	if direction == "前" {
		d = -d
	}

	return now.Add(d), true
}

// parseChineseTime parses Chinese time expressions.
func (p *Parser) parseChineseTime(input string, now time.Time) (time.Time, error) {
	result := now

	// Parse date part
	dateModified := false

	// Check relative dates (今天/明天/后天/昨天)
	for keyword, offset := range relDateOffsets {
		if strings.Contains(input, keyword) {
			result = result.AddDate(0, 0, offset)
			dateModified = true
			break
		}
	}

	// Check weekday patterns
	if !dateModified {
		if weekday, ok := p.parseWeekday(input, now); ok {
			result = weekday
			dateModified = true
		}
	}

	// Parse time part
	hour, minute, timeFound := p.parseTimePart(input)

	if timeFound {
		result = time.Date(result.Year(), result.Month(), result.Day(),
			hour, minute, 0, 0, p.timezone)
		return result, nil
	}

	// If only date was found, default to 9:00
	if dateModified {
		result = time.Date(result.Year(), result.Month(), result.Day(),
			9, 0, 0, 0, p.timezone)
		return result, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", input)
}

// parseWeekday parses weekday expressions.
func (p *Parser) parseWeekday(input string, now time.Time) (time.Time, bool) {
	// Current weekday (Monday = 0)
	currentWeekday := int(now.Weekday())
	if currentWeekday == 0 {
		currentWeekday = 7
	}
	currentWeekday-- // Convert to Monday = 0

	// Next week - check FIRST to avoid matching "周X" in "下周X"
	if matches := nextWeekdayPattern.FindStringSubmatch(input); len(matches) > 1 {
		targetWeekday := weekdayMap[matches[1]]
		// Days until next Monday + target weekday
		daysUntilNextMonday := 7 - currentWeekday
		diff := daysUntilNextMonday + targetWeekday
		return now.AddDate(0, 0, diff), true
	}

	// Last week - check SECOND
	if matches := lastWeekdayPattern.FindStringSubmatch(input); len(matches) > 1 {
		targetWeekday := weekdayMap[matches[1]]
		// Go back to last Monday, then add target weekday
		daysToLastMonday := currentWeekday + 7
		diff := -daysToLastMonday + targetWeekday
		return now.AddDate(0, 0, diff), true
	}

	// This week - check LAST
	if matches := weekdayPattern.FindStringSubmatch(input); len(matches) > 1 {
		targetWeekday := weekdayMap[matches[1]]
		diff := targetWeekday - currentWeekday
		return now.AddDate(0, 0, diff), true
	}

	return time.Time{}, false
}

// parseTimePart parses the time part of an expression.
func (p *Parser) parseTimePart(input string) (hour, minute int, found bool) {
	hour = -1
	minute = 0

	// Try HH:MM format
	if matches := hourMinPattern.FindStringSubmatch(input); len(matches) > 2 {
		h, _ := strconv.Atoi(matches[1]) //nolint:errcheck // regex ensures digits
		m, _ := strconv.Atoi(matches[2]) //nolint:errcheck // regex ensures digits
		if h >= 0 && h <= 24 && m >= 0 && m < 60 {
			return h, m, true
		}
	}

	// Try Chinese hour (X点)
	for _, cn := range chineseNums {
		if strings.Contains(input, cn.pattern+"点") {
			hour = cn.value
			break
		}
	}

	// Try Arabic number + 点
	if hour == -1 {
		matches := numberPattern.FindAllStringSubmatch(input, -1)
		for _, m := range matches {
			if len(m) > 1 {
				h, _ := strconv.Atoi(m[1]) //nolint:errcheck // regex ensures digits
				if h >= 0 && h <= 24 && strings.Contains(input, m[1]+"点") {
					hour = h
					break
				}
			}
		}
	}

	// Apply AM/PM modifiers
	if hour != -1 && hour <= 12 {
		hasPMModifier := strings.Contains(input, "下午") || strings.Contains(input, "晚上") ||
			strings.Contains(input, "傍晚") || strings.Contains(input, "夜里")
		hasAMModifier := strings.Contains(input, "上午") || strings.Contains(input, "早上") ||
			strings.Contains(input, "凌晨") || strings.Contains(input, "中午")

		// Determine AM/PM based on modifiers or reasonable defaults
		// - With explicit PM modifier: always PM
		// - With explicit AM modifier: always AM
		// - No modifier + hour 1-6: default to PM (13:00-18:00) - afternoon meeting times
		// - No modifier + hour 7-11: keep as AM (07:00-11:00) - morning meeting times
		// - hour 12: noon, keep as 12:00
		if hasPMModifier {
			if hour < 12 {
				hour += 12
			}
		} else if !hasAMModifier && hour >= 1 && hour <= 6 {
			// Ambiguous 1-6点 defaults to PM (more common for reminders)
			hour += 12
		}
		// hour 7-11 without modifier stays as AM (common work hours)
		// hour 12 stays as 12 (noon)
	}

	// Parse minutes
	if matches := minutePattern.FindStringSubmatch(input); len(matches) > 1 {
		minute, _ = strconv.Atoi(matches[1]) //nolint:errcheck // regex ensures digits
	} else if strings.Contains(input, "半") {
		minute = 30
	}

	// If only period keyword, use default hour
	if hour == -1 {
		for keyword, h := range periodHours {
			if strings.Contains(input, keyword) {
				hour = h
				found = true
				break
			}
		}
	}

	if hour != -1 {
		found = true
	}

	return hour, minute, found
}
