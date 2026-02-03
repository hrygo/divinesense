package v1

import (
	"context"
	"fmt"
	"sync"
	"time"

	pluginai "github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/memory"
	"github.com/hrygo/divinesense/ai/router"
	aistats "github.com/hrygo/divinesense/ai/stats"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/server/middleware"
	aichat "github.com/hrygo/divinesense/server/router/api/v1/ai"
	"github.com/hrygo/divinesense/store"
)

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
	routerService            *router.Service
	chatEventBus             *aichat.EventBus
	Store                    *store.Store
	contextBuilder           *aichat.ContextBuilder
	conversationSummarizer   *aichat.ConversationSummarizer
	EmbeddingModel           string
	persister                *aistats.Persister // session stats async persister
	routerServiceMu          sync.RWMutex
	chatEventBusMu           sync.RWMutex
	contextBuilderMu         sync.RWMutex
	conversationSummarizerMu sync.RWMutex
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
func (s *AIService) getRouterService() *router.Service {
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
	var llmClient router.LLMClient
	if s.LLMService != nil {
		llmClient = &routerLLMClient{llm: s.LLMService}
	}

	s.routerService = router.NewService(router.Config{
		MemoryService: memService,
		LLMClient:     llmClient,
	})

	return s.routerService
}

// routerLLMClient adapts LLMService to router.LLMClient interface.
type routerLLMClient struct {
	llm pluginai.LLMService
}

func (c *routerLLMClient) Complete(ctx context.Context, prompt string, config router.ModelConfig) (string, error) {
	// Convert router request to LLM chat
	messages := []pluginai.Message{
		{Role: "system", Content: "You are an intent classifier. Respond only with the intent type."},
		{Role: "user", Content: prompt},
	}
	// Apply model configuration for the LLM call
	// Note: Currently the LLM service uses global configuration, but config.MaxTokens
	// and config.Temperature are available here for future per-request configuration.
	return c.llm.Chat(ctx, messages)
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
