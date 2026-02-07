# Chat Apps Integration - Technical Specification

> **Issue**: [#53](https://github.com/hrygo/divinesense/issues/53)
> **Version**: 1.0
> **Status**: DRAFT
> **Created**: 2026-02-03

---

## 1. Overview

Add multi-platform Chat Apps integration to DivineSense, enabling users to interact with AI agents through Telegram, WhatsApp, and DingTalk bots.

### 1.1 Goals

- Enable seamless access to DivineSense AI agents from popular chat platforms
- Support multimedia messages (text, images, voice, files)
- Maintain conversation context across channels
- Leverage existing AI infrastructure (Parrot agents, SSE streaming)

### 1.2 Non-Goals

- Group chat @mention triggering (future enhancement)
- Cross-platform message synchronization (future enhancement)
- Custom bot command menus (future enhancement)

---

## 2. Architecture

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Chat Apps Gateway                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │   Telegram   │  │   WhatsApp   │  │      DingTalk        │ │
│  │   Channel    │  │   Channel    │  │      Channel         │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────────────┘ │
└─────────┼──────────────────┼───────────────────┼────────────────┘
          │                  │                   │
          └──────────────────┼───────────────────┘
                             ▼
                  ┌─────────────────────┐
                  │  ChatChannel Router │
                  │  (user verification)│
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   MediaHandler      │
                  │ (OCR/Voice-to-Text) │
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   ChatRouter        │
                  │  (existing AI agent) │
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   Parrot Agents     │
                  │ (Memo/Schedule/...) │
                  └─────────────────────┘
```

### 2.2 Data Flow

```
User Message (Chat App)
    ↓
Platform Webhook
    ↓
ChatChannel.{Receive}()
    ↓ [Verify User]
ChatChannelRouter.Route()
    ↓ [Extract Media]
MediaHandler.Process()
    ↓ [Convert to Text]
ChatRouter.Route()
    ↓
Parrot Agent.Execute()
    ↓ [Stream Response]
ChatChannel.Send()
    ↓
User receives message
```

---

## 3. Database Schema

### 3.1 New Table: `chat_app_credential`

```sql
-- Migration: 20260203_chat_apps_credential.up.sql

CREATE TABLE chat_app_credential (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,              -- 'telegram', 'whatsapp', 'dingtalk'
  platform_user_id TEXT NOT NULL,      -- Platform-specific user ID
  platform_chat_id TEXT,               -- Platform-specific chat ID (for DM)
  access_token TEXT,                   -- Encrypted access token
  webhook_url TEXT,                    -- Webhook URL (DingTalk)
  enabled BOOLEAN DEFAULT true,
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  UNIQUE(user_id, platform)
);

CREATE INDEX idx_chat_app_credential_user ON chat_app_credential(user_id);
CREATE INDEX idx_chat_app_credential_platform ON chat_app_credential(platform);
```

### 3.2 Modify Existing Table: `conversation_context`

```sql
-- Migration: 20260203_conversation_channel_type.up.sql

ALTER TABLE conversation_context
ADD COLUMN channel_type TEXT DEFAULT 'web';

-- Valid values: 'web', 'telegram', 'whatsapp', 'dingtalk'
ALTER TABLE conversation_context
ADD CONSTRAINT check_channel_type
CHECK (channel_type IN ('web', 'telegram', 'whatsapp', 'dingtalk'));
```

---

## 4. API Design

### 4.1 Proto Definition: `proto/api/v1/chat_app_service.proto`

```protobuf
syntax = "proto3";

package memos.api.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "google/protobuf/empty.proto";

option go_package = "gen/api/v1";

// ChatAppService manages chat app integrations.
service ChatAppService {
  // RegisterCredential binds a chat app account to a user.
  rpc RegisterCredential(RegisterCredentialRequest) returns (Credential) {
    option (google.api.http) = {
      post: "/api/v1/chat-apps/credentials"
      body: "*"
    };
  }

  // ListCredentials returns all registered chat app credentials for the user.
  rpc ListCredentials(ListCredentialsRequest) returns (ListCredentialsResponse) {
    option (google.api.http) = {
      get: "/api/v1/chat-apps/credentials"
    };
  }

  // DeleteCredential removes a chat app binding.
  rpc DeleteCredential(DeleteCredentialRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/chat-apps/credentials/{platform}"
    };
  }

  // HandleWebhook processes incoming webhook from chat platforms.
  rpc HandleWebhook(WebhookRequest) returns (WebhookResponse) {
    option (google.api.http) = {
      post: "/api/v1/chat-apps/webhook/{platform}"
      body: "*"
    };
  }

  // SendMessage sends a message to a chat app channel.
  rpc SendMessage(SendMessageRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/api/v1/chat-apps/send"
      body: "*"
    };
  }
}

