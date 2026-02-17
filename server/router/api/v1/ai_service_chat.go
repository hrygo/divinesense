package v1

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/agents/orchestrator"
	ctxpkg "github.com/hrygo/divinesense/ai/context"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	aichat "github.com/hrygo/divinesense/server/router/api/v1/ai"
	"github.com/hrygo/divinesense/store"
)

// getChatEventBus returns the chat event bus, initializing it on first use.
func (s *AIService) getChatEventBus() *aichat.EventBus {
	s.chatEventBusMu.Lock()
	defer s.chatEventBusMu.Unlock()

	if s.chatEventBus == nil {
		s.chatEventBus = aichat.NewEventBus()
		s.conversationService = aichat.NewConversationService(s.Store)
		s.conversationService.Subscribe(s.chatEventBus)
	}

	return s.chatEventBus
}

// getContextBuilder returns the context builder, initializing it on first use.
func (s *AIService) getContextBuilder() *aichat.ContextBuilder {
	s.contextBuilderMu.Lock()
	defer s.contextBuilderMu.Unlock()

	if s.contextBuilder == nil {
		s.contextBuilder = aichat.NewContextBuilder(s.Store)
	}
	return s.contextBuilder
}

// getConversationSummarizer returns the conversation summarizer, initializing on first use.
func (s *AIService) getConversationSummarizer() *aichat.ConversationSummarizer {
	s.conversationSummarizerMu.Lock()
	defer s.conversationSummarizerMu.Unlock()

	if s.conversationSummarizer == nil {
		s.conversationSummarizer = aichat.NewConversationSummarizerWithStore(
			s.Store,
			s.LLMService,
			11, // Default threshold
		)
	}
	return s.conversationSummarizer
}

