package v1

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/hrygo/divinesense/internal/util"
)

const (
	InstanceSettingNamePrefix  = "instance/settings/"
	UserNamePrefix             = "users/"
	MemoNamePrefix             = "memos/"
	AttachmentNamePrefix       = "attachments/"
	ReactionNamePrefix         = "reactions/"
	InboxNamePrefix            = "inboxes/"
	IdentityProviderNamePrefix = "identity-providers/"
	ActivityNamePrefix         = "activities/"
	WebhookNamePrefix          = "webhooks/"
)

// GetNameParentTokens returns the tokens from a resource name.
func GetNameParentTokens(name string, tokenPrefixes ...string) ([]string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2*len(tokenPrefixes) {
		return nil, errors.Errorf("invalid request %q", name)
	}

	var tokens []string
	for i, tokenPrefix := range tokenPrefixes {
		if fmt.Sprintf("%s/", parts[2*i]) != tokenPrefix {
			return nil, errors.Errorf("invalid prefix %q in request %q", tokenPrefix, name)
		}
		if parts[2*i+1] == "" {
			return nil, errors.Errorf("invalid request %q with empty prefix %q", name, tokenPrefix)
		}
		tokens = append(tokens, parts[2*i+1])
	}
	return tokens, nil
}

func ExtractInstanceSettingKeyFromName(name string) (string, error) {
	const prefix = "instance/settings/"
	if !strings.HasPrefix(name, prefix) {
		return "", errors.Errorf("invalid instance setting name: expected prefix %q, got %q", prefix, name)
	}

	settingKey := strings.TrimPrefix(name, prefix)
	if settingKey == "" {
		return "", errors.Errorf("invalid instance setting name: empty setting key in %q", name)
	}

	// Ensure there are no additional path segments
	if strings.Contains(settingKey, "/") {
		return "", errors.Errorf("invalid instance setting name: setting key cannot contain '/' in %q", name)
	}

	return settingKey, nil
}

// ExtractUserIDFromName returns the uid from a resource name.
func ExtractUserIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix)
	if err != nil {
		return 0, err
	}
	id, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid user ID %q", tokens[0])
	}
	return id, nil
}

// extractUserIdentifierFromName extracts the identifier (ID or username) from a user resource name.
// Supports: "users/101" or "users/steven"
// Returns the identifier string (e.g., "101" or "steven").
func extractUserIdentifierFromName(name string) string {
	tokens, err := GetNameParentTokens(name, UserNamePrefix)
	if err != nil || len(tokens) == 0 {
		return ""
	}
	return tokens[0]
}

// ExtractMemoUIDFromName returns the memo UID from a resource name.
// e.g., "memos/uuid" -> "uuid".
func ExtractMemoUIDFromName(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, MemoNamePrefix)
	if err != nil {
		return "", err
	}
	id := tokens[0]
	return id, nil
}

// ExtractAttachmentUIDFromName returns the attachment UID from a resource name.
func ExtractAttachmentUIDFromName(name string) (string, error) {
	tokens, err := GetNameParentTokens(name, AttachmentNamePrefix)
	if err != nil {
		return "", err
	}
	id := tokens[0]
	return id, nil
}

// ExtractMemoReactionIDFromName returns the memo UID and reaction ID from a resource name.
// e.g., "memos/abc/reactions/123" -> ("abc", 123).
func ExtractMemoReactionIDFromName(name string) (string, int32, error) {
	tokens, err := GetNameParentTokens(name, MemoNamePrefix, ReactionNamePrefix)
	if err != nil {
		return "", 0, err
	}
	memoUID := tokens[0]
	reactionID, err := util.ConvertStringToInt32(tokens[1])
	if err != nil {
		return "", 0, errors.Errorf("invalid reaction ID %q", tokens[1])
	}
	return memoUID, reactionID, nil
}

// ExtractInboxIDFromName returns the inbox ID from a resource name.
func ExtractInboxIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, InboxNamePrefix)
	if err != nil {
		return 0, err
	}
	id, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid inbox ID %q", tokens[0])
	}
	return id, nil
}

func ExtractIdentityProviderIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, IdentityProviderNamePrefix)
	if err != nil {
		return 0, err
	}
	id, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid identity provider ID %q", tokens[0])
	}
	return id, nil
}

func ExtractActivityIDFromName(name string) (int32, error) {
	tokens, err := GetNameParentTokens(name, ActivityNamePrefix)
	if err != nil {
		return 0, err
	}
	id, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, errors.Errorf("invalid activity ID %q", tokens[0])
	}
	return id, nil
}

// --- Compound resource name extractors ---

const (
	ShortcutNamePrefix            = "shortcuts/"
	PersonalAccessTokenNamePrefix = "personalAccessTokens/"
	SettingNamePrefix             = "settings/"
)

// ExtractUserAndShortcutIDFromName extracts user ID and shortcut ID from "users/{user}/shortcuts/{shortcut}".
func ExtractUserAndShortcutIDFromName(name string) (int32, string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix, ShortcutNamePrefix)
	if err != nil {
		return 0, "", errors.Errorf("invalid shortcut name format: %s", name)
	}
	userID, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, "", errors.Errorf("invalid user ID %q", tokens[0])
	}
	return userID, tokens[1], nil
}

// ExtractUserAndWebhookIDFromName extracts user ID and webhook ID from "users/{user}/webhooks/{webhook}".
func ExtractUserAndWebhookIDFromName(name string) (int32, string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix, WebhookNamePrefix)
	if err != nil {
		return 0, "", errors.Errorf("invalid webhook name format: %s", name)
	}
	userID, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, "", errors.Errorf("invalid user ID %q", tokens[0])
	}
	return userID, tokens[1], nil
}

// ExtractUserAndSettingKeyFromName extracts user ID and setting key from "users/{user}/settings/{key}".
func ExtractUserAndSettingKeyFromName(name string) (int32, string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix, SettingNamePrefix)
	if err != nil {
		return 0, "", errors.Errorf("invalid resource name format: %s", name)
	}
	userID, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, "", errors.Errorf("invalid user ID: %s", tokens[0])
	}
	return userID, tokens[1], nil
}

// ExtractUserAndPATIDFromName extracts user ID and token ID from "users/{user}/personalAccessTokens/{tokenID}".
func ExtractUserAndPATIDFromName(name string) (int32, string, error) {
	tokens, err := GetNameParentTokens(name, UserNamePrefix, PersonalAccessTokenNamePrefix)
	if err != nil {
		return 0, "", errors.Errorf("invalid personal access token name: %s", name)
	}
	userID, err := util.ConvertStringToInt32(tokens[0])
	if err != nil {
		return 0, "", errors.Errorf("invalid user ID %q", tokens[0])
	}
	return userID, tokens[1], nil
}

// ExtractWebhookIDFromResourceName extracts webhook ID from "users/{user}/webhooks/{webhook_id}".
// Lightweight variant that only returns the webhook ID.
func ExtractWebhookIDFromResourceName(name string) string {
	_, webhookID, err := ExtractUserAndWebhookIDFromName(name)
	if err != nil {
		return ""
	}
	return webhookID
}
