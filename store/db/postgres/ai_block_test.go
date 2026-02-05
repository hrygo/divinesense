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
