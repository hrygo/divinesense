package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/ai"
	agentpkg "github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/agents/geek"
	"github.com/hrygo/divinesense/ai/agents/orchestrator"
	ctxpkg "github.com/hrygo/divinesense/ai/context"
	"github.com/hrygo/divinesense/ai/memory"
	"github.com/hrygo/divinesense/ai/routing"
	aistats "github.com/hrygo/divinesense/ai/services/stats"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/internal/errors"
	"github.com/hrygo/divinesense/server/internal/observability"
	"github.com/hrygo/divinesense/store"
)

// ChatStream represents the streaming response interface for AI chat.
type ChatStream interface {
	Send(*v1pb.ChatResponse) error
	Context() context.Context
}

// ParrotHandler handles all parrot agent requests (DEFAULT, MEMO, SCHEDULE, GENERAL, CREATIVE).
type ParrotHandler struct {
	factory                *AgentFactory
	llm                    ai.LLMService
	chatRouter             *agentpkg.ChatRouter
	chatRouterWithMetadata *agentpkg.ChatRouterWithMetadata // P0 fix: sticky routing with metadata
	orchestrator           *orchestrator.Orchestrator       // Orchestrator for complex/multi-intent requests
	capabilityMap          *orchestrator.CapabilityMap      // CapabilityMap for handoff expert lookup
	persister              *aistats.Persister               // session stats persister
	blockManager           *BlockManager                    // Phase 5: Unified Block Model support
	titleGenerator         *ai.TitleGenerator               // Title generator for auto-naming conversations
	metadataMgr            *ctxpkg.MetadataManager          // Context engineering: metadata-based sticky routing
	contextBuilder         *ctxpkg.Service                  // P0 fix: backend-driven context construction
	memoryGenerator        memory.Generator                 // Phase 3: async episodic memory generation (extension point)
	geekRunner             *agentpkg.CCRunner               // Singleton CCRunner for Geek mode
	evoRunner              *agentpkg.CCRunner               // Singleton CCRunner for Evolution mode
}

// NewParrotHandler creates a new parrot handler.
func NewParrotHandler(factory *AgentFactory, llm ai.LLMService, persister *aistats.Persister, blockManager *BlockManager, titleGenerator *ai.TitleGenerator) *ParrotHandler {
	// Create singletons for CC execution. Evolution and Geek use isolated runners.
	geekRunner, err := agentpkg.NewCCRunner(30*time.Minute, slog.Default()) // Long timeout for active shell
	if err != nil {
		slog.Warn("Failed to create geekRunner in init (CLI not found?)", "error", err)
	}
	evoRunner, err := agentpkg.NewCCRunner(30*time.Minute, slog.Default())
	if err != nil {
		slog.Warn("Failed to create evoRunner in init (CLI not found?)", "error", err)
	}

	return &ParrotHandler{
		factory:        factory,
		llm:            llm,
		persister:      persister,
		blockManager:   blockManager, // Phase 5
		titleGenerator: titleGenerator,
		geekRunner:     geekRunner,
		evoRunner:      evoRunner,
	}
}

// SetChatRouter configures the intelligent chat router for auto-routing.
func (h *ParrotHandler) SetChatRouter(router *agentpkg.ChatRouter) {
	h.chatRouter = router
}

// SetChatRouterWithMetadata configures the chat router with metadata-based sticky routing.
// This enables persistent routing state across sessions using database-stored metadata.
// P0 fix: enables context-engineering.md Phase 2 sticky routing.
func (h *ParrotHandler) SetChatRouterWithMetadata(router *agentpkg.ChatRouterWithMetadata) {
	h.chatRouterWithMetadata = router
	// Also set base router for fallback
	if router != nil {
		h.chatRouter = router.ChatRouter
	}
}

// SetOrchestrator configures the orchestrator for complex/multi-intent requests.
func (h *ParrotHandler) SetOrchestrator(orch *orchestrator.Orchestrator) {
	h.orchestrator = orch
}

// SetCapabilityMap configures the capability map for handoff expert lookup.
// Orchestrator uses this to find alternative experts when an expert reports inability.
func (h *ParrotHandler) SetCapabilityMap(cm *orchestrator.CapabilityMap) {
	h.capabilityMap = cm
}

// SetMetadataManager configures the metadata manager for context engineering.
// This enables metadata-based sticky routing and state persistence.
func (h *ParrotHandler) SetMetadataManager(mgr *ctxpkg.MetadataManager) {
	h.metadataMgr = mgr
}

// Close gracefully shuts down all managed singleton runners and active sessions.
func (h *ParrotHandler) Close() error {
	slog.Info("Shutting down ParrotHandler singletons")

	if h.geekRunner != nil {
		h.geekRunner.Close()
	}
	if h.evoRunner != nil {
		h.evoRunner.Close()
	}

	return nil
}

// SetContextBuilder configures the context builder for backend-driven context construction.
// P0 fix: enables context-engineering.md Phase 1 backend-driven context.
func (h *ParrotHandler) SetContextBuilder(builder *ctxpkg.Service) {
	h.contextBuilder = builder
}

// SetMemoryGenerator configures the memory generator for async episodic memory creation.
// Phase 3: enables context-engineering.md Phase 3 memory generation.
// Default is NoOpGenerator (no-op). Use simple.Generator for dev/test, or integrate
// with professional memory services (Mem0, Letta) for production.
func (h *ParrotHandler) SetMemoryGenerator(gen memory.Generator) {
	h.memoryGenerator = gen
}

// maybeGenerateConversationTitle auto-generates a conversation title for the first block.
// Only generates if title_source is "default" (never been auto-generated or user-edited).
// Runs asynchronously in a background goroutine to avoid blocking the chat flow.
// Optimization: Called immediately after block creation (not after block completion) for parallel execution.
func (h *ParrotHandler) maybeGenerateConversationTitle(ctx context.Context, conversationID int32, userMessage string) {
	// Run asynchronously in background - don't block the chat flow
	go h.generateTitleAsync(conversationID, userMessage)
}

// generateTitleAsync generates and updates the conversation title in the background.
// Uses only userMessage for early title generation (parallel with Orchestrator processing).
func (h *ParrotHandler) generateTitleAsync(conversationID int32, userMessage string) {
	// Use a fresh context with timeout for the title generation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if this is the first block for this conversation
	blocks, err := h.factory.store.ListAIBlocks(ctx, &store.FindAIBlock{
		ConversationID: &conversationID,
	})
	if err != nil {
		slog.Warn("Failed to list blocks for title generation",
			"conversation_id", conversationID,
			"error", err.Error(),
		)
		return
	}

	// Only generate title for the first block
	if len(blocks) != 1 {
		return
	}

	// Check conversation title_source
	conversations, err := h.factory.store.ListAIConversations(ctx, &store.FindAIConversation{
		ID: &conversationID,
	})
	if err != nil || len(conversations) == 0 {
		slog.Warn("Failed to get conversation for title generation",
			"conversation_id", conversationID,
			"error", err.Error(),
		)
		return
	}

	conv := conversations[0]
	// Only generate if title_source is "default" (never been auto-generated or user-edited)
	if conv.TitleSource != store.TitleSourceDefault {
		return
	}

	// Generate title from user message only (parallel optimization)
	// AI response is empty for early generation
	title, err := h.titleGenerator.Generate(ctx, userMessage, "")
	if err != nil {
		slog.Warn("Failed to generate conversation title",
			"conversation_id", conversationID,
			"error", err.Error(),
		)
		return
	}

	// Update conversation with generated title
	_, err = h.factory.store.UpdateAIConversation(ctx, &store.UpdateAIConversation{
		ID:          conversationID,
		Title:       &title,
		TitleSource: storePtr(store.TitleSourceAuto),
	})
	if err != nil {
		slog.Warn("Failed to update conversation title",
			"conversation_id", conversationID,
			"error", err.Error(),
		)
		return
	}

	slog.Info("Auto-generated conversation title",
		"conversation_id", conversationID,
		"title", title,
	)
}

// storePtr returns a pointer to the given value.
func storePtr[T any](v T) *T {
	return &v
}

