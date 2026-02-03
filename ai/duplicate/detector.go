// Package duplicate provides memo duplicate detection for P2-C002.
package duplicate

import (
	"context"
)

// DuplicateDetector detects duplicate and related memos.
type DuplicateDetector interface {
	// Detect finds duplicate and related memos for given content.
	Detect(ctx context.Context, req *DetectRequest) (*DetectResponse, error)

	// Merge merges source memo into target memo.
	Merge(ctx context.Context, userID int32, sourceID, targetID string) error

	// Link creates a bidirectional relation between two memos.
	Link(ctx context.Context, userID int32, memoID1, memoID2 string) error
}

// DetectRequest contains input for duplicate detection.
type DetectRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
	TopK    int      `json:"top_k,omitempty"`
	UserID  int32    `json:"user_id"`
}

// DetectResponse contains detection results.
type DetectResponse struct {
	Duplicates   []SimilarMemo `json:"duplicates,omitempty"`
	Related      []SimilarMemo `json:"related,omitempty"`
	LatencyMs    int64         `json:"latency_ms"`
	HasDuplicate bool          `json:"has_duplicate"`
	HasRelated   bool          `json:"has_related"`
}

// SimilarMemo represents a memo similar to the input.
type SimilarMemo struct {
	Breakdown  *Breakdown `json:"breakdown,omitempty"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Title      string     `json:"title"`
	Snippet    string     `json:"snippet"`
	Level      string     `json:"level"`
	SharedTags []string   `json:"shared_tags,omitempty"`
	Similarity float64    `json:"similarity"`
}

// Breakdown shows how similarity was calculated.
type Breakdown struct {
	Vector     float64 `json:"vector"`
	TagCoOccur float64 `json:"tag_co_occur"`
	TimeProx   float64 `json:"time_prox"`
}

// Thresholds for duplicate detection.
const (
	DuplicateThreshold = 0.9 // >90% = duplicate
	RelatedThreshold   = 0.7 // 70-90% = related
	DefaultTopK        = 5
)

// Weights for similarity calculation.
type Weights struct {
	Vector     float64
	TagCoOccur float64
	TimeProx   float64
}

// DefaultWeights are the default weights for similarity calculation.
var DefaultWeights = Weights{
	Vector:     0.5,
	TagCoOccur: 0.3,
	TimeProx:   0.2,
}
