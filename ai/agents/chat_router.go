package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/hrygo/divinesense/ai/internal/strutil"
	routerpkg "github.com/hrygo/divinesense/ai/routing"
)

// IntentKeywordProvider defines the interface for providing intent keywords for sticky routing.
// This enables dynamic loading of keywords from expert configurations.
type IntentKeywordProvider interface {
	// GetIntentKeywords returns a map of intent names to their related keywords.
	GetIntentKeywords() map[string][]string
}

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

// intentKeywords maps intent names to their related keywords for sticky routing.
// This enables sticky routing for inputs that are semantically related to the last intent.
// Note: These keywords should match the routing.keywords in expert config files (memo.yaml, schedule.yaml).
// The actual values are loaded from config at runtime via IntentKeywordProvider.
var intentKeywords = map[string][]string{
	// Memo-related intents
	"memo_search": {
		// Core keywords (from memo.yaml routing.keywords)
		"笔记", "搜索", "找", "记录", "memo",
		// Sticky routing: follow-up to previous memo search
		"总结", "详细内容", "相关", "这条", "note", "search", "find", "look",
	},
	"create_note": {"记", "记录", "写", "创建", "note", "record", "create", "write"},
	"delete_note": {"删除", "笔记", "note", "delete", "remove"},
	"update_note": {"修改", "更新", "编辑", "笔记", "note", "edit", "update", "modify"},

	// Schedule-related intents
	"schedule_manage": {
		// Core keywords (from schedule.yaml routing.keywords)
		"日程", "会议", "提醒", "安排", "schedule",
		// Sticky routing: follow-up to previous schedule management
		"修改", "取消", "时间", "地点", "参会", "meeting", "event", "calendar",
	},
	"create_event":   {"日程", "会议", "安排", "创建", "schedule", "meeting", "event", "create", "add"},
	"delete_event":   {"删除", "日程", "会议", "schedule", "meeting", "delete", "remove", "cancel"},
	"query_schedule": {"日程", "会议", "查看", "查询", "schedule", "meeting", "query", "check", "show"},
	"update_event":   {"修改", "更新", "调整", "日程", "schedule", "edit", "update", "modify", "change"},
}

// isRelatedToLastIntent checks if the input is semantically related to the last intent.
// This enables sticky routing for follow-up questions that don't use confirmation words.
// If provider is nil, falls back to hardcoded intentKeywords map.
func isRelatedToLastIntentWithProvider(input string, lastIntent string, provider IntentKeywordProvider) bool {
	if lastIntent == "" {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(input))

	// Try to get keywords from provider first, fallback to hardcoded
	var keywords []string
	var exists bool

	if provider != nil {
		keywords, exists = provider.GetIntentKeywords()[lastIntent]
	}

	// Fallback to hardcoded keywords if provider doesn't have them
	if !exists || len(keywords) == 0 {
		keywords, exists = intentKeywords[lastIntent]
		if !exists {
			// Fallback: check if any keyword contains the intent name
			lowerIntent := strings.ToLower(lastIntent)
			for key, kws := range intentKeywords {
				if strings.Contains(lowerIntent, key) || strings.Contains(key, lowerIntent) {
					keywords = kws
					break
				}
			}
		}
	}

	// Check if input contains any of the keywords
	for _, kw := range keywords {
		if strings.Contains(normalized, kw) {
			return true
		}
	}

	return false
}

// ExtractIntent extracts the intent from route type for metadata storage.
// This is a standalone function to avoid DRY violation between packages.
func ExtractIntent(route ChatRouteType) string {
	switch route {
	case RouteTypeMemo:
		return "memo_search"
	case RouteTypeSchedule:
		return "schedule_manage"
	default:
		return "unknown"
	}
}

// ChatRouteResult represents the routing classification result.
type ChatRouteResult struct {
	Route              ChatRouteType `json:"route"`
	Method             string        `json:"method"`
	Confidence         float64       `json:"confidence"`
	NeedsOrchestration bool          `json:"needs_orchestration"`
	// Handoff indicates whether a handoff occurred during sticky route execution.
	Handoff bool `json:"handoff"`
	// HandoffResult contains the result of a handoff operation (if Handoff is true).
	HandoffResult *HandoffResult `json:"handoff_result,omitempty"`
	// ExecutionResult contains the result of expert execution (if not handoff).
	ExecutionResult string `json:"execution_result,omitempty"`
}

// HandoffResult contains the result of a handoff operation from ChatRouter.
type HandoffResult struct {
	// Success indicates whether the handoff was successful.
	Success bool `json:"success"`
	// FromExpert is the original expert that could not handle the request.
	FromExpert string `json:"from_expert"`
	// ToExpert is the expert that took over (if successful).
	ToExpert string `json:"to_expert,omitempty"`
	// Error is the error message if handoff failed.
	Error string `json:"error,omitempty"`
	// FallbackMessage is a user-friendly message when handoff fails.
	FallbackMessage string `json:"fallback_message,omitempty"`
}

