// Package telegram implements the Telegram Bot channel.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/hrygo/divinesense/plugin/chat_apps"
	"github.com/hrygo/divinesense/plugin/chat_apps/channels"
)

const (
	MaxPhotoSizeMB    = 20 // Telegram photo size limit
	MaxDocumentSizeMB = 50 // Telegram document size limit
	MaxAudioSizeMB    = 50 // Telegram audio file size limit
	DefaultParseMode  = "Markdown"
)

// TelegramConfig holds configuration for the Telegram channel.
type TelegramConfig struct {
	BotToken string
}

// TelegramChannel implements ChatChannel for Telegram Bot API.
type TelegramChannel struct {
	bot    *tgbotapi.BotAPI
	config *TelegramConfig
	client *http.Client
}

// NewTelegramChannel creates a new Telegram channel.
func NewTelegramChannel(config *TelegramConfig) (*TelegramChannel, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &TelegramChannel{
		bot:    bot,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableCompression: true,
			},
		},
	}, nil
}

// Name returns the platform name.
func (t *TelegramChannel) Name() chat_apps.Platform {
	return chat_apps.PlatformTelegram
}

// ValidateWebhook verifies the incoming webhook request.
func (t *TelegramChannel) ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) error {
	// Telegram Bot API handles webhook verification internally
	return nil
}

// ParseMessage parses the incoming webhook payload into an IncomingMessage.
func (t *TelegramChannel) ParseMessage(ctx context.Context, payload []byte) (*chat_apps.IncomingMessage, error) {
	var update tgbotapi.Update
	if err := json.Unmarshal(payload, &update); err != nil {
		slog.Warn("telegram: failed to parse webhook payload", "error", err)
		return nil, channels.ErrInvalidPayload
	}

	// Extract message from update
	var tgMsg *tgbotapi.Message
	switch {
	case update.Message != nil:
		tgMsg = update.Message
	case update.EditedMessage != nil:
		tgMsg = update.EditedMessage
	case update.CallbackQuery != nil:
		tgMsg = update.CallbackQuery.Message
	default:
		return nil, channels.ErrInvalidPayload
	}

	if tgMsg == nil {
		return nil, channels.ErrInvalidPayload
	}

	msg := &chat_apps.IncomingMessage{
		Platform:       chat_apps.PlatformTelegram,
		PlatformUserID: strconv.FormatInt(tgMsg.From.ID, 10),
		PlatformChatID: strconv.FormatInt(tgMsg.Chat.ID, 10),
		Content:        tgMsg.Text,
		Timestamp:      time.Now(),
		Metadata:       make(map[string]string),
	}

	// Store metadata
	msg.Metadata["update_id"] = strconv.Itoa(update.UpdateID)
	msg.Metadata["username"] = tgMsg.From.UserName
	msg.Metadata["language_code"] = tgMsg.From.LanguageCode

	// Handle different message types
	switch {
	case len(tgMsg.Photo) > 0:
		msg.Type = chat_apps.MessageTypePhoto
		photos := tgMsg.Photo
		largest := photos[len(photos)-1]
		msg.MediaURL = fmt.Sprintf("telegram://file/%s", largest.FileID)
		msg.Content = tgMsg.Caption

	case tgMsg.Voice != nil:
		msg.Type = chat_apps.MessageTypeAudio
		msg.MediaURL = fmt.Sprintf("telegram://file/%s", tgMsg.Voice.FileID)
		msg.MimeType = "audio/ogg"

	case tgMsg.Audio != nil:
		msg.Type = chat_apps.MessageTypeAudio
		msg.MediaURL = fmt.Sprintf("telegram://file/%s", tgMsg.Audio.FileID)
		msg.MimeType = tgMsg.Audio.MimeType
		msg.FileName = tgMsg.Audio.FileName

	case tgMsg.Video != nil:
		msg.Type = chat_apps.MessageTypeVideo
		msg.MediaURL = fmt.Sprintf("telegram://file/%s", tgMsg.Video.FileID)
		msg.MimeType = tgMsg.Video.MimeType

	case tgMsg.Document != nil:
		msg.Type = chat_apps.MessageTypeDocument
		msg.MediaURL = fmt.Sprintf("telegram://file/%s", tgMsg.Document.FileID)
		msg.MimeType = tgMsg.Document.MimeType
		msg.FileName = tgMsg.Document.FileName

	default:
		msg.Type = chat_apps.MessageTypeText
	}

	return msg, nil
}

