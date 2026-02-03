package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchScheduleParser_DetectRecurrence(t *testing.T) {
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		name     string
		input    string
		wantType RecurrenceType
		wantDays []int
	}{
		{
			name:     "daily",
			input:    "每天早上9点站会",
			wantType: RecurrenceTypeDaily,
		},
		{
			name:     "daily_alt",
			input:    "每日下午2点汇报",
			wantType: RecurrenceTypeDaily,
		},
		{
			name:     "workdays",
			input:    "每个工作日上午10点例会",
			wantType: RecurrenceTypeWeekly,
			wantDays: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "weekly_monday",
			input:    "每周一下午2点例会",
			wantType: RecurrenceTypeWeekly,
			wantDays: []int{1},
		},
		{
			name:     "weekly_friday",
			input:    "每周五下午3点周会",
			wantType: RecurrenceTypeWeekly,
			wantDays: []int{5},
		},
		{
			name:     "weekly_multiple",
			input:    "每周一三五上午9点站会",
			wantType: RecurrenceTypeWeekly,
			wantDays: []int{1, 3, 5},
		},
		{
			name:     "monthly",
			input:    "每月15号下午汇报",
			wantType: RecurrenceTypeMonthly,
		},
		{
			name:     "monthly_1st",
			input:    "每月1日团队总结",
			wantType: RecurrenceTypeMonthly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := parser.detectRecurrence(tt.input)
			require.NotNil(t, rule, "should detect recurrence for: %s", tt.input)
			assert.Equal(t, tt.wantType, rule.Type)
			if len(tt.wantDays) > 0 {
				assert.Equal(t, tt.wantDays, rule.Weekdays)
			}
		})
	}
}

func TestBatchScheduleParser_DetectRecurrence_NoMatch(t *testing.T) {
	parser := NewBatchScheduleParser(nil)

	inputs := []string{
		"明天下午开会",
		"周一开会",
		"下周三见面",
	}

	for _, input := range inputs {
		rule := parser.detectRecurrence(input)
		assert.Nil(t, rule, "should not detect recurrence for: %s", input)
	}
}

func TestBatchScheduleParser_ExtractBatchTitle(t *testing.T) {
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		input string
		want  string
	}{
		{"每周一下午2点例会", "例会"},
		{"每天早上9点站会", "站会"},
		{"每个工作日上午10点晨会", "晨会"},
		{"每月15号下午团队汇报", "团队汇报"},
		{"每周五下午3点周总结会议", "周总结会议"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parser.extractBatchTitle(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBatchScheduleParser_SimpleTimeExtract(t *testing.T) {
	parser := NewBatchScheduleParser(nil)
	now := time.Now()

	tests := []struct {
		input    string
		wantHour int
		wantMin  int
	}{
		{"上午9点", 9, 0},
		{"早上10点30分", 10, 30},
		{"下午2点", 14, 0},
		{"下午3点15分", 15, 15},
		{"14点", 14, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parser.simpleTimeExtract(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHour, got.Hour())
			assert.Equal(t, tt.wantMin, got.Minute())
			assert.Equal(t, now.Year(), got.Year())
		})
	}
}

func TestBatchScheduleParser_Parse(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		name         string
		input        string
		wantTitle    string
		wantType     RecurrenceType
		wantCountGte int
		wantBatch    bool
	}{
		{
			name:         "weekly_monday_meeting",
			input:        "每周一下午2点例会",
			wantBatch:    true,
			wantTitle:    "例会",
			wantType:     RecurrenceTypeWeekly,
			wantCountGte: 10,
		},
		{
			name:         "daily_standup",
			input:        "每天早上9点站会",
			wantBatch:    true,
			wantTitle:    "站会",
			wantType:     RecurrenceTypeDaily,
			wantCountGte: 10,
		},
		{
			name:         "workday_meeting",
			input:        "每个工作日上午10点晨会",
			wantBatch:    true,
			wantTitle:    "晨会",
			wantType:     RecurrenceTypeWeekly,
			wantCountGte: 10,
		},
		{
			name:         "monthly_report",
			input:        "每月15号下午3点月度汇报",
			wantBatch:    true,
			wantTitle:    "月度汇报",
			wantType:     RecurrenceTypeMonthly,
			wantCountGte: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(ctx, tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBatch, result.CanBatchCreate, "CanBatchCreate for: %s", tt.input)

			if tt.wantBatch {
				assert.Equal(t, tt.wantTitle, result.Request.Title)
				assert.Equal(t, tt.wantType, result.Request.Recurrence.Type)
				assert.GreaterOrEqual(t, result.TotalCount, tt.wantCountGte)
				assert.NotEmpty(t, result.Preview)
			}
		})
	}
}

func TestBatchScheduleParser_Parse_MissingFields(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		name         string
		input        string
		missingField string
	}{
		{
			name:         "no_recurrence",
			input:        "明天下午开会",
			missingField: "recurrence",
		},
		{
			name:         "no_time",
			input:        "每周一开会",
			missingField: "time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(ctx, tt.input)
			require.NoError(t, err)
			assert.False(t, result.CanBatchCreate)
			assert.Contains(t, result.MissingFields, tt.missingField)
		})
	}
}

