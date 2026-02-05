package sqlite

import (
	"context"
	"errors"

	"github.com/hrygo/divinesense/store"
)

// AI conversation features are NOT supported on SQLite.
// SQLite is intended for development/testing only and does not support
// the advanced AI features including conversation persistence, vector search,
// and reranking. Use PostgreSQL for production AI features.
//
// Ref: https://github.com/hrygo/divinesense/docs/dev-guides/BACKEND_DB.md

var (
	// ErrSQLiteAINotSupported is returned when AI features are requested on SQLite.
	ErrSQLiteAINotSupported = errors.New("AI conversation features are not supported on SQLite. Use PostgreSQL for production AI features")
)

func (d *DB) CreateAIConversation(ctx context.Context, create *store.AIConversation) (*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) ListAIConversations(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) UpdateAIConversation(ctx context.Context, update *store.UpdateAIConversation) (*store.AIConversation, error) {
	return nil, ErrSQLiteAINotSupported
}

func (d *DB) DeleteAIConversation(ctx context.Context, delete *store.DeleteAIConversation) error {
	return ErrSQLiteAINotSupported
}

// ai_message functions removed: ALL IN Block!
// Use AIBlock functions instead for message persistence.

// ListAIConversationsBasic is a minimal implementation for UI rendering only.
// Returns empty list since SQLite doesn't support AI conversations.
// This prevents the UI from breaking but doesn't provide actual functionality.
func (d *DB) ListAIConversationsBasic(ctx context.Context, find *store.FindAIConversation) ([]*store.AIConversation, error) {
	// Return empty list - UI will show "no conversations" message
	return []*store.AIConversation{}, nil
}
