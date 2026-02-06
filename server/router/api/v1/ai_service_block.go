package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
)

// emptyMetadata is the default empty JSON object for metadata.
const emptyMetadata = "{}"

// ListBlocks retrieves blocks for a conversation.
func (s *AIService) ListBlocks(ctx context.Context, req *v1pb.ListBlocksRequest) (*v1pb.ListBlocksResponse, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Verify conversation ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.ConversationId,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	// Build filter
	find := &store.FindAIBlock{
		ConversationID: &req.ConversationId,
	}

	// Optional filters
	if req.Status != v1pb.BlockStatus_BLOCK_STATUS_UNSPECIFIED {
		status := convertBlockStatusFromProto(req.Status)
		find.Status = &status
	}
	if req.Mode != v1pb.BlockMode_BLOCK_MODE_UNSPECIFIED {
		mode := convertBlockModeFromProto(req.Mode)
		find.Mode = &mode
	}
	if req.CcSessionId != "" {
		find.CCSessionID = &req.CcSessionId
	}

	blocks, err := s.Store.ListAIBlocks(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list blocks: %v", err)
	}

	response := &v1pb.ListBlocksResponse{
		Blocks: make([]*v1pb.Block, 0, len(blocks)),
	}
	for _, b := range blocks {
		response.Blocks = append(response.Blocks, convertBlockFromStore(b))
	}

	return response, nil
}

// GetBlock retrieves a specific block.
func (s *AIService) GetBlock(ctx context.Context, req *v1pb.GetBlockRequest) (*v1pb.Block, error) {
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	// Verify conversation ownership
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	return convertBlockFromStore(block), nil
}

// CreateBlock creates a new conversation block.
func (s *AIService) CreateBlock(ctx context.Context, req *v1pb.CreateBlockRequest) (*v1pb.Block, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Verify conversation ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.ConversationId,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	now := time.Now().Unix()

	// Parse metadata
	var metadata map[string]any
	if req.Metadata != "" {
		if err := json.Unmarshal([]byte(req.Metadata), &metadata); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid metadata JSON: %v", err)
		}
	} else {
		metadata = make(map[string]any)
	}

	// Convert user inputs
	userInputs := make([]store.UserInput, len(req.UserInputs))
	for i, ui := range req.UserInputs {
		userInputs[i] = store.UserInput{
			Content:   ui.Content,
			Timestamp: ui.Timestamp,
			Metadata:  parseMetadata(ui.Metadata),
		}
	}

	// Determine block type and mode
	blockType := convertBlockTypeFromProto(req.BlockType)
	if blockType == "" {
		blockType = store.AIBlockTypeMessage
	}
	mode := convertBlockModeFromProto(req.Mode)
	if mode == "" {
		mode = store.AIBlockModeNormal
	}

	block, err := s.Store.CreateAIBlockWithRound(ctx, &store.CreateAIBlock{
		UID:            shortuuid.New(),
		ConversationID: req.ConversationId,
		BlockType:      blockType,
		Mode:           mode,
		UserInputs:     userInputs,
		Metadata:       metadata,
		CCSessionID:    req.CcSessionId,
		Status:         store.AIBlockStatusPending,
		CreatedTs:      now,
		UpdatedTs:      now,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create block: %v", err)
	}

	return convertBlockFromStore(block), nil
}

// UpdateBlock updates a block.
func (s *AIService) UpdateBlock(ctx context.Context, req *v1pb.UpdateBlockRequest) (*v1pb.Block, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get block and verify ownership
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	// Build update request
	update := &store.UpdateAIBlock{
		ID: req.Id,
	}

	if req.AssistantContent != nil {
		update.AssistantContent = req.AssistantContent
	}
	if len(req.EventStream) > 0 {
		eventStream := make([]store.BlockEvent, len(req.EventStream))
		for i, e := range req.EventStream {
			eventStream[i] = store.BlockEvent{
				Type:      e.Type,
				Content:   e.Content,
				Timestamp: e.Timestamp,
				Meta:      parseMetadata(e.Meta),
			}
		}
		update.EventStream = &eventStream
	}
	if req.SessionStats != nil {
		update.SessionStats = convertSessionStatsToStore(req.SessionStats)
	}
	if req.CcSessionId != nil {
		update.CCSessionID = req.CcSessionId
	}
	if req.Status != nil && *req.Status != v1pb.BlockStatus_BLOCK_STATUS_UNSPECIFIED {
		status := convertBlockStatusFromProto(*req.Status)
		update.Status = &status
	}

	// Parse metadata for merging
	var metadata map[string]any
	if req.Metadata != "" {
		if err := json.Unmarshal([]byte(req.Metadata), &metadata); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid metadata JSON: %v", err)
		}
		update.Metadata = metadata
	}

	updated, err := s.Store.UpdateAIBlock(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update block: %v", err)
	}

	return convertBlockFromStore(updated), nil
}

