package store

import "context"

// AIBlock represents a conversation block (round)
type AIBlock struct {
	ID                 int64
	UID                string
	ConversationID     int32
	RoundNumber        int32
	BlockType          AIBlockType
	Mode               AIBlockMode
	UserInputs         []UserInput
	AssistantContent   string
	AssistantTimestamp int64
	EventStream        []BlockEvent
	SessionStats       *SessionStats
	CCSessionID        string
	Status             AIBlockStatus
	Metadata           map[string]any
	ParentBlockID      *int64 // Parent block ID for tree branching (null for root blocks)
	BranchPath         string // Branch path for ordering (e.g., "0/1/3")
	CreatedTs          int64
	UpdatedTs          int64
}

// AIBlockType represents the block type
type AIBlockType string

const (
	AIBlockTypeMessage          AIBlockType = "message"
	AIBlockTypeContextSeparator AIBlockType = "context_separator"
)

// AIBlockMode represents the AI mode
type AIBlockMode string

const (
	AIBlockModeNormal    AIBlockMode = "normal"
	AIBlockModeGeek      AIBlockMode = "geek"
	AIBlockModeEvolution AIBlockMode = "evolution"
)

// UserInput represents a single user input in the block
type UserInput struct {
	Content   string         `json:"content"`
	Timestamp int64          `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// BlockEvent represents an event in the event stream
type BlockEvent struct {
	Type      string         `json:"type"` // "thinking", "tool_use", "tool_result", "answer", "error"
	Content   string         `json:"content,omitempty"`
	Timestamp int64          `json:"timestamp"`
	Meta      map[string]any `json:"meta,omitempty"`
}

// AIBlockStatus represents the block status
type AIBlockStatus string

const (
	AIBlockStatusPending   AIBlockStatus = "pending"
	AIBlockStatusStreaming AIBlockStatus = "streaming"
	AIBlockStatusCompleted AIBlockStatus = "completed"
	AIBlockStatusError     AIBlockStatus = "error"
)

// SessionStats represents session statistics (compatible with agent_session_stats)
type SessionStats struct {
	SessionID            string   `json:"session_id"`
	UserID               int32    `json:"user_id"`
	AgentType            string   `json:"agent_type"`
	TotalDurationMs      int64    `json:"total_duration_ms"`
	ThinkingDurationMs   int64    `json:"thinking_duration_ms"`
	ToolDurationMs       int64    `json:"tool_duration_ms"`
	GenerationDurationMs int64    `json:"generation_duration_ms"`
	InputTokens          int      `json:"input_tokens"`
	OutputTokens         int      `json:"output_tokens"`
	CacheWriteTokens     int      `json:"cache_write_tokens"`
	CacheReadTokens      int      `json:"cache_read_tokens"`
	TotalTokens          int      `json:"total_tokens"`
	TotalCostUsd         float64  `json:"total_cost_usd"`
	ToolCallCount        int      `json:"tool_call_count"`
	ToolsUsed            []string `json:"tools_used,omitempty"`
	FilesModified        int      `json:"files_modified"`
	FilePaths            []string `json:"file_paths,omitempty"`
	ModelUsed            string   `json:"model_used,omitempty"`
	IsError              bool     `json:"is_error"`
	ErrorMessage         string   `json:"error_message,omitempty"`
}

// CreateAIBlock represents the input for creating a block
type CreateAIBlock struct {
	UID            string
	ConversationID int32
	BlockType      AIBlockType
	Mode           AIBlockMode
	UserInputs     []UserInput
	Metadata       map[string]any
	CCSessionID    string
	Status         AIBlockStatus
	ParentBlockID  *int64 // Parent block ID for forking (null for new root)
	CreatedTs      int64
	UpdatedTs      int64
}

// UpdateAIBlock represents the input for updating a block
type UpdateAIBlock struct {
	ID               int64
	UserInputs       *[]UserInput   // Replace user inputs
	AssistantContent *string        // Update AI response
	EventStream      *[]BlockEvent  // Replace event stream
	SessionStats     *SessionStats  // Update session stats
	CCSessionID      *string        // Update CC session ID
	Status           *AIBlockStatus // Update status
	Metadata         map[string]any // Merge metadata
	UpdatedTs        *int64         // Update timestamp
}

// FindAIBlock represents the filter for finding blocks
type FindAIBlock struct {
	ID             *int64
	UID            *string
	ConversationID *int32
	Status         *AIBlockStatus
	Mode           *AIBlockMode
	CCSessionID    *string
	ParentBlockID  *int64 // Filter by parent block (for branch queries)
}

// AIBlockStore defines the interface for block storage operations
type AIBlockStore interface {
	// CreateBlock creates a new block
	CreateBlock(ctx context.Context, create *CreateAIBlock) (*AIBlock, error)

	// GetBlock retrieves a block by ID
	GetBlock(ctx context.Context, id int64) (*AIBlock, error)

	// ListBlocks retrieves blocks for a conversation
	ListBlocks(ctx context.Context, find *FindAIBlock) ([]*AIBlock, error)

	// UpdateBlock updates a block
	UpdateBlock(ctx context.Context, update *UpdateAIBlock) (*AIBlock, error)

	// AppendUserInput appends a user input to an existing block
	AppendUserInput(ctx context.Context, blockID int64, input UserInput) error

	// AppendEvent appends an event to the event stream
	AppendEvent(ctx context.Context, blockID int64, event BlockEvent) error

	// AppendEventsBatch appends multiple events to the event stream in a single query
	AppendEventsBatch(ctx context.Context, blockID int64, events []BlockEvent) error

	// UpdateStatus updates the block status
	UpdateStatus(ctx context.Context, blockID int64, status AIBlockStatus) error

	// DeleteBlock deletes a block
	DeleteBlock(ctx context.Context, id int64) error

	// GetLatestBlock retrieves the latest block for a conversation
	GetLatestBlock(ctx context.Context, conversationID int32) (*AIBlock, error)

	// GetPendingBlocks retrieves all pending/streaming blocks for cleanup
	GetPendingBlocks(ctx context.Context) ([]*AIBlock, error)
}
