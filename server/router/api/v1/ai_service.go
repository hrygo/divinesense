package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pluginai "github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/routing"
	"github.com/hrygo/divinesense/ai/services/memory"
	aistats "github.com/hrygo/divinesense/ai/services/stats"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/server/middleware"
	aichat "github.com/hrygo/divinesense/server/router/api/v1/ai"
	"github.com/hrygo/divinesense/store"
	"github.com/sashabaranov/go-openai"
)

// jsonMap is a map[string]any that implements json.Marshaler for OpenAI compatibility.
type jsonMap map[string]any

func (m jsonMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any(m))
}

// Global AI rate limiter.
var globalAILimiter = middleware.NewRateLimiter()

// Default history retention count for router memory service.
const DefaultHistoryRetention = 10

// AIService provides AI-powered features for memo management.
type AIService struct {
	v1pb.UnimplementedAIServiceServer
	RerankerService          pluginai.RerankerService
	EmbeddingService         pluginai.EmbeddingService
	LLMService               pluginai.LLMService
	conversationService      *aichat.ConversationService
	AdaptiveRetriever        *retrieval.AdaptiveRetriever
	IntentClassifierConfig   *pluginai.IntentClassifierConfig
	UniversalParrotConfig    *pluginai.UniversalParrotConfig // Phase 2: Config-driven parrots
	agentFactory             *aichat.AgentFactory            // Cached agent factory
	routerService            *routing.Service
	chatEventBus             *aichat.EventBus
	Store                    *store.Store
	contextBuilder           *aichat.ContextBuilder
	conversationSummarizer   *aichat.ConversationSummarizer
	TitleGenerator           *pluginai.TitleGenerator // Conversation title generator
	EmbeddingModel           string
	persister                *aistats.Persister // session stats async persister
	routerServiceMu          sync.RWMutex
	chatEventBusMu           sync.RWMutex
	contextBuilderMu         sync.RWMutex
	conversationSummarizerMu sync.RWMutex
	agentFactoryMu           sync.RWMutex
	mu                       sync.RWMutex
}

// Close gracefully shuts down the AI service, including the persister.
func (s *AIService) Close(timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	if s.persister != nil {
		if err := s.persister.Close(timeout); err != nil {
			errs = append(errs, fmt.Errorf("persister close failed: %w", err))
		}
		s.persister = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("AIService close errors: %v", errs)
	}
	return nil
}

// IsEnabled returns whether AI features are enabled.
// For basic features (embedding, search), only EmbeddingService is required.
// For Agent features (Memo, Schedule, etc.), both EmbeddingService and LLMService are required.
func (s *AIService) IsEnabled() bool {
	return s.EmbeddingService != nil
}

// IsLLMEnabled returns whether LLM features are enabled (required for Agents).
func (s *AIService) IsLLMEnabled() bool {
	return s.LLMService != nil
}

// getRouterService returns the router service, initializing it on first use.
// Returns nil if Store is not available, which is safe as callers check for nil.
// Thread-safe: uses RWMutex for lazy initialization with support for re-initialization
// when Store becomes available after initial nil check.
func (s *AIService) getRouterService() *routing.Service {
	// Fast path: read lock
	s.routerServiceMu.RLock()
	if s.routerService != nil {
		s.routerServiceMu.RUnlock()
		return s.routerService
	}
	s.routerServiceMu.RUnlock()

	// Slow path: write lock for initialization
	s.routerServiceMu.Lock()
	defer s.routerServiceMu.Unlock()

	// Double-check after acquiring write lock
	if s.routerService != nil {
		return s.routerService
	}

	if s.Store == nil {
		// Store not available, routerService remains nil
		// Next call will retry when Store becomes available
		return nil
	}

	// Create memory service for router
	memService := memory.NewService(s.Store, DefaultHistoryRetention)

	// Create LLM client wrapper for router
	// Use dedicated intent classifier LLM service when available (per recommended strategy)
	var llmClient routing.LLMClient
	if s.IntentClassifierConfig != nil && s.IntentClassifierConfig.Enabled {
		// Use dedicated SiliconFlow + Qwen2.5-7B-Instruct for intent classification
		llmClient = &routerIntentLLMClient{
			apiKey:  s.IntentClassifierConfig.APIKey,
			baseURL: s.IntentClassifierConfig.BaseURL,
			model:   s.IntentClassifierConfig.Model,
		}
	} else if s.LLMService != nil {
		// Fallback to main LLM service
		llmClient = &routerLLMClient{llm: s.LLMService}
	}

	s.routerService = routing.NewService(routing.Config{
		MemoryService: memService,
		LLMClient:     llmClient,
	})

	return s.routerService
}

