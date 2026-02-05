package ai

import (
	"context"
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
	agentpkg "github.com/hrygo/divinesense/ai/agent"
	"github.com/hrygo/divinesense/ai/router"
	aistats "github.com/hrygo/divinesense/ai/stats"
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

// ParrotHandler handles all parrot agent requests (DEFAULT, MEMO, SCHEDULE, AMAZING, CREATIVE).
type ParrotHandler struct {
	factory      *AgentFactory
	llm          ai.LLMService
	chatRouter   *agentpkg.ChatRouter
	persister    *aistats.Persister // session stats persister
	blockManager *BlockManager      // Phase 5: Unified Block Model support
}

// NewParrotHandler creates a new parrot handler.
func NewParrotHandler(factory *AgentFactory, llm ai.LLMService, persister *aistats.Persister, blockManager *BlockManager) *ParrotHandler {
	return &ParrotHandler{
		factory:      factory,
		llm:          llm,
		persister:    persister,
		blockManager: blockManager, // Phase 5
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
		} else {
			logger.Info("Created block for chat round",
				slog.Int64("block_id", currentBlock.ID),
				slog.Int64("conversation_id", int64(req.ConversationID)),
			)
		}
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
		} else if eventType == agentpkg.EventTypeSessionStats {
			// Handle session_stats event (from CCRunner result message)
			// Extract and store total cost for final BlockSummary
			if sessionStatsData, ok := eventData.(*agentpkg.SessionStatsData); ok {
				costMu.Lock()
				totalCostUsd = sessionStatsData.TotalCostUSD
				costMu.Unlock()
				logger.Info("Session stats received",
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
			switch v := eventData.(type) {
			case string:
				dataStr = v
			case error:
				dataStr = v.Error()
			default:
				dataStr = fmt.Sprintf("%v", v)
			}
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
				}
			}

			// Append event asynchronously with error logging (don't block streaming)
			// Note: Persistence failures are logged with structured "metric" attribute for monitoring/alerting
			go func(blockID int64, evtType string) {
				// Use WithoutCancel to detach from request context - allows persistence to complete
				// even if the request is cancelled or the client disconnects
				bgCtx := context.WithoutCancel(ctx)
				if err := h.blockManager.AppendEvent(bgCtx, blockID, evtType, dataStr, eventMetaForBlock); err != nil {
					logger.Warn("Failed to append event to block",
						slog.String("metric", "ai.event_persistence_failure"), // Structured attribute for monitoring
						slog.Int64("block_id", blockID),
						slog.String("event_type", evtType),
						slog.String("error", err.Error()))
				}
			}(currentBlock.ID, eventType)

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
	execErr := agent.ExecuteWithCallback(ctx, req.Message, req.History, callback)
	logger.Info("Agent: ExecuteWithCallback completed",
		slog.String("execErr", fmt.Sprintf("%v", execErr)),
		slog.Int64("duration_ms", time.Since(sessionStartTime).Milliseconds()))
	if execErr != nil {
		logger.Error("Agent execution failed", execErr)
		// Don't return here, continue to send session summary
	}

	// Prepare session summary

	// Calculate session summary
	sessionTotalDuration = time.Since(sessionStartTime).Milliseconds()
	logger.Info("Agent: preparing session summary",
		slog.Int64("duration_ms", sessionTotalDuration))

	// Try to get detailed stats from agent if available (GeekParrot/EvolutionParrot)
	// 尝试从 agent 获取详细统计数据（如果可用，如 GeekParrot/EvolutionParrot）
	var detailedStats *agentpkg.SessionStats
	if statsProvider, ok := agent.(agentpkg.SessionStatsProvider); ok {
		detailedStats = statsProvider.GetSessionStats()
		logger.Info("Agent: got detailed stats from SessionStatsProvider")
	} else {
		logger.Info("Agent: agent is not a SessionStatsProvider")
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

	// Set SessionId from detailedStats (Geek/Evolution modes use real UUID session IDs)
	// If no detailed stats available, fall back to conversation ID format for backward compatibility
	if detailedStats != nil && detailedStats.SessionID != "" {
		blockSummary.SessionId = detailedStats.SessionID
	} else {
		blockSummary.SessionId = fmt.Sprintf("conv_%d", req.ConversationID)
	}

	// NOTE: BlockSummary.Mode has been removed - Block.mode is the single source of truth.
	// The mode is stored in the Block (currentBlock.mode) and should be read from there.

	// Add detailed stats if available (from GeekParrot/EvolutionParrot)
	if detailedStats != nil {
		blockSummary.TotalDurationMs = detailedStats.TotalDurationMs
		blockSummary.ThinkingDurationMs = detailedStats.ThinkingDurationMs
		blockSummary.ToolDurationMs = detailedStats.ToolDurationMs
		blockSummary.GenerationDurationMs = detailedStats.GenerationDurationMs
		blockSummary.TotalInputTokens = detailedStats.InputTokens
		blockSummary.TotalOutputTokens = detailedStats.OutputTokens
		blockSummary.TotalCacheWriteTokens = detailedStats.CacheWriteTokens
		blockSummary.TotalCacheReadTokens = detailedStats.CacheReadTokens
		blockSummary.ToolCallCount = detailedStats.ToolCallCount
		if len(detailedStats.ToolsUsed) > 0 {
			tools := make([]string, 0, len(detailedStats.ToolsUsed))
			for tool := range detailedStats.ToolsUsed {
				tools = append(tools, tool)
			}
			blockSummary.ToolsUsed = tools
		}
		blockSummary.FilesModified = detailedStats.FilesModified
		blockSummary.FilePaths = detailedStats.FilePaths
	}

	// Safely send done marker
	streamMu.Lock()
	logger.Info("Agent: sending done marker with block summary",
		slog.String("session_id", blockSummary.SessionId),
		slog.Int64("duration_ms", blockSummary.TotalDurationMs),
		slog.Int64("tool_calls", int64(blockSummary.ToolCallCount)),
		slog.Bool("done", true),
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
		logger.Error("Agent: failed to send done marker", sendErr,
			slog.String("error", sendErr.Error()))
	} else {
		logger.Info("Agent: done marker sent successfully")
	}

	// Phase 5: Complete or mark error on Block
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
			}
		}
	}

	if sendErr != nil {
		logger.Error("Agent: failed to send done marker", sendErr)
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
// If cfg is nil, only rule-based routing is enabled (no LLM fallback).
func NewChatRouter(cfg *ai.IntentClassifierConfig, routerSvc *router.Service) *agentpkg.ChatRouter {
	routerCfg := agentpkg.ChatRouterConfig{}
	if cfg != nil {
		routerCfg.APIKey = cfg.APIKey
		routerCfg.BaseURL = cfg.BaseURL
		routerCfg.Model = cfg.Model
	}
	return agentpkg.NewChatRouter(routerCfg, routerSvc)
}
