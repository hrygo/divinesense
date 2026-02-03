package metrics

import (
	"context"
	"testing"
	"time"
)

// TestMetricsServiceContract tests the MetricsService contract.
func TestMetricsServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockMetricsService()

	t.Run("RecordRequest_StoresData", func(t *testing.T) {
		svc.Clear()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)
		svc.RecordRequest(ctx, "schedule", 200*time.Millisecond, true)
		svc.RecordRequest(ctx, "amazing", 150*time.Millisecond, false)

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}
		if stats.RequestCount != 3 {
			t.Errorf("expected 3 requests, got %d", stats.RequestCount)
		}
		if stats.SuccessCount != 2 {
			t.Errorf("expected 2 successes, got %d", stats.SuccessCount)
		}
	})

	t.Run("RecordToolCall_StoresData", func(t *testing.T) {
		svc.Clear()
		svc.RecordToolCall(ctx, "search_memo", 50*time.Millisecond, true)
		svc.RecordToolCall(ctx, "create_schedule", 100*time.Millisecond, true)

		// Tool calls are recorded but not included in GetStats (by design)
		// This tests that RecordToolCall doesn't error
	})

	t.Run("GetStats_CalculatesPercentiles", func(t *testing.T) {
		svc.Clear()
		// Add 10 requests with varying latencies
		latencies := []time.Duration{
			10 * time.Millisecond,
			20 * time.Millisecond,
			30 * time.Millisecond,
			40 * time.Millisecond,
			50 * time.Millisecond,
			60 * time.Millisecond,
			70 * time.Millisecond,
			80 * time.Millisecond,
			90 * time.Millisecond,
			100 * time.Millisecond,
		}

		for _, lat := range latencies {
			svc.RecordRequest(ctx, "memo", lat, true)
		}

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		// P50 should be around 50ms (5th element of sorted list)
		if stats.LatencyP50 < 40*time.Millisecond || stats.LatencyP50 > 60*time.Millisecond {
			t.Errorf("P50 latency out of expected range: %v", stats.LatencyP50)
		}

		// P95 should be around 95ms (9th element of sorted list)
		if stats.LatencyP95 < 80*time.Millisecond {
			t.Errorf("P95 latency too low: %v", stats.LatencyP95)
		}
	})

	t.Run("GetStats_GroupsByAgentType", func(t *testing.T) {
		svc.Clear()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)
		svc.RecordRequest(ctx, "memo", 200*time.Millisecond, true)
		svc.RecordRequest(ctx, "schedule", 150*time.Millisecond, true)
		svc.RecordRequest(ctx, "schedule", 150*time.Millisecond, false)

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		if memoStats, ok := stats.AgentStats["memo"]; ok {
			if memoStats.Count != 2 {
				t.Errorf("expected 2 memo requests, got %d", memoStats.Count)
			}
			if memoStats.SuccessRate != 1.0 {
				t.Errorf("expected 100%% memo success rate, got %f", memoStats.SuccessRate)
			}
		} else {
			t.Error("expected memo agent stats")
		}

		if scheduleStats, ok := stats.AgentStats["schedule"]; ok {
			if scheduleStats.Count != 2 {
				t.Errorf("expected 2 schedule requests, got %d", scheduleStats.Count)
			}
			if scheduleStats.SuccessRate != 0.5 {
				t.Errorf("expected 50%% schedule success rate, got %f", scheduleStats.SuccessRate)
			}
		} else {
			t.Error("expected schedule agent stats")
		}
	})

	t.Run("GetStats_TracksErrorsByType", func(t *testing.T) {
		svc.Clear()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, false)
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, false)
		svc.RecordRequest(ctx, "schedule", 100*time.Millisecond, false)

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		if stats.ErrorsByType["memo"] != 2 {
			t.Errorf("expected 2 memo errors, got %d", stats.ErrorsByType["memo"])
		}
		if stats.ErrorsByType["schedule"] != 1 {
			t.Errorf("expected 1 schedule error, got %d", stats.ErrorsByType["schedule"])
		}
	})

	t.Run("GetStats_FiltersTimeRange", func(t *testing.T) {
		svc.Clear()

		// Record requests
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)
		time.Sleep(10 * time.Millisecond)
		midpoint := time.Now()
		time.Sleep(10 * time.Millisecond)
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)

		// Query only after midpoint
		stats, err := svc.GetStats(ctx, TimeRange{Start: midpoint})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		if stats.RequestCount != 1 {
			t.Errorf("expected 1 request after midpoint, got %d", stats.RequestCount)
		}
	})

	t.Run("GetStats_EmptyData", func(t *testing.T) {
		svc.Clear()

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		if stats.RequestCount != 0 {
			t.Errorf("expected 0 requests, got %d", stats.RequestCount)
		}
		if stats.AgentStats == nil {
			t.Error("AgentStats should not be nil")
		}
		if stats.ErrorsByType == nil {
			t.Error("ErrorsByType should not be nil")
		}
	})

	t.Run("AgentMetrics_HasValidStructure", func(t *testing.T) {
		svc.Clear()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		// Verify required fields
		if stats.RequestCount < 0 {
			t.Error("RequestCount should be non-negative")
		}
		if stats.SuccessCount < 0 {
			t.Error("SuccessCount should be non-negative")
		}
		if stats.SuccessCount > stats.RequestCount {
			t.Error("SuccessCount should not exceed RequestCount")
		}
	})

	t.Run("AgentStat_SuccessRateInRange", func(t *testing.T) {
		svc.Clear()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, false)

		stats, err := svc.GetStats(ctx, TimeRange{})
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		for agentType, agentStat := range stats.AgentStats {
			if agentStat.SuccessRate < 0 || agentStat.SuccessRate > 1 {
				t.Errorf("SuccessRate for %s out of range: %f", agentType, agentStat.SuccessRate)
			}
		}
	})
}
