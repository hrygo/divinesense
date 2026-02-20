// Package v1 provides the ChatAppService handlers for chat app integrations.
package v1

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
	"github.com/hrygo/divinesense/plugin/chat_apps/metrics"
	chatstore "github.com/hrygo/divinesense/plugin/chat_apps/store"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/server/auth"
	aichat "github.com/hrygo/divinesense/server/router/api/v1/ai"
	"github.com/hrygo/divinesense/store"
)

type ChatAppService struct {
	v1pb.UnimplementedChatAppServiceServer
	Store             *store.Store
	Secret            string
	Profile           *profile.Profile
	AIService         *AIService
	chatChannelRouter *channels.ChannelRouter
	chatAppStore      *chatstore.ChatAppStore
}

// contextKey is a typed context key to prevent collisions.
// Using private struct type prevents other packages from colliding with our key.
type contextKey struct{}

// userIDContextKey is the context key for user ID.
var userIDContextKey = contextKey{}

// RegisterCredential binds a chat app account to the current user.
func (s *ChatAppService) RegisterCredential(ctx context.Context, request *v1pb.RegisterCredentialRequest) (*v1pb.Credential, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Validate platform
	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}
	if request.PlatformUserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "platform_user_id is required")
	}

	// Validate required fields per platform
	platform := convertPlatformFromProto(request.Platform)
	if !platform.IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid platform")
	}

	// Platform-specific validation
	switch platform {
	case chat_apps.PlatformTelegram:
		if request.AccessToken == "" {
			return nil, status.Errorf(codes.InvalidArgument, "access_token (bot token) is required for Telegram")
		}
	case chat_apps.PlatformDingTalk:
		if request.AccessToken == "" {
			return nil, status.Errorf(codes.InvalidArgument, "access_token (app secret) is required for DingTalk")
		}
	}

	// Get the chat app store
	chatAppStore := s.getChatAppStore()

	// Encrypt the access token before storing
	encryptedToken, err := s.encryptAccessToken(request.AccessToken)
	if err != nil {
		slog.Error("failed to encrypt access token", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to secure access token")
	}

	// Encrypt the app secret (for platforms like DingTalk)
	encryptedAppSecret, err := s.encryptAccessToken(request.AppSecret)
	if err != nil {
		slog.Error("failed to encrypt app secret", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to secure app secret")
	}

	// Create credential
	createReq := &chatstore.CreateCredentialRequest{
		UserID:         userID,
		Platform:       platform,
		PlatformUserID: request.PlatformUserId,
		PlatformChatID: request.PlatformChatId,
		AccessToken:    encryptedToken,
		AppSecret:      encryptedAppSecret,
		WebhookURL:     request.WebhookUrl,
	}

	cred, err := chatAppStore.CreateCredential(ctx, createReq)
	if err != nil {
		slog.Error("failed to create credential", "user_id", userID, "platform", platform, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create credential")
	}

	slog.Info("chat app credential registered",
		"user_id", userID,
		"platform", platform,
		"platform_user_id", request.PlatformUserId,
	)

	return convertCredentialToProto(cred), nil
}

// ListCredentials returns all registered chat app credentials for the current user.
func (s *ChatAppService) ListCredentials(ctx context.Context, request *v1pb.ListCredentialsRequest) (*v1pb.ListCredentialsResponse, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	chatAppStore := s.getChatAppStore()
	platformFilter := convertPlatformFromProto(request.Platform)

	credentials, err := chatAppStore.ListCredentials(ctx, userID, platformFilter)
	if err != nil {
		slog.Error("failed to list credentials", "user_id", userID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list credentials")
	}

	var protoCredentials []*v1pb.Credential
	for _, cred := range credentials {
		protoCredentials = append(protoCredentials, convertCredentialToProto(cred))
	}

	return &v1pb.ListCredentialsResponse{
		Credentials: protoCredentials,
	}, nil
}

// DeleteCredential removes a chat app binding for the current user.
func (s *ChatAppService) DeleteCredential(ctx context.Context, request *v1pb.DeleteCredentialRequest) (*emptypb.Empty, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}

	platform := convertPlatformFromProto(request.Platform)
	chatAppStore := s.getChatAppStore()

	// First, get the credential to verify ownership
	cred, err := chatAppStore.GetCredentialByPlatform(ctx, userID, platform)
	if err != nil {
		slog.Error("failed to get credential", "user_id", userID, "platform", platform, "error", err)
		return nil, status.Errorf(codes.NotFound, "credential not found")
	}

	// Verify ownership
	if cred.UserID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not your credential")
	}

	// Delete the credential
	if err := chatAppStore.DeleteCredential(ctx, cred.ID); err != nil {
		slog.Error("failed to delete credential", "id", cred.ID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete credential")
	}

	slog.Info("chat app credential deleted",
		"user_id", userID,
		"platform", platform,
	)

	return &emptypb.Empty{}, nil
}

// UpdateCredential modifies an existing credential.
func (s *ChatAppService) UpdateCredential(ctx context.Context, request *v1pb.UpdateCredentialRequest) (*v1pb.Credential, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}

	platform := convertPlatformFromProto(request.Platform)
	chatAppStore := s.getChatAppStore()

	// Get existing credential
	cred, err := chatAppStore.GetCredentialByPlatform(ctx, userID, platform)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "credential not found")
	}

	// Verify ownership
	if cred.UserID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "not your credential")
	}

	// Prepare update request
	updateReq := &chatstore.UpdateCredentialRequest{
		ID: cred.ID,
	}

	if request.AccessToken != nil {
		encryptedToken, err := s.encryptAccessToken(*request.AccessToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to secure access token")
		}
		updateReq.AccessToken = &encryptedToken
	}

	if request.AppSecret != nil {
		encryptedSecret, err := s.encryptAccessToken(*request.AppSecret)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to secure app secret")
		}
		updateReq.AppSecret = &encryptedSecret
	}

	if request.WebhookUrl != nil {
		updateReq.WebhookURL = request.WebhookUrl
	}

	// Note: enabled field is handled separately via SetEnabled
	if request.Enabled != nil {
		if err := chatAppStore.SetEnabled(ctx, cred.ID, *request.Enabled); err != nil {
			slog.Error("failed to set enabled state", "id", cred.ID, "enabled", *request.Enabled, "error", err)
			return nil, status.Errorf(codes.Internal, "failed to update credential")
		}
	}

	updatedCred, err := chatAppStore.UpdateCredential(ctx, updateReq)
	if err != nil {
		slog.Error("failed to update credential", "id", cred.ID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update credential")
	}

	slog.Info("chat app credential updated",
		"user_id", userID,
		"platform", platform,
	)

	return convertCredentialToProto(updatedCred), nil
}