// DeleteBlock deletes a block.
func (s *AIService) DeleteBlock(ctx context.Context, req *v1pb.DeleteBlockRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get block and verify ownership
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	if err := s.Store.DeleteAIBlock(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete block: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// AppendUserInput appends a user input to an existing block.
func (s *AIService) AppendUserInput(ctx context.Context, req *v1pb.AppendUserInputRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get block and verify ownership
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	input := store.UserInput{
		Content:   req.Input.Content,
		Timestamp: req.Input.Timestamp,
		Metadata:  parseMetadata(req.Input.Metadata),
	}

	if err := s.Store.AppendUserInput(ctx, req.Id, input); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to append user input: %v", err)
	}

	slog.Debug("Appended user input to block",
		"block_id", req.Id,
		"content_length", len(req.Input.Content),
	)

	return &emptypb.Empty{}, nil
}

// AppendEvent appends an event to the block's event stream.
func (s *AIService) AppendEvent(ctx context.Context, req *v1pb.AppendEventRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get block and verify ownership
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	event := store.BlockEvent{
		Type:      req.Event.Type,
		Content:   req.Event.Content,
		Timestamp: req.Event.Timestamp,
		Meta:      parseMetadata(req.Event.Meta),
	}

	if err := s.Store.AppendEvent(ctx, req.Id, event); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to append event: %v", err)
	}

	slog.Debug("Appended event to block",
		"block_id", req.Id,
		"event_type", req.Event.Type,
	)

	return &emptypb.Empty{}, nil
}

// ========== Converter Functions ==========

// convertBlockFromStore converts a store.AIBlock to protobuf Block.
func convertBlockFromStore(b *store.AIBlock) *v1pb.Block {
	pbBlock := &v1pb.Block{
		Id:                 b.ID,
		Uid:                b.UID,
		ConversationId:     b.ConversationID,
		RoundNumber:        b.RoundNumber,
		BlockType:          convertBlockTypeToProto(b.BlockType),
		Mode:               convertBlockModeToProto(b.Mode),
		AssistantContent:   b.AssistantContent,
		AssistantTimestamp: b.AssistantTimestamp,
		CcSessionId:        b.CCSessionID,
		Status:             convertBlockStatusToProto(b.Status),
		CreatedTs:          b.CreatedTs,
		UpdatedTs:          b.UpdatedTs,
	}

	// Convert user inputs
	pbBlock.UserInputs = make([]*v1pb.UserInput, len(b.UserInputs))
	for i, ui := range b.UserInputs {
		pbBlock.UserInputs[i] = &v1pb.UserInput{
			Content:   ui.Content,
			Timestamp: ui.Timestamp,
			Metadata:  formatMetadata(ui.Metadata),
		}
	}

	// Convert event stream
	pbBlock.EventStream = make([]*v1pb.BlockEvent, len(b.EventStream))
	for i, e := range b.EventStream {
		pbBlock.EventStream[i] = &v1pb.BlockEvent{
			Type:      e.Type,
			Content:   e.Content,
			Timestamp: e.Timestamp,
			Meta:      formatMetadata(e.Meta),
		}
	}

	// Convert session stats
	if b.SessionStats != nil {
		pbBlock.SessionStats = convertSessionStatsFromStore(b.SessionStats)
	}

	// Convert tree branching fields
	if b.ParentBlockID != nil {
		pbBlock.ParentBlockId = *b.ParentBlockID
	}
	pbBlock.BranchPath = b.BranchPath

	// Convert metadata to JSON string
	pbBlock.Metadata = formatMetadata(b.Metadata)

	return pbBlock
}

