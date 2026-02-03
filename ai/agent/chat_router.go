package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/router"
	"github.com/sashabaranov/go-openai"
)

// ChatRouteType represents the type of chat routing.
type ChatRouteType string

const (
	// RouteTypeMemo routes to MemoParrot (灰灰) for memo search and retrieval.
	RouteTypeMemo ChatRouteType = "memo"

	// RouteTypeSchedule routes to ScheduleParrot (时巧) for schedule management.
	RouteTypeSchedule ChatRouteType = "schedule"

	// RouteTypeAmazing routes to AmazingParrot (折衷) for comprehensive assistance.
	RouteTypeAmazing ChatRouteType = "amazing"
)

// ChatRouteResult represents the routing classification result.
type ChatRouteResult struct {
	Route      ChatRouteType `json:"route"`
	Method     string        `json:"method"`
	Confidence float64       `json:"confidence"`
}

// ChatRouterConfig holds configuration for the chat router.
type ChatRouterConfig struct {
	APIKey  string
	BaseURL string
	Model   string // Default: Qwen/Qwen2.5-7B-Instruct
}

// ChatRouter routes user input to the appropriate Parrot agent.
// It uses a hybrid approach: fast rule matching first, then LLM for uncertain cases.
// Optionally uses the three-layer router.Service for enhanced history matching.
type ChatRouter struct {
	routerService *router.Service // Optional: three-layer router service
	client        *openai.Client  // Reserved for LLM classification
	model         string
}

