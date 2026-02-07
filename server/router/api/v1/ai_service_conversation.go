package v1

import (
	"context"
	"log/slog"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
)

// MaxBlockLimit is the maximum number of blocks to return in a single request.
const MaxBlockLimit = 100

func (s *AIService) ListAIConversations(ctx context.Context, _ *v1pb.ListAIConversationsRequest) (*v1pb.ListAIConversationsResponse, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// BlockCount is now populated by SQL JOIN in store layer (N+1 fix)
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conversations: %v", err)
	}

	response := &v1pb.ListAIConversationsResponse{
		Conversations: make([]*v1pb.AIConversation, 0, len(conversations)),
	}
	for _, c := range conversations {
		pbConv := convertAIConversationFromStore(c)
		pbConv.BlockCount = c.BlockCount // Use pre-fetched block count
		response.Conversations = append(response.Conversations, pbConv)
	}

	return response, nil
}

func (s *AIService) GetAIConversation(ctx context.Context, req *v1pb.GetAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	conversation := conversations[0]

	// Load blocks from database
	blocks, err := s.Store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversation.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list blocks: %v", err)
	}

	pbConversation := convertAIConversationFromStore(conversation)
	pbConversation.Blocks = convertBlocksFromStore(blocks)
	pbConversation.BlockCount = int32(len(blocks))

	return pbConversation, nil
}

func (s *AIService) CreateAIConversation(ctx context.Context, req *v1pb.CreateAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	now := time.Now().Unix()
	conversation, err := s.Store.CreateAIConversation(ctx, &store.AIConversation{
		UID:       shortuuid.New(),
		CreatorID: user.ID,
		Title:     req.Title,
		ParrotID:  req.ParrotId.String(),
		CreatedTs: now,
		UpdatedTs: now,
		RowStatus: store.Normal,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create conversation: %v", err)
	}

	return convertAIConversationFromStore(conversation), nil
}

func (s *AIService) UpdateAIConversation(ctx context.Context, req *v1pb.UpdateAIConversationRequest) (*v1pb.AIConversation, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Check ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	update := &store.UpdateAIConversation{
		ID:        req.Id,
		UpdatedTs: func() *int64 { t := time.Now().Unix(); return &t }(),
	}
	if req.Title != nil {
		update.Title = req.Title
	}

	updated, err := s.Store.UpdateAIConversation(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update conversation: %v", err)
	}

	return convertAIConversationFromStore(updated), nil
}

func (s *AIService) DeleteAIConversation(ctx context.Context, req *v1pb.DeleteAIConversationRequest) (*emptypb.Empty, error) {
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Check ownership
	conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
		ID:        &req.Id,
		CreatorID: &user.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}
	if len(conversations) == 0 {
		return nil, status.Errorf(codes.NotFound, "conversation not found")
	}

	if err := s.Store.DeleteAIConversation(ctx, &store.DeleteAIConversation{ID: req.Id}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete conversation: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *AIService) AddContextSeparator(ctx context.Context, req *v1pb.AddContextSeparatorRequest) (*emptypb.Empty, error) {
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

	// Prevent duplicate SEPARATOR: check if the last block is already a SEPARATOR
	blocks, err := s.Store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &req.ConversationId,
	})
	if err == nil && len(blocks) > 0 {
		lastBlock := blocks[len(blocks)-1]
		if lastBlock.BlockType == store.AIBlockTypeContextSeparator {
			// Last block is already a SEPARATOR, silently succeed (idempotent)
			return &emptypb.Empty{}, nil
		}
	}

	// Create SEPARATOR block
	_, err = s.Store.CreateAIBlock(ctx, &store.CreateAIBlock{
		UID:            shortuuid.New(),
		ConversationID: req.ConversationId,
		BlockType:      store.AIBlockTypeContextSeparator,
		Mode:           store.AIBlockModeNormal,
		UserInputs:     []store.UserInput{},
		CreatedTs:      time.Now().UnixMilli(),
		UpdatedTs:      time.Now().UnixMilli(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create separator block: %v", err)
	}

	// Update conversation timestamp
	now := time.Now().Unix()
	_, err = s.Store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
		ID:        req.ConversationId,
		UpdatedTs: &now,
	})
	if err != nil {
		slog.Default().Warn("Failed to update conversation timestamp after adding separator",
			"conversation_id", req.ConversationId,
			"error", err,
		)
	}

	return &emptypb.Empty{}, nil
}

