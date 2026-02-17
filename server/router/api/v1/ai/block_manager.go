package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
	"github.com/lithammer/shortuuid/v4"
)

const (
	// serializerTimeout is the maximum time a serializer can exist without completion.
	// After this timeout, the serializer is stopped to prevent resource leaks.
	// 30 minutes is chosen as a safe upper bound for normal chat completion.
	serializerTimeout = 30 * time.Minute

	// serializerStopTimeout is the maximum time to wait for serializer drain during stop.
	// After this timeout, the serializer is forcibly stopped to prevent indefinite blocking.
	serializerStopTimeout = 5 * time.Second
)

// BlockManager manages the lifecycle of conversation blocks.
//
// It handles creating blocks for new chat rounds, appending events during
// streaming, and updating block status upon completion.
type BlockManager struct {
	store *store.Store

	// Event serialization: ensures events are persisted in order
	// Key: blockID, Value: event serializer for that block
	serializers sync.Map // map[int64]*eventSerializer
}

// NewBlockManager creates a new BlockManager.
func NewBlockManager(store *store.Store) *BlockManager {
	return &BlockManager{store: store}
}

// eventSerializer serializes event persistence for a single block.
// Events are queued and persisted in order by a dedicated goroutine.
type eventSerializer struct {
	blockID    int64
	manager    *BlockManager
	channel    chan *blockEvent
	wg         sync.WaitGroup
	stopCh     chan struct{}
	once       sync.Once
	createTime time.Time // Track creation time for timeout cleanup
}

type blockEvent struct {
	eventType string
	content   string
	metadata  map[string]any
}

// start begins the event processing goroutine.
func (s *eventSerializer) start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ctx := context.Background()
		for {
			select {
			case <-s.stopCh:
				// Drain remaining events before stopping
				// Use select with default to avoid blocking when channel is empty
				for {
					select {
					case event := <-s.channel:
						s.persist(ctx, event)
					default:
						// Channel is empty, safe to exit
						return
					}
				}
			case event := <-s.channel:
				s.persist(ctx, event)
			}
		}
	}()
}

// persist writes a single event to the database.
func (s *eventSerializer) persist(ctx context.Context, event *blockEvent) {
	blockEvent := store.BlockEvent{
		Type:      event.eventType,
		Content:   event.content,
		Timestamp: time.Now().UnixMilli(),
		Meta:      event.metadata,
	}
	if err := s.manager.store.AppendEvent(ctx, s.blockID, blockEvent); err != nil {
		slog.Error("Failed to append event",
			"block_id", s.blockID,
			"event_type", event.eventType,
			"error", err,
		)
	}
}

// enqueue adds an event to the serialization queue.
//
// Returns false if the serializer has been stopped OR if the channel is full.
// The non-blocking select with default case means events may be dropped when
// the channel buffer (100 events) is saturated. This is intentional: it prevents
// slow event persistence from blocking the streaming response. In practice, with
// a 100-event buffer and typical persistence latency (<1ms per event), the channel
// should rarely be full. If events are being dropped, consider:
// 1. Increasing the buffer size in getOrCreateSerializer
// 2. Adding metrics to track dropped events
// 3. Using a blocking enqueue with timeout
func (s *eventSerializer) enqueue(eventType string, content string, metadata map[string]any) bool {
	select {
	case s.channel <- &blockEvent{eventType, content, metadata}:
		return true
	case <-s.stopCh:
		return false
	default:
		// Channel is full - event is dropped
		// TODO: Add metrics to track dropped events
		return false
	}
}

// stop gracefully shuts down the serializer after draining the queue.
// Uses a timeout to prevent indefinite blocking if the drain takes too long.
func (s *eventSerializer) stop() {
	s.once.Do(func() {
		close(s.stopCh)

		// Add timeout to prevent indefinite blocking during drain
		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Normal shutdown
		case <-time.After(serializerStopTimeout):
			slog.Warn("Event serializer stop timeout, forcing shutdown",
				"block_id", s.blockID,
				"timeout_seconds", serializerStopTimeout.Seconds(),
			)
		}
	})
}

// getOrCreateSerializer retrieves or creates an event serializer for the given block.
func (m *BlockManager) getOrCreateSerializer(blockID int64) *eventSerializer {
	if v, ok := m.serializers.Load(blockID); ok {
		// Defensive type assertion with safety check
		if s, ok := v.(*eventSerializer); ok {
			return s
		}
		// Unexpected type in map - log and delete to recover
		slog.Error("Unexpected type in serializers map, deleting and recreating",
			"block_id", blockID,
			"actual_type", fmt.Sprintf("%T", v),
		)
		m.serializers.Delete(blockID)
		// Continue to create new serializer
	}

	s := &eventSerializer{
		blockID:    blockID,
		manager:    m,
		channel:    make(chan *blockEvent, 100), // Buffered channel for throughput
		stopCh:     make(chan struct{}),
		createTime: time.Now(), // Track creation time for timeout cleanup
	}
	s.start()

	if actual, loaded := m.serializers.LoadOrStore(blockID, s); loaded {
		// Another goroutine created a serializer first, use it and stop ours
		s.stop()
		// Defensive type assertion for the loaded value
		if serializer, ok := actual.(*eventSerializer); ok {
			return serializer
		}
		// Should never happen, but log and return ours as fallback
		slog.Error("Loaded unexpected type from serializers map",
			"block_id", blockID,
			"actual_type", fmt.Sprintf("%T", actual),
		)
		return s
	}
	return s
}

