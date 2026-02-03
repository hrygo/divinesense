// Package v1 provides the ChatAppService handlers for chat app integrations.
package v1

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
	"github.com/hrygo/divinesense/plugin/chat_apps/store"
	"github.com/hrygo/divinesense/server/auth"
)

// RegisterCredential binds a chat app account to the current user.
func (s *APIV1Service) RegisterCredential(ctx context.Context, request *v1pb.RegisterCredentialRequest) (*v1pb.Credential, error) {
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

	// Create credential
	createReq := &store.CreateCredentialRequest{
		UserID:         userID,
		Platform:       platform,
		PlatformUserID: request.PlatformUserId,
		PlatformChatID: request.PlatformChatId,
		AccessToken:    encryptedToken,
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
func (s *APIV1Service) ListCredentials(ctx context.Context, request *v1pb.ListCredentialsRequest) (*v1pb.ListCredentialsResponse, error) {
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
func (s *APIV1Service) DeleteCredential(ctx context.Context, request *v1pb.DeleteCredentialRequest) (*emptypb.Empty, error) {
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
func (s *APIV1Service) UpdateCredential(ctx context.Context, request *v1pb.UpdateCredentialRequest) (*v1pb.Credential, error) {
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
	updateReq := &store.UpdateCredentialRequest{
		ID: cred.ID,
	}

	if request.AccessToken != nil {
		encryptedToken, err := s.encryptAccessToken(*request.AccessToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to secure access token")
		}
		updateReq.AccessToken = &encryptedToken
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
func (s *APIV1Service) HandleWebhook(ctx context.Context, request *v1pb.WebhookRequest) (*v1pb.WebhookResponse, error) {
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
			Message:  fmt.Sprintf("platform %s not configured", platform),
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
			Message:  "webhook validation failed",
		}, nil
	}

	// Parse message
	msg, err := channel.ParseMessage(ctx, request.Payload)
	if err != nil {
		slog.Warn("failed to parse webhook message",
			"platform", platform,
			"error", err,
		)
		return &v1pb.WebhookResponse{
			Success: false,
			Message:  "failed to parse message",
		}, nil
	}

	// Route to AI agent
	// TODO: Implement actual AI routing through ChatRouter
	slog.Info("webhook message received",
		"platform", platform,
		"platform_user_id", msg.PlatformUserID,
		"platform_chat_id", msg.PlatformChatID,
		"type", msg.Type,
	)

	// For now, just acknowledge receipt
	return &v1pb.WebhookResponse{
		Success: true,
		Message:  "message received",
	}, nil
}

// SendMessage sends a message to a chat app channel.
// This is used internally to deliver AI responses to users.
func (s *APIV1Service) SendMessage(ctx context.Context, request *v1pb.SendMessageRequest) (*emptypb.Empty, error) {
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
func (s *APIV1Service) GetWebhookInfo(ctx context.Context, request *v1pb.GetWebhookInfoRequest) (*v1pb.WebhookInfo, error) {
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
		WebhookUrl:          webhookURL,
		SetupInstructions:   instructions,
		Headers:             headers,
		RequiresVerification: requiresVerification,
	}, nil
}

// Helper functions

func (s *APIV1Service) getChatAppStore() *store.ChatAppStore {
	// Get the underlying database connection from the Store's driver
	// TODO: Inject ChatAppStore into APIV1Service during initialization
	return store.NewChatAppStore(s.Store.GetDriver().GetDB())
}

func (s *APIV1Service) getChannelRegistry() *channelRegistryImpl {
	// In a real implementation, this would be a singleton service
	// For now, return a placeholder
	// TODO: Create and inject ChannelRegistry during service initialization
	return &channelRegistryImpl{
		channels: make(map[chat_apps.Platform]channels.ChatChannel),
	}
}

func (s *APIV1Service) getBaseURL() string {
	// TODO: Get from configuration
	return ""
}

func (s *APIV1Service) encryptAccessToken(token string) (string, error) {
	// Get encryption key from environment
	secretKey := os.Getenv("DIVINESENSE_CHAT_APPS_SECRET_KEY")
	if secretKey == "" {
		// Generate a warning - in production this should be configured
		slog.Warn("DIVINESENSE_CHAT_APPS_SECRET_KEY not set, using insecure storage")
		return token, nil
	}

	// The key needs to be 32 bytes for AES-256
	// If it's base64 encoded, decode it first
	// For now, we'll assume the key is properly configured
	return store.EncryptToken(token, secretKey)
}

// ChannelRegistry manages chat channel instances.
type ChannelRegistry interface {
	GetChannel(platform chat_apps.Platform) channels.ChatChannel
	Register(channel channels.ChatChannel)
}

// channelRegistryImpl is a simple in-memory channel registry.
type channelRegistryImpl struct {
	channels map[chat_apps.Platform]channels.ChatChannel
}

func (r *channelRegistryImpl) GetChannel(platform chat_apps.Platform) channels.ChatChannel {
	return r.channels[platform]
}

func (r *channelRegistryImpl) Register(channel channels.ChatChannel) {
	r.channels[channel.Name()] = channel
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