func convertAIConversationFromStore(c *store.AIConversation) *v1pb.AIConversation {
	// Convert ParrotID string to AgentType enum
	// Handle both short format ("MEMO") and long format ("AGENT_TYPE_MEMO")
	// DEFAULT and CREATIVE are deprecated - map to AMAZING
	var parrotId int32

	// Try direct lookup first (long format like "AGENT_TYPE_MEMO")
	if val, ok := v1pb.AgentType_value[c.ParrotID]; ok {
		parrotId = val
	} else {
		// Try short format lookup ("MEMO" → "AGENT_TYPE_MEMO")
		// Legacy: DEFAULT/CREATIVE → AMAZING
		shortToLong := map[string]v1pb.AgentType{
			"MEMO":     v1pb.AgentType_AGENT_TYPE_MEMO,
			"SCHEDULE": v1pb.AgentType_AGENT_TYPE_SCHEDULE,
			"AMAZING":  v1pb.AgentType_AGENT_TYPE_AMAZING,
			"DEFAULT":  v1pb.AgentType_AGENT_TYPE_AMAZING, // Legacy alias
			"CREATIVE": v1pb.AgentType_AGENT_TYPE_AMAZING, // Legacy alias
		}
		if val, ok := shortToLong[c.ParrotID]; ok {
			parrotId = int32(val)
		} else {
			// Unknown value, log warning and fallback to AMAZING
			slog.Default().Warn("Unknown ParrotID in conversation, falling back to AMAZING",
				"conversation_id", c.ID,
				"parrot_id", c.ParrotID,
			)
			parrotId = int32(v1pb.AgentType_AGENT_TYPE_AMAZING)
		}
	}

	return &v1pb.AIConversation{
		Id:        c.ID,
		Uid:       c.UID,
		CreatorId: c.CreatorID,
		Title:     c.Title,
		ParrotId:  v1pb.AgentType(parrotId),
		Pinned:    c.Pinned,
		CreatedTs: c.CreatedTs,
		UpdatedTs: c.UpdatedTs,
	}
}

// convertBlocksFromStore converts store.AIBlock slices to protobuf Block slices.
func convertBlocksFromStore(blocks []*store.AIBlock) []*v1pb.Block {
	result := make([]*v1pb.Block, 0, len(blocks))
	for _, b := range blocks {
		result = append(result, convertBlockFromStore(b))
	}
	return result
}

// ClearConversationMessages deletes all blocks in a conversation.
func (s *AIService) ClearConversationMessages(ctx context.Context, req *v1pb.ClearConversationMessagesRequest) (*emptypb.Empty, error) {
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

	// Delete all blocks in the conversation
	blocks, err := s.Store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &req.ConversationId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list blocks: %v", err)
	}

	for _, block := range blocks {
		if err := s.Store.DeleteAIBlock(ctx, block.ID); err != nil {
			slog.Default().Warn("Failed to delete block",
				"block_id", block.ID,
				"error", err,
			)
		}
	}

	// Update conversation timestamp
	now := time.Now().Unix()
	_, err = s.Store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
		ID:        req.ConversationId,
		UpdatedTs: &now,
	})
	if err != nil {
		slog.Default().Warn("Failed to update conversation timestamp after clearing messages",
			"conversation_id", req.ConversationId,
			"error", err,
		)
	}

	return &emptypb.Empty{}, nil
}
