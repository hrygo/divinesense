// Package review - service implementation for P3-C002.
package review

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// Service provides intelligent memo review functionality.
type Service struct {
	store  *store.Store
	config ReviewConfig
}

// NewService creates a new review service.
func NewService(s *store.Store) *Service {
	return &Service{
		store:  s,
		config: DefaultConfig(),
	}
}

// NewServiceWithConfig creates a service with custom configuration.
func NewServiceWithConfig(s *store.Store, config ReviewConfig) *Service {
	return &Service{
		store:  s,
		config: config,
	}
}

// GetDueReviews returns memos that are due for review, sorted by priority.
// Returns items (limited), total count of all due items, and error.
func (s *Service) GetDueReviews(ctx context.Context, userID int32, limit int) ([]ReviewItem, int, error) {
	now := time.Now()

	// Get all user's memos
	memos, err := s.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list memos: %w", err)
	}

	// Get review states from user settings
	states, err := s.getReviewStates(ctx, userID)
	if err != nil {
		slog.Warn("failed to get review states", "user_id", userID, "error", err)
		states = make(map[string]*ReviewState)
	}

	var items []ReviewItem

	for _, memo := range memos {
		// Get or create review state
		state := s.getOrCreateState(memo.UID, states)

		// Skip if not due yet
		if state.NextReview.After(now) {
			continue
		}

		item := ReviewItem{
			MemoID:      memo.UID,
			MemoName:    fmt.Sprintf("memos/%s", memo.UID),
			Title:       extractTitle(memo.Content),
			Snippet:     extractSnippet(memo.Content, 150),
			Tags:        extractTags(memo),
			LastReview:  state.LastReview,
			ReviewCount: state.ReviewCount,
			NextReview:  state.NextReview,
			CreatedAt:   time.Unix(memo.CreatedTs, 0),
		}

		// Calculate priority
		item.Priority = s.calculatePriority(item, now)
		items = append(items, item)
	}

	// Sort by priority (highest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority > items[j].Priority
	})

	// Store total count before limiting
	totalCount := len(items)

	// Apply limit (use config.MaxDailyReviews if limit not specified)
	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = s.config.MaxDailyReviews
	}
	if effectiveLimit > 0 && len(items) > effectiveLimit {
		items = items[:effectiveLimit]
	}

	return items, totalCount, nil
}

// RecordReview records a review and updates the spaced repetition state.
func (s *Service) RecordReview(ctx context.Context, userID int32, memoUID string, quality ReviewQuality) error {
	// Verify memo exists and belongs to user
	memos, err := s.store.ListMemos(ctx, &store.FindMemo{
		UID:       &memoUID,
		CreatorID: &userID,
	})
	if err != nil || len(memos) == 0 {
		return fmt.Errorf("memo not found: %s", memoUID)
	}

	// Get current states
	states, err := s.getReviewStates(ctx, userID)
	if err != nil {
		states = make(map[string]*ReviewState)
	}

	// Get or create state for this memo
	state := s.getOrCreateState(memoUID, states)

	// Apply SM-2 algorithm
	state = s.applySM2Algorithm(state, quality)

	// Update states map
	states[memoUID] = state

	// Persist to user settings
	return s.saveReviewStates(ctx, userID, states)
}

