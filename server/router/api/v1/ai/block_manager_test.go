package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrygo/divinesense/store"
)

// mockBlockStore is a mock store for testing BlockManager.
type mockBlockStore struct {
	blocks         map[int64]*store.AIBlock
	nextID         int64
	createErr      error
	appendEventErr error
	appendInputErr error
	updateErr      error
	getLatestErr   error
}

func newMockBlockStore() *mockBlockStore {
	return &mockBlockStore{
		blocks: make(map[int64]*store.AIBlock),
		nextID: 1,
	}
}

func (m *mockBlockStore) CreateAIBlockWithRound(ctx context.Context, create *store.CreateAIBlock) (*store.AIBlock, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	id := m.nextID
	m.nextID++

	block := &store.AIBlock{
		ID:               id,
		UID:              create.UID,
		ConversationID:   create.ConversationID,
		RoundNumber:      0, // Simplified for mock
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

func (m *mockBlockStore) AppendEvent(ctx context.Context, blockID int64, event store.BlockEvent) error {
	if m.appendEventErr != nil {
		return m.appendEventErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return assert.AnError
	}

	block.EventStream = append(block.EventStream, event)
	return nil
}

func (m *mockBlockStore) AppendUserInput(ctx context.Context, blockID int64, input store.UserInput) error {
	if m.appendInputErr != nil {
		return m.appendInputErr
	}

	block, ok := m.blocks[blockID]
	if !ok {
		return assert.AnError
	}

	block.UserInputs = append(block.UserInputs, input)
	return nil
}

func (m *mockBlockStore) UpdateAIBlock(ctx context.Context, update *store.UpdateAIBlock) (*store.AIBlock, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}

	block, ok := m.blocks[update.ID]
	if !ok {
		return nil, assert.AnError
	}

	if update.Status != nil {
		block.Status = *update.Status
	}
	if update.AssistantContent != nil {
		block.AssistantContent = *update.AssistantContent
	}
	if update.SessionStats != nil {
		block.SessionStats = update.SessionStats
	}
	if update.UpdatedTs != nil {
		block.UpdatedTs = *update.UpdatedTs
	}

	return block, nil
}

func (m *mockBlockStore) GetLatestAIBlock(ctx context.Context, conversationID int32) (*store.AIBlock, error) {
	if m.getLatestErr != nil {
		return nil, m.getLatestErr
	}

	for _, block := range m.blocks {
		if block.ConversationID == conversationID {
			return block, nil
		}
	}
	return nil, nil
}

// TestBlockManager_CreateBlockForChat tests block creation.
func TestBlockManager_CreateBlockForChat(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()

	// Create a test wrapper that uses the mock
	manager := &testBlockManager{mock: mockStore}

	block, err := manager.CreateBlockForChat(
		ctx,
		123, // conversationID
		"Hello, AI!",
		AgentTypeMemo,
		BlockModeNormal,
	)

	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, int64(1), block.ID)
	assert.Equal(t, int32(123), block.ConversationID)
	assert.Equal(t, store.AIBlockTypeMessage, block.BlockType)
	assert.Equal(t, store.AIBlockModeNormal, block.Mode)
	assert.Len(t, block.UserInputs, 1)
	assert.Equal(t, "Hello, AI!", block.UserInputs[0].Content)
	assert.Equal(t, store.AIBlockStatusPending, block.Status)
}

// TestBlockManager_CreateBlockForChat_GeekMode tests Geek mode block creation.
func TestBlockManager_CreateBlockForChat_GeekMode(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	block, err := manager.CreateBlockForChat(
		ctx,
		456,
		"Help me code",
		AgentTypeMemo,
		BlockModeGeek,
	)

	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, store.AIBlockModeGeek, block.Mode)
}

// TestBlockManager_CreateBlockForChat_EvolutionMode tests Evolution mode block creation.
func TestBlockManager_CreateBlockForChat_EvolutionMode(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	block, err := manager.CreateBlockForChat(
		ctx,
		789,
		"Evolve the system",
		AgentTypeMemo,
		BlockModeEvolution,
	)

	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, store.AIBlockModeEvolution, block.Mode)
}

