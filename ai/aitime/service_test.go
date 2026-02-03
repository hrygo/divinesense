package aitime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_StandardFormats(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	parser := NewParser(loc)

	tests := []struct {
		name     string
		input    string
		wantTime string // Expected time in format "2006-01-02 15:04"
	}{
		{"ISO date", "2026-01-28", "2026-01-28 00:00"},
		{"ISO datetime", "2026-01-28 15:30", "2026-01-28 15:30"},
		{"Slash date", "2026/01/28", "2026-01-28 00:00"},
		{"Chinese date", "2026年01月28日", "2026-01-28 00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTime, got.Format("2006-01-02 15:04"))
		})
	}
}

func TestParser_RelativeDates(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// Fix "now" to 2026-01-27 10:00
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)
	parser := &Parser{timezone: loc, now: func() time.Time { return fixedNow }}

	tests := []struct {
		name     string
		input    string
		wantDate string // Expected date in format "2006-01-02"
	}{
		{"今天", "今天", "2026-01-27"},
		{"明天", "明天", "2026-01-28"},
		{"后天", "后天", "2026-01-29"},
		{"昨天", "昨天", "2026-01-26"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDate, got.Format("2006-01-02"))
		})
	}
}

func TestParser_ChineseTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)
	parser := &Parser{timezone: loc, now: func() time.Time { return fixedNow }}

	tests := []struct {
		name     string
		input    string
		wantTime string // Expected time in format "15:04"
	}{
		{"3点", "3点", "15:00"}, // 1-6点 defaults to PM
		{"下午3点", "下午3点", "15:00"},
		{"上午9点", "上午9点", "09:00"},
		{"晚上8点", "晚上8点", "20:00"},
		{"3点半", "3点半", "15:30"}, // 1-6点 defaults to PM
		{"15点30分", "15点30分", "15:30"},
		{"三点", "三点", "15:00"}, // 1-6点 defaults to PM
		{"十二点", "十二点", "12:00"},
		{"9点", "9点", "09:00"},   // 7-11点 stays as AM (common work hours)
		{"10点", "10点", "10:00"}, // 7-11点 stays as AM
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTime, got.Format("15:04"))
		})
	}
}

func TestParser_CombinedExpressions(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)
	parser := &Parser{timezone: loc, now: func() time.Time { return fixedNow }}

	tests := []struct {
		name     string
		input    string
		wantTime string // Expected in format "2006-01-02 15:04"
	}{
		{"明天下午3点", "明天下午3点", "2026-01-28 15:00"},
		{"后天上午9点", "后天上午9点", "2026-01-29 09:00"},
		{"明天10点30分", "明天10点30分", "2026-01-28 10:30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTime, got.Format("2006-01-02 15:04"))
		})
	}
}

func TestParser_RelativeTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)
	parser := &Parser{timezone: loc, now: func() time.Time { return fixedNow }}

	tests := []struct {
		name     string
		input    string
		wantTime string
	}{
		{"1小时后", "1小时后", "2026-01-27 11:00"},
		{"2小时后", "2小时后", "2026-01-27 12:00"},
		{"30分钟后", "30分钟后", "2026-01-27 10:30"},
		{"1天后", "1天后", "2026-01-28 10:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTime, got.Format("2006-01-02 15:04"))
		})
	}
}

func TestService_Normalize(t *testing.T) {
	svc := NewService("Asia/Shanghai")
	ctx := context.Background()

	t.Run("StandardFormat", func(t *testing.T) {
		got, err := svc.Normalize(ctx, "2026-01-28 15:00", "Asia/Shanghai")
		require.NoError(t, err)
		assert.Equal(t, "2026-01-28 15:00", got.Format("2006-01-02 15:04"))
	})

	t.Run("ChineseExpression", func(t *testing.T) {
		got, err := svc.Normalize(ctx, "下午3点", "Asia/Shanghai")
		require.NoError(t, err)
		assert.Equal(t, 15, got.Hour())
	})
}

func TestService_ParseNaturalTime(t *testing.T) {
	svc := NewService("Asia/Shanghai")
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Shanghai")
	ref := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)

	t.Run("DayRange_今天", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "今天", ref)
		require.NoError(t, err)
		assert.Equal(t, "2026-01-27", tr.Start.Format("2006-01-02"))
		assert.Equal(t, "2026-01-28", tr.End.Format("2006-01-02"))
	})

	t.Run("WeekRange_这周", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "这周", ref)
		require.NoError(t, err)
		// 2026-01-27 is Tuesday, so Monday is 2026-01-26
		assert.Equal(t, "2026-01-26", tr.Start.Format("2006-01-02"))
		assert.Equal(t, "2026-02-02", tr.End.Format("2006-01-02"))
	})

	t.Run("SpecificTime", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "明天下午3点", ref)
		require.NoError(t, err)
		assert.Equal(t, "2026-01-28 15:00", tr.Start.Format("2006-01-02 15:04"))
		assert.Equal(t, time.Hour, tr.End.Sub(tr.Start))
	})
}

func TestParser_Weekday(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	// 2026-01-27 is Tuesday
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, loc)
	parser := &Parser{timezone: loc, now: func() time.Time { return fixedNow }}

	tests := []struct {
		name     string
		input    string
		wantDate string
	}{
		{"周一", "周一", "2026-01-26"},   // Monday of current week
		{"周三", "周三", "2026-01-28"},   // Wednesday of current week
		{"周日", "周日", "2026-02-01"},   // Sunday of current week
		{"下周一", "下周一", "2026-02-02"}, // Monday of next week
		{"下周三", "下周三", "2026-02-04"}, // Wednesday of next week
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDate, got.Format("2006-01-02"))
		})
	}
}
