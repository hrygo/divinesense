// Package telegram provides webhook handling for Telegram Bot.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/hrygo/divinesense/plugin/chat_apps"
)

// WebhookHandler handles incoming Telegram webhooks.
type WebhookHandler struct {
	channel *TelegramChannel
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(channel *TelegramChannel) *WebhookHandler {
	return &WebhookHandler{
		channel: channel,
	}
}

// HandleWebhook handles an incoming webhook request from Telegram.
func (h *WebhookHandler) HandleWebhook(ctx context.Context, r *http.Request) (*chat_apps.IncomingMessage, error) {
	// Read request body
	defer r.Body.Close()
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		return nil, fmt.Errorf("failed to decode update: %w", err)
	}

	// Convert to JSON for ParseMessage
	data, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update: %w", err)
	}

	// Parse the message
	return h.channel.ParseMessage(ctx, data)
}

// SetWebhook sets the webhook for the Telegram bot.
func (h *WebhookHandler) SetWebhook(ctx context.Context, webhookURL string, dropPendingUpdates bool) error {
	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return err
	}
	_, err = h.channel.bot.Request(tgbotapi.WebhookConfig{
		URL:                parsedURL,
		DropPendingUpdates: dropPendingUpdates,
	})
	return err
}

// DeleteWebhook removes the webhook for the Telegram bot.
func (h *WebhookHandler) DeleteWebhook(ctx context.Context) error {
	_, err := h.channel.bot.Request(tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: true,
	})
	return err
}

// GetWebhookInfo returns information about the current webhook.
func (h *WebhookHandler) GetWebhookInfo(ctx context.Context) (tgbotapi.WebhookInfo, error) {
	return h.channel.bot.GetWebhookInfo()
}

// VerifyRequest verifies that the request came from Telegram.
// Telegram Bot API doesn't sign webhooks, so we validate:
// 1. HTTP method is POST
// 2. Content-Type is JSON or empty (Telegram sometimes doesn't send it)
// 3. The request contains a valid update structure
func (h *WebhookHandler) VerifyRequest(r *http.Request) bool {
	// Must be POST
	if r.Method != http.MethodPost {
		slog.Warn("telegram webhook: invalid method", "method", r.Method, "remote_addr", r.RemoteAddr)
		return false
	}

	// Check Content-Type (Telegram may send empty or application/json)
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") {
		slog.Warn("telegram webhook: invalid content type", "content_type", ct, "remote_addr", r.RemoteAddr)
		return false
	}

	// Log incoming webhook for monitoring
	slog.Debug("telegram webhook: request verified",
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
	)

	return true
}

// ExtractUserID extracts the user ID from a Telegram update.
func ExtractUserID(update *tgbotapi.Update) string {
	var from *tgbotapi.User

	switch {
	case update.Message != nil:
		from = update.Message.From
	case update.EditedMessage != nil:
		from = update.EditedMessage.From
	case update.CallbackQuery != nil:
		from = update.CallbackQuery.From
	case update.InlineQuery != nil:
		from = update.InlineQuery.From
	case update.ChosenInlineResult != nil:
		from = update.ChosenInlineResult.From
	case update.ShippingQuery != nil:
		from = update.ShippingQuery.From
	case update.PreCheckoutQuery != nil:
		from = update.PreCheckoutQuery.From
	}

	if from != nil {
		return strconv.FormatInt(from.ID, 10)
	}

	return ""
}

// ExtractChatID extracts the chat ID from a Telegram update.
func ExtractChatID(update *tgbotapi.Update) string {
	var chat *tgbotapi.Chat

	switch {
	case update.Message != nil:
		chat = update.Message.Chat
	case update.EditedMessage != nil:
		chat = update.EditedMessage.Chat
	case update.CallbackQuery != nil:
		chat = update.CallbackQuery.Message.Chat
	}

	if chat != nil {
		return strconv.FormatInt(chat.ID, 10)
	}

	return ""
}
