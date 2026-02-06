package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Message represents a chat message.
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// LLMCallStats represents statistics for a single LLM call.
// This provides token usage and timing metrics for session summary and cost tracking.
type LLMCallStats struct {
	// PromptTokens is the number of tokens in the input prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the generated response.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int `json:"total_tokens"`

	// CacheReadTokens is the number of tokens read from cache (for providers that support it).
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`

	// CacheWriteTokens is the number of tokens written to cache.
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`

	// ThinkingDurationMs is the time from request start to first chunk (TTFT - Time To First Token).
	// For non-streaming requests, this is the total request duration.
	ThinkingDurationMs int64 `json:"thinking_duration_ms"`

	// GenerationDurationMs is the time spent generating the response content.
	// For streaming, this is from first chunk to last chunk. For non-streaming, this is 0.
	GenerationDurationMs int64 `json:"generation_duration_ms,omitempty"`

	// TotalDurationMs is the total wall-clock time for the request.
	TotalDurationMs int64 `json:"total_duration_ms"`
}

// LLMService is the LLM service interface.
type LLMService interface {
	// Chat performs synchronous chat. Returns content, statistics, and error.
	Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error)

	// ChatStream performs streaming chat. Returns content channel, stats channel, and error channel.
	// The stats channel is closed after sending the final stats when stream completes.
	ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error)

	// ChatWithTools performs chat with function calling support. Returns response, statistics, and error.
	ChatWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (*ChatResponse, *LLMCallStats, error)
}

// ToolDescriptor represents a function/tool available to the LLM.
type ToolDescriptor struct {
	Name        string
	Description string
	Parameters  string // JSON Schema string
}

// ChatResponse represents the LLM response including potential tool calls.
type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolCall represents a request to call a tool.
type ToolCall struct {
	ID       string
	Type     string
	Function FunctionCall
}

// FunctionCall represents the function details.
type FunctionCall struct {
	Name      string
	Arguments string
}

type llmService struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float32
}

// newHTTPClient creates a custom HTTP client with appropriate timeouts.
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// NewLLMService creates a new LLMService.
func NewLLMService(cfg *LLMConfig) (LLMService, error) {
	var clientConfig openai.ClientConfig

	// Create custom HTTP client with timeout
	httpClient := newHTTPClient()

	switch cfg.Provider {
	case "deepseek":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.deepseek.com"
		}
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		clientConfig.BaseURL = baseURL
		clientConfig.HTTPClient = httpClient

	case "openai":
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		if cfg.BaseURL != "" {
			clientConfig.BaseURL = cfg.BaseURL
		}
		clientConfig.HTTPClient = httpClient

	case "siliconflow":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.siliconflow.cn/v1"
		}
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		clientConfig.BaseURL = baseURL
		clientConfig.HTTPClient = httpClient

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &llmService{
		client:      client,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
	}, nil
}

