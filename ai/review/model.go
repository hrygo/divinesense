// Package review provides intelligent memo review system based on spaced repetition.
package review

import (
	"time"
)

// ReviewItem represents a memo in the review queue.
type ReviewItem struct {
	LastReview  time.Time `json:"last_review"`
	NextReview  time.Time `json:"next_review"`
	CreatedAt   time.Time `json:"created_at"`
	MemoID      string    `json:"memo_id"`
	MemoName    string    `json:"memo_name"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet"`
	Tags        []string  `json:"tags"`
	ReviewCount int       `json:"review_count"`
	Priority    float64   `json:"priority"`
}

// ReviewState stores the spaced repetition state for a memo.
type ReviewState struct {
	LastReview   time.Time `json:"last_review"`
	NextReview   time.Time `json:"next_review"`
	MemoUID      string    `json:"memo_uid"`
	ReviewCount  int       `json:"review_count"`
	EaseFactor   float64   `json:"ease_factor"`
	IntervalDays int       `json:"interval_days"`
}

// ReviewQuality represents the user's assessment of recall difficulty.
type ReviewQuality int

const (
	// QualityAgain - complete blackout, wrong response.
	QualityAgain ReviewQuality = 0
	// QualityHard - correct but with serious difficulty.
	QualityHard ReviewQuality = 1
	// QualityGood - correct with some hesitation.
	QualityGood ReviewQuality = 2
	// QualityEasy - perfect response.
	QualityEasy ReviewQuality = 3
)

// ReviewStats contains statistics about the user's review progress.
type ReviewStats struct {
	TotalMemos      int `json:"total_memos"`
	DueToday        int `json:"due_today"`
	ReviewedToday   int `json:"reviewed_today"`
	NewMemos        int `json:"new_memos"`
	MasteredMemos   int `json:"mastered_memos"` // interval > 30 days
	StreakDays      int `json:"streak_days"`
	TotalReviews    int `json:"total_reviews"`
	AverageAccuracy int `json:"average_accuracy"` // percentage
}

// DefaultEaseFactor is the initial ease factor for new items.
const DefaultEaseFactor = 2.5

// MinEaseFactor is the minimum ease factor to prevent intervals from getting too short.
const MinEaseFactor = 1.3

// ReviewConfig contains configuration for the review system.
type ReviewConfig struct {
	MaxDailyReviews     int
	NewCardsPerDay      int
	DailyReviewTime     int
	EnableNotifications bool
}

// DefaultConfig returns the default review configuration.
func DefaultConfig() ReviewConfig {
	return ReviewConfig{
		MaxDailyReviews:     20,
		NewCardsPerDay:      5,
		DailyReviewTime:     9,
		EnableNotifications: true,
	}
}
