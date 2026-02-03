// Package metrics provides webhook health monitoring for chat apps.
package metrics

import (
	"sync"
	"time"
)

// EventType represents the type of webhook event being tracked.
type EventType string

const (
	EventWebhookReceived   EventType = "webhook_received"
	EventWebhookValidated  EventType = "webhook_validated"
	EventWebhookParseError EventType = "webhook_parse_error"
	EventMessageProcessed  EventType = "message_processed"
	EventResponseSent      EventType = "response_sent"
	EventResponseError     EventType = "response_error"
)

// WebhookMetrics tracks delivery metrics for webhooks.
type WebhookMetrics struct {
	mu sync.RWMutex

	// Counters
	totalReceived     int64
	totalValidated    int64
	parseErrors       int64
	messagesProcessed int64
	responsesSent     int64
	responseErrors    int64

	// Timing
	lastReceived     time.Time
	lastValidated    time.Time
	lastError        time.Time
	avgProcessTime   time.Duration
	totalProcessTime int64

	// Error tracking
	recentErrors []ErrorRecord
}

// ErrorRecord records details of an error.
type ErrorRecord struct {
	Timestamp time.Time
	EventType EventType
	Error     string
	Platform  string
}

// MetricsKey identifies a metric entry (platform + credential ID).
type MetricsKey struct {
	Platform string
	CredID   int64
}

// Registry holds metrics for all registered webhook endpoints.
type Registry struct {
	mu      sync.RWMutex
	metrics map[MetricsKey]*WebhookMetrics
}

// Global registry instance.
var globalRegistry = &Registry{
	metrics: make(map[MetricsKey]*WebhookMetrics),
}

// GetRegistry returns the global metrics registry.
func GetRegistry() *Registry {
	return globalRegistry
}

// RecordEvent records a webhook event.
func (r *Registry) RecordEvent(platform string, credID int64, eventType EventType, processDuration time.Duration, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := MetricsKey{Platform: platform, CredID: credID}
	m, exists := r.metrics[key]
	if !exists {
		m = &WebhookMetrics{
			recentErrors: make([]ErrorRecord, 0, 10),
		}
		r.metrics[key] = m
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	switch eventType {
	case EventWebhookReceived:
		m.totalReceived++
		m.lastReceived = now

	case EventWebhookValidated:
		m.totalValidated++
		m.lastValidated = now

	case EventWebhookParseError:
		m.parseErrors++
		m.lastError = now
		if err != nil {
			m.addErrorRecord(now, eventType, err.Error(), platform)
		}

	case EventMessageProcessed:
		m.messagesProcessed++
		if processDuration > 0 {
			m.totalProcessTime += int64(processDuration)
			// Update average: new_avg = (total + new) / count
			m.avgProcessTime = time.Duration(m.totalProcessTime / m.messagesProcessed)
		}

	case EventResponseSent:
		m.responsesSent++

	case EventResponseError:
		m.responseErrors++
		m.lastError = now
		if err != nil {
			m.addErrorRecord(now, eventType, err.Error(), platform)
		}
	}
}

// addErrorRecord adds an error to the recent errors list.
func (m *WebhookMetrics) addErrorRecord(ts time.Time, eventType EventType, errMsg string, platform string) {
	record := ErrorRecord{
		Timestamp: ts,
		EventType: eventType,
		Error:     errMsg,
		Platform:  platform,
	}

	// Keep only last 10 errors
	m.recentErrors = append(m.recentErrors, record)
	if len(m.recentErrors) > 10 {
		m.recentErrors = m.recentErrors[1:]
	}
}

// GetMetrics returns a snapshot of metrics for a specific key.
func (r *Registry) GetMetrics(platform string, credID int64) *WebhookMetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := MetricsKey{Platform: platform, CredID: credID}
	m, exists := r.metrics[key]
	if !exists {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return &WebhookMetricsSnapshot{
		TotalReceived:     m.totalReceived,
		TotalValidated:    m.totalValidated,
		ParseErrors:       m.parseErrors,
		MessagesProcessed: m.messagesProcessed,
		ResponsesSent:     m.responsesSent,
		ResponseErrors:    m.responseErrors,
		LastReceived:      m.lastReceived,
		LastValidated:     m.lastValidated,
		LastError:         m.lastError,
		AvgProcessTime:    m.avgProcessTime,
		RecentErrors:      append([]ErrorRecord{}, m.recentErrors...),
	}
}

// GetAllMetrics returns snapshots of all metrics.
func (r *Registry) GetAllMetrics() map[MetricsKey]*WebhookMetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[MetricsKey]*WebhookMetricsSnapshot)
	for key, m := range r.metrics {
		m.mu.RLock()
		result[key] = &WebhookMetricsSnapshot{
			TotalReceived:     m.totalReceived,
			TotalValidated:    m.totalValidated,
			ParseErrors:       m.parseErrors,
			MessagesProcessed: m.messagesProcessed,
			ResponsesSent:     m.responsesSent,
			ResponseErrors:    m.responseErrors,
			LastReceived:      m.lastReceived,
			LastValidated:     m.lastValidated,
			LastError:         m.lastError,
			AvgProcessTime:    m.avgProcessTime,
			RecentErrors:      append([]ErrorRecord{}, m.recentErrors...),
		}
		m.mu.RUnlock()
	}
	return result
}

// Clear removes metrics for a specific key.
func (r *Registry) Clear(platform string, credID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := MetricsKey{Platform: platform, CredID: credID}
	delete(r.metrics, key)
}

// WebhookMetricsSnapshot is a thread-safe snapshot of webhook metrics.
type WebhookMetricsSnapshot struct {
	TotalReceived     int64
	TotalValidated    int64
	ParseErrors       int64
	MessagesProcessed int64
	ResponsesSent     int64
	ResponseErrors    int64
	LastReceived      time.Time
	LastValidated     time.Time
	LastError         time.Time
	AvgProcessTime    time.Duration
	RecentErrors      []ErrorRecord
}

// SuccessRate calculates the success rate (validated / received).
func (s *WebhookMetricsSnapshot) SuccessRate() float64 {
	if s.TotalReceived == 0 {
		return 100.0
	}
	return float64(s.TotalValidated) / float64(s.TotalReceived) * 100.0
}

// IsHealthy checks if the webhook is healthy (received within last 5 minutes).
func (s *WebhookMetricsSnapshot) IsHealthy() bool {
	if s.LastReceived.IsZero() {
		return false // No data yet
	}
	return time.Since(s.LastReceived) < 5*time.Minute
}

// ErrorRate calculates the error rate (errors / processed).
func (s *WebhookMetricsSnapshot) ErrorRate() float64 {
	if s.MessagesProcessed == 0 {
		return 0.0
	}
	return float64(s.ResponseErrors) / float64(s.MessagesProcessed) * 100.0
}