func (s *llmService) Chat(ctx context.Context, messages []Message) (string, *LLMCallStats, error) {
	// Add timeout protection - shorter than HTTP client timeout for context cancellation
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	slog.Debug("LLM: Chat request",
		"model", s.model,
		"messages_count", len(messages),
		"max_tokens", s.maxTokens,
	)

	startTime := time.Now()

	req := openai.ChatCompletionRequest{
		Model:       s.model,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
		Messages:    convertMessages(messages),
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		slog.Error("LLM: Chat request failed", "error", err)
		return "", nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		slog.Warn("LLM: Empty response from LLM")
		return "", nil, fmt.Errorf("empty response from LLM")
	}

	totalDuration := time.Since(startTime)

	// Extract token usage from response
	stats := &LLMCallStats{
		PromptTokens:       resp.Usage.PromptTokens,
		CompletionTokens:   resp.Usage.CompletionTokens,
		TotalTokens:        resp.Usage.TotalTokens,
		ThinkingDurationMs: totalDuration.Milliseconds(),
		TotalDurationMs:    totalDuration.Milliseconds(),
	}

	// Handle cached tokens (provider-specific, mostly OpenAI)
	if resp.Usage.PromptTokensDetails != nil && resp.Usage.PromptTokensDetails.CachedTokens > 0 {
		stats.CacheReadTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	slog.Debug("LLM: Chat response received",
		"content_length", len(resp.Choices[0].Message.Content),
		"total_tokens", stats.TotalTokens,
		"duration_ms", totalDuration.Milliseconds(),
	)

	return resp.Choices[0].Message.Content, stats, nil
}

func (s *llmService) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDescriptor) (*ChatResponse, *LLMCallStats, error) {
	// Add timeout protection
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	openaiTools := make([]openai.Tool, len(tools))
	for i, t := range tools {
		// Parse JSON schema definition for parameters
		// Assuming Parameters is already a valid JSON Schema string
		// Note: user must provide "type": "object" wrapper if not present?
		// LangChainGo usually provides full schema.
		// We'll treat it as json.RawMessage if possible, but go-openai expects struct or map?
		// go-openai Parameters is interface{}.

		// For simplicity, we assume we need to unmarshal the string into something generic
		// or pass it directly if library supports it.
		// Looking at go-openai: FunctionDefinition.Parameters is interface{}.
		// We should unmarshal the JSON string to map[string]interface{}.

		openaiTools[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  json.RawMessage(t.Parameters),
			},
		}
	}

	// Use lower temperature for tool calls to ensure consistent, deterministic behavior
	// High temperature (0.7) causes the model to be creative and inconsistent with tool formats
	// Low temperature (0.1) ensures the model follows the tool calling format correctly
	toolCallTemperature := float32(0.1)
	if s.temperature < 0.1 {
		toolCallTemperature = s.temperature // Respect even lower temperature if configured
	}

	startTime := time.Now()

	req := openai.ChatCompletionRequest{
		Model:       s.model,
		MaxTokens:   s.maxTokens,
		Temperature: toolCallTemperature,
		Messages:    convertMessages(messages),
		Tools:       openaiTools,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM chat with tools failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, nil, fmt.Errorf("empty response from LLM")
	}

	totalDuration := time.Since(startTime)

	// Extract token usage from response
	stats := &LLMCallStats{
		PromptTokens:       resp.Usage.PromptTokens,
		CompletionTokens:   resp.Usage.CompletionTokens,
		TotalTokens:        resp.Usage.TotalTokens,
		ThinkingDurationMs: totalDuration.Milliseconds(),
		TotalDurationMs:    totalDuration.Milliseconds(),
	}

	// Handle cached tokens (provider-specific, mostly OpenAI)
	if resp.Usage.PromptTokensDetails != nil && resp.Usage.PromptTokensDetails.CachedTokens > 0 {
		stats.CacheReadTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	choice := resp.Choices[0]
	response := &ChatResponse{
		Content: choice.Message.Content,
	}

	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			response.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return response, stats, nil
}

