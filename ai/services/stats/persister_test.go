package stats

import (
	"context"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/store"
)

func TestPersister_Enqueue(t *testing.T) {
	// Create a mock store for testing
	mockStore := &mockAgentStatsStore{}
	p := NewPersister(mockStore, 10, nil)

	stats := &agent.AgentSessionStatsForStorage{
		SessionID:       "test-session-123",
		ConversationID:  1,
		UserID:          1,
		AgentType:       "geek",
		StartTime:       time.Now(),
		EndedAt:         time.Now().Add(30 * time.Second),
		TotalDurationMs: 30000,
		TotalCostUSD:    0.50,
	}

	// Test enqueue
	if !p.Enqueue(stats) {
		t.Fatal("failed to enqueue stats")
	}

	// Verify queue size
	if p.QueueSize() != 1 {
		t.Errorf("expected queue size 1, got %d", p.QueueSize())
	}

	// Close the persister
	if err := p.Close(5 * time.Second); err != nil {
		t.Fatalf("failed to close persister: %v", err)
	}

	// Verify stats were saved
	if !mockStore.saved {
		t.Error("stats were not saved to store")
	}
}

func TestPersister_QueueFull(t *testing.T) {
	mockStore := &mockAgentStatsStore{slow: true}
	p := NewPersister(mockStore, 2, nil)

	// Fill the queue
	for i := 0; i < 5; i++ {
		stats := &agent.AgentSessionStatsForStorage{
			SessionID:       "test-session-123",
			ConversationID:  1,
			UserID:          1,
			AgentType:       "geek",
			StartTime:       time.Now(),
			EndedAt:         time.Now(),
			TotalDurationMs: 30000,
			TotalCostUSD:    0.50,
		}
		p.Enqueue(stats)
	}

	// Close and drain
	if err := p.Close(5 * time.Second); err != nil {
		t.Fatalf("failed to close persister: %v", err)
	}

	if p.QueueSize() != 0 {
		t.Errorf("expected queue to be drained, got %d", p.QueueSize())
	}
}

func TestPersister_EnqueueSessionStatsData(t *testing.T) {
	mockStore := &mockAgentStatsStore{}
	p := NewPersister(mockStore, 10, nil)

	data := &agent.SessionStatsData{
		SessionID:       "test-session-456",
		ConversationID:  1,
		UserID:          1,
		AgentType:       "geek",
		StartTime:       time.Now().Unix(),
		EndTime:         time.Now().Unix(),
		TotalDurationMs: 15000,
		TotalCostUSD:    0.25,
	}

	// Test enqueue via SessionStatsData
	if !p.EnqueueSessionStatsData(data) {
		t.Fatal("failed to enqueue session stats data")
	}

	if err := p.Close(5 * time.Second); err != nil {
		t.Fatalf("failed to close persister: %v", err)
	}

	if !mockStore.saved {
		t.Error("stats were not saved to store")
	}
}

func TestCostAlert_String(t *testing.T) {
	tests := []struct {
		name     string
		alert    *CostAlert
		expected string
	}{
		{
			name: "session threshold exceeded",
			alert: &CostAlert{
				Type:         "session_threshold_exceeded",
				CostUSD:      6.0,
				ThresholdUSD: 5.0,
			},
			expected: "Session cost $6.0000 exceeds threshold $5.0000",
		},
		{
			name: "daily budget exceeded",
			alert: &CostAlert{
				Type:         "daily_budget_exceeded",
				DailyCostUSD: 11.0,
				BudgetUSD:    10.0,
				OverByUSD:    1.0,
			},
			expected: "Daily cost $11.0000 exceeds budget $10.0000 by $1.0000",
		},
		{
			name: "daily budget warning",
			alert: &CostAlert{
				Type:         "daily_budget_warning",
				DailyCostUSD: 9.5,
				BudgetUSD:    10.0,
				OverByUSD:    0.5,
			},
			expected: "Daily cost $9.5000, $0.5000 budget remaining",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alert.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// mockAgentStatsStore is a mock implementation of AgentStatsStore for testing.
type mockAgentStatsStore struct {
	saved bool
	slow  bool
}

func (m *mockAgentStatsStore) SaveSessionStats(ctx context.Context, stats *store.AgentSessionStats) error {
	if m.slow {
		time.Sleep(100 * time.Millisecond)
	}
	m.saved = true
	return nil
}

func (m *mockAgentStatsStore) GetSessionStats(ctx context.Context, sessionID string) (*store.AgentSessionStats, error) {
	return nil, nil
}

func (m *mockAgentStatsStore) ListSessionStats(ctx context.Context, userID int32, limit, offset int) ([]*store.AgentSessionStats, int64, error) {
	return nil, 0, nil
}

func (m *mockAgentStatsStore) GetDailyCostUsage(ctx context.Context, userID int32, startDate, endDate time.Time) (float64, error) {
	return 0, nil
}

func (m *mockAgentStatsStore) GetCostStats(ctx context.Context, userID int32, days int) (*store.CostStats, error) {
	return nil, nil
}

func (m *mockAgentStatsStore) GetUserCostSettings(ctx context.Context, userID int32) (*store.UserCostSettings, error) {
	return nil, nil
}

func (m *mockAgentStatsStore) SetUserCostSettings(ctx context.Context, settings *store.UserCostSettings) error {
	return nil
}
