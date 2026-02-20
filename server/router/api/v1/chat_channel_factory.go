// Package v1 provides chat channel factory and initialization.
package v1

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels/dingtalk"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels/telegram"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels/whatsapp"
	"github.com/hrygo/divinesense/plugin/chat_apps/store"
)

// validateChatAppsConfig validates the chat apps configuration at startup.
// This should be called during service initialization to fail fast if config is invalid.
func validateChatAppsConfig() error {
	secretKey := os.Getenv("DIVINESENSE_CHAT_APPS_SECRET_KEY")
	if secretKey == "" {
		return fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be set for secure token storage")
	}
	if len(secretKey) != 32 {
		return fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be exactly 32 bytes, got %d bytes", len(secretKey))
	}
	return nil
}

// initializeChatChannels loads all enabled credentials from the database
// and initializes the corresponding chat channels.
// This should be called during service startup.
func (s *ChatAppService) initializeChatChannels(ctx context.Context) error {
	slog.Info("initializing chat channels")

	chatAppStore := store.NewChatAppStore(s.Store.GetDriver().GetDB())
	creds, err := chatAppStore.ListAllEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	slog.Info("found enabled credentials", "count", len(creds))

	// Create and register channels for each credential
	for _, cred := range creds {
		ch, err := s.createChannelForCredential(cred)
		if err != nil {
			slog.Warn("failed to create channel",
				"platform", cred.Platform,
				"user_id", cred.UserID,
				"error", err,
			)
			continue
		}

		// Register the channel
		s.chatChannelRouter.Register(ch)
		slog.Info("channel registered",
			"platform", cred.Platform,
			"user_id", cred.UserID,
		)
	}

	return nil
}

// createChannelForCredential creates and initializes a channel for the given credential.
func (s *ChatAppService) createChannelForCredential(cred *chat_apps.Credential) (channels.ChatChannel, error) {
	// Decrypt the access token for channel initialization
	accessToken, err := s.decryptAccessToken(cred.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	switch cred.Platform {
	case chat_apps.PlatformTelegram:
		ch, err := telegram.NewTelegramChannel(&telegram.TelegramConfig{
			BotToken: accessToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create telegram channel: %w", err)
		}
		return ch, nil

	case chat_apps.PlatformDingTalk:
		// For DingTalk, accessToken is the AppKey, AppSecret is stored separately
		var appSecret string
		if cred.AppSecret != "" {
			appSecret, err = s.decryptAccessToken(cred.AppSecret)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt app secret: %w", err)
			}
		}

		ch, err := dingtalk.NewDingTalkChannel(&dingtalk.DingTalkConfig{
			AppKey:    accessToken,
			AppSecret: appSecret,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create dingtalk channel: %w", err)
		}
		// Set custom webhook URL if provided
		if cred.WebhookURL != "" {
			ch.SetWebhookURL(cred.WebhookURL)
		}
		return ch, nil

	case chat_apps.PlatformWhatsApp:
		ch, err := whatsapp.NewWhatsAppChannel(cred.WebhookURL, accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create whatsapp channel: %w", err)
		}
		return ch, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", cred.Platform)
	}
}

// decryptAccessToken decrypts an encrypted access token.
func (s *ChatAppService) decryptAccessToken(encryptedToken string) (string, error) {
	secretKey := os.Getenv("DIVINESENSE_CHAT_APPS_SECRET_KEY")
	if secretKey == "" {
		// FAIL FAST - Do not allow insecure decryption
		return "", fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be set for secure token storage")
	}

	// Validate key length
	if len(secretKey) != 32 {
		return "", fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be exactly 32 bytes, got %d bytes", len(secretKey))
	}

	return store.DecryptToken(encryptedToken, secretKey)
}
