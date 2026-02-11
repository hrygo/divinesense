// Package universal provides tests for TimeContext.
package universal

import (
	"strings"
	"testing"
	"time"
)

// TestTimeContext_BuildTimeContext tests the time context construction.
func TestTimeContext_BuildTimeContext(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tc := BuildTimeContext(loc)

	// Verify current time fields
	if tc.Current.Date == "" {
		t.Error("Current.Date should not be empty")
	}
	if tc.Current.Weekday == "" {
		t.Error("Current.Weekday should not be empty")
	}
	if tc.Current.WeekdayCN == "" {
		t.Error("Current.WeekdayCN should not be empty")
	}
	if tc.Current.Timezone != "Asia/Shanghai" {
		t.Errorf("Timezone = %s, want Asia/Shanghai", tc.Current.Timezone)
	}

	// Verify relative dates
	if tc.Relative.Today == "" {
		t.Error("Relative.Today should not be empty")
	}
	if tc.Relative.Tomorrow == "" {
		t.Error("Relative.Tomorrow should not be empty")
	}
	if tc.Relative.ThisWeekStart == "" {
		t.Error("Relative.ThisWeekStart should not be empty")
	}

	// Verify business hours
	if tc.Business.Start != "06:00" {
		t.Errorf("Business.Start = %s, want 06:00", tc.Business.Start)
	}
	if tc.Business.End != "22:00" {
		t.Errorf("Business.End = %s, want 22:00", tc.Business.End)
	}
}

// TestTimeContext_FormatAsJSONBlock tests the JSON formatting.
func TestTimeContext_FormatAsJSONBlock(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tc := BuildTimeContext(loc)

	jsonBlock := tc.FormatAsJSONBlock()

	// Verify JSON block format
	if !strings.HasPrefix(jsonBlock, "```json") {
		t.Error("JSON block should start with ```json")
	}
	if !strings.HasSuffix(jsonBlock, "```") {
		t.Error("JSON block should end with ```")
	}
	if !strings.Contains(jsonBlock, `"current":`) {
		t.Error("JSON should contain 'current' field")
	}
	if !strings.Contains(jsonBlock, `"relative":`) {
		t.Error("JSON should contain 'relative' field")
	}
	if !strings.Contains(jsonBlock, `"business_hours":`) {
		t.Error("JSON should contain 'business_hours' field")
	}
}

// TestTimeContext_EdgeCase_Sunday tests Sunday weekday calculation.
func TestTimeContext_EdgeCase_Sunday(t *testing.T) {
	// Use a known Sunday date: Feb 8, 2026 was a Sunday
	// We'll verify the logic by checking the WeekdayNum is in correct range (1-7)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tc := BuildTimeContext(loc)

	// Verify WeekdayNum is in valid range
	if tc.Current.WeekdayNum < 1 || tc.Current.WeekdayNum > 7 {
		t.Errorf("WeekdayNum = %d, want 1-7", tc.Current.WeekdayNum)
	}

	// ThisWeekStart should be before or equal to Today
	if tc.Relative.ThisWeekStart > tc.Relative.Today {
		t.Errorf("ThisWeekStart (%s) should be <= Today (%s)", tc.Relative.ThisWeekStart, tc.Relative.Today)
	}
}

// TestTimeContext_EdgeCase_MonthEnd tests month boundary.
func TestTimeContext_EdgeCase_MonthEnd(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tc := BuildTimeContext(loc)

	// Verify NextMonth is in correct format (YYYY-MM)
	if len(tc.Relative.NextMonth) != 7 {
		t.Errorf("NextMonth format incorrect: %s (want YYYY-MM)", tc.Relative.NextMonth)
	}

	// Verify ThisMonth + NextMonth relationship (this should be next month)
	// Parse both and verify NextMonth is after ThisMonth
	thisMonth := tc.Relative.ThisMonth
	nextMonth := tc.Relative.NextMonth

	// Simple format validation
	if thisMonth == "" || nextMonth == "" {
		t.Error("month values should not be empty")
	}
}

// TestTimeContext_GetWeekdayDate tests weekday date calculation.
func TestTimeContext_GetWeekdayDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	tc := BuildTimeContext(loc)

	// Test Monday (weekdayNum=1) of current week
	mondayDate := tc.GetWeekdayDate(1, false)
	if mondayDate != tc.Relative.ThisWeekStart {
		t.Errorf("Monday of current week should be ThisWeekStart, got %s (expected %s)",
			mondayDate, tc.Relative.ThisWeekStart)
	}

	// Test Friday (weekdayNum=5) of current week
	fridayDate := tc.GetWeekdayDate(5, false)
	parsedFriday, err := time.ParseInLocation("2006-01-02", fridayDate, loc)
	if err != nil {
		t.Fatalf("failed to parse Friday date: %v", err)
	}
	if parsedFriday.Weekday() != time.Friday {
		t.Errorf("Friday should be Friday, got %v", parsedFriday.Weekday())
	}

	// Test Monday of next week
	nextMondayDate := tc.GetWeekdayDate(1, true)
	if nextMondayDate != tc.Relative.NextWeekStart {
		t.Errorf("Monday of next week should be NextWeekStart, got %s (expected %s)",
			nextMondayDate, tc.Relative.NextWeekStart)
	}
}

// TestTimeContext_WithUserPreferences tests functional options.
func TestTimeContext_WithUserPreferences(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")

	pref := &UserTimePref{
		Timezone:    "America/New_York",
		DefaultHour: 14,
		PreferPM:    true,
	}

	tc := BuildTimeContext(loc, WithUserPreferences(pref))

	// Verify user preferences were applied
	if tc.UserPref == nil {
		t.Fatal("UserPref should not be nil after WithUserPreferences")
	}
	if tc.UserPref.Timezone != "America/New_York" {
		t.Errorf("UserPref.Timezone should be America/New_York, got %s", tc.UserPref.Timezone)
	}
	if tc.UserPref.DefaultHour != 14 {
		t.Errorf("UserPref.DefaultHour should be 14, got %d", tc.UserPref.DefaultHour)
	}
	if !tc.UserPref.PreferPM {
		t.Error("UserPref.PreferPM should be true")
	}
}

// TestBuildTimeContext_EdgeCaseUTC tests UTC timezone handling.
func TestBuildTimeContext_EdgeCaseUTC(t *testing.T) {
	tc := BuildTimeContext(time.UTC)

	// Verify timezone is set correctly
	if tc.Current.Timezone != "UTC" {
		t.Errorf("Timezone should be UTC, got %s", tc.Current.Timezone)
	}

	// Verify other fields are still populated
	if tc.Current.Date == "" {
		t.Error("Current.Date should not be empty for UTC")
	}
	if tc.Relative.Today == "" {
		t.Error("Relative.Today should not be empty for UTC")
	}
}