// Handle implements Handler interface for parrot agent requests.
func (h *ParrotHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	// IMPORTANT: Log at INFO level to see evolution_mode value
	slog.Info("AI chat handler received request",
		"agent_type", req.AgentType,
		"geek_mode", req.GeekMode,
		"evolution_mode", req.EvolutionMode,
		"evolution_mode_raw", fmt.Sprintf("%v", req.EvolutionMode),
	)

	// PRIORITY CHECK: EvolutionMode has highest priority (admin-only, self-evolution)
	// 优先检查：进化模式具有最高优先级（仅管理员，自我进化）
	if req.EvolutionMode {
		slog.Info("Evolution mode detected, routing to EvolutionParrot")
		return h.handleEvolutionMode(ctx, req, stream)
	}

	// PRIORITY CHECK: GeekMode bypasses ALL normal routing
	// 优先检查：极客模式绕过所有常规路由
	if req.GeekMode {
		return h.handleGeekMode(ctx, req, stream)
	}

	// PROGRESS EVENT: Send received event immediately to acknowledge message receipt
	// 进度事件：立即发送 received 事件以确认收到消息，消除死寂感
	if err := stream.Send(&v1pb.ChatResponse{
		EventType: "received",
	}); err != nil {
		slog.Warn("failed to send received event", "error", err)
		// Non-critical error, continue processing
	}

	if h.llm == nil {
		return status.Error(codes.Unavailable, "LLM service is not available")
	}

	// Auto-route if AgentType is AUTO
	agentType := req.AgentType
	var needsOrchestration bool

	if agentType == AgentTypeAuto && h.chatRouter != nil {
		// Add user ID to context for history matching.
		// Note: req.UserID is already authenticated by the gRPC interceptor middleware.
		ctx = routing.WithUserID(ctx, req.UserID)

		// PROGRESS EVENT: Send routing_start event before intent routing
		// 进度事件：发送 routing_start 事件表示开始理解意图
		startTime := time.Now()
		if err := stream.Send(&v1pb.ChatResponse{
			EventType: "routing_start",
			EventData: `{"layer":"fastrouter"}`,
		}); err != nil {
			slog.Warn("failed to send routing_start event", "error", err)
		}

		// Execute routing with metadata-based sticky routing if available
		// P0 fix: enables context-engineering.md Phase 2 sticky routing
		var routeResult *agentpkg.ChatRouteResult
		var err error
		if h.chatRouterWithMetadata != nil && req.ConversationID > 0 {
			// Use sticky routing with metadata (blockID=0 means persist later after block creation)
			routeResult, err = h.chatRouterWithMetadata.RouteWithContextWithMetadata(
				ctx, req.Message, nil, req.ConversationID, 0)
			if err == nil && routeResult.Method == "metadata_sticky" {
				slog.Info("route reused from metadata sticky",
					"conversation_id", req.ConversationID,
					"route", routeResult.Route,
					"confidence", routeResult.Confidence)
			}
		} else {
			// Fallback to standard routing
			routeResult, err = h.chatRouter.Route(ctx, req.Message)
		}
		duration := time.Since(startTime)

		if err != nil {
			// Router error → Orchestrator
			needsOrchestration = true
			slog.Warn("chat router failed, using orchestrator",
				"error", err,
				"message", req.Message[:min(len(req.Message), 30)])
		} else {
			needsOrchestration = routeResult.NeedsOrchestration

			// Map ChatRouteType to AgentType
			switch routeResult.Route {
			case agentpkg.RouteTypeMemo:
				agentType = AgentTypeMemo
			case agentpkg.RouteTypeSchedule:
				agentType = AgentTypeSchedule
			default:
				// Empty route indicates unknown intent
				agentType = "" // Will trigger orchestration
			}

			// PROGRESS EVENT: Send routing_end event with agent info
			// 进度事件：发送 routing_end 事件，告知用户路由结果
			if err := stream.Send(&v1pb.ChatResponse{
				EventType: "routing_end",
				EventData: fmt.Sprintf(`{"agent":"%s","needs_orchestration":%v,"duration_ms":%d}`,
					agentType.String(), needsOrchestration, duration.Milliseconds()),
			}); err != nil {
				slog.Warn("failed to send routing_end event", "error", err)
			}

			slog.Info("chat auto-routed",
				"route", routeResult.Route,
				"method", routeResult.Method,
				"confidence", routeResult.Confidence,
				"needs_orchestration", needsOrchestration)

			// Store route result for metadata persistence
			if !needsOrchestration && routeResult.Route != "" {
				req.RouteResult = &RouteResultMeta{
					Route:      string(routeResult.Route),
					Confidence: routeResult.Confidence,
					Method:     routeResult.Method,
				}

				// Phase 2 fix: Immediately update cache for sticky routing
				// This enables the next request to use the current routing result
				// without waiting for block completion.
				if h.metadataMgr != nil && req.ConversationID > 0 {
					h.metadataMgr.UpdateCacheOnly(
						req.ConversationID,
						string(routeResult.Route),
						agentpkg.ExtractIntent(routeResult.Route),
						float32(routeResult.Confidence),
					)
				}
			}
		}
	} else if agentType == AgentTypeAuto {
		// No router configured, use orchestrator
		needsOrchestration = true
	}

	// Core branch: direct to Expert vs Orchestrator
	if needsOrchestration && h.orchestrator != nil {
		// Use Orchestrator for complex/multi-intent requests
		return h.executeWithOrchestrator(ctx, req, stream)
	} else if needsOrchestration {
		// No orchestrator available, fallback to Memo agent
		agentType = AgentTypeMemo
	}

	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), agentType.String(), req.UserID)
	logger.Info("ai.chat.started",
		slog.String("user_input", req.Message),
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", 0), // History now built by backend (context-engineering.md Phase 1)
	)

	// Create agent using factory
	agent, err := h.factory.Create(ctx, &CreateConfig{
		Type:     agentType,
		UserID:   req.UserID,
		Timezone: req.Timezone,
	})
	if err != nil {
		logger.Error("Failed to create agent", err)
		return status.Error(codes.Internal, fmt.Sprintf("failed to create agent: %v", err))
	}

	logger.Debug("Agent created",
		slog.String("agent_name", agent.Name()),
	)

	// Execute agent with streaming
	if err := h.executeAgent(ctx, agent, req, stream, logger); err != nil {
		logger.Error("AI chat failed", err)
		return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
	}

	logger.Info("ai.chat.completed",
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
}

// handleGeekMode creates and executes GeekParrot directly.
// handleGeekMode 创建并直接执行 GeekParrot。
// GeekMode bypasses all LLM processing and routing, providing direct
// access to Claude Code CLI.
// 极客模式绕过所有 LLM 处理和路由，提供对 Claude Code CLI 的直接访问。
func (h *ParrotHandler) handleGeekMode(
	ctx context.Context,
	req *ChatRequest,
	stream ChatStream,
) error {
	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), "geek", req.UserID)
	logger.Info("ai.chat.started",
		slog.String("mode", "geek"),
		slog.String("user_input", req.Message),
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", 0), // History now built by backend (context-engineering.md Phase 1)
	)

	// Send received event for immediate feedback (Task 1.2: Progress Feedback)
	// 发送 received 事件提供即时反馈（200ms 心理阈值）
	if err := stream.Send(&v1pb.ChatResponse{
		EventType: "received",
	}); err != nil {
		slog.Warn("failed to send received event", "error", err)
	}

	// Generate a stable session ID based on Conversation ID using UUID v5
	// 统一 Namespace 规则：使用模式名称(geek)作为前缀并结合 UserID，确保跨用户、跨模式完全隔离
	namespaceBase := fmt.Sprintf("geek_%d", req.UserID)
	namespace := uuid.NewMD5(uuid.NameSpaceOID, []byte(namespaceBase))
	sessionID := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("conversation_%d", req.ConversationID))).String()

	if h.geekRunner == nil {
		logger.Error("GeekRunner global singleton is null, cannot perform Hot-Multiplexing", nil)
		return status.Error(codes.Unavailable, "GeekMode CLI runner not initialized")
	}

	// Create GeekParrot directly (no LLM dependency)
	// 直接创建 GeekParrot（无 LLM 依赖），注入全局 geekRunner 单例
	geekParrot, err := geek.NewGeekParrot(
		h.geekRunner,
		h.getWorkDirForUser(req.UserID),
		req.UserID,
		sessionID,
	)
	if err != nil {
		logger.Error("Failed to create GeekParrot", err)
		return status.Error(codes.Internal, fmt.Sprintf("failed to create GeekParrot: %v", err))
	}

	// Pass detailed device context to GeekParrot
	// 将详细的设备上下文传递给极客鹦鹉
	geekParrot.SetDeviceContext(req.DeviceContext)

	logger.Debug("GeekParrot created",
		slog.String("agent_name", geekParrot.Name()),
		slog.String("work_dir", geekParrot.GetWorkDir()),
		slog.String("session_id", sessionID),
	)

	// Execute with streaming (same pattern as other agents)
	// 执行并流式输出（与其他 Agent 相同的模式）
	if err := h.executeAgent(ctx, geekParrot, req, stream, logger); err != nil {
		logger.Error("GeekMode execution failed", err)
		return status.Error(codes.Internal, fmt.Sprintf("GeekMode execution failed: %v", err))
	}

	logger.Info("ai.chat.completed",
		slog.String("mode", "geek"),
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
}

