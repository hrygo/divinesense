package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hrygo/divinesense/store"
)

// CreateRouterFeedback creates a new router feedback entry.
func (d *DB) CreateRouterFeedback(ctx context.Context, create *store.CreateRouterFeedback) error {
	now := time.Now().Unix()
	if create.Timestamp == 0 {
		create.Timestamp = now
	}

	stmt := `INSERT INTO router_feedback (user_id, input, predicted_intent, actual_intent, feedback_type, timestamp, source)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` +
		placeholder(5) + `, ` + placeholder(6) + `, ` + placeholder(7) + `)`

	_, err := d.db.ExecContext(ctx, stmt,
		create.UserID, create.Input, create.Predicted, create.Actual,
		create.Feedback, create.Timestamp, create.Source)
	if err != nil {
		return fmt.Errorf("failed to create router feedback: %w", err)
	}

	return nil
}

// ListRouterFeedback retrieves router feedback entries.
func (d *DB) ListRouterFeedback(ctx context.Context, find *store.FindRouterFeedback) ([]*store.RouterFeedback, error) {
	query := `SELECT id, user_id, input, predicted_intent, actual_intent, feedback_type, timestamp, source
		FROM router_feedback WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if find.UserID != nil {
		query += fmt.Sprintf(" AND user_id = %s", placeholder(argIdx))
		args = append(args, *find.UserID)
		argIdx++
	}
	if find.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= %s", placeholder(argIdx))
		args = append(args, *find.StartTime)
		argIdx++
	}
	if find.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= %s", placeholder(argIdx))
		args = append(args, *find.EndTime)
		argIdx++
	}
	if find.Feedback != nil {
		query += fmt.Sprintf(" AND feedback_type = %s", placeholder(argIdx))
		args = append(args, *find.Feedback)
	}

	query += " ORDER BY timestamp DESC"
	if find.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", find.Limit)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list router feedback: %w", err)
	}
	defer rows.Close()

	var feedbacks []*store.RouterFeedback
	for rows.Next() {
		var fb store.RouterFeedback
		err := rows.Scan(&fb.ID, &fb.UserID, &fb.Input, &fb.Predicted, &fb.Actual, &fb.Feedback, &fb.Timestamp, &fb.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to scan router feedback: %w", err)
		}
		feedbacks = append(feedbacks, &fb)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating router feedback rows: %w", err)
	}

	return feedbacks, nil
}

// GetRouterStats retrieves routing accuracy statistics.
func (d *DB) GetRouterStats(ctx context.Context, get *store.GetRouterStats) (*store.RouterStats, error) {
	cutoff := time.Now().Add(-get.TimeRange).Unix()

	// Get total counts by feedback type
	statsQuery := `SELECT
		COUNT(*) as total,
		COUNT(*) FILTER (WHERE feedback_type = 'positive') as correct,
		COUNT(*) FILTER (WHERE feedback_type != 'positive') as incorrect
		FROM router_feedback
		WHERE user_id = ` + placeholder(1) + ` AND timestamp >= ` + placeholder(2)

	var total, correct, incorrect int64
	err := d.db.QueryRowContext(ctx, statsQuery, get.UserID, cutoff).Scan(&total, &correct, &incorrect)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get router stats: %w", err)
	}

	// Get breakdown by intent
	intentQuery := `SELECT predicted_intent, COUNT(*) as count
		FROM router_feedback
		WHERE user_id = ` + placeholder(1) + ` AND timestamp >= ` + placeholder(2) + `
		GROUP BY predicted_intent`

	intentRows, err := d.db.QueryContext(ctx, intentQuery, get.UserID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get router stats by intent: %w", err)
	}
	defer intentRows.Close()

	byIntent := make(map[string]int64)
	for intentRows.Next() {
		var intent string
		var count int64
		if err := intentRows.Scan(&intent, &count); err != nil {
			return nil, fmt.Errorf("failed to scan intent stats: %w", err)
		}
		byIntent[intent] = count
	}
	if err := intentRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating intent rows: %w", err)
	}

	// Get breakdown by source
	sourceQuery := `SELECT source, COUNT(*) as count
		FROM router_feedback
		WHERE user_id = ` + placeholder(1) + ` AND timestamp >= ` + placeholder(2) + `
		GROUP BY source`

	sourceRows, err := d.db.QueryContext(ctx, sourceQuery, get.UserID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get router stats by source: %w", err)
	}
	defer sourceRows.Close()

	bySource := make(map[string]int64)
	for sourceRows.Next() {
		var source string
		var count int64
		if err := sourceRows.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("failed to scan source stats: %w", err)
		}
		bySource[source] = count
	}
	if err := sourceRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating source rows: %w", err)
	}

	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total)
	}

	return &store.RouterStats{
		TotalPredictions: total,
		CorrectCount:     correct,
		IncorrectCount:   incorrect,
		Accuracy:         accuracy,
		ByIntent:         byIntent,
		BySource:         bySource,
		LastUpdated:      time.Now().Unix(),
	}, nil
}

// UpsertRouterWeight upserts a router weight entry.
func (d *DB) UpsertRouterWeight(ctx context.Context, upsert *store.UpsertRouterWeight) error {
	now := time.Now().Unix()

	stmt := `INSERT INTO router_weight (user_id, category, keyword, weight, created_ts, updated_ts)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` + placeholder(5) + `, ` + placeholder(6) + `)
		ON CONFLICT (user_id, category, keyword) DO UPDATE SET
			weight = EXCLUDED.weight,
			updated_ts = EXCLUDED.updated_ts
		RETURNING user_id, category, keyword, weight, created_ts, updated_ts`

	_, err := d.db.ExecContext(ctx, stmt,
		upsert.UserID, upsert.Category, upsert.Keyword, upsert.Weight, now, now)
	if err != nil {
		return fmt.Errorf("failed to upsert router weight: %w", err)
	}

	return nil
}

// ListRouterWeights retrieves router weights for a user.
func (d *DB) ListRouterWeights(ctx context.Context, find *store.FindRouterWeight) ([]*store.RouterWeight, error) {
	query := `SELECT user_id, category, keyword, weight, created_ts, updated_ts
		FROM router_weight WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if find.UserID != nil {
		query += fmt.Sprintf(" AND user_id = %s", placeholder(argIdx))
		args = append(args, *find.UserID)
		argIdx++
	}
	if find.Category != nil {
		query += fmt.Sprintf(" AND category = %s", placeholder(argIdx))
		args = append(args, *find.Category)
	}

	query += " ORDER BY category, keyword"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list router weights: %w", err)
	}
	defer rows.Close()

	var weights []*store.RouterWeight
	for rows.Next() {
		var w store.RouterWeight
		err := rows.Scan(&w.UserID, &w.Category, &w.Keyword, &w.Weight, &w.CreatedTs, &w.UpdatedTs)
		if err != nil {
			return nil, fmt.Errorf("failed to scan router weight: %w", err)
		}
		weights = append(weights, &w)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating router weight rows: %w", err)
	}

	return weights, nil
}

