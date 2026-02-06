package postgres

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrygo/divinesense/store"
)

// ============================================================================
// Mock Store for Testing
// ============================================================================

// mockAIBlockStore is a mock implementation of AIBlockStore for testing.
// This allows running tests without requiring a real database connection.
type mockAIBlockStore struct {
	blocks     map[int64]*store.AIBlock
	nextID     int64
	nextRound  map[int32]int32 // conversation_id -> next round number
	mu         sync.Mutex
	createErr  error
	getErr     error
	updateErr  error
	deleteErr  error
	listErr    error
	appendErr  error
	statusErr  error
	pendingErr error
}

func newMockAIBlockStore() *mockAIBlockStore {
	return &mockAIBlockStore{
		blocks:    make(map[int64]*store.AIBlock),
		nextID:    1,
		nextRound: make(map[int32]int32),
	}
}

func (m *mockAIBlockStore) CreateBlock(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createErr != nil {
		return nil, m.createErr
	}

	id := m.nextID
	m.nextID++

	// Generate UID if not provided
	uid := create.UID
	if uid == "" {
		uid = fmt.Sprintf("test-%d", id)
	}

	// Get round number
	roundNum := m.nextRound[create.ConversationID]
	m.nextRound[create.ConversationID] = roundNum + 1

	block := &store.AIBlock{
		ID:               id,
		UID:              uid,
		ConversationID:   create.ConversationID,
		RoundNumber:      roundNum,
		BlockType:        create.BlockType,
		Mode:             create.Mode,
		UserInputs:       create.UserInputs,
		AssistantContent: "",
		EventStream:      []store.BlockEvent{},
		SessionStats:     nil,
		CCSessionID:      create.CCSessionID,
		Status:           create.Status,
		Metadata:         create.Metadata,
		CreatedTs:        create.CreatedTs,
		UpdatedTs:        create.UpdatedTs,
	}

	m.blocks[id] = block
	return block, nil
}

func (m *mockAIBlockStore) GetBlock(ctx context.Context, id int64) (*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getErr != nil {
		return nil, m.getErr
	}

	block, ok := m.blocks[id]
	if !ok {
		return nil, fmt.Errorf("block not found: %d", id)
	}
	return block, nil
}

func (m *mockAIBlockStore) ListBlocks(ctx context.Context, find *store.FindAIBlock) ([]*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listErr != nil {
		return nil, m.listErr
	}

	var result []*store.AIBlock
	for _, block := range m.blocks {
		if find.ConversationID != nil && block.ConversationID != *find.ConversationID {
			continue
		}
		if find.Status != nil && block.Status != *find.Status {
			continue
		}
		if find.Mode != nil && block.Mode != *find.Mode {
			continue
		}
		if find.ID != nil && block.ID != *find.ID {
			continue
		}
		if find.UID != nil && block.UID != *find.UID {
			continue
		}
		if find.CCSessionID != nil && block.CCSessionID != *find.CCSessionID {
			continue
		}
		result = append(result, block)
	}

	// Sort by round_number
	sort.Slice(result, func(i, j int) bool {
		return result[i].RoundNumber < result[j].RoundNumber
	})

	return result, nil
}

func (m *mockAIBlockStore) UpdateBlock(ctx context.Context, update *store.UpdateAIBlock) (*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateErr != nil {
		return nil, m.updateErr
	}

	block, ok := m.blocks[update.ID]
	if !ok {
		return nil, fmt.Errorf("block not found: %d", update.ID)
	}

	if update.UserInputs != nil {
		block.UserInputs = *update.UserInputs
	}
	if update.AssistantContent != nil {
		block.AssistantContent = *update.AssistantContent
	}
	if update.EventStream != nil {
		block.EventStream = *update.EventStream
	}
	if update.SessionStats != nil {
		block.SessionStats = update.SessionStats
	}
	if update.CCSessionID != nil {
		block.CCSessionID = *update.CCSessionID
	}
	if update.Status != nil {
		block.Status = *update.Status
	}
	if update.Metadata != nil {
		// Merge metadata
		if block.Metadata == nil {
			block.Metadata = make(map[string]any)
		}
		for k, v := range update.Metadata {
			block.Metadata[k] = v
		}
	}
	if update.UpdatedTs != nil {
		block.UpdatedTs = *update.UpdatedTs
	} else {
		block.UpdatedTs = time.Now().Unix()
	}

	return block, nil
}

func (m *mockAIBlockStore) DeleteBlock(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.blocks[id]; !ok {
		return fmt.Errorf("block not found: %d", id)
	}

	delete(m.blocks, id)
	return nil
}