// parseMetadata parses a JSON string into map[string]any.
func parseMetadata(metadataStr string) map[string]any {
	if metadataStr == "" {
		return make(map[string]any)
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		return make(map[string]any)
	}
	return metadata
}

// formatMetadata formats map[string]any into a JSON string.
func formatMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return emptyMetadata
	}
	if jsonBytes, err := json.Marshal(metadata); err == nil {
		return string(jsonBytes)
	}
	return emptyMetadata
}

// convertBlockTypeToProto converts store.AIBlockType to protobuf.
func convertBlockTypeToProto(bt store.AIBlockType) v1pb.BlockType {
	switch bt {
	case store.AIBlockTypeMessage:
		return v1pb.BlockType_BLOCK_TYPE_MESSAGE
	case store.AIBlockTypeContextSeparator:
		return v1pb.BlockType_BLOCK_TYPE_CONTEXT_SEPARATOR
	default:
		return v1pb.BlockType_BLOCK_TYPE_UNSPECIFIED
	}
}

// convertBlockTypeFromProto converts protobuf BlockType to store.
func convertBlockTypeFromProto(bt v1pb.BlockType) store.AIBlockType {
	switch bt {
	case v1pb.BlockType_BLOCK_TYPE_MESSAGE:
		return store.AIBlockTypeMessage
	case v1pb.BlockType_BLOCK_TYPE_CONTEXT_SEPARATOR:
		return store.AIBlockTypeContextSeparator
	default:
		return store.AIBlockTypeMessage // Default
	}
}

// convertBlockModeToProto converts store.AIBlockMode to protobuf.
func convertBlockModeToProto(bm store.AIBlockMode) v1pb.BlockMode {
	switch bm {
	case store.AIBlockModeNormal:
		return v1pb.BlockMode_BLOCK_MODE_NORMAL
	case store.AIBlockModeGeek:
		return v1pb.BlockMode_BLOCK_MODE_GEEK
	case store.AIBlockModeEvolution:
		return v1pb.BlockMode_BLOCK_MODE_EVOLUTION
	default:
		return v1pb.BlockMode_BLOCK_MODE_UNSPECIFIED
	}
}

// convertBlockModeFromProto converts protobuf BlockMode to store.
func convertBlockModeFromProto(bm v1pb.BlockMode) store.AIBlockMode {
	switch bm {
	case v1pb.BlockMode_BLOCK_MODE_NORMAL:
		return store.AIBlockModeNormal
	case v1pb.BlockMode_BLOCK_MODE_GEEK:
		return store.AIBlockModeGeek
	case v1pb.BlockMode_BLOCK_MODE_EVOLUTION:
		return store.AIBlockModeEvolution
	default:
		return store.AIBlockModeNormal // Default
	}
}

// convertBlockStatusToProto converts store.AIBlockStatus to protobuf.
func convertBlockStatusToProto(bs store.AIBlockStatus) v1pb.BlockStatus {
	switch bs {
	case store.AIBlockStatusPending:
		return v1pb.BlockStatus_BLOCK_STATUS_PENDING
	case store.AIBlockStatusStreaming:
		return v1pb.BlockStatus_BLOCK_STATUS_STREAMING
	case store.AIBlockStatusCompleted:
		return v1pb.BlockStatus_BLOCK_STATUS_COMPLETED
	case store.AIBlockStatusError:
		return v1pb.BlockStatus_BLOCK_STATUS_ERROR
	default:
		return v1pb.BlockStatus_BLOCK_STATUS_UNSPECIFIED
	}
}

// convertBlockStatusFromProto converts protobuf BlockStatus to store.
func convertBlockStatusFromProto(bs v1pb.BlockStatus) store.AIBlockStatus {
	switch bs {
	case v1pb.BlockStatus_BLOCK_STATUS_PENDING:
		return store.AIBlockStatusPending
	case v1pb.BlockStatus_BLOCK_STATUS_STREAMING:
		return store.AIBlockStatusStreaming
	case v1pb.BlockStatus_BLOCK_STATUS_COMPLETED:
		return store.AIBlockStatusCompleted
	case v1pb.BlockStatus_BLOCK_STATUS_ERROR:
		return store.AIBlockStatusError
	default:
		return store.AIBlockStatusPending // Default
	}
}