// stopSerializer stops and removes the serializer for a block.
// This should be called when the block is completed.
func (m *BlockManager) stopSerializer(blockID int64) {
	if v, ok := m.serializers.LoadAndDelete(blockID); ok {
		// Defensive type assertion with safety check
		if s, ok := v.(*eventSerializer); ok {
			s.stop()
		} else {
			slog.Error("Unexpected type in stopSerializer",
				"block_id", blockID,
				"actual_type", fmt.Sprintf("%T", v),
			)
		}
	}
}

// CleanupStaleSerializers removes serializers that have been active longer than the timeout.
// This should be called periodically (e.g., via a ticker) to prevent resource leaks
// from blocks that never complete due to client disconnection or crashes.
func (m *BlockManager) CleanupStaleSerializers() int {
	cleaned := 0
	now := time.Now()
	m.serializers.Range(func(key, value any) bool {
		// Defensive type assertion with safety check
		s, ok := value.(*eventSerializer)
		if !ok {
			slog.Error("Unexpected type in CleanupStaleSerializers, removing",
				"key", key,
				"actual_type", fmt.Sprintf("%T", value),
			)
			m.serializers.Delete(key)
			return true
		}

		if now.Sub(s.createTime) > serializerTimeout {
			if m.serializers.CompareAndDelete(key, value) {
				s.stop()
				cleaned++
				slog.Warn("Cleaned up stale event serializer",
					"block_id", s.blockID,
					"age_minutes", now.Sub(s.createTime).Minutes(),
				)
			}
		}
		return true
	})
	return cleaned
}