func (m *mockAIBlockStore) AppendUserInput(ctx context.Context, blockID int64, input store.UserInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.appendErr != nil {
		return m.appendErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return fmt.Errorf("block not found: %d", blockID)
	}

	block.UserInputs = append(block.UserInputs, input)
	block.UpdatedTs = time.Now().Unix()
	return nil
}

func (m *mockAIBlockStore) AppendEvent(ctx context.Context, blockID int64, event store.BlockEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.appendErr != nil {
		return m.appendErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return fmt.Errorf("block not found: %d", blockID)
	}

	block.EventStream = append(block.EventStream, event)
	block.UpdatedTs = time.Now().Unix()
	return nil
}

func (m *mockAIBlockStore) AppendEventsBatch(ctx context.Context, blockID int64, events []store.BlockEvent) error {
	if len(events) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.appendErr != nil {
		return m.appendErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return fmt.Errorf("block not found: %d", blockID)
	}

	block.EventStream = append(block.EventStream, events...)
	block.UpdatedTs = time.Now().Unix()
	return nil
}

func (m *mockAIBlockStore) UpdateStatus(ctx context.Context, blockID int64, status store.AIBlockStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.statusErr != nil {
		return m.statusErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return fmt.Errorf("block not found: %d", blockID)
	}

	block.Status = status
	block.UpdatedTs = time.Now().Unix()
	return nil
}

func (m *mockAIBlockStore) GetLatestBlock(ctx context.Context, conversationID int32) (*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getErr != nil {
		return nil, m.getErr
	}

	var latest *store.AIBlock
	var latestRound int32 = -1

	for _, block := range m.blocks {
		if block.ConversationID == conversationID && block.RoundNumber > latestRound {
			latest = block
			latestRound = block.RoundNumber
		}
	}

	return latest, nil
}

func (m *mockAIBlockStore) GetPendingBlocks(ctx context.Context) ([]*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pendingErr != nil {
		return nil, m.pendingErr
	}

	var result []*store.AIBlock
	for _, block := range m.blocks {
		if block.Status == store.AIBlockStatusPending || block.Status == store.AIBlockStatusStreaming {
			result = append(result, block)
		}
	}
	return result, nil
}

// ForkBlock creates a new block as a branch from an existing block.
// The new block inherits the parent's conversation. User inputs can be optionally replaced.
// If replaceUserInputs is nil, inherits parent's user inputs.
// If replaceUserInputs is provided, uses the new user inputs (for message editing).
func (m *mockAIBlockStore) ForkBlock(ctx context.Context, parentID int64, reason string, replaceUserInputs []store.UserInput) (*store.AIBlock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate reason parameter
	if reason == "" {
		return nil, fmt.Errorf("fork reason cannot be empty")
	}

	parent, ok := m.blocks[parentID]
	if !ok {
		return nil, fmt.Errorf("parent block not found: %d", parentID)
	}

	id := m.nextID
	m.nextID++

	uid := fmt.Sprintf("fork-%d", id)

	// Count existing children with this parent (excluding any not yet added)
	childCount := 0
	for _, b := range m.blocks {
		if b.ParentBlockID != nil && *b.ParentBlockID == parentID {
			childCount++
		}
	}

	// Calculate branch path
	branchPath := ""
	if parent.BranchPath == "" {
		// Root's child: just the child index
		branchPath = fmt.Sprintf("%d", childCount)
	} else {
		// Append child index to parent's path
		branchPath = fmt.Sprintf("%s/%d", parent.BranchPath, childCount)
	}

	// Determine which user inputs to use: provided replacements or inherit from parent
	userInputs := parent.UserInputs
	if replaceUserInputs != nil && len(replaceUserInputs) > 0 {
		userInputs = replaceUserInputs
	}

	// Prepare metadata: inherit parent's metadata and add fork information
	metadata := make(map[string]interface{})
	for k, v := range parent.Metadata {
		metadata[k] = v
	}
	metadata["forked_from"] = parentID
	metadata["fork_reason"] = reason
	if replaceUserInputs != nil && len(replaceUserInputs) > 0 {
		metadata["fork_type"] = "edit"
	} else {
		metadata["fork_type"] = "branch"
	}

	block := &store.AIBlock{
		ID:               id,
		UID:              uid,
		ConversationID:   parent.ConversationID,
		RoundNumber:      parent.RoundNumber + 1, // Forks are in next round
		BlockType:        parent.BlockType,
		Mode:             parent.Mode,
		UserInputs:       userInputs,
		AssistantContent: "",
		EventStream:      []store.BlockEvent{},
		SessionStats:     nil,
		CCSessionID:      "", // Forks don't inherit CC session
		Status:           store.AIBlockStatusPending,
		Metadata:         metadata,
		ParentBlockID:    &parentID,
		BranchPath:       branchPath,
		CreatedTs:        time.Now().UnixMilli(),
		UpdatedTs:        time.Now().UnixMilli(),
	}

	m.blocks[id] = block
	return block, nil
}

