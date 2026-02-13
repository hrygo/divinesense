package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/internal/strutil"
	"github.com/hrygo/divinesense/ai/routing"
)

// IntentResult represents the LLM classification result.
type IntentResult struct {
	Intent     TaskIntent `json:"intent"`
	Reasoning  string     `json:"reasoning,omitempty"`
	Confidence float64    `json:"confidence"`
}

// LLMIntentClassifier uses a lightweight LLM for intent classification.
// This provides better accuracy than rule-based matching, especially for
// nuanced natural language inputs.
//
// Deprecated: Use routing.Service.ClassifyIntent directly for new code.
// This classifier is kept for backward compatibility and LLM-based classification.
type LLMIntentClassifier struct {
	llm      llm.Service
	fallback routing.IntentClassifier
}

// LLMIntentConfig holds configuration for the LLM intent classifier.
//
// Deprecated: Use NewLLMIntentClassifierWithLLM(llmService) directly.
// This config is kept for backward compatibility.
type LLMIntentConfig struct {
	APIKey  string
	BaseURL string
	Model   string // Recommended: Qwen/Qwen2.5-7B-Instruct
}

// NewLLMIntentClassifier creates a new LLM-based intent classifier.
//
// Deprecated: Use NewLLMIntentClassifierWithLLM(llmService) instead.
// This constructor is kept for backward compatibility.
func NewLLMIntentClassifier(cfg LLMIntentConfig) *LLMIntentClassifier {
	// Create LLM service from config (backward compatibility)
	llmCfg := &llm.Config{
		Provider:    "generic",
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   100,
		Temperature: 0,
	}
	llmService, err := llm.NewService(llmCfg)
	if err != nil {
		slog.Error("failed to create LLM service for intent classifier", "error", err)
		return nil
	}
	return &LLMIntentClassifier{
		llm:      llmService,
		fallback: routing.NewService(routing.Config{EnableCache: true}),
	}
}

// NewLLMIntentClassifierWithLLM creates a new LLM-based intent classifier with an existing LLMService.
// This is the preferred constructor for dependency injection.
// Panics if llmService is nil.
func NewLLMIntentClassifierWithLLM(llmService llm.Service) *LLMIntentClassifier {
	if llmService == nil {
		panic("agent: NewLLMIntentClassifierWithLLM: llmService cannot be nil")
	}
	return &LLMIntentClassifier{
		llm:      llmService,
		fallback: routing.NewService(routing.Config{EnableCache: true}),
	}
}

// NewLLMIntentClassifierWithFallback creates a classifier with a custom fallback.
// Use this when you want to share the same routing.Service instance.
func NewLLMIntentClassifierWithFallback(llmService llm.Service, fallback routing.IntentClassifier) *LLMIntentClassifier {
	if llmService == nil {
		panic("agent: NewLLMIntentClassifierWithFallback: llmService cannot be nil")
	}
	if fallback == nil {
		fallback = routing.NewService(routing.Config{EnableCache: true})
	}
	return &LLMIntentClassifier{
		llm:      llmService,
		fallback: fallback,
	}
}

// Classify determines the intent of the user input using LLM.
func (ic *LLMIntentClassifier) Classify(ctx context.Context, input string) (TaskIntent, error) {
	result, err := ic.ClassifyWithDetails(ctx, input)
	if err != nil {
		slog.Warn("LLM intent classification failed, using fallback",
			"error", err,
			"input", strutil.Truncate(input, 50))
		// Use routing.IntentClassifier as fallback
		intent, _, _, ferr := ic.fallback.ClassifyIntent(ctx, input)
		if ferr != nil {
			return routing.IntentScheduleCreate, ferr
		}
		return intent, nil
	}
	return result.Intent, nil
}