// HandleWebhook processes incoming webhook events from chat platforms.
func (s *ChatAppService) HandleWebhook(ctx context.Context, request *v1pb.WebhookRequest) (*v1pb.WebhookResponse, error) {
	startTime := time.Now()

	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}

	platform := convertPlatformFromProto(request.Platform)

	// Get the channel for this platform
	channelRegistry := s.getChannelRegistry()
	channel := channelRegistry.GetChannel(platform)
	if channel == nil {
		slog.Warn("no channel registered for platform", "platform", platform)
		return &v1pb.WebhookResponse{
			Success: false,
			Message: fmt.Sprintf("platform %s not configured", platform),
		}, nil
	}

	// Prepare headers map
	headers := make(map[string]string)
	for k, v := range request.Headers {
		headers[k] = v
	}
	// Add query string for DingTalk signature validation
	headers["Query-String"] = request.QueryString

	// Validate webhook
	if err := channel.ValidateWebhook(ctx, headers, request.Payload); err != nil {
		slog.Warn("webhook validation failed",
			"platform", platform,
			"error", err,
		)
		return &v1pb.WebhookResponse{
			Success: false,
			Message: "webhook validation failed",
		}, nil
	}

	// Parse message
	msg, err := channel.ParseMessage(ctx, request.Payload)
	if err != nil {
		slog.Warn("failed to parse webhook message",
			"platform", platform,
			"error", err,
		)
		registry := metrics.GetRegistry()
		registry.RecordEvent(string(platform), 0, metrics.EventWebhookParseError, 0, err)
		return &v1pb.WebhookResponse{
			Success: false,
			Message: "failed to parse message",
		}, nil
	}

	// Look up user credential by platform user ID
	chatAppStore := s.getChatAppStore()
	cred, err := chatAppStore.GetCredentialByPlatformUserID(ctx, platform, msg.PlatformUserID)
	if err != nil {
		slog.Warn("no credential found for platform user",
			"platform", platform,
			"platform_user_id", msg.PlatformUserID,
			"error", err,
		)
		return &v1pb.WebhookResponse{
			Success: false,
			Message: "user not bound",
		}, nil
	}

	if !cred.Enabled {
		slog.Info("credential is disabled",
			"user_id", cred.UserID,
			"platform", platform,
		)
		return &v1pb.WebhookResponse{
			Success: false,
			Message: "credential disabled",
		}, nil
	}

	// Record webhook received with correct credID
	registry := metrics.GetRegistry()
	registry.RecordEvent(string(platform), cred.ID, metrics.EventWebhookReceived, 0, nil)
	registry.RecordEvent(string(platform), cred.ID, metrics.EventWebhookValidated, 0, nil)

	// Process message asynchronously - don't block webhook response
	// NOTE: This uses an in-memory goroutine without persistence.
	// If the server restarts, pending messages will be lost.
	// For production, this should be replaced with a persistent task queue (e.g. Redis/Postgres).
	go s.processChatAppMessage(context.Background(), cred, msg, platform, startTime, registry)

	// Immediately acknowledge receipt (webhooks should respond quickly)
	return &v1pb.WebhookResponse{
		Success: true,
		Message: "message received",
	}, nil
}