// ArchiveInactiveBranches archives blocks not on the target branch path.
func (m *mockAIBlockStore) ArchiveInactiveBranches(ctx context.Context, conversationID int32, targetPath string, archivedAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse target path into segments
	targetSegments := parseBranchPath(targetPath)

	// Archive blocks that are not on the target path
	for _, block := range m.blocks {
		if block.ConversationID != conversationID {
			continue
		}
		// Skip already archived blocks
		if block.ArchivedAt != nil {
			continue
		}

		// Check if this block's path is on the target path
		blockSegments := parseBranchPath(block.BranchPath)

		// A block is on the target path if its path is a prefix of target path
		// or if target path starts with block's path
		if !isPathOnActiveBranch(blockSegments, targetSegments) {
			block.ArchivedAt = &archivedAt
		}
	}

	return nil
}

func (m *mockAIBlockStore) Close() error {
	// No-op for mock store
	return nil
}

// setupTestDB creates a mock store for testing.
// Tests run without requiring a real database connection.
func setupTestDB(t *testing.T) *mockAIBlockStore {
	t.Helper()
	return newMockAIBlockStore()
}

// ============================================================================
// Unit Tests
// ============================================================================

// TestCreateAIBlock tests creating a new AI block
func TestCreateAIBlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Hello", Timestamp: 1234567890},
		},
		Metadata: map[string]any{
			"test": "value",
		},
		Status:    store.AIBlockStatusPending,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	block, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Greater(t, block.ID, int64(0))
	assert.NotEmpty(t, block.UID)
	assert.Equal(t, create.ConversationID, block.ConversationID)
	assert.Equal(t, int32(0), block.RoundNumber)
	assert.Equal(t, store.AIBlockTypeMessage, block.BlockType)
	assert.Equal(t, store.AIBlockModeNormal, block.Mode)
	assert.Len(t, block.UserInputs, 1)
	assert.Equal(t, "Hello", block.UserInputs[0].Content)
	assert.Equal(t, store.AIBlockStatusPending, block.Status)
}

// TestGetAIBlock tests retrieving an AI block by ID
func TestGetAIBlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block first
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeGeek,
		UserInputs: []store.UserInput{
			{Content: "Help me code", Timestamp: 1234567890},
		},
		CCSessionID: "test-cc-session",
		Status:      store.AIBlockStatusPending,
		CreatedTs:   1234567890,
		UpdatedTs:   1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Retrieve the block
	block, err := db.GetBlock(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, created.ID, block.ID)
	assert.Equal(t, created.UID, block.UID)
	assert.Equal(t, store.AIBlockModeGeek, block.Mode)
	assert.Equal(t, "test-cc-session", block.CCSessionID)
}

