package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPersisterConfig(t *testing.T) {
	config := DefaultPersisterConfig()

	assert.Equal(t, time.Hour, config.FlushInterval)
	assert.Equal(t, 30*24*time.Hour, config.RetentionPeriod)
	assert.Equal(t, 24*time.Hour, config.CleanupInterval)
}

func TestNewPersister_ConfigDefaults(t *testing.T) {
	agg := NewAggregator()

	t.Run("Zero config uses defaults", func(t *testing.T) {
		persister := NewPersister(nil, agg, PersisterConfig{})

		assert.Equal(t, time.Hour, persister.flushInterval)
		assert.Equal(t, 30*24*time.Hour, persister.retentionPeriod)
		assert.Equal(t, 24*time.Hour, persister.cleanupInterval)
	})

	t.Run("Custom config", func(t *testing.T) {
		customCfg := PersisterConfig{
			FlushInterval:   30 * time.Minute,
			RetentionPeriod: 7 * 24 * time.Hour,
			CleanupInterval: 12 * time.Hour,
		}

		persister := NewPersister(nil, agg, customCfg)

		assert.Equal(t, 30*time.Minute, persister.flushInterval)
		assert.Equal(t, 7*24*time.Hour, persister.retentionPeriod)
		assert.Equal(t, 12*time.Hour, persister.cleanupInterval)
	})
}

func TestNewPersister_WithAggregator(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())

	assert.NotNil(t, persister)
	assert.NotNil(t, persister.aggregator)
	assert.Same(t, agg, persister.aggregator)
}

func TestPersister_Start(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())

	// Start should not panic
	persister.Start()

	// Give goroutines time to start
	time.Sleep(10 * time.Millisecond)

	// Close should clean up
	persister.Close()
}

func TestPersister_Close(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())

	// Close without starting should not panic
	persister.Close()
}

func TestPersister_StartAndClose(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())

	persister.Start()
	time.Sleep(10 * time.Millisecond)
	persister.Close()

	// Should be able to start again after close
	persister2 := NewPersister(nil, agg, DefaultPersisterConfig())
	persister2.Start()
	persister2.Close()
}

func TestPersister_Flush(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())

	ctx := context.Background()

	// Record metrics in current hour - these won't be flushed
	// because Flush only returns past hour buckets
	agg.RecordAgentRequest("memo", 100*time.Millisecond, true)
	agg.RecordAgentRequest("schedule", 200*time.Millisecond, false)

	// Flush will have no past buckets to process, so no store calls needed
	err := persister.Flush(ctx)
	assert.NoError(t, err)
}

func TestPersister_FlushAggregatorOnly(t *testing.T) {
	agg := NewAggregator()
	_ = NewPersister(nil, agg, DefaultPersisterConfig())

	// Record current hour metrics (won't be flushed)
	agg.RecordAgentRequest("memo", 100*time.Millisecond, true)

	// Call aggregator's flush directly to verify it works
	currentHour := truncateToHour(time.Now())
	snapshots := agg.FlushAgentMetrics(currentHour)

	// No past hour buckets, so snapshots should be empty
	assert.Empty(t, snapshots)
}

func TestPersister_ConcurrentFlush(t *testing.T) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())
	ctx := context.Background()

	// Record many metrics
	for i := 0; i < 100; i++ {
		agg.RecordAgentRequest("memo", time.Duration(i)*time.Millisecond, i%2 == 0)
		agg.RecordToolCall("search", 50*time.Millisecond, true)
	}

	// Concurrent flushes (no past buckets, so store won't be called)
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			_ = persister.Flush(ctx)
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// Benchmark tests

func BenchmarkPersister_Flush(b *testing.B) {
	agg := NewAggregator()
	persister := NewPersister(nil, agg, DefaultPersisterConfig())
	ctx := context.Background()

	// Pre-populate with metrics
	for i := 0; i < 1000; i++ {
		agg.RecordAgentRequest("memo", 100*time.Millisecond, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = persister.Flush(ctx)
	}
}

func BenchmarkAggregator_RecordAgentRequest(b *testing.B) {
	agg := NewAggregator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.RecordAgentRequest("memo", 100*time.Millisecond, true)
	}
}

func BenchmarkAggregator_RecordToolCall(b *testing.B) {
	agg := NewAggregator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.RecordToolCall("search", 50*time.Millisecond, true)
	}
}

func BenchmarkAggregator_GetCurrentStats(b *testing.B) {
	agg := NewAggregator()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		agg.RecordAgentRequest("memo", time.Duration(i)*time.Millisecond, true)
		agg.RecordToolCall("search", 50*time.Millisecond, true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.GetCurrentStats()
	}
}

func BenchmarkTruncateToHour(b *testing.B) {
	t := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = truncateToHour(t)
	}
}

func BenchmarkMakeAgentKey(b *testing.B) {
	hour := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = makeAgentKey(hour, "memo")
	}
}

func BenchmarkMakeToolKey(b *testing.B) {
	hour := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = makeToolKey(hour, "search")
	}
}

func BenchmarkSumLatencies(b *testing.B) {
	latencies := make([]int64, 1000)
	for i := range latencies {
		latencies[i] = int64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sumLatencies(latencies)
	}
}

func BenchmarkPercentile_Sorted(b *testing.B) {
	latencies := make([]int64, 1000)
	for i := range latencies {
		latencies[i] = int64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = percentile(latencies, 50)
	}
}

func BenchmarkPercentile_Unsorted(b *testing.B) {
	latencies := make([]int64, 1000)
	for i := range latencies {
		latencies[i] = int64(i)
	}
	// Shuffle
	for i := range latencies {
		j := int(i) % len(latencies)
		latencies[i], latencies[j] = latencies[j], latencies[i]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = percentile(latencies, 95)
	}
}
