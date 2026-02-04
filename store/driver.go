package store

import (
	"context"
	"database/sql"
	"time"
)

// Driver is an interface for store driver.
// It contains all methods that store database driver should implement.
type Driver interface {
	GetDB() *sql.DB
	Close() error

	IsInitialized(ctx context.Context) (bool, error)

	// Activity model related methods.
	CreateActivity(ctx context.Context, create *Activity) (*Activity, error)
	ListActivities(ctx context.Context, find *FindActivity) ([]*Activity, error)

	// Attachment model related methods.
	CreateAttachment(ctx context.Context, create *Attachment) (*Attachment, error)
	ListAttachments(ctx context.Context, find *FindAttachment) ([]*Attachment, error)
	UpdateAttachment(ctx context.Context, update *UpdateAttachment) error
	DeleteAttachment(ctx context.Context, delete *DeleteAttachment) error

	// Memo model related methods.
	CreateMemo(ctx context.Context, create *Memo) (*Memo, error)
	ListMemos(ctx context.Context, find *FindMemo) ([]*Memo, error)
	UpdateMemo(ctx context.Context, update *UpdateMemo) error
	DeleteMemo(ctx context.Context, delete *DeleteMemo) error

	// UpdateMemoEmbedding updates the embedding vector for a memo.
	UpdateMemoEmbedding(ctx context.Context, id int32, embedding []float32) error

	// SearchMemosByVector performs semantic search using vector similarity.
	// Returns memos and their similarity scores.
	SearchMemosByVector(ctx context.Context, embedding []float32, limit int) ([]*Memo, []float32, error)

	// MemoRelation model related methods.
	UpsertMemoRelation(ctx context.Context, create *MemoRelation) (*MemoRelation, error)
	ListMemoRelations(ctx context.Context, find *FindMemoRelation) ([]*MemoRelation, error)
	DeleteMemoRelation(ctx context.Context, delete *DeleteMemoRelation) error

	// InstanceSetting model related methods.
	UpsertInstanceSetting(ctx context.Context, upsert *InstanceSetting) (*InstanceSetting, error)
	ListInstanceSettings(ctx context.Context, find *FindInstanceSetting) ([]*InstanceSetting, error)
	DeleteInstanceSetting(ctx context.Context, delete *DeleteInstanceSetting) error

	// User model related methods.
	CreateUser(ctx context.Context, create *User) (*User, error)
	UpdateUser(ctx context.Context, update *UpdateUser) (*User, error)
	ListUsers(ctx context.Context, find *FindUser) ([]*User, error)
	DeleteUser(ctx context.Context, delete *DeleteUser) error

	// UserSetting model related methods.
	UpsertUserSetting(ctx context.Context, upsert *UserSetting) (*UserSetting, error)
	ListUserSettings(ctx context.Context, find *FindUserSetting) ([]*UserSetting, error)
	GetUserByPATHash(ctx context.Context, tokenHash string) (*PATQueryResult, error)

	// IdentityProvider model related methods.
	CreateIdentityProvider(ctx context.Context, create *IdentityProvider) (*IdentityProvider, error)
	ListIdentityProviders(ctx context.Context, find *FindIdentityProvider) ([]*IdentityProvider, error)
	UpdateIdentityProvider(ctx context.Context, update *UpdateIdentityProvider) (*IdentityProvider, error)
	DeleteIdentityProvider(ctx context.Context, delete *DeleteIdentityProvider) error

	// Inbox model related methods.
	CreateInbox(ctx context.Context, create *Inbox) (*Inbox, error)
	ListInboxes(ctx context.Context, find *FindInbox) ([]*Inbox, error)
	UpdateInbox(ctx context.Context, update *UpdateInbox) (*Inbox, error)
	DeleteInbox(ctx context.Context, delete *DeleteInbox) error

	// Reaction model related methods.
	UpsertReaction(ctx context.Context, create *Reaction) (*Reaction, error)
	ListReactions(ctx context.Context, find *FindReaction) ([]*Reaction, error)
	GetReaction(ctx context.Context, find *FindReaction) (*Reaction, error)
	DeleteReaction(ctx context.Context, delete *DeleteReaction) error

	// MemoEmbedding model related methods.
	UpsertMemoEmbedding(ctx context.Context, embedding *MemoEmbedding) (*MemoEmbedding, error)
	ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error)
	DeleteMemoEmbedding(ctx context.Context, memoID int32) error
	FindMemosWithoutEmbedding(ctx context.Context, find *FindMemosWithoutEmbedding) ([]*Memo, error)
	VectorSearch(ctx context.Context, opts *VectorSearchOptions) ([]*MemoWithScore, error)
	BM25Search(ctx context.Context, opts *BM25SearchOptions) ([]*BM25Result, error)

	// Schedule model related methods.
	CreateSchedule(ctx context.Context, create *Schedule) (*Schedule, error)
	ListSchedules(ctx context.Context, find *FindSchedule) ([]*Schedule, error)
	UpdateSchedule(ctx context.Context, update *UpdateSchedule) error
	DeleteSchedule(ctx context.Context, delete *DeleteSchedule) error

	// AIConversation model related methods.
	CreateAIConversation(ctx context.Context, create *AIConversation) (*AIConversation, error)
	ListAIConversations(ctx context.Context, find *FindAIConversation) ([]*AIConversation, error)
	UpdateAIConversation(ctx context.Context, update *UpdateAIConversation) (*AIConversation, error)
	DeleteAIConversation(ctx context.Context, delete *DeleteAIConversation) error

	// AIMessage model related methods.
	CreateAIMessage(ctx context.Context, create *AIMessage) (*AIMessage, error)
	ListAIMessages(ctx context.Context, find *FindAIMessage) ([]*AIMessage, error)
	DeleteAIMessage(ctx context.Context, delete *DeleteAIMessage) error

	// AIBlock model related methods (Unified Block Model).
	CreateAIBlock(ctx context.Context, create *CreateAIBlock) (*AIBlock, error)
	GetAIBlock(ctx context.Context, id int64) (*AIBlock, error)
	ListAIBlocks(ctx context.Context, find *FindAIBlock) ([]*AIBlock, error)
	UpdateAIBlock(ctx context.Context, update *UpdateAIBlock) (*AIBlock, error)
	DeleteAIBlock(ctx context.Context, id int64) error
	AppendUserInput(ctx context.Context, blockID int64, input UserInput) error
	AppendEvent(ctx context.Context, blockID int64, event BlockEvent) error
	AppendEventsBatch(ctx context.Context, blockID int64, events []BlockEvent) error
	UpdateAIBlockStatus(ctx context.Context, blockID int64, status AIBlockStatus) error
	GetLatestAIBlock(ctx context.Context, conversationID int32) (*AIBlock, error)
	GetPendingAIBlocks(ctx context.Context) ([]*AIBlock, error)
	CreateAIBlockWithRound(ctx context.Context, create *CreateAIBlock) (*AIBlock, error)

	// CompleteBlock atomically marks a block as completed with content and stats.
	CompleteBlock(ctx context.Context, blockID int64, assistantContent string, sessionStats *SessionStats) error

	// EpisodicMemory model related methods.
	CreateEpisodicMemory(ctx context.Context, create *EpisodicMemory) (*EpisodicMemory, error)
	ListEpisodicMemories(ctx context.Context, find *FindEpisodicMemory) ([]*EpisodicMemory, error)
	ListActiveUserIDs(ctx context.Context, cutoff time.Time) ([]int32, error)
	DeleteEpisodicMemory(ctx context.Context, delete *DeleteEpisodicMemory) error

	// UserPreferences model related methods.
	UpsertUserPreferences(ctx context.Context, upsert *UpsertUserPreferences) (*UserPreferences, error)
	GetUserPreferences(ctx context.Context, find *FindUserPreferences) (*UserPreferences, error)

	// AgentMetrics model related methods.
	UpsertAgentMetrics(ctx context.Context, upsert *UpsertAgentMetrics) (*AgentMetrics, error)
	ListAgentMetrics(ctx context.Context, find *FindAgentMetrics) ([]*AgentMetrics, error)
	DeleteAgentMetrics(ctx context.Context, delete *DeleteAgentMetrics) error

	// ToolMetrics model related methods.
	UpsertToolMetrics(ctx context.Context, upsert *UpsertToolMetrics) (*ToolMetrics, error)
	ListToolMetrics(ctx context.Context, find *FindToolMetrics) ([]*ToolMetrics, error)
	DeleteToolMetrics(ctx context.Context, delete *DeleteToolMetrics) error

	// AgentStatsStore provides session statistics persistence.
	AgentStatsStore() AgentStatsStore

	// SecurityAuditStore provides security audit logging.
	SecurityAuditStore() SecurityAuditStore
}
