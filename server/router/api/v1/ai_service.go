package v1

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pluginai "github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/routing"
	aistats "github.com/hrygo/divinesense/ai/services/stats"
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

	// FastRouter: cache -> rule (no LLM layer)
	// Complex/low-confidence requests are handled by Orchestrator
	s.routerService = routing.NewService(routing.Config{
		EnableCache: true,
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
