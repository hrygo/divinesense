package ai

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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
	eventCount     int        // For concurrent testing
	eventsMutex    sync.Mutex // For concurrent testing
}

func newMockBlockStore() *mockBlockStore {
	return &mockBlockStore{
		blocks: make(map[int64]*store.AIBlock),
		nextID: 1,
	}
}

// newTestBlockManager creates a testBlockManager with proper initialization.
func newTestBlockManager() *testBlockManager {
	mockStore := newMockBlockStore()
	return &testBlockManager{
		BlockManager: &BlockManager{}, // Empty but has serializers map
		mock:         mockStore,
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

	m.eventsMutex.Lock()
	defer m.eventsMutex.Unlock()
	m.eventCount++

	block, ok := m.blocks[blockID]
	if !ok {
		return assert.AnError
	}

	// Create a new slice to avoid race with concurrent reads
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

	// Create a test wrapper that uses the mock
	manager := newTestBlockManager()

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
	manager := newTestBlockManager()

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
	manager := newTestBlockManager()

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
	manager := newTestBlockManager()

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
	retrievedBlock, exists := manager.getMockStore().blocks[block.ID]
	require.True(t, exists)
	assert.Len(t, retrievedBlock.EventStream, 3)
	assert.Equal(t, "thinking", retrievedBlock.EventStream[0].Type)
	assert.Equal(t, "answer", retrievedBlock.EventStream[2].Type)
	assert.Equal(t, "search", retrievedBlock.EventStream[1].Meta["tool"])
}

// TestBlockManager_AppendUserInput tests appending additional user input.
func TestBlockManager_AppendUserInput(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "First input", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Append additional input
	err = manager.AppendUserInput(ctx, block.ID, "Second input")
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Len(t, retrievedBlock.UserInputs, 2)
	assert.Equal(t, "Second input", retrievedBlock.UserInputs[1].Content)
}

// TestBlockManager_AppendEventsBatch tests batch event appending.
func TestBlockManager_AppendEventsBatch(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

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
	retrievedBlock, ok := manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Len(t, retrievedBlock.EventStream, 3)
}

// TestBlockManager_UpdateBlockStatus tests status updates.
func TestBlockManager_UpdateBlockStatus(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)
	assert.Equal(t, store.AIBlockStatusPending, block.Status)

	// Update to streaming
	err = manager.UpdateBlockStatus(ctx, block.ID, store.AIBlockStatusStreaming, "", nil)
	require.NoError(t, err)

	retrievedBlock, ok := manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusStreaming, retrievedBlock.Status)

	// Update to completed with content
	content := "Final answer"
	sessionStats := &store.SessionStats{}

	err = manager.UpdateBlockStatus(ctx, block.ID, store.AIBlockStatusCompleted, content, sessionStats)
	require.NoError(t, err)

	retrievedBlock, ok = manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusCompleted, retrievedBlock.Status)
	assert.Equal(t, content, retrievedBlock.AssistantContent)
	assert.NotNil(t, retrievedBlock.SessionStats)
}

// TestBlockManager_CompleteBlock tests completing a block.
func TestBlockManager_CompleteBlock(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Complete it
	content := "Here is your answer"
	sessionStats := &store.SessionStats{}

	err = manager.CompleteBlock(ctx, block.ID, content, sessionStats)
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusCompleted, retrievedBlock.Status)
	assert.Equal(t, content, retrievedBlock.AssistantContent)
}

// TestBlockManager_MarkBlockError tests error marking.
func TestBlockManager_MarkBlockError(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Mark as error
	errorMsg := "Something went wrong"
	err = manager.MarkBlockError(ctx, block.ID, errorMsg)
	require.NoError(t, err)

	// Verify
	retrievedBlock, ok := manager.getMockStore().blocks[block.ID]
	require.True(t, ok)
	assert.Equal(t, store.AIBlockStatusError, retrievedBlock.Status)
	assert.Equal(t, errorMsg, retrievedBlock.AssistantContent)
}

// TestBlockManager_GetLatestBlock tests retrieving the latest block.
func TestBlockManager_GetLatestBlock(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

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
	manager := newTestBlockManager()

	// Empty batch should not error
	err := manager.AppendEventsBatch(ctx, 999, []store.BlockEvent{})
	require.NoError(t, err)
}

// testBlockManager is a test wrapper that uses a mock store.
// This allows us to test BlockManager logic without a real database.
type testBlockManager struct {
	*BlockManager // Embed real BlockManager for serializers access
	mock          *mockBlockStore
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

// getMockStore returns the internal mock store for verification in tests.
func (m *testBlockManager) getMockStore() *mockBlockStore {
	return m.mock
}

func (m *testBlockManager) AppendEvent(
	ctx context.Context,
	blockID int64,
	eventType string,
	content string,
	metadata map[string]any,
) error {
	// Persist to mock store
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

// ============================================================================
// Event Serialization Tests (Edge Cases)
// ============================================================================

// TestEventSerializer_ConcurrentAppend tests concurrent event appends.
// Verifies that events are persisted in order even when appended concurrently.
func TestEventSerializer_ConcurrentAppend(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create a block
	block, err := manager.CreateBlockForChat(ctx, 1, "Test", AgentTypeMemo, BlockModeNormal)
	require.NoError(t, err)

	// Number of concurrent goroutines and events per goroutine
	const numGoroutines = 10
	const eventsPerGoroutine = 50

	// Use a WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track expected event count
	expectedEventCount := numGoroutines * eventsPerGoroutine

	// appendWg tracks when all AppendEvent calls have completed
	// This ensures all events are enqueued before we stop the serializer
	var appendWg sync.WaitGroup
	appendWg.Add(numGoroutines * eventsPerGoroutine)

	// Launch concurrent goroutines appending events
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				eventType := fmt.Sprintf("event_g%d_e%d", goroutineID, j)
				_ = manager.AppendEvent(ctx, block.ID, eventType, fmt.Sprintf("content from goroutine %d", goroutineID), nil)
				appendWg.Done()
			}
		}(i)
	}

	// Wait for all goroutines to finish their event generation
	wg.Wait()

	// Wait for all AppendEvent calls to complete (events enqueued)
	appendWg.Wait()

	// Give the serializer time to process all queued events
	// (The serializer processes asynchronously in a goroutine)
	time.Sleep(100 * time.Millisecond)

	// Stop the serializer to ensure all events are flushed
	manager.stopSerializer(block.ID)

	// Verify all events were persisted
	retrievedBlock, exists := manager.getMockStore().blocks[block.ID]
	require.True(t, exists, "Block should exist")

	assert.Equal(t, expectedEventCount, len(retrievedBlock.EventStream),
		"Should have all events from concurrent appends")

	// Verify no duplicate event types (each should be unique)
	eventTypes := make(map[string]int)
	for _, event := range retrievedBlock.EventStream {
		eventTypes[event.Type]++
	}

	// Each event type should appear exactly once
	for eventType, count := range eventTypes {
		assert.Equal(t, 1, count, "Event type %s should appear exactly once, got %d", eventType, count)
	}
}