// GetReviewStats returns statistics about the user's review progress.
func (s *Service) GetReviewStats(ctx context.Context, userID int32) (*ReviewStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	memos, err := s.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("list memos: %w", err)
	}

	states, err := s.getReviewStates(ctx, userID)
	if err != nil {
		states = make(map[string]*ReviewState)
	}

	stats := &ReviewStats{
		TotalMemos: len(memos),
	}

	// Track review dates for streak calculation
	reviewDates := make(map[string]bool)

	for _, memo := range memos {
		state := s.getOrCreateState(memo.UID, states)

		// Due today
		if !state.NextReview.After(now) {
			stats.DueToday++
		}

		// Reviewed today
		if state.LastReview.After(today) {
			stats.ReviewedToday++
		}

		// New memos (never reviewed)
		if state.ReviewCount == 0 {
			stats.NewMemos++
		}

		// Mastered (interval > 30 days)
		if state.IntervalDays > 30 {
			stats.MasteredMemos++
		}

		stats.TotalReviews += state.ReviewCount

		// Track dates for streak calculation
		if !state.LastReview.IsZero() {
			dateKey := state.LastReview.Format("2006-01-02")
			reviewDates[dateKey] = true
		}
	}

	// Calculate streak days (consecutive days with reviews ending today or yesterday)
	stats.StreakDays = calculateStreak(reviewDates, today)

	// Average accuracy: estimate based on mastered ratio (simplified)
	if stats.TotalMemos > 0 {
		stats.AverageAccuracy = (stats.MasteredMemos * 100) / stats.TotalMemos
	}

	return stats, nil
}

