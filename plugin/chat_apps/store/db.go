// Package store provides database operations for chat app credentials.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/hrygo/divinesense/plugin/chat_apps"
)

// Validation limits for chat app credential fields.
const (
	MaxPlatformUserID = 255  // Maximum length for platform user ID
	MaxAccessToken    = 2048 // Maximum length for access tokens
	MaxAppSecret      = 2048 // Maximum length for app secrets
	MaxWebhookURL     = 2048 // Maximum length for webhook URLs
)

// ChatAppStore manages chat app credentials in the database.
type ChatAppStore struct {
	db *sql.DB
}

// NewChatAppStore creates a new chat app store.
func NewChatAppStore(db *sql.DB) *ChatAppStore {
	return &ChatAppStore{db: db}
}

// validateCreateCredentialRequest validates the create credential request.
func validateCreateCredentialRequest(req *CreateCredentialRequest) error {
	// Validate platform is supported
	if !req.Platform.IsValid() {
		slog.Warn("invalid platform specified", "platform", req.Platform)
		return fmt.Errorf("invalid platform: %s", req.Platform)
	}

	// Validate platform_user_id length
	if len(req.PlatformUserID) == 0 {
		return fmt.Errorf("platform_user_id is required")
	}
	if len(req.PlatformUserID) > MaxPlatformUserID {
		return fmt.Errorf("platform_user_id too long: max %d characters", MaxPlatformUserID)
	}

	// Validate access_token length
	if len(req.AccessToken) > MaxAccessToken {
		return fmt.Errorf("access_token too long: max %d characters", MaxAccessToken)
	}

	// Validate app_secret length (optional for some platforms)
	if len(req.AppSecret) > MaxAppSecret {
		return fmt.Errorf("app_secret too long: max %d characters", MaxAppSecret)
	}

	// Validate webhook_url length (optional)
	if len(req.WebhookURL) > MaxWebhookURL {
		return fmt.Errorf("webhook_url too long: max %d characters", MaxWebhookURL)
	}

	return nil
}

