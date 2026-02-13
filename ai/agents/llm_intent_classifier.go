package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
	"github.com/hrygo/divinesense/ai/internal/strutil"
	"github.com/sashabaranov/go-openai"
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
type LLMIntentClassifier struct {
	client   *openai.Client
	fallback *IntentClassifier
	model    string
}

// LLMIntentConfig holds configuration for the LLM intent classifier.
type LLMIntentConfig struct {
	APIKey  string
	BaseURL string
	Model   string // Recommended: Qwen/Qwen2.5-7B-Instruct
}

// NewLLMIntentClassifier creates a new LLM-based intent classifier.
// Uses a lightweight model optimized for fast classification.
func NewLLMIntentClassifier(cfg LLMIntentConfig) *LLMIntentClassifier {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.siliconflow.cn/v1"
	}

	model := cfg.Model
	if model == "" {
		// Default to a fast, cost-effective model for classification
		model = "Qwen/Qwen2.5-7B-Instruct"
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = baseURL

	return &LLMIntentClassifier{
		client:   openai.NewClientWithConfig(clientConfig),
		model:    model,
		fallback: NewIntentClassifier(),
	}
}

// Classify determines the intent of the user input using LLM.
func (ic *LLMIntentClassifier) Classify(ctx context.Context, input string) (TaskIntent, error) {
	result, err := ic.ClassifyWithDetails(ctx, input)
	if err != nil {
		slog.Warn("LLM intent classification failed, using fallback",
			"error", err,
			"input", strutil.Truncate(input, 50))
		return ic.fallback.Classify(input), nil
	}
	return result.Intent, nil
}

// ClassifyWithDetails returns the full classification result including confidence.
func (ic *LLMIntentClassifier) ClassifyWithDetails(ctx context.Context, input string) (*IntentResult, error) {
	// Set timeout for classification (should be fast)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	prompt := ic.buildPrompt(input)

	req := openai.ChatCompletionRequest{
		Model:       ic.model,
		MaxTokens:   50, // Strict schema ensures minimal output
		Temperature: 0,  // Deterministic output
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: intentSystemPromptStrict,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "intent_classification",
				Strict: true,
				Schema: intentJSONSchema,
			},
		},
	}

	start := time.Now()
	resp, err := ic.client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		slog.Error("llm_intent_classification_failed",
			"prompt_version", "v1",
			"model", ic.model,
			"error", err,
			"latency_ms", latency.Milliseconds())
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content
	result, err := ic.parseResponse(content)
	if err != nil {
		slog.Warn("llm_intent_parse_failed",
			"prompt_version", "v1",
			"model", ic.model,
			"content", content,
			"error", err)
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	slog.Debug("llm_intent_classification_success",
		"prompt_version", "v1",
		"model", ic.model,
		"input", strutil.Truncate(input, 30),
		"intent", result.Intent,
		"confidence", result.Confidence,
		"latency_ms", latency.Milliseconds(),
		"tokens_total", resp.Usage.TotalTokens,
		"tokens_prompt", resp.Usage.PromptTokens,
		"tokens_completion", resp.Usage.CompletionTokens)

	return result, nil
}

// buildPrompt constructs the classification prompt.
func (ic *LLMIntentClassifier) buildPrompt(input string) string {
	return fmt.Sprintf("用户输入: %s", input)
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
		return IntentSimpleCreate
	case "schedule_query", "simple_query", "query", "list":
		return IntentSimpleQuery
	case "schedule_update", "simple_update", "update", "modify", "change":
		return IntentSimpleUpdate
	case "schedule_batch", "batch_create", "batch", "recurring":
		return IntentBatchCreate
	case "schedule_conflict", "conflict_resolve", "conflict":
		return IntentConflictResolve
	// Memo intents
	case "memo_search", "search":
		return IntentMemoSearch
	case "memo_create":
		return IntentMemoCreate
	default:
		slog.Warn("Unknown intent from LLM, defaulting to schedule_create",
			"raw_intent", s)
		return IntentSimpleCreate
	}
}

// ShouldUsePlanExecute returns true if the intent should use Plan-Execute mode.
func (ic *LLMIntentClassifier) ShouldUsePlanExecute(intent TaskIntent) bool {
	return intent == IntentBatchCreate
}

// ClassifyAndRoute is a convenience method that classifies and returns the execution mode.
func (ic *LLMIntentClassifier) ClassifyAndRoute(ctx context.Context, input string) (TaskIntent, bool, error) {
	intent, err := ic.Classify(ctx, input)
	if err != nil {
		return IntentSimpleCreate, false, err
	}
	usePlanExecute := ic.ShouldUsePlanExecute(intent)
	return intent, usePlanExecute, nil
}

// intentSystemPromptStrict is a minimal prompt for strict JSON schema mode.
// The schema enforces the output format, so we only need classification rules.
const intentSystemPromptStrict = `AI 助手意图分类器。判断用户意图并路由到对应 Agent：

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
3. 默认: schedule_create`

// intentJSONSchema defines the strict output schema for intent classification.
// Using enum to constrain intent values and prevent hallucination.
var intentJSONSchema = &llm.JSONSchema{
	Type: "object",
	Properties: map[string]*llm.JSONSchema{
		"intent": {
			Type: "string",
			Enum: []string{
				"schedule_create",
				"schedule_query",
				"schedule_update",
				"schedule_batch",
				"schedule_conflict",
				"memo_search",
				"memo_create",
			},
			Description: "The classified intent type",
		},
		"confidence": {
			Type:        "number",
			Description: "Confidence score between 0 and 1",
		},
	},
	Required:             []string{"intent", "confidence"},
	AdditionalProperties: false,
}
