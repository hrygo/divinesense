# Chat Apps 集成开发者指南

> DivineSense 聊天应用集成模块的技术文档和开发指南

---

## 目录

1. [架构概述](#架构概述)
2. [数据模型](#数据模型)
3. [API 端点](#api-端点)
4. [平台集成](#平台集成)
5. [安全机制](#安全机制)
6. [开发调试](#开发调试)
7. [部署指南](#部署指南)

---

## 架构概述

### 系统架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Frontend (React)                                 │
│                   Settings → Chat Apps Configuration                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                         │
                                         │ REST/Connect RPC
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           DivineSense Backend (Go)                            │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐ │
│  │  ChatAppService │    │  ChatRouter     │    │   Parrot Agents         │ │
│  │  (凭证管理)      │ -> │  (消息路由)      │ -> │   (AI 处理)             │ │
│  └─────────────────┘    └─────────────────┘    └─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
                    ▼                    ▼                    ▼
         ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
         │   Telegram      │  │     WhatsApp    │  │     钉钉         │
         │   Bot API       │  │   Baileys       │  │   Open API      │
         │                 │  │   Bridge        │  │                 │
         │ (webhook 推送)   │  │   (HTTP 轮询)    │  │  (webhook 推送)  │
         └─────────────────┘  └─────────────────┘  └─────────────────┘
```

### 核心组件

| 组件 | 路径 | 职责 |
|:-----|:-----|:-----|
| **ChatAppService** | `server/service/chat_app/` | 凭证管理、消息发送 |
| **Chat Handler** | `server/router/api/v1/chat_apps/` | Webhook 处理、请求路由 |
| **Channel 层** | `plugin/chat_apps/channels/` | 平台适配器 |
| **Baileys Bridge** | `plugin/chat_apps/channels/whatsapp/bridge/` | WhatsApp Node.js 桥接服务 |

### 消息流程

```
┌─────────────┐     ①发送消息      ┌─────────────┐
│   用户      │ ──────────────────► │  聊天平台    │
└─────────────┘                    └─────────────┘
                                            │
                                            │ ② Webhook 推送
                                            ▼
                                    ┌─────────────┐
                                    │ DivineSense │
                                    │   Webhook   │
                                    └─────────────┘
                                            │
                         ┌──────────────────┼──────────────────┐
                         │                  │                  │
                         ▼                  ▼                  ▼
                  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
                  │ 验证凭证     │   │ 解析消息     │   │ 路由到 AI    │
                  └─────────────┘   └─────────────┘   └─────────────┘
                                                         │
                                                         ▼
                                                  ┌─────────────┐
                                                  │ Parrot      │
                                                  │ 处理请求     │
                                                  └─────────────┘
                                                         │
                                                         ▼
                                                  ┌─────────────┐
                                                  │ ③发送回复   │
                                                  └─────────────┘
```

---

## 数据模型

### Platform 枚举

```go
type Platform string

const (
    PlatformTelegram  Platform = "PLATFORM_TELEGRAM"
    PlatformWhatsApp  Platform = "PLATFORM_WHATSAPP"
    PlatformDingtalk  Platform = "PLATFORM_DINGTALK"
)
```

### ChatAppCredential

```go
type ChatAppCredential struct {
    ID               int64     `json:"id"`
    CreatorID        int64     `json:"creator_id"`        // 所属用户
    Platform         Platform  `json:"platform"`          // 平台类型
    PlatformUserID   string    `json:"platform_user_id"`  // 平台用户 ID
    AccessToken      string    `json:"-"`                 // 访问令牌（加密存储）
    AppSecret        string    `json:"-"`                 // 应用密钥（加密存储）
    BridgeURL        string    `json:"bridge_url"`        // 桥接服务 URL（WhatsApp）
    Enabled          bool      `json:"enabled"`           // 是否启用
    CreatedTs        int64     `json:"created_ts"`
    UpdatedTs        int64     `json:"updated_ts"`
}
```

### Webhook 请求

```go
type ChatAppWebhookRequest struct {
    Platform Platform           `json:"platform"`           // 平台类型
    Headers  map[string]string  `json:"headers"`            // 请求头（签名验证）
    Payload  []byte             `json:"payload"`            // 消息内容
}
```

### 发送消息请求

```go
type ChatAppSendMessageRequest struct {
    Platform   Platform `json:"platform"`             // 目标平台
    ToUserID   string   `json:"to_user_id"`           // 接收者 ID
    Content    string   `json:"content"`              // 消息内容
    ParseMode  string   `json:"parse_mode,omitempty"` // 格式化（Markdown/HTML）
}
```

---

## API 端点

### 1. 注册凭证

```http
POST /api/v1/chat_apps/credentials
Authorization: Bearer <token>
Content-Type: application/json

{
  "platform": "PLATFORM_TELEGRAM",
  "platform_user_id": "123456789",
  "access_token": "bot_token_here",
  "app_secret": "app_secret_here"  // 可选，钉钉需要
}
```

**响应**：
```json
{
  "id": 1,
  "platform": "PLATFORM_TELEGRAM",
  "platform_user_id": "123456789",
  "enabled": true,
  "created_ts": 1736809200000
}
```

### 2. 列出凭证

```http
GET /api/v1/chat_apps/credentials
Authorization: Bearer <token>
```

**响应**：
```json
{
  "credentials": [
    {
      "id": 1,
      "platform": "PLATFORM_TELEGRAM",
      "platform_user_id": "123456789",
      "enabled": true,
      "bridge_url": ""
    },
    {
      "id": 2,
      "platform": "PLATFORM_WHATSAPP",
      "platform_user_id": "",
      "enabled": true,
      "bridge_url": "http://localhost:3001"
    }
  ]
}
```

### 3. 删除凭证

```http
DELETE /api/v1/chat_apps/credentials/{id}
Authorization: Bearer <token>
```

### 4. Webhook 接收

```http
POST /api/v1/chat_apps/webhook?platform={platform}
Content-Type: application/json

// Telegram
{
  "update_id": 123456789,
  "message": {
    "message_id": 1,
    "from": { "id": 123456789, "first_name": "User" },
    "chat": { "id": 123456789, "type": "private" },
    "text": "你好"
  }
}

// 钉钉
{
  "msgtype": "text",
  "text": { "content": "你好" },
  "chatbotUserId": "xxx",
  "timestamp": 1736809200000
}
```

### 5. 发送消息

```http
POST /api/v1/chat_apps/send
Authorization: Bearer <token>
Content-Type: application/json

{
  "platform": "PLATFORM_TELEGRAM",
  "to_user_id": "123456789",
  "content": "你好，这是 DivineSense AI 的回复",
  "parse_mode": "Markdown"
}
```

### 6. 切换启用状态

```http
PUT /api/v1/chat_apps/credentials/{id}/toggle
Authorization: Bearer <token>
```

---

## 平台集成

### Telegram

#### Webhook 验证

- Bot Token 直接验证
- 从请求中提取 `message.chat.id` 作为平台用户 ID

#### 发送消息

使用 Telegram Bot API：
```go
func sendTelegramMessage(token, chatID, content, parseMode string) error {
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
    payload := map[string]interface{}{
        "chat_id":    chatID,
        "text":       content,
        "parse_mode": parseMode,
    }
    // ... 发送 HTTP 请求
}
```

#### 消息类型支持

| 类型 | 支持 | 说明 |
|:-----|:-----|:-----|
| 文本消息 | ✅ | 支持 Markdown/HTML 格式 |
| 图片 | ✅ | 通过 URL 发送 |
| 文档 | ✅ | 支持文件上传 |

### WhatsApp (Baileys)

#### 架构

由于 WhatsApp Web API 的特殊性，使用 Node.js 桥接服务：

```
┌──────────────┐      HTTP       ┌─────────────────┐
│  WhatsApp    │ ◄─────────────► │  Baileys Bridge │
│  (手机/网页)  │                 │  (Node.js)      │
└──────────────┘                 └─────────────────┘
                                            │
                                            │ Webhook
                                            ▼
                                    ┌─────────────────┐
                                    │  DivineSense    │
                                    │  Backend        │
                                    └─────────────────┘
```

#### 桥接服务 API

| 端点 | 方法 | 描述 |
|:-----|:-----|:-----|
| `/health` | GET | 健康检查 |
| `/info` | GET | 获取连接状态和 QR 码 |
| `/webhook` | POST | 接收 Baileys 消息 |
| `/send` | POST | 发送消息到 WhatsApp |
| `/download` | GET | 下载媒体文件 |

#### JID 格式

WhatsApp 使用 JID（Jabber ID）标识用户：
- 个人用户: `1234567890@s.whatsapp.net`
- 群组: `1234567890@g.us`
- 广播列表: `1234567890@broadcast`

#### 消息类型

```javascript
// 文本消息
{
  jid: "1234567890@s.whatsapp.net",
  type: "conversation",
  content: "Hello"
}

// 图片消息
{
  jid: "1234567890@s.whatsapp.net",
  type: "imageMessage",
  media: "https://...",
  mimetype: "image/jpeg",
  caption: "图片说明"
}
```

### 钉钉

#### 签名验证

使用 HMAC-SHA256 验证请求来源：

```go
func verifyDingtalkSignature(timestamp, secret, signature string) bool {
    stringToSign := timestamp + "\n" + secret
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(stringToSign))
    computed := base64.StdEncoding.EncodeToString(h.Sum(nil))
    return computed == signature
}
```

#### 消息格式

钉钉使用特定的消息格式：
```json
{
  "msgtype": "text",
  "text": {
    "content": "消息内容"
  }
}
```

#### 支持的消息类型

| 类型 | msgtype | 说明 |
|:-----|:---------|:-----|
| 文本 | text | 纯文本消息 |
| Markdown | markdown | 支持 Markdown 格式 |
| 链接 | link | 链接卡片 |
| ActionCard | actionCard | 独立跳转卡片 |

---

## 安全机制

### Token 加密存储

所有敏感凭证使用 AES-256-GCM 加密：

```go
const (
    keySize   = 32  // AES-256
    nonceSize = 12  // GCM 标准大小
)

func encryptToken(plaintext, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, nonceSize)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### 启动时验证

服务启动时会验证加密密钥配置：

```go
func validateChatAppsConfig() error {
    secretKey := os.Getenv("DIVINESENSE_CHAT_APPS_SECRET_KEY")
    if secretKey == "" {
        return fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be set")
    }
    if len(secretKey) != 32 {
        return fmt.Errorf("DIVINESENSE_CHAT_APPS_SECRET_KEY must be exactly 32 bytes")
    }
    return nil
}
```

### 输入验证

所有用户输入在存储前都会验证：

| 字段 | 最大长度 | 验证规则 |
|:-----|:---------|:---------|
| `platform_user_id` | 255 字符 | 平台白名单验证 |
| `access_token` | 2048 字符 | 长度限制 |
| `app_secret` | 2048 字符 | 长度限制（可选） |
| `webhook_url` | 2048 字符 | URL 格式验证（可选） |

```go
func validateCreateCredentialRequest(req *CreateCredentialRequest) error {
    // 验证平台是否支持
    if !req.Platform.IsValid() {
        return fmt.Errorf("invalid platform: %s", req.Platform)
    }
    // ... 长度验证
}
```

### 环境变量

```bash
# 必需：32字节加密密钥
DIVINESENSE_CHAT_APPS_SECRET_KEY=<your-32-byte-key>

# 实例 URL（用于生成 Webhook URL）
DIVINESENSE_INSTANCE_URL=https://your-domain.com
```

### 生成加密密钥

```bash
# Linux/Mac
openssl rand -base64 32 | head -c 32

# Python
python3 -c "import secrets; print(secrets.token_urlsafe(32)[:32])"
```

### Webhook 安全

| 平台 | 验证方式 | 防护措施 |
|:-----|:---------|:---------|
| Telegram | Bot Token（查询数据库验证） | Token 匹配 + 用户绑定 |
| WhatsApp | 桥接服务内部验证 | 连接状态检查 + 请求源验证 |
| 钉钉 | HMAC-SHA256 签名 + 时间戳 | 签名验证 + 重放攻击防护 |

**钉钉重放攻击防护**：

```go
// 验证时间戳（5分钟窗口）
ts, err := strconv.ParseInt(timestamp, 10, 64)
timeDiff := now - ts/1000
if timeDiff < 0 {
    timeDiff = -timeDiff
}
if timeDiff > int64(MaxTimestampSkew.Seconds()) {
    return channels.ErrInvalidSignature
}

// 常量时间比较防止时序攻击
if !hmac.Equal([]byte(sign), []byte(expectedSign)) {
    return channels.ErrInvalidSignature
}
```

**并发安全**：

Token 缓存使用单一 Mutex 代替 RWMutex，避免竞态条件：

```go
type DingTalkChannel struct {
    tokenMu sync.Mutex  // 使用单一 Mutex 防止竞态
    // ...
}
```

---

## 开发调试

### 本地开发环境

1. **启动 DivineSense 后端**：
   ```bash
   make run
   ```

2. **启动 WhatsApp Bridge**（需要时）：
   ```bash
   cd plugin/chat_apps/channels/whatsapp/bridge
   npm install
   npm start
   ```

3. **配置内网穿透**（用于 Webhook 测试）：
   ```bash
   # 使用 ngrok
   ngrok http 28081

   # 或使用 cloudflare tunnel
   cloudflared tunnel --url http://localhost:28081
   ```

### 调试日志

启用详细日志：

```bash
# Go 后端
DIVINESENSE_LOG_LEVEL=debug make run

# WhatsApp Bridge
DEBUG=* npm start
```

### 测试 Webhook

使用 curl 测试：

```bash
# Telegram Webhook
curl -X POST "http://localhost:28081/api/v1/chat_apps/webhook?platform=telegram" \
  -H "Content-Type: application/json" \
  -d '{
    "update_id": 123456789,
    "message": {
      "message_id": 1,
      "from": {"id": 123456789, "first_name": "Test"},
      "chat": {"id": 123456789, "type": "private"},
      "text": "测试消息"
    }
  }'

# 钉钉 Webhook
curl -X POST "http://localhost:28081/api/v1/chat_apps/webhook?platform=dingtalk" \
  -H "Content-Type: application/json" \
  -d '{
    "msgtype": "text",
    "text": {"content": "测试消息"},
    "chatbotUserId": "test",
    "timestamp": 1736809200000
  }'
```

### 单元测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./plugin/chat_apps/... -v

# 运行特定测试
go test ./plugin/chat_apps/channels/telegram/ -run TestSendMessage
```

### 集成测试

```go
// plugin/chat_apps/integration_test.go
func TestChatAppIntegration(t *testing.T) {
    // 1. 创建测试凭证
    cred := &ChatAppCredential{
        Platform:       PlatformTelegram,
        PlatformUserID: "test_user",
        AccessToken:    "test_token",
    }

    // 2. 测试消息发送
    err := sendMessage(context.Background(), cred, "test message")
    assert.NoError(t, err)

    // 3. 测试 Webhook 处理
    // ...
}
```

---

## 部署指南

### 生产环境检查清单

- [ ] 设置 `DIVINESENSE_CHAT_APPS_SECRET_KEY`（32 字节）
- [ ] 配置 `DIVINESENSE_INSTANCE_URL`（生成 Webhook URL）
- [ ] 启用 HTTPS（Let's Encrypt）
- [ ] 配置防火墙规则
- [ ] 部署 WhatsApp Bridge（如果使用）

### Docker 部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  divinesense:
    image: hrygo/divinesense:latest
    environment:
      - DIVINESENSE_CHAT_APPS_SECRET_KEY=${SECRET_KEY}
      - DIVINESENSE_INSTANCE_URL=https://your-domain.com
    ports:
      - "5230:5230"
    depends_on:
      - postgres

  baileys-bridge:
    build: ./plugin/chat_apps/channels/whatsapp/bridge
    environment:
      - PORT=3001
      - DIVINESENSE_WEBHOOK_URL=http://divinesense:5230/api/v1/chat_apps/webhook
    ports:
      - "3001:3001"
```

### PM2 部署（WhatsApp Bridge）

```javascript
// ecosystem.config.js
module.exports = {
  apps: [{
    name: 'baileys-bridge',
    script: './src/index.js',
    cwd: '/opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge',
    env: {
      PORT: 3001,
      DIVINESENSE_WEBHOOK_URL: 'https://your-domain.com/api/v1/chat_apps/webhook',
      NODE_ENV: 'production'
    },
    error_file: '/var/log/baileys-bridge/error.log',
    out_file: '/var/log/baileys-bridge/out.log',
    log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M'
  }]
};
```

启动：
```bash
pm2 start ecosystem.config.js
pm2 save
pm2 startup
```

### Systemd 服务

```ini
# /etc/systemd/system/baileys-bridge.service
[Unit]
Description=DivineSense Baileys WhatsApp Bridge
After=network.target

[Service]
Type=simple
User=divinesense
WorkingDirectory=/opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge
ExecStart=/usr/bin/node src/index.js
Restart=always
RestartSec=10
Environment=PORT=3001
Environment=DIVINESENSE_WEBHOOK_URL=https://your-domain.com/api/v1/chat_apps/webhook

[Install]
WantedBy=multi-user.target
```

启用并启动：
```bash
sudo systemctl daemon-reload
sudo systemctl enable baileys-bridge
sudo systemctl start baileys-bridge
```

---

## 扩展新平台

### 添加新平台步骤

1. **定义平台枚举**：
   ```go
   // plugin/chat_apps/types.go
   const (
       // ...
       PlatformDiscord Platform = "PLATFORM_DISCORD"
   )
   ```

2. **实现 Channel 接口**：
   ```go
   // plugin/chat_apps/channels/discord/discord.go
   type DiscordChannel struct {
       client *http.Client
   }

   func (c *DiscordChannel) SendMessage(ctx context.Context, req *SendMessageRequest) error {
       // 实现 Discord 消息发送
   }

   func (c *DiscordChannel) ParseWebhook(r *http.Request) (*WebhookMessage, error) {
       // 实现 Discord Webhook 解析
   }
   ```

3. **注册 Channel**：
   ```go
   // plugin/chat_apps/factory.go
   func NewChannel(platform Platform) (Channel, error) {
       switch platform {
       case PlatformTelegram:
           return NewTelegramChannel(), nil
       case PlatformDiscord:
           return NewDiscordChannel(), nil
       // ...
       }
   }
   ```

4. **添加测试**：
   ```go
   // plugin/chat_apps/channels/discord/discord_test.go
   func TestDiscordSendMessage(t *testing.T) {
       // ...
   }
   ```

---

**最后更新**: 2026-02-03
**相关文档**: [用户指南](../user-guides/CHAT_APPS.md) | [系统架构](../dev-guides/ARCHITECTURE.md) | [后端开发](../dev-guides/BACKEND_DB.md)
