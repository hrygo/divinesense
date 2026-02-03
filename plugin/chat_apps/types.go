// Package chat_apps provides multi-platform chat app integration for DivineSense.
// Supported platforms: Telegram, WhatsApp (via Baileys bridge), DingTalk.
package chat_apps

import "time"

// MessageType represents the type of message.
type MessageType int

const (
	MessageTypeText MessageType = iota
	MessageTypePhoto
	MessageTypeAudio
	MessageTypeVideo
	MessageTypeDocument
)

// String returns the string representation of MessageType.
func (m MessageType) String() string {
	switch m {
	case MessageTypeText:
		return "text"
	case MessageTypePhoto:
		return "photo"
	case MessageTypeAudio:
		return "audio"
	case MessageTypeVideo:
		return "video"
	case MessageTypeDocument:
		return "document"
	default:
		return "unknown"
	}
}

// Platform represents a supported chat platform.
type Platform string

const (
	PlatformTelegram Platform = "telegram"
	PlatformWhatsApp Platform = "whatsapp"
	PlatformDingTalk Platform = "dingtalk"
	PlatformWeb      Platform = "web"
)

// IsValid checks if the platform is valid.
func (p Platform) IsValid() bool {
	switch p {
	case PlatformTelegram, PlatformWhatsApp, PlatformDingTalk, PlatformWeb:
		return true
	default:
		return false
	}
}

// IncomingMessage represents a message from a chat platform.
type IncomingMessage struct {
	Platform       Platform          // Source platform
	PlatformUserID string            // Platform-specific user ID
	PlatformChatID string            // Platform-specific chat ID
	Type           MessageType       // Message type
	Content        string            // Text content
	MediaURL       string            // URL for media download
	MediaData      []byte            // Downloaded media data
	FileName       string            // Original filename
	MimeType       string            // MIME type
	Metadata       map[string]string // Additional platform-specific metadata
	Timestamp      time.Time         // Message timestamp
}

// OutgoingMessage represents a message to send to a chat platform.
type OutgoingMessage struct {
	PlatformChatID string      // Destination chat ID
	Type           MessageType // Message type
	Content        string      // Text content
	MediaData      []byte      // Media data for non-text messages
	MimeType       string      // MIME type of media
	FileName       string      // Original filename
	ParseMode      string      // Markdown/HTML parsing mode (optional)
}

// Credential represents a stored chat app credential.
type Credential struct {
	ID             int64
	UserID         int32
	Platform       Platform
	PlatformUserID string
	PlatformChatID string
	AccessToken    string // Encrypted at rest
	AppSecret      string // Encrypted at rest (e.g., DingTalk AppSecret)
	WebhookURL     string // For DingTalk
	Enabled        bool
	CreatedTs      int64
	UpdatedTs      int64
}

// WebhookInfo contains webhook setup information for a platform.
type WebhookInfo struct {
	URL                  string            // The webhook URL
	SetupInstructions    string            // Human-readable setup instructions
	Headers              map[string]string // Required headers for verification
	RequiresVerification bool              // Whether signature verification is required
}