// processChatAppMessage handles the actual message processing and AI routing.
// Accepts optional metrics tracking (startTime and registry can be zero/nil).
func (s *ChatAppService) processChatAppMessage(
	ctx context.Context,
	cred *chat_apps.Credential,
	msg *chat_apps.IncomingMessage,
	platform chat_apps.Platform,
	startTime time.Time,
	registry *metrics.Registry,
) {
	// Set the user ID in context for AI processing
	ctx = context.WithValue(ctx, userIDContextKey, cred.UserID)

	slog.Info("processing chat app message",
		"user_id", cred.UserID,
		"platform", platform,
		"platform_user_id", msg.PlatformUserID,
		"content", msg.Content,
	)

	// Get the channel for sending response
	channelRegistry := s.getChannelRegistry()
	channel := channelRegistry.GetChannel(platform)
	if channel == nil {
		slog.Warn("no channel available for response",
			"platform", platform,
			"user_id", cred.UserID,
		)
		if registry != nil {
			registry.RecordEvent(string(platform), cred.ID, metrics.EventResponseError, time.Since(startTime), fmt.Errorf("no channel available"))
		}
		return
	}

	// Route to AI agent and send response back with optional metrics
	s.routeAndSendAIResponse(ctx, cred, msg, platform, channel, startTime, registry)
}

