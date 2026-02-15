package store

import "context"

// MemoTagSource represents the source of a tag.
type MemoTagSource string

const (
	// MemoTagSourceLLM is the LLM-generated tag source.
	MemoTagSourceLLM MemoTagSource = "llm"
	// MemoTagSourceRules is the rule-based tag source.
	MemoTagSourceRules MemoTagSource = "rules"
	// MemoTagSourceStatistics is the statistics-based tag source.
	MemoTagSourceStatistics MemoTagSource = "statistics"
	// MemoTagSourceUser is the user-provided tag source.
	MemoTagSourceUser MemoTagSource = "user"
)

// MemoTag represents a tag associated with a memo.
type MemoTag struct {
	ID         int32
	MemoID     int32
	Tag        string
	Confidence float32
	Source     MemoTagSource
	CreatedTs  int64
}

// FindMemoTag is the find condition for memo tags.
type FindMemoTag struct {
	MemoID *int32
	Tag    *string
	Limit  *int
	Offset *int
}

// UpsertMemoTag is the upsert condition for memo tag.
type UpsertMemoTag struct {
	MemoID     int32
	Tag        string
	Confidence float32
	Source     MemoTagSource
}

// UpsertMemoTag inserts or updates a memo tag.
func (s *Store) UpsertMemoTag(ctx context.Context, upsert *UpsertMemoTag) (*MemoTag, error) {
	return s.driver.UpsertMemoTag(ctx, upsert)
}

// UpsertMemoTags inserts or updates multiple memo tags.
func (s *Store) UpsertMemoTags(ctx context.Context, upserts []*UpsertMemoTag) error {
	return s.driver.UpsertMemoTags(ctx, upserts)
}

// ListMemoTags lists memo tags.
func (s *Store) ListMemoTags(ctx context.Context, find *FindMemoTag) ([]*MemoTag, error) {
	return s.driver.ListMemoTags(ctx, find)
}

// DeleteMemoTag deletes a memo tag.
func (s *Store) DeleteMemoTag(ctx context.Context, memoID int32, tag string) error {
	return s.driver.DeleteMemoTag(ctx, memoID, tag)
}

// DeleteAllMemoTags deletes all tags for a memo.
func (s *Store) DeleteAllMemoTags(ctx context.Context, memoID int32) error {
	return s.driver.DeleteAllMemoTags(ctx, memoID)
}
