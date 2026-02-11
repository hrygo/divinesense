package metrics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_RecordAgentRequest(t *testing.T) {
	agg := NewAggregator()

	t.Run("SingleRequest", func(t *testing.T) {
		agg.RecordAgentRequest("memo", 100*time.Millisecond, true)

		stats := agg.GetCurrentStats()
		assert.Equal(t, int64(1), stats.RequestCount)
		assert.Equal(t, int64(1), stats.SuccessCount)
		assert.Contains(t, stats.AgentStats, "memo")
		assert.Equal(t, int64(1), stats.AgentStats["memo"].Count)
		assert.Equal(t, float32(1.0), stats.AgentStats["memo"].SuccessRate)
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		agg := NewAggregator()
		agg.RecordAgentRequest("schedule", 50*time.Millisecond, true)
		agg.RecordAgentRequest("schedule", 150*time.Millisecond, true)
		agg.RecordAgentRequest("schedule", 200*time.Millisecond, false)

		stats := agg.GetCurrentStats()
		assert.Equal(t, int64(3), stats.RequestCount)
		assert.Equal(t, int64(2), stats.SuccessCount)

		scheduleStat := stats.AgentStats["schedule"]
		require.NotNil(t, scheduleStat)
		assert.Equal(t, int64(3), scheduleStat.Count)
		assert.InDelta(t, 0.666, scheduleStat.SuccessRate, 0.01)
	})
}

func TestAggregator_RecordToolCall(t *testing.T) {
	agg := NewAggregator()

	agg.RecordToolCall("create_memo", 30*time.Millisecond, true)
	agg.RecordToolCall("create_memo", 40*time.Millisecond, true)
	agg.RecordToolCall("search_memo", 100*time.Millisecond, false)

	// Tool metrics are separate from agent stats
	stats := agg.GetCurrentStats()
	assert.Equal(t, int64(0), stats.RequestCount) // Agent requests only
}

func TestAggregator_Percentiles(t *testing.T) {
	agg := NewAggregator()

	// Record 100 requests with varying latencies
	for i := 1; i <= 100; i++ {
		agg.RecordAgentRequest("memo", time.Duration(i)*time.Millisecond, true)
	}

	stats := agg.GetCurrentStats()
	// P50 should be around 50ms
	assert.InDelta(t, 50, stats.LatencyP50.Milliseconds(), 5)
	// P95 should be around 95ms
	assert.InDelta(t, 95, stats.LatencyP95.Milliseconds(), 5)
}

func TestAggregator_FlushAgentMetrics(t *testing.T) {
	agg := NewAggregator()

	// Record some metrics
	agg.RecordAgentRequest("memo", 100*time.Millisecond, true)
	agg.RecordAgentRequest("memo", 200*time.Millisecond, true)

	// Flush metrics for past hours (not current hour)
	// Since we just recorded, they're in the current hour bucket
	currentHour := truncateToHour(time.Now())
	snapshots := agg.FlushAgentMetrics(currentHour)

	// Should be empty since current hour is not flushed
	assert.Empty(t, snapshots)

	// Verify metrics are still in aggregator
	stats := agg.GetCurrentStats()
	assert.Equal(t, int64(2), stats.RequestCount)
}

func TestAggregator_FlushToolMetrics(t *testing.T) {
	agg := NewAggregator()

	agg.RecordToolCall("create_memo", 50*time.Millisecond, true)

	currentHour := truncateToHour(time.Now())
	snapshots := agg.FlushToolMetrics(currentHour)

	// Should be empty since current hour is not flushed
	assert.Empty(t, snapshots)
}

func TestAggregator_ConcurrentAccess(t *testing.T) {
	agg := NewAggregator()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			agg.RecordAgentRequest("memo", 10*time.Millisecond, true)
		}()
		go func() {
			defer wg.Done()
			agg.RecordToolCall("create_memo", 5*time.Millisecond, true)
		}()
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = agg.GetCurrentStats()
		}()
	}

	wg.Wait()

	stats := agg.GetCurrentStats()
	assert.Equal(t, int64(100), stats.RequestCount)
}

func TestService_RecordAndGetStats(t *testing.T) {
	// Create service without persistence
	svc := NewService(nil, DefaultPersisterConfig())
	defer svc.Close()

	t.Run("RecordRequest", func(t *testing.T) {
		ctx := context.Background()
		svc.RecordRequest(ctx, "memo", 100*time.Millisecond, true)
		svc.RecordRequest(ctx, "schedule", 200*time.Millisecond, false)

		stats, err := svc.GetStats(ctx, TimeRange{
			Start: time.Now().Add(-time.Hour),
			End:   time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.RequestCount)
		assert.Equal(t, int64(1), stats.SuccessCount)
	})

	t.Run("RecordToolCall", func(t *testing.T) {
		ctx := context.Background()
		svc.RecordToolCall(ctx, "create_memo", 50*time.Millisecond, true)

		// Tool calls don't affect agent request count
		stats, err := svc.GetStats(ctx, TimeRange{
			Start: time.Now().Add(-time.Hour),
			End:   time.Now(),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), stats.RequestCount) // Same as before
	})
}

func TestService_NoPersistence(t *testing.T) {
	svc := NewService(nil, DefaultPersisterConfig())
	defer svc.Close()

	assert.False(t, svc.HasPersistence())

	err := svc.Flush(context.Background())
	assert.ErrorIs(t, err, ErrMetricsNotConfigured)
}

func TestService_Close(t *testing.T) {
	svc := NewService(nil, DefaultPersisterConfig())

	// Should not panic
	svc.Close()
}

func TestPercentile(t *testing.T) {
	tests := []struct {
		name      string
		latencies []int64
		p         int
		want      int64
	}{
		{"empty", []int64{}, 50, 0},
		{"single", []int64{100}, 50, 100},
		{"p50", []int64{10, 20, 30, 40, 50}, 50, 30},
		{"p95", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, 95, 90},
		{"p0", []int64{10, 20, 30}, 0, 10},
		{"p100", []int64{10, 20, 30}, 100, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.latencies, tt.p)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateToHour(t *testing.T) {
	input := time.Date(2026, 1, 27, 14, 35, 22, 123456789, time.UTC)
	expected := time.Date(2026, 1, 27, 14, 0, 0, 0, time.UTC)

	result := truncateToHour(input)
	assert.Equal(t, expected, result)
}