// CreateCredential creates a new chat app credential.
func (s *ChatAppStore) CreateCredential(ctx context.Context, req *CreateCredentialRequest) (*chat_apps.Credential, error) {
	// Validate request before database operations
	if err := validateCreateCredentialRequest(req); err != nil {
		slog.Warn("invalid create credential request", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	slog.Info("creating chat app credential",
		"user_id", req.UserID,
		"platform", req.Platform,
	)
	query := `
		INSERT INTO chat_app_credential
		(user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, created_ts, updated_ts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
	`

	now := time.Now().Unix()
	row := s.db.QueryRowContext(ctx, query,
		req.UserID,
		req.Platform,
		req.PlatformUserID,
		req.PlatformChatID,
		req.AccessToken,
		req.AppSecret,
		req.WebhookURL,
		now,
		now,
	)

	var cred chat_apps.Credential
	err := row.Scan(
		&cred.ID,
		&cred.UserID,
		&cred.Platform,
		&cred.PlatformUserID,
		&cred.PlatformChatID,
		&cred.AccessToken,
		&cred.AppSecret,
		&cred.WebhookURL,
		&cred.Enabled,
		&cred.CreatedTs,
		&cred.UpdatedTs,
	)
	if err != nil {
		slog.Error("failed to create chat app credential",
			"user_id", req.UserID,
			"platform", req.Platform,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	slog.Info("chat app credential created",
		"id", cred.ID,
		"user_id", cred.UserID,
		"platform", cred.Platform,
	)
	return &cred, nil
}

// ListCredentials lists all credentials for a user.
func (s *ChatAppStore) ListCredentials(ctx context.Context, userID int32, platformFilter chat_apps.Platform) ([]*chat_apps.Credential, error) {
	query := `
		SELECT id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
		FROM chat_app_credential
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if platformFilter != "" {
		query += " AND platform = $2"
		args = append(args, string(platformFilter))
	}

	query += " ORDER BY created_ts DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var credentials []*chat_apps.Credential
	for rows.Next() {
		var cred chat_apps.Credential
		err := rows.Scan(
			&cred.ID,
			&cred.UserID,
			&cred.Platform,
			&cred.PlatformUserID,
			&cred.PlatformChatID,
			&cred.AccessToken,
			&cred.AppSecret,
			&cred.WebhookURL,
			&cred.Enabled,
			&cred.CreatedTs,
			&cred.UpdatedTs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		credentials = append(credentials, &cred)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return credentials, nil
}

// GetCredentialByPlatform retrieves a credential by user ID and platform.
func (s *ChatAppStore) GetCredentialByPlatform(ctx context.Context, userID int32, platform chat_apps.Platform) (*chat_apps.Credential, error) {
	query := `
		SELECT id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
		FROM chat_app_credential
		WHERE user_id = $1 AND platform = $2
	`

	row := s.db.QueryRowContext(ctx, query, userID, string(platform))

	var cred chat_apps.Credential
	err := row.Scan(
		&cred.ID,
		&cred.UserID,
		&cred.Platform,
		&cred.PlatformUserID,
		&cred.PlatformChatID,
		&cred.AccessToken,
		&cred.AppSecret,
		&cred.WebhookURL,
		&cred.Enabled,
		&cred.CreatedTs,
		&cred.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	return &cred, nil
}

// GetCredentialByPlatformUserID retrieves a credential by platform user ID.
// Used during webhook handling to find the DivineSense user.
func (s *ChatAppStore) GetCredentialByPlatformUserID(ctx context.Context, platform chat_apps.Platform, platformUserID string) (*chat_apps.Credential, error) {
	slog.Debug("looking up credential by platform user ID",
		"platform", platform,
		"platform_user_id", platformUserID,
	)
	query := `
		SELECT id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
		FROM chat_app_credential
		WHERE platform = $1 AND platform_user_id = $2 AND enabled = true
	`

	row := s.db.QueryRowContext(ctx, query, string(platform), platformUserID)

	var cred chat_apps.Credential
	err := row.Scan(
		&cred.ID,
		&cred.UserID,
		&cred.Platform,
		&cred.PlatformUserID,
		&cred.PlatformChatID,
		&cred.AccessToken,
		&cred.AppSecret,
		&cred.WebhookURL,
		&cred.Enabled,
		&cred.CreatedTs,
		&cred.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	return &cred, nil
}

// DeleteCredential deletes a credential by ID.
func (s *ChatAppStore) DeleteCredential(ctx context.Context, id int64) error {
	slog.Info("deleting chat app credential", "id", id)
	query := `DELETE FROM chat_app_credential WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		slog.Error("failed to delete chat app credential",
			"id", id,
			"error", err,
		)
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		slog.Warn("chat app credential not found for deletion", "id", id)
		return fmt.Errorf("credential not found")
	}

	slog.Info("chat app credential deleted", "id", id)
	return nil
}

// UpdateCredential updates a credential.
func (s *ChatAppStore) UpdateCredential(ctx context.Context, req *UpdateCredentialRequest) (*chat_apps.Credential, error) {
	query := `
		UPDATE chat_app_credential
		SET access_token = COALESCE($2, access_token),
		    app_secret = COALESCE($3, app_secret),
		    webhook_url = COALESCE($4, webhook_url),
		    updated_ts = $5
		WHERE id = $1
		RETURNING id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
	`

	now := time.Now().Unix()
	row := s.db.QueryRowContext(ctx, query, req.ID, req.AccessToken, req.AppSecret, req.WebhookURL, now)

	var cred chat_apps.Credential
	err := row.Scan(
		&cred.ID,
		&cred.UserID,
		&cred.Platform,
		&cred.PlatformUserID,
		&cred.PlatformChatID,
		&cred.AccessToken,
		&cred.AppSecret,
		&cred.WebhookURL,
		&cred.Enabled,
		&cred.CreatedTs,
		&cred.UpdatedTs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update credential: %w", err)
	}

	return &cred, nil
}

// SetEnabled enables or disables a credential.
func (s *ChatAppStore) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	slog.Info("setting chat app credential enabled state",
		"id", id,
		"enabled", enabled,
	)
	query := `
		UPDATE chat_app_credential
		SET enabled = $2, updated_ts = $3
		WHERE id = $1
	`
	result, err := s.db.ExecContext(ctx, query, id, enabled, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to set enabled: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

// Request/Response types

type CreateCredentialRequest struct {
	UserID         int32
	Platform       chat_apps.Platform
	PlatformUserID string
	PlatformChatID string
	AccessToken    string // Already encrypted
	AppSecret      string // Already encrypted
	WebhookURL     string
}

type UpdateCredentialRequest struct {
	ID          int64
	AccessToken *string
	AppSecret   *string
	WebhookURL  *string
}

// ListAllEnabled lists all enabled credentials across all users.
// Used during service startup to initialize chat channels.
func (s *ChatAppStore) ListAllEnabled(ctx context.Context) ([]*chat_apps.Credential, error) {
	query := `
		SELECT id, user_id, platform, platform_user_id, platform_chat_id, access_token, app_secret, webhook_url, enabled, created_ts, updated_ts
		FROM chat_app_credential
		WHERE enabled = true
		ORDER BY platform, user_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled credentials: %w", err)
	}
	defer rows.Close()

	var credentials []*chat_apps.Credential
	for rows.Next() {
		var cred chat_apps.Credential
		err := rows.Scan(
			&cred.ID,
			&cred.UserID,
			&cred.Platform,
			&cred.PlatformUserID,
			&cred.PlatformChatID,
			&cred.AccessToken,
			&cred.AppSecret,
			&cred.WebhookURL,
			&cred.Enabled,
			&cred.CreatedTs,
			&cred.UpdatedTs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}
		credentials = append(credentials, &cred)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return credentials, nil
}