func TestBatchScheduleService_GenerateSchedules(t *testing.T) {
	svc := NewBatchScheduleService(nil)

	// Find next Monday
	now := time.Now()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMonday := now.AddDate(0, 0, daysUntilMonday)
	nextMonday = time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 14, 0, 0, 0, now.Location())

	req := &BatchCreateRequest{
		Title:     "周例会",
		StartTime: nextMonday,
		Duration:  60,
		Recurrence: &RecurrenceRule{
			Type:     RecurrenceTypeWeekly,
			Interval: 1,
			Weekdays: []int{1},
		},
		Count: 4,
	}

	schedules, err := svc.GenerateSchedules(req)
	require.NoError(t, err)
	assert.Len(t, schedules, 4)

	for _, s := range schedules {
		assert.Equal(t, "周例会", s.Title)
		assert.Equal(t, 60, s.Duration)
		assert.Equal(t, time.Monday, s.StartTime.Weekday())
	}
}

func TestBatchScheduleService_ValidateRequest(t *testing.T) {
	svc := NewBatchScheduleService(nil)

	tests := []struct {
		req     *BatchCreateRequest
		name    string
		wantErr bool
	}{
		{
			name:    "nil_request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "missing_title",
			req: &BatchCreateRequest{
				StartTime:  time.Now(),
				Recurrence: &RecurrenceRule{Type: RecurrenceTypeDaily, Interval: 1},
			},
			wantErr: true,
		},
		{
			name: "missing_start_time",
			req: &BatchCreateRequest{
				Title:      "Test",
				Recurrence: &RecurrenceRule{Type: RecurrenceTypeDaily, Interval: 1},
			},
			wantErr: true,
		},
		{
			name: "missing_recurrence",
			req: &BatchCreateRequest{
				Title:     "Test",
				StartTime: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "valid_request",
			req: &BatchCreateRequest{
				Title:      "Test",
				StartTime:  time.Now(),
				Recurrence: &RecurrenceRule{Type: RecurrenceTypeDaily, Interval: 1},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseWeekday(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"一", 1},
		{"二", 2},
		{"三", 3},
		{"四", 4},
		{"五", 5},
		{"六", 6},
		{"日", 7},
		{"天", 7},
		{"x", 0},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, parseWeekday(tt.input))
	}
}

func TestParseMultipleWeekdays(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"一三五", []int{1, 3, 5}},
		{"二四", []int{2, 4}},
		{"一二三四五", []int{1, 2, 3, 4, 5}},
		{"六日", []int{6, 7}},
		{"一一一", []int{1}}, // Deduplication
	}

	for _, tt := range tests {
		got := parseMultipleWeekdays(tt.input)
		assert.Equal(t, tt.want, got, "for input: %s", tt.input)
	}
}

func TestBatchCreateResult_Preview(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	result, err := parser.Parse(ctx, "每周一下午2点周例会")
	require.NoError(t, err)
	require.True(t, result.CanBatchCreate)

	// Check preview has consistent data
	for i, schedule := range result.Preview {
		assert.Equal(t, "周例会", schedule.Title, "schedule %d title", i)
		assert.False(t, schedule.StartTime.IsZero(), "schedule %d start time", i)
		assert.False(t, schedule.EndTime.IsZero(), "schedule %d end time", i)
		assert.Equal(t, 60, schedule.Duration, "schedule %d duration", i)

		// Verify all schedules fall on Monday (weekday 1)
		weekday := int(schedule.StartTime.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		assert.Equal(t, 1, weekday, "schedule %d should be Monday", i)
	}
}

func TestBatchScheduleParser_AlignToFirstWeekday(t *testing.T) {
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		name        string
		inputDay    time.Weekday
		weekdays    []int
		expectDelta int
	}{
		{"already_on_target", time.Monday, []int{1}, 0},
		{"wed_to_monday", time.Wednesday, []int{1}, 5},                   // Wed->Mon = +5 days
		{"wed_to_friday", time.Wednesday, []int{5}, 2},                   // Wed->Fri = +2 days
		{"sunday_to_monday", time.Sunday, []int{1}, 1},                   // Sun->Mon = +1 day
		{"monday_to_wed_fri", time.Monday, []int{3, 5}, 2},               // Mon->Wed = +2 days
		{"saturday_to_workdays", time.Saturday, []int{1, 2, 3, 4, 5}, 2}, // Sat->Mon = +2 days
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a base time and find the next occurrence of inputDay
			base := time.Date(2026, 1, 27, 14, 0, 0, 0, time.Local) // A known Monday
			// Adjust base to be on the input day
			daysToInputDay := int(tt.inputDay) - int(base.Weekday())
			if daysToInputDay < 0 {
				daysToInputDay += 7
			}
			inputTime := base.AddDate(0, 0, daysToInputDay)

			result := parser.alignToFirstWeekday(inputTime, tt.weekdays)
			delta := int(result.Sub(inputTime).Hours() / 24)

			assert.Equal(t, tt.expectDelta, delta, "days delta for %s", tt.name)
		})
	}
}

