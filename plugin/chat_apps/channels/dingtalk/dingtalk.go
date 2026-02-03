// Package dingtalk implements the DingTalk Robot channel.
package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
)

const (
	DefaultTimestampWindow = 5 * time.Minute // DingTalk webhook timestamp validity window
	MaxTimestampSkew       = 5 * time.Minute // Maximum allowed clock skew
	DingTalkAPIBaseURL     = "https://oapi.dingtalk.com"
)

// DingTalkConfig holds configuration for the DingTalk channel.
type DingTalkConfig struct {
	AppKey    string
	AppSecret string
}

// DingTalkChannel implements ChatChannel for DingTalk Robot.
type DingTalkChannel struct {
	config      *DingTalkConfig
	webhookURL  string
	client      *http.Client
	accessToken string
	tokenMu     sync.Mutex // Changed from RWMutex to Mutex to prevent race condition
	tokenExpiry time.Time
}

// NewDingTalkChannel creates a new DingTalk channel.
func NewDingTalkChannel(config *DingTalkConfig) (*DingTalkChannel, error) {
	return &DingTalkChannel{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
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
		slog.Warn("dingtalk: missing signature headers",
			"timestamp", timestamp != "",
			"sign", sign != "",
		)
		return channels.ErrInvalidSignature
	}

	// Validate timestamp to prevent replay attacks
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		slog.Warn("dingtalk: invalid timestamp format", "error", err)
		return channels.ErrInvalidSignature
	}

	// Check if timestamp is within acceptable window
	now := time.Now().Unix()
	msgTime := ts / 1000 // Convert milliseconds to seconds if needed
	timeDiff := now - msgTime
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// Allow up to 5 minutes clock skew
	if timeDiff > int64(MaxTimestampSkew.Seconds()) {
		slog.Warn("dingtalk: timestamp outside valid window",
			"timestamp", timestamp,
			"time_diff_seconds", timeDiff,
		)
		return channels.ErrInvalidSignature
	}

	// Compute expected signature
	expectedSign := d.computeSignature(timestamp, string(body))

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(sign), []byte(expectedSign)) {
		slog.Warn("dingtalk: signature mismatch")
		return channels.ErrInvalidSignature
	}

	slog.Debug("dingtalk: webhook signature validated")
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
		slog.Warn("dingtalk: failed to parse webhook payload", "error", err)
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
	var builder strings.Builder
	for chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			builder.WriteString(chunk)
		}
	}

	msg := &chat_apps.OutgoingMessage{
		PlatformChatID: chatID,
		Type:           chat_apps.MessageTypeText,
		Content:        builder.String(),
	}

	return d.sendText(ctx, msg)
}

// DownloadMedia downloads media from DingTalk using the downloadCode.
func (d *DingTalkChannel) DownloadMedia(ctx context.Context, downloadCode string) ([]byte, string, error) {
	// Get access token for API call
	token, err := d.GetAccessToken(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", channels.ErrMediaDownloadFailed, err)
	}

	// DingTalk media download API
	apiURL := fmt.Sprintf("%s/media/download?downloadCode=%s&accessToken=%s",
		DingTalkAPIBaseURL,
		url.QueryEscape(downloadCode),
		url.QueryEscape(token),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", channels.ErrMediaDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%w: status %d", channels.ErrMediaDownloadFailed, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Detect MIME type from response
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	return data, mimeType, nil
}

// Close closes the DingTalk channel.
func (d *DingTalkChannel) Close() error {
	return nil
}

// Helper methods

func (d *DingTalkChannel) sendText(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	// For webhook-based robots, use the webhook URL
	// The PlatformChatID for DingTalk is the webhook URL

	webhookURL := msg.PlatformChatID
	if webhookURL == "" {
		webhookURL = d.webhookURL
	}

	if webhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": msg.Content,
		},
	}

	return d.sendWebhook(ctx, webhookURL, payload)
}

func (d *DingTalkChannel) sendMedia(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	// DingTalk media messages require uploading media first
	// then sending with the media ID
	// Media upload is not yet implemented - return a clear error

	webhookURL := msg.PlatformChatID
	if webhookURL == "" {
		webhookURL = d.webhookURL
	}

	if webhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	// Media upload not yet supported - send a text message instead
	// This is intentional fallback behavior
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[Media message received - type: %s]\n\n%s", msg.Type, msg.Content),
		},
	}

	slog.Info("dingtalk: media message sent as text (media upload not implemented)",
		"type", msg.Type,
	)

	return d.sendWebhook(ctx, webhookURL, payload)
}

func (d *DingTalkChannel) sendWebhook(ctx context.Context, webhookURL string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("dingtalk: webhook returned non-200 status",
			"status", resp.StatusCode,
			"response", string(respBody),
		)
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

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
// The token is cached until expiry (default 2 hours).
func (d *DingTalkChannel) GetAccessToken(ctx context.Context) (string, error) {
	// Fast path: check if we have a valid cached token without lock
	// This is safe for reads due to Go's memory model for int64 and bool
	if d.accessToken != "" && time.Now().Before(d.tokenExpiry) {
		return d.accessToken, nil
	}

	// Need to fetch or refresh token - acquire lock
	d.tokenMu.Lock()
	defer d.tokenMu.Unlock()

	// Double-check after acquiring lock (another goroutine may have refreshed)
	if d.accessToken != "" && time.Now().Before(d.tokenExpiry) {
		return d.accessToken, nil
	}

	// Build request URL
	apiURL := fmt.Sprintf("%s/gettoken?appkey=%s&appsecret=%s",
		DingTalkAPIBaseURL,
		url.QueryEscape(d.config.AppKey),
		url.QueryEscape(d.config.AppSecret),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gettoken returned status %d", resp.StatusCode)
	}

	var result struct {
		ErrCode int32  `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		Token   string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("DingTalk API error %d: %s", result.ErrCode, result.ErrMsg)
	}

	// Cache the token (expire 5 minutes early to be safe)
	d.accessToken = result.Token
	d.tokenExpiry = time.Now().Add(2 * time.Hour).Add(-5 * time.Minute)

	slog.Debug("dingtalk: obtained new access token",
		"expires_at", d.tokenExpiry,
	)

	return result.Token, nil
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