// Platform represents the supported chat platforms.
enum Platform {
  PLATFORM_UNSPECIFIED = 0;
  PLATFORM_TELEGRAM = 1;
  PLATFORM_WHATSAPP = 2;
  PLATFORM_DINGTALK = 3;
}

// RegisterCredentialRequest creates a new chat app credential.
message RegisterCredentialRequest {
  Platform platform = 1 [(google.api.field_behavior) = REQUIRED];
  string platform_user_id = 2 [(google.api.field_behavior) = REQUIRED];
  string platform_chat_id = 3;
  string access_token = 4;
  string webhook_url = 5;
}

// Credential represents a chat app binding.
message Credential {
  int32 id = 1;
  int32 user_id = 2;
  Platform platform = 3;
  string platform_user_id = 4;
  string platform_chat_id = 5;
  bool enabled = 6;
  int64 created_ts = 7;
  int64 updated_ts = 8;
  // Token is never returned for security
}

// ListCredentialsRequest queries credentials.
message ListCredentialsRequest {
  Platform platform = 1; // Filter by platform (optional)
}

// ListCredentialsResponse contains credential list.
message ListCredentialsResponse {
  repeated Credential credentials = 1;
}

// DeleteCredentialRequest removes a credential.
message DeleteCredentialRequest {
  Platform platform = 1 [(google.api.field_behavior) = REQUIRED];
}

// WebhookRequest is an incoming webhook from a chat platform.
message WebhookRequest {
  Platform platform = 1 [(google.api.field_behavior) = REQUIRED];
  bytes payload = 2; // Raw webhook payload (platform-specific format)
  map<string, string> headers = 3; // HTTP headers for signature verification
  string query_string = 4; // Raw query string
}

// WebhookResponse is the response to a webhook.
message WebhookResponse {
  bool success = 1;
  string message = 2;
}

// SendMessageRequest sends a message to a chat app.
message SendMessageRequest {
  Platform platform = 1 [(google.api.field_behavior) = REQUIRED];
  string platform_chat_id = 2 [(google.api.field_behavior) = REQUIRED];
  string content = 3 [(google.api.field_behavior) = REQUIRED];
  MessageType message_type = 4; // Default: TEXT
  bytes media_data = 5; // For media messages
  string media_mime_type = 6;
}

// MessageType specifies the type of message to send.
enum MessageType {
  MESSAGE_TYPE_UNSPECIFIED = 0;
  MESSAGE_TYPE_TEXT = 1;
  MESSAGE_TYPE_PHOTO = 2;
  MESSAGE_TYPE_AUDIO = 3;
  MESSAGE_TYPE_VIDEO = 4;
  MESSAGE_TYPE_DOCUMENT = 5;
}
```

### 4.2 Store Interface: `proto/store/chat_app_service.proto`

```protobuf
syntax = "proto3";

package memos.store;

option go_package = "gen/store";