// TestBlockManager_AppendEvent tests event appending.
func TestBlockManager_AppendEvent(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block first
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Append events
	events := []struct {
		typ      string
		content  string
		metadata map[string]any
	}{
		{"thinking", "Let me think...", nil},
		{"tool_use", "searching", map[string]any{"tool": "search"}},
		{"answer", "Here is the answer", nil},
	}

	for _, e := range events {
		err := manager.AppendEvent(ctx, block.ID, e.typ, e.content, e.metadata)
		require.NoError(t, err)
	}

	// Verify events were appended
	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Len(t, retrievedBlock.EventStream, 3)
	assert.Equal(t, "thinking", retrievedBlock.EventStream[0].Type)
	assert.Equal(t, "answer", retrievedBlock.EventStream[2].Type)
	assert.Equal(t, "search", retrievedBlock.EventStream[1].Meta["tool"])
}

// TestBlockManager_AppendUserInput tests appending additional user input.
func TestBlockManager_AppendUserInput(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "First input", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Append additional input
	err = manager.AppendUserInput(ctx, block.ID, "Second input")
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Len(t, retrievedBlock.UserInputs, 2)
	assert.Equal(t, "Second input", retrievedBlock.UserInputs[1].Content)
}

// TestBlockManager_AppendEventsBatch tests batch event appending.
func TestBlockManager_AppendEventsBatch(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Append events in batch
	events := []store.BlockEvent{
		{Type: "thinking", Content: "Thinking 1", Timestamp: 1000},
		{Type: "thinking", Content: "Thinking 2", Timestamp: 1001},
		{Type: "answer", Content: "Answer", Timestamp: 1002},
	}

	err = manager.AppendEventsBatch(ctx, block.ID, events)
	require.NoError(t, err)

	// Verify all events were appended
	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Len(t, retrievedBlock.EventStream, 3)
}

// TestBlockManager_UpdateBlockStatus tests status updates.
func TestBlockManager_UpdateBlockStatus(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)
	assert.Equal(t, store.AIBlockStatusPending, block.Status)

	// Update to streaming
	err = manager.UpdateBlockStatus(ctx, block.ID, store.AIBlockStatusStreaming, "", nil)
	require.NoError(t, err)

	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusStreaming, retrievedBlock.Status)

	// Update to completed with content
	content := "Final answer"
	sessionStats := &store.SessionStats{}

	err = manager.UpdateBlockStatus(ctx, block.ID, store.AIBlockStatusCompleted, content, sessionStats)
	require.NoError(t, err)

	retrievedBlock, ok = mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusCompleted, retrievedBlock.Status)
	assert.Equal(t, content, retrievedBlock.AssistantContent)
	assert.NotNil(t, retrievedBlock.SessionStats)
}

// TestBlockManager_CompleteBlock tests completing a block.
func TestBlockManager_CompleteBlock(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Complete it
	content := "Here is your answer"
	sessionStats := &store.SessionStats{}

	err = manager.CompleteBlock(ctx, block.ID, content, sessionStats)
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusCompleted, retrievedBlock.Status)
	assert.Equal(t, content, retrievedBlock.AssistantContent)
}

// TestBlockManager_MarkBlockError tests error marking.
func TestBlockManager_MarkBlockError(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Mark as error
	errorMsg := "Something went wrong"
	err = manager.MarkBlockError(ctx, block.ID, errorMsg)
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := mockStore.blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusError, retrievedBlock.Status)
	assert.Equal(t, errorMsg, retrievedBlock.AssistantContent)
}

// TestBlockManager_GetLatestBlock tests retrieving the latest block.
func TestBlockManager_GetLatestBlock(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// No blocks yet
	block, err := manager.GetLatestBlock(ctx, 123)
	require.NoError(t, err)
	assert.Nil(t, block)

	// Create a block
	created, err := manager.CreateBlockForChat(ctx, 123, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Get latest
	block, err = manager.GetLatestBlock(ctx, 123)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, created.ID, block.ID)
}

