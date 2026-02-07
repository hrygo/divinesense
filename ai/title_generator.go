package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sashabaranov/go-openai"
)

// TitleGenerator generates meaningful titles for AI conversations.
// It uses a lightweight LLM (Qwen2.5-7B-Instruct) to analyze the first
// user-AI exchange and generate a concise, descriptive title.
type TitleGenerator struct {
	client *openai.Client
	model  string
}

// TitleGeneratorConfig holds configuration for the title generator.
type TitleGeneratorConfig struct {
	APIKey  string
	BaseURL string
	Model   string // Recommended: Qwen/Qwen2.5-7B-Instruct
}

// GeneratedTitle represents the result of title generation.
type GeneratedTitle struct {
	Title string `json:"title"`
}

// NewTitleGenerator creates a new title generator instance.
func NewTitleGenerator(cfg TitleGeneratorConfig) *TitleGenerator {
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

	return &TitleGenerator{
		client: openai.NewClientWithConfig(clientConfig),
		model:  model,
	}
}

// Generate generates a title based on the conversation content.
// The input should include the user's first message and the AI's response.
func (tg *TitleGenerator) Generate(ctx context.Context, userMessage, aiResponse string) (string, error) {
	// Set timeout for title generation (should be fast)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	prompt := tg.buildPrompt(userMessage, aiResponse)

	// Use low temperature and top_p for stable, predictable output
	// MaxTokens=20 is sufficient for short titles (typically 5-15 chars)
	temperature := float32(0.1) // Very low for high consistency
	topP := float32(0.5)        // Restrict sampling for stability
	maxTokens := 20             // Reduced from 30 to save cost

	req := openai.ChatCompletionRequest{
		Model:       tg.model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		TopP:        topP,   // Value type, not pointer
		Stop:        []string{"\n"}, // Stop at newline for cleaner output
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: titleSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "title_generation",
				Strict: true,
				Schema: titleJSONSchema,
			},
		},
	}

	start := time.Now()
	resp, err := tg.client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		slog.Error("title_generation_failed",
			"model", tg.model,
			"error", err,
			"latency_ms", latency.Milliseconds())
		return "", fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	content := resp.Choices[0].Message.Content
	result, err := tg.parseResponse(content)
	if err != nil {
		slog.Warn("title_generation_parse_failed",
			"model", tg.model,
			"content", content,
			"error", err)
		return "", fmt.Errorf("parse response failed: %w", err)
	}

	slog.Debug("title_generation_success",
		"model", tg.model,
		"title", result.Title,
		"latency_ms", latency.Milliseconds(),
		"tokens_total", resp.Usage.TotalTokens)

	return result.Title, nil
}

// buildPrompt constructs the title generation prompt.
func (tg *TitleGenerator) buildPrompt(userMessage, aiResponse string) string {
	// Truncate if too long to stay within token limits
	maxLen := 500
	if len(userMessage) > maxLen {
		userMessage = userMessage[:maxLen] + "..."
	}
	if len(aiResponse) > maxLen {
		aiResponse = aiResponse[:maxLen] + "..."
	}

	return fmt.Sprintf("用户消息: %s\n\nAI 回复: %s\n\n请为这段对话生成一个简短的标题。", userMessage, aiResponse)
}

// parseResponse parses the LLM JSON response.
func (tg *TitleGenerator) parseResponse(content string) (*GeneratedTitle, error) {
	var result GeneratedTitle
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Validate title
	if result.Title == "" {
		return nil, fmt.Errorf("empty title in response")
	}

	// Truncate to max length (rune-aware for UTF-8)
	// Use []rune to properly handle multi-byte characters (e.g., Chinese)
	if len([]rune(result.Title)) > 50 {
		runes := []rune(result.Title)
		result.Title = string(runes[:50])
	}

	return &result, nil
}

// titleSystemPrompt is the system prompt for title generation.
const titleSystemPrompt = `你是一个专业的对话标题生成助手。你的任务是根据用户和AI的对话内容，生成一个简洁、准确的标题。

要求：
1. 标题长度：3-15个字符（中文）或 3-8个单词（英文）
2. 标题应该反映对话的核心主题
3. 使用简洁的语言，避免使用"关于..."、"讨论了..."等冗余表述
4. 如果是问题，可以直接用问题本身作为标题
5. 如果是任务，可以用任务描述作为标题
6. 保持中立客观的语气

示例：
- 输入: "如何用Go连接PostgreSQL数据库？" -> 输出: "Go连接PostgreSQL"
- 输入: "帮我写一个二分查找算法" -> 输出: "二分查找算法实现"
- 输入: "今天天气怎么样？" -> 输出: "天气查询"
- 输入: "我的日程安排" -> 输出: "日程管理"
`

// titleJSONSchema defines the JSON schema for title generation response.
var titleJSONSchema = &jsonSchema{
	Type: "object",
	Properties: map[string]*jsonSchema{
		"title": {
			Type:        "string",
			Description: "生成的对话标题，3-15个字符",
		},
	},
	Required:             []string{"title"},
	AdditionalProperties: false,
}

// jsonSchema implements json.Marshaler for OpenAI's JSON Schema format.
type jsonSchema struct {
	Properties           map[string]*jsonSchema `json:"properties,omitempty"`
	Type                 string                 `json:"type"`
	Description          string                 `json:"description,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

func (s *jsonSchema) MarshalJSON() ([]byte, error) {
	type alias jsonSchema
	return json.Marshal((*alias)(s))
}

// GenerateTitleFromBlocks generates a title from a slice of blocks.
// This is a convenience method that extracts the first user-AI exchange.
func (tg *TitleGenerator) GenerateTitleFromBlocks(ctx context.Context, blocks []BlockContent) (string, error) {
	var userMessage, aiResponse string

	for _, block := range blocks {
		// Get first user input
		if userMessage == "" && block.UserInput != "" {
			userMessage = block.UserInput
		}
		// Get first AI response
		if aiResponse == "" && block.AssistantContent != "" {
			aiResponse = block.AssistantContent
		}
		// Stop when we have both
		if userMessage != "" && aiResponse != "" {
			break
		}
	}

	if userMessage == "" {
		return "", fmt.Errorf("no user message found in blocks")
	}

	return tg.Generate(ctx, userMessage, aiResponse)
}

// BlockContent represents a simplified block for title generation.
type BlockContent struct {
	UserInput        string
	AssistantContent string
}