// handleEvolutionMode creates and executes EvolutionParrot for self-evolution.
// handleEvolutionMode 创建并执行 EvolutionParrot 进行自我进化。
// Evolution Mode allows DivineSense to modify its own source code under
// strict safety constraints and with mandatory PR review.
// 进化模式允许 DivineSense 在严格的安全约束下修改自己的源代码，并强制进行 PR 审查。
func (h *ParrotHandler) handleEvolutionMode(
	ctx context.Context,
	req *ChatRequest,
	stream ChatStream,
) error {
	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), "evolution", req.UserID)
	logger.Info("ai.chat.started",
		slog.String("mode", "evolution"),
		slog.String("user_input", req.Message),
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", 0), // History now built by backend (context-engineering.md Phase 1)
	)

	// Send received event for immediate feedback (Task 1.2: Progress Feedback)
	// 发送 received 事件提供即时反馈（200ms 心理阈值）
	if err := stream.Send(&v1pb.ChatResponse{
		EventType: "received",
	}); err != nil {
		slog.Warn("failed to send received event", "error", err)
	}

	// Get source directory (DivineSense root)
	sourceDir, err := h.getSourceDir()
	if err != nil {
		logger.Error("Failed to get source directory", err)
		return status.Error(codes.Internal, "evolution mode requires source directory configuration")
	}

	// Generate a stable session ID based on Conversation ID using UUID v5
	// 统一 Namespace 规则：使用模式名称(evolution)作为前缀并结合 UserID，确保跨用户、跨模式完全隔离
	namespaceBase := fmt.Sprintf("evolution_%d", req.UserID)
	namespace := uuid.NewMD5(uuid.NameSpaceOID, []byte(namespaceBase))
	sessionID := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("conversation_%d", req.ConversationID))).String()

	if h.evoRunner == nil {
		logger.Error("EvoRunner global singleton is null, cannot perform Hot-Multiplexing", nil)
		return status.Error(codes.Unavailable, "EvolutionMode CLI runner not initialized")
	}

	// Create EvolutionParrot (pass store for admin verification, inject global evoRunner)
	evoParrot, err := geek.NewEvolutionParrot(h.evoRunner, sourceDir, req.UserID, sessionID, h.factory.store)
	if err != nil {
		logger.Error("Failed to create EvolutionParrot", err)
		return status.Error(codes.Internal, fmt.Sprintf("failed to create EvolutionParrot: %v", err))
	}

	// Pass device context
	evoParrot.SetDeviceContext(req.DeviceContext)

	logger.Debug("EvolutionParrot created",
		slog.String("agent_name", evoParrot.Name()),
		slog.String("source_dir", sourceDir),
		slog.String("task_id", evoParrot.GetTaskID()),
	)

	// Execute with streaming
	if err := h.executeAgent(ctx, evoParrot, req, stream, logger); err != nil {
		logger.Error("EvolutionMode execution failed", err)
		return status.Error(codes.Internal, fmt.Sprintf("EvolutionMode execution failed: %v", err))
	}

	logger.Info("ai.chat.completed",
		slog.String("mode", "evolution"),
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
}