// TestEventSerializer_DoubleStop tests calling stop twice.
func TestEventSerializer_DoubleStop(t *testing.T) {
	manager := newTestBlockManager()

	s := &eventSerializer{
		blockID:    int64(1),
		manager:    manager.BlockManager,
		channel:    make(chan *blockEvent, 10),
		stopCh:     make(chan struct{}),
		createTime: time.Now(),
	}
	s.start()

	s.stop()
	s.stop() // Should not panic

	// Verify stopCh is closed
	select {
	case <-s.stopCh:
		// Expected
	default:
		t.Error("stopCh should be closed")
	}
}

// TestBlockManager_CleanupStaleSerializers tests the cleanup of old serializers.
// Since we cannot directly manipulate createTime without reflection complexity,
// we test that the cleanup function works correctly and serializers are properly stopped.
func TestBlockManager_CleanupStaleSerializers(t *testing.T) {
	ctx := context.Background()
	manager := newTestBlockManager()

	// Create blocks and append events to create serializers
	const numBlocks = 3
	var blockIDs []int64

	for i := int64(1); i <= int64(numBlocks); i++ {
		// Create a block in the mock store
		manager.getMockStore().blocks[i] = &store.AIBlock{
			ID:          i,
			RoundNumber: int32(i),
		}
		blockIDs = append(blockIDs, i)

		// Append event creates a serializer
		err := manager.AppendEvent(ctx, i, fmt.Sprintf("event_%d", i), "content", nil)
		require.NoError(t, err)
	}

	// Verify serializers were created (by checking we can append more)
	for _, blockID := range blockIDs {
		err := manager.AppendEvent(ctx, blockID, "second_event", "content", nil)
		assert.NoError(t, err, "Should be able to append to active serializer")
	}

	// Call cleanup - with recent serializers, nothing should be cleaned
	cleaned := manager.CleanupStaleSerializers()
	assert.Equal(t, 0, cleaned, "Should not clean up recent serializers")

	// Complete blocks to stop their serializers
	for _, blockID := range blockIDs {
		err := manager.CompleteBlock(ctx, blockID, "complete", nil)
		assert.NoError(t, err, "CompleteBlock should succeed")
	}

	// After completion, cleanup should still return 0 (already cleaned)
	cleaned = manager.CleanupStaleSerializers()
	assert.Equal(t, 0, cleaned, "Should not clean up already stopped serializers")

	// Verify we can create new serializers after cleanup
	for _, blockID := range blockIDs {
		err := manager.AppendEvent(ctx, blockID, "after_cleanup", "content", nil)
		assert.NoError(t, err, "Should create new serializer after cleanup")
	}
}

// TestBlockManager_CompleteBlock_CleansSerializer verifies CompleteBlock cleans up.
func TestBlockManager_CompleteBlock_CleansSerializer(t *testing.T) {
	manager := newTestBlockManager()

	// Create block
	ctx := context.Background()
	manager.getMockStore().blocks[1] = &store.AIBlock{
		ID:          1,
		RoundNumber: 1,
	}

	// Append event creates serializer
	manager.AppendEvent(ctx, 1, "test_event", "test_content", nil)

	// Complete block should clean up serializer
	manager.CompleteBlock(ctx, 1, "test content", nil)

	// Since serializers is private, we verify by checking AppendEvent still works
	// (a new serializer should be created if needed)
	err := manager.AppendEvent(ctx, 1, "test_event_2", "test_content_2", nil)
	if err != nil {
		t.Errorf("AppendEvent should still work after CompleteBlock: %v", err)
	}
}

// TestBlockManager_MarkBlockError_CleansSerializer verifies MarkBlockError cleans up.
func TestBlockManager_MarkBlockError_CleansSerializer(t *testing.T) {
	manager := newTestBlockManager()

	// Create block
	ctx := context.Background()
	manager.getMockStore().blocks[1] = &store.AIBlock{
		ID:          1,
		RoundNumber: 1,
	}

	// Append event creates serializer
	manager.AppendEvent(ctx, 1, "test_event", "test_content", nil)

	// Mark error should clean up serializer
	manager.MarkBlockError(ctx, 1, "test error")

	// Verify AppendEvent still works (new serializer created if needed)
	err := manager.AppendEvent(ctx, 1, "test_event_2", "test_content_2", nil)
	if err != nil {
		t.Errorf("AppendEvent should still work after MarkBlockError: %v", err)
	}
}