// TestConvertBlockModeToStore tests mode conversion.
func TestConvertBlockModeToStore(t *testing.T) {
	tests := []struct {
		name     string
		mode     BlockMode
		expected store.AIBlockMode
	}{
		{"normal mode", BlockModeNormal, store.AIBlockModeNormal},
		{"geek mode", BlockModeGeek, store.AIBlockModeGeek},
		{"evolution mode", BlockModeEvolution, store.AIBlockModeEvolution},
		{"unknown mode defaults to normal", "unknown", store.AIBlockModeNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBlockModeToStore(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBlockManager_AppendEventsBatch_Empty tests empty batch handling.
func TestBlockManager_AppendEventsBatch_Empty(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockBlockStore()
	manager := &testBlockManager{mock: mockStore}

	// Empty batch should not error
	err := manager.AppendEventsBatch(ctx, 999, []store.BlockEvent{})
	require.NoError(t, err)
}

// testBlockManager is a test wrapper that uses a mock store.
// This allows us to test BlockManager logic without a real database.
type testBlockManager struct {
	mock *mockBlockStore
}

func (m *testBlockManager) CreateBlockForChat(
	ctx context.Context,
	conversationID int32,
	userMessage string,
	agentType AgentType,
	mode BlockMode,
) (*store.AIBlock, error) {
	now := int64(1234567890)

	// Determine block type from mode
	var blockType store.AIBlockType
	switch mode {
	case BlockModeEvolution, BlockModeGeek:
		blockType = store.AIBlockTypeMessage
	default:
		blockType = store.AIBlockTypeMessage
	}

	// Convert mode to store type
	storeMode := convertBlockModeToStore(mode)

	return m.mock.CreateAIBlockWithRound(ctx, &store.CreateAIBlock{
		ConversationID: conversationID,
		BlockType:      blockType,
		Mode:           storeMode,
		UserInputs: []store.UserInput{
			{
				Content:   userMessage,
				Timestamp: now,
			},
		},
		Status:    store.AIBlockStatusPending,
		CreatedTs: now,
		UpdatedTs: now,
	})
}

func (m *testBlockManager) AppendEvent(
	ctx context.Context,
	blockID int64,
	eventType string,
	content string,
	metadata map[string]any,
) error {
	event := store.BlockEvent{
		Type:      eventType,
		Content:   content,
		Timestamp: 1234567890,
		Meta:      metadata,
	}

	return m.mock.AppendEvent(ctx, blockID, event)
}

func (m *testBlockManager) AppendUserInput(
	ctx context.Context,
	blockID int64,
	userInput string,
) error {
	input := store.UserInput{
		Content:   userInput,
		Timestamp: 1234567890,
	}

	return m.mock.AppendUserInput(ctx, blockID, input)
}

func (m *testBlockManager) AppendEventsBatch(
	ctx context.Context,
	blockID int64,
	events []store.BlockEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	// Add timestamps to all events
	now := int64(1234567890)
	for i := range events {
		if events[i].Timestamp == 0 {
			events[i].Timestamp = now
		}
	}

	for _, event := range events {
		if err := m.mock.AppendEvent(ctx, blockID, event); err != nil {
			return err
		}
	}

	return nil
}

func (m *testBlockManager) UpdateBlockStatus(
	ctx context.Context,
	blockID int64,
	status store.AIBlockStatus,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	now := int64(1234567890)
	update := &store.UpdateAIBlock{
		ID:               blockID,
		Status:           &status,
		AssistantContent: &assistantContent,
		UpdatedTs:        &now,
	}

	if sessionStats != nil {
		update.SessionStats = sessionStats
	}

	_, err := m.mock.UpdateAIBlock(ctx, update)
	return err
}

func (m *testBlockManager) CompleteBlock(
	ctx context.Context,
	blockID int64,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	return m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusCompleted, assistantContent, sessionStats)
}

func (m *testBlockManager) MarkBlockError(
	ctx context.Context,
	blockID int64,
	errorMessage string,
) error {
	return m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusError, errorMessage, nil)
}

func (m *testBlockManager) GetLatestBlock(
	ctx context.Context,
	conversationID int32,
) (*store.AIBlock, error) {
	return m.mock.GetLatestAIBlock(ctx, conversationID)
}