// ChatAppService manages chat app credentials in the database.
service ChatAppService {
  // CreateCredential creates a new credential.
  rpc CreateCredential(CreateCredentialRequest) returns (Credential) {
    option (google.api.http) = {};
  }

  // ListCredentials lists all credentials for a user.
  rpc ListCredentials(ListCredentialsRequest) returns (ListCredentialsResponse) {
    option (google.api.http) = {};
  }

  // GetCredentialByPlatform gets a credential by user ID and platform.
  rpc GetCredentialByPlatform(GetCredentialByPlatformRequest) returns (Credential) {
    option (google.api.http) = {};
  }

  // DeleteCredential deletes a credential.
  rpc DeleteCredential(DeleteCredentialRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {};
  }

  // UpdateCredential updates a credential.
  rpc UpdateCredential(UpdateCredentialRequest) returns (Credential) {
    option (google.api.http) = {};
  }
}

message CreateCredentialRequest {
  int32 user_id = 1;
  string platform = 2;
  string platform_user_id = 3;
  string platform_chat_id = 4;
  string access_token = 5;
  string webhook_url = 6;
}

message Credential {
  int32 id = 1;
  int32 user_id = 2;
  string platform = 3;
  string platform_user_id = 4;
  string platform_chat_id = 5;
  string access_token = 6;
  string webhook_url = 7;
  bool enabled = 8;
  int64 created_ts = 9;
  int64 updated_ts = 10;
}

message ListCredentialsRequest {
  int32 user_id = 1;
  optional string platform = 2;
}

message ListCredentialsResponse {
  repeated Credential credentials = 1;
}

message GetCredentialByPlatformRequest {
  int32 user_id = 1;
  string platform = 2;
}

message DeleteCredentialRequest {
  int32 id = 1;
}

message UpdateCredentialRequest {
  int32 id = 1;
  optional bool enabled = 2;
  optional string access_token = 3;
  optional string webhook_url = 4;
  optional int64 updated_ts = 5;
}
```

---

## 5. Component Structure

```
plugin/chat_apps/
├── channels/
│   ├── base.go                    # ChatChannel interface definition
│   ├── router.go                  # Channel router (user verification)
│   ├── telegram/
│   │   ├── telegram.go            # Telegram Bot implementation
│   │   ├── types.go               # Telegram API types
│   │   └── webhook.go             # Webhook handler
│   ├── whatsapp/
│   │   ├── bridge.go              # WhatsApp Baileys bridge
│   │   ├── protocol.go            # Communication protocol
│   │   └── client.go              # Node.js bridge client
│   └── dingtalk/
│       ├── dingtalk.go            # DingTalk bot implementation
│       ├── crypto.go              # Signature verification
│       └── types.go               # DingTalk API types
├── media/
│   ├── handler.go                 # MediaHandler interface
│   ├── whisper.go                 # Voice-to-text (OpenAI Whisper)
│   ├── ocr.go                     # Image OCR (Tesseract or API)
│   └── types.go                   # Media types
├── store/
│   ├── db.go                      # Database operations
│   └── crypto.go                  # Token encryption/decryption
└── types.go                       # Common types

server/router/api/v1/chat_apps/
├── handler.go                     # HTTP handlers
├── middleware.go                  # Auth middleware
└── webhook_handler.go             # Webhook handlers
```

---

## 6. Interface Definitions

### 6.1 ChatChannel Interface

```go
// plugin/chat_apps/channels/base.go

package channels

import (
    "context"
    "io"
)

// MessageType represents the type of message.
type MessageType int

const (
    MessageTypeText MessageType = iota
    MessageTypePhoto
    MessageTypeAudio
    MessageTypeVideo
    MessageTypeDocument
)

// IncomingMessage represents a message from a chat platform.
type IncomingMessage struct {
    Platform      string      // "telegram", "whatsapp", "dingtalk"
    PlatformUserID string      // Platform-specific user ID
    PlatformChatID string      // Platform-specific chat ID
    Type          MessageType
    Content       string      // Text content
    MediaURL      string      // URL for media download
    MediaData     []byte      // Downloaded media data
    FileName      string      // Original filename
    MimeType      string      // MIME type
    Metadata      map[string]string // Additional metadata
}