// SimpleHandoffHandler is a simple interface for handling handoff between experts.
// This avoids circular imports by using interface{} and type assertions.
type SimpleHandoffHandler interface {
	// HandleSimpleHandoff processes a simplified handoff request.
	HandleSimpleHandoff(req SimpleHandoffRequest) SimpleHandoverResult
}

// SimpleHandoffRequest is a simplified request for handoff.
type SimpleHandoffRequest struct {
	TaskID   string
	Agent    string
	Input    string
	Capacity string
	Reason   string
}

// SimpleHandoverResult is a simplified result for handoff.
type SimpleHandoverResult struct {
	Success         bool
	FromExpert      string
	ToExpert        string
	NewTaskInput    string
	Error           string
	FallbackMessage string
}

// ExpertRegistryInterface defines the interface for accessing expert agents.
type ExpertRegistryInterface interface {
	ExecuteExpert(ctx context.Context, expertName string, input string, callback func(eventType string, eventData string) error) error
}

// ChatRouter routes user input to the appropriate Parrot agent.
// It is a thin adapter over routing.Service (two-layer: cache -> rule).
type ChatRouter struct {
	routerService  *routerpkg.Service      // Two-layer router service (cache -> rule)
	handoffHandler SimpleHandoffHandler    // For handoff support
	expertRegistry ExpertRegistryInterface // For expert execution
}

// NewChatRouter creates a new chat router.
// routerSvc is required and provides the two-layer routing (cache -> rule).
func NewChatRouter(routerSvc *routerpkg.Service) *ChatRouter {
	if routerSvc == nil {
		panic("routing.Service is required for ChatRouter")
	}
	return &ChatRouter{
		routerService: routerSvc,
	}
}