func BenchmarkBatchScheduleParser_Parse(b *testing.B) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)
	input := "每周一下午2点例会"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(ctx, input)
	}
}

func BenchmarkBatchScheduleService_GenerateSchedules(b *testing.B) {
	svc := NewBatchScheduleService(nil)
	req := &BatchCreateRequest{
		Title:     "周例会",
		StartTime: time.Now(),
		Duration:  60,
		Recurrence: &RecurrenceRule{
			Type:     RecurrenceTypeWeekly,
			Interval: 1,
			Weekdays: []int{1},
		},
		Count: 52,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GenerateSchedules(req)
	}
}

// ============ Edge Case Tests ============

func TestBatchScheduleParser_MaxInstances(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	// Daily schedule should be capped at 52 instances
	result, err := parser.Parse(ctx, "每天上午9点站会")
	require.NoError(t, err)
	require.True(t, result.CanBatchCreate)

	// applyBatchDefaults caps Count at 52
	assert.LessOrEqual(t, result.Request.Count, 52, "count should be capped at 52")
	assert.LessOrEqual(t, len(result.Preview), 52, "preview should not exceed 52 instances")
}

func TestBatchScheduleParser_MonthlyBoundary(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		input       string
		expectedDay int
	}{
		{"每月15号下午3点汇报", 15},
		{"每月1号上午10点月度会议", 1},
		{"每月28号下午2点发薪日提醒", 28},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.Parse(ctx, tt.input)
			require.NoError(t, err)
			require.True(t, result.CanBatchCreate)
			require.NotNil(t, result.Request.Recurrence)

			assert.Equal(t, RecurrenceTypeMonthly, result.Request.Recurrence.Type)
			assert.Equal(t, tt.expectedDay, result.Request.Recurrence.MonthDay)
		})
	}
}

func TestBatchScheduleParser_WorkdaysOnly(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	result, err := parser.Parse(ctx, "每个工作日上午9点晨会")
	require.NoError(t, err)
	require.True(t, result.CanBatchCreate)
	require.NotNil(t, result.Request.Recurrence)

	// Should have weekdays 1-5 (Monday to Friday)
	assert.Equal(t, RecurrenceTypeWeekly, result.Request.Recurrence.Type)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, result.Request.Recurrence.Weekdays)
}

func TestBatchScheduleParser_MultipleWeekdays(t *testing.T) {
	ctx := context.Background()
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		input            string
		expectedWeekdays []int
	}{
		{"每周一三五下午2点健身", []int{1, 3, 5}},
		{"每周二四下午3点培训", []int{2, 4}},
		{"每周六日上午10点家庭活动", []int{6, 7}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.Parse(ctx, tt.input)
			require.NoError(t, err)
			require.True(t, result.CanBatchCreate)
			require.NotNil(t, result.Request.Recurrence)

			assert.Equal(t, RecurrenceTypeWeekly, result.Request.Recurrence.Type)
			assert.Equal(t, tt.expectedWeekdays, result.Request.Recurrence.Weekdays)
		})
	}
}

func TestBatchScheduleService_GenerateSchedules_EdgeCases(t *testing.T) {
	svc := NewBatchScheduleService(nil)

	t.Run("nil_request", func(t *testing.T) {
		_, err := svc.GenerateSchedules(nil)
		assert.Error(t, err)
	})

	t.Run("nil_recurrence", func(t *testing.T) {
		req := &BatchCreateRequest{
			Title:      "Test",
			StartTime:  time.Now(),
			Recurrence: nil,
		}
		_, err := svc.GenerateSchedules(req)
		assert.Error(t, err)
	})

	t.Run("end_date_limit", func(t *testing.T) {
		// Create request with end date 1 month from now
		endDate := time.Now().AddDate(0, 1, 0)
		req := &BatchCreateRequest{
			Title:     "Monthly Limited",
			StartTime: time.Now(),
			Duration:  60,
			Recurrence: &RecurrenceRule{
				Type:     RecurrenceTypeDaily,
				Interval: 1,
			},
			EndDate: &endDate,
		}

		schedules, err := svc.GenerateSchedules(req)
		require.NoError(t, err)

		// Should have approximately 30 instances (daily for 1 month)
		assert.Greater(t, len(schedules), 25)
		assert.LessOrEqual(t, len(schedules), 35)
	})
}

func TestBatchScheduleParser_TimeExtraction_EdgeCases(t *testing.T) {
	parser := NewBatchScheduleParser(nil)

	tests := []struct {
		input        string
		expectedHour int
		expectedMin  int
	}{
		{"下午12点", 12, 0},     // Noon
		{"下午12点30分", 12, 30}, // Noon with minutes
		{"上午11点45分", 11, 45}, // Late morning
		{"下午6点", 18, 0},      // Evening
		{"下午11点", 23, 0},     // Late night
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.simpleTimeExtract(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedHour, result.Hour(), "hour for %s", tt.input)
			assert.Equal(t, tt.expectedMin, result.Minute(), "minute for %s", tt.input)
		})
	}
}