// NewChatRouter creates a new chat router with hybrid rule+LLM classification.
// If routerSvc is provided, uses the three-layer router (rule + history + LLM).
// Otherwise, falls back to two-layer routing (rule + LLM only).
func NewChatRouter(cfg ChatRouterConfig, routerSvc *router.Service) *ChatRouter {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	model := cfg.Model
	if model == "" {
		model = "Qwen/Qwen2.5-7B-Instruct"
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = baseURL

	return &ChatRouter{
		routerService: routerSvc,
		client:        openai.NewClientWithConfig(clientConfig),
		model:         model,
	}
}

// Route determines the appropriate Parrot agent for the user input.
// Uses hybrid approach: rule matching (0ms) → history matching (~10ms) → LLM classification (~400ms) if uncertain.
func (r *ChatRouter) Route(ctx context.Context, input string) (*ChatRouteResult, error) {
	// Step 1: Try router.Service (three-layer routing) if available
	if r.routerService != nil {
		intent, confidence, err := r.routerService.ClassifyIntent(ctx, input)
		if err != nil {
			slog.Warn("router service failed, defaulting to amazing",
				"error", err,
				"input", truncateString(input, 30))
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

	// Step 2: Try rule-based matching (fast path, for backward compatibility)
	if result := r.routeByRules(input); result != nil {
		slog.Info("chat routed by rules",
			"input", truncateString(input, 30),
			"route", result.Route,
			"confidence", result.Confidence)
		return result, nil
	}

	// Step 3: Use LLM for uncertain cases
	result, err := r.routeByLLM(ctx, input)
	if err != nil {
		slog.Error("LLM routing failed, defaulting to amazing",
			"error", err,
			"input", truncateString(input, 100),
			"model", r.model)
		return &ChatRouteResult{
			Route:      RouteTypeAmazing,
			Confidence: 0.5,
			Method:     "fallback",
		}, nil
	}

	slog.Debug("chat routed by LLM",
		"input", truncateString(input, 30),
		"route", result.Route,
		"confidence", result.Confidence)

	return result, nil
}

// routeByRules attempts fast rule-based routing.
// Returns nil if no confident match is found.
func (*ChatRouter) routeByRules(input string) *ChatRouteResult {
	lower := strings.ToLower(input)

	// Schedule: high-confidence keywords
	scheduleKeywords := []string{
		"日程", "schedule", "安排", "几点", "有空", "空闲",
		"会议", "meeting", "提醒", "remind", "预约", "约",
	}
	scheduleTimePatterns := []string{
		"今天", "明天", "后天", "下周", "这周", "周一", "周二", "周三", "周四", "周五", "周六", "周日",
		"上午", "下午", "晚上", "早上", "中午",
		"点", "时", "分",
	}

	scheduleScore := 0
	for _, kw := range scheduleKeywords {
		if strings.Contains(lower, kw) {
			scheduleScore += 2
		}
	}
	for _, pat := range scheduleTimePatterns {
		if strings.Contains(lower, pat) {
			scheduleScore++
		}
	}

	// If strong schedule signal, route to schedule
	if scheduleScore >= 3 {
		return &ChatRouteResult{
			Route:      RouteTypeSchedule,
			Confidence: 0.85,
			Method:     "rule",
		}
	}

	// Memo: high-confidence keywords
	memoKeywords := []string{
		"笔记", "memo", "note", "记录", "搜索", "search", "查找", "find",
		"写过", "记过", "提到", "关于",
	}

	memoScore := 0
	for _, kw := range memoKeywords {
		if strings.Contains(lower, kw) {
			memoScore += 2
		}
	}

	// If strong memo signal without schedule signal, route to memo
	if memoScore >= 2 && scheduleScore < 2 {
		return &ChatRouteResult{
			Route:      RouteTypeMemo,
			Confidence: 0.80,
			Method:     "rule",
		}
	}

	// Amazing: explicit comprehensive keywords
	amazingKeywords := []string{
		"综合", "总结一下", "分析", "overview", "summary",
		"本周工作", "今日总结", "周报",
	}

	for _, kw := range amazingKeywords {
		if strings.Contains(lower, kw) {
			return &ChatRouteResult{
				Route:      RouteTypeAmazing,
				Confidence: 0.80,
				Method:     "rule",
			}
		}
	}

	// No confident match - need LLM
	return nil
}

// routeByLLM uses LLM for semantic understanding of uncertain inputs.
func (r *ChatRouter) routeByLLM(ctx context.Context, input string) (*ChatRouteResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:       r.model,
		MaxTokens:   30,
		Temperature: 0,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: chatRouterSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: input,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "chat_routing",
				Strict: true,
				Schema: chatRouterJSONSchema,
			},
		},
	}

	start := time.Now()
	resp, err := r.client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		slog.Error("chat router LLM request failed",
			"error", err,
			"latency_ms", latency.Milliseconds())
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content

	var raw struct {
		Route      string  `json:"route"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	route := r.mapRoute(raw.Route)

	slog.Debug("chat router LLM completed",
		"route", route,
		"confidence", raw.Confidence,
		"latency_ms", latency.Milliseconds(),
		"tokens", resp.Usage.TotalTokens)

	return &ChatRouteResult{
		Route:      route,
		Confidence: raw.Confidence,
		Method:     "llm",
	}, nil
}

// mapRoute converts string route to ChatRouteType.
func (r *ChatRouter) mapRoute(s string) ChatRouteType {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "memo", "笔记":
		return RouteTypeMemo
	case "schedule", "日程":
		return RouteTypeSchedule
	default:
		return RouteTypeAmazing
	}
}

// chatRouterSystemPrompt is a minimal prompt for chat routing.
const chatRouterSystemPrompt = `聊天路由分类器。判断用户意图路由到哪个助手：

memo: 笔记搜索、查找记录、回忆内容
schedule: 日程管理、时间安排、会议提醒、空闲查询
amazing: 综合分析、无法明确归类、需要笔记+日程

默认: amazing`

// chatRouterJSONSchema defines the strict output schema.
var chatRouterJSONSchema = &jsonSchema{
	Type: "object",
	Properties: map[string]*jsonSchema{
		"route": {
			Type:        "string",
			Enum:        []string{"memo", "schedule", "amazing"},
			Description: "The routing target",
		},
		"confidence": {
			Type:        "number",
			Description: "Confidence score 0-1",
		},
	},
	Required:             []string{"route", "confidence"},
	AdditionalProperties: false,
}

// mapIntentToRouteType converts router.Intent to ChatRouteType.
// Uses the canonical IntentToAgentType mapping from router package.
func mapIntentToRouteType(intent router.Intent) ChatRouteType {
	switch router.IntentToAgentType(intent) {
	case router.AgentTypeMemo:
		return RouteTypeMemo
	case router.AgentTypeSchedule:
		return RouteTypeSchedule
	case router.AgentTypeAmazing:
		return RouteTypeAmazing
	default:
		return RouteTypeAmazing
	}
}
