package v1

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	pluginai "github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents/orchestrator"
	"github.com/hrygo/divinesense/ai/agents/tools"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/enrichment"
	"github.com/hrygo/divinesense/ai/routing"
	aistats "github.com/hrygo/divinesense/ai/services/stats"
	"github.com/hrygo/divinesense/ai/tags"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/server/middleware"
	aichat "github.com/hrygo/divinesense/server/router/api/v1/ai"
	"github.com/hrygo/divinesense/store"
	dbpostgres "github.com/hrygo/divinesense/store/db/postgres"
)

// Global AI rate limiter.
var globalAILimiter = middleware.NewRateLimiter()

// embeddingProviderAdapter adapts pluginai.EmbeddingService to routing.EmbeddingProvider.
type embeddingProviderAdapter struct {
	service pluginai.EmbeddingService
}

func (a *embeddingProviderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.service.Embed(ctx, text)
}

// newWeightStorageAdapter creates a weight storage adapter from a store.
func newWeightStorageAdapter(st *store.Store) routing.RouterWeightStorage {
	// Try to get the postgres driver
	if st != nil {
		driver := st.GetDriver()
		if db, ok := driver.(*dbpostgres.DB); ok {
			return routing.NewPostgresWeightStorage(db)
		}
	}
	// Fallback to in-memory storage
	return routing.NewInMemoryWeightStorage()
}

// Default history retention count for router memory service.
const DefaultHistoryRetention = 10

// AIService provides AI-powered features for memo management.
type AIService struct {
	v1pb.UnimplementedAIServiceServer
	RerankerService          pluginai.RerankerService
	EmbeddingService         pluginai.EmbeddingService
	LLMService               pluginai.LLMService
	IntentLLMService         pluginai.LLMService // Simple tasks: title, summary, tags
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
	persister                *aistats.Persister  // session stats async persister
	enrichmentTrigger        *enrichment.Trigger // Async enrichment trigger
	routerServiceMu          sync.RWMutex
	chatEventBusMu           sync.RWMutex
	contextBuilderMu         sync.RWMutex
	conversationSummarizerMu sync.RWMutex
	agentFactoryMu           sync.RWMutex
	enrichmentTriggerMu      sync.RWMutex
	mu                       sync.RWMutex
}

// Close gracefully shuts down the AI service, including the persister and router service.
func (s *AIService) Close(timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	// Shutdown enrichment trigger (waits for background goroutines)
	if s.enrichmentTrigger != nil {
		s.enrichmentTrigger.Stop()
		s.enrichmentTrigger = nil
	}

	// Shutdown router service (waits for background goroutines)
	if s.routerService != nil {
		s.routerService.Shutdown()
	}

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

	// Build config-driven capability map from expert registry
	var capabilityMap routing.KeywordCapabilitySource
	var routingMatcher routing.RoutingMatcher
	var semanticMatcher routing.SemanticMatcher

	if factory := s.getAgentFactory(); factory != nil {
		// Get expert configurations from factory
		expertConfigs := factory.GetSelfCognitionConfigs()

		if len(expertConfigs) > 0 {
			// Build CapabilityMap from expert configs
			cm := orchestrator.NewCapabilityMap()
			cm.BuildFromConfigs(expertConfigs)
			// Build keyword index for Layer 2 rule-based routing
			cm.BuildKeywordIndex(expertConfigs)

			// Build semantic index for Layer 3 semantic routing (if embedding service available)
			if s.EmbeddingService != nil {
				provider := &embeddingProviderAdapter{service: s.EmbeddingService}
				cm.BuildSemanticIndex(context.Background(), expertConfigs, provider)
				semanticMatcher = cm // CapabilityMap implements SemanticMatcher interface
				slog.Info("semantic index initialized for routing")
			}

			// Set expert resolver for HandoffHandler (supports fuzzy matching)
			tools.SetExpertResolver(cm)

			capabilityMap = cm
			routingMatcher = cm // CapabilityMap implements RoutingMatcher interface
		}
	}

	// FastRouter: cache -> rule (no LLM layer)
	// Complex/low-confidence requests are handled by Orchestrator
	weightStorage := newWeightStorageAdapter(s.Store)
	s.routerService = routing.NewService(routing.Config{
		EnableCache:     true,
		WeightStorage:   weightStorage,
		EnableFeedback:  true,
		CapabilityMap:   capabilityMap,
		RoutingMatcher:  routingMatcher,
		SemanticMatcher: semanticMatcher,
	})

	return s.routerService
}

