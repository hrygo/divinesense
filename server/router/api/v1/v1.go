package v1

import (
	"context"
	"net/http"
	"time"

	"log/slog"

	"connectrpc.com/connect"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/semaphore"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	aistats "github.com/hrygo/divinesense/ai/services/stats"
	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
	chatstore "github.com/hrygo/divinesense/plugin/chat_apps/store"
	"github.com/hrygo/divinesense/plugin/markdown"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/store"
)

type APIV1Service struct {
	// Domain Services
	UserService             *UserService
	MemoService             *MemoService
	AuthService             *AuthService
	AttachmentService       *AttachmentService
	ShortcutService         *ShortcutService
	InstanceService         *InstanceService
	IdentityProviderService *IdentityProviderService
	ActivityService         *ActivityService
	ChatAppService          *ChatAppService
	AIService               *AIService
	ScheduleService         *ScheduleService

	// Shared Infra
	MarkdownService    markdown.Service
	Profile            *profile.Profile
	Store              *store.Store
	thumbnailSemaphore *semaphore.Weighted
	Secret             string
	chatChannelRouter  *channels.ChannelRouter
	chatAppStore       *chatstore.ChatAppStore
}

func NewAPIV1Service(secret string, profile *profile.Profile, store *store.Store) *APIV1Service {
	markdownService := markdown.NewService(
		markdown.WithTagExtension(),
	)
	service := &APIV1Service{
		Secret:             secret,
		Profile:            profile,
		Store:              store,
		MarkdownService:    markdownService,
		thumbnailSemaphore: semaphore.NewWeighted(3), // Limit to 3 concurrent thumbnail generations
		ScheduleService:    &ScheduleService{Store: store},
		chatChannelRouter:  channels.NewChannelRouter(nil),
		chatAppStore:       chatstore.NewChatAppStore(store.GetDriver().GetDB()),
	}

	// Initialize AI service if enabled
	// AI features are supported on PostgreSQL (with pgvector) and SQLite (with application-layer vector search)
	if profile.IsAIEnabled() && (profile.Driver == "postgres" || profile.Driver == "sqlite") {
		aiConfig := ai.NewConfigFromProfile(profile)
		if err := aiConfig.Validate(); err == nil {
			embeddingService, err := ai.NewEmbeddingService(&aiConfig.Embedding)
			if err == nil {
				rerankerService := ai.NewRerankerService(&aiConfig.Reranker)
				var llmService ai.LLMService
				if aiConfig.LLM.Provider != "" {
					var llmErr error
					llmService, llmErr = ai.NewLLMService(&aiConfig.LLM)
					if llmErr != nil {
						slog.Warn("Failed to initialize LLM service",
							"provider", aiConfig.LLM.Provider,
							"error", llmErr,
							"note", "Agent features will be disabled",
						)
					} else {
						slog.Info("LLM service initialized",
							"provider", aiConfig.LLM.Provider,
							"model", aiConfig.LLM.Model,
						)
						// Warmup LLM connection asynchronously to reduce first-request latency
						// This is best-effort: warmup failures don't affect service startup
						go func() {
							warmupCtx, warmupCancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer warmupCancel()
							// Type assertion to access Warmup method (implementation detail)
							if warmupable, ok := llmService.(interface{ Warmup(ctx context.Context) }); ok {
								warmupable.Warmup(warmupCtx)
							}
						}()
					}
				}

				// 创建自适应检索器
				adaptiveRetriever := retrieval.NewAdaptiveRetriever(store, embeddingService, rerankerService)

				// 创建 session stats 持久化器
				persister := aistats.NewPersister(store.AgentStatsStore, 100, slog.Default())

				// 创建简单任务 LLM 服务（标题生成、摘要、标签等）
				// 使用 Intent 配置 (siliconflow)，未配置时回退到主 LLM
				intentLLMService := ai.NewSimpleTaskLLMService(profile, llmService)

				// 创建会话标题生成器（使用简单任务 LLM 服务）
				var titleGenerator *ai.TitleGenerator
				if intentLLMService != nil {
					titleGenerator = ai.NewTitleGeneratorWithLLM(intentLLMService)
					slog.Info("Title generator initialized with simple task LLM service")
				}

				service.AIService = &AIService{
					Store:                  store,
					EmbeddingService:       embeddingService,
					EmbeddingModel:         aiConfig.Embedding.Model,
					RerankerService:        rerankerService,
					LLMService:             llmService,
					IntentLLMService:       intentLLMService,
					AdaptiveRetriever:      adaptiveRetriever,
					IntentClassifierConfig: &aiConfig.IntentClassifier,
					UniversalParrotConfig:  &aiConfig.UniversalParrot, // Phase 2: Config-driven parrots
					TitleGenerator:         titleGenerator,
					persister:              persister,
				}
				// Warmup router service (build semantic index) asynchronously
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					if rs := service.AIService.getRouterService(); rs != nil {
						slog.Info("Router service warmed up successfully")
					} else {
						slog.Warn("Router service warmup returned nil")
					}
					_ = ctx // Context used for timeout
				}()
				// Initialize ScheduleService with LLM service for natural language parsing
				service.ScheduleService = &ScheduleService{
					Store:      store,
					LLMService: llmService,
				}

			} else {
				slog.Warn("Failed to initialize embedding service", "error", err)
			}
		} else {
			slog.Warn("AI config validation failed", "error", err)
		}
	} else {
		slog.Info("AI features disabled",
			"enabled", profile.IsAIEnabled(),
			"driver", profile.Driver,
		)
	}

	service.UserService = &UserService{Store: store}
	service.MemoService = &MemoService{Store: store, AIService: service.AIService, MarkdownService: markdownService, Profile: profile}
	service.AuthService = &AuthService{Store: store, Secret: secret, Profile: profile}
	service.AttachmentService = &AttachmentService{Store: store, Profile: profile, thumbnailSemaphore: service.thumbnailSemaphore}
	service.ShortcutService = &ShortcutService{Store: store, Profile: profile}
	service.InstanceService = &InstanceService{Store: store, Profile: profile, AIService: service.AIService}
	service.IdentityProviderService = &IdentityProviderService{Store: store}
	service.ActivityService = &ActivityService{Store: store}
	service.ChatAppService = &ChatAppService{Store: store, Secret: secret, Profile: profile, AIService: service.AIService, chatChannelRouter: service.chatChannelRouter, chatAppStore: service.chatAppStore}

	return service
}

