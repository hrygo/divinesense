package store

import "context"

// MemoSummaryStatus represents the generation status of a memo summary.
type MemoSummaryStatus string

const (
	// MemoSummaryStatusPending means the summary has not been generated yet.
	MemoSummaryStatusPending MemoSummaryStatus = "PENDING"
	// MemoSummaryStatusGenerating means the summary is being generated.
	MemoSummaryStatusGenerating MemoSummaryStatus = "GENERATING"
	// MemoSummaryStatusCompleted means the summary has been generated successfully.
	MemoSummaryStatusCompleted MemoSummaryStatus = "COMPLETED"
	// MemoSummaryStatusFailed means the summary generation failed.
	MemoSummaryStatusFailed MemoSummaryStatus = "FAILED"
)

// MemoSummary represents the AI-generated summary of a memo.
type MemoSummary struct {
	ID           int32
	MemoID       int32
	Summary      string
	Status       MemoSummaryStatus
	ErrorMessage *string
	CreatedTs    int64
	UpdatedTs    int64
}

// FindMemoSummary is the find condition for memo summaries.
type FindMemoSummary struct {
	MemoID *int32
	ID     *int32
	Limit  *int
	Offset *int
	Status *MemoSummaryStatus
}

// UpsertMemoSummary is the upsert condition for memo summary.
type UpsertMemoSummary struct {
	MemoID       int32
	Summary      string
	Status       MemoSummaryStatus
	ErrorMessage *string
}

// GetMemoSummary gets the summary of a specific memo.
func (s *Store) GetMemoSummary(ctx context.Context, memoID int32) (*MemoSummary, error) {
	find := &FindMemoSummary{
		MemoID: &memoID,
	}
	list, err := s.ListMemoSummarys(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// UpsertMemoSummary inserts or updates a memo summary.
func (s *Store) UpsertMemoSummary(ctx context.Context, upsert *UpsertMemoSummary) (*MemoSummary, error) {
	return s.driver.UpsertMemoSummary(ctx, upsert)
}

// ListMemoSummarys lists memo summaries.
func (s *Store) ListMemoSummarys(ctx context.Context, find *FindMemoSummary) ([]*MemoSummary, error) {
	return s.driver.ListMemoSummarys(ctx, find)
}

// DeleteMemoSummary deletes a memo summary.
func (s *Store) DeleteMemoSummary(ctx context.Context, memoID int32) error {
	return s.driver.DeleteMemoSummary(ctx, memoID)
}

// FindMemosWithoutSummary finds memos that don't have summaries.
func (s *Store) FindMemosWithoutSummary(ctx context.Context, limit int) ([]*Memo, error) {
	return s.driver.FindMemosWithoutSummary(ctx, limit)
}
