package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/hrygo/divinesense/store"
)

// PostgresWeightStorage implements RouterWeightStorage using PostgreSQL.
type PostgresWeightStorage struct {
	db StoreInterface
}

// StoreInterface defines the database interface needed for weight storage.
type StoreInterface interface {
	GetUserRouterWeightsMap(ctx context.Context, userID int32) (map[string]map[string]int, error)
	SaveUserRouterWeights(ctx context.Context, userID int32, weights map[string]map[string]int) error
	CreateRouterFeedback(ctx context.Context, create *store.CreateRouterFeedback) error
	GetRouterStats(ctx context.Context, get *store.GetRouterStats) (*store.RouterStats, error)
}

// NewPostgresWeightStorage creates a new PostgreSQL-backed weight storage.
func NewPostgresWeightStorage(db StoreInterface) *PostgresWeightStorage {
	return &PostgresWeightStorage{db: db}
}

// GetWeights retrieves custom weights for a user from PostgreSQL.
func (s *PostgresWeightStorage) GetWeights(ctx context.Context, userID int32) (map[string]map[string]int, error) {
	return s.db.GetUserRouterWeightsMap(ctx, userID)
}

// SaveWeights saves custom weights for a user to PostgreSQL.
func (s *PostgresWeightStorage) SaveWeights(ctx context.Context, userID int32, weights map[string]map[string]int) error {
	return s.db.SaveUserRouterWeights(ctx, userID, weights)
}

// RecordFeedback records a feedback event to PostgreSQL.
func (s *PostgresWeightStorage) RecordFeedback(ctx context.Context, feedback *RouterFeedback) error {
	create := &store.CreateRouterFeedback{
		UserID:    feedback.UserID,
		Input:     feedback.Input,
		Predicted: string(feedback.Predicted),
		Actual:    string(feedback.Actual),
		Feedback:  string(feedback.Feedback),
		Timestamp: feedback.Timestamp,
		Source:    feedback.Source,
	}
	return s.db.CreateRouterFeedback(ctx, create)
}

// GetStats retrieves routing statistics from PostgreSQL.
func (s *PostgresWeightStorage) GetStats(ctx context.Context, userID int32, timeRange time.Duration) (*RouterStats, error) {
	storeStats, err := s.db.GetRouterStats(ctx, &store.GetRouterStats{
		UserID:    userID,
		TimeRange: timeRange,
	})
	if err != nil {
		return nil, err
	}

	// Convert store stats to router stats
	result := &RouterStats{
		TotalPredictions: storeStats.TotalPredictions,
		CorrectCount:     storeStats.CorrectCount,
		IncorrectCount:   storeStats.IncorrectCount,
		Accuracy:         storeStats.Accuracy,
		ByIntent:         make(map[Intent]int64),
		BySource:         storeStats.BySource,
		LastUpdated:      storeStats.LastUpdated,
	}

	for k, v := range storeStats.ByIntent {
		result.ByIntent[Intent(k)] = v
	}

	return result, nil
}

// ConvertStoreToRouterStats converts store.RouterStats to router.RouterStats.
func ConvertStoreToRouterStats(storeStats *store.RouterStats) *RouterStats {
	result := &RouterStats{
		TotalPredictions: storeStats.TotalPredictions,
		CorrectCount:     storeStats.CorrectCount,
		IncorrectCount:   storeStats.IncorrectCount,
		Accuracy:         storeStats.Accuracy,
		ByIntent:         make(map[Intent]int64),
		BySource:         storeStats.BySource,
		LastUpdated:      storeStats.LastUpdated,
	}

	for k, v := range storeStats.ByIntent {
		result.ByIntent[Intent(k)] = v
	}

	return result
}

// VerifyPostgresWeightStorage verifies that the PostgreSQL storage is properly configured.
func VerifyPostgresWeightStorage(ctx context.Context, db StoreInterface) error {
	// Try to get weights for user 0 (a non-existent user)
	_, err := db.GetUserRouterWeightsMap(ctx, 0)
	if err != nil {
		return fmt.Errorf("postgres weight storage verification failed: %w", err)
	}
	return nil
}
