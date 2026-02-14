package agent

import (
	"context"
	"log/slog"

	ctxpkg "github.com/hrygo/divinesense/ai/context"
	"github.com/hrygo/divinesense/ai/internal/strutil"
)

// ChatRouterWithMetadata extends ChatRouter with metadata-based sticky routing.
// This implements the "Stateful Routing" principle from context-engineering.md:
// routing decisions are based on persisted database state (AIBlock.Metadata),
// not just in-memory session state.
type ChatRouterWithMetadata struct {
	*ChatRouter // Embed original router
	metadataMgr *ctxpkg.MetadataManager
}

// NewChatRouterWithMetadata creates a new chat router with metadata support.
func NewChatRouterWithMetadata(
	baseRouter *ChatRouter,
	metadataMgr *ctxpkg.MetadataManager,
) *ChatRouterWithMetadata {
	if baseRouter == nil {
		panic("ChatRouter is required")
	}
	return &ChatRouterWithMetadata{
		ChatRouter:  baseRouter,
		metadataMgr: metadataMgr,
	}
}

// RouteWithContextWithMetadata routes with metadata-based sticky routing.
// This method extends the base routing with:
// 1. Metadata-based sticky state (persistent across sessions)
// 2. Confidence-based sticky window decay
func (r *ChatRouterWithMetadata) RouteWithContextWithMetadata(
	ctx context.Context,
	input string,
	sessionCtx *ConversationContext,
	conversationID int32,
	blockID int64, // For persisting routing result
) (*ChatRouteResult, error) {
	// Layer 0: Check metadata-based sticky state first
	if r.metadataMgr != nil && conversationID > 0 {
		if isSticky, meta := r.metadataMgr.IsStickyValid(ctx, conversationID); isSticky && meta != nil {
			// Check if input is a short confirmation
			if isShortConfirmation(input) {
				slog.Debug("route reused from metadata sticky",
					"input", strutil.Truncate(input, 30),
					"route", meta.LastAgent,
					"confidence", meta.LastIntentConfidence,
					"method", "metadata_sticky")
				return &ChatRouteResult{
					Route:              ChatRouteType(meta.LastAgent),
					Confidence:         float64(meta.LastIntentConfidence),
					Method:             "metadata_sticky",
					NeedsOrchestration: false,
				}, nil
			}
		}
	}

	// Layer 1: Fall back to original routing logic
	result, err := r.RouteWithContext(ctx, input, sessionCtx)
	if err != nil {
		return result, err
	}

	// Layer 2: Persist routing result to metadata (if successful)
	if r.metadataMgr != nil && conversationID > 0 && blockID > 0 {
		if result.Route != "" && !result.NeedsOrchestration {
			intent := r.extractIntent(result.Route)
			if err := r.metadataMgr.SetCurrentAgent(
				ctx,
				conversationID,
				blockID,
				string(result.Route),
				intent,
				float32(result.Confidence),
			); err != nil {
				slog.Warn("failed to persist routing metadata",
					"error", err,
					"conversation_id", conversationID,
					"route", result.Route)
			}
		}
	}

	return result, nil
}

// extractIntent extracts the intent from route type for metadata storage.
func (r *ChatRouterWithMetadata) extractIntent(route ChatRouteType) string {
	switch route {
	case RouteTypeMemo:
		return "memo_search"
	case RouteTypeSchedule:
		return "schedule_manage"
	default:
		return "unknown"
	}
}

// InvalidateStickyCache invalidates the sticky cache for a conversation.
// Call this when the conversation context changes significantly.
func (r *ChatRouterWithMetadata) InvalidateStickyCache(conversationID int32) {
	if r.metadataMgr != nil {
		r.metadataMgr.Invalidate(conversationID)
	}
}