// ClassifyWithDetails returns the full classification result including confidence.
func (ic *LLMIntentClassifier) ClassifyWithDetails(ctx context.Context, input string) (*IntentResult, error) {
	// Set timeout for classification (should be fast)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	messages := []llm.Message{
		llm.SystemPrompt(intentSystemPromptV2),
		llm.UserMessage(fmt.Sprintf("用户输入: %s", input)),
	}

	start := time.Now()
	content, stats, err := ic.llm.Chat(ctx, messages)
	latency := time.Since(start)

	if err != nil {
		slog.Error("llm_intent_classification_failed",
			"prompt_version", "v2",
			"error", err,
			"latency_ms", latency.Milliseconds())
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if content == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	result, err := ic.parseResponse(content)
	if err != nil {
		slog.Warn("llm_intent_parse_failed",
			"prompt_version", "v2",
			"content", content,
			"error", err)
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	slog.Debug("llm_intent_classification_success",
		"prompt_version", "v2",
		"input", strutil.Truncate(input, 30),
		"intent", result.Intent,
		"confidence", result.Confidence,
		"latency_ms", latency.Milliseconds(),
		"tokens_total", stats.TotalTokens,
		"tokens_prompt", stats.PromptTokens,
		"tokens_completion", stats.CompletionTokens)

	return result, nil
}

// parseResponse parses the LLM JSON response.
func (ic *LLMIntentClassifier) parseResponse(content string) (*IntentResult, error) {
	// Try to extract JSON from response
	content = strings.TrimSpace(content)

	// Handle potential markdown code blocks
	if strings.HasPrefix(content, "```") {
		re := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			content = matches[1]
		}
	}

	var raw struct {
		Intent     string  `json:"intent"`
		Reasoning  string  `json:"reasoning"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Map string to TaskIntent
	intent := ic.mapIntent(raw.Intent)

	return &IntentResult{
		Intent:     intent,
		Confidence: raw.Confidence,
		Reasoning:  raw.Reasoning,
	}, nil
}

// mapIntent converts string intent to TaskIntent enum.
func (ic *LLMIntentClassifier) mapIntent(s string) TaskIntent {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	// Schedule intents
	case "schedule_create", "simple_create", "create", "add":
		return routing.IntentScheduleCreate
	case "schedule_query", "simple_query", "query", "list":
		return routing.IntentScheduleQuery
	case "schedule_update", "simple_update", "update", "modify", "change":
		return routing.IntentScheduleUpdate
	case "schedule_batch", "batch_create", "batch", "recurring":
		return routing.IntentBatchSchedule
	case "schedule_conflict", "conflict_resolve", "conflict":
		return "schedule_conflict" // Not in routing.Intent yet
	// Memo intents
	case "memo_search", "search":
		return routing.IntentMemoSearch
	case "memo_create":
		return routing.IntentMemoCreate
	default:
		slog.Warn("Unknown intent from LLM, defaulting to schedule_create",
			"raw_intent", s)
		return routing.IntentScheduleCreate
	}
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (ic *LLMIntentClassifier) ShouldUsePlanExecute(intent TaskIntent) bool {
	return intent == routing.IntentBatchSchedule
}

// ClassifyAndRoute is a convenience method that classifies and returns the execution mode.
func (ic *LLMIntentClassifier) ClassifyAndRoute(ctx context.Context, input string) (TaskIntent, bool, error) {
	intent, err := ic.Classify(ctx, input)
	if err != nil {
		return routing.IntentScheduleCreate, false, err
	}
	usePlanExecute := ic.ShouldUsePlanExecute(intent)
	return intent, usePlanExecute, nil
}

// intentSystemPromptV2 is the system prompt for intent classification.
// Uses prompt instructions to ensure JSON output format (no JSON Schema required).
const intentSystemPromptV2 = `AI 助手意图分类器。判断用户意图并路由到对应 Agent。

## 日程 Agent (schedule)
- schedule_create: 创建单个日程 (有时间+事件)
- schedule_query: 查询日程/空闲 (问句)
- schedule_update: 修改/删除日程
- schedule_batch: 重复日程 (每天/每周/工作日)
- schedule_conflict: 处理冲突

## 笔记 Agent (memo)
- memo_search: 搜索笔记 (关键词)
- memo_create: 创建笔记 (记录内容)

## 分类规则
1. 含"笔记/记录/搜索" → memo_search
2. 含"今天/明天/会议" → schedule_create 或 schedule_query
3. 默认: schedule_create

## 输出格式
必须返回JSON格式：{"intent": "意图类型", "confidence": 0.95, "reasoning": "简短原因"}
confidence 取值范围 0-1，表示分类置信度。`

// Suppress unused import warning for ai package (used for type aliases)
var _ = ai.LLMConfig{}
