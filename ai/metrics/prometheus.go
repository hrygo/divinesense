// Package metrics provides Prometheus metrics export for AI modules.
package metrics

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusExporter exports AI metrics in Prometheus format.
type PrometheusExporter struct {
	registry *prometheus.Registry

	// Chat metrics
	chatLatency  *prometheus.HistogramVec
	chatRequests *prometheus.CounterVec
	chatActive   prometheus.Gauge

	// Tool call metrics
	toolCalls   *prometheus.CounterVec
	toolLatency *prometheus.HistogramVec
	toolErrors  *prometheus.CounterVec

	// Cache metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec

	// LLM token metrics
	llmTokensUsed   *prometheus.CounterVec
	llmTokensCached *prometheus.CounterVec
	llmLatency      *prometheus.HistogramVec

	// Agent-specific metrics
	agentErrors      *prometheus.CounterVec
	agentSuccessRate *prometheus.GaugeVec

	mu       sync.RWMutex
	handlers map[string]http.Handler
}

// Config configures the Prometheus exporter.
type Config struct {
	// Registry to use (if nil, creates a new one)
	Registry *prometheus.Registry

	// Buckets for latency histograms (in seconds)
	LatencyBuckets []float64
}

// DefaultConfig returns default Prometheus configuration.
func DefaultConfig() Config {
	return Config{
		LatencyBuckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60},
	}
}

// NewPrometheusExporter creates a new Prometheus metrics exporter.
func NewPrometheusExporter(cfg Config) *PrometheusExporter {
	if len(cfg.LatencyBuckets) == 0 {
		cfg.LatencyBuckets = DefaultConfig().LatencyBuckets
	}

	registry := cfg.Registry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	e := &PrometheusExporter{
		registry: registry,
		handlers: make(map[string]http.Handler),
	}

	// Chat metrics
	e.chatLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "chat_latency_seconds",
			Help:      "Chat request latency in seconds",
			Buckets:   cfg.LatencyBuckets,
		},
		[]string{"agent_type", "mode"},
	)

	e.chatRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "chat_requests_total",
			Help:      "Total number of chat requests",
		},
		[]string{"agent_type", "mode", "status"},
	)

	e.chatActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "chat_active",
			Help:      "Number of active chat sessions",
		},
	)

	// Tool call metrics
	e.toolCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "tool_calls_total",
			Help:      "Total number of tool calls",
		},
		[]string{"tool_name", "status"},
	)

	e.toolLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "tool_latency_seconds",
			Help:      "Tool call latency in seconds",
			Buckets:   cfg.LatencyBuckets,
		},
		[]string{"tool_name"},
	)

	e.toolErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "tool_errors_total",
			Help:      "Total number of tool errors",
		},
		[]string{"tool_name", "error_type"},
	)

	// Cache metrics
	e.cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	e.cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	// LLM token metrics
	e.llmTokensUsed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "llm_tokens_total",
			Help:      "Total LLM tokens consumed",
		},
		[]string{"model", "token_type"},
	)

	e.llmTokensCached = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "llm_tokens_cached_total",
			Help:      "Total LLM tokens served from cache",
		},
		[]string{"model"},
	)

	e.llmLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "llm_latency_seconds",
			Help:      "LLM request latency in seconds",
			Buckets:   cfg.LatencyBuckets,
		},
		[]string{"model", "provider"},
	)

	// Agent error metrics
	e.agentErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "agent_errors_total",
			Help:      "Total number of agent errors",
		},
		[]string{"agent_type", "error_type"},
	)

	e.agentSuccessRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "divinesense",
			Subsystem: "ai",
			Name:      "agent_success_rate",
			Help:      "Agent success rate (0-1)",
		},
		[]string{"agent_type"},
	)

	// Register all metrics
	registry.MustRegister(
		e.chatLatency,
		e.chatRequests,
		e.chatActive,
		e.toolCalls,
		e.toolLatency,
		e.toolErrors,
		e.cacheHits,
		e.cacheMisses,
		e.llmTokensUsed,
		e.llmTokensCached,
		e.llmLatency,
		e.agentErrors,
		e.agentSuccessRate,
	)

	return e
}

// RecordChatRequest records a chat request metric.
func (e *PrometheusExporter) RecordChatRequest(agentType, mode string, latency time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	e.chatRequests.WithLabelValues(agentType, mode, status).Inc()
	e.chatLatency.WithLabelValues(agentType, mode).Observe(latency.Seconds())
}

// RecordToolCall records a tool call metric.
func (e *PrometheusExporter) RecordToolCall(toolName string, latency time.Duration, success bool, errorType string) {
	status := "success"
	if !success {
		status = "error"
		if errorType != "" {
			e.toolErrors.WithLabelValues(toolName, errorType).Inc()
		}
	}

	e.toolCalls.WithLabelValues(toolName, status).Inc()
	e.toolLatency.WithLabelValues(toolName).Observe(latency.Seconds())
}

