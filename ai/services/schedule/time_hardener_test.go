package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/aitime"
)

// mockTimeService implements aitime.TimeService for testing.
type mockTimeService struct {
	normalizeFunc func(ctx context.Context, input string, timezone string) (time.Time, error)
}

func (m *mockTimeService) Normalize(ctx context.Context, input string, timezone string) (time.Time, error) {
	if m.normalizeFunc != nil {
		return m.normalizeFunc(ctx, input, timezone)
	}
	// Default: parse simple formats
	return parseSimpleTime(input, timezone)
}

func (m *mockTimeService) ParseNaturalTime(ctx context.Context, input string, reference time.Time) (aitime.TimeRange, error) {
	return aitime.TimeRange{}, nil
}

// parseSimpleTime is a helper for tests to parse normalized time strings.
func parseSimpleTime(input string, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.Local
	}

	// Try common formats
	formats := []string{
		"2006年1月2日15点04分",
		"2006年1月2日15点",
		"2006年1月2日下午3点",
		"2006年1月2日上午10点",
		"今天15点",
		"明天15点",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, loc); err == nil {
			return t, nil
		}
	}

	// For test purposes, return a fixed future time
	return time.Now().In(loc).Add(time.Hour), nil
}

func TestTimeHardener_ConvertChineseNumbers(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected string
	}{
		{"下午三点", "下午3点"},
		{"十点", "10点"},
		{"十一点", "11点"},
		{"十二点", "12点"},
		{"二十点", "20点"},
		{"二十一点", "21点"},
		{"三十分", "30分"},
		{"下午三点半", "下午3点半"},
		{"一月二十八日", "1月28日"},
		{"两点", "2点"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.convertChineseNumbers(tt.input)
			if result != tt.expected {
				t.Errorf("convertChineseNumbers(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_NormalizeChineseFormat(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected string
	}{
		{"早上8点", "上午8点"},
		{"早晨9点", "上午9点"},
		{"傍晚6点", "下午6点"},
		{"3点钟", "3点"},
		{"3点半", "3点30分"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.normalizeChineseFormat(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeChineseFormat(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_HasTimeButNoDate(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected bool
	}{
		{"15:00", true},
		{"下午3点", true},
		{"3点", true},
		{"今天3点", false},
		{"明天下午3点", false},
		{"2026年1月28日3点", false},
		{"2026-01-28 15:00", false},
		{"下周一3点", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.hasTimeButNoDate(tt.input)
			if result != tt.expected {
				t.Errorf("hasTimeButNoDate(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_ExtractHour(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected int
	}{
		{"15点", 15},
		{"3点30分", 3},
		{"下午3点", 3},
		{"15:30", 15},
		{"没有时间", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.extractHour(tt.input)
			if result != tt.expected {
				t.Errorf("extractHour(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_ExtractMinute(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected int
	}{
		{"15点30分", 30},
		{"3:45", 45},
		{"15点", 0},
		{"没有时间", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.extractMinute(tt.input)
			if result != tt.expected {
				t.Errorf("extractMinute(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_InferDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 14, 0, 0, 0, loc) // 2026-01-27 14:00

	h := &TimeHardener{
		timezone: loc,
		now:      func() time.Time { return fixedNow },
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"15点", "今天15点"},     // 15:00 > 14:00, today
		{"13点", "明天13点"},     // 13:00 < 14:00, tomorrow
		{"下午3点", "今天下午3点"},   // 15:00 > 14:00, today
		{"上午10点", "明天上午10点"}, // 10:00 < 14:00, tomorrow
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.inferDate(tt.input)
			if result != tt.expected {
				t.Errorf("inferDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_ValidateTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 14, 0, 0, 0, loc)

	h := &TimeHardener{
		timezone: loc,
		now:      func() time.Time { return fixedNow },
	}

	tests := []struct {
		time      time.Time
		name      string
		wantError bool
	}{
		{
			name:      "future time within 1 year",
			time:      fixedNow.Add(24 * time.Hour),
			wantError: false,
		},
		{
			name:      "past time",
			time:      fixedNow.Add(-1 * time.Hour),
			wantError: true,
		},
		{
			name:      "time more than 1 year away",
			time:      fixedNow.AddDate(1, 1, 0),
			wantError: true,
		},
		{
			name:      "time within buffer (now)",
			time:      fixedNow.Add(-3 * time.Minute), // Within 5 minute buffer
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.validateTime(tt.time)
			if (err != nil) != tt.wantError {
				t.Errorf("validateTime() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTimeHardener_PreprocessLLMOutput(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc) // 10:00 AM

	h := &TimeHardener{
		timezone: loc,
		now:      func() time.Time { return fixedNow },
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"下午三点", "今天下午3点"},
		{"早上八点", "明天上午8点"}, // 8:00 < 10:00, tomorrow
		{"明天十一点", "明天11点"},
		{"2026年1月28日下午三点", "2026年1月28日下午3点"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.preprocessLLMOutput(tt.input)
			if result != tt.expected {
				t.Errorf("preprocessLLMOutput(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_HardenTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)

	mockService := &mockTimeService{
		normalizeFunc: func(ctx context.Context, input string, timezone string) (time.Time, error) {
			// Return a fixed future time for successful cases
			return fixedNow.Add(5 * time.Hour), nil
		},
	}

	h := &TimeHardener{
		timeService: mockService,
		timezone:    loc,
		now:         func() time.Time { return fixedNow },
	}

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid Chinese time",
			input:     "下午三点",
			wantError: false,
		},
		{
			name:      "empty input",
			input:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.HardenTime(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("HardenTime() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTimeHardener_ValidateTimeRange(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)

	h := &TimeHardener{
		timezone: loc,
		now:      func() time.Time { return fixedNow },
	}

	tests := []struct {
		start     time.Time
		end       time.Time
		name      string
		wantError bool
	}{
		{
			name:      "valid range",
			start:     fixedNow.Add(time.Hour),
			end:       fixedNow.Add(2 * time.Hour),
			wantError: false,
		},
		{
			name:      "end before start",
			start:     fixedNow.Add(2 * time.Hour),
			end:       fixedNow.Add(time.Hour),
			wantError: true,
		},
		{
			name:      "duration too long",
			start:     fixedNow.Add(time.Hour),
			end:       fixedNow.Add(26 * time.Hour), // 25 hours duration > 24 hours
			wantError: true,
		},
		{
			name:      "start in past",
			start:     fixedNow.Add(-2 * time.Hour),
			end:       fixedNow.Add(time.Hour),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.ValidateTimeRange(tt.start, tt.end)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTimeRange() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTimeHardener_ApplyDefaultHour(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input       string
		defaultHour int
		expected    string
	}{
		{"上午", 10, "上午10点"},
		{"下午", 14, "下午14点"},
		{"晚上", 19, "晚上19点"},
		{"中午", 12, "中午12点"},
		{"下午3点", 14, "下午3点"}, // Already has time
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.applyDefaultHour(tt.input, tt.defaultHour)
			if result != tt.expected {
				t.Errorf("applyDefaultHour(%q, %d) = %q, want %q", tt.input, tt.defaultHour, result, tt.expected)
			}
		})
	}
}

func TestTimeHardener_WithTimezone(t *testing.T) {
	loc1, _ := time.LoadLocation("Asia/Shanghai")
	loc2, _ := time.LoadLocation("America/New_York")

	mockService := &mockTimeService{}
	h1 := NewTimeHardener(mockService, loc1)
	h2 := h1.WithTimezone(loc2)

	if h1.timezone.String() != "Asia/Shanghai" {
		t.Errorf("original timezone = %s, want Asia/Shanghai", h1.timezone.String())
	}

	if h2.timezone.String() != "America/New_York" {
		t.Errorf("new timezone = %s, want America/New_York", h2.timezone.String())
	}

	// Ensure they don't share state
	if h1 == h2 {
		t.Error("WithTimezone should return a new instance")
	}
}

func TestTimeHardener_ParseChineseNumber(t *testing.T) {
	h := &TimeHardener{}

	tests := []struct {
		input    string
		expected string
	}{
		{"一", "1"},
		{"十", "10"},
		{"十一", "11"},
		{"十二", "12"},
		{"二十", "20"},
		{"二十一", "21"},
		{"三十", "30"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := h.parseChineseNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseChineseNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
