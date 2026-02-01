package ai

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/plugin/ai"
	agentpkg "github.com/hrygo/divinesense/plugin/ai/agent"
	"github.com/hrygo/divinesense/plugin/ai/router"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/internal/errors"
	"github.com/hrygo/divinesense/server/internal/observability"
)

// ChatStream represents the streaming response interface for AI chat.
type ChatStream interface {
	Send(*v1pb.ChatResponse) error
	Context() context.Context
}

// ParrotHandler handles all parrot agent requests (DEFAULT, MEMO, SCHEDULE, AMAZING, CREATIVE).
type ParrotHandler struct {
	factory    *AgentFactory
	llm        ai.LLMService
	chatRouter *agentpkg.ChatRouter
}

// NewParrotHandler creates a new parrot handler.
func NewParrotHandler(factory *AgentFactory, llm ai.LLMService) *ParrotHandler {
	return &ParrotHandler{
		factory: factory,
		llm:     llm,
	}
}

// SetChatRouter configures the intelligent chat router for auto-routing.
func (h *ParrotHandler) SetChatRouter(router *agentpkg.ChatRouter) {
	h.chatRouter = router
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

	if h.llm == nil {
		return status.Error(codes.Unavailable, "LLM service is not available")
	}

	// Auto-route if AgentType is AUTO
	agentType := req.AgentType
	if agentType == AgentTypeAuto && h.chatRouter != nil {
		// Add user ID to context for history matching.
		// Note: req.UserID is already authenticated by the gRPC interceptor middleware.
		ctx = router.WithUserID(ctx, req.UserID)
		routeResult, err := h.chatRouter.Route(ctx, req.Message)
		if err != nil {
			slog.Warn("chat router failed, defaulting to amazing",
				"error", err,
				"message", req.Message[:min(len(req.Message), 30)])
			agentType = AgentTypeAmazing
		} else {
			// Map ChatRouteType to AgentType
			switch routeResult.Route {
			case agentpkg.RouteTypeMemo:
				agentType = AgentTypeMemo
			case agentpkg.RouteTypeSchedule:
				agentType = AgentTypeSchedule
			default:
				agentType = AgentTypeAmazing
			}
			slog.Info("chat auto-routed",
				"route", routeResult.Route,
				"method", routeResult.Method,
				"confidence", routeResult.Confidence)
		}
	} else if agentType == AgentTypeAuto {
		// No router configured, fallback to amazing
		agentType = AgentTypeAmazing
	}

	// Create logger for this request
	logger := observability.NewRequestContext(slog.Default(), agentType.String(), req.UserID)
	logger.Info("AI chat started (parrot agent)",
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", len(req.History)),
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

	logger.Info("AI chat completed",
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
	logger.Info("AI chat started (Geek Mode - direct Claude Code)",
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", len(req.History)),
	)

	// Generate a stable session ID based on Conversation ID using UUID v5
	// Using a fixed namespace ensures the same conversation ID always generates the same UUID
	// 使用固定的命名空间确保相同的 Conversation ID 总是生成相同的 UUID
	namespace := uuid.MustParse("00000000-0000-0000-0000-000000000000") // Null UUID as namespace
	sessionID := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("conversation_%d", req.ConversationID))).String()

	// Create GeekParrot directly (no factory needed, no LLM dependency)
	// 直接创建 GeekParrot（无需工厂，无 LLM 依赖）
	geekParrot, err := agentpkg.NewGeekParrot(
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

	logger.Info("AI chat completed (Geek Mode)",
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
	logger.Info("AI chat started (Evolution Mode - self-evolution)",
		slog.Int(observability.LogFieldMessageLen, len(req.Message)),
		slog.Int("history_count", len(req.History)),
	)

	// Get source directory (DivineSense root)
	sourceDir, err := h.getSourceDir()
	if err != nil {
		logger.Error("Failed to get source directory", err)
		return status.Error(codes.Internal, "evolution mode requires source directory configuration")
	}

	// Generate session ID for evolution (must be valid UUID for Claude Code CLI)
	// Using user-specific namespace to isolate Evolution sessions from Geek sessions
	// 使用用户特定的命名空间隔离 Evolution 和 Geek 会话
	// Format: 00000000-0000-0000-0000-<user_id_padded_to_12_hex>
	namespace := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", req.UserID))
	sessionID := uuid.NewSHA1(namespace, []byte(fmt.Sprintf("evolution_%d", req.ConversationID))).String()

	// Create EvolutionParrot (pass store for admin verification)
	evoParrot, err := agentpkg.NewEvolutionParrot(sourceDir, req.UserID, sessionID, h.factory.store)
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

	logger.Info("AI chat completed (Evolution Mode)",
		slog.Int64(observability.LogFieldDuration, logger.DurationMs()),
	)

	return nil
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
	// Track events for logging (protected by countMu)
	eventCounts := make(map[string]int)
	var countMu sync.Mutex

	var totalChunks int
	var streamMu sync.Mutex

	// Track session start time for summary
	sessionStartTime := time.Now()
	var sessionTotalDuration int64

	// Track tool calls for session summary
	var toolsUsed []string
	var toolMu sync.Mutex

	// Track last event time for heartbeats
	lastEventTime := atomic.Int64{}
	lastEventTime.Store(time.Now().UnixNano())

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

		// Log important events
		if eventType == "tool_use" || eventType == "tool_result" {
			logger.Info("Agent event", // Use Info level for visibility
				slog.String(observability.LogFieldEventType, eventType),
				slog.String("event_data", TruncateString(fmt.Sprintf("%v", eventData), 100)),
				slog.Int("occurrence", currentCount),
			)
		} else {
			logger.Debug("Agent event",
				slog.String(observability.LogFieldEventType, eventType),
				slog.Int("occurrence", currentCount),
			)
		}

		// Convert event data to string for streaming
		var dataStr string
		var eventMeta *v1pb.EventMetadata

		// Check if eventData is EventWithMeta (from CCRunner)
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
		} else {
			// Handle legacy event types (string, error)
			switch v := eventData.(type) {
			case string:
				dataStr = v
			case error:
				dataStr = v.Error()
			default:
				dataStr = fmt.Sprintf("%v", v)
			}
		}

		// Thread-safe send
		streamMu.Lock()
		defer streamMu.Unlock()

		return stream.Send(&v1pb.ChatResponse{
			EventType: eventType,
			EventData: dataStr,
			EventMeta: eventMeta,
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
					// Just send a lightweight ping chunk
					_ = stream.Send(&v1pb.ChatResponse{
						EventType: "ping",
						EventData: ".", // Minimal data
					})
					streamMu.Unlock()
				}
			}
		}
	}()

	// Execute agent
	execErr := agent.ExecuteWithCallback(ctx, req.Message, req.History, callback)
	close(heartbeatDone) // Stop heartbeat immediately after execution finishes
	if execErr != nil {
		logger.Error("Agent execution failed", execErr)
		// Don't return here, continue to send session summary
	}

	// Prepare session summary

	// Calculate session summary
	sessionTotalDuration = time.Since(sessionStartTime).Milliseconds()

	// Try to get detailed stats from agent if available (GeekParrot/EvolutionParrot)
	// 尝试从 agent 获取详细统计数据（如果可用，如 GeekParrot/EvolutionParrot）
	var detailedStats *agentpkg.SessionStats
	if statsProvider, ok := agent.(agentpkg.SessionStatsProvider); ok {
		detailedStats = statsProvider.GetSessionStats()
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

	// Build session summary with available data
	// 使用可用数据构建会话摘要
	sessionSummary := &v1pb.SessionSummary{
		SessionId:       fmt.Sprintf("conv_%d", req.ConversationID),
		TotalDurationMs: sessionTotalDuration,
		Status:          status,
		ToolCallCount:   finalToolCallCount,
		ToolsUsed:       finalToolsUsed,
	}

	// Add detailed stats if available (from GeekParrot/EvolutionParrot)
	// 添加详细统计数据（如果可用，来自 GeekParrot/EvolutionParrot）
	if detailedStats != nil {
		sessionSummary.TotalDurationMs = detailedStats.TotalDurationMs
		sessionSummary.ThinkingDurationMs = detailedStats.ThinkingDurationMs
		sessionSummary.ToolDurationMs = detailedStats.ToolDurationMs
		sessionSummary.GenerationDurationMs = detailedStats.GenerationDurationMs
		sessionSummary.TotalInputTokens = detailedStats.InputTokens
		sessionSummary.TotalOutputTokens = detailedStats.OutputTokens
		sessionSummary.TotalCacheWriteTokens = detailedStats.CacheWriteTokens
		sessionSummary.TotalCacheReadTokens = detailedStats.CacheReadTokens
		sessionSummary.ToolCallCount = detailedStats.ToolCallCount
		if len(detailedStats.ToolsUsed) > 0 {
			tools := make([]string, 0, len(detailedStats.ToolsUsed))
			for tool := range detailedStats.ToolsUsed {
				tools = append(tools, tool)
			}
			sessionSummary.ToolsUsed = tools
		}
		sessionSummary.FilesModified = detailedStats.FilesModified
		sessionSummary.FilePaths = detailedStats.FilePaths
	}

	// Safely send done marker
	streamMu.Lock()
	sendErr := stream.Send(&v1pb.ChatResponse{
		Done:           true,
		SessionSummary: sessionSummary,
	})
	streamMu.Unlock()

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

	logger.Debug("Agent execution completed",
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

// ToChatRequest converts a protobuf request to an internal ChatRequest.
func ToChatRequest(pbReq *v1pb.ChatRequest) *ChatRequest {
	return &ChatRequest{
		Message:            pbReq.Message,
		History:            pbReq.History,
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
// Optionally accepts a router.Service for enhanced three-layer routing.
func NewChatRouter(cfg *ai.IntentClassifierConfig, routerSvc *router.Service) *agentpkg.ChatRouter {
	return agentpkg.NewChatRouter(agentpkg.ChatRouterConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	}, routerSvc)
}