// routeAndSendAIResponse routes the message to AI and sends the response back.
// Uses streaming for better UX with long AI responses.
// Accepts optional metrics tracking (startTime and registry can be zero/nil).
func (s *ChatAppService) routeAndSendAIResponse(
	ctx context.Context,
	cred *chat_apps.Credential,
	msg *chat_apps.IncomingMessage,
	platform chat_apps.Platform,
	channel channels.ChatChannel,
	startTime time.Time,
	registry *metrics.Registry,
) {
	// Build prompt for AI
	prompt := s.buildAIPrompt(msg)

	// Use streaming response for better UX
	s.sendStreamingResponse(ctx, cred, msg, platform, channel, prompt)

	// Record metrics if registry provided
	if registry != nil {
		registry.RecordEvent(string(platform), cred.ID, metrics.EventResponseSent, time.Since(startTime), nil)
	}
}

// buildAIPrompt builds a prompt for the AI based on the incoming message.
func (s *ChatAppService) buildAIPrompt(msg *chat_apps.IncomingMessage) string {
	// Build a simple prompt with context
	prompt := msg.Content
	if msg.Type == chat_apps.MessageTypePhoto || msg.Type == chat_apps.MessageTypeVideo {
		prompt = "[用户发送了一张图片/视频，请询问用户希望我如何处理]"
	}
	return prompt
}

// sendSimpleResponse sends a simple text response to the chat platform.
func (s *ChatAppService) sendSimpleResponse(
	ctx context.Context,
	cred *chat_apps.Credential,
	msg *chat_apps.IncomingMessage,
	platform chat_apps.Platform,
	channel channels.ChatChannel,
	text string,
) {
	outgoingMsg := &chat_apps.OutgoingMessage{
		PlatformChatID: msg.PlatformChatID,
		Type:           chat_apps.MessageTypeText,
		Content:        text,
	}

	if err := channel.SendMessage(ctx, outgoingMsg); err != nil {
		slog.Error("failed to send response to chat platform",
			"user_id", cred.UserID,
			"platform", platform,
			"error", err,
		)
		return
	}

	slog.Info("response sent to chat platform",
		"user_id", cred.UserID,
		"platform", platform,
		"platform_chat_id", msg.PlatformChatID,
	)
}

