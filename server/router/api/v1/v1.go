package v1

import (
	"context"
	"net/http"

	"log/slog"

	"connectrpc.com/connect"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/semaphore"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/plugin/markdown"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/store"
)

type APIV1Service struct {
	v1pb.UnimplementedAIServiceServer
	v1pb.UnimplementedScheduleAgentServiceServer
	v1pb.UnimplementedUserServiceServer
	v1pb.UnimplementedScheduleServiceServer
	v1pb.UnimplementedAttachmentServiceServer
	v1pb.UnimplementedShortcutServiceServer
	v1pb.UnimplementedActivityServiceServer
	v1pb.UnimplementedIdentityProviderServiceServer
	v1pb.UnimplementedAuthServiceServer
	v1pb.UnimplementedInstanceServiceServer
	v1pb.UnimplementedMemoServiceServer
	MarkdownService      markdown.Service
	Profile              *profile.Profile
	Store                *store.Store
	AIService            *AIService
	ScheduleService      *ScheduleService
	ScheduleAgentService *ScheduleAgentService
	thumbnailSemaphore   *semaphore.Weighted
	Secret               string
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
	}

	// Initialize AI service if enabled
	if profile.IsAIEnabled() && profile.Driver == "postgres" {
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
					}
				}

				// 创建自适应检索器
				adaptiveRetriever := retrieval.NewAdaptiveRetriever(store, embeddingService, rerankerService)

				service.AIService = &AIService{
					Store:                  store,
					EmbeddingService:       embeddingService,
					EmbeddingModel:         aiConfig.Embedding.Model,
					RerankerService:        rerankerService,
					LLMService:             llmService,
					AdaptiveRetriever:      adaptiveRetriever,
					IntentClassifierConfig: &aiConfig.IntentClassifier,
				}
				// Initialize ScheduleService with LLM service for natural language parsing
				service.ScheduleService = &ScheduleService{
					Store:      store,
					LLMService: llmService,
				}

				// Initialize ScheduleAgentService
				service.ScheduleAgentService = NewScheduleAgentService(store, llmService, profile)
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

	return service
}

// RegisterGateway registers the gRPC-Gateway and Connect handlers with the given Echo instance.
func (s *APIV1Service) RegisterGateway(ctx context.Context, echoServer *echo.Echo) error {
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
	if err := v1pb.RegisterInstanceServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterAuthServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterUserServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterMemoServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterAttachmentServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterShortcutServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterActivityServiceHandlerServer(ctx, gwMux, s); err != nil {
		return err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandlerServer(ctx, gwMux, s); err != nil {
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
	// Register ScheduleAgent service if available
	if s.ScheduleAgentService != nil {
		if err := v1pb.RegisterScheduleAgentServiceHandlerServer(ctx, gwMux, s.ScheduleAgentService); err != nil {
			return err
		}
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

	return nil
}
