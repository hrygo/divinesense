package aitime

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MockTimeService is a mock implementation of TimeService for testing.
type MockTimeService struct {
	// FixedNow can be set to use a fixed "now" for testing
	FixedNow *time.Time
}

// NewMockTimeService creates a new MockTimeService.
func NewMockTimeService() *MockTimeService {
	return &MockTimeService{}
}

// Normalize standardizes time expressions.
func (m *MockTimeService) Normalize(ctx context.Context, input string, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.Local
	}

	now := m.now().In(loc)
	input = strings.TrimSpace(input)

	// Try standard formats first
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"15:04:05",
		"15:04",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, loc); err == nil {
			// If only time, use today's date
			if format == "15:04:05" || format == "15:04" {
				return time.Date(now.Year(), now.Month(), now.Day(),
					t.Hour(), t.Minute(), t.Second(), 0, loc), nil
			}
			return t, nil
		}
	}

	// Parse Chinese expressions
	return m.parseChineseTime(input, now, loc)
}

// ParseNaturalTime parses natural language time expressions.
func (m *MockTimeService) ParseNaturalTime(ctx context.Context, input string, reference time.Time) (TimeRange, error) {
	input = strings.TrimSpace(input)

	// Simple patterns
	patterns := map[string]func(time.Time) TimeRange{
		"今天": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location())
			end := start.Add(24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"明天": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(24 * time.Hour)
			end := start.Add(24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"后天": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(48 * time.Hour)
			end := start.Add(24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"昨天": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(-24 * time.Hour)
			end := start.Add(24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"这周": func(ref time.Time) TimeRange {
			weekday := int(ref.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(-time.Duration(weekday-1) * 24 * time.Hour)
			end := start.Add(7 * 24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"下周": func(ref time.Time) TimeRange {
			weekday := int(ref.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(time.Duration(8-weekday) * 24 * time.Hour)
			end := start.Add(7 * 24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"上周": func(ref time.Time) TimeRange {
			weekday := int(ref.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(-time.Duration(weekday+6) * 24 * time.Hour)
			end := start.Add(7 * 24 * time.Hour)
			return TimeRange{Start: start, End: end}
		},
		"这个月": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location())
			end := start.AddDate(0, 1, 0)
			return TimeRange{Start: start, End: end}
		},
		"下个月": func(ref time.Time) TimeRange {
			start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location()).AddDate(0, 1, 0)
			end := start.AddDate(0, 1, 0)
			return TimeRange{Start: start, End: end}
		},
	}

	for pattern, fn := range patterns {
		if strings.Contains(input, pattern) {
			return fn(reference), nil
		}
	}

	// Try to parse specific time
	t, err := m.Normalize(ctx, input, reference.Location().String())
	if err == nil {
		// Default to 1-hour duration for specific times
		return TimeRange{Start: t, End: t.Add(time.Hour)}, nil
	}

	return TimeRange{}, fmt.Errorf("unable to parse time expression: %s", input)
}

// parseChineseTime parses Chinese time expressions.
func (m *MockTimeService) parseChineseTime(input string, now time.Time, loc *time.Location) (time.Time, error) {
	result := now

	// Day offset patterns
	if strings.Contains(input, "明天") {
		result = result.AddDate(0, 0, 1)
	} else if strings.Contains(input, "后天") {
		result = result.AddDate(0, 0, 2)
	} else if strings.Contains(input, "昨天") {
		result = result.AddDate(0, 0, -1)
	}

	// Time of day patterns
	hour := -1
	minute := 0

	// Extract numbers
	numPattern := regexp.MustCompile(`(\d+)`)

	// Chinese numbers ordered by length (longest first) to ensure correct matching
	// e.g., "十一点" should match 11, not 10
	chineseNums := []struct {
		pattern string
		value   int
	}{
		{"十二", 12}, // longest first
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
		{"一", 1},
	}

	// Check for Chinese numbers (longest match first)
	for _, cn := range chineseNums {
		if strings.Contains(input, cn.pattern+"点") {
			hour = cn.value
			break
		}
	}

	// Check for Arabic numbers
	if hour == -1 {
		matches := numPattern.FindStringSubmatch(input)
		if len(matches) > 1 {
			h, _ := strconv.Atoi(matches[1])
			if h >= 0 && h <= 24 {
				hour = h
			}
		}
	}

	// AM/PM patterns
	if hour != -1 && hour <= 12 {
		if strings.Contains(input, "下午") || strings.Contains(input, "晚上") {
			if hour < 12 {
				hour += 12
			}
		} else if strings.Contains(input, "上午") || strings.Contains(input, "早上") {
			// Keep as is
		} else if hour < 6 {
			// Ambiguous small hours, assume PM for convenience
			hour += 12
		}
	}

	// Extract minutes
	minutePattern := regexp.MustCompile(`(\d+)分`)
	if matches := minutePattern.FindStringSubmatch(input); len(matches) > 1 {
		minute, _ = strconv.Atoi(matches[1])
	} else if strings.Contains(input, "半") {
		minute = 30
	}

	if hour != -1 {
		result = time.Date(result.Year(), result.Month(), result.Day(),
			hour, minute, 0, 0, loc)
		return result, nil
	}

	return result, fmt.Errorf("unable to parse time: %s", input)
}

// now returns the current time (or fixed time for testing).
func (m *MockTimeService) now() time.Time {
	if m.FixedNow != nil {
		return *m.FixedNow
	}
	return time.Now()
}

// Ensure MockTimeService implements TimeService.
var _ TimeService = (*MockTimeService)(nil)