// executeWithOrchestrator uses Orchestrator for complex/multi-intent requests.
// executeWithOrchestrator 使用 Orchestrator 处理复杂/多意图请求。
func (h *ParrotHandler) executeWithOrchestrator(
	ctx context.Context,
	req *ChatRequest,
	stream ChatStream,
) error {
	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), "orchestrator", req.UserID)
	startTime := time.Now()

	// ========== Phase 1: Build conversation history ==========
	var history []string
	var historyCount int
	if h.contextBuilder != nil && req.ConversationID > 0 {
		sessionID := fmt.Sprintf("conv_%d", req.ConversationID)

		// Get conversation length for dynamic budget adjustment
		historyLength := 0
		if historyLen, err := h.contextBuilder.GetHistoryLength(ctx, sessionID); err == nil {
			historyLength = historyLen
		}

		ctxReq := &ctxpkg.ContextRequest{
			SessionID:     sessionID,
			CurrentQuery:  req.Message,
			AgentType:     "orchestrator",
			UserID:        req.UserID,
			HistoryLength: historyLength,
		}
		builtHistory, err := h.contextBuilder.BuildHistory(ctx, ctxReq)
		if err != nil {
			logger.Error("Failed to build history for orchestrator", err)
			return status.Error(codes.Internal, "failed to build context")
		}
		if builtHistory == nil {
			builtHistory = []string{}
		}
		history = builtHistory
		historyCount = len(history)
	}

	logger.Info("ai.chat.started",
		slog.String("mode", "orchestrator"),
		slog.String("user_input", req.Message),
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", historyCount),
	)

	// ========== Phase 2: Create Block for this chat round ==========
	var currentBlock *store.AIBlock
	var blockID int64
	if h.blockManager != nil && req.ConversationID > 0 && !req.IsTempConversation {
		var createErr error
		currentBlock, createErr = h.blockManager.CreateBlockForChat(
			ctx,
			req.ConversationID,
			req.Message,
			req.AgentType,
			h.determineBlockMode(req),
		)
		if createErr != nil {
			logger.Warn("Failed to create block for orchestrator",
				slog.String("error", createErr.Error()))
		} else if currentBlock != nil {
			blockID = currentBlock.ID
		}
	}

	// Variable to collect AI response content
	var assistantContent strings.Builder
	var assistantContentMu sync.Mutex

	// ========== Phase 2.5: Send block_created event immediately ==========
	// CRITICAL: Send blockId to frontend BEFORE orchestrator starts processing
	// This allows frontend to create optimistic block immediately for instant UI feedback
	// Without this, frontend won't know the blockId until the first orchestrator event
	if blockID > 0 {
		if err := stream.Send(&v1pb.ChatResponse{
			BlockId:   blockID,
			EventType: "block_created",
			EventData: req.Message,
		}); err != nil {
			logger.Warn("Failed to send block_created event", slog.String("error", err.Error()))
		}

		// Early title generation: Start immediately after block creation for parallel execution
		// This runs concurrently with Orchestrator processing, reducing perceived latency
		if h.titleGenerator != nil && req.ConversationID > 0 {
			h.maybeGenerateConversationTitle(ctx, req.ConversationID, req.Message)
		}
	}

	// ========== Phase 3: Inject orchestrator context ==========
	// Pass request-level data through the call chain without modifying function signatures
	orchCtx := &ctxpkg.OrchestratorContext{
		UserID:         req.UserID,
		ConversationID: req.ConversationID,
		BlockID:        blockID,
		History:        history,
		AgentType:      string(req.AgentType),
		SessionID:      fmt.Sprintf("conv_%d", req.ConversationID),
	}
	ctx = ctxpkg.WithOrchestratorContext(ctx, orchCtx)

	// Create callback adapter for streaming events
	// Phase 4 fix: Include BlockId in all orchestrator events for frontend optimistic block creation
	callback := func(eventType string, eventData string) {
		// Parse event data for tool_use and tool_result events
		// Format from expert_registry: {"data": "...", "meta": {...}}
		var finalData string
		var eventMeta *v1pb.EventMetadata

		if (eventType == "tool_use" || eventType == "tool_result") && strings.HasPrefix(eventData, `{"data":`) {
			var parsed struct {
				Data string         `json:"data"`
				Meta map[string]any `json:"meta"`
			}
			if err := json.Unmarshal([]byte(eventData), &parsed); err == nil {
				finalData = parsed.Data
				if parsed.Meta != nil {
					// Extract meta fields
					getString := func(key string) string {
						if v, ok := parsed.Meta[key]; ok {
							if s, ok := v.(string); ok {
								return s
							}
						}
						return ""
					}
					getInt64 := func(key string) int64 {
						if v, ok := parsed.Meta[key]; ok {
							if n, ok := v.(float64); ok {
								return int64(n)
							}
						}
						return 0
					}
					getInt32 := func(key string) int32 {
						if v, ok := parsed.Meta[key]; ok {
							if n, ok := v.(float64); ok {
								return int32(n)
							}
						}
						return 0
					}

					toolName := getString("tool_name")
					if toolName != "" || getString("status") != "" {
						eventMeta = &v1pb.EventMetadata{
							DurationMs:      getInt64("duration_ms"),
							TotalDurationMs: getInt64("total_duration_ms"),
							ToolName:        toolName,
							ToolId:          getString("tool_id"),
							Status:          getString("status"),
							ErrorMsg:        getString("error_msg"),
							InputSummary:    getString("input_summary"),
							OutputSummary:   getString("output_summary"),
							FilePath:        getString("file_path"),
							LineCount:       getInt32("line_count"),
						}
					}
				}
			} else {
				finalData = eventData
			}
		} else {
			finalData = eventData
		}

		// Collect AI response content for block persistence
		// Note: Orchestrator sends "answer" and "aggregation" events (not "content" or "text")
		if eventType == "answer" || eventType == "content" || eventType == "aggregation" {
			assistantContentMu.Lock()
			assistantContent.WriteString(finalData)
			assistantContentMu.Unlock()
		}

		if err := stream.Send(&v1pb.ChatResponse{
			BlockId:   blockID,
			EventType: eventType,
			EventData: finalData,
			EventMeta: eventMeta,
		}); err != nil {
			slog.Warn("failed to send orchestrator event", "error", err, "event_type", eventType)
		}

		// CRITICAL FIX: Persist tool_use and tool_result events to database
		// Without this, frontend cannot display tool call status (always shows "pending")
		if eventType == "tool_use" || eventType == "tool_result" {
			if currentBlock == nil || h.blockManager == nil {
				// Log warning when tool events cannot be persisted - this affects frontend display
				logger.Warn("orchestrator: cannot persist tool event - block or manager unavailable",
					slog.String("event_type", eventType),
					slog.Bool("has_block", currentBlock != nil),
					slog.Bool("has_manager", h.blockManager != nil))
			} else {
				// Build metadata for block event (same as non-orchestrator mode)
				var eventMetaForBlock map[string]any
				if eventMeta != nil {
					eventMetaForBlock = map[string]any{
						"duration_ms":       eventMeta.DurationMs,
						"total_duration_ms": eventMeta.TotalDurationMs,
						"tool_name":         eventMeta.ToolName,
						"tool_id":           eventMeta.ToolId,
						"status":            eventMeta.Status,
						"error_msg":         eventMeta.ErrorMsg,
						"input_tokens":      eventMeta.InputTokens,
						"output_tokens":     eventMeta.OutputTokens,
						"input_summary":     eventMeta.InputSummary,
						"output_summary":    eventMeta.OutputSummary,
						"file_path":         eventMeta.FilePath,
						"line_count":        eventMeta.LineCount,
						// Frontend compatibility fields (extractToolCalls expects these)
						"is_error":  eventMeta.Status == "error",
						"duration":  eventMeta.DurationMs,
						"exit_code": 0,
					}
				}

				// Append event to database
				if err := h.blockManager.AppendEvent(ctx, currentBlock.ID, eventType, finalData, eventMetaForBlock); err != nil {
					logger.Warn("orchestrator: failed to persist event",
						slog.String("event_type", eventType),
						slog.Int64("block_id", currentBlock.ID),
						slog.String("error", err.Error()))
				}
			}
		}
	}

	// Execute Orchestrator
	result, err := h.orchestrator.Process(ctx, req.Message, callback)
	if err != nil {
		logger.Error("Orchestrator execution failed", err)
		return status.Error(codes.Internal, fmt.Sprintf("orchestrator failed: %v", err))
	}

	// Send completion event
	// Phase 4 fix: Include BlockId in done event
	durationMs := time.Since(startTime).Milliseconds()

	// Determine status from errors
	status := "completed"
	if len(result.Errors) > 0 {
		status = "error"
	}

	// Collect tools used from task plan
	var toolsUsed []string
	var toolCallCount int
	if result.Plan != nil && len(result.Plan.Tasks) > 0 {
		seen := make(map[string]bool)
		for _, task := range result.Plan.Tasks {
			if task.Agent != "" && !seen[task.Agent] {
				toolsUsed = append(toolsUsed, task.Agent)
				seen[task.Agent] = true
			}
			// Each task counts as at least one tool execution
			if task.Status == orchestrator.TaskStatusCompleted {
				toolCallCount++
			}
		}
	}

	// ========== Phase 5: Persist block to database BEFORE sending done ==========
	// CRITICAL: Complete block BEFORE sending done marker to prevent race condition
	// This ensures that when frontend receives done=true and refetches blocks,
	// the assistantContent is already persisted in the database.
	// This fixes the "Initializing..." stuck issue.
	if currentBlock != nil && h.blockManager != nil {
		assistantContentMu.Lock()
		finalContent := assistantContent.String()
		assistantContentMu.Unlock()

		// Build session stats from orchestrator result
		blockSessionStats := &store.SessionStats{
			SessionID:        fmt.Sprintf("conv_%d", req.ConversationID),
			UserID:           req.UserID,
			AgentType:        string(req.AgentType),
			TotalDurationMs:  durationMs,
			InputTokens:      int(result.TokenUsage.InputTokens),
			OutputTokens:     int(result.TokenUsage.OutputTokens),
			CacheWriteTokens: int(result.TokenUsage.CacheWriteTokens),
			CacheReadTokens:  int(result.TokenUsage.CacheReadTokens),
			ToolCallCount:    toolCallCount,
		}

		if completeErr := h.blockManager.CompleteBlock(ctx, currentBlock.ID, finalContent, blockSessionStats); completeErr != nil {
			logger.Warn("Failed to complete orchestrator block",
				slog.String("error", completeErr.Error()))
		} else {
			logger.Info("ai.block.completed",
				slog.Int64("block_id", currentBlock.ID),
				slog.Int("content_length", len(finalContent)),
			)
			// Title generation moved to block creation time (Phase 2) for parallel execution
		}
	}

	// Now send the done marker - frontend can safely refetch blocks
	stream.Send(&v1pb.ChatResponse{
		BlockId: blockID,
		Done:    true,
		BlockSummary: &v1pb.BlockSummary{
			TotalDurationMs:       durationMs,
			Status:                status,
			ToolCallCount:         int32(toolCallCount),
			ToolsUsed:             toolsUsed,
			TotalInputTokens:      result.TokenUsage.InputTokens,
			TotalOutputTokens:     result.TokenUsage.OutputTokens,
			TotalCacheWriteTokens: result.TokenUsage.CacheWriteTokens,
			TotalCacheReadTokens:  result.TokenUsage.CacheReadTokens,
		},
	})

	logger.Info("ai.chat.completed",
		slog.String("mode", "orchestrator"),
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
}

// determineBlockMode determines the block mode from the chat request.
// determineBlockMode 根据聊天请求确定 Block 模式。
func (h *ParrotHandler) determineBlockMode(req *ChatRequest) BlockMode {
	if req.EvolutionMode {
		return BlockModeEvolution
	}
	if req.GeekMode {
		return BlockModeGeek
	}
	return BlockModeNormal
}

// getSourceDir returns the DivineSense source code directory.
// getSourceDir 返回 DivineSense 源代码目录。
func (h *ParrotHandler) getSourceDir() (string, error) {
	// Try to get from environment variable first
	if dir := os.Getenv("DIVINESENSE_SOURCE_DIR"); dir != "" {
		return dir, nil
	}

	// Fallback to current working directory
	// This works when running from the project root
	return os.Getwd()
}

// getWorkDirForUser returns the working directory for Claude Code CLI for a specific user.
// getWorkDirForUser 返回特定用户的 Claude Code CLI 工作目录。
// Each user gets an isolated working directory for security and session management.
// 每个用户都有独立的工作目录，用于安全和会话管理。
func (h *ParrotHandler) getWorkDirForUser(userID int32) string {
	// Use persistent directory in user's home to avoid data loss on restart
	// 使用用户主目录下的持久化目录，避免重启时数据丢失
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp" // Fallback if home dir cannot be determined
	}

	return fmt.Sprintf("%s/.divinesense/claude/user_%d", homeDir, userID)
}