func (s *llmService) ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan *LLMCallStats, <-chan error) {
	contentChan := make(chan string, 10)
	statsChan := make(chan *LLMCallStats, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(contentChan)
		defer close(statsChan)
		defer close(errChan)

		// Add timeout protection
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// For streaming, enable usage information in the response
		// This is supported by OpenAI-compatible APIs (DeepSeek, SiliconFlow)
		streamOptions := &openai.StreamOptions{
			IncludeUsage: true,
		}

		req := openai.ChatCompletionRequest{
			Model:         s.model,
			MaxTokens:     s.maxTokens,
			Temperature:   s.temperature,
			Messages:      convertMessages(messages),
			StreamOptions: streamOptions,
		}

		startTime := time.Now()
		var firstChunkTime time.Time

		slog.Debug("LLM ChatStream starting", "model", s.model, "messages", len(messages))
		stream, err := s.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			slog.Error("LLM ChatStream failed to create", "error", err)
			select {
			case errChan <- fmt.Errorf("create stream failed: %w", err):
			case <-ctx.Done():
			}
			return
		}
		defer func() { _ = stream.Close() }() //nolint:errcheck // stream cleanup on error

		chunkCount := 0
		var totalTokens int

		for {
			response, err := stream.Recv()
			if err != nil {
				if strings.Contains(err.Error(), "EOF") || err.Error() == "EOF" {
					// Stream completed normally
					totalDuration := time.Since(startTime)
					var generationDurationMs int64
					if !firstChunkTime.IsZero() {
						generationDurationMs = time.Since(firstChunkTime).Milliseconds()
					}

					// Send stats even if usage info is not available in streaming
					stats := &LLMCallStats{
						TotalTokens:          totalTokens,
						ThinkingDurationMs:   firstChunkTime.Sub(startTime).Milliseconds(),
						GenerationDurationMs: generationDurationMs,
						TotalDurationMs:      totalDuration.Milliseconds(),
					}

					slog.Debug("LLM ChatStream completed", "chunks", chunkCount, "duration_ms", totalDuration.Milliseconds())
					statsChan <- stats
					return
				}
				slog.Error("LLM ChatStream receive error", "error", err, "chunks_so_far", chunkCount)
				select {
				case errChan <- fmt.Errorf("stream recv failed: %w", err):
				case <-ctx.Done():
				}
				return
			}

			// Record first chunk time for TTFT (Time To First Token)
			if firstChunkTime.IsZero() && len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
				firstChunkTime = time.Now()
			}

			// Check for final usage information (sent with last chunk when include_usage=true)
			if response.Usage != nil && response.Usage.TotalTokens > 0 {
				totalDuration := time.Since(startTime)
				var generationDurationMs int64
				if !firstChunkTime.IsZero() {
					generationDurationMs = time.Since(firstChunkTime).Milliseconds()
				}

				stats := &LLMCallStats{
					PromptTokens:         response.Usage.PromptTokens,
					CompletionTokens:     response.Usage.CompletionTokens,
					TotalTokens:          response.Usage.TotalTokens,
					ThinkingDurationMs:   firstChunkTime.Sub(startTime).Milliseconds(),
					GenerationDurationMs: generationDurationMs,
					TotalDurationMs:      totalDuration.Milliseconds(),
				}

				// Handle cached tokens if available
				if response.Usage.PromptTokensDetails != nil && response.Usage.PromptTokensDetails.CachedTokens > 0 {
					stats.CacheReadTokens = response.Usage.PromptTokensDetails.CachedTokens
				}

				slog.Debug("LLM ChatStream finished with usage",
					"reason", response.Choices[0].FinishReason,
					"chunks", chunkCount,
					"total_tokens", stats.TotalTokens,
					"duration_ms", totalDuration.Milliseconds(),
				)

				// Send stats before closing
				statsChan <- stats
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta.Content
			if delta != "" {
				chunkCount++
				select {
				case contentChan <- delta:
				case <-ctx.Done():
					slog.Warn("LLM ChatStream context cancelled during send", "chunks", chunkCount)
					return
				}
			}

			// Check if stream is finished (no usage info available)
			if response.Choices[0].FinishReason != "" {
				totalDuration := time.Since(startTime)
				var generationDurationMs int64
				if !firstChunkTime.IsZero() {
					generationDurationMs = time.Since(firstChunkTime).Milliseconds()
				}

				// Estimate tokens if usage info not available (rough estimate: ~4 chars per token)
				estimatedTokens := chunkCount * 10 // rough estimate
				var thinkingDurationMs int64
				if !firstChunkTime.IsZero() {
					thinkingDurationMs = firstChunkTime.Sub(startTime).Milliseconds()
				}
				stats := &LLMCallStats{
					TotalTokens:          estimatedTokens,
					ThinkingDurationMs:   thinkingDurationMs,
					GenerationDurationMs: generationDurationMs,
					TotalDurationMs:      totalDuration.Milliseconds(),
				}

				slog.Debug("LLM ChatStream finished (no usage)",
					"reason", response.Choices[0].FinishReason,
					"chunks", chunkCount,
					"duration_ms", totalDuration.Milliseconds(),
				)

				statsChan <- stats
				return
			}
		}
	}()

	return contentChan, statsChan, errChan
}

func convertMessages(messages []Message) []openai.ChatCompletionMessage {
	llmMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		switch m.Role {
		case "system":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: m.Content,
			}
		case "user":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			}
		case "assistant":
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: m.Content,
			}
		default:
			// Default to user for unknown roles
			llmMessages[i] = openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			}
		}
	}
	return llmMessages
}

// Helper for creating system prompts.
func SystemPrompt(content string) Message {
	return Message{Role: "system", Content: content}
}

// Helper for creating user messages.
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// Helper for creating assistant messages.
func AssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// FormatMessages formats messages for prompt templates.
func FormatMessages(systemPrompt string, userContent string, history []Message) []Message {
	messages := []Message{}
	if systemPrompt != "" {
		messages = append(messages, SystemPrompt(systemPrompt))
	}
	messages = append(messages, history...)
	messages = append(messages, UserMessage(userContent))
	return messages
}
