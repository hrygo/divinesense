// Package dingtalk implements the DingTalk Robot channel.
package dingtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
)

const (
	DefaultTimestampWindow = time.Hour
	DingTalkAPIBaseURL     = "https://oapi.dingtalk.com"
)

// DingTalkConfig holds configuration for the DingTalk channel.
type DingTalkConfig struct {
	AppKey    string
	AppSecret string
}

// DingTalkChannel implements ChatChannel for DingTalk Robot.
type DingTalkChannel struct {
	config     *DingTalkConfig
	webhookURL string
}

// NewDingTalkChannel creates a new DingTalk channel.
func NewDingTalkChannel(config *DingTalkConfig) (*DingTalkChannel, error) {
	return &DingTalkChannel{
		config: config,
	}, nil
}

// Name returns the platform name.
func (d *DingTalkChannel) Name() chat_apps.Platform {
	return chat_apps.PlatformDingTalk
}

// SetWebhookURL sets the custom webhook URL for this channel.
func (d *DingTalkChannel) SetWebhookURL(webhookURL string) {
	d.webhookURL = webhookURL
}

// ValidateWebhook verifies the incoming webhook request using DingTalk signature.
func (d *DingTalkChannel) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) error {
	// DingTalk sends signature in headers or query string
	// The signature is computed as: base64(hmac_sha256(timestamp + "\n" + secret, body))

	timestamp := headers["X-DingTalk-Timestamp"]
	sign := headers["X-DingTalk-Signature"]

	if timestamp == "" || sign == "" {
		// Try query string parameters
		values, err := url.ParseQuery(headers["Query-String"])
		if err == nil {
			timestamp = values.Get("timestamp")
			sign = values.Get("sign")
		}
	}

	if timestamp == "" || sign == "" {
		return channels.ErrInvalidSignature
	}

	// Compute expected signature
	expectedSign := d.computeSignature(timestamp, string(body))

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(sign), []byte(expectedSign)) {
		return channels.ErrInvalidSignature
	}

	return nil
}

// ParseMessage parses the incoming webhook payload.
func (d *DingTalkChannel) ParseMessage(ctx context.Context, payload []byte) (*chat_apps.IncomingMessage, error) {
	// DingTalk message format (JSON):
	// {
	//   "chatType": "group",
	//   "msgId": "...",
	//   "senderNick": "...",
	//   "senderStaffId": "...",
	//   "text": {"content": "..."},
	//   "msgtype": "text",
	//   "createAt": 1234567890
	// }

	var dm DingTalkMessage
	if err := json.Unmarshal(payload, &dm); err != nil {
		return nil, channels.ErrInvalidPayload
	}

	msg := &chat_apps.IncomingMessage{
		Platform:       chat_apps.PlatformDingTalk,
		PlatformUserID: dm.SenderStaffID,
		Content:        dm.Text.Content,
		Metadata:       make(map[string]string),
	}

	// Store metadata
	msg.Metadata["msg_id"] = dm.MsgID
	msg.Metadata["sender_nick"] = dm.SenderNick
	msg.Metadata["chat_type"] = dm.ChatType

	// Parse timestamp
	if dm.CreateAt > 0 {
		msg.Timestamp = time.Unix(dm.CreateAt/1000, 0)
	} else {
		msg.Timestamp = time.Now()
	}

	// Handle different message types
	switch dm.MsgType {
	case "text":
		msg.Type = chat_apps.MessageTypeText
	case "image":
		msg.Type = chat_apps.MessageTypePhoto
		msg.MediaURL = dm.Image.MediaID
		// Download media using media_id
	case "audio":
		msg.Type = chat_apps.MessageTypeAudio
		msg.MediaURL = dm.Audio.MediaID
	case "video":
		msg.Type = chat_apps.MessageTypeVideo
		msg.MediaURL = dm.Video.MediaID
	case "file":
		msg.Type = chat_apps.MessageTypeDocument
		msg.MediaURL = dm.File.MediaID
		msg.FileName = dm.File.FileName
	default:
		msg.Type = chat_apps.MessageTypeText
	}

	return msg, nil
}