// calculateStreak counts consecutive days with reviews ending today or yesterday.
func calculateStreak(reviewDates map[string]bool, today time.Time) int {
	streak := 0
	checkDate := today

	// Allow starting from today or yesterday
	if !reviewDates[checkDate.Format("2006-01-02")] {
		checkDate = checkDate.AddDate(0, 0, -1)
		if !reviewDates[checkDate.Format("2006-01-02")] {
			return 0
		}
	}

	// Count consecutive days backwards
	for reviewDates[checkDate.Format("2006-01-02")] {
		streak++
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	return streak
}

// getReviewStates retrieves all review states for a user from user settings.
func (s *Service) getReviewStates(ctx context.Context, userID int32) (map[string]*ReviewState, error) {
	userSetting, err := s.store.GetUserSetting(ctx, &store.FindUserSetting{
		UserID: &userID,
		Key:    storepb.UserSetting_REVIEW_STATES,
	})
	if err != nil {
		return nil, err
	}

	states := make(map[string]*ReviewState)
	if userSetting == nil {
		return states, nil
	}

	reviewStates := userSetting.GetReviewStates()
	if reviewStates == nil {
		return states, nil
	}

	for _, rs := range reviewStates.States {
		states[rs.MemoUid] = &ReviewState{
			MemoUID:      rs.MemoUid,
			ReviewCount:  int(rs.ReviewCount),
			LastReview:   time.Unix(rs.LastReviewTs, 0),
			NextReview:   time.Unix(rs.NextReviewTs, 0),
			EaseFactor:   rs.EaseFactor,
			IntervalDays: int(rs.IntervalDays),
		}
	}

	return states, nil
}

// getOrCreateState returns existing state or creates a new one.
func (s *Service) getOrCreateState(memoUID string, states map[string]*ReviewState) *ReviewState {
	if state, ok := states[memoUID]; ok {
		return state
	}
	// New memo: schedule first review for tomorrow
	return &ReviewState{
		MemoUID:      memoUID,
		EaseFactor:   DefaultEaseFactor,
		IntervalDays: 0,
		NextReview:   time.Now().AddDate(0, 0, 1),
	}
}

// saveReviewStates persists all review states to user settings.
func (s *Service) saveReviewStates(ctx context.Context, userID int32, states map[string]*ReviewState) error {
	pbStates := make([]*storepb.ReviewStatesUserSetting_ReviewState, 0, len(states))
	for _, state := range states {
		pbStates = append(pbStates, &storepb.ReviewStatesUserSetting_ReviewState{
			MemoUid:      state.MemoUID,
			ReviewCount:  int32(state.ReviewCount),
			LastReviewTs: state.LastReview.Unix(),
			NextReviewTs: state.NextReview.Unix(),
			EaseFactor:   state.EaseFactor,
			IntervalDays: int32(state.IntervalDays),
		})
	}

	_, err := s.store.UpsertUserSetting(ctx, &storepb.UserSetting{
		UserId: userID,
		Key:    storepb.UserSetting_REVIEW_STATES,
		Value: &storepb.UserSetting_ReviewStates{
			ReviewStates: &storepb.ReviewStatesUserSetting{
				States: pbStates,
			},
		},
	})
	return err
}

// applySM2Algorithm applies the SM-2 spaced repetition algorithm.
func (s *Service) applySM2Algorithm(state *ReviewState, quality ReviewQuality) *ReviewState {
	now := time.Now()

	// Update review count
	state.ReviewCount++
	state.LastReview = now

	// Calculate new ease factor
	// EF' = EF + (0.1 - (3 - q) * (0.08 + (3 - q) * 0.02))
	q := float64(quality)
	state.EaseFactor = state.EaseFactor + (0.1 - (3-q)*(0.08+(3-q)*0.02))
	if state.EaseFactor < MinEaseFactor {
		state.EaseFactor = MinEaseFactor
	}

	// Calculate new interval
	switch quality {
	case QualityAgain:
		// Reset to beginning
		state.IntervalDays = 1
	case QualityHard:
		// Reduce interval slightly
		state.IntervalDays = int(float64(state.IntervalDays) * 1.2)
		if state.IntervalDays < 1 {
			state.IntervalDays = 1
		}
	case QualityGood:
		// Standard progression
		if state.IntervalDays == 0 {
			state.IntervalDays = 1
		} else if state.IntervalDays == 1 {
			state.IntervalDays = 3
		} else {
			state.IntervalDays = int(float64(state.IntervalDays) * state.EaseFactor)
		}
	case QualityEasy:
		// Accelerated progression
		if state.IntervalDays == 0 {
			state.IntervalDays = 3
		} else {
			state.IntervalDays = int(float64(state.IntervalDays) * state.EaseFactor * 1.3)
		}
	}

	// Cap maximum interval at 365 days
	if state.IntervalDays > 365 {
		state.IntervalDays = 365
	}

	// Set next review date
	state.NextReview = now.AddDate(0, 0, state.IntervalDays)

	return state
}

// calculatePriority computes the review priority for sorting.
func (s *Service) calculatePriority(item ReviewItem, now time.Time) float64 {
	priority := 0.0

	// 1. Overdue factor (0-1.0)
	if !item.NextReview.IsZero() {
		overdueDays := now.Sub(item.NextReview).Hours() / 24
		if overdueDays > 0 {
			priority += math.Min(overdueDays*0.1, 1.0)
		}
	}

	// 2. Importance tags (0-0.5)
	importantTags := []string{"重要", "核心", "important", "key", "critical"}
	for _, tag := range item.Tags {
		tagLower := strings.ToLower(tag)
		for _, important := range importantTags {
			if strings.Contains(tagLower, important) {
				priority += 0.5
				break
			}
		}
	}

	// 3. New item bonus (0-0.3)
	if item.ReviewCount < 3 {
		priority += 0.3 * float64(3-item.ReviewCount) / 3.0
	}

	// 4. Recency penalty for very new content (-0.2)
	if time.Since(item.CreatedAt) < time.Hour {
		priority -= 0.2
	}

	return priority
}

// extractTitle extracts the title from memo content.
func extractTitle(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	title := strings.TrimSpace(lines[0])
	for strings.HasPrefix(title, "#") {
		title = strings.TrimPrefix(title, "#")
	}
	title = strings.TrimSpace(title)

	runes := []rune(title)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return title
}

// extractSnippet extracts a snippet from memo content.
func extractSnippet(content string, maxLen int) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) > 1 {
		content = strings.TrimSpace(lines[1])
	}

	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.TrimSpace(content)

	runes := []rune(content)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return content
}

// extractTags extracts tags from memo payload.
func extractTags(memo *store.Memo) []string {
	if memo == nil || memo.Payload == nil {
		return nil
	}
	return memo.Payload.Tags
}