// convertSessionStatsFromStore converts store.SessionStats to protobuf.
func convertSessionStatsFromStore(stats *store.SessionStats) *v1pb.SessionStats {
	pbStats := &v1pb.SessionStats{
		SessionId:            stats.SessionID,
		AgentType:            stats.AgentType,
		TotalDurationMs:      stats.TotalDurationMs,
		ThinkingDurationMs:   stats.ThinkingDurationMs,
		ToolDurationMs:       stats.ToolDurationMs,
		GenerationDurationMs: stats.GenerationDurationMs,
		InputTokens:          int32(stats.InputTokens),
		OutputTokens:         int32(stats.OutputTokens),
		CacheWriteTokens:     int32(stats.CacheWriteTokens),
		CacheReadTokens:      int32(stats.CacheReadTokens),
		TotalTokens:          int32(stats.TotalTokens),
		TotalCostUsd:         stats.TotalCostUsd,
		ToolCallCount:        int32(stats.ToolCallCount),
		ToolsUsed:            stats.ToolsUsed,
		FilesModified:        int32(stats.FilesModified),
		FilePaths:            stats.FilePaths,
		ModelUsed:            stats.ModelUsed,
		IsError:              stats.IsError,
		ErrorMessage:         stats.ErrorMessage,
	}

	return pbStats
}

// convertSessionStatsToStore converts protobuf SessionStats to store.
func convertSessionStatsToStore(stats *v1pb.SessionStats) *store.SessionStats {
	return &store.SessionStats{
		SessionID:            stats.SessionId,
		AgentType:            stats.AgentType,
		TotalDurationMs:      stats.TotalDurationMs,
		ThinkingDurationMs:   stats.ThinkingDurationMs,
		ToolDurationMs:       stats.ToolDurationMs,
		GenerationDurationMs: stats.GenerationDurationMs,
		InputTokens:          int(stats.InputTokens),
		OutputTokens:         int(stats.OutputTokens),
		CacheWriteTokens:     int(stats.CacheWriteTokens),
		CacheReadTokens:      int(stats.CacheReadTokens),
		TotalTokens:          int(stats.TotalTokens),
		TotalCostUsd:         stats.TotalCostUsd,
		ToolCallCount:        int(stats.ToolCallCount),
		ToolsUsed:            stats.ToolsUsed,
		FilesModified:        int(stats.FilesModified),
		FilePaths:            stats.FilePaths,
		ModelUsed:            stats.ModelUsed,
		IsError:              stats.IsError,
		ErrorMessage:         stats.ErrorMessage,
	}
}

// ========== Tree Branching Handlers (Issue #79) ==========

