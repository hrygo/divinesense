package store

type AIConversation struct {
	UID       string
	Title     string
	ParrotID  string
	RowStatus RowStatus
	CreatedTs int64
	UpdatedTs int64
	ID        int32
	CreatorID int32
	Pinned    bool
}

type FindAIConversation struct {
	ID        *int32
	UID       *string
	CreatorID *int32
	Pinned    *bool
	RowStatus *RowStatus
}

type UpdateAIConversation struct {
	Title     *string
	ParrotID  *string
	Pinned    *bool
	RowStatus *RowStatus
	UpdatedTs *int64
	ID        int32
}

type DeleteAIConversation struct {
	ID int32
}

// AIMessage types removed: ALL IN Block!
// Use AIBlock from ai_block.go instead for all conversation persistence.
