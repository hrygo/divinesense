package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateInstances_Daily(t *testing.T) {
	// Start: 2024-01-01 00:00:00 UTC (Monday)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	startTs := start.Unix()

	rule := &RecurrenceRule{Type: "daily", Interval: 1}

	// Generate for 5 days: Jan 1 to Jan 6 (Exclusive of end bound usually? Code says Inclusive Equal)
	end := start.AddDate(0, 0, 5).Unix() // Jan 6
	instances := rule.GenerateInstances(startTs, end)

	// Jan 1, 2, 3, 4, 5, 6 -> 6 instances.
	assert.Len(t, instances, 6)
	assert.Equal(t, startTs, instances[0])
	assert.Equal(t, start.AddDate(0, 0, 1).Unix(), instances[1])
	assert.Equal(t, end, instances[5])
}

func TestGenerateInstances_Weekly(t *testing.T) {
	// Start: 2024-01-01 (Monday)
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	startTs := start.Unix()

	// Every Monday
	rule := &RecurrenceRule{Type: "weekly", Interval: 1, Weekdays: []int{1}}

	// Generate for 21 days (3 weeks)
	end := start.AddDate(0, 0, 21).Unix()
	instances := rule.GenerateInstances(startTs, end)

	// Jan 1 (Mon), Jan 8 (Mon), Jan 15 (Mon), Jan 22 (Mon).
	// Jan 1 + 21 = Jan 22. Inclusive.
	assert.Len(t, instances, 4)
	assert.Equal(t, startTs, instances[0])
	assert.Equal(t, start.AddDate(0, 0, 7).Unix(), instances[1])
}

func TestGenerateInstances_Monthly_EdgeCase(t *testing.T) {
	// Start: Jan 31 2024
	start := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	startTs := start.Unix()
	rule := &RecurrenceRule{Type: "monthly", MonthDay: 31, Interval: 1}

	// End: April 30 2024 (exclusive, so we get Jan, Feb, Mar, Apr)
	// Using April 30 at end of day to avoid including May 1
	end := time.Date(2024, 4, 30, 23, 59, 59, 0, time.UTC).Unix()

	instances := rule.GenerateInstances(startTs, end)

	// Expected: Jan 31, Feb 29 (Leap), Mar 31, Apr 30.
	require.Len(t, instances, 4)

	// Check Feb (Index 1)
	febInst := time.Unix(instances[1], 0).In(time.UTC)
	assert.Equal(t, time.Month(2), febInst.Month())
	assert.Equal(t, 29, febInst.Day())

	// Check Apr (Index 3)
	aprInst := time.Unix(instances[3], 0).In(time.UTC)
	assert.Equal(t, time.Month(4), aprInst.Month())
	assert.Equal(t, 30, aprInst.Day())
}

func TestGenerateInstances_Limit(t *testing.T) {
	// Generate many instances
	start := time.Now().Unix()
	rule := &RecurrenceRule{Type: "daily", Interval: 1}

	// No end time -> Max instances 1000 OR 1 year limit (approx 366 days)
	instances := rule.GenerateInstances(start, 0)
	assert.True(t, len(instances) >= 365 && len(instances) <= 367, "Should generate approx 1 year of instances")
}

func TestParseRecurrenceRuleFromJSON_Normalization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string // Expected normalized type
	}{
		{
			name:     "lowercase",
			json:     `{"type": "daily", "interval": 1}`,
			expected: "daily",
		},
		{
			name:     "uppercase",
			json:     `{"type": "DAILY", "interval": 1}`,
			expected: "daily",
		},
		{
			name:     "mixed case",
			json:     `{"type": "Weekly", "interval": 1}`,
			expected: "weekly",
		},
		{
			name:     "title case",
			json:     `{"type": "Monthly", "interval": 1}`,
			expected: "monthly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := ParseRecurrenceRuleFromJSON(tt.json)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(rule.Type))
		})
	}
}

func TestMatchesWeekday_NoMutation(t *testing.T) {
	// Test that matchesWeekday doesn't modify the receiver
	originalRule := &RecurrenceRule{
		Type:     RecurrenceTypeWeekly,
		Interval: 1,
		Weekdays: []int{1}, // Only Monday
	}

	// Create a copy to compare later
	originalWeekdays := make([]int, len(originalRule.Weekdays))
	copy(originalWeekdays, originalRule.Weekdays)

	// Call matchesWeekday multiple times
	monday := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)  // Monday
	tuesday := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC) // Tuesday

	_ = originalRule.matchesWeekday(monday)  // Should return true
	_ = originalRule.matchesWeekday(tuesday) // Should return false

	// Verify Weekdays wasn't modified
	assert.Equal(t, originalWeekdays, originalRule.Weekdays, "Weekdays should not be modified")
	assert.Equal(t, []int{1}, originalRule.Weekdays, "Weekdays should still contain only Monday")
}

