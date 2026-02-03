// Package metrics provides tests for webhook health monitoring.
package metrics

import (
	"sync"
	"testing"
	"time"
)

// TestRecordEvent tests recording various webhook events.
func TestRecordEvent(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 123

	// Record multiple events
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)
	registry.RecordEvent(platform, credID, EventMessageProcessed, 100*time.Millisecond, nil)
	registry.RecordEvent(platform, credID, EventResponseSent, 0, nil)

	// Get metrics snapshot
	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Verify counts
	if snapshot.TotalReceived != 1 {
		t.Errorf("TotalReceived = %d, want 1", snapshot.TotalReceived)
	}
	if snapshot.TotalValidated != 1 {
		t.Errorf("TotalValidated = %d, want 1", snapshot.TotalValidated)
	}
	if snapshot.MessagesProcessed != 1 {
		t.Errorf("MessagesProcessed = %d, want 1", snapshot.MessagesProcessed)
	}
	if snapshot.ResponsesSent != 1 {
		t.Errorf("ResponsesSent = %d, want 1", snapshot.ResponsesSent)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestRecordError tests recording error events.
func TestRecordError(t *testing.T) {
	registry := GetRegistry()
	platform := "dingtalk"
	var credID int64 = 456
	testErr := Err("test error")

	// Record error events
	registry.RecordEvent(platform, credID, EventWebhookParseError, 0, testErr)
	registry.RecordEvent(platform, credID, EventResponseError, 50*time.Millisecond, testErr)

	// Get metrics snapshot
	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Verify error counts
	if snapshot.ParseErrors != 1 {
		t.Errorf("ParseErrors = %d, want 1", snapshot.ParseErrors)
	}
	if snapshot.ResponseErrors != 1 {
		t.Errorf("ResponseErrors = %d, want 1", snapshot.ResponseErrors)
	}

	// Verify recent errors
	if len(snapshot.RecentErrors) != 2 {
		t.Errorf("RecentErrors length = %d, want 2", len(snapshot.RecentErrors))
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestSuccessRate tests the success rate calculation.
func TestSuccessRate(t *testing.T) {
	registry := GetRegistry()
	platform := "whatsapp"
	var credID int64 = 789

	// Record events with some failures
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookParseError, 0, Err("parse error"))

	// Get metrics snapshot
	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Success rate should be 60% (3 validated out of 5 received)
	expectedRate := 60.0
	actualRate := snapshot.SuccessRate()
	if actualRate != expectedRate {
		t.Errorf("SuccessRate = %.2f, want %.2f", actualRate, expectedRate)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestErrorRate tests the error rate calculation.
func TestErrorRate(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 999

	// Record events
	registry.RecordEvent(platform, credID, EventMessageProcessed, 0, nil)
	registry.RecordEvent(platform, credID, EventMessageProcessed, 0, nil)
	registry.RecordEvent(platform, credID, EventMessageProcessed, 0, nil)
	registry.RecordEvent(platform, credID, EventMessageProcessed, 0, nil)
	registry.RecordEvent(platform, credID, EventResponseSent, 0, nil)
	registry.RecordEvent(platform, credID, EventResponseSent, 0, nil)
	registry.RecordEvent(platform, credID, EventResponseSent, 0, nil)
	registry.RecordEvent(platform, credID, EventResponseError, 0, Err("response error"))
	registry.RecordEvent(platform, credID, EventResponseError, 0, Err("response error"))

	// Get metrics snapshot
	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Error rate should be 40% (2 errors out of 4 processed + 2 errors = 6 total operations affecting 4 messages)
	expectedRate := 50.0 // 2 errors / 4 processed
	actualRate := snapshot.ErrorRate()
	if actualRate != expectedRate {
		t.Errorf("ErrorRate = %.2f, want %.2f", actualRate, expectedRate)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestIsHealthy tests the health check logic.
func TestIsHealthy(t *testing.T) {
	registry := GetRegistry()

	t.Run("no data yet", func(t *testing.T) {
		snapshot := &WebhookMetricsSnapshot{}
		if snapshot.IsHealthy() {
			t.Error("expected unhealthy when no data, got healthy")
		}
	})

	t.Run("recent activity", func(t *testing.T) {
		platform := "telegram"
		var credID int64 = 111
		registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)

		snapshot := registry.GetMetrics(platform, credID)
		if snapshot == nil {
			t.Fatal("expected metrics snapshot, got nil")
		}
		if !snapshot.IsHealthy() {
			t.Error("expected healthy with recent activity, got unhealthy")
		}

		registry.Clear(platform, credID)
	})

	t.Run("stale data", func(t *testing.T) {
		snapshot := &WebhookMetricsSnapshot{
			LastReceived: time.Now().Add(-10 * time.Minute),
		}
		if snapshot.IsHealthy() {
			t.Error("expected unhealthy with stale data, got healthy")
		}
	})
}

// TestConcurrentAccess tests thread-safety of the registry.
func TestConcurrentAccess(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 222

	const numGoroutines = 10
	const numEventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numEventsPerGoroutine; j++ {
				registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
				registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)
				registry.GetMetrics(platform, credID)
			}
		}()
	}

	wg.Wait()

	// Verify final counts
	snapshot := registry.GetMetrics(platform, credID)
	expectedReceived := int64(numGoroutines * numEventsPerGoroutine)
	if snapshot.TotalReceived != expectedReceived {
		t.Errorf("TotalReceived = %d, want %d", snapshot.TotalReceived, expectedReceived)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestGetAllMetrics tests retrieving all metrics.
func TestGetAllMetrics(t *testing.T) {
	registry := GetRegistry()

	// Record events for multiple credentials
	registry.RecordEvent("telegram", 1, EventWebhookReceived, 0, nil)
	registry.RecordEvent("dingtalk", 2, EventWebhookReceived, 0, nil)
	registry.RecordEvent("whatsapp", 3, EventWebhookReceived, 0, nil)

	// Get all metrics
	allMetrics := registry.GetAllMetrics()

	// Should have 3 entries
	if len(allMetrics) != 3 {
		t.Errorf("GetAllMetrics() returned %d entries, want 3", len(allMetrics))
	}

	// Clean up
	registry.Clear("telegram", 1)
	registry.Clear("dingtalk", 2)
	registry.Clear("whatsapp", 3)
}

// TestRecentErrors tests the recent errors tracking.
func TestRecentErrors(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 333

	// Add 15 errors (more than the limit of 10)
	for i := 0; i < 15; i++ {
		registry.RecordEvent(platform, credID, EventResponseError, 0, Err("error"))
	}

	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Should only keep last 10 errors
	if len(snapshot.RecentErrors) != 10 {
		t.Errorf("RecentErrors length = %d, want 10", len(snapshot.RecentErrors))
	}

	// Verify the oldest error was removed
	oldestError := snapshot.RecentErrors[0]
	if oldestError.Error != "error" {
		t.Errorf("oldest error = %s, want 'error'", oldestError.Error)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestAvgProcessTime tests average processing time calculation.
func TestAvgProcessTime(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 444

	// Record processing times
	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		400 * time.Millisecond,
		500 * time.Millisecond,
	}

	for _, d := range durations {
		registry.RecordEvent(platform, credID, EventMessageProcessed, d, nil)
	}

	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot, got nil")
	}

	// Average should be (100+200+300+400+500)/5 = 300ms
	expectedAvg := 300 * time.Millisecond
	if snapshot.AvgProcessTime != expectedAvg {
		t.Errorf("AvgProcessTime = %v, want %v", snapshot.AvgProcessTime, expectedAvg)
	}

	// Clean up
	registry.Clear(platform, credID)
}

// TestClear tests clearing metrics.
func TestClear(t *testing.T) {
	registry := GetRegistry()
	platform := "telegram"
	var credID int64 = 555

	// Record some events
	registry.RecordEvent(platform, credID, EventWebhookReceived, 0, nil)
	registry.RecordEvent(platform, credID, EventWebhookValidated, 0, nil)

	// Verify metrics exist
	snapshot := registry.GetMetrics(platform, credID)
	if snapshot == nil {
		t.Fatal("expected metrics snapshot before clear, got nil")
	}

	// Clear metrics
	registry.Clear(platform, credID)

	// Verify metrics are gone
	snapshot = registry.GetMetrics(platform, credID)
	if snapshot != nil {
		t.Error("expected nil after clear, got non-nil snapshot")
	}
}

// Err creates a test error.
func Err(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
