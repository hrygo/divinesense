package agent

import (
	"context"
	"log/slog"
	"strings"

	routerpkg "github.com/hrygo/divinesense/ai/routing"
)

// ChatRouteType represents the type of chat routing.
type ChatRouteType string

const (
	// RouteTypeMemo routes to Memo Parrot (灰灰) for memo search and retrieval.
	// Implemented by UniversalParrot with memo.yaml configuration.
	RouteTypeMemo ChatRouteType = "memo"

	// RouteTypeSchedule routes to Schedule Parrot (时巧) for schedule management.
	// Implemented by UniversalParrot with schedule.yaml configuration.
	RouteTypeSchedule ChatRouteType = "schedule"

	// Note: RouteTypeAmazing removed - Orchestrator handles complex/ambiguous requests
)

// shortConfirmations contains common short confirmation words that should
// reuse the last route for intent stickiness (Issue #163).
var shortConfirmations = map[string]bool{
	// English
	"ok": true, "yes": true, "yeah": true, "yep": true, "sure": true, "right": true,
	"correct": true, "good": true, "fine": true, "alright": true, "okay": true,
	// Chinese
	"好": true, "好的": true, "嗯": true, "行": true, "可以": true,
	"没问题": true, "确认": true, "对": true, "是的": true, "同意": true, "确定": true,
}

// isShortConfirmation checks if the input is a short confirmation word.
func isShortConfirmation(input string) bool {
	normalized := strings.ToLower(strings.TrimSpace(input))
	// Remove common punctuation
	normalized = strings.TrimRight(normalized, "。！？.!?")
	return shortConfirmations[normalized]
}

// ChatRouteResult represents the routing classification result.
type ChatRouteResult struct {
	Route              ChatRouteType `json:"route"`
	Method             string        `json:"method"`
	Confidence         float64       `json:"confidence"`
	NeedsOrchestration bool          `json:"needs_orchestration"`
}

// ChatRouter routes user input to the appropriate Parrot agent.
// It is a thin adapter over routing.Service (three-layer routing).
type ChatRouter struct {
	routerService *routerpkg.Service // Three-layer router service (required)
}

// NewChatRouter creates a new chat router.
// routerSvc is required and provides the three-layer routing (cache → rule → history → LLM).
func NewChatRouter(routerSvc *routerpkg.Service) *ChatRouter {
	if routerSvc == nil {
		panic("routing.Service is required for ChatRouter")
	}
	return &ChatRouter{
		routerService: routerSvc,
	}
}

// Route determines the appropriate Parrot agent for the user input.
// Delegates to routing.Service which implements: cache → rule → history → LLM.
func (r *ChatRouter) Route(ctx context.Context, input string) (*ChatRouteResult, error) {
	return r.RouteWithContext(ctx, input, nil)
}

// RouteWithContext determines the appropriate Parrot agent with session context support.
// If the input is a short confirmation (e.g., "OK", "好的"), it reuses the last route
// for intent stickiness, enabling seamless multi-turn conversations (Issue #163).
func (r *ChatRouter) RouteWithContext(ctx context.Context, input string, sessionCtx *ConversationContext) (*ChatRouteResult, error) {
	// Check for short confirmation - reuse last route for intent stickiness
	if sessionCtx != nil && isShortConfirmation(input) {
		if lastRoute, ok := sessionCtx.GetLastRoute(); ok {
			slog.Debug("route reused for short confirmation",
				"input", TruncateString(input, 30),
				"route", lastRoute,
				"method", "session_sticky")
			return &ChatRouteResult{
				Route:              lastRoute,
				Confidence:         0.95,
				Method:             "session_sticky",
				NeedsOrchestration: false,
			}, nil
		}
	}

	// FastRouter: cache -> rule
	intent, confidence, needsOrch, err := r.routerService.ClassifyIntent(ctx, input)
	if err != nil {
		slog.Warn("router service failed, needs orchestration",
			"error", err,
			"input", TruncateString(input, 30))
		return &ChatRouteResult{
			Route:              "", // Empty route indicates orchestration needed
			Confidence:         0.5,
			Method:             "fallback",
			NeedsOrchestration: true,
		}, nil
	}

	result := &ChatRouteResult{
		Route:              mapIntentToRouteType(intent),
		Confidence:         float64(confidence),
		Method:             "router",
		NeedsOrchestration: needsOrch,
	}

	// Store the route for future stickiness (only if confident)
	if sessionCtx != nil && result.Route != "" && !needsOrch {
		sessionCtx.SetLastRoute(result.Route)
	}

	return result, nil
}

// mapIntentToRouteType converts routing.Intent to ChatRouteType.
// Uses the canonical IntentToAgentType mapping from routing package.
func mapIntentToRouteType(intent routerpkg.Intent) ChatRouteType {
	switch routerpkg.IntentToAgentType(intent) {
	case routerpkg.AgentTypeMemo:
		return RouteTypeMemo
	case routerpkg.AgentTypeSchedule:
		return RouteTypeSchedule
	default:
		return "" // Empty indicates unknown - needs orchestration
	}
}
