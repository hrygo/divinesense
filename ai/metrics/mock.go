package metrics

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MockMetricsService is a mock implementation of MetricsService for testing.
type MockMetricsService struct {
	requests  []requestRecord
	toolCalls []toolCallRecord
	mu        sync.RWMutex
}

type requestRecord struct {
	Timestamp time.Time
	AgentType string
	Latency   time.Duration
	Success   bool
}

type toolCallRecord struct {
	Timestamp time.Time
	ToolName  string
	Latency   time.Duration
	Success   bool
}

// NewMockMetricsService creates a new MockMetricsService.
func NewMockMetricsService() *MockMetricsService {
	return &MockMetricsService{
		requests:  make([]requestRecord, 0),
		toolCalls: make([]toolCallRecord, 0),
	}
}

// RecordRequest records request metrics.
func (m *MockMetricsService) RecordRequest(ctx context.Context, agentType string, latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, requestRecord{
		AgentType: agentType,
		Latency:   latency,
		Success:   success,
		Timestamp: time.Now(),
	})
}

// RecordToolCall records tool call metrics.
func (m *MockMetricsService) RecordToolCall(ctx context.Context, toolName string, latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.toolCalls = append(m.toolCalls, toolCallRecord{
		ToolName:  toolName,
		Latency:   latency,
		Success:   success,
		Timestamp: time.Now(),
	})
}

// GetStats retrieves statistics data.
func (m *MockMetricsService) GetStats(ctx context.Context, timeRange TimeRange) (*AgentMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := &AgentMetrics{
		AgentStats:   make(map[string]*AgentStat),
		ErrorsByType: make(map[string]int64),
	}

	// Filter requests by time range
	var filteredRequests []requestRecord
	for _, r := range m.requests {
		if (timeRange.Start.IsZero() || !r.Timestamp.Before(timeRange.Start)) &&
			(timeRange.End.IsZero() || !r.Timestamp.After(timeRange.End)) {
			filteredRequests = append(filteredRequests, r)
		}
	}

	// Calculate metrics
	metrics.RequestCount = int64(len(filteredRequests))

	var latencies []time.Duration
	agentData := make(map[string]*struct {
		count        int64
		successCount int64
		totalLatency time.Duration
	})

	for _, r := range filteredRequests {
		if r.Success {
			metrics.SuccessCount++
		} else {
			metrics.ErrorsByType[r.AgentType]++
		}

		latencies = append(latencies, r.Latency)

		if _, ok := agentData[r.AgentType]; !ok {
			agentData[r.AgentType] = &struct {
				count        int64
				successCount int64
				totalLatency time.Duration
			}{}
		}
		agentData[r.AgentType].count++
		if r.Success {
			agentData[r.AgentType].successCount++
		}
		agentData[r.AgentType].totalLatency += r.Latency
	}

	// Calculate percentiles
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		p50Index := len(latencies) * 50 / 100
		p95Index := len(latencies) * 95 / 100
		if p95Index >= len(latencies) {
			p95Index = len(latencies) - 1
		}

		metrics.LatencyP50 = latencies[p50Index]
		metrics.LatencyP95 = latencies[p95Index]
	}

	// Build agent stats
	for agentType, data := range agentData {
		var successRate float32
		if data.count > 0 {
			successRate = float32(data.successCount) / float32(data.count)
		}

		metrics.AgentStats[agentType] = &AgentStat{
			Count:       data.count,
			SuccessRate: successRate,
			AvgLatency:  data.totalLatency / time.Duration(data.count),
		}
	}

	return metrics, nil
}

// Clear removes all recorded metrics (for testing).
func (m *MockMetricsService) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = make([]requestRecord, 0)
	m.toolCalls = make([]toolCallRecord, 0)
}

// Ensure MockMetricsService implements MetricsService.
var _ MetricsService = (*MockMetricsService)(nil)