// OutgoingMessage represents a message to send to a chat platform.
type OutgoingMessage struct {
    PlatformChatID string
    Type          MessageType
    Content       string
    MediaData     []byte
    MimeType      string
    FileName      string
}

// ChatChannel defines the interface for all chat platform integrations.
type ChatChannel interface {
    // Name returns the platform name.
    Name() string

    // ValidateWebhook verifies the incoming webhook request.
    ValidateWebhook(ctx context.Context, headers map[string]string, body []byte) error

    // ParseMessage parses the incoming webhook into an IncomingMessage.
    ParseMessage(ctx context.Context, payload []byte) (*IncomingMessage, error)

    // SendMessage sends a message to the chat platform.
    SendMessage(ctx context.Context, msg *OutgoingMessage) error

    // SendChunkedMessage sends streaming content chunks.
    SendChunkedMessage(ctx context.Context, chatID string, chunks <-chan string) error

    // DownloadMedia downloads media from the platform.
    DownloadMedia(ctx context.Context, url string) ([]byte, string, error)

    // Close closes any open connections.
    Close() error
}
```

### 6.2 MediaHandler Interface

```go
// plugin/chat_apps/media/handler.go

package media

import (
    "context"
)

// MediaHandler processes multimedia messages.
type MediaHandler interface {
    // ProcessAudio converts audio to text using Whisper.
    ProcessAudio(ctx context.Context, data []byte, mimeType string) (string, error)

    // ProcessImage extracts text from images using OCR.
    ProcessImage(ctx context.Context, data []byte) (string, error)
}

// Config for media processing.
type MediaConfig struct {
    WhisperEndpoint string // OpenAI Whisper API endpoint
    WhisperAPIKey   string
    OCREngine       string // "tesseract" or "api"
}
```

---

## 7. Implementation Phases

### Phase 1: Foundation (Week 1)

| Task | Description |
|:-----|:-----------|
| 1.1 | Create `plugin/chat_apps/` directory structure |
| 1.2 | Define `ChatChannel` interface |
| 1.3 | Add database migrations |
| 1.4 | Implement `ChatAppService` store interface |
| 1.5 | Add proto definitions and generate code |

### Phase 2: Telegram Bot (Week 2)

| Task | Description |
|:-----|:-----------|
| 2.1 | Implement Telegram channel using `telegram-bot-api` |
| 2.2 | Add webhook handler |
| 2.3 | Implement message type handling (text, photo, voice, document) |
| 2.4 | Add user binding flow |
| 2.5 | Integration testing with Telegram Bot API |

### Phase 3: WhatsApp Bridge (Week 3)

| Task | Description |
|:-----|:-----------|
| 3.1 | Set up Baileys Node.js bridge service |
| 3.2 | Define communication protocol (gRPC or HTTP) |
| 3.3 | Implement Go client for the bridge |
| 3.4 | Add message handling |
| 3.5 | Integration testing |

### Phase 4: DingTalk Bot (Week 4)

| Task | Description |
|:-----|:-----------|
| 4.1 | Implement DingTalk signature verification |
| 4.2 | Add callback handler |
| 4.3 | Implement message encryption/decryption |
| 4.4 | Add enterprise-specific features |
| 4.5 | Integration testing |

### Phase 5: Media Processing (Week 5)

| Task | Description |
|:-----|:-----------|
| 5.1 | Integrate OpenAI Whisper for voice-to-text |
| 5.2 | Implement OCR (Tesseract or cloud API) |
| 5.3 | Add file attachment archiving |
| 5.4 | Add size limits and validation |

### Phase 6: Frontend & Polish (Week 6)

| Task | Description |
|:-----|:-----------|
| 6.1 | Create Chat Apps settings panel |
| 6.2 | Add credential management UI |
| 6.3 | Add connection status indicators |
| 6.4 | Update i18n translations |
| 6.5 | Documentation updates |

---

## 8. Security Considerations

### 8.1 Token Storage

- Encrypt `access_token` using AES-256-GCM
- Store encryption key in environment variable
- Never return tokens in API responses

### 8.2 Webhook Verification

- **Telegram**: Verify against bot token
- **DingTalk**: Verify HMAC signature
- **WhatsApp**: Verify webhook secret

### 8.3 User Authorization

- Only bind credentials to authenticated users
- Verify user owns the platform account (challenge-response)
- Rate limiting per user per platform

---

## 9. Testing Strategy

### 9.1 Unit Tests

- `ChatChannel` interface mock implementations
- Webhook parsing logic
- Message formatting logic
- Token encryption/decryption

### 9.2 Integration Tests

- Telegram Bot API test endpoint
- DingTalk callback simulator
- Media processing with sample files

### 9.3 E2E Tests

- Full message flow from webhook to AI response
- Credential binding/unbinding flow
- Media upload and processing flow

---

## 10. Monitoring & Observability

### 10.1 Metrics

- Webhook接收计数
- 消息处理延迟 (P50, P95, P99)
- 错误率 (按平台分类)
- Token 使用量

### 10.2 Logging

- 结构化日志使用 `log/slog`
- 记录所有 webhook 请求
- 记录所有 AI 代理调用
- 敏感信息脱敏

### 10.3 Health Checks

- 各平台连接状态
- WhatsApp 桥接服务状态
- 媒体处理服务状态

---

## 11. Configuration

### Environment Variables

```bash
# Chat Apps General
DIVINESENSE_CHAT_APPS_ENABLED=true
DIVINESENSE_CHAT_APPS_SECRET_KEY=<encryption-key>