// CreateBlockForChat creates a new block for a chat round.
//
// This should be called when starting a new chat round (user sends message).
func (m *BlockManager) CreateBlockForChat(
	ctx context.Context,
	conversationID int32,
	userMessage string,
	agentType AgentType,
	mode BlockMode,
) (*store.AIBlock, error) {
	now := time.Now().UnixMilli()

	// All modes use MESSAGE type (context_separator is created separately)
	blockType := store.AIBlockTypeMessage

	// Convert mode to store type
	storeMode := convertBlockModeToStore(mode)

	block, err := m.store.CreateAIBlockWithRound(ctx, &store.CreateAIBlock{
		UID:            shortuuid.New(),
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
	if err != nil {
		slog.Error("Failed to create block",
			"conversation_id", conversationID,
			"error", err,
		)
		return nil, err
	}

	slog.Info("Created block for chat",
		"block_id", block.ID,
		"conversation_id", conversationID,
		"round_number", block.RoundNumber,
	)

	return block, nil
}

// AppendEvent appends an event to the block's event stream.
//
// Events are queued and persisted in order by a dedicated goroutine per block.
// This ensures that events are written to the database in the same order
// they are received, preventing race conditions during concurrent appends.
//
// This should be called during streaming to record thinking, tool_use, tool_result, and answer events.
//
// Metadata key convention: Use snake_case keys (Go convention) for metadata.
// The TypeScript frontend will access these using the same snake_case keys.
// Example: {"tool_name": "search", "query": "test"} → frontend accesses as meta.tool_name
func (m *BlockManager) AppendEvent(
	ctx context.Context,
	blockID int64,
	eventType string,
	content string,
	metadata map[string]any,
) error {
	serializer := m.getOrCreateSerializer(blockID)
	if !serializer.enqueue(eventType, content, metadata) {
		return fmt.Errorf("event serializer stopped for block %d", blockID)
	}

	slog.Debug("Enqueued event for persistence",
		"block_id", blockID,
		"event_type", eventType,
	)

	return nil
}

// AppendUserInput appends an additional user input to an existing block.
//
// This is used when the user provides follow-up input during streaming.
func (m *BlockManager) AppendUserInput(
	ctx context.Context,
	blockID int64,
	userInput string,
) error {
	input := store.UserInput{
		Content:   userInput,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := m.store.AppendUserInput(ctx, blockID, input); err != nil {
		slog.Error("Failed to append user input",
			"block_id", blockID,
			"error", err,
		)
		return err
	}

	slog.Debug("Appended user input to block",
		"block_id", blockID,
	)

	return nil
}

// AppendEventsBatch appends multiple events to the block's event stream in a single query.
//
// This is more efficient than calling AppendEvent multiple times,
// especially for streaming responses with many events.
func (m *BlockManager) AppendEventsBatch(
	ctx context.Context,
	blockID int64,
	events []store.BlockEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	// Add timestamps to all events
	now := time.Now().UnixMilli()
	for i := range events {
		if events[i].Timestamp == 0 {
			events[i].Timestamp = now
		}
	}

	if err := m.store.AppendEventsBatch(ctx, blockID, events); err != nil {
		slog.Error("Failed to append events batch",
			"block_id", blockID,
			"event_count", len(events),
			"error", err,
		)
		return err
	}

	slog.Debug("Appended events batch to block",
		"block_id", blockID,
		"event_count", len(events),
	)

	return nil
}

// UpdateBlockStatus updates the status of a block.
//
// This should be called when streaming completes or fails.
func (m *BlockManager) UpdateBlockStatus(
	ctx context.Context,
	blockID int64,
	status store.AIBlockStatus,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	now := time.Now().UnixMilli()
	update := &store.UpdateAIBlock{
		ID:               blockID,
		Status:           &status,
		AssistantContent: &assistantContent,
		AssistantTs:      &now, // Set assistant timestamp for context building
		UpdatedTs:        &now,
	}

	if sessionStats != nil {
		update.SessionStats = sessionStats

		// P1-A006: Convert SessionStats to TokenUsage for ai_block token_usage column
		// This ensures token usage is persisted in the JSONB field for future analysis
		update.TokenUsage = &store.TokenUsage{
			PromptTokens:     int32(sessionStats.InputTokens),
			CompletionTokens: int32(sessionStats.OutputTokens),
			TotalTokens:      int32(sessionStats.TotalTokens),
			CacheReadTokens:  int32(sessionStats.CacheReadTokens),
			CacheWriteTokens: int32(sessionStats.CacheWriteTokens),
		}

		// ai-block-fields-extension: Set model_version
		if sessionStats.ModelUsed != "" {
			update.ModelVersion = &sessionStats.ModelUsed
		}
	}

	block, err := m.store.UpdateAIBlock(ctx, update)
	if err != nil {
		slog.Error("Failed to update block status",
			"block_id", blockID,
			"status", status,
			"error", err,
		)
		return err
	}

	slog.Info("Updated block status",
		"block_id", blockID,
		"status", status,
		"round_number", block.RoundNumber,
	)

	return nil
}

// CompleteBlock marks a block as completed with the final assistant content.
//
// Stops the event serializer for this block after updating status.
//
// Safety: This is safe even if UpdateBlockStatus fails because:
//  1. stopSerializer uses sync.Once, so multiple calls are idempotent
//  2. The serializer drain in stop() ensures queued events are persisted
//  3. If status update fails, the block remains in streaming state but events
//     continue to be persisted until a subsequent CompleteBlock/MarkBlockError call
func (m *BlockManager) CompleteBlock(
	ctx context.Context,
	blockID int64,
	assistantContent string,
	sessionStats *store.SessionStats,
) error {
	err := m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusCompleted, assistantContent, sessionStats)
	// Stop the event serializer after completing the block
	// Even if UpdateBlockStatus failed, we stop the serializer to prevent resource leaks
	m.stopSerializer(blockID)
	return err
}

// MarkBlockError marks a block as failed with error status.
//
// Stops the event serializer for this block after updating status.
//
// Safety: This is safe even if UpdateBlockStatus fails because:
//  1. stopSerializer uses sync.Once, so multiple calls are idempotent
//  2. The serializer drain in stop() ensures queued events are persisted
//  3. If status update fails, the block remains in streaming state but events
//     continue to be persisted until a subsequent CompleteBlock/MarkBlockError call
func (m *BlockManager) MarkBlockError(
	ctx context.Context,
	blockID int64,
	errorMessage string,
) error {
	err := m.UpdateBlockStatus(ctx, blockID, store.AIBlockStatusError, errorMessage, nil)
	// Stop the event serializer after marking error
	// Even if UpdateBlockStatus failed, we stop the serializer to prevent resource leaks
	m.stopSerializer(blockID)
	return err
}

// GetLatestBlock retrieves the most recent block for a conversation.
func (m *BlockManager) GetLatestBlock(
	ctx context.Context,
	conversationID int32,
) (*store.AIBlock, error) {
	return m.store.GetLatestAIBlock(ctx, conversationID)
}

// ============================================================================
// Helper Functions
// ============================================================================

// BlockMode represents the AI mode for a block.
type BlockMode string

const (
	BlockModeNormal    BlockMode = "normal"
	BlockModeGeek      BlockMode = "geek"
	BlockModeEvolution BlockMode = "evolution"
)

func convertBlockModeToStore(mode BlockMode) store.AIBlockMode {
	switch mode {
	case BlockModeNormal:
		return store.AIBlockModeNormal
	case BlockModeGeek:
		return store.AIBlockModeGeek
	case BlockModeEvolution:
		return store.AIBlockModeEvolution
	default:
		return store.AIBlockModeNormal
	}
}
