package agent

import (
	"context"
	"log/slog"

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

	// RouteTypeAmazing routes to Amazing Parrot (折衷) for comprehensive assistance.
	// Implemented by UniversalParrot with amazing.yaml configuration.
	RouteTypeAmazing ChatRouteType = "amazing"
)

// ChatRouteResult represents the routing classification result.
type ChatRouteResult struct {
	Route      ChatRouteType `json:"route"`
	Method     string        `json:"method"`
	Confidence float64       `json:"confidence"`
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
	intent, confidence, err := r.routerService.ClassifyIntent(ctx, input)
	if err != nil {
		slog.Warn("router service failed, defaulting to amazing",
			"error", err,
			"input", TruncateString(input, 30))
		return &ChatRouteResult{
			Route:      RouteTypeAmazing,
			Confidence: 0.5,
			Method:     "fallback",
		}, nil
	}
	return &ChatRouteResult{
		Route:      mapIntentToRouteType(intent),
		Confidence: float64(confidence),
		Method:     "router",
	}, nil
}

// mapIntentToRouteType converts routing.Intent to ChatRouteType.
// Uses the canonical IntentToAgentType mapping from routing package.
func mapIntentToRouteType(intent routerpkg.Intent) ChatRouteType {
	switch routerpkg.IntentToAgentType(intent) {
	case routerpkg.AgentTypeMemo:
		return RouteTypeMemo
	case routerpkg.AgentTypeSchedule:
		return RouteTypeSchedule
	case routerpkg.AgentTypeAmazing:
		return RouteTypeAmazing
	default:
		return RouteTypeAmazing
	}
}
