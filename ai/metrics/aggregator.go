package metrics

import (
	"sort"
	"sync"
	"time"
)

// Aggregator aggregates metrics in memory before persisting to database.
type Aggregator struct {
	agentMetrics map[string]*agentBucket
	toolMetrics  map[string]*toolBucket
	mu           sync.RWMutex
}

type agentBucket struct {
	hourBucket   time.Time
	agentType    string
	latencies    []int64
	requestCount int64
	successCount int64
}

const maxLatencySamples = 10000 // Cap latency samples to prevent memory bloat

type toolBucket struct {
	hourBucket   time.Time
	toolName     string
	callCount    int64
	successCount int64
	latencySum   int64 // in milliseconds
}

// NewAggregator creates a new metrics aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{
		agentMetrics: make(map[string]*agentBucket),
		toolMetrics:  make(map[string]*toolBucket),
	}
}

// RecordAgentRequest records a single agent request.
func (a *Aggregator) RecordAgentRequest(agentType string, latency time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	hourBucket := truncateToHour(time.Now())
	key := makeAgentKey(hourBucket, agentType)

	bucket, exists := a.agentMetrics[key]
	if !exists {
		bucket = &agentBucket{
			hourBucket: hourBucket,
			agentType:  agentType,
			latencies:  make([]int64, 0, 100),
		}
		a.agentMetrics[key] = bucket
	}

	bucket.requestCount++
	if success {
		bucket.successCount++
	}
	// Reservoir sampling: keep maxLatencySamples samples for percentile calculation
	if len(bucket.latencies) < maxLatencySamples {
		bucket.latencies = append(bucket.latencies, latency.Milliseconds())
	}
}

// RecordToolCall records a single tool call.
func (a *Aggregator) RecordToolCall(toolName string, latency time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	hourBucket := truncateToHour(time.Now())
	key := makeToolKey(hourBucket, toolName)

	bucket, exists := a.toolMetrics[key]
	if !exists {
		bucket = &toolBucket{
			hourBucket: hourBucket,
			toolName:   toolName,
		}
		a.toolMetrics[key] = bucket
	}

	bucket.callCount++
	if success {
		bucket.successCount++
	}
	bucket.latencySum += latency.Milliseconds()
}

// AgentSnapshot represents a snapshot of agent metrics for persistence.
type AgentSnapshot struct {
	HourBucket   time.Time
	AgentType    string
	RequestCount int64
	SuccessCount int64
	LatencySumMs int64
	LatencyP50Ms int32
	LatencyP95Ms int32
}

// ToolSnapshot represents a snapshot of tool metrics for persistence.
type ToolSnapshot struct {
	HourBucket   time.Time
	ToolName     string
	CallCount    int64
	SuccessCount int64
	LatencySumMs int64
}

// FlushAgentMetrics returns and clears all agent metrics for the given hour.
// Returns nil if no metrics exist for hours before the current hour.
func (a *Aggregator) FlushAgentMetrics(beforeHour time.Time) []*AgentSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	var snapshots []*AgentSnapshot
	keysToDelete := make([]string, 0)

	for key, bucket := range a.agentMetrics {
		if bucket.hourBucket.Before(beforeHour) {
			snapshot := &AgentSnapshot{
				HourBucket:   bucket.hourBucket,
				AgentType:    bucket.agentType,
				RequestCount: bucket.requestCount,
				SuccessCount: bucket.successCount,
				LatencySumMs: sumLatencies(bucket.latencies),
				LatencyP50Ms: int32(percentile(bucket.latencies, 50)),
				LatencyP95Ms: int32(percentile(bucket.latencies, 95)),
			}
			snapshots = append(snapshots, snapshot)
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(a.agentMetrics, key)
	}

	return snapshots
}

// FlushToolMetrics returns and clears all tool metrics for hours before the given time.
func (a *Aggregator) FlushToolMetrics(beforeHour time.Time) []*ToolSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	var snapshots []*ToolSnapshot
	keysToDelete := make([]string, 0)

	for key, bucket := range a.toolMetrics {
		if bucket.hourBucket.Before(beforeHour) {
			snapshot := &ToolSnapshot{
				HourBucket:   bucket.hourBucket,
				ToolName:     bucket.toolName,
				CallCount:    bucket.callCount,
				SuccessCount: bucket.successCount,
				LatencySumMs: bucket.latencySum,
			}
			snapshots = append(snapshots, snapshot)
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(a.toolMetrics, key)
	}

	return snapshots
}

// GetCurrentStats returns aggregated stats from memory for the current hour.
func (a *Aggregator) GetCurrentStats() *AgentMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := &AgentMetrics{
		AgentStats:   make(map[string]*AgentStat),
		ErrorsByType: make(map[string]int64),
	}

	// Temporary aggregation for each agent type
	type agentAgg struct {
		totalRequests int64
		totalSuccess  int64
		latencySum    int64
	}
	agentAggs := make(map[string]*agentAgg)

	// Aggregate all agent metrics
	allLatencies := make([]int64, 0)
	for _, bucket := range a.agentMetrics {
		stats.RequestCount += bucket.requestCount
		stats.SuccessCount += bucket.successCount
		allLatencies = append(allLatencies, bucket.latencies...)

		agg, exists := agentAggs[bucket.agentType]
		if !exists {
			agg = &agentAgg{}
			agentAggs[bucket.agentType] = agg
		}
		agg.totalRequests += bucket.requestCount
		agg.totalSuccess += bucket.successCount
		agg.latencySum += sumLatencies(bucket.latencies)
	}

	// Build AgentStats from aggregations
	for agentType, agg := range agentAggs {
		stat := &AgentStat{Count: agg.totalRequests}
		if agg.totalRequests > 0 {
			stat.SuccessRate = float32(agg.totalSuccess) / float32(agg.totalRequests)
			stat.AvgLatency = time.Duration(agg.latencySum/agg.totalRequests) * time.Millisecond
		}
		stats.AgentStats[agentType] = stat
	}

	stats.LatencyP50 = time.Duration(percentile(allLatencies, 50)) * time.Millisecond
	stats.LatencyP95 = time.Duration(percentile(allLatencies, 95)) * time.Millisecond

	return stats
}

// Helper functions

func truncateToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

func makeAgentKey(hourBucket time.Time, agentType string) string {
	return hourBucket.Format(time.RFC3339) + "|" + agentType
}

func makeToolKey(hourBucket time.Time, toolName string) string {
	return hourBucket.Format(time.RFC3339) + "|" + toolName
}

func sumLatencies(latencies []int64) int64 {
	var sum int64
	for _, l := range latencies {
		sum += l
	}
	return sum
}

func percentile(latencies []int64, p int) int64 {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]int64, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := (len(sorted) - 1) * p / 100
	return sorted[idx]
}