# Telegram
DIVINESENSE_TELEGRAM_BOT_TOKEN=<bot-token>
DIVINESENSE_TELEGRAM_WEBHOOK_SECRET=<secret>

# WhatsApp Bridge
DIVINESENSE_WHATSAPP_BRIDGE_URL=http://localhost:3000
DIVINESENSE_WHATSAPP_ENABLED=true

# DingTalk
DIVINESENSE_DINGTALK_APP_KEY=<app-key>
DIVINESENSE_DINGTALK_APP_SECRET=<app-secret>
DIVINESENSE_DINGTALK_ENABLED=false

# Media Processing
DIVINESENSE_WHISPER_API_KEY=<openai-key>
DIVINESENSE_WHISPER_ENDPOINT=https://api.openai.com/v1/audio/transcriptions
DIVINESENSE_OCR_ENGINE=tesseract
DIVINESENSE_MAX_MEDIA_SIZE_MB=20
```

---

## 12. Dependencies

### Go Packages

```go
require (
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/tmc/langchaingo v0.0.0 // For Whisper integration
)
```

### Node.js (WhatsApp Bridge)

```json
{
  "@whiskeysockets/baileys": "^6.6.0",
  "@grpc/grpc-js": "^1.9.0",
  "pino": "^8.16.0"
}
```

---

## 13. Acceptance Criteria

- [x] Research phase complete
- [ ] `make check-all` passes
- [ ] Telegram Bot sends/receives messages
- [ ] WhatsApp bridge sends/receives messages
- [ ] DingTalk bot sends/receives messages
- [ ] Image OCR produces correct text
- [ ] Voice-to-text produces correct transcription
- [ ] Files are archived as attachments
- [ ] User authorization works correctly
- [ ] i18n translations complete
- [ ] Documentation updated

---

## 14. References

- [Issue #53](https://github.com/hrygo/divinesense/issues/53)
- [Research Report](../research/chat-apps-integration-research.md)
- [Telegram Bot API](https://core.telegram.org/bots/api)
- [Baileys Documentation](https://github.com/WhiskeySockets/Baileys)
- [DingTalk Robot API](https://open.dingtalk.com/document/dingstart/robot-receive-message)
- [Nanobot Reference](https://github.com/HKUDS/nanobot)

---

**Document Owner**: Loki Mode (Autonomous Agent)
**Last Updated**: 2026-02-03
**Status**: DRAFT → Pending Review