// executeAgent executes the agent and streams responses.
func (h *ParrotHandler) executeAgent(
	ctx context.Context,
	agent agentpkg.ParrotAgent,
	req *ChatRequest,
	stream ChatStream,
	logger *observability.RequestContext,
) error {
	// Phase 5: Create Block for this chat round
	// Determine block mode from request
	var blockMode BlockMode
	if req.EvolutionMode {
		blockMode = BlockModeEvolution
	} else if req.GeekMode {
		blockMode = BlockModeGeek
	} else {
		blockMode = BlockModeNormal
	}

	// Only create block for non-temporary conversations with valid ID
	var currentBlock *store.AIBlock
	if h.blockManager != nil && req.ConversationID > 0 && !req.IsTempConversation {
		var createErr error
		currentBlock, createErr = h.blockManager.CreateBlockForChat(
			ctx,
			req.ConversationID,
			req.Message,
			req.AgentType,
			blockMode,
		)
		if createErr != nil {
			logger.Warn("Failed to create block, continuing without block",
				slog.String("error", createErr.Error()),
			)
		} else if currentBlock != nil {
			// Early title generation: Start immediately after block creation for parallel execution
			// This runs concurrently with agent processing, reducing perceived latency
			if h.titleGenerator != nil {
				h.maybeGenerateConversationTitle(ctx, req.ConversationID, req.Message)
			}
		}
		// Note: BlockManager already logs "Created block for chat" with round_number
	}

	// Track events for logging (protected by countMu)
	eventCounts := make(map[string]int)
	var countMu sync.Mutex

	var totalChunks int
	var streamMu sync.Mutex

	// Track session start time for summary
	sessionStartTime := time.Now()
	var sessionTotalDuration int64

	// Track tool calls for session summary
	// Preallocate to avoid frequent reallocation during streaming
	toolsUsed := make([]string, 0, 10)
	var toolMu sync.Mutex

	// Track total cost from session_stats event
	var totalCostUsd float64
	var costMu sync.Mutex

	// Track last event time for heartbeats
	lastEventTime := atomic.Int64{}
	lastEventTime.Store(time.Now().UnixNano())

	// Track assistant content for block completion
	var assistantContent strings.Builder
	var assistantContentMu sync.Mutex

	// Track inability report for handoff mechanism
	// Expert reports what it CANNOT do. Orchestrator finds the appropriate expert via CapabilityMap.
	var inabilityReport struct {
		detected   bool
		capability string
		reason     string
	}
	var inabilityMu sync.Mutex

	// Create stream adapter
	streamAdapter := agentpkg.NewParrotStreamAdapter(func(eventType string, eventData any) error {
		// Update last event time
		lastEventTime.Store(time.Now().UnixNano())

		// Atomically increment event count
		countMu.Lock()
		currentCount := eventCounts[eventType] + 1
		eventCounts[eventType] = currentCount
		countMu.Unlock()

		if eventType == "answer" || eventType == "content" {
			totalChunks++
		}

		// Convert event data to string for streaming
		var dataStr string
		var eventMeta *v1pb.EventMetadata

		// Check if eventData is EventWithMeta (from CCRunner or Agent)
		if eventWithMeta, ok := eventData.(*agentpkg.EventWithMeta); ok {
			dataStr = eventWithMeta.EventData
			if eventWithMeta.Meta != nil {
				eventMeta = &v1pb.EventMetadata{
					DurationMs:      eventWithMeta.Meta.DurationMs,
					TotalDurationMs: eventWithMeta.Meta.TotalDurationMs,
					ToolName:        eventWithMeta.Meta.ToolName,
					ToolId:          eventWithMeta.Meta.ToolID,
					Status:          eventWithMeta.Meta.Status,
					ErrorMsg:        eventWithMeta.Meta.ErrorMsg,
					InputTokens:     eventWithMeta.Meta.InputTokens,
					OutputTokens:    eventWithMeta.Meta.OutputTokens,
					InputSummary:    eventWithMeta.Meta.InputSummary,
					OutputSummary:   eventWithMeta.Meta.OutputSummary,
					FilePath:        eventWithMeta.Meta.FilePath,
					LineCount:       eventWithMeta.Meta.LineCount,
				}

				// Track tools for session summary
				if eventType == "tool_use" && eventWithMeta.Meta.ToolName != "" {
					toolMu.Lock()
					toolsUsed = append(toolsUsed, eventWithMeta.Meta.ToolName)
					toolMu.Unlock()
				}
			}
		} else if eventType == agentpkg.EventTypeSessionStats {
			// Handle session_stats event (from CCRunner result message)
			// Extract and store total cost for final BlockSummary
			if sessionStatsData, ok := eventData.(*agentpkg.SessionStatsData); ok {
				costMu.Lock()
				totalCostUsd = sessionStatsData.TotalCostUSD
				costMu.Unlock()
				logger.Info("ai.session.stats.received",
					slog.Float64("total_cost_usd", sessionStatsData.TotalCostUSD),
					slog.Int("total_tokens", int(sessionStatsData.TotalTokens)),
					slog.Int64("duration_ms", sessionStatsData.TotalDurationMs))

				// Enqueue for async persistence
				if h.persister != nil {
					enqueued := h.persister.EnqueueSessionStatsData(sessionStatsData)
					if !enqueued {
						// Log as error since cost tracking data is lost
						logger.Error("Failed to enqueue session stats - cost tracking will be inaccurate",
							fmt.Errorf("queue full: size=%d", h.persister.QueueSize()),
							slog.String("session_id", sessionStatsData.SessionID),
							slog.Int("queue_size", h.persister.QueueSize()),
							slog.Float64("total_cost_usd", sessionStatsData.TotalCostUSD),
							slog.Int64("total_tokens", int64(sessionStatsData.TotalTokens)))
					}
				}
			}
			// Don't stream session_stats to frontend (it's included in final BlockSummary)
			return nil
		} else {
			// Handle legacy event types (string, error)
			// Also handle JSON format from orchestrator: {"data": "...", "meta": {...}}
			switch v := eventData.(type) {
			case string:
				dataStr = v
				// Try to parse JSON format to extract metadata (for tool_use/tool_result events)
				// This handles events from orchestrator that were converted to JSON format
				if (eventType == "tool_use" || eventType == "tool_result") && strings.HasPrefix(v, "{\"data\":") {
					var parsed struct {
						Data string         `json:"data"`
						Meta map[string]any `json:"meta"`
					}
					if err := json.Unmarshal([]byte(v), &parsed); err == nil && parsed.Data != "" {
						dataStr = parsed.Data
						// Extract metadata from JSON
						if parsed.Meta != nil {
							getString := func(key string) string {
								if val, ok := parsed.Meta[key]; ok {
									if s, ok := val.(string); ok {
										return s
									}
								}
								return ""
							}
							getInt64 := func(key string) int64 {
								if val, ok := parsed.Meta[key]; ok {
									if n, ok := val.(float64); ok {
										return int64(n)
									}
								}
								return 0
							}
							eventMeta = &v1pb.EventMetadata{
								ToolName:      getString("tool_name"),
								ToolId:        getString("tool_id"),
								Status:        getString("status"),
								ErrorMsg:      getString("error_msg"),
								InputSummary:  getString("input_summary"),
								OutputSummary: getString("output_summary"),
								DurationMs:    getInt64("duration_ms"),
							}
						}
					}
				}
			case error:
				dataStr = v.Error()
			default:
				dataStr = fmt.Sprintf("%v", v)
			}
		}

		// Log important events (after data extraction for meaningful content)
		if eventType == "tool_use" || eventType == "tool_result" {
			attrs := []slog.Attr{
				slog.String("event_type", eventType),
				slog.Int("occurrence", currentCount),
			}
			// Add tool-specific fields if available
			if eventMeta != nil {
				if eventMeta.ToolName != "" {
					attrs = append(attrs, slog.String("tool_name", eventMeta.ToolName))
				}
				if eventMeta.Status != "" {
					attrs = append(attrs, slog.String("status", eventMeta.Status))
				}
				if eventMeta.DurationMs > 0 {
					attrs = append(attrs, slog.Int64("duration_ms", eventMeta.DurationMs))
				}
			}
			// Add truncated content if available
			if dataStr != "" {
				attrs = append(attrs, slog.String("content", TruncateString(dataStr, 200)))
			}
			logger.Info("ai.agent.event", attrs...)
		} else {
			logger.Debug("ai.agent.event",
				slog.String("event_type", eventType),
				slog.Int("occurrence", currentCount),
			)
		}

		// Phase 5: Append event to Block (async with error logging)
		if currentBlock != nil && h.blockManager != nil {
			// Build metadata for block event
			var eventMetaForBlock map[string]any
			if eventMeta != nil {
				eventMetaForBlock = map[string]any{
					"duration_ms":       eventMeta.DurationMs,
					"total_duration_ms": eventMeta.TotalDurationMs,
					"tool_name":         eventMeta.ToolName,
					"tool_id":           eventMeta.ToolId,
					"status":            eventMeta.Status,
					"error_msg":         eventMeta.ErrorMsg,
					"input_tokens":      eventMeta.InputTokens,
					"output_tokens":     eventMeta.OutputTokens,
					"input_summary":     eventMeta.InputSummary,
					"output_summary":    eventMeta.OutputSummary,
					"file_path":         eventMeta.FilePath,
					"line_count":        eventMeta.LineCount,
					// Frontend compatibility fields (extractToolCalls expects these)
					"is_error":  eventMeta.Status == "error",
					"duration":  eventMeta.DurationMs,
					"exit_code": 0, // No exit code in EventMetadata, default to 0
				}
			}

			// Debug: log eventMetaForBlock for tool_use/tool_result
			if eventType == "tool_use" || eventType == "tool_result" {
				if eventMetaForBlock == nil {
					logger.Warn("tool event without metadata",
						slog.String("event_type", eventType),
						slog.String("event_data_type", fmt.Sprintf("%T", eventData)),
					)
				} else {
					toolName := ""
					if v, ok := eventMetaForBlock["tool_name"].(string); ok {
						toolName = v
					}
					inputSummary := ""
					if v, ok := eventMetaForBlock["input_summary"].(string); ok {
						inputSummary = v
						if len(inputSummary) > 100 {
							inputSummary = inputSummary[:100] + "..."
						}
					}
					logger.Debug("ai.block.event_metadata",
						slog.String("event_type", eventType),
						slog.String("tool_name", toolName),
						slog.String("input_summary", inputSummary),
					)
				}
			}

			// Append event synchronously (non-blocking because AppendEvent internally queues)
			// This ensures events are persisted in order by the BlockManager's serializer
			if err := h.blockManager.AppendEvent(ctx, currentBlock.ID, eventType, dataStr, eventMetaForBlock); err != nil {
				logger.Warn("Failed to enqueue event for persistence",
					slog.String("metric", "ai.event_persistence_failure"), // Structured attribute for monitoring
					slog.Int64("block_id", currentBlock.ID),
					slog.String("event_type", eventType),
					slog.String("error", err.Error()))
			}

			// Detect INABILITY_REPORTED for handoff mechanism
			// Expert reports what it CANNOT do. Orchestrator finds the appropriate expert via CapabilityMap.
			if eventType == "tool_result" && strings.Contains(dataStr, "INABILITY_REPORTED:") {
				inabilityMu.Lock()
				inabilityReport.detected = true
				// Parse the inability report: "INABILITY_REPORTED: <capability> - <reason>"
				inabilityReport.capability, inabilityReport.reason = agentpkg.ParseInabilityReport(dataStr)
				cap := inabilityReport.capability
				reason := inabilityReport.reason
				inabilityMu.Unlock()
				logger.Info("handoff: inability reported by expert",
					slog.String("capability", cap),
					slog.String("reason", reason))
			}

			// Collect assistant content for block completion
			if eventType == "answer" || eventType == "content" {
				assistantContentMu.Lock()
				assistantContent.WriteString(dataStr)
				assistantContentMu.Unlock()
			}
		}

		// Thread-safe send
		streamMu.Lock()
		defer streamMu.Unlock()

		// Phase 4: Include BlockId in all streaming events
		var blockId int64
		if currentBlock != nil {
			blockId = currentBlock.ID
		}

		return stream.Send(&v1pb.ChatResponse{
			EventType: eventType,
			EventData: dataStr,
			EventMeta: eventMeta,
			BlockId:   blockId,
		})
	})

	// Create callback wrapper
	callback := func(eventType string, eventData any) error {
		return streamAdapter.Send(eventType, eventData)
	}

	// Start Heartbeat Goroutine
	// Sends a "thinking" event every 5 seconds if no other events occur.
	// This prevents load balancers and clients from closing the connection due to timeout.
	heartbeatDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Check time since last activity
				lastTime := time.Unix(0, lastEventTime.Load())
				if time.Since(lastTime) > 5*time.Second {
					// Send heartbeat
					streamMu.Lock()
					// Phase 4: Include BlockId in heartbeat
					var blockId int64
					if currentBlock != nil {
						blockId = currentBlock.ID
					}
					// Just send a lightweight ping chunk
					err := stream.Send(&v1pb.ChatResponse{
						EventType: "ping",
						EventData: ".", // Minimal data
						BlockId:   blockId,
					})
					streamMu.Unlock()
					// If send fails, client disconnected - stop heartbeat early
					if err != nil {
						logger.Debug("Heartbeat send failed, stopping", slog.String("error", err.Error()))
						return
					}
				}
			}
		}
	}()

	// Execute agent
	defer close(heartbeatDone) // Ensure heartbeat stops even on panic

	// Backend-driven context: use contextBuilder to build history
	// No longer accept req.History from frontend (Backend as Source of Truth)
	// This implements context-engineering.md Phase 1
	// Issue #211: Phase 3 - Get conversation length for dynamic budget adjustment
	//
	// OPTIMIZATION: GeekMode and EvolutionMode bypass context building entirely.
	// They execute Claude Code CLI directly without LLM-based conversation history.
	var history []string
	if req.GeekMode || req.EvolutionMode {
		// Geek/Evolution mode: no context needed, direct CLI execution
		history = []string{}
		logger.Debug("Skipping context build for direct CLI mode",
			slog.Bool("geek_mode", req.GeekMode),
			slog.Bool("evolution_mode", req.EvolutionMode))
	} else if h.contextBuilder != nil && req.ConversationID > 0 {
		sessionID := fmt.Sprintf("conv_%d", req.ConversationID)

		// Get conversation length for dynamic budget adjustment (Issue #211: Phase 3)
		historyLength := 0
		if historyLen, err := h.contextBuilder.GetHistoryLength(ctx, sessionID); err == nil {
			historyLength = historyLen
		}

		ctxReq := &ctxpkg.ContextRequest{
			SessionID:     sessionID,
			CurrentQuery:  req.Message,
			AgentType:     req.AgentType.String(),
			UserID:        req.UserID,
			HistoryLength: historyLength,
		}
		builtHistory, err := h.contextBuilder.BuildHistory(ctx, ctxReq)
		if err != nil {
			logger.Error("Failed to build history from context engine", err)
			// Return error instead of falling back to req.History
			return err
		}
		// Handle nil history gracefully
		if builtHistory == nil {
			logger.Warn("BuildHistory returned nil, using empty history")
			builtHistory = []string{}
		}
		history = builtHistory
		logger.Debug("Using backend-driven context",
			slog.Int("history_count", len(history)),
			slog.Int("history_length", historyLength),
			slog.String("source", "context_builder"))
	} else if req.ConversationID <= 0 {
		// Temporary conversation: use empty history
		history = []string{}
		logger.Debug("Using empty history for temp conversation")
	} else {
		// contextBuilder not initialized: this should not happen in production
		err := fmt.Errorf("context builder not initialized for conversation %d", req.ConversationID)
		logger.Error("Context builder not initialized", err)
		return err
	}

	execErr := agent.Execute(ctx, req.Message, history, callback)
	logger.Info("ai.agent.completed",
		slog.String("execErr", fmt.Sprintf("%v", execErr)),
		slog.Int64("duration_ms", time.Since(sessionStartTime).Milliseconds()))
	if execErr != nil {
		logger.Error("Agent execution failed", execErr)
		// Don't return here, continue to send session summary
	}

	// Handle handoff if inability was reported
	// Expert reports what it CANNOT do. Orchestrator uses CapabilityMap to find the appropriate expert.
	inabilityMu.Lock()
	shouldHandoff := inabilityReport.detected
	handoffCapability := inabilityReport.capability
	handoffReason := inabilityReport.reason
	inabilityMu.Unlock()

	if shouldHandoff && handoffCapability != "" && h.capabilityMap != nil {
		// Use CapabilityMap to find alternative experts that can handle the missing capability
		alternatives := h.capabilityMap.FindAlternativeExperts(handoffCapability, agent.Name())

		if len(alternatives) == 0 {
			logger.Warn("handoff: no alternative expert found for capability",
				slog.String("capability", handoffCapability),
				slog.String("from_agent", agent.Name()))
		} else {
			// Use the first alternative expert
			targetExpert := alternatives[0]
			handoffAgent := targetExpert.Name

			logger.Info("handoff: executing handoff to alternative expert",
				slog.String("from_agent", agent.Name()),
				slog.String("to_agent", handoffAgent),
				slog.String("capability", handoffCapability),
				slog.String("reason", handoffReason))

			// Send handoff event to frontend
			streamMu.Lock()
			var blockIdForEvent int64
			if currentBlock != nil {
				blockIdForEvent = currentBlock.ID
			}
			// Use json.Marshal to properly escape special characters
			handoffStartData, _ := json.Marshal(map[string]string{
				"from":       agent.Name(),
				"to":         handoffAgent,
				"capability": handoffCapability,
				"reason":     handoffReason,
			})
			stream.Send(&v1pb.ChatResponse{
				EventType: "handoff_start",
				EventData: string(handoffStartData),
				BlockId:   blockIdForEvent,
			})
			streamMu.Unlock()

			// Map expert name to AgentType (internal system convention)
			var handoffAgentType AgentType
			switch handoffAgent {
			case "schedule":
				handoffAgentType = AgentTypeSchedule
			case "memo":
				handoffAgentType = AgentTypeMemo
			default:
				// Unknown agent type
				logger.Warn("handoff: unknown agent type",
					slog.String("agent", handoffAgent))
				handoffAgentType = AgentTypeMemo // Default fallback
			}

			// Create the handoff expert using factory
			handoffExpert, handoffCreateErr := h.factory.Create(ctx, &CreateConfig{
				Type:     handoffAgentType,
				UserID:   req.UserID,
				Timezone: req.Timezone,
			})

			if handoffCreateErr != nil {
				logger.Error("handoff: failed to create expert", handoffCreateErr)
				failData, _ := json.Marshal(map[string]string{"error": "failed to create expert: " + handoffCreateErr.Error()})
				streamMu.Lock()
				stream.Send(&v1pb.ChatResponse{
					EventType: "handoff_fail",
					EventData: string(failData),
					BlockId:   blockIdForEvent,
				})
				streamMu.Unlock()
			} else {
				// Create callback for handoff execution
				handoffCallback := func(eventType string, eventData any) error {
					streamMu.Lock()
					var blockId int64
					if currentBlock != nil {
						blockId = currentBlock.ID
					}

					// Convert eventData to string
					var dataStr string
					switch v := eventData.(type) {
					case string:
						dataStr = v
					case []byte:
						dataStr = string(v)
					default:
						dataStr = fmt.Sprintf("%v", v)
					}

					stream.Send(&v1pb.ChatResponse{
						EventType: eventType,
						EventData: dataStr,
						BlockId:   blockId,
					})
					streamMu.Unlock()

					// Collect handoff content
					if eventType == "answer" || eventType == "content" {
						assistantContentMu.Lock()
						assistantContent.WriteString(dataStr)
						assistantContentMu.Unlock()
					}
					return nil
				}

				// Execute handoff expert
				handoffErr := handoffExpert.Execute(ctx, req.Message, history, handoffCallback)
				if handoffErr != nil {
					logger.Error("handoff: execution failed", handoffErr)
					execFailData, _ := json.Marshal(map[string]string{"error": handoffErr.Error()})
					streamMu.Lock()
					stream.Send(&v1pb.ChatResponse{
						EventType: "handoff_fail",
						EventData: string(execFailData),
						BlockId:   blockIdForEvent,
					})
					streamMu.Unlock()
				} else {
					logger.Info("handoff: completed successfully",
						slog.String("to_agent", handoffAgent))

					// Send handoff success event
					endData, _ := json.Marshal(map[string]string{"to": handoffAgent})
					streamMu.Lock()
					stream.Send(&v1pb.ChatResponse{
						EventType: "handoff_end",
						EventData: string(endData),
						BlockId:   blockIdForEvent,
					})
					streamMu.Unlock()
				}
			}
		}
	}

	// Prepare session summary

	// Calculate session summary
	sessionTotalDuration = time.Since(sessionStartTime).Milliseconds()
	logger.Info("ai.session.summary.preparing",
		slog.Int64("duration_ms", sessionTotalDuration))

	// Get session stats from agent via ParrotAgent interface
	// All parrot agents now implement GetSessionStats() returning *agentpkg.NormalSessionStats
	var normalStats *agentpkg.NormalSessionStats
	if agent != nil {
		normalStats = agent.GetSessionStats()
		if normalStats != nil {
			// For tool-based agents, token stats may be zero - log tool metrics instead
			if normalStats.PromptTokens == 0 && normalStats.CompletionTokens == 0 {
				logger.Info("ai.agent.stats.tool_based",
					slog.Int("tool_calls", normalStats.ToolCallCount),
					slog.Int64("duration_ms", normalStats.TotalDurationMs))
			} else {
				logger.Info("ai.agent.stats.normal",
					slog.Int("prompt_tokens", normalStats.PromptTokens),
					slog.Int("completion_tokens", normalStats.CompletionTokens),
					slog.Int64("duration_ms", normalStats.TotalDurationMs))
			}
		} else {
			logger.Info("ai.agent.stats.unavailable")
		}
	}

	// Safely get tool usage stats
	toolMu.Lock()
	finalToolCallCount := int32(len(toolsUsed))
	finalToolsUsed := make([]string, len(toolsUsed))
	copy(finalToolsUsed, toolsUsed)
	toolMu.Unlock()

	// Determine status
	status := "success"
	if execErr != nil {
		status = "error"
	}

	// Build block summary with available data
	blockSummary := &v1pb.BlockSummary{
		TotalDurationMs: sessionTotalDuration,
		Status:          status,
		ToolCallCount:   finalToolCallCount,
		ToolsUsed:       finalToolsUsed,
		TotalCostUsd:    totalCostUsd,
	}

	// Set SessionId - use conversation ID as default
	// Note: Only Geek/Evolution modes have real UUID session IDs
	blockSummary.SessionId = fmt.Sprintf("conv_%d", req.ConversationID)

	// NOTE: BlockSummary.Mode has been removed - Block.mode is the single source of truth.
	// The mode is stored in the Block (currentBlock.mode) and should be read from there.

	// Add stats from normalStats (all parrot agents now return NormalSessionStats)
	if normalStats != nil {
		// P1-A006: Include NormalSessionStats in BlockSummary for normal mode agents
		statsSnapshot := normalStats.GetStatsSnapshot()
		blockSummary.TotalDurationMs = statsSnapshot.TotalDurationMs
		blockSummary.ThinkingDurationMs = statsSnapshot.ThinkingDurationMs
		blockSummary.GenerationDurationMs = statsSnapshot.GenerationDurationMs
		blockSummary.TotalInputTokens = int32(statsSnapshot.PromptTokens)
		blockSummary.TotalOutputTokens = int32(statsSnapshot.CompletionTokens)
		blockSummary.TotalCacheWriteTokens = int32(statsSnapshot.CacheWriteTokens)
		blockSummary.TotalCacheReadTokens = int32(statsSnapshot.CacheReadTokens)
		blockSummary.ToolCallCount = int32(statsSnapshot.ToolCallCount)
		if len(statsSnapshot.ToolsUsed) > 0 {
			blockSummary.ToolsUsed = statsSnapshot.ToolsUsed
		}
		if len(statsSnapshot.FilePaths) > 0 {
			blockSummary.FilePaths = statsSnapshot.FilePaths
			blockSummary.FilesModified = int32(len(statsSnapshot.FilePaths))
		}
		// Convert milli-cents to USD (1 USD = 100000 milli-cents)
		if statsSnapshot.TotalCostMilliCents > 0 {
			blockSummary.TotalCostUsd = float64(statsSnapshot.TotalCostMilliCents) / 100000
		}
		// Log meaningful stats based on agent type
		if statsSnapshot.PromptTokens == 0 && statsSnapshot.CompletionTokens == 0 {
			logger.Info("ai.agent.completed",
				slog.Int("tool_calls", statsSnapshot.ToolCallCount),
				slog.String("tools", formatToolsList(statsSnapshot.ToolsUsed)),
				slog.Int64("duration_ms", statsSnapshot.TotalDurationMs))
		} else {
			logger.Info("ai.block.summary.applied",
				slog.Int("prompt_tokens", statsSnapshot.PromptTokens),
				slog.Int("completion_tokens", statsSnapshot.CompletionTokens),
				slog.Int64("duration_ms", statsSnapshot.TotalDurationMs),
				slog.Int64("cost_milli_cents", statsSnapshot.TotalCostMilliCents),
				slog.Float64("cost_usd", blockSummary.TotalCostUsd),
			)
		}
	}

	// Phase 5: Complete or mark error on Block BEFORE sending done marker
	// This ensures that when the frontend calls refetchBlocks() after receiving the done event,
	// the Block's assistantContent is already persisted in the database.
	// This fixes the "Initializing..." stuck issue caused by the race condition where
	// refetchBlocks() executes before CompleteBlock() completes.
	if currentBlock != nil && h.blockManager != nil {
		assistantContentMu.Lock()
		finalContent := assistantContent.String()
		assistantContentMu.Unlock()

		// Convert BlockSummary to store.SessionStats
		// blockSummary is always non-nil (created on line 604)
		blockSessionStats := &store.SessionStats{
			SessionID:            blockSummary.SessionId,
			UserID:               req.UserID,
			AgentType:            string(req.AgentType),
			TotalDurationMs:      blockSummary.TotalDurationMs,
			ThinkingDurationMs:   blockSummary.ThinkingDurationMs,
			ToolDurationMs:       blockSummary.ToolDurationMs,
			GenerationDurationMs: blockSummary.GenerationDurationMs,
			InputTokens:          int(blockSummary.TotalInputTokens),
			OutputTokens:         int(blockSummary.TotalOutputTokens),
			CacheWriteTokens:     int(blockSummary.TotalCacheWriteTokens),
			CacheReadTokens:      int(blockSummary.TotalCacheReadTokens),
			TotalCostUsd:         blockSummary.TotalCostUsd,
			ToolCallCount:        int(blockSummary.ToolCallCount),
			ToolsUsed:            blockSummary.ToolsUsed,
			FilesModified:        int(blockSummary.FilesModified),
			FilePaths:            blockSummary.FilePaths,
		}

		if execErr != nil {
			// Mark block as error
			if markErr := h.blockManager.MarkBlockError(ctx, currentBlock.ID, execErr.Error()); markErr != nil {
				logger.Warn("Failed to mark block as error",
					slog.Int64("block_id", currentBlock.ID),
					slog.String("error", markErr.Error()),
				)
			}
		} else {
			// Complete block successfully
			if completeErr := h.blockManager.CompleteBlock(ctx, currentBlock.ID, finalContent, blockSessionStats); completeErr != nil {
				logger.Warn("Failed to complete block",
					slog.Int64("block_id", currentBlock.ID),
					slog.String("error", completeErr.Error()),
				)
			} else {
				logger.Info("ai.block.completed",
					slog.Int64("block_id", currentBlock.ID),
					slog.Int("content_length", len(finalContent)),
				)

				// Phase 3: Async episodic memory generation
				// Trigger memory generation after successful block completion
				if h.memoryGenerator != nil && len(currentBlock.UserInputs) > 0 {
					h.memoryGenerator.GenerateAsync(ctx, memory.MemoryRequest{
						BlockID:   currentBlock.ID,
						UserID:    req.UserID,
						AgentType: string(req.AgentType),
						UserInput: currentBlock.UserInputs[0].Content,
						Outcome:   finalContent,
					})
				}

				// Context Engineering: Persist routing metadata for sticky routing
				if h.metadataMgr != nil && req.RouteResult != nil && currentBlock.ConversationID > 0 {
					if err := h.metadataMgr.SetCurrentAgent(
						ctx,
						currentBlock.ConversationID,
						currentBlock.ID,
						req.RouteResult.Route,
						agentpkg.ExtractIntent(agentpkg.ChatRouteType(req.RouteResult.Route)),
						float32(req.RouteResult.Confidence),
					); err != nil {
						logger.Warn("Failed to persist routing metadata",
							slog.String("error", err.Error()),
						)
					}
				}
				// Title generation moved to block creation time for parallel execution
			}
		}
	}

	// Safely send done marker AFTER Block is completed
	streamMu.Lock()
	logger.Info("ai.block.summary.sending",
		slog.String("session_id", blockSummary.SessionId),
		slog.Int64("duration_ms", blockSummary.TotalDurationMs),
		slog.Int("input_tokens", int(blockSummary.TotalInputTokens)),
		slog.Int("output_tokens", int(blockSummary.TotalOutputTokens)),
		slog.Int64("tool_calls", int64(blockSummary.ToolCallCount)),
		slog.Float64("cost_usd", blockSummary.TotalCostUsd),
	)
	// Phase 4: Include BlockId in done marker
	var blockId int64
	if currentBlock != nil {
		blockId = currentBlock.ID
	}
	sendErr := stream.Send(&v1pb.ChatResponse{
		Done:         true,
		BlockSummary: blockSummary,
		BlockId:      blockId,
	})
	streamMu.Unlock()

	if sendErr != nil {
		logger.Error("ai.block.summary.send_failed", sendErr,
			slog.String("error", sendErr.Error()))
	}

	if sendErr != nil {
		// If send fails, return the error (prefer execErr if it exists)
		if execErr != nil {
			return execErr
		}
		return sendErr
	}

	// Safely get unique event count
	countMu.Lock()
	uniqueEventTokenCount := len(eventCounts)
	countMu.Unlock()

	logger.Debug("ai.agent.execution_completed",
		slog.Int("total_chunks", totalChunks),
		slog.Int("unique_events", uniqueEventTokenCount),
		slog.Int64("duration_ms", sessionTotalDuration),
		slog.Int("tool_calls", int(finalToolCallCount)),
		slog.Any("error", execErr),
	)

	return execErr
}