// SendMessage sends a message to Telegram.
func (t *TelegramChannel) SendMessage(ctx context.Context, msg *chat_apps.OutgoingMessage) error {
	slog.Debug("telegram: sending message",
		"chat_id", msg.PlatformChatID,
		"type", msg.Type,
	)

	chatID, err := strconv.ParseInt(msg.PlatformChatID, 10, 64)
	if err != nil {
		slog.Error("telegram: invalid chat ID", "chat_id", msg.PlatformChatID, "error", err)
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	switch msg.Type {
	case chat_apps.MessageTypePhoto:
		return t.sendPhoto(ctx, chatID, msg)
	case chat_apps.MessageTypeAudio:
		return t.sendAudio(ctx, chatID, msg)
	case chat_apps.MessageTypeVideo:
		return t.sendVideo(ctx, chatID, msg)
	case chat_apps.MessageTypeDocument:
		return t.sendDocument(ctx, chatID, msg)
	default:
		return t.sendText(ctx, chatID, msg)
	}
}

// SendChunkedMessage sends streaming content chunks.
func (t *TelegramChannel) SendChunkedMessage(ctx context.Context, chatID string, chunks <-chan string) error {
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Accumulate and send as single message
	var builder strings.Builder
	for chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			builder.WriteString(chunk)
		}
	}

	fullContent := builder.String()
	tgMsg := tgbotapi.NewMessage(id, fullContent)
	tgMsg.ParseMode = DefaultParseMode
	_, err = t.bot.Send(tgMsg)
	return err
}

// DownloadMedia downloads a file from Telegram.
func (t *TelegramChannel) DownloadMedia(ctx context.Context, fileID string) ([]byte, string, error) {
	// Get file info from Telegram
	file, err := t.bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		slog.Error("telegram: failed to get file info", "file_id", fileID, "error", err)
		return nil, "", fmt.Errorf("%w: %w", channels.ErrMediaDownloadFailed, err)
	}

	// file.Link is a method that takes the bot token
	fileURL := file.Link(t.bot.Token)
	if fileURL == "" {
		return nil, "", fmt.Errorf("empty file link from Telegram")
	}

	// Download file using the Link method
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		slog.Error("telegram: failed to download file", "url", fileURL, "error", err)
		return nil, "", fmt.Errorf("%w: %w", channels.ErrMediaDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("telegram: non-200 status downloading file", "status", resp.StatusCode)
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

	slog.Debug("telegram: downloaded media",
		"file_id", fileID,
		"size", len(data),
		"mime_type", mimeType,
	)

	return data, mimeType, nil
}

// Close closes the Telegram channel.
func (t *TelegramChannel) Close() error {
	return nil
}

// Helper methods

func (t *TelegramChannel) sendText(ctx context.Context, chatID int64, msg *chat_apps.OutgoingMessage) error {
	tgMsg := tgbotapi.NewMessage(chatID, msg.Content)
	if msg.ParseMode != "" {
		tgMsg.ParseMode = msg.ParseMode
	}
	_, err := t.bot.Send(tgMsg)
	return err
}

func (t *TelegramChannel) sendPhoto(ctx context.Context, chatID int64, msg *chat_apps.OutgoingMessage) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  msg.FileName,
		Bytes: msg.MediaData,
	})
	photo.Caption = msg.Content
	if msg.ParseMode != "" {
		photo.ParseMode = msg.ParseMode
	}
	_, err := t.bot.Send(photo)
	return err
}

func (t *TelegramChannel) sendAudio(ctx context.Context, chatID int64, msg *chat_apps.OutgoingMessage) error {
	audio := tgbotapi.NewAudio(chatID, tgbotapi.FileBytes{
		Name:  msg.FileName,
		Bytes: msg.MediaData,
	})
	audio.Caption = msg.Content
	_, err := t.bot.Send(audio)
	return err
}

func (t *TelegramChannel) sendVideo(ctx context.Context, chatID int64, msg *chat_apps.OutgoingMessage) error {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FileBytes{
		Name:  msg.FileName,
		Bytes: msg.MediaData,
	})
	video.Caption = msg.Content
	if msg.ParseMode != "" {
		video.ParseMode = msg.ParseMode
	}
	_, err := t.bot.Send(video)
	return err
}

func (t *TelegramChannel) sendDocument(ctx context.Context, chatID int64, msg *chat_apps.OutgoingMessage) error {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  msg.FileName,
		Bytes: msg.MediaData,
	})
	doc.Caption = msg.Content
	_, err := t.bot.Send(doc)
	return err
}

// Ensure TelegramChannel implements ChatChannel
var _ channels.ChatChannel = (*TelegramChannel)(nil)