// NewChatRouterWithHandoff creates a new chat router with handoff support.
// routerSvc is required and provides the two-layer routing.
// handoffHandler is optional - when provided, enables handoff on MissingCapability errors.
// expertRegistry is required when handoffHandler is provided.
func NewChatRouterWithHandoff(routerSvc *routerpkg.Service, handoffHandler SimpleHandoffHandler, expertRegistry ExpertRegistryInterface) *ChatRouter {
	if routerSvc == nil {
		panic("routing.Service is required for ChatRouter")
	}
	if handoffHandler != nil && expertRegistry == nil {
		panic("expertRegistry is required when handoffHandler is provided")
	}
	return &ChatRouter{
		routerService:  routerSvc,
		handoffHandler: handoffHandler,
		expertRegistry: expertRegistry,
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
				"input", strutil.Truncate(input, 30),
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
			"input", strutil.Truncate(input, 30))
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

// RouteAndExecute determines the appropriate Parrot agent and executes it for sticky routes.
// This method is used when the user wants to execute the expert directly with handoff support.
// It handles MissingCapability errors by triggering handoff to another expert.
func (r *ChatRouter) RouteAndExecute(ctx context.Context, input string, sessionCtx *ConversationContext) (*ChatRouteResult, error) {
	// First, get the route decision
	routeResult, err := r.RouteWithContext(ctx, input, sessionCtx)
	if err != nil {
		return routeResult, err
	}

	// If no handoff handler or expert registry, just return the route result
	if r.handoffHandler == nil || r.expertRegistry == nil {
		return routeResult, nil
	}

	// If not a sticky route (session_sticky), just return the route result
	if routeResult.Method != "session_sticky" {
		return routeResult, nil
	}

	// Execute the expert for sticky route with handoff support
	expertName := r.routeTypeToExpertName(routeResult.Route)
	if expertName == "" {
		return routeResult, nil
	}

	slog.Debug("chatrouter: executing sticky route",
		"expert", expertName,
		"input", strutil.Truncate(input, 30))

	// Execute the expert
	resultChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		// Create a callback to collect results
		var mu sync.Mutex
		var result strings.Builder
		callback := func(eventType string, eventData string) error {
			// Collect content events
			if eventType == "content" || eventType == "text" || eventType == "response" {
				mu.Lock()
				result.WriteString(eventData)
				mu.Unlock()
			}
			return nil
		}

		err := r.expertRegistry.ExecuteExpert(ctx, expertName, input, callback)
		if err != nil {
			errChan <- err
			return
		}
		mu.Lock()
		resultChan <- result.String()
		mu.Unlock()
	}()

	// Wait for result or error
	select {
	case result := <-resultChan:
		routeResult.ExecutionResult = result

		// Check if result contains INABILITY_REPORTED (for Handoff mechanism)
		// When an expert reports inability via report_inability tool, we need to trigger handoff
		if strings.Contains(result, "INABILITY_REPORTED:") && r.handoffHandler != nil {
			slog.Info("chatrouter: INABILITY_REPORTED detected in result, triggering handoff",
				"expert", expertName,
				"result_preview", result[:min(len(result), 100)])

			// Parse the inability report to extract capability and reason
			capability, reason := ParseInabilityReport(result)

			// Create a simplified handoff request
			req := SimpleHandoffRequest{
				TaskID:   "task_" + strings.ReplaceAll(expertName, " ", "_"),
				Agent:    expertName,
				Input:    input,
				Capacity: capability,
				Reason:   reason,
			}

			// Handle handoff
			handoffResult := r.handoffHandler.HandleSimpleHandoff(req)

			// Convert to ChatRouter.HandoffResult
			routeResult.Handoff = true
			routeResult.HandoffResult = &HandoffResult{
				Success:         handoffResult.Success,
				FromExpert:      expertName,
				ToExpert:        handoffResult.ToExpert,
				Error:           handoffResult.Error,
				FallbackMessage: handoffResult.FallbackMessage,
			}

			// If handoff succeeded, execute with the new expert
			if handoffResult.Success && handoffResult.NewTaskInput != "" {
				slog.Info("chatrouter: handoff succeeded, executing with new expert",
					"from", expertName,
					"to", handoffResult.ToExpert)

				// Execute with new expert
				var newResult strings.Builder
				newCallback := func(eventType string, eventData string) error {
					if eventType == "content" || eventType == "text" || eventType == "response" {
						newResult.WriteString(eventData)
					}
					return nil
				}

				newErr := r.expertRegistry.ExecuteExpert(ctx, handoffResult.ToExpert, handoffResult.NewTaskInput, newCallback)
				if newErr != nil {
					slog.Error("chatrouter: failed to execute with handoff expert", "error", newErr)
					routeResult.ExecutionResult = handoffResult.FallbackMessage
				} else {
					routeResult.ExecutionResult = newResult.String()
				}
			} else {
				// Handoff failed, use fallback message
				routeResult.ExecutionResult = handoffResult.FallbackMessage
			}
		}

		return routeResult, nil
	case err := <-errChan:
		// Check for MissingCapability error
		var missingCap *MissingCapability
		if errors.As(err, &missingCap) {
			slog.Info("chatrouter: MissingCapability detected, triggering handoff",
				"expert", expertName,
				"missing", missingCap.MissingCapabilities)

			// Create a simplified handoff request
			req := SimpleHandoffRequest{
				TaskID:   "task_" + strings.ReplaceAll(expertName, " ", "_"),
				Agent:    expertName,
				Input:    input,
				Capacity: strings.Join(missingCap.MissingCapabilities, ", "),
				Reason:   err.Error(),
			}

			// Handle handoff
			handoffResult := r.handoffHandler.HandleSimpleHandoff(req)

			// Convert to ChatRouter.HandoffResult
			routeResult.Handoff = true
			routeResult.HandoffResult = &HandoffResult{
				Success:         handoffResult.Success,
				FromExpert:      expertName,
				ToExpert:        handoffResult.ToExpert,
				Error:           handoffResult.Error,
				FallbackMessage: handoffResult.FallbackMessage,
			}

			// If handoff succeeded, execute with the new expert
			if handoffResult.Success && handoffResult.NewTaskInput != "" {
				slog.Info("chatrouter: handoff succeeded, executing with new expert",
					"from", expertName,
					"to", handoffResult.ToExpert)

				// Execute with new expert
				var newResult strings.Builder
				newCallback := func(eventType string, eventData string) error {
					if eventType == "content" || eventType == "text" || eventType == "response" {
						newResult.WriteString(eventData)
					}
					return nil
				}

				newErr := r.expertRegistry.ExecuteExpert(ctx, handoffResult.ToExpert, handoffResult.NewTaskInput, newCallback)
				if newErr != nil {
					slog.Error("chatrouter: failed to execute with handoff expert", "error", newErr)
					routeResult.ExecutionResult = handoffResult.FallbackMessage
				} else {
					routeResult.ExecutionResult = newResult.String()
				}
			} else {
				// Handoff failed, use fallback message
				routeResult.ExecutionResult = handoffResult.FallbackMessage
			}

			return routeResult, nil
		}

		// Not a MissingCapability error, return sanitized error
		routeResult.ExecutionResult = "执行出错，请稍后重试"
		slog.Error("chatrouter: expert execution failed",
			"expert", expertName,
			"error_type", fmt.Sprintf("%T", err))
		return routeResult, nil
	}
}

// routeTypeToExpertName converts a ChatRouteType to the expert name.
func (r *ChatRouter) routeTypeToExpertName(routeType ChatRouteType) string {
	switch routeType {
	case RouteTypeMemo:
		return "memo"
	case RouteTypeSchedule:
		return "schedule"
	default:
		return ""
	}
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

// ParseInabilityReport parses the INABILITY_REPORTED message to extract capability and reason.
// This is a public function to avoid DRY violation between packages.
// Format: "INABILITY_REPORTED: <capability> - <reason> (suggested_agent: <agent>)"
func ParseInabilityReport(report string) (capability, reason string) {
	prefix := "INABILITY_REPORTED:"
	if !strings.HasPrefix(report, prefix) {
		return "", report
	}

	content := strings.TrimPrefix(report, prefix)
	content = strings.TrimSpace(content)

	// Split by " - " to separate capability and reason
	before, after, found := strings.Cut(content, " - ")
	if found {
		capability = strings.TrimSpace(before)
		reason = strings.TrimSpace(after)
		// Remove suggested_agent part if present
		if before, _, found := strings.Cut(reason, " (suggested_agent:"); found {
			reason = strings.TrimSpace(before)
		}
		return capability, reason
	}

	return "", content
}