// sendStreamingResponse sends a streaming AI response to the chat platform.
// This uses the SendChunkedMessage interface for better UX with long AI responses.
func (s *ChatAppService) sendStreamingResponse(
	ctx context.Context,
	cred *chat_apps.Credential,
	msg *chat_apps.IncomingMessage,
	platform chat_apps.Platform,
	channel channels.ChatChannel,
	prompt string,
) {
	// Check if AI is enabled
	if s.AIService == nil || !s.AIService.IsLLMEnabled() {
		s.sendSimpleResponse(ctx, cred, msg, platform, channel,
			"AI 功能未启用。请在服务器配置中启用 AI 服务。")
		return
	}

	// Create a channel for streaming chunks
	// Note: Channel is closed by the goroutine, not here
	chunks := make(chan string, 10)

	// Start streaming in background goroutine
	// The goroutine owns the channel and is responsible for closing it
	done := make(chan struct{})
	go func() {
		defer close(chunks)
		defer close(done)

		// Create agent factory
		factory := aichat.NewAgentFactory(
			s.AIService.LLMService,
			s.AIService.AdaptiveRetriever,
			s.Store,
		)

		// Initialize UniversalParrot if configured
		if s.AIService.UniversalParrotConfig != nil {
			if err := factory.Initialize(s.AIService.UniversalParrotConfig); err != nil {
				slog.Error("Failed to initialize AgentFactory for chat app",
					"error", err,
					"user_id", cred.UserID,
					"platform", platform,
				)
				chunks <- "抱歉，AI 服务配置错误。请联系管理员。"
				return
			}
		}

		// Create AUTO agent for intelligent routing
		agent, err := factory.Create(ctx, &aichat.CreateConfig{
			Type:     aichat.AgentTypeAuto,
			UserID:   cred.UserID,
			Timezone: "Asia/Shanghai",
		})
		if err != nil {
			slog.Error("failed to create agent for streaming",
				"user_id", cred.UserID,
				"platform", platform,
				"error", err,
			)
			chunks <- "抱歉，AI 服务暂时不可用。请稍后再试。"
			return
		}

		// Create streaming callback
		streamCallback := func(eventType string, eventData interface{}) error {
			if eventType == agentpkg.EventTypeAnswer {
				if chunk, ok := eventData.(string); ok {
					select {
					case chunks <- chunk:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return nil
		}

		// Execute agent with streaming - use new context variable to avoid shadowing
		agentCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()

		if err := agent.Execute(agentCtx, prompt, nil, streamCallback); err != nil {
			slog.Error("agent streaming failed",
				"user_id", cred.UserID,
				"platform", platform,
				"error", err,
			)
			select {
			case chunks <- "抱歉，AI 服务出现错误。请稍后再试。":
			case <-agentCtx.Done():
			}
		}
	}()

	// Send chunks to chat platform
	// SendChunkedMessage will consume the channel until closed
	if err := channel.SendChunkedMessage(ctx, msg.PlatformChatID, chunks); err != nil {
		slog.Error("failed to send streaming response",
			"user_id", cred.UserID,
			"platform", platform,
			"error", err,
		)
	}

	// Wait for goroutine to finish before returning
	<-done
}

// SendMessage sends a message to a chat app channel.
// This is used internally to deliver AI responses to users.
func (s *ChatAppService) SendMessage(ctx context.Context, request *v1pb.SendMessageRequest) (*emptypb.Empty, error) {
	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}
	if request.PlatformChatId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "platform_chat_id is required")
	}
	if request.Content == "" && len(request.MediaData) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "content or media_data is required")
	}

	platform := convertPlatformFromProto(request.Platform)
	channelRegistry := s.getChannelRegistry()
	channel := channelRegistry.GetChannel(platform)
	if channel == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "platform not configured")
	}

	msg := &chat_apps.OutgoingMessage{
		PlatformChatID: request.PlatformChatId,
		Type:           convertMessageTypeFromProto(request.MessageType),
		Content:        request.Content,
		MediaData:      request.MediaData,
		MimeType:       request.MediaMimeType,
		FileName:       request.FileName,
	}

	if err := channel.SendMessage(ctx, msg); err != nil {
		slog.Error("failed to send message",
			"platform", platform,
			"chat_id", request.PlatformChatId,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to send message")
	}

	slog.Debug("message sent",
		"platform", platform,
		"chat_id", request.PlatformChatId,
	)

	return &emptypb.Empty{}, nil
}

// GetWebhookInfo returns webhook configuration for a platform.
func (s *ChatAppService) GetWebhookInfo(ctx context.Context, request *v1pb.GetWebhookInfoRequest) (*v1pb.WebhookInfo, error) {
	if request.Platform == v1pb.Platform_PLATFORM_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "platform is required")
	}

	baseURL := s.getBaseURL()
	if baseURL == "" {
		baseURL = "https://your-domain.com" // Fallback
	}

	platform := convertPlatformFromProto(request.Platform)
	webhookURL := fmt.Sprintf("%s/api/v1/chat-apps/webhook/%s", baseURL, platform)

	var instructions string
	var requiresVerification bool
	headers := make(map[string]string)

	switch platform {
	case chat_apps.PlatformTelegram:
		instructions = `1. Create a bot via @BotFather on Telegram
2. Get the Bot Token from BotFather
3. Set the webhook URL in your DivineSense settings
4. Use this webhook URL: ` + webhookURL + `
5. The bot token is your access_token`
		requiresVerification = false // Telegram doesn't sign webhooks

	case chat_apps.PlatformDingTalk:
		instructions = `1. Create a DingTalk Robot in the DingTalk Open Platform
2. Get the App Key and App Secret
3. Configure the webhook URL in DingTalk Open Platform: ` + webhookURL + `
4. Add the App Secret as your access_token`
		requiresVerification = true
		headers["X-DingTalk-Signature"] = "computed signature"

	case chat_apps.PlatformWhatsApp:
		instructions = `1. Set up the Baileys bridge service
2. Configure the webhook URL in Meta for Developers: ` + webhookURL + `
3. Verify the phone number in WhatsApp Business settings`
		requiresVerification = true
		headers["X-Hub-Signature"] = "SHA256 signature"
	}

	return &v1pb.WebhookInfo{
		WebhookUrl:           webhookURL,
		SetupInstructions:    instructions,
		Headers:              headers,
		RequiresVerification: requiresVerification,
	}, nil
}