// Chat streams a chat response with AI agents.
// Emits events for conversation persistence (handled by ConversationService).
func (s *AIService) Chat(req *v1pb.ChatRequest, stream v1pb.AIService_ChatServer) error {
	ctx := stream.Context()

	if !s.IsEnabled() {
		return status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	if !s.IsLLMEnabled() {
		return status.Errorf(codes.Unavailable, "LLM service is not available")
	}

	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	userKey := strconv.FormatInt(int64(user.ID), 10)
	if !globalAILimiter.Allow(userKey) {
		return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	chatReq := aichat.ToChatRequest(req)
	chatReq.UserID = user.ID

	if chatReq.Timezone == "" || !aichat.IsValidTimezone(chatReq.Timezone) {
		chatReq.Timezone = aichat.GetDefaultTimezone()
	}

	// Get event bus (initializes on first use)
	eventBus := s.getChatEventBus()

	// Emit conversation start event to trigger conversation creation
	event := &aichat.ChatEvent{
		Type:               aichat.EventConversationStart,
		UserID:             user.ID,
		AgentType:          chatReq.AgentType,
		ConversationID:     chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:          time.Now().Unix(),
	}
	results, err := eventBus.Publish(ctx, event)
	if err != nil {
		slog.Default().Warn("Conversation persistence issue during start",
			"user_id", user.ID,
			"error", err,
		)
	}

	// Get conversation ID from listener result
	if len(results) > 0 {
		if convID, ok := results[0].(int32); ok && convID != 0 {
			chatReq.ConversationID = convID
		}
	}

	// Handle separator (---) - emit event and return without agent processing
	if req.Message == "---" && chatReq.ConversationID != 0 {
		_, _ = eventBus.Publish(ctx, &aichat.ChatEvent{
			Type:               aichat.EventSeparator,
			UserID:             user.ID,
			AgentType:          chatReq.AgentType,
			SeparatorContent:   "Context cleared",
			ConversationID:     chatReq.ConversationID,
			IsTempConversation: chatReq.IsTempConversation,
			Timestamp:          time.Now().Unix(),
		})
		return stream.Send(&v1pb.ChatResponse{Done: true})
	}

	// Emit user message event
	// Emit user message event
	slog.Info("ai.chat.user_message",
		"user_id", user.ID,
		"conversation_id", chatReq.ConversationID,
		"message", req.Message,
		"agent_type", chatReq.AgentType,
	)
	_, _ = eventBus.Publish(ctx, &aichat.ChatEvent{
		Type:               aichat.EventUserMessage,
		UserID:             user.ID,
		AgentType:          chatReq.AgentType,
		UserMessage:        req.Message,
		ConversationID:     chatReq.ConversationID,
		IsTempConversation: chatReq.IsTempConversation,
		Timestamp:          time.Now().Unix(),
	})

	// Build conversation context from backend
	// This ensures SEPARATOR filtering is enforced server-side
	var history []string
	if chatReq.ConversationID != 0 {
		builder := s.getContextBuilder()
		builtContext, err := builder.BuildContext(ctx, chatReq.ConversationID, &aichat.ContextControl{
			// Pending messages: the current user message (not yet persisted)
			PendingMessages: []aichat.Message{
				{
					Content: req.Message,
					Role:    "user",
					Type:    "MESSAGE",
				},
			},
		})
		if err != nil {
			slog.Default().Warn("Failed to build context from backend",
				"conversation_id", chatReq.ConversationID,
				"error", err,
			)
		} else {
			// Exclude the current message from history (it's the last pending message)
			if len(builtContext.Messages) > 0 {
				history = builtContext.Messages[:len(builtContext.Messages)-1]
			}
			slog.Default().Debug("Built context from backend",
				"conversation_id", chatReq.ConversationID,
				"message_count", len(history),
				"token_count", builtContext.TokenCount,
				"separator_pos", builtContext.SeparatorPos,
				"has_pending", builtContext.HasPending,
			)
		}
	}

	// Note: History field removed - backend-driven context construction (context-engineering.md Phase 1)
	// History is now built by ContextBuilder in handler.go
	// chatReq.History = history // Removed

	// Create handler and process request
	handler := s.createChatHandler()

	// Wrap stream to collect assistant response
	collectingStream := &eventCollectingStream{
		grpcStreamWrapper: &grpcStreamWrapper{stream: stream},
		service:           s,
		eventBus:          eventBus,
		userID:            user.ID,
		agentType:         chatReq.AgentType,
		conversationID:    chatReq.ConversationID,
		isTemp:            chatReq.IsTempConversation,
	}

	if err := handler.Handle(ctx, chatReq, collectingStream); err != nil {
		return aichat.HandleError(err)
	}

	return nil
}

// createChatHandler creates the chat handler.
func (s *AIService) createChatHandler() aichat.Handler {
	// Get cached agent factory (initializes on first use)
	factory := s.getAgentFactory()

	// Phase 5: Create BlockManager for Unified Block Model support
	blockManager := aichat.NewBlockManager(s.Store)
	parrotHandler := aichat.NewParrotHandler(factory, s.LLMService, s.persister, blockManager, s.TitleGenerator)

	// Configure chat router for auto-routing.
	// routerSvc provides two-layer routing (cache → rule).
	// Orchestrator handles LLM-based task decomposition when needed.
	routerSvc := s.getRouterService()
	chatRouter := aichat.NewChatRouter(routerSvc)
	if s.IntentClassifierConfig != nil && s.IntentClassifierConfig.Enabled {
		slog.Info("Chat router enabled with cache + rule routing",
			"model", s.IntentClassifierConfig.Model,
		)
	} else {
		slog.Info("Chat router enabled with rule-based routing (no LLM fallback)")
	}

	// P0 fix: Enable metadata-based sticky routing (context-engineering.md Phase 2)
	// This allows routing decisions to be based on persisted database state (AIBlock.Metadata),
	// not just in-memory session state.
	metadataMgr := ctxpkg.NewMetadataManager(s.Store, 5*time.Minute) // 5 min cache TTL
	chatRouterWithMetadata := agentpkg.NewChatRouterWithMetadata(chatRouter, metadataMgr)
	parrotHandler.SetChatRouterWithMetadata(chatRouterWithMetadata)
	parrotHandler.SetMetadataManager(metadataMgr)
	slog.Info("Chat router with metadata-based sticky routing enabled")

	// P0-2: Enable backend-driven context construction (context-engineering.md Phase 1)
	// This replaces client-side history with server-side context building.
	// The ContextBuilder fetches history from AIBlockStore instead of trusting req.History.
	storeAdapter := ctxpkg.NewStoreAdapter(s.Store)
	msgProvider := ctxpkg.NewBlockStoreMessageProvider(storeAdapter, 0) // userID not used in GetRecentMessages
	contextBuilder := ctxpkg.NewService(ctxpkg.DefaultConfig()).WithMessageProvider(msgProvider)

	// Phase 3: Inject EpisodicProvider for long-term memory retrieval
	// This enables semantic search over past conversation episodes.
	if s.EmbeddingService != nil {
		vectorSearchAdapter := ctxpkg.NewVectorSearchStoreAdapter(s.Store)
		episodicProvider := ctxpkg.NewEpisodicProvider(
			vectorSearchAdapter,
			s.EmbeddingService, // EmbeddingService implements ctxpkg.EmbeddingService
			ctxpkg.DefaultEpisodicConfig(),
			"", // agentType is set per-request
		)
		contextBuilder = contextBuilder.WithEpisodicProvider(episodicProvider)
		slog.Info("Episodic memory provider enabled for context building")
	}

	parrotHandler.SetContextBuilder(contextBuilder)
	slog.Info("Backend-driven context construction enabled")

	// P0-3: Create and inject Orchestrator for handoff support
	// Orchestrator handles: (1) needs_orchestration=true requests, (2) expert handoff when report_inability is called
	// This enables seamless expert switching when the initial expert cannot handle the task.
	if s.LLMService != nil && factory.GetParrotFactory() != nil {
		// Get expert configurations from factory
		expertConfigs := factory.GetSelfCognitionConfigs()

		// Build CapabilityMap for handoff expert lookup
		// CapabilityMap knows all experts' capabilities, used to find alternative experts
		if len(expertConfigs) > 0 {
			cm := orchestrator.NewCapabilityMap()
			cm.BuildFromConfigs(expertConfigs)
			cm.BuildKeywordIndex(expertConfigs)
			parrotHandler.SetCapabilityMap(cm)
			slog.Info("CapabilityMap initialized for handoff support")
		}

		// Create ExpertRegistry from ParrotFactory
		// Note: userID is set per-request in ExecuteExpert, so we use 0 here as placeholder
		expertRegistry := orchestrator.NewParrotExpertRegistry(factory.GetParrotFactory(), 0)

		// Create Orchestrator with handoff enabled
		orch := orchestrator.NewOrchestrator(
			s.LLMService,
			expertRegistry,
			orchestrator.WithHandoff(true),
			orchestrator.WithAggregation(true),
		)
		parrotHandler.SetOrchestrator(orch)
		slog.Info("Orchestrator enabled with handoff support")
	}

	return aichat.NewRoutingHandler(parrotHandler)
}

// grpcStreamWrapper wraps the gRPC stream to implement aichat.ChatStream.
type grpcStreamWrapper struct {
	stream v1pb.AIService_ChatServer
}

func (w *grpcStreamWrapper) Send(resp *v1pb.ChatResponse) error {
	return w.stream.Send(resp)
}

func (w *grpcStreamWrapper) Context() context.Context {
	return w.stream.Context()
}

// eventCollectingStream wraps the stream and emits assistant response events.
type eventCollectingStream struct {
	*grpcStreamWrapper
	service        *AIService
	eventBus       *aichat.EventBus
	agentType      aichat.AgentType
	builder        strings.Builder
	mu             sync.Mutex
	userID         int32
	conversationID int32
	isTemp         bool
}

func (s *eventCollectingStream) Send(resp *v1pb.ChatResponse) error {
	// Log for debugging
	if resp.Done {
		slog.Info("eventCollectingStream: Sending done=true to frontend",
			"has_summary", resp.BlockSummary != nil,
			"event_type", resp.EventType)
	}

	// Collect content from "answer" or "content" events
	if resp.EventType == "answer" || resp.EventType == "content" {
		s.mu.Lock()
		s.builder.WriteString(resp.EventData)
		s.mu.Unlock()
	}

	// When stream is done, emit assistant response event
	if resp.Done {
		s.mu.Lock()
		response := s.builder.String()
		s.mu.Unlock()

		if response != "" {
			_, _ = s.eventBus.Publish(s.Context(), &aichat.ChatEvent{
				Type:               aichat.EventAssistantResponse,
				UserID:             s.userID,
				AgentType:          s.agentType,
				AssistantResponse:  response,
				ConversationID:     s.conversationID,
				IsTempConversation: s.isTemp,
				Timestamp:          time.Now().Unix(),
			})
		}

		// Check if summarization is needed (async, don't block response)
		// Only summarize for non-temporary conversations
		if !s.isTemp && s.conversationID != 0 {
			// Use WithoutCancel to detach from request context while preserving service shutdown
			// TODO: Move to a proper background worker pool with lifecycle management
			bgCtx := context.WithoutCancel(s.Context())
			go func() {
				// Add panic recovery to prevent goroutine leaks
				defer func() {
					if r := recover(); r != nil {
						slog.Default().Error("Summarization goroutine panic",
							"conversation_id", s.conversationID,
							"panic", r,
						)
					}
				}()

				summarizer := s.service.getConversationSummarizer()
				if shouldSummarize, count := summarizer.ShouldSummarize(bgCtx, s.conversationID); shouldSummarize {
					slog.Default().Info("Conversation threshold reached, triggering summarization",
						"conversation_id", s.conversationID,
						"message_count", count,
					)
					// Use independent timeout for summarization (not tied to request)
					summarizeCtx, cancel := context.WithTimeout(bgCtx, 30*time.Second)
					defer cancel()
					if err := summarizer.Summarize(summarizeCtx, s.conversationID); err != nil {
						slog.Default().Warn("Failed to summarize conversation",
							"conversation_id", s.conversationID,
							"error", err,
						)
					}
				}
			}()
		}
	}

	return s.grpcStreamWrapper.Send(resp)
}

// StopChat cancels an ongoing chat stream and terminates the associated session.
// This is the implementation for session.stop from the async architecture spec.
// StopChat 取消正在进行的聊天流并终止相关会话。
// 这是异步架构规范中 session.stop 的实现。
//
// Architecture Note: Session termination is primarily client-driven.
//   - The client should cancel the streaming request (gRPC/HTTP) to immediately stop processing.
//   - This method emits monitoring events for observability and metrics collection.
//   - Active sessions are cleaned up after a 30-minute idle timeout (CCSessionManager).
//   - For server-initiated termination in future, consider adding a session registry
//     that maps conversationID to active context.CancelFunc for immediate cancellation.
func (s *AIService) StopChat(ctx context.Context, req *v1pb.StopChatRequest) (*emptypb.Empty, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Authorization check: verify user owns the conversation
	// 权限检查：验证用户是否拥有该会话
	if req.ConversationId > 0 {
		conversations, err := s.Store.ListAIConversations(ctx, &store.FindAIConversation{
			ID: &req.ConversationId,
		})
		if err != nil {
			slog.Warn("StopChat: conversation lookup failed",
				"user_id", user.ID,
				"conversation_id", req.ConversationId,
				"error", err,
			)
			// Don't fail on lookup error - may be a transient issue
			// 查找失败时不返回错误（可能是临时问题）
		} else if len(conversations) == 0 {
			slog.Warn("StopChat: conversation not found",
				"user_id", user.ID,
				"conversation_id", req.ConversationId,
			)
			// Conversation may have been deleted - don't fail
			// 会话可能已被删除 - 不返回错误
		} else if conversations[0].CreatorID != user.ID {
			slog.Warn("StopChat: user attempted to stop another user's conversation",
				"user_id", user.ID,
				"conversation_id", req.ConversationId,
				"conversation_owner", conversations[0].CreatorID,
			)
			return nil, status.Errorf(codes.PermissionDenied, "you can only stop your own conversations")
		}
	}

	slog.Info("StopChat called",
		"user_id", user.ID,
		"conversation_id", req.ConversationId,
		"reason", req.Reason,
	)

	// Emit stop event for monitoring, metrics, and potential async cleanup handlers
	if eventBus := s.getChatEventBus(); eventBus != nil {
		_, _ = eventBus.Publish(ctx, &aichat.ChatEvent{
			Type:           "chat_stop",
			UserID:         user.ID,
			ConversationID: req.ConversationId,
			Timestamp:      time.Now().Unix(),
		})
	}

	// Note: The primary mechanism for stopping is client-side stream closure.
	// The backend will clean up idle sessions via the 30-minute timeout.
	// For immediate cleanup, the client should cancel the streaming request.
	//
	// Future enhancement: Add a session registry to track active requests and enable
	// server-initiated cancellation via context.CancelFunc.
	// Example:
	//   if cancelFunc, ok := s.getActiveSessionCancel(req.ConversationId); ok {
	//       cancelFunc()
	//   }

	return &emptypb.Empty{}, nil
}
