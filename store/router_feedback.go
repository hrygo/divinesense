package store

import "time"

// RouterFeedback represents a single feedback event for a routing decision.
type RouterFeedback struct {
	ID        int64  `json:"id"`
	UserID    int32  `json:"user_id"`
	Input     string `json:"input"`
	Predicted string `json:"predicted"` // What the router predicted (as Intent string)
	Actual    string `json:"actual"`    // What the user actually wanted
	Feedback  string `json:"feedback"`  // FeedbackType: positive, rephrase, switch
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"` // "rule", "history", "llm"
}

// FindRouterFeedback specifies conditions for finding router feedback.
type FindRouterFeedback struct {
	UserID    *int32
	StartTime *int64
	EndTime   *int64
	Feedback  *string
	Limit     int
}

// RouterStats represents routing accuracy statistics.
type RouterStats struct {
	TotalPredictions int64            `json:"total_predictions"`
	CorrectCount     int64            `json:"correct_count"`
	IncorrectCount   int64            `json:"incorrect_count"`
	Accuracy         float64          `json:"accuracy"`
	ByIntent         map[string]int64 `json:"by_intent"`
	BySource         map[string]int64 `json:"by_source"`
	LastUpdated      int64            `json:"last_updated"`
}

// CreateRouterFeedback specifies data for creating a router feedback entry.
type CreateRouterFeedback struct {
	UserID    int32
	Input     string
	Predicted string
	Actual    string
	Feedback  string
	Timestamp int64
	Source    string
}

// RouterWeight represents a weight adjustment for a keyword.
type RouterWeight struct {
	UserID    int32  `json:"user_id"`
	Category  string `json:"category"` // "schedule", "memo", "amazing"
	Keyword   string `json:"keyword"`
	Weight    int    `json:"weight"`
	CreatedTs int64  `json:"created_ts"`
	UpdatedTs int64  `json:"updated_ts"`
}

// FindRouterWeight specifies conditions for finding router weights.
type FindRouterWeight struct {
	UserID   *int32
	Category *string
}

// UpsertRouterWeight specifies data for upserting router weights.
type UpsertRouterWeight struct {
	UserID   int32
	Category string
	Keyword  string
	Weight   int
}

// GetRouterStats specifies parameters for router statistics.
type GetRouterStats struct {
	UserID    int32
	TimeRange time.Duration
}
