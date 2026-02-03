// Package router provides the LLM routing service.
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LLMClient defines the interface for LLM API calls.
type LLMClient interface {
	// Complete sends a completion request and returns the response.
	Complete(ctx context.Context, prompt string, config ModelConfig) (string, error)
}

// LLMClassifier implements Layer 3 LLM-based intent classification.
// Target: ~400ms latency, handle only ~20% of requests that pass Layer 1&2.
type LLMClassifier struct {
	client              LLMClient
	confidenceThreshold float32
}

// NewLLMClassifier creates a new LLM classifier.
func NewLLMClassifier(client LLMClient) *LLMClassifier {
	return &LLMClassifier{
		client:              client,
		confidenceThreshold: 0.7,
	}
}

// LLMClassifyResult contains the result of LLM classification.
type LLMClassifyResult struct {
	Intent     Intent
	Reasoning  string
	Confidence float32
}

// ClassificationPrompt is the prompt template for intent classification.
const ClassificationPrompt = `你是一个意图分类助手。请分析用户输入，判断其意图类型。

可选的意图类型：
- memo_search: 搜索或查找笔记
- memo_create: 创建或记录新笔记
- schedule_query: 查询日程安排
- schedule_create: 创建新日程或提醒
- schedule_update: 修改或取消日程
- batch_schedule: 批量创建日程
- amazing: 综合性问题、分析、总结等
- unknown: 无法明确分类

用户输入: %s

请以JSON格式输出，包含以下字段：
- intent: 意图类型（上述之一）
- confidence: 置信度（0-1之间的小数）
- reasoning: 简要说明判断理由

只输出JSON，不要有其他内容。`

// Classify classifies user intent using LLM.
// This is the fallback layer for truly ambiguous inputs.
func (c *LLMClassifier) Classify(ctx context.Context, input string) (*LLMClassifyResult, error) {
	if c.client == nil {
		return &LLMClassifyResult{
			Intent:     IntentUnknown,
			Confidence: 0,
			Reasoning:  "LLM client not configured",
		}, nil
	}

	// Build prompt
	prompt := fmt.Sprintf(ClassificationPrompt, input)

	// Call LLM
	config := ModelConfig{
		Provider:    "cloud",
		Model:       "deepseek-chat",
		MaxTokens:   256,
		Temperature: 0.1, // Low temperature for classification
	}

	response, err := c.client.Complete(ctx, prompt, config)
	if err != nil {
		return nil, fmt.Errorf("LLM classification failed: %w", err)
	}

	// Parse response
	result, err := c.parseResponse(response)
	if err != nil {
		//nolint:nilerr // Return partial result on parse failure
		return &LLMClassifyResult{
			Intent:     IntentUnknown,
			Confidence: 0.3,
			Reasoning:  "Failed to parse LLM response: " + err.Error(),
		}, nil
	}

	// Apply confidence threshold
	if result.Confidence < c.confidenceThreshold {
		result.Intent = IntentUnknown
	}

	return result, nil
}

// llmResponse is the expected JSON structure from LLM.
type llmResponse struct {
	Intent     string  `json:"intent"`
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"`
}

// parseResponse parses the LLM JSON response.
func (c *LLMClassifier) parseResponse(response string) (*LLMClassifyResult, error) {
	// Clean up response - extract JSON if surrounded by markdown
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var jsonLines []string
		inJSON := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inJSON = !inJSON
				continue
			}
			if inJSON {
				jsonLines = append(jsonLines, line)
			}
		}
		response = strings.Join(jsonLines, "\n")
	}

	var resp llmResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		return nil, err
	}

	intent := c.stringToIntent(resp.Intent)

	return &LLMClassifyResult{
		Intent:     intent,
		Confidence: float32(resp.Confidence),
		Reasoning:  resp.Reasoning,
	}, nil
}

// stringToIntent converts string to Intent type.
func (c *LLMClassifier) stringToIntent(s string) Intent {
	switch strings.ToLower(s) {
	case "memo_search":
		return IntentMemoSearch
	case "memo_create":
		return IntentMemoCreate
	case "schedule_query":
		return IntentScheduleQuery
	case "schedule_create":
		return IntentScheduleCreate
	case "schedule_update":
		return IntentScheduleUpdate
	case "batch_schedule":
		return IntentBatchSchedule
	case "amazing":
		return IntentAmazing
	default:
		return IntentUnknown
	}
}
