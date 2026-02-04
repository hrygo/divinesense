package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrygo/divinesense/store"
)

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

	block, err := db.CreateAIBlock(ctx, create)
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

	created, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Retrieve the block
	block, err := db.GetAIBlock(ctx, created.ID)
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
		_, err := db.CreateAIBlock(ctx, create)
		require.NoError(t, err)
	}

	// List all blocks for conversation
	blocks, err := db.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &convID,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 3)

	// Filter by status
	pending := store.AIBlockStatusPending
	blocks, err = db.ListAIBlocks(ctx, &store.FindAIBlock{
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

	created, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Update the block
	content := "Here is the answer"
	status := store.AIBlockStatusCompleted
	updated, err := db.UpdateAIBlock(ctx, &store.UpdateAIBlock{
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

	created, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Append another input
	err = db.AppendUserInput(ctx, created.ID, store.UserInput{
		Content:   "Second input",
		Timestamp: 1234567891,
	})
	require.NoError(t, err)

	// Verify
	block, err := db.GetAIBlock(ctx, created.ID)
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

	created, err := db.CreateAIBlock(ctx, create)
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
	block, err := db.GetAIBlock(ctx, created.ID)
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

	created, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Update status
	err = db.UpdateAIBlockStatus(ctx, created.ID, store.AIBlockStatusCompleted)
	require.NoError(t, err)

	// Verify
	block, err := db.GetAIBlock(ctx, created.ID)
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
	block, err := db.GetLatestAIBlock(ctx, convID)
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
		_, err := db.CreateAIBlock(ctx, create)
		require.NoError(t, err)
	}

	// Get latest (should have round_number = 2)
	block, err = db.GetLatestAIBlock(ctx, convID)
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
		_, err := db.CreateAIBlock(ctx, create)
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
	_, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Get pending blocks
	blocks, err := db.GetPendingAIBlocks(ctx)
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

	created, err := db.CreateAIBlock(ctx, create)
	require.NoError(t, err)

	// Delete the block
	err = db.DeleteAIBlock(ctx, created.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = db.GetAIBlock(ctx, created.ID)
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
		block, err := db.CreateAIBlockWithRound(ctx, create)
		require.NoError(t, err)
		assert.Equal(t, int32(i), block.RoundNumber)
	}

	// List and verify order
	blocks, err := db.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &convID,
	})
	require.NoError(t, err)
	assert.Len(t, blocks, 3)
	assert.Equal(t, int32(0), blocks[0].RoundNumber)
	assert.Equal(t, int32(1), blocks[1].RoundNumber)
	assert.Equal(t, int32(2), blocks[2].RoundNumber)
}

// setupTestDB creates a test database connection
// In a real scenario, this would create a temporary database
// For now, this is a placeholder that will need integration test setup
func setupTestDB(t *testing.T) *DB {
	// This is a placeholder - integration tests would set up a real test database
	// For now, we'll skip tests that require a real DB connection
	t.Skip("Integration test - requires test database setup")
	return nil
}