// ForkBlock creates a new block as a branch from an existing block.
func (s *AIService) ForkBlock(ctx context.Context, req *v1pb.ForkBlockRequest) (*v1pb.Block, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get parent block and verify ownership
	parentBlock, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "parent block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &parentBlock.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	// Determine reason for fork (default to "user_fork")
	reason := "user_fork"
	if req.Reason != nil && *req.Reason != "" {
		reason = *req.Reason
	}

	// Convert replace user inputs if provided
	var replaceUserInputs []store.UserInput
	if len(req.ReplaceUserInputs) > 0 {
		replaceUserInputs = make([]store.UserInput, len(req.ReplaceUserInputs))
		for i, ui := range req.ReplaceUserInputs {
			replaceUserInputs[i] = store.UserInput{
				Content:   ui.Content,
				Timestamp: ui.Timestamp,
				Metadata:  parseMetadata(ui.Metadata),
			}
		}
	}

	// Call store to fork block
	newBlock, err := s.Store.ForkBlock(ctx, req.Id, reason, replaceUserInputs)
	if err != nil {
		// Check for validation errors (e.g., empty reason)
		if strings.Contains(err.Error(), "cannot be empty") {
			return nil, status.Errorf(codes.InvalidArgument, "fork reason: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to fork block: %v", err)
	}

	slog.Info("Forked block",
		"parent_id", req.Id,
		"new_block_id", newBlock.ID,
		"reason", reason,
		"user_id", user.ID,
		"replaced_inputs", len(replaceUserInputs) > 0,
		"fork_type", newBlock.Metadata["fork_type"],
	)

	return convertBlockFromStore(newBlock), nil
}

// ListBlockBranches lists all child blocks of a given block.
func (s *AIService) ListBlockBranches(ctx context.Context, req *v1pb.ListBlockBranchesRequest) (*v1pb.ListBlockBranchesResponse, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get root block and verify ownership
	rootBlock, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "root block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &rootBlock.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	// Get all child blocks recursively
	children, err := s.Store.ListChildBlocks(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list child blocks: %v", err)
	}

	// Get active path for this conversation
	activePath := ""
	activeBlocks, err := s.Store.GetActivePath(ctx, rootBlock.ConversationID)
	if err == nil && len(activeBlocks) > 0 {
		// Build active path from branch_path of latest block
		for _, b := range activeBlocks {
			if b.BranchPath != "" {
				activePath = b.BranchPath
			}
		}
	}

	// Build branch tree structure
	branches := buildBranchTree(children, activePath)

	return &v1pb.ListBlockBranchesResponse{
		Branches:         branches,
		ActiveBranchPath: activePath,
	}, nil
}

// SwitchBranch switches the active branch for a conversation.
func (s *AIService) SwitchBranch(ctx context.Context, req *v1pb.SwitchBranchRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Verify conversation ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.ConversationId,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	// Archive all blocks that are not on the target branch
	// This is done by setting archived_at for blocks with different branch paths
	now := time.Now().Unix()
	if err := s.Store.ArchiveInactiveBranches(ctx, req.ConversationId, req.TargetBranchPath, now); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to switch branch: %v", err)
	}

	slog.Info("Switched branch",
		"conversation_id", req.ConversationId,
		"target_branch_path", req.TargetBranchPath,
		"user_id", user.ID,
	)

	return &emptypb.Empty{}, nil
}

// DeleteBranch deletes a block and all its descendants.
func (s *AIService) DeleteBranch(ctx context.Context, req *v1pb.DeleteBranchRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Get block and verify ownership
	block, err := s.Store.GetAIBlock(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &block.ConversationID,
		CreatorID: &user.ID,
	})
	if err != nil || len(conversations) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "access denied to this block")
	}

	// Delete branch with optional cascade
	if err := s.Store.DeleteBranch(ctx, req.Id, req.Cascade); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete branch: %v", err)
	}

	slog.Info("Deleted branch",
		"block_id", req.Id,
		"cascade", req.Cascade,
		"user_id", user.ID,
	)

	return &emptypb.Empty{}, nil
}

// buildBranchTree builds a hierarchical tree structure from flat block list.
func buildBranchTree(blocks []*store.AIBlock, activePath string) []*v1pb.BlockBranch {
	// Create a map from parent ID to children
	childrenMap := make(map[int64][]*store.AIBlock)

	for _, b := range blocks {
		if b.ParentBlockID != nil {
			childrenMap[*b.ParentBlockID] = append(childrenMap[*b.ParentBlockID], b)
		}
	}

	// Build tree recursively
	result := make([]*v1pb.BlockBranch, 0, len(blocks))

	for _, b := range blocks {
		branch := convertBlockToBranch(b, childrenMap, activePath)
		result = append(result, branch)
	}

	return result
}

// convertBlockToBranch converts a block to BlockBranch with recursive children.
func convertBlockToBranch(b *store.AIBlock, childrenMap map[int64][]*store.AIBlock, activePath string) *v1pb.BlockBranch {
	branch := &v1pb.BlockBranch{
		Block:      convertBlockFromStore(b),
		BranchPath: b.BranchPath,
		IsActive:   b.BranchPath == activePath,
		Children:   []*v1pb.BlockBranch{},
	}

	// Recursively add children
	children := childrenMap[b.ID]
	for _, child := range children {
		branch.Children = append(branch.Children, convertBlockToBranch(child, childrenMap, activePath))
	}

	return branch
}