// RegisterGateway registers the gRPC-Gateway and Connect handlers with the given Echo instance.
func (s *APIV1Service) RegisterGateway(ctx context.Context, echoServer *echo.Echo) error {
	// Validate chat apps configuration at startup
	if err := validateChatAppsConfig(); err != nil {
		slog.Warn("chat apps configuration invalid, chat apps features will be disabled",
			"error", err,
		)
		// Don't fail startup, just log a warning
	}

	// Auth middleware for gRPC-Gateway - runs after routing, has access to method name.
	// Uses the same PublicMethods config as the Connect AuthInterceptor.
	authenticator := auth.NewAuthenticator(s.Store, s.Secret)
	gatewayAuthMiddleware := func(next runtime.HandlerFunc) runtime.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			ctx := r.Context()

			// Get the RPC method name from context (set by grpc-gateway after routing)
			rpcMethod, ok := runtime.RPCMethod(ctx)

			// Extract credentials from HTTP headers
			authHeader := r.Header.Get("Authorization")

			result := authenticator.Authenticate(ctx, authHeader)

			// Enforce authentication for non-public methods
			// If rpcMethod cannot be determined, allow through, service layer will handle visibility checks
			if result == nil && ok && !IsPublicMethod(rpcMethod) {
				http.Error(w, `{"code": 16, "message": "authentication required"}`, http.StatusUnauthorized)
				return
			}

			// Set context based on auth result (may be nil for public endpoints)
			if result != nil {
				if result.Claims != nil {
					// Access Token V2 - stateless, use claims
					ctx = auth.SetUserClaimsInContext(ctx, result.Claims)
					ctx = context.WithValue(ctx, auth.UserIDContextKey, result.Claims.UserID)
				} else if result.User != nil {
					// PAT - have full user
					ctx = auth.SetUserInContext(ctx, result.User, result.AccessToken)
				}
				r = r.WithContext(ctx)
			}

			next(w, r, pathParams)
		}
	}

	// Create gRPC-Gateway mux with auth middleware.
	gwMux := runtime.NewServeMux(
		runtime.WithMiddlewares(gatewayAuthMiddleware),
	)
	if err := v1pb.RegisterInstanceServiceHandlerServer(ctx, gwMux, s.InstanceService); err != nil {
		return err
	}
	if err := v1pb.RegisterAuthServiceHandlerServer(ctx, gwMux, s.AuthService); err != nil {
		return err
	}
	if err := v1pb.RegisterUserServiceHandlerServer(ctx, gwMux, s.UserService); err != nil {
		return err
	}
	if err := v1pb.RegisterMemoServiceHandlerServer(ctx, gwMux, s.MemoService); err != nil {
		return err
	}
	if err := v1pb.RegisterAttachmentServiceHandlerServer(ctx, gwMux, s.AttachmentService); err != nil {
		return err
	}
	if err := v1pb.RegisterShortcutServiceHandlerServer(ctx, gwMux, s.ShortcutService); err != nil {
		return err
	}
	if err := v1pb.RegisterActivityServiceHandlerServer(ctx, gwMux, s.ActivityService); err != nil {
		return err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandlerServer(ctx, gwMux, s.IdentityProviderService); err != nil {
		return err
	}
	// Register AI service if available
	if s.AIService != nil {
		if err := v1pb.RegisterAIServiceHandlerServer(ctx, gwMux, s.AIService); err != nil {
			return err
		}
	}
	// Register Schedule service
	if err := v1pb.RegisterScheduleServiceHandlerServer(ctx, gwMux, s.ScheduleService); err != nil {
		return err
	}

	// Register ChatAppService
	if err := v1pb.RegisterChatAppServiceHandlerServer(ctx, gwMux, s.ChatAppService); err != nil {
		return err
	}
	gwGroup := echoServer.Group("")
	gwGroup.Use(middleware.CORS())
	handler := echo.WrapHandler(gwMux)

	gwGroup.Any("/api/v1/*", handler)
	gwGroup.Any("/file/*", handler)

	// Connect handlers for browser clients (replaces grpc-web).
	logStacktraces := s.Profile.IsDev()
	connectInterceptors := connect.WithInterceptors(
		NewMetadataInterceptor(), // Convert HTTP headers to gRPC metadata first
		NewLoggingInterceptor(logStacktraces),
		NewRecoveryInterceptor(logStacktraces),
		NewAuthInterceptor(s.Store, s.Secret),
	)
	connectMux := http.NewServeMux()
	connectHandler := NewConnectServiceHandler(s)
	connectHandler.RegisterConnectHandlers(connectMux, connectInterceptors)

	// Wrap with CORS for browser access
	corsHandler := middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: func(_ string) (bool, error) {
			return true, nil
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	})
	connectGroup := echoServer.Group("", corsHandler)
	connectGroup.Any("/memos.api.v1.*", echo.WrapHandler(connectMux))

	// Register metrics routes (direct REST endpoints)
	systemGroup := echoServer.Group("/api/v1/system", corsHandler)
	systemGroup.GET("/metrics/overview", s.GetMetricsOverview)

	// Initialize chat channels from database
	if err := s.ChatAppService.initializeChatChannels(ctx); err != nil {
		slog.Warn("failed to initialize chat channels", "error", err)
		// Don't fail startup if chat channels fail to initialize
	}

	return nil
}