func TestMatchesWeekday_DefaultWeekdays(t *testing.T) {
	// Test default weekdays (Mon-Fri) when Weekdays is empty
	rule := &RecurrenceRule{
		Type:     RecurrenceTypeWeekly,
		Interval: 1,
		Weekdays: []int{}, // Empty
	}

	// Test Monday-Friday should match
	monday := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)    // Monday
	tuesday := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)   // Tuesday
	wednesday := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC) // Wednesday
	thursday := time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC)  // Thursday
	friday := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)    // Friday

	assert.True(t, rule.matchesWeekday(monday))
	assert.True(t, rule.matchesWeekday(tuesday))
	assert.True(t, rule.matchesWeekday(wednesday))
	assert.True(t, rule.matchesWeekday(thursday))
	assert.True(t, rule.matchesWeekday(friday))

	// Test Saturday and Sunday should NOT match
	saturday := time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC) // Saturday
	sunday := time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC)   // Sunday

	assert.False(t, rule.matchesWeekday(saturday))
	assert.False(t, rule.matchesWeekday(sunday))

	// Verify Weekdays is still empty (not modified)
	assert.Empty(t, rule.Weekdays, "Weekdays should still be empty")
}

func TestGenerateInstances_Daily_Interval(t *testing.T) {
	// Test daily recurrence with interval > 1 (P0-1 fix verification)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	startTs := start.Unix()

	rule := &RecurrenceRule{Type: "daily", Interval: 3} // Every 3 days

	// End: Jan 16 exclusive (Jan 1 + 15 days)
	end := start.AddDate(0, 0, 15).Unix()
	instances := rule.GenerateInstances(startTs, end)

	// Jan 1, Jan 4, Jan 7, Jan 10, Jan 13, Jan 16 -> 6 instances (inclusive)
	// This verifies the fix: AddDate(0, 0, r.Interval) not AddDate(r.Interval, 0, 0)
	assert.Len(t, instances, 6)
	assert.Equal(t, startTs, instances[0])
	assert.Equal(t, start.AddDate(0, 0, 3).Unix(), instances[1])
	assert.Equal(t, start.AddDate(0, 0, 6).Unix(), instances[2])
	assert.Equal(t, start.AddDate(0, 0, 9).Unix(), instances[3])
	assert.Equal(t, start.AddDate(0, 0, 12).Unix(), instances[4])
	assert.Equal(t, start.AddDate(0, 0, 15).Unix(), instances[5])
}

func TestGenerateInstances_Weekly_Interval(t *testing.T) {
	// Start: 2024-01-01 (Monday)
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	startTs := start.Unix()

	// Every 2 weeks on Monday
	rule := &RecurrenceRule{Type: "weekly", Interval: 2, Weekdays: []int{1}}

	// Generate for 30 days
	end := start.AddDate(0, 0, 30).Unix()
	instances := rule.GenerateInstances(startTs, end)

	// Jan 1 (Mon), Jan 15 (Mon), Jan 29 (Mon)
	assert.Len(t, instances, 3)
	assert.Equal(t, startTs, instances[0])
	assert.Equal(t, start.AddDate(0, 0, 14).Unix(), instances[1])
	assert.Equal(t, start.AddDate(0, 0, 28).Unix(), instances[2])
}

func TestGenerateInstances_Monthly_FebruaryLeapYear(t *testing.T) {
	// Test Feb 29 in leap year and Feb 28 in non-leap year
	start := time.Date(2024, 1, 31, 10, 0, 0, 0, time.UTC)
	rule := &RecurrenceRule{Type: "monthly", MonthDay: 31, Interval: 1}

	// Generate through Feb 2025 (non-leap year)
	end := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC).Unix()
	instances := rule.GenerateInstances(start.Unix(), end)

	// Verify 2024 is leap year (Feb 29) and 2025 is not (Feb 28)
	for _, ts := range instances {
		tm := time.Unix(ts, 0).In(time.UTC)
		if tm.Month() == time.February {
			if tm.Year() == 2024 {
				assert.Equal(t, 29, tm.Day(), "Feb 2024 should have 29 days (leap year)")
			} else if tm.Year() == 2025 {
				assert.Equal(t, 28, tm.Day(), "Feb 2025 should have 28 days (non-leap year)")
			}
		}
	}
}