// RecordRouterFeedback records user feedback for routing decisions.
// This enables HILT (Human-In-The-Loop) learning - the system learns from user corrections.
func (s *AIService) RecordRouterFeedback(ctx context.Context, req *v1pb.RecordRouterFeedbackRequest) (*emptypb.Empty, error) {
	routerSvc := s.getRouterService()
	if routerSvc == nil {
		return nil, fmt.Errorf("router service not available")
	}

	userID := auth.GetUserID(ctx)

	// Convert string to routing.FeedbackType
	var fbType routing.FeedbackType
	switch req.Feedback {
	case "positive":
		fbType = routing.FeedbackPositive
	case "rephrase":
		fbType = routing.FeedbackRephrase
	case "switch":
		fbType = routing.FeedbackSwitch
	default:
		fbType = routing.FeedbackPositive
	}

	feedback := &routing.RouterFeedback{
		UserID:    userID,
		Input:     req.Input,
		Predicted: routing.Intent(req.Predicted),
		Actual:    routing.Intent(req.Actual),
		Feedback:  fbType,
		Timestamp: time.Now().Unix(),
		Source:    "user_feedback",
	}

	if err := routerSvc.RecordFeedback(ctx, feedback); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
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

// getEnrichmentTrigger returns the enrichment trigger, initializing it on first use.
// Thread-safe: uses RWMutex for lazy initialization.
func (s *AIService) getEnrichmentTrigger() *enrichment.Trigger {
	// Fast path: read lock
	s.enrichmentTriggerMu.RLock()
	if s.enrichmentTrigger != nil {
		s.enrichmentTriggerMu.RUnlock()
		return s.enrichmentTrigger
	}
	s.enrichmentTriggerMu.RUnlock()

	// Slow path: write lock for initialization
	s.enrichmentTriggerMu.Lock()
	defer s.enrichmentTriggerMu.Unlock()

	// Double-check after acquiring write lock
	if s.enrichmentTrigger != nil {
		return s.enrichmentTrigger
	}

	// Create enrichers
	var enrichers []enrichment.Enricher

	// Use IntentLLMService for simple tasks (summary, tags, title)
	// Falls back to LLMService if IntentLLMService is not configured
	llmForEnrichment := s.IntentLLMService
	if llmForEnrichment == nil {
		llmForEnrichment = s.LLMService
	}

	// Add summary enricher if LLM is available
	if llmForEnrichment != nil {
		enrichers = append(enrichers, enrichment.NewSummaryEnricher(llmForEnrichment))
	}

	// Add tags enricher if store AND LLM are available
	if s.Store != nil && llmForEnrichment != nil {
		suggester := tags.NewTagSuggester(s.Store, llmForEnrichment, nil)
		enrichers = append(enrichers, enrichment.NewTagsEnricher(suggester))
	}

	// Add title enricher if LLM is available
	if llmForEnrichment != nil {
		enrichers = append(enrichers, enrichment.NewTitleEnricher(llmForEnrichment))
	}

	// Create pipeline and trigger
	pipeline := enrichment.NewPipeline(enrichers...)
	trigger := enrichment.NewTrigger(pipeline, 3) // 3 workers
	trigger.Start()

	s.enrichmentTrigger = trigger
	slog.Info("Enrichment trigger initialized", "enrichers", len(enrichers))
	return s.enrichmentTrigger
}

// TriggerEnrichment triggers async enrichment for a memo.
// This is called after memo creation/update to generate summary, tags, title.
func (s *AIService) TriggerEnrichment(memoID string, content string, title string, userID int32) {
	trigger := s.getEnrichmentTrigger()
	if trigger == nil {
		return
	}
	trigger.TriggerAsync(&enrichment.MemoContent{
		MemoID:  memoID,
		Content: content,
		Title:   title,
		UserID:  userID,
	})
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
