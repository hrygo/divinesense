// Package channels provides the ChatChannel interface for all chat platform integrations.
package channels

import (
	"context"
	"io"
	"sync"

	"github.com/hrygo/divinesense/plugin/chat_apps"
)

// ChatChannel defines the interface for all chat platform integrations.
// Each platform (Telegram, WhatsApp, DingTalk) implements this interface.
type ChatChannel interface {
	// Name returns the platform name (e.g., "telegram", "whatsapp", "dingtalk").
	Name() chat_apps.Platform

	// ValidateWebhook verifies the incoming webhook request.
	// Returns an error if the request signature is invalid or the request is malformed.
	ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) error

	// ParseMessage parses the incoming webhook payload into an IncomingMessage.
	// The payload format is platform-specific (JSON for most platforms).
	ParseMessage(ctx context.Context, payload []byte) (*chat_apps.IncomingMessage, error)

	// SendMessage sends a single message to the chat platform.
	SendMessage(ctx context.Context, msg *chat_apps.OutgoingMessage) error

	// SendChunkedMessage sends streaming content chunks.
	// The channels function closes when all chunks are sent or an error occurs.
	// This is used for streaming AI responses.
	SendChunkedMessage(ctx context.Context, chatID string, chunks <-chan string) error

	// DownloadMedia downloads media from the platform's CDN.
	// Returns the media data, MIME type, and an error if any.
	DownloadMedia(ctx context.Context, url string) ([]byte, string, error)

	// Close closes any open connections and releases resources.
	Close() error
}

// MediaHandler processes multimedia messages (audio, images).
type MediaHandler interface {
	// ProcessAudio converts audio data to text using Whisper.
	ProcessAudio(ctx context.Context, data []byte, mimeType string) (string, error)

	// ProcessImage extracts text from images using OCR.
	ProcessImage(ctx context.Context, data []byte) (string, error)
}

// ChannelRouter routes incoming messages to the appropriate handler.
// It verifies user credentials and forwards messages to the AI agent system.
// Concurrent-safe for Register and GetChannel operations.
type ChannelRouter struct {
	mu       sync.RWMutex
	registry map[chat_apps.Platform]ChatChannel
	media    MediaHandler
}

// NewChannelRouter creates a new channel router.
func NewChannelRouter(media MediaHandler) *ChannelRouter {
	return &ChannelRouter{
		registry: make(map[chat_apps.Platform]ChatChannel),
		media:    media,
	}
}

// Register registers a chat channel for a platform.
// Concurrent-safe: uses write lock.
func (r *ChannelRouter) Register(channel ChatChannel) {
	r.mu.Lock()
	r.registry[channel.Name()] = channel
	r.mu.Unlock()
}

// GetChannel returns the channel for a platform, or nil if not registered.
// Concurrent-safe: uses read lock.
func (r *ChannelRouter) GetChannel(platform chat_apps.Platform) ChatChannel {
	r.mu.RLock()
	ch := r.registry[platform]
	r.mu.RUnlock()
	return ch
}

// HandleWebhook handles an incoming webhook request.
func (r *ChannelRouter) HandleWebhook(ctx context.Context, platform chat_apps.Platform, headers map[string]string, body []byte) (*chat_apps.IncomingMessage, error) {
	channel := r.GetChannel(platform)
	if channel == nil {
		return nil, ErrNoChannelForPlatform
	}

	// Validate webhook signature
	if err := channel.ValidateWebhook(ctx, headers, body); err != nil {
		return nil, err
	}

	// Parse message
	msg, err := channel.ParseMessage(ctx, body)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// SendChunkedResponse sends a streaming AI response to a chat platform.
func (r *ChannelRouter) SendChunkedResponse(ctx context.Context, platform chat_apps.Platform, chatID string, chunks <-chan string) error {
	channel := r.GetChannel(platform)
	if channel == nil {
		return ErrNoChannelForPlatform
	}

	return channel.SendChunkedMessage(ctx, chatID, chunks)
}

// SendResponse sends a single response message to a chat platform.
func (r *ChannelRouter) SendResponse(ctx context.Context, platform chat_apps.Platform, msg *chat_apps.OutgoingMessage) error {
	channel := r.GetChannel(platform)
	if channel == nil {
		return ErrNoChannelForPlatform
	}

	return channel.SendMessage(ctx, msg)
}

// Errors
var (
	ErrNoChannelForPlatform = &ChannelError{Code: "NO_CHANNEL", Message: "no channel registered for platform"}
	ErrInvalidSignature     = &ChannelError{Code: "INVALID_SIGNATURE", Message: "webhook signature validation failed"}
	ErrInvalidPayload       = &ChannelError{Code: "INVALID_PAYLOAD", Message: "could not parse webhook payload"}
	ErrUnauthorized         = &ChannelError{Code: "UNAUTHORIZED", Message: "user not authorized for this platform"}
	ErrMediaDownloadFailed  = &ChannelError{Code: "MEDIA_FAILED", Message: "failed to download media"}
)

// ChannelError represents an error in channel operations.
type ChannelError struct {
	Code    string
	Message string
	Err     error
}

func (e *ChannelError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *ChannelError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is transient and the operation can be retried.
func (e *ChannelError) IsRetryable() bool {
	switch e.Code {
	case "NO_CHANNEL", "INVALID_SIGNATURE", "UNAUTHORIZED":
		return false
	default:
		return true
	}
}

// io.Closer interface for cleanup
var _ io.Closer = (*ChannelRouter)(nil)

// Close closes all registered channels.
// Concurrent-safe: uses write lock.
func (r *ChannelRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error
	for _, channel := range r.registry {
		if err := channel.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