// SendMessage sends a message to DingTalk.
func (d *DingTalkChannel) SendMessage(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	// For DingTalk, we send to the user's webhook URL (outgoing webhook)
	// or use the conversation API

	switch msg.Type {
	case chat_apps.MessageTypePhoto, chat_apps.MessageTypeVideo, chat_apps.MessageTypeDocument:
		return d.sendMedia(ctx, msg)
	default:
		return d.sendText(ctx, msg)
	}
}

// SendChunkedMessage sends streaming content chunks.
// DingTalk doesn't support message editing, so we send chunks as separate messages.
func (d *DingTalkChannel) SendChunkedMessage(ctx context.Context, chatID string, chunks <-chan string) error {
	// Accumulate and send as a single message for better UX
	var fullContent string
	for chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fullContent += chunk
		}
	}

	msg := &chat_apps.OutgoingMessage{
		PlatformChatID: chatID,
		Type:           chat_apps.MessageTypeText,
		Content:        fullContent,
	}

	return d.sendText(ctx, msg)
}

// DownloadMedia downloads media from DingTalk using the downloadCode.
func (d *DingTalkChannel) DownloadMedia(ctx context.Context, downloadCode string) ([]byte, string, error) {
	// DingTalk uses downloadCode for temporary file download
	// We need to call the media download API

	// url := fmt.Sprintf("%s/media/download?downloadCode=%s",
	// 	DingTalkAPIBaseURL, downloadCode)

	// Make HTTP request with authentication
	// Implementation omitted for brevity

	return nil, "", fmt.Errorf("not implemented")
}

// Close closes the DingTalk channel.
func (d *DingTalkChannel) Close() error {
	return nil
}

// Helper methods

func (d *DingTalkChannel) sendText(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	// Use outgoing webhook or conversation API
	// Implementation depends on DingTalk robot type

	// For webhook-based robots:
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": msg.Content,
		},
	}

	return d.sendWebhook(ctx, msg.PlatformChatID, payload)
}

func (d *DingTalkChannel) sendMedia(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	// DingTalk media messages require special handling
	// Implementation depends on media type

	return fmt.Errorf("media sending not yet implemented")
}

func (d *DingTalkChannel) sendWebhook(ctx context.Context, url string, payload interface{}) error {
	// Send to DingTalk webhook
	// Implementation omitted

	return nil
}

func (d *DingTalkChannel) computeSignature(timestamp, body string) string {
	// DingTalk signature: base64(hmac_sha256(timestamp + "\n" + secret, body))
	stringToSign := timestamp + "\n" + body

	h := hmac.New(sha256.New, []byte(d.config.AppSecret))
	h.Write([]byte(stringToSign))

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GetAccessToken retrieves an access token from DingTalk.
func (d *DingTalkChannel) GetAccessToken(ctx context.Context) (string, error) {
	// Implementation: call gettoken API
	// GET https://oapi.dingtalk.com/gettoken?appkey=xxx&appsecret=xxx
	return "", fmt.Errorf("not implemented")
}

// DingTalkMessage represents a message from DingTalk.
type DingTalkMessage struct {
	ChatType      string `json:"chatType"`
	MsgID         string `json:"msgId"`
	SenderNick    string `json:"senderNick"`
	SenderStaffID string `json:"senderStaffId"`
	MsgType       string `json:"msgtype"`
	CreateAt      int64  `json:"createAt"`

	Text  DingTalkTextContent  `json:"text"`
	Image DingTalkMediaContent `json:"image"`
	Audio DingTalkMediaContent `json:"audio"`
	Video DingTalkMediaContent `json:"video"`
	File  DingTalkFileContent  `json:"file"`
}

type DingTalkTextContent struct {
	Content string `json:"content"`
}

type DingTalkMediaContent struct {
	MediaID string `json:"media_id"`
}

type DingTalkFileContent struct {
	MediaID  string `json:"media_id"`
	FileName string `json:"file_name"`
}

// Ensure DingTalkChannel implements ChatChannel
var _ channels.ChatChannel = (*DingTalkChannel)(nil)