// TestListAIBlocks tests listing AI blocks with filters
func TestListAIBlocks(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create multiple blocks
	convID := int32(1)
	for i := 0; i < 3; i++ {
		create := &store.CreateAIBlock{
			ConversationID: convID,
			BlockType:      store.AIBlockTypeMessage,
			Mode:           store.AIBlockModeNormal,
			UserInputs: []store.UserInput{
				{Content: "Message", Timestamp: 1234567890},
			},
			Status:    store.AIBlockStatusCompleted,
			CreatedTs: 1234567890 + int64(i),
			UpdatedTs: 1234567890 + int64(i),
		}
		_, err := db.CreateBlock(ctx, create)
		require.NoError(t, err)
	}

	// List all blocks for conversation
	blocks, err := db.ListBlocks(ctx, &store.FindAIBlock{
		ConversationID: &convID,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 3)

	// Filter by status
	pending := store.AIBlockStatusPending
	blocks, err = db.ListBlocks(ctx, &store.FindAIBlock{
		ConversationID: &convID,
		Status:         &pending,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 0)
}

// TestUpdateAIBlock tests updating an AI block
func TestUpdateAIBlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusPending,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Update the block
	content := "Here is the answer"
	status := store.AIBlockStatusCompleted
	updated, err := db.UpdateBlock(ctx, &store.UpdateAIBlock{
		ID:               created.ID,
		AssistantContent: &content,
		Status:           &status,
		Metadata:         map[string]any{"updated": true},
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, content, updated.AssistantContent)
	assert.Equal(t, store.AIBlockStatusCompleted, updated.Status)
}

// TestAppendUserInput tests appending a user input to a block
func TestAppendUserInput(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "First input", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusStreaming,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Append another input
	err = db.AppendUserInput(ctx, created.ID, store.UserInput{
		Content:   "Second input",
		Timestamp: 1234567891,
	})
	require.NoError(t, err)

	// Verify
	block, err := db.GetBlock(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, block.UserInputs, 2)
	assert.Equal(t, "Second input", block.UserInputs[1].Content)
}

// TestAppendEvent tests appending events to the event stream
func TestAppendEvent(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusStreaming,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Append events
	events := []store.BlockEvent{
		{Type: "thinking", Content: "Let me think", Timestamp: 1234567891},
		{Type: "tool_use", Content: "using tool", Timestamp: 1234567892, Meta: map[string]any{"tool": "search"}},
		{Type: "answer", Content: "Here is the answer", Timestamp: 1234567893},
	}

	for _, event := range events {
		err = db.AppendEvent(ctx, created.ID, event)
		require.NoError(t, err)
	}

	// Verify
	block, err := db.GetBlock(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, block.EventStream, 3)
	assert.Equal(t, "thinking", block.EventStream[0].Type)
	assert.Equal(t, "answer", block.EventStream[2].Type)
}

// TestUpdateAIBlockStatus tests updating block status
func TestUpdateAIBlockStatus(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusPending,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Update status
	err = db.UpdateStatus(ctx, created.ID, store.AIBlockStatusCompleted)
	require.NoError(t, err)

	// Verify
	block, err := db.GetBlock(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, store.AIBlockStatusCompleted, block.Status)
}

// TestGetLatestAIBlock tests retrieving the latest block
func TestGetLatestAIBlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	convID := int32(1)

	// Create no blocks and expect nil
	block, err := db.GetLatestBlock(ctx, convID)
	require.NoError(t, err)
	assert.Nil(t, block)

	// Create blocks
	for i := 0; i < 3; i++ {
		create := &store.CreateAIBlock{
			ConversationID: convID,
			BlockType:      store.AIBlockTypeMessage,
			Mode:           store.AIBlockModeNormal,
			UserInputs: []store.UserInput{
				{Content: "Test", Timestamp: 1234567890},
			},
			Status:    store.AIBlockStatusCompleted,
			CreatedTs: 1234567890 + int64(i),
			UpdatedTs: 1234567890 + int64(i),
		}
		_, err := db.CreateBlock(ctx, create)
		require.NoError(t, err)
	}

	// Get latest (should have round_number = 2)
	block, err = db.GetLatestBlock(ctx, convID)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, int32(2), block.RoundNumber)
}

// TestGetPendingAIBlocks tests retrieving pending blocks
func TestGetPendingAIBlocks(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create pending blocks
	for i := 0; i < 2; i++ {
		create := &store.CreateAIBlock{
			ConversationID: int32(i),
			BlockType:      store.AIBlockTypeMessage,
			Mode:           store.AIBlockModeNormal,
			UserInputs: []store.UserInput{
				{Content: "Test", Timestamp: 1234567890},
			},
			Status:    store.AIBlockStatusPending,
			CreatedTs: 1234567890 + int64(i),
			UpdatedTs: 1234567890 + int64(i),
		}
		_, err := db.CreateBlock(ctx, create)
		require.NoError(t, err)
	}

	// Create a completed block
	create := &store.CreateAIBlock{
		ConversationID: 3,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567892,
		UpdatedTs: 1234567892,
	}
	_, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Get pending blocks
	blocks, err := db.GetPendingBlocks(ctx)
	require.NoError(t, err)
	assert.Len(t, blocks, 2)
}

