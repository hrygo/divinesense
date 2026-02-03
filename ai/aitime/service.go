package aitime

import (
	"context"
	"fmt"
	"time"
)

// Service implements TimeService with rule-based parsing.
type Service struct {
	defaultTimezone *time.Location
}

// NewService creates a new time service.
func NewService(defaultTimezone string) *Service {
	loc, err := time.LoadLocation(defaultTimezone)
	if err != nil {
		loc = time.Local
	}
	return &Service{
		defaultTimezone: loc,
	}
}

// Normalize standardizes time expressions.
func (s *Service) Normalize(_ context.Context, input string, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = s.defaultTimezone
	}

	parser := NewParser(loc)
	return parser.Parse(input)
}

// ParseNaturalTime parses natural language time expressions.
func (s *Service) ParseNaturalTime(_ context.Context, input string, reference time.Time) (TimeRange, error) {
	// First try to parse as a time range keyword
	tr, err := s.parseRangeKeyword(input, reference)
	if err == nil {
		return tr, nil
	}

	// Then try to parse as a specific time
	// Create parser with reference time as "now" for relative date parsing
	parser := &Parser{
		timezone: reference.Location(),
		now:      func() time.Time { return reference },
	}
	t, err := parser.Parse(input)
	if err != nil {
		return TimeRange{}, err
	}

	// For specific times, default to 1-hour duration
	return TimeRange{
		Start: t,
		End:   t.Add(time.Hour),
	}, nil
}

// parseRangeKeyword parses time range keywords like "今天", "这周".
func (s *Service) parseRangeKeyword(input string, ref time.Time) (TimeRange, error) {
	loc := ref.Location()
	dayStart := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, loc)

	// Day ranges
	dayRanges := map[string]int{
		"今天": 0,
		"明天": 1,
		"后天": 2,
		"昨天": -1,
		"前天": -2,
	}

	for keyword, offset := range dayRanges {
		if input == keyword || input == keyword+"的" {
			start := dayStart.AddDate(0, 0, offset)
			return TimeRange{Start: start, End: start.Add(24 * time.Hour)}, nil
		}
	}

	// Week ranges
	weekday := int(ref.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	mondayOffset := -(weekday - 1)

	weekRanges := map[string]int{
		"这周":  0,
		"本周":  0,
		"这个周": 0,
		"下周":  7,
		"下一周": 7,
		"上周":  -7,
		"上一周": -7,
	}

	for keyword, offset := range weekRanges {
		if input == keyword {
			monday := dayStart.AddDate(0, 0, mondayOffset+offset)
			return TimeRange{Start: monday, End: monday.Add(7 * 24 * time.Hour)}, nil
		}
	}

	// Month ranges
	monthStart := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, loc)

	switch input {
	case "这个月", "本月":
		nextMonth := monthStart.AddDate(0, 1, 0)
		return TimeRange{Start: monthStart, End: nextMonth}, nil
	case "下个月", "下月":
		start := monthStart.AddDate(0, 1, 0)
		end := start.AddDate(0, 1, 0)
		return TimeRange{Start: start, End: end}, nil
	case "上个月", "上月":
		start := monthStart.AddDate(0, -1, 0)
		return TimeRange{Start: start, End: monthStart}, nil
	}

	return TimeRange{}, fmt.Errorf("unable to parse time expression: %s", input)
}

// Ensure Service implements TimeService.
var _ TimeService = (*Service)(nil)
