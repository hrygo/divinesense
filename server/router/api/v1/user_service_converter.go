package v1

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// Helper functions for webhook operations

// generateUserWebhookID generates a unique ID for user webhooks.
func generateUserWebhookID() string {
	b := make([]byte, 8)
	// nolint:errcheck
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// parseUserWebhookName parses a webhook name and returns the webhook ID and user ID.
// Format: users/{user}/webhooks/{webhook}.
// Delegates to the standardized resource name extractor.
func parseUserWebhookName(name string) (string, int32, error) {
	userID, webhookID, err := ExtractUserAndWebhookIDFromName(name)
	if err != nil {
		return "", 0, err
	}
	return webhookID, userID, nil
}

// convertUserWebhookFromUserSetting converts a storepb webhook to a v1pb UserWebhook.
func convertUserWebhookFromUserSetting(webhook *storepb.WebhooksUserSetting_Webhook, userID int32) *v1pb.UserWebhook {
	return &v1pb.UserWebhook{
		Name:        fmt.Sprintf("users/%d/webhooks/%s", userID, webhook.Id),
		Url:         webhook.Url,
		DisplayName: webhook.Title,
		// Note: create_time and update_time are not available in the user setting webhook structure
		// This is a limitation of storing webhooks in user settings vs the dedicated webhook table
	}
}

func convertUserFromStore(user *store.User) *v1pb.User {
	userpb := &v1pb.User{
		Name:        fmt.Sprintf("%s%d", UserNamePrefix, user.ID),
		State:       convertStateFromStore(user.RowStatus),
		CreateTime:  timestamppb.New(time.Unix(user.CreatedTs, 0)),
		UpdateTime:  timestamppb.New(time.Unix(user.UpdatedTs, 0)),
		Role:        convertUserRoleFromStore(user.Role),
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.Nickname,
		AvatarUrl:   user.AvatarURL,
		Description: user.Description,
	}
	// Use the avatar URL instead of raw base64 image data to reduce the response size.
	if user.AvatarURL != "" {
		// Check if avatar url is base64 format.
		_, _, err := extractImageInfo(user.AvatarURL)
		if err == nil {
			userpb.AvatarUrl = fmt.Sprintf("/file/%s/avatar", userpb.Name)
		} else {
			userpb.AvatarUrl = user.AvatarURL
		}
	}
	return userpb
}

func convertUserRoleFromStore(role store.Role) v1pb.User_Role {
	switch role {
	case store.RoleHost:
		return v1pb.User_HOST
	case store.RoleAdmin:
		return v1pb.User_ADMIN
	case store.RoleUser:
		return v1pb.User_USER
	default:
		return v1pb.User_ROLE_UNSPECIFIED
	}
}

func convertUserRoleToStore(role v1pb.User_Role) store.Role {
	switch role {
	case v1pb.User_HOST:
		return store.RoleHost
	case v1pb.User_ADMIN:
		return store.RoleAdmin
	default:
		return store.RoleUser
	}
}

// extractImageInfo extracts image type and base64 data from a data URI.
// Data URI format: data:image/png;base64,iVBORw0KGgo...
func extractImageInfo(dataURI string) (string, string, error) {
	dataURIRegex := regexp.MustCompile(`^data:(?P<type>.+);base64,(?P<base64>.+)`)
	matches := dataURIRegex.FindStringSubmatch(dataURI)
	if len(matches) != 3 {
		return "", "", errors.New("invalid data URI format")
	}
	imageType := matches[1]
	base64Data := matches[2]
	return imageType, base64Data, nil
}

// Helper functions for user settings

// ExtractUserIDAndSettingKeyFromName extracts user ID and setting key from resource name.
// e.g., "users/123/settings/general" -> 123, "general".
// Delegates to the standardized resource name extractor.
func ExtractUserIDAndSettingKeyFromName(name string) (int32, string, error) {
	return ExtractUserAndSettingKeyFromName(name)
}

// convertSettingKeyToStore converts API setting key to store enum.
func convertSettingKeyToStore(key string) (storepb.UserSetting_Key, error) {
	switch key {
	case v1pb.UserSetting_Key_name[int32(v1pb.UserSetting_GENERAL)]:
		return storepb.UserSetting_GENERAL, nil
	case v1pb.UserSetting_Key_name[int32(v1pb.UserSetting_WEBHOOKS)]:
		return storepb.UserSetting_WEBHOOKS, nil
	default:
		return storepb.UserSetting_KEY_UNSPECIFIED, errors.Errorf("unknown setting key: %s", key)
	}
}

// convertSettingKeyFromStore converts store enum to API setting key.
func convertSettingKeyFromStore(key storepb.UserSetting_Key) string {
	switch key {
	case storepb.UserSetting_GENERAL:
		return v1pb.UserSetting_Key_name[int32(v1pb.UserSetting_GENERAL)]
	case storepb.UserSetting_SHORTCUTS:
		return "SHORTCUTS" // Not defined in API proto
	case storepb.UserSetting_WEBHOOKS:
		return v1pb.UserSetting_Key_name[int32(v1pb.UserSetting_WEBHOOKS)]
	default:
		return "unknown"
	}
}

// convertUserSettingFromStore converts store UserSetting to API UserSetting.
func convertUserSettingFromStore(storeSetting *storepb.UserSetting, userID int32, key storepb.UserSetting_Key) *v1pb.UserSetting {
	if storeSetting == nil {
		// Return default setting if none exists
		settingKey := convertSettingKeyFromStore(key)
		setting := &v1pb.UserSetting{
			Name: fmt.Sprintf("users/%d/settings/%s", userID, settingKey),
		}

		switch key {
		case storepb.UserSetting_WEBHOOKS:
			setting.Value = &v1pb.UserSetting_WebhooksSetting_{
				WebhooksSetting: &v1pb.UserSetting_WebhooksSetting{
					Webhooks: []*v1pb.UserWebhook{},
				},
			}
		default:
			// Default to general setting
			setting.Value = &v1pb.UserSetting_GeneralSetting_{
				GeneralSetting: getDefaultUserGeneralSetting(),
			}
		}
		return setting
	}

	settingKey := convertSettingKeyFromStore(storeSetting.Key)
	setting := &v1pb.UserSetting{
		Name: fmt.Sprintf("users/%d/settings/%s", userID, settingKey),
	}

	switch storeSetting.Key {
	case storepb.UserSetting_GENERAL:
		if general := storeSetting.GetGeneral(); general != nil {
			setting.Value = &v1pb.UserSetting_GeneralSetting_{
				GeneralSetting: &v1pb.UserSetting_GeneralSetting{
					Locale:         general.Locale,
					MemoVisibility: general.MemoVisibility,
					Theme:          general.Theme,
				},
			}
		} else {
			setting.Value = &v1pb.UserSetting_GeneralSetting_{
				GeneralSetting: getDefaultUserGeneralSetting(),
			}
		}
	case storepb.UserSetting_WEBHOOKS:
		webhooks := storeSetting.GetWebhooks()
		apiWebhooks := make([]*v1pb.UserWebhook, 0, len(webhooks.Webhooks))
		for _, webhook := range webhooks.Webhooks {
			apiWebhook := &v1pb.UserWebhook{
				Name:        fmt.Sprintf("users/%d/webhooks/%s", userID, webhook.Id),
				Url:         webhook.Url,
				DisplayName: webhook.Title,
			}
			apiWebhooks = append(apiWebhooks, apiWebhook)
		}
		setting.Value = &v1pb.UserSetting_WebhooksSetting_{
			WebhooksSetting: &v1pb.UserSetting_WebhooksSetting{
				Webhooks: apiWebhooks,
			},
		}
	default:
		// Default to general setting if unknown key
		setting.Value = &v1pb.UserSetting_GeneralSetting_{
			GeneralSetting: getDefaultUserGeneralSetting(),
		}
	}

	return setting
}

// convertUserSettingToStore converts API UserSetting to store UserSetting.
func convertUserSettingToStore(apiSetting *v1pb.UserSetting, userID int32, key storepb.UserSetting_Key) (*storepb.UserSetting, error) {
	storeSetting := &storepb.UserSetting{
		UserId: userID,
		Key:    key,
	}

	switch key {
	case storepb.UserSetting_GENERAL:
		if general := apiSetting.GetGeneralSetting(); general != nil {
			storeSetting.Value = &storepb.UserSetting_General{
				General: &storepb.GeneralUserSetting{
					Locale:         general.Locale,
					MemoVisibility: general.MemoVisibility,
					Theme:          general.Theme,
				},
			}
		} else {
			return nil, errors.Errorf("general setting is required")
		}
	case storepb.UserSetting_WEBHOOKS:
		if webhooks := apiSetting.GetWebhooksSetting(); webhooks != nil {
			storeWebhooks := make([]*storepb.WebhooksUserSetting_Webhook, 0, len(webhooks.Webhooks))
			for _, webhook := range webhooks.Webhooks {
				storeWebhook := &storepb.WebhooksUserSetting_Webhook{
					Id:    extractWebhookIDFromName(webhook.Name),
					Title: webhook.DisplayName,
					Url:   webhook.Url,
				}
				storeWebhooks = append(storeWebhooks, storeWebhook)
			}
			storeSetting.Value = &storepb.UserSetting_Webhooks{
				Webhooks: &storepb.WebhooksUserSetting{
					Webhooks: storeWebhooks,
				},
			}
		} else {
			return nil, errors.Errorf("webhooks setting is required")
		}
	default:
		return nil, errors.Errorf("unsupported setting key: %v", key)
	}

	return storeSetting, nil
}

// extractWebhookIDFromName extracts webhook ID from resource name.
// e.g., "users/123/webhooks/webhook-id" -> "webhook-id".
// Delegates to the standardized resource name extractor.
func extractWebhookIDFromName(name string) string {
	return ExtractWebhookIDFromResourceName(name)
}

// convertInboxToUserNotification converts a storage-layer inbox to an API notification.
// This handles the mapping between the internal inbox representation and the public API.
func (s *UserService) convertInboxToUserNotification(_ context.Context, inbox *store.Inbox) (*v1pb.UserNotification, error) {
	notification := &v1pb.UserNotification{
		Name:       fmt.Sprintf("users/%d/notifications/%d", inbox.ReceiverID, inbox.ID),
		Sender:     fmt.Sprintf("%s%d", UserNamePrefix, inbox.SenderID),
		CreateTime: timestamppb.New(time.Unix(inbox.CreatedTs, 0)),
	}

	// Convert status from storage enum to API enum
	switch inbox.Status {
	case store.UNREAD:
		notification.Status = v1pb.UserNotification_UNREAD
	case store.ARCHIVED:
		notification.Status = v1pb.UserNotification_ARCHIVED
	default:
		notification.Status = v1pb.UserNotification_STATUS_UNSPECIFIED
	}

	// Extract notification type and activity ID from inbox message
	if inbox.Message != nil {
		switch inbox.Message.Type {
		case storepb.InboxMessage_MEMO_COMMENT:
			notification.Type = v1pb.UserNotification_MEMO_COMMENT
		default:
			notification.Type = v1pb.UserNotification_TYPE_UNSPECIFIED
		}

		if inbox.Message.ActivityId != nil {
			notification.ActivityId = inbox.Message.ActivityId
		}
	}

	return notification, nil
}

// ExtractNotificationIDFromName extracts the notification ID from a resource name.
// Expected format: users/{user_id}/notifications/{notification_id}.
func ExtractNotificationIDFromName(name string) (int32, error) {
	pattern := regexp.MustCompile(`^users/(\d+)/notifications/(\d+)$`)
	matches := pattern.FindStringSubmatch(name)
	if len(matches) != 3 {
		return 0, errors.Errorf("invalid notification name: %s", name)
	}

	id, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, errors.Errorf("invalid notification id: %s", matches[2])
	}

	return int32(id), nil
}