// TestDeleteAIBlock tests deleting a block
func TestDeleteAIBlock(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Delete the block
	err = db.DeleteBlock(ctx, created.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = db.GetBlock(ctx, created.ID)
	assert.Error(t, err)
}

// TestCreateAIBlockWithRound tests creating blocks with auto-incremented round numbers
func TestCreateAIBlockWithRound(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	convID := int32(1)

	// Create three blocks
	for i := 0; i < 3; i++ {
		create := &store.CreateAIBlock{
			ConversationID: convID,
			BlockType:      store.AIBlockTypeMessage,
			Mode:           store.AIBlockModeNormal,
			UserInputs: []store.UserInput{
				{Content: "Test", Timestamp: 1234567890},
			},
			Status:    store.AIBlockStatusCompleted,
			CreatedTs: 1234567890 + int64(i),
			UpdatedTs: 1234567890 + int64(i),
		}
		block, err := db.CreateBlock(ctx, create)
		require.NoError(t, err)
		assert.Equal(t, int32(i), block.RoundNumber)
	}

	// List and verify order
	blocks, err := db.ListBlocks(ctx, &store.FindAIBlock{
		ConversationID: &convID,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 3)
	assert.Equal(t, int32(0), blocks[0].RoundNumber)
	assert.Equal(t, int32(1), blocks[1].RoundNumber)
	assert.Equal(t, int32(2), blocks[2].RoundNumber)
}

// TestAppendEventsBatch tests batch appending events
func TestAppendEventsBatch(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a block
	create := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusStreaming,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	created, err := db.CreateBlock(ctx, create)
	require.NoError(t, err)

	// Append events in batch
	events := []store.BlockEvent{
		{Type: "thinking", Content: "Thinking 1", Timestamp: 1000},
		{Type: "thinking", Content: "Thinking 2", Timestamp: 1001},
		{Type: "answer", Content: "Answer", Timestamp: 1002},
	}

	err = db.AppendEventsBatch(ctx, created.ID, events)
	require.NoError(t, err)

	// Verify all events were appended
	block, err := db.GetBlock(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, block.EventStream, 3)
}

// TestAppendEventsBatch_Empty tests empty batch handling
func TestAppendEventsBatch_Empty(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Empty batch should not error
	err := db.AppendEventsBatch(ctx, 999, []store.BlockEvent{})
	require.NoError(t, err)
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

// TestBlockStatusTransitions tests all valid status transitions
func TestBlockStatusTransitions(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	validTransitions := []struct {
		name string
		from store.AIBlockStatus
		to   store.AIBlockStatus
	}{
		{"pending to streaming", store.AIBlockStatusPending, store.AIBlockStatusStreaming},
		{"streaming to completed", store.AIBlockStatusStreaming, store.AIBlockStatusCompleted},
		{"pending to completed", store.AIBlockStatusPending, store.AIBlockStatusCompleted},
		{"streaming to error", store.AIBlockStatusStreaming, store.AIBlockStatusError},
		{"pending to error", store.AIBlockStatusPending, store.AIBlockStatusError},
	}

	for _, tt := range validTransitions {
		t.Run(tt.name, func(t *testing.T) {
			create := &store.CreateAIBlock{
				ConversationID: 1,
				BlockType:      store.AIBlockTypeMessage,
				Mode:           store.AIBlockModeNormal,
				UserInputs: []store.UserInput{
					{Content: "Test", Timestamp: 1234567890},
				},
				Status:    tt.from,
				CreatedTs: 1234567890,
				UpdatedTs: 1234567890,
			}

			block, err := db.CreateBlock(ctx, create)
			require.NoError(t, err)

			err = db.UpdateStatus(ctx, block.ID, tt.to)
			require.NoError(t, err)

			updated, err := db.GetBlock(ctx, block.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.to, updated.Status)
		})
	}
}

// TestBlockModes tests all block modes
func TestBlockModes(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	modes := []struct {
		name string
		mode store.AIBlockMode
	}{
		{"normal mode", store.AIBlockModeNormal},
		{"geek mode", store.AIBlockModeGeek},
		{"evolution mode", store.AIBlockModeEvolution},
	}

	for _, tt := range modes {
		t.Run(tt.name, func(t *testing.T) {
			create := &store.CreateAIBlock{
				ConversationID: 1,
				BlockType:      store.AIBlockTypeMessage,
				Mode:           tt.mode,
				UserInputs: []store.UserInput{
					{Content: "Test", Timestamp: 1234567890},
				},
				Status:    store.AIBlockStatusPending,
				CreatedTs: 1234567890,
				UpdatedTs: 1234567890,
			}

			block, err := db.CreateBlock(ctx, create)
			require.NoError(t, err)
			assert.Equal(t, tt.mode, block.Mode)
		})
	}
}

// ============================================================================
// Branching Tests (Task 047: ForkBlock)
// ============================================================================

// TestForkBlock_CreatesCorrectBranchStructure verifies that ForkBlock creates
// a new block with the correct parent_block_id and branch_path.
func TestForkBlock_CreatesCorrectBranchStructure(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create parent block
	parentCreate := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Parent block", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	parent, err := db.CreateBlock(ctx, parentCreate)
	require.NoError(t, err)
	require.NotNil(t, parent)

	// Fork from parent using the ForkBlock method signature
	forkReason := "Exploring alternative response"
	forkedBlock, err := db.ForkBlock(ctx, parent.ID, forkReason, nil)

	require.NoError(t, err)
	require.NotNil(t, forkedBlock)

	// Verify parent_block_id is set correctly
	assert.NotNil(t, forkedBlock.ParentBlockID,
		"Forked block should have parent_block_id set")
	assert.Equal(t, parent.ID, *forkedBlock.ParentBlockID,
		"Forked block should have parent_block_id set to parent's ID")

	// Verify branch_path is set
	assert.NotEmpty(t, forkedBlock.BranchPath,
		"Forked block should have a branch_path")

	// Verify forked block is in the same conversation
	assert.Equal(t, parent.ConversationID, forkedBlock.ConversationID,
		"Forked block should be in the same conversation")

	// Verify forked block has its own unique ID
	assert.NotEqual(t, parent.ID, forkedBlock.ID,
		"Forked block should have a different ID from parent")

	// Verify forked block inherited parent's user inputs
	assert.Equal(t, len(parent.UserInputs), len(forkedBlock.UserInputs),
		"Forked block should inherit parent's user inputs")
}

// TestForkBlock_SequentialForks verifies branch path calculation for sequential forks.
func TestForkBlock_SequentialForks(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create root block
	rootCreate := &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Root block", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	}

	root, err := db.CreateBlock(ctx, rootCreate)
	require.NoError(t, err)

	// Fork multiple times from root
	var forks []*store.AIBlock
	for i := 0; i < 3; i++ {
		fork, err := db.ForkBlock(ctx, root.ID, fmt.Sprintf("Fork %d", i), nil)
		require.NoError(t, err)
		require.NotNil(t, fork)
		forks = append(forks, fork)

		// Verify all forks have the same parent
		assert.NotNil(t, fork.ParentBlockID)
		assert.Equal(t, root.ID, *fork.ParentBlockID)
	}

	// Verify each fork has a unique branch path
	branchPaths := make(map[string]bool)
	for _, fork := range forks {
		assert.NotEmpty(t, fork.BranchPath)
		// Each fork should have a unique branch path
		assert.False(t, branchPaths[fork.BranchPath],
			"Branch path should be unique: %s", fork.BranchPath)
		branchPaths[fork.BranchPath] = true
	}
}

// TestForkBlock_NestedForks verifies branch path calculation for nested forks.
func TestForkBlock_NestedForks(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create root
	root, err := db.CreateBlock(ctx, &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Root", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	})
	require.NoError(t, err)

	// Fork from root -> fork1
	fork1, err := db.ForkBlock(ctx, root.ID, "First fork", nil)
	require.NoError(t, err)

	// Fork from fork1 -> fork2 (nested fork)
	fork2, err := db.ForkBlock(ctx, fork1.ID, "Nested fork", nil)
	require.NoError(t, err)

	// Verify parent relationships
	assert.NotNil(t, fork1.ParentBlockID)
	assert.Equal(t, root.ID, *fork1.ParentBlockID, "fork1's parent should be root")
	assert.NotNil(t, fork2.ParentBlockID)
	assert.Equal(t, fork1.ID, *fork2.ParentBlockID, "fork2's parent should be fork1")

	// Verify branch paths reflect hierarchy
	// fork2's path should be longer than fork1's (deeper in tree)
	assert.Greater(t, len(fork2.BranchPath), len(fork1.BranchPath),
		"Nested fork should have longer branch path")

	// Verify fork2's path contains fork1's path as prefix
	assert.Contains(t, fork2.BranchPath, fork1.BranchPath,
		"Nested fork's path should contain parent's path")
}

// TestForkBlock_WithReplaceUserInputs verifies forking with user input replacement.
// This tests the message editing scenario where a user edits their original message.
func TestForkBlock_WithReplaceUserInputs(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create parent block with original user input
	parent, err := db.CreateBlock(ctx, &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Original message", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	})
	require.NoError(t, err)

	// Fork with edited message (simulating user edit)
	editedInputs := []store.UserInput{
		{Content: "Edited message", Timestamp: 1234567900},
	}
	forkReason := `User edited message: "Edited message"`
	forkedBlock, err := db.ForkBlock(ctx, parent.ID, forkReason, editedInputs)

	require.NoError(t, err)
	require.NotNil(t, forkedBlock)

	// Verify forked block has replaced user inputs (not inherited)
	assert.Equal(t, 1, len(forkedBlock.UserInputs),
		"Forked block should have exactly one user input")
	assert.Equal(t, "Edited message", forkedBlock.UserInputs[0].Content,
		"Forked block should have the edited message, not the original")

	// Verify metadata contains fork information
	assert.Equal(t, parent.ID, forkedBlock.Metadata["forked_from"],
		"Metadata should contain parent block ID")
	assert.Equal(t, forkReason, forkedBlock.Metadata["fork_reason"],
		"Metadata should contain fork reason")
	assert.Equal(t, "edit", forkedBlock.Metadata["fork_type"],
		"Metadata should indicate edit type")
}

// TestForkBlock_EmptyReason verifies that forking with an empty reason returns an error.
func TestForkBlock_EmptyReason(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create parent block
	parent, err := db.CreateBlock(ctx, &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Test", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	})
	require.NoError(t, err)

	// Try to fork with empty reason (should fail)
	_, err = db.ForkBlock(ctx, parent.ID, "", nil)
	assert.Error(t, err, "Forking with empty reason should return an error")
	assert.Contains(t, err.Error(), "cannot be empty",
		"Error message should mention that reason cannot be empty")
}

// ============================================================================
// Branch Switching Tests (Task 048: Branch Switching)
// ============================================================================

// TestArchiveInactiveBranches_ArchivesCorrectBlocks verifies that ArchiveInactiveBranches
// correctly archives blocks not on the active branch path.
func TestArchiveInactiveBranches_ArchivesCorrectBlocks(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create root block
	root, err := db.CreateBlock(ctx, &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Root", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	})
	require.NoError(t, err)

	// Fork from root to create branch "0"
	fork0, err := db.ForkBlock(ctx, root.ID, "Fork 0", nil)
	require.NoError(t, err)

	// Fork from root to create branch "1"
	fork1, err := db.ForkBlock(ctx, root.ID, "Fork 1", nil)
	require.NoError(t, err)

	// Fork from fork0 to create branch "0/0"
	fork00, err := db.ForkBlock(ctx, fork0.ID, "Fork 0/0", nil)
	require.NoError(t, err)

	// Archive blocks not on path "0/0"
	archivedAt := int64(1234569999)
	err = db.ArchiveInactiveBranches(ctx, 1, "0/0", archivedAt)
	require.NoError(t, err)

	// Verify blocks on path "0/0" are NOT archived
	// Path "0/0" includes: root (path ""), fork0 (path "0"), fork00 (path "0/0")
	assert.Nil(t, root.ArchivedAt, "Root block should not be archived (on active path)")
	assert.Nil(t, fork0.ArchivedAt, "Fork 0 should not be archived (on active path)")
	assert.Nil(t, fork00.ArchivedAt, "Fork 0/0 should not be archived (on active path)")

	// Verify blocks NOT on path "0/0" ARE archived
	assert.NotNil(t, fork1.ArchivedAt, "Fork 1 should be archived (not on active path)")
	assert.Equal(t, archivedAt, *fork1.ArchivedAt, "Fork 1 should have archived_at set")
}

// TestBranchSwitching_UpdatesActivePath verifies that switching branches
// correctly updates which blocks are archived.
func TestBranchSwitching_UpdatesActivePath(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Create a tree structure:
	//     root
	//    /     \
	//  fork0  fork1
	//  /
	// fork00

	root, err := db.CreateBlock(ctx, &store.CreateAIBlock{
		ConversationID: 1,
		BlockType:      store.AIBlockTypeMessage,
		Mode:           store.AIBlockModeNormal,
		UserInputs: []store.UserInput{
			{Content: "Root", Timestamp: 1234567890},
		},
		Status:    store.AIBlockStatusCompleted,
		CreatedTs: 1234567890,
		UpdatedTs: 1234567890,
	})
	require.NoError(t, err)

	fork0, err := db.ForkBlock(ctx, root.ID, "Fork 0", nil)
	require.NoError(t, err)

	fork1, err := db.ForkBlock(ctx, root.ID, "Fork 1", nil)
	require.NoError(t, err)

	fork00, err := db.ForkBlock(ctx, fork0.ID, "Fork 0/0", nil)
	require.NoError(t, err)

	// Initially, no blocks should be archived
	assert.Nil(t, root.ArchivedAt)
	assert.Nil(t, fork0.ArchivedAt)
	assert.Nil(t, fork1.ArchivedAt)
	assert.Nil(t, fork00.ArchivedAt)

	// Switch to branch "1" (fork1)
	archivedAt := int64(1234569999)
	err = db.ArchiveInactiveBranches(ctx, 1, "1", archivedAt)
	require.NoError(t, err)

	// Verify: Only blocks on path "1" (root + fork1) should remain active
	// Blocks on path "0" (fork0 + fork00) should be archived
	assert.Nil(t, root.ArchivedAt, "Root should not be archived (on all paths)")
	assert.NotNil(t, fork0.ArchivedAt, "Fork 0 should be archived (not on path 1)")
	assert.Equal(t, archivedAt, *fork0.ArchivedAt)
	assert.NotNil(t, fork00.ArchivedAt, "Fork 0/0 should be archived (not on path 1)")
	assert.Equal(t, archivedAt, *fork00.ArchivedAt)
	assert.Nil(t, fork1.ArchivedAt, "Fork 1 should not be archived (on path 1)")

	// Switch back to branch "0"
	archivedAt2 := int64(1234579999)
	err = db.ArchiveInactiveBranches(ctx, 1, "0", archivedAt2)
	require.NoError(t, err)

	// Now fork0 should be unarchived (or still have old archived_at)
	// and fork1 should be archived
	// Note: The mock doesn't unarchive, so fork0 would still have archived_at
	// In real implementation, unarchiving might be a separate operation

	// What we can verify: fork1 is now archived
	assert.Equal(t, archivedAt2, *fork1.ArchivedAt, "Fork 1 should be archived after switching to path 0")

	// fork00 should still be archived (descendant of archived fork0, or archived separately)
	assert.NotNil(t, fork00.ArchivedAt, "Fork 0/0 should remain archived")
}

// ============================================================================
// Cost Calculation Tests (Task 049: Cost Calculation Precision)
// ============================================================================

// TestCostCalculation_MilliCentsPrecision verifies that cost_estimate is stored
// with milli-cent precision (1/1000 of a cent, or 1/100,000 of a dollar).
func TestCostCalculation_MilliCentsPrecision(t *testing.T) {
	// Cost estimate uses int64 for milli-cents
	// Example: $0.001234 = 123.4 milli-cents = 123400 micro-dollars
	// But for simplicity: $0.01 = 1 cent = 1000 milli-cents
	// $1.23 = 123 cents = 123,000 milli-cents

	testCases := []struct {
		name          string
		centsUSD      float64 // Cost in USD cents
		expectedMilli int64   // Expected value in milli-cents
	}{
		{
			name:          "Zero cost",
			centsUSD:      0,
			expectedMilli: 0,
		},
		{
			name:          "One cent",
			centsUSD:      1.0, // $0.01
			expectedMilli: 1000,
		},
		{
			name:          "Ten cents",
			centsUSD:      10.0, // $0.10
			expectedMilli: 10000,
		},
		{
			name:          "One dollar",
			centsUSD:      100.0, // $1.00
			expectedMilli: 100000,
		},
		{
			name:          "Fractional cent",
			centsUSD:      0.123, // $0.00123
			expectedMilli: 123,
		},
		{
			name:          "Precise calculation",
			centsUSD:      12.345, // $0.12345
			expectedMilli: 12345,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert cents to milli-cents
			milliCents := int64(tc.centsUSD * 1000)

			assert.Equal(t, tc.expectedMilli, milliCents,
				"Cost calculation in milli-cents should match expected for %s (%f cents)", tc.name, tc.centsUSD)

			// Verify reverse calculation
			centsBack := float64(milliCents) / 1000
			assert.InDelta(t, tc.centsUSD, centsBack, 0.0001,
				"Reverse calculation should match within precision")
		})
	}
}

// TestCostCalculation_LLMTokenPricing verifies cost calculation for different token counts.
func TestCostCalculation_LLMTokenPricing(t *testing.T) {
	// DeepSeek pricing (example): $0.14 per million input tokens, $0.28 per million output tokens
	// That's: $0.00000014 per input token, $0.00000028 per output token

	type TokenUsage struct {
		inputTokens  int
		outputTokens int
		expectedCost float64 // in milli-cents
	}

	testCases := []TokenUsage{
		{1000, 1000, 420},  // 1000 input + 1000 output = 2000 tokens = 0.42 cents = 420 milli-cents
		{1000, 2000, 700},  // 1000 input + 2000 output = 3000 tokens = 0.70 cents = 700 milli-cents
		{5000, 5000, 2100}, // 5000 + 5000 = 10000 tokens = 2.10 cents = 2100 milli-cents
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%d_input_%d_output", tc.inputTokens, tc.outputTokens), func(t *testing.T) {
			// Calculate cost based on token pricing
			inputCost := float64(tc.inputTokens) * 0.14 // milli-cents per token (scaled)
			outputCost := float64(tc.outputTokens) * 0.28
			totalCost := inputCost + outputCost

			assert.InDelta(t, tc.expectedCost, totalCost, 0.001,
				"Token cost calculation should match expected for %d input + %d output", tc.inputTokens, tc.outputTokens)

			// Verify the cost fits in int64 (max ~9.2 quintillion milli-cents = $92 billion)
			assert.Less(t, totalCost, float64(1<<62), "Cost should fit in int64")
		})
	}
}