// RecordCacheHit records a cache hit.
func (e *PrometheusExporter) RecordCacheHit(cacheType string) {
	e.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss.
func (e *PrometheusExporter) RecordCacheMiss(cacheType string) {
	e.cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordLLMTokens records LLM token usage.
func (e *PrometheusExporter) RecordLLMTokens(model, tokenType string, count int) {
	e.llmTokensUsed.WithLabelValues(model, tokenType).Add(float64(count))
}

// RecordLLMCachedTokens records cached LLM tokens.
func (e *PrometheusExporter) RecordLLMCachedTokens(model string, count int) {
	e.llmTokensCached.WithLabelValues(model).Add(float64(count))
}

// RecordLLMLatency records LLM request latency.
func (e *PrometheusExporter) RecordLLMLatency(model, provider string, latency time.Duration) {
	e.llmLatency.WithLabelValues(model, provider).Observe(latency.Seconds())
}

// SetActiveChats sets the number of active chat sessions.
func (e *PrometheusExporter) SetActiveChats(count int) {
	e.chatActive.Set(float64(count))
}

// RecordAgentError records an agent error.
func (e *PrometheusExporter) RecordAgentError(agentType, errorType string) {
	e.agentErrors.WithLabelValues(agentType, errorType).Inc()
}

// SetAgentSuccessRate sets the success rate for an agent.
func (e *PrometheusExporter) SetAgentSuccessRate(agentType string, rate float64) {
	e.agentSuccessRate.WithLabelValues(agentType).Set(rate)
}

// GetHandler returns the HTTP handler for Prometheus metrics.
func (e *PrometheusExporter) GetHandler() http.Handler {
	return promhttp.HandlerFor(e.registry, promhttp.HandlerOpts{})
}

// Handler returns an HTTP handler for the metrics endpoint.
func (e *PrometheusExporter) Handler() http.Handler {
	return e.GetHandler()
}

// RegisterHandler registers a custom handler for a specific path.
func (e *PrometheusExporter) RegisterHandler(path string, handler http.Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[path] = handler
}

// ServeHTTP implements http.Handler for the metrics endpoint.
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.GetHandler().ServeHTTP(w, r)
}

// GetRegistry returns the Prometheus registry.
func (e *PrometheusExporter) GetRegistry() *prometheus.Registry {
	return e.registry
}

// Snapshot captures a snapshot of all metrics for debugging.
func (e *PrometheusExporter) Snapshot() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	snapshot := make(map[string]interface{})
	snapshot["timestamp"] = time.Now().Unix()
	gatherResult, err := e.registry.Gather()
	if err != nil {
		slog.Error("failed to gather metrics", "error", err)
	}
	snapshot["registry"] = gatherResult

	return snapshot
}

// MetricFamily represents a Prometheus metric family for export.
type MetricFamily struct {
	Name    string   `json:"name"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`
	Metrics []Metric `json:"metrics"`
}

// Metric represents a single metric.
type Metric struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Value     float64           `json:"value,omitempty"`
	Histogram *Histogram        `json:"histogram,omitempty"`
}

// Histogram represents histogram data.
type Histogram struct {
	Sum     float64  `json:"sum"`
	Count   int64    `json:"count"`
	Buckets []Bucket `json:"buckets"`
}

// Bucket represents a histogram bucket.
type Bucket struct {
	UpperBound float64 `json:"upper_bound"`
	Count      int64   `json:"count"`
}

// ExportText exports metrics in Prometheus text format.
func (e *PrometheusExporter) ExportText() (string, error) {
	var sb strings.Builder

	metrics, err := e.registry.Gather()
	if err != nil {
		return "", err
	}

	for _, mf := range metrics {
		sb.WriteString("# HELP ")
		sb.WriteString(mf.GetName())
		sb.WriteString(" ")
		sb.WriteString(mf.GetHelp())
		sb.WriteString("\n")

		sb.WriteString("# TYPE ")
		sb.WriteString(mf.GetName())
		sb.WriteString(" ")
		sb.WriteString(mf.GetType().String())
		sb.WriteString("\n")

		for _, m := range mf.GetMetric() {
			sb.WriteString(mf.GetName())

			// Labels
			if len(m.GetLabel()) > 0 {
				sb.WriteString("{")
				labels := make([]string, 0, len(m.GetLabel()))
				for _, label := range m.GetLabel() {
					labels = append(labels, label.GetName()+"=\""+label.GetValue()+"\"")
				}
				sort.Strings(labels)
				sb.WriteString(strings.Join(labels, ","))
				sb.WriteString("}")
			}

			sb.WriteString(" ")

			// Value based on type
			metricType := mf.GetType().String()
			switch metricType {
			case "COUNTER":
				if c := m.GetCounter(); c != nil {
					sb.WriteString(strconv.FormatFloat(c.GetValue(), 'f', -1, 64))
				}
			case "GAUGE":
				if g := m.GetGauge(); g != nil {
					sb.WriteString(strconv.FormatFloat(g.GetValue(), 'f', -1, 64))
				}
			case "HISTOGRAM":
				if h := m.GetHistogram(); h != nil {
					sb.WriteString(strconv.FormatFloat(h.GetSampleSum(), 'f', -1, 64))
					for _, b := range h.GetBucket() {
						sb.WriteString("\n")
						sb.WriteString(mf.GetName())
						sb.WriteString("_bucket{le=\"")
						sb.WriteString(strconv.FormatFloat(b.GetUpperBound(), 'f', -1, 64))
						sb.WriteString("\"}")
						sb.WriteString(strconv.FormatUint(b.GetCumulativeCount(), 10))
					}
				}
			default:
				// Unknown type, skip value
				goto nextMetric
			}

			sb.WriteString(" ")
			sb.WriteString(strconv.FormatInt(m.GetTimestampMs(), 10))
			sb.WriteString("\n")
		nextMetric:
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// Close cleans up resources.
func (e *PrometheusExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Clear handlers map
	e.handlers = make(map[string]http.Handler)
	return nil
}