// DeleteRouterWeights deletes router weights for a user and category.
func (d *DB) DeleteRouterWeights(ctx context.Context, userID int32, category string) error {
	stmt := `DELETE FROM router_weight WHERE user_id = ` + placeholder(1)
	args := []interface{}{userID}
	argIdx := 2

	if category != "" {
		stmt += fmt.Sprintf(" AND category = %s", placeholder(argIdx))
		args = append(args, category)
	}

	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("failed to delete router weights: %w", err)
	}

	return nil
}

// GetUserRouterWeightsMap retrieves router weights as a nested map for a user.
// Returns category -> keyword -> weight.
func (d *DB) GetUserRouterWeightsMap(ctx context.Context, userID int32) (map[string]map[string]int, error) {
	find := &store.FindRouterWeight{
		UserID: &userID,
	}

	weights, err := d.ListRouterWeights(ctx, find)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]int)
	for _, w := range weights {
		if result[w.Category] == nil {
			result[w.Category] = make(map[string]int)
		}
		result[w.Category][w.Keyword] = w.Weight
	}

	return result, nil
}

// SaveUserRouterWeights saves router weights from a nested map.
// This replaces all weights for the given user and categories.
func (d *DB) SaveUserRouterWeights(ctx context.Context, userID int32, weights map[string]map[string]int) error {
	now := time.Now().Unix()

	// Begin transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing weights for this user
	_, err = tx.ExecContext(ctx, "DELETE FROM router_weight WHERE user_id = "+placeholder(1), userID)
	if err != nil {
		return fmt.Errorf("failed to delete existing weights: %w", err)
	}

	// Insert new weights
	stmt := `INSERT INTO router_weight (user_id, category, keyword, weight, created_ts, updated_ts)
		VALUES (` + placeholder(1) + `, ` + placeholder(2) + `, ` + placeholder(3) + `, ` + placeholder(4) + `, ` + placeholder(5) + `, ` + placeholder(6) + `)`

	for category, keywords := range weights {
		for keyword, weight := range keywords {
			_, err = tx.ExecContext(ctx, stmt, userID, category, keyword, weight, now, now)
			if err != nil {
				return fmt.Errorf("failed to insert weight %s.%s: %w", category, keyword, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
