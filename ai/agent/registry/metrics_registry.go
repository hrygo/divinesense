// Package registry provides metrics collection for UniversalParrot.
package registry

import (
	"sync"
	"time"
)

// MetricsRegistry collects performance metrics for parrot executions.
// This enables observability and performance optimization.
type MetricsRegistry struct {
	mu      sync.RWMutex
	metrics map[string]*ParrotMetrics
}

// ParrotMetrics holds metrics for a single parrot type.
type ParrotMetrics struct {
	ExecutionCount int64
	TotalLatencyMs int64
	ErrorCount     int64
	LastExecution  time.Time
	CacheHitCount  int64
	CacheMissCount int64
}

// Global metrics registry instance.
var metricsRegistry = &MetricsRegistry{
	metrics: make(map[string]*ParrotMetrics),
}

// RecordExecution records an execution event.
func RecordExecution(parrotName string, latencyMs int64, success bool) {
	metricsRegistry.mu.Lock()
	defer metricsRegistry.mu.Unlock()

	metrics, ok := metricsRegistry.metrics[parrotName]
	if !ok {
		metrics = &ParrotMetrics{}
		metricsRegistry.metrics[parrotName] = metrics
	}

	metrics.ExecutionCount++
	metrics.TotalLatencyMs += latencyMs
	metrics.LastExecution = time.Now()

	if !success {
		metrics.ErrorCount++
	}
}

// RecordCacheHit records a cache hit event.
func RecordCacheHit(parrotName string) {
	metricsRegistry.mu.Lock()
	defer metricsRegistry.mu.Unlock()

	metrics, ok := metricsRegistry.metrics[parrotName]
	if !ok {
		metrics = &ParrotMetrics{}
		metricsRegistry.metrics[parrotName] = metrics
	}

	metrics.CacheHitCount++
}

// RecordCacheMiss records a cache miss event.
func RecordCacheMiss(parrotName string) {
	metricsRegistry.mu.Lock()
	defer metricsRegistry.mu.Unlock()

	metrics, ok := metricsRegistry.metrics[parrotName]
	if !ok {
		metrics = &ParrotMetrics{}
		metricsRegistry.metrics[parrotName] = metrics
	}

	metrics.CacheMissCount++
}

// GetMetrics retrieves metrics for a parrot.
func GetMetrics(parrotName string) *ParrotMetrics {
	metricsRegistry.mu.RLock()
	defer metricsRegistry.mu.RUnlock()

	if metrics, ok := metricsRegistry.metrics[parrotName]; ok {
		// Return a copy to avoid race conditions
		return &ParrotMetrics{
			ExecutionCount: metrics.ExecutionCount,
			TotalLatencyMs: metrics.TotalLatencyMs,
			ErrorCount:     metrics.ErrorCount,
			LastExecution:  metrics.LastExecution,
			CacheHitCount:  metrics.CacheHitCount,
			CacheMissCount: metrics.CacheMissCount,
		}
	}

	return &ParrotMetrics{}
}

// GetAllMetrics returns metrics for all parrots.
func GetAllMetrics() map[string]*ParrotMetrics {
	metricsRegistry.mu.RLock()
	defer metricsRegistry.mu.RUnlock()

	result := make(map[string]*ParrotMetrics)
	for name, metrics := range metricsRegistry.metrics {
		result[name] = &ParrotMetrics{
			ExecutionCount: metrics.ExecutionCount,
			TotalLatencyMs: metrics.TotalLatencyMs,
			ErrorCount:     metrics.ErrorCount,
			LastExecution:  metrics.LastExecution,
			CacheHitCount:  metrics.CacheHitCount,
			CacheMissCount: metrics.CacheMissCount,
		}
	}

	return result
}

// ResetMetrics clears all metrics. Primarily used for testing.
func ResetMetrics() {
	metricsRegistry.mu.Lock()
	defer metricsRegistry.mu.Unlock()

	metricsRegistry.metrics = make(map[string]*ParrotMetrics)
}

// GetAverageLatency returns the average latency in milliseconds.
func GetAverageLatency(parrotName string) float64 {
	metrics := GetMetrics(parrotName)
	if metrics.ExecutionCount == 0 {
		return 0
	}
	return float64(metrics.TotalLatencyMs) / float64(metrics.ExecutionCount)
}

// GetCacheHitRate returns the cache hit rate as a percentage.
func GetCacheHitRate(parrotName string) float64 {
	metrics := GetMetrics(parrotName)
	total := metrics.CacheHitCount + metrics.CacheMissCount
	if total == 0 {
		return 0
	}
	return float64(metrics.CacheHitCount) / float64(total) * 100
}

// GetErrorRate returns the error rate as a percentage.
func GetErrorRate(parrotName string) float64 {
	metrics := GetMetrics(parrotName)
	if metrics.ExecutionCount == 0 {
		return 0
	}
	return float64(metrics.ErrorCount) / float64(metrics.ExecutionCount) * 100
}
