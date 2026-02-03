// Package whatsapp implements WhatsApp integration via Baileys Node.js bridge.
package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
)

// BaileysBridgeClient communicates with the Node.js Baileys bridge service.
type BaileysBridgeClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewBaileysBridgeClient creates a new client for the Baileys bridge.
func NewBaileysBridgeClient(bridgeURL, apiKey string) *BaileysBridgeClient {
	return &BaileysBridgeClient{
		baseURL: bridgeURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 0, // No timeout for streaming connections
		},
	}
}

// WhatsAppMessage represents a message from WhatsApp.
type WhatsAppMessage struct {
	Key     WhatsAppKey `json:"key"`
	Message struct {
		Conversation string `json:"conversation"` // Text content
	} `json:"message"`
	MessageType string `json:"messageType"` // "conversation", "imageMessage", etc.
}

type WhatsAppKey struct {
	RemoteJID string `json:"remoteJid"` // Phone number with @s.whatsapp.net
	FromMe    bool   `json:"fromMe"`
	ID        string `json:"id"`
}

// WhatsAppChannel implements ChatChannel for WhatsApp via Baileys.
type WhatsAppChannel struct {
	bridge *BaileysBridgeClient
}

// NewWhatsAppChannel creates a new WhatsApp channel.
func NewWhatsAppChannel(bridgeURL, apiKey string) (*WhatsAppChannel, error) {
	bridge := NewBaileysBridgeClient(bridgeURL, apiKey)

	// Verify bridge is running
	if err := bridge.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("baileys bridge not reachable: %w", err)
	}

	return &WhatsAppChannel{
		bridge: bridge,
	}, nil
}

// Name returns the platform name.
func (w *WhatsAppChannel) Name() chat_apps.Platform {
	return chat_apps.PlatformWhatsApp
}

// ValidateWebhook verifies the incoming webhook from the Baileys bridge.
func (w *WhatsAppChannel) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) error {
	// The bridge should sign requests, verify signature here
	return nil
}

// ParseMessage parses the incoming webhook payload.
func (w *WhatsAppChannel) ParseMessage(ctx context.Context, payload []byte) (*chat_apps.IncomingMessage, error) {
	var waMsg WhatsAppMessage
	if err := json.Unmarshal(payload, &waMsg); err != nil {
		return nil, channels.ErrInvalidPayload
	}

	msg := &chat_apps.IncomingMessage{
		Platform:       chat_apps.PlatformWhatsApp,
		PlatformUserID: waMsg.Key.RemoteJID,
		PlatformChatID: waMsg.Key.RemoteJID,
		Content:        waMsg.Message.Conversation,
		Metadata:       make(map[string]string),
	}

	// Store metadata
	msg.Metadata["message_id"] = waMsg.Key.ID
	msg.Metadata["from_me"] = fmt.Sprintf("%v", waMsg.Key.FromMe)

	// Determine message type
	switch waMsg.MessageType {
	case "imageMessage":
		msg.Type = chat_apps.MessageTypePhoto
	case "audioMessage", "pttMessage": // ptt = push-to-talk (voice note)
		msg.Type = chat_apps.MessageTypeAudio
	case "videoMessage":
		msg.Type = chat_apps.MessageTypeVideo
	case "documentMessage":
		msg.Type = chat_apps.MessageTypeDocument
	case "conversation":
		msg.Type = chat_apps.MessageTypeText
	default:
		msg.Type = chat_apps.MessageTypeText
	}

	return msg, nil
}

// SendMessage sends a message to WhatsApp.
func (w *WhatsAppChannel) SendMessage(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	return w.bridge.SendMessage(ctx, &SendMessageRequest{
		JID:      msg.PlatformChatID,
		Type:     msg.Type.String(),
		Content:  msg.Content,
		Media:    msg.MediaData,
		MimeType: msg.MimeType,
		FileName: msg.FileName,
	})
}

// SendChunkedMessage sends streaming content chunks.
func (w *WhatsAppChannel) SendChunkedMessage(ctx context.Context, chatID string, chunks <-chan string) error {
	// WhatsApp doesn't support message editing
	// Send each chunk as a separate message or accumulate
	return w.bridge.SendStream(ctx, chatID, chunks)
}

// DownloadMedia downloads media from WhatsApp.
func (w *WhatsAppChannel) DownloadMedia(ctx context.Context, url string) ([]byte, string, error) {
	return w.bridge.DownloadMedia(ctx, url)
}

// Close closes the WhatsApp channel.
func (w *WhatsAppChannel) Close() error {
	return nil
}

// Bridge API methods

// SendMessageRequest sends a message via the bridge.
type SendMessageRequest struct {
	JID      string `json:"jid"`
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	Media    []byte `json:"media,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

// HealthCheck verifies the bridge is running and WhatsApp is connected.
func (b *BaileysBridgeClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", b.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	if b.apiKey != "" {
		req.Header.Set("x-bridge-api-key", b.apiKey)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	// Parse response to verify WhatsApp connection status
	var result struct {
		Status    string `json:"status"`    // "connected", "disconnected", etc.
		Connected bool   `json:"connected"` // Direct boolean flag
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If we can't parse the response, at least we know the bridge is running
		slog.Debug("whatsapp: could not parse health check response", "error", err)
		return nil
	}

	// Check if WhatsApp is actually connected
	if result.Status == "disconnected" || (result.Status == "" && !result.Connected) {
		return fmt.Errorf("whatsapp not connected: bridge is running but WhatsApp session is not active")
	}

	return nil
}

// SendMessage sends a message via the bridge.
func (b *BaileysBridgeClient) SendMessage(ctx context.Context, req *SendMessageRequest) error {
	// Serialize request
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/send", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		httpReq.Header.Set("x-bridge-api-key", b.apiKey)
	}
	httpReq.Body = io.NopCloser(strings.NewReader(string(data)))

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send failed: status %d", resp.StatusCode)
	}

	return nil
}

// SendStream sends streaming chunks via the bridge.
func (b *BaileysBridgeClient) SendStream(ctx context.Context, jid string, chunks <-chan string) error {
	// Use SSE or WebSocket for streaming
	// Implementation depends on bridge protocol

	return nil
}

// DownloadMedia downloads media from WhatsApp.
func (b *BaileysBridgeClient) DownloadMedia(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.baseURL+"/download?url="+url, nil)
	if err != nil {
		return nil, "", err
	}

	if b.apiKey != "" {
		req.Header.Set("x-bridge-api-key", b.apiKey)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	return data, mimeType, nil
}

// Ensure WhatsAppChannel implements ChatChannel
var _ channels.ChatChannel = (*WhatsAppChannel)(nil)
