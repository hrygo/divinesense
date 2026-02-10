// Package universal provides structured time context for LLM consumption.
package universal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// TimeContext represents structured time metadata for LLM consumption.
// JSON format provides 20%+ better parsing accuracy than free-form text.
type TimeContext struct {
	Current  CurrentTime   `json:"current"`
	Relative RelativeDates `json:"relative"`
	Business BusinessHours `json:"business_hours"`
	UserPref *UserTimePref `json:"user_pref,omitempty"`
}

// CurrentTime represents the current time information.
type CurrentTime struct {
	Date       string `json:"date"`        // 2026-02-10
	Time       string `json:"time"`        // 19:06:00
	DateTime   string `json:"datetime"`    // 2026-02-10 19:06:00
	Weekday    string `json:"weekday"`     // Monday
	WeekdayCN  string `json:"weekday_cn"`  // 周一
	WeekdayNum int    `json:"weekday_num"` // 1-7 (1=Monday)
	Timezone   string `json:"timezone"`    // Asia/Shanghai
	Timestamp  int64  `json:"timestamp"`   // Unix timestamp
}

// RelativeDates represents commonly used relative dates.
type RelativeDates struct {
	Today            string `json:"today"`              // 2026-02-10
	Tomorrow         string `json:"tomorrow"`           // 2026-02-11
	DayAfterTomorrow string `json:"day_after_tomorrow"` // 2026-02-12
	ThisWeekStart    string `json:"this_week_start"`    // Monday of this week
	ThisWeekEnd      string `json:"this_week_end"`      // Sunday of this week
	NextWeekStart    string `json:"next_week_start"`    // Monday of next week
	NextWeekEnd      string `json:"next_week_end"`      // Sunday of next week
	ThisMonth        string `json:"this_month"`         // 2026-02
	NextMonth        string `json:"next_month"`         // 2026-03
}

// BusinessHours represents the acceptable scheduling hours.
type BusinessHours struct {
	Start      string `json:"start"`       // 06:00
	End        string `json:"end"`         // 22:00
	StartHour  int    `json:"start_hour"`  // 6
	EndHour    int    `json:"end_hour"`    // 22
	DefaultAM  string `json:"default_am"`  // 09:00
	DefaultPM  string `json:"default_pm"`  // 14:00
	DefaultEve string `json:"default_eve"` // 19:00 (for "今晚")
}

// UserTimePref represents user-specific time preferences.
type UserTimePref struct {
	Timezone    string `json:"timezone,omitempty"`
	DefaultHour int    `json:"default_hour,omitempty"` // Preferred hour for meetings
	PreferAM    bool   `json:"prefer_am,omitempty"`
	PreferPM    bool   `json:"prefer_pm,omitempty"`
}

// TimeContextOption is a functional option for BuildTimeContext.
type TimeContextOption func(*TimeContext)

// WithUserPreferences applies user-specific time preferences.
func WithUserPreferences(pref *UserTimePref) TimeContextOption {
	return func(tc *TimeContext) {
		tc.UserPref = pref
	}
}

// BuildTimeContext creates a structured time context for the LLM.
func BuildTimeContext(loc *time.Location, opts ...TimeContextOption) *TimeContext {
	now := time.Now().In(loc)

	// Calculate this week's Monday and Sunday (ISO week: Monday=1, Sunday=7)
	weekday := now.Weekday()
	var daysSinceMonday int
	if weekday == time.Sunday {
		daysSinceMonday = 6
	} else {
		daysSinceMonday = int(weekday - time.Monday)
	}

	thisWeekMonday := now.AddDate(0, 0, -daysSinceMonday)
	thisWeekSunday := thisWeekMonday.AddDate(0, 0, 6)
	nextWeekMonday := thisWeekMonday.AddDate(0, 0, 7)
	nextWeekSunday := nextWeekMonday.AddDate(0, 0, 6)

	// Weekday number (1=Monday, 7=Sunday)
	weekdayNum := int(weekday)
	if weekday == time.Sunday {
		weekdayNum = 7
	}

	tc := &TimeContext{
		Current: CurrentTime{
			Date:       now.Format("2006-01-02"),
			Time:       now.Format("15:04:05"),
			DateTime:   now.Format("2006-01-02 15:04:05"),
			Weekday:    now.Format("Monday"),
			WeekdayCN:  getWeekdayCN(now.Weekday()),
			WeekdayNum: weekdayNum,
			Timezone:   loc.String(),
			Timestamp:  now.Unix(),
		},
		Relative: RelativeDates{
			Today:            now.Format("2006-01-02"),
			Tomorrow:         now.AddDate(0, 0, 1).Format("2006-01-02"),
			DayAfterTomorrow: now.AddDate(0, 0, 2).Format("2006-01-02"),
			ThisWeekStart:    thisWeekMonday.Format("2006-01-02"),
			ThisWeekEnd:      thisWeekSunday.Format("2006-01-02"),
			NextWeekStart:    nextWeekMonday.Format("2006-01-02"),
			NextWeekEnd:      nextWeekSunday.Format("2006-01-02"),
			ThisMonth:        now.Format("2006-01"),
			NextMonth:        now.AddDate(0, 1, 0).Format("2006-01"),
		},
		Business: BusinessHours{
			Start:      "06:00",
			End:        "22:00",
			StartHour:  6,
			EndHour:    22,
			DefaultAM:  "09:00",
			DefaultPM:  "14:00",
			DefaultEve: "19:00",
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(tc)
	}

	return tc
}

// getWeekdayCN returns the Chinese name of a weekday.
func getWeekdayCN(w time.Weekday) string {
	weekdaysCN := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return weekdaysCN[w]
}

// FormatAsJSONBlock formats the time context as a JSON code block.
// This format is most easily parsed by LLMs.
// On marshal failure, returns a minimal valid fallback with current timestamp.
func (tc *TimeContext) FormatAsJSONBlock() string {
	data, err := json.MarshalIndent(tc, "  ", "  ")
	if err != nil {
		// Log error for monitoring but provide fallback to avoid breaking LLM prompts
		slog.Error("failed to marshal time context, using fallback",
			"error", err,
			"time_context", fmt.Sprintf("%+v", tc),
		)
		// Return a minimal valid fallback with current timestamp
		fallback := map[string]interface{}{
			"current": map[string]interface{}{
				"timestamp": time.Now().Unix(),
				"timezone":  "UTC",
			},
		}
		data, _ = json.MarshalIndent(fallback, "  ", "  ")
	}
	return "```json\n  " + string(data) + "\n```"
}

// GetWeekdayDate calculates the date for a weekday in the current or next week.
func (tc *TimeContext) GetWeekdayDate(weekdayNum int, nextWeek bool) string {
	refDate := tc.Relative.ThisWeekStart
	if nextWeek {
		refDate = tc.Relative.NextWeekStart
	}

	t, err := time.ParseInLocation("2006-01-02", refDate, time.Local)
	if err != nil {
		return refDate
	}
	targetDate := t.AddDate(0, 0, weekdayNum-1)
	return targetDate.Format("2006-01-02")
}