// Helper functions

func (s *ChatAppService) getChatAppStore() *chatstore.ChatAppStore {
	return s.chatAppStore
}

func (s *ChatAppService) getChannelRegistry() *channels.ChannelRouter {
	// Return the actual channel router initialized at service startup
	return s.chatChannelRouter
}

func (s *ChatAppService) getBaseURL() string {
	return s.Profile.InstanceURL
}

func (s *ChatAppService) encryptAccessToken(token string) (string, error) {
	// Get encryption key from environment
	secretKey := os.Getenv("DIVINESENSE_CHAT_APPS_SECRET_KEY")
	if secretKey == "" {
		// FAIL FAST - Do not allow plaintext token storage in production
		return "", fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be set for secure token storage")
	}

	// Validate key length - AES-256 requires exactly 32 bytes
	if len(secretKey) != 32 {
		return "", fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be exactly 32 bytes, got %d bytes", len(secretKey))
	}

	return chatstore.EncryptToken(token, secretKey)
}

// Conversion functions

func convertPlatformFromProto(platform v1pb.Platform) chat_apps.Platform {
	switch platform {
	case v1pb.Platform_PLATFORM_TELEGRAM:
		return chat_apps.PlatformTelegram
	case v1pb.Platform_PLATFORM_WHATSAPP:
		return chat_apps.PlatformWhatsApp
	case v1pb.Platform_PLATFORM_DINGTALK:
		return chat_apps.PlatformDingTalk
	default:
		return ""
	}
}

func convertPlatformToProto(platform chat_apps.Platform) v1pb.Platform {
	switch platform {
	case chat_apps.PlatformTelegram:
		return v1pb.Platform_PLATFORM_TELEGRAM
	case chat_apps.PlatformWhatsApp:
		return v1pb.Platform_PLATFORM_WHATSAPP
	case chat_apps.PlatformDingTalk:
		return v1pb.Platform_PLATFORM_DINGTALK
	default:
		return v1pb.Platform_PLATFORM_UNSPECIFIED
	}
}

func convertMessageTypeFromProto(msgType v1pb.MessageType) chat_apps.MessageType {
	switch msgType {
	case v1pb.MessageType_MESSAGE_TYPE_TEXT:
		return chat_apps.MessageTypeText
	case v1pb.MessageType_MESSAGE_TYPE_PHOTO:
		return chat_apps.MessageTypePhoto
	case v1pb.MessageType_MESSAGE_TYPE_AUDIO:
		return chat_apps.MessageTypeAudio
	case v1pb.MessageType_MESSAGE_TYPE_VIDEO:
		return chat_apps.MessageTypeVideo
	case v1pb.MessageType_MESSAGE_TYPE_DOCUMENT:
		return chat_apps.MessageTypeDocument
	default:
		return chat_apps.MessageTypeText
	}
}

func convertCredentialToProto(cred *chat_apps.Credential) *v1pb.Credential {
	return &v1pb.Credential{
		Id:             int32(cred.ID),
		UserId:         cred.UserID,
		Platform:       convertPlatformToProto(cred.Platform),
		PlatformUserId: cred.PlatformUserID,
		PlatformChatId: cred.PlatformChatID,
		Enabled:        cred.Enabled,
		CreatedTs:      cred.CreatedTs,
		UpdatedTs:      cred.UpdatedTs,
		// AccessToken intentionally omitted for security
	}
}
