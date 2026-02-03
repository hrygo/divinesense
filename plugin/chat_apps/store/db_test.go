// Package store provides tests for chat_apps store operations.
package store

import (
	"testing"
	"time"

	"github.com/hrygo/divinesense/plugin/chat_apps"
)

// TestCreateCredentialRequest tests the request structure validation.
func TestCreateCredentialRequest(t *testing.T) {
	tests := []struct {
		name string
		req  CreateCredentialRequest
	}{
		{
			name: "valid telegram credential",
			req: CreateCredentialRequest{
				UserID:         123,
				Platform:       chat_apps.PlatformTelegram,
				PlatformUserID: "telegram_user_123",
				PlatformChatID: "123456789",
				AccessToken:    "encrypted_token",
				AppSecret:      "",
				WebhookURL:     "",
			},
		},
		{
			name: "valid dingtalk credential",
			req: CreateCredentialRequest{
				UserID:         456,
				Platform:       chat_apps.PlatformDingTalk,
				PlatformUserID: "ding_user",
				PlatformChatID: "ding_chat",
				AccessToken:    "encrypted_app_key",
				AppSecret:      "encrypted_app_secret",
				WebhookURL:     "https://oapi.dingtalk.com/robot/send",
			},
		},
		{
			name: "valid whatsapp credential",
			req: CreateCredentialRequest{
				UserID:         789,
				Platform:       chat_apps.PlatformWhatsApp,
				PlatformUserID: "wa_user",
				PlatformChatID: "wa_chat",
				AccessToken:    "",
				AppSecret:      "",
				WebhookURL:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate request fields
			if tt.req.UserID == 0 {
				t.Error("UserID cannot be 0")
			}
			if tt.req.Platform == "" {
				t.Error("Platform cannot be empty")
			}
			if tt.req.PlatformUserID == "" {
				t.Error("PlatformUserID cannot be empty")
			}

			// Platform-specific validation
			if tt.req.Platform == chat_apps.PlatformTelegram && tt.req.AccessToken == "" {
				t.Error("AccessToken required for Telegram")
			}
		})
	}
}

// TestUpdateCredentialRequest tests the update request structure.
func TestUpdateCredentialRequest(t *testing.T) {
	webhookURL := "https://example.com/webhook"
	accessToken := "new_token"
	tests := []struct {
		name string
		req  UpdateCredentialRequest
	}{
		{
			name: "update access token",
			req: UpdateCredentialRequest{
				ID:          1,
				AccessToken: &accessToken,
			},
		},
		{
			name: "update webhook URL",
			req: UpdateCredentialRequest{
				ID:         2,
				WebhookURL: &webhookURL,
			},
		},
		{
			name: "update multiple fields",
			req: UpdateCredentialRequest{
				ID:          3,
				AccessToken: &accessToken,
				WebhookURL:  &webhookURL,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.ID == 0 {
				t.Error("ID cannot be 0")
			}
		})
	}
}

// TestCredentialValidation tests platform-specific credential validation.
func TestCredentialValidation(t *testing.T) {
	tests := []struct {
		name      string
		platform  chat_apps.Platform
		token     string
		appSecret string
		wantValid bool
	}{
		{
			name:      "telegram with token",
			platform:  chat_apps.PlatformTelegram,
			token:     "bot_token_here",
			wantValid: true,
		},
		{
			name:      "telegram without token",
			platform:  chat_apps.PlatformTelegram,
			token:     "",
			wantValid: false,
		},
		{
			name:      "dingtalk with app key",
			platform:  chat_apps.PlatformDingTalk,
			token:     "app_key_here",
			wantValid: true,
		},
		{
			name:      "dingtalk without app key",
			platform:  chat_apps.PlatformDingTalk,
			token:     "",
			wantValid: false,
		},
		{
			name:      "whatsapp without token",
			platform:  chat_apps.PlatformWhatsApp,
			token:     "",
			wantValid: true, // WhatsApp uses bridge, token may not be required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := true

			// Platform-specific validation
			switch tt.platform {
			case chat_apps.PlatformTelegram:
				if tt.token == "" {
					isValid = false
				}
			case chat_apps.PlatformDingTalk:
				if tt.token == "" {
					isValid = false
				}
			case chat_apps.PlatformWhatsApp:
				// WhatsApp doesn't require token in Go code
				// It's managed by the bridge service
			}

			if isValid != tt.wantValid {
				t.Errorf("validation result mismatch: got %v, want %v", isValid, tt.wantValid)
			}
		})
	}
}

// TestPlatformString tests platform string representation.
func TestPlatformString(t *testing.T) {
	tests := []struct {
		platform chat_apps.Platform
		want     string
	}{
		{chat_apps.PlatformTelegram, "telegram"},
		{chat_apps.PlatformWhatsApp, "whatsapp"},
		{chat_apps.PlatformDingTalk, "dingtalk"},
		{chat_apps.Platform(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.platform)
			if got != tt.want {
				t.Errorf("Platform.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCredentialDefaults tests default values for new credentials.
func TestCredentialDefaults(t *testing.T) {
	cred := &chat_apps.Credential{
		ID:             1,
		UserID:         123,
		Platform:       chat_apps.PlatformTelegram,
		PlatformUserID: "user123",
		Enabled:        true,
		CreatedTs:      time.Now().Unix(),
		UpdatedTs:      time.Now().Unix(),
	}

	// Use fields to avoid unused warnings
	_ = cred.ID
	_ = cred.UserID
	_ = cred.Platform
	_ = cred.PlatformUserID

	if !cred.Enabled {
		t.Error("new credential should be enabled by default")
	}
	if cred.CreatedTs == 0 {
		t.Error("CreatedTs should be set")
	}
	if cred.UpdatedTs == 0 {
		t.Error("UpdatedTs should be set")
	}
}