// getAgentFactory returns the agent factory, initializing it on first use.
// Thread-safe: uses RWMutex for lazy initialization.
func (s *AIService) getAgentFactory() *aichat.AgentFactory {
	// Fast path: read lock
	s.agentFactoryMu.RLock()
	if s.agentFactory != nil {
		s.agentFactoryMu.RUnlock()
		return s.agentFactory
	}
	s.agentFactoryMu.RUnlock()

	// Slow path: write lock for initialization
	s.agentFactoryMu.Lock()
	defer s.agentFactoryMu.Unlock()

	// Double-check after acquiring write lock
	if s.agentFactory != nil {
		return s.agentFactory
	}

	// Create new agent factory
	factory := aichat.NewAgentFactory(
		s.LLMService,
		s.AdaptiveRetriever,
		s.Store,
	)

	// Initialize UniversalParrot if configured
	if s.UniversalParrotConfig != nil && s.UniversalParrotConfig.Enabled {
		if err := factory.Initialize(s.UniversalParrotConfig); err != nil {
			slog.Warn("Failed to initialize AgentFactory, parrot creation may fail",
				"error", err)
		} else {
			slog.Info("AgentFactory initialized successfully")
		}
	} else {
		slog.Info("UniversalParrot not enabled, using legacy agent creation")
	}

	s.agentFactory = factory
	return s.agentFactory
}

// routerLLMClient adapts LLMService to routing.LLMClient interface.
// Used as fallback when intent classifier is not configured.
type routerLLMClient struct {
	llm pluginai.LLMService
}

func (c *routerLLMClient) Complete(ctx context.Context, prompt string, config routing.ModelConfig) (string, error) {
	messages := []pluginai.Message{
		{Role: "system", Content: "You are an intent classifier. Respond only with the intent type."},
		{Role: "user", Content: prompt},
	}
	result, _, err := c.llm.Chat(ctx, messages)
	return result, err
}

// routerIntentLLMClient is a dedicated LLM client for intent classification.
// Uses SiliconFlow + Qwen2.5-7B-Instruct (per recommended strategy).
type routerIntentLLMClient struct {
	apiKey  string
	baseURL string
	model   string
}

func (c *routerIntentLLMClient) Complete(ctx context.Context, prompt string, config routing.ModelConfig) (string, error) {
	// Build classification prompt with JSON schema
	systemPrompt := `You are an intent classifier. Analyze the user input and return a JSON response:
{
  "intent": "memo_search|memo_create|schedule_query|schedule_create|schedule_update|batch_schedule|amazing|unknown",
  "confidence": 0.0-1.0
}

Intent types:
- memo_search: search or find notes
- memo_create: create or record new note
- schedule_query: query or check schedules
- schedule_create: create new schedule
- schedule_update: modify or cancel schedule
- batch_schedule: batch create schedules
- amazing: comprehensive assistance
- unknown: cannot determine`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	// Create OpenAI client
	clientConfig := openai.DefaultConfig(c.apiKey)
	clientConfig.BaseURL = c.baseURL
	client := openai.NewClientWithConfig(clientConfig)

	// Use model from config if provided, otherwise fall back to struct field
	model := config.Model
	if model == "" {
		model = c.model
	}
	if model == "" {
		model = "Qwen/Qwen2.5-7B-Instruct"
	}

	// Build JSON schema for structured response
	jsonSchema := jsonMap{
		"type": "object",
		"properties": map[string]any{
			"intent": map[string]any{
				"type": "string",
				"enum": []any{"memo_search", "memo_create", "schedule_query", "schedule_create", "schedule_update", "batch_schedule", "amazing", "unknown"},
			},
			"confidence": map[string]any{
				"type": "number",
			},
		},
		"required":             []string{"intent", "confidence"},
		"additionalProperties": false,
	}

	// Call LLM with JSON schema
	req := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   50,
		Temperature: 0,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "intent_classification",
				Strict: true,
				Schema: jsonSchema,
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from intent classifier")
	}

	return resp.Choices[0].Message.Content, nil
}

// getCurrentUser gets the authenticated user from context.
func getCurrentUser(ctx context.Context, st *store.Store) (*store.User, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("user not found in context")
	}
	user, err := st.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %d not found", userID)
	}
	return user, nil
}
