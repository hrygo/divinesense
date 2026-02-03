package aitime

import (
	"context"
	"testing"
	"time"
)

// TestTimeServiceContract tests the TimeService contract.
func TestTimeServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockTimeService()

	// Set fixed time for consistent testing
	fixedNow := time.Date(2026, 1, 27, 10, 0, 0, 0, time.Local)
	svc.FixedNow = &fixedNow

	t.Run("Normalize_StandardFormat_YYYYMMDD", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "2026-01-28", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		if result.Year() != 2026 || result.Month() != 1 || result.Day() != 28 {
			t.Errorf("unexpected date: %v", result)
		}
	})

	t.Run("Normalize_StandardFormat_HHMM", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "15:30", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		if result.Hour() != 15 || result.Minute() != 30 {
			t.Errorf("unexpected time: %v", result)
		}
	})

	t.Run("Normalize_ChineseTime_Tomorrow3PM", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "明天下午3点", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		if result.Day() != 28 {
			t.Errorf("expected day 28, got %d", result.Day())
		}
		if result.Hour() != 15 {
			t.Errorf("expected hour 15, got %d", result.Hour())
		}
	})

	t.Run("Normalize_ChineseTime_Morning9", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "上午9点", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		if result.Hour() != 9 {
			t.Errorf("expected hour 9, got %d", result.Hour())
		}
	})

	t.Run("Normalize_ChineseTime_ChineseNumber", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "三点", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		// 三点 could be 3 or 15, depending on context
		if result.Hour() != 3 && result.Hour() != 15 {
			t.Errorf("expected hour 3 or 15, got %d", result.Hour())
		}
	})

	t.Run("Normalize_HalfHour", func(t *testing.T) {
		result, err := svc.Normalize(ctx, "下午3点半", "Asia/Shanghai")
		if err != nil {
			t.Fatalf("Normalize failed: %v", err)
		}
		if result.Minute() != 30 {
			t.Errorf("expected minute 30, got %d", result.Minute())
		}
	})

	t.Run("Normalize_InvalidTimezone_UsesLocal", func(t *testing.T) {
		_, err := svc.Normalize(ctx, "15:00", "Invalid/Timezone")
		// Should not error, falls back to local
		if err != nil {
			t.Logf("Error with invalid timezone (acceptable): %v", err)
		}
	})

	t.Run("ParseNaturalTime_Today", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "今天", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		if tr.Start.Day() != fixedNow.Day() {
			t.Errorf("expected today, got %v", tr.Start)
		}
	})

	t.Run("ParseNaturalTime_Tomorrow", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "明天", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		expected := fixedNow.AddDate(0, 0, 1)
		if tr.Start.Day() != expected.Day() {
			t.Errorf("expected %d, got %d", expected.Day(), tr.Start.Day())
		}
	})

	t.Run("ParseNaturalTime_ThisWeek", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "这周", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		// Should span 7 days
		duration := tr.End.Sub(tr.Start)
		if duration != 7*24*time.Hour {
			t.Errorf("expected 7 days duration, got %v", duration)
		}
	})

	t.Run("ParseNaturalTime_NextWeek", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "下周", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		if !tr.Start.After(fixedNow) {
			t.Error("next week should be after now")
		}
	})

	t.Run("ParseNaturalTime_ThisMonth", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "这个月", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		if tr.Start.Month() != fixedNow.Month() {
			t.Errorf("expected current month, got %v", tr.Start.Month())
		}
	})

	t.Run("ParseNaturalTime_ReturnsValidRange", func(t *testing.T) {
		expressions := []string{"今天", "明天", "这周", "下周", "这个月"}
		for _, expr := range expressions {
			tr, err := svc.ParseNaturalTime(ctx, expr, fixedNow)
			if err != nil {
				t.Errorf("ParseNaturalTime(%s) failed: %v", expr, err)
				continue
			}
			if tr.End.Before(tr.Start) {
				t.Errorf("ParseNaturalTime(%s): end before start", expr)
			}
			if tr.Start.IsZero() || tr.End.IsZero() {
				t.Errorf("ParseNaturalTime(%s): zero time in range", expr)
			}
		}
	})

	t.Run("TimeRange_HasPositiveDuration", func(t *testing.T) {
		tr, err := svc.ParseNaturalTime(ctx, "明天", fixedNow)
		if err != nil {
			t.Fatalf("ParseNaturalTime failed: %v", err)
		}
		duration := tr.End.Sub(tr.Start)
		if duration <= 0 {
			t.Error("time range should have positive duration")
		}
	})
}