// RoutingHandler routes all agent requests through the parrot handler.
// All agent types (including DEFAULT) are now implemented as standard parrots.
type RoutingHandler struct {
	parrotHandler *ParrotHandler
}

// NewRoutingHandler creates a new routing handler.
func NewRoutingHandler(parrot *ParrotHandler) *RoutingHandler {
	return &RoutingHandler{
		parrotHandler: parrot,
	}
}

// Handle implements Handler interface by routing to the appropriate handler.
func (h *RoutingHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	// All agent types (including DEFAULT) now use parrot handler
	// DEFAULT parrot (羽飞/Navi) is implemented as a standard parrot with pure LLM mode
	return h.parrotHandler.Handle(ctx, req, stream)
}

// Close gracefully shuts down the underlying ParrotHandler and its singletons.
func (h *RoutingHandler) Close() error {
	if h.parrotHandler != nil {
		return h.parrotHandler.Close()
	}
	return nil
}

// ToChatRequest converts a protobuf request to an internal ChatRequest.
// Note: History field removed - backend-driven context construction
func ToChatRequest(pbReq *v1pb.ChatRequest) *ChatRequest {
	return &ChatRequest{
		Message:            pbReq.Message,
		AgentType:          AgentTypeFromProto(pbReq.AgentType),
		Timezone:           pbReq.UserTimezone,
		ConversationID:     pbReq.ConversationId,
		IsTempConversation: pbReq.IsTempConversation,
		GeekMode:           pbReq.GeekMode,
		EvolutionMode:      pbReq.EvolutionMode,
		DeviceContext:      pbReq.DeviceContext,
	}
}

// HandleError converts an error to an appropriate gRPC status error.
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a gRPC status error, return as-is
	if _, ok := status.FromError(err); ok {
		return err
	}

	// If it's an AIError, convert it
	if aiErr, ok := err.(*errors.AIError); ok {
		return FromAIError(aiErr)
	}

	// Default to internal error
	return status.Error(codes.Internal, err.Error())
}

// NewChatRouter creates a new chat router for auto-routing based on intent classification.
// routerSvc is required and provides two-layer routing (cache → rule).
func NewChatRouter(routerSvc *routing.Service) *agentpkg.ChatRouter {
	return agentpkg.NewChatRouter(routerSvc)
}

// formatToolsList formats a list of tool names for logging.
// Example: ["schedule_query", "schedule_add"] → "schedule_query, schedule_add"
func formatToolsList(tools []string) string {
	if len(tools) == 0 {
		return "none"
	}
	return strings.Join(tools, ", ")
}
