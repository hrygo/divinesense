# DivineSense Baileys WhatsApp Bridge

> Node.js 服务，用于连接 DivineSense 和 WhatsApp（使用 Baileys 库）

---

## 概述

本服务充当 DivineSense 后端与 WhatsApp 之间的桥接层。由于 WhatsApp Web API 的特殊性，需要使用 Node.js 运行 Baileys 库来维持与 WhatsApp 的连接。

### 架构

```
┌─────────────┐     WebSocket     ┌─────────────────┐     HTTP Webhook     ┌──────────────┐
│  WhatsApp   │ ◄───────────────► │  Baileys Bridge │ ◄───────────────────► │ DivineSense  │
│  (手机/网页) │                   │  (Node.js 服务) │                       │  (Go 后端)   │
└─────────────┘                   └─────────────────┘                       └──────────────┘
                                        │
                                        │ HTTP API
                                        ▼
                                   ┌─────────┐
                                   │  QR 码   │
                                   │ 配对界面 │
                                   └─────────┘
```

---

## 功能特性

| 功能 | 描述 |
|:-----|:-----|
| **消息接收** | 接收 WhatsApp 消息并转发到 DivineSense |
| **消息发送** | 从 DivineSense 发送消息到 WhatsApp |
| **媒体支持** | 支持图片、视频、文档、音频等媒体类型 |
| **自动重连** | 连接断开时自动重新连接 |
| **健康检查** | 提供 HTTP 健康检查端点 |
| **QR 码配对** | 终端和 HTTP 接口提供 QR 码 |

---

## 快速开始

### 前置条件

- Node.js >= 18.0.0
- npm 或 yarn
- WhatsApp 账号（手机应用）

### 安装

```bash
cd plugin/chat_apps/channels/whatsapp/bridge
npm install
```

### 配置

创建 `.env` 文件：

```bash
# HTTP 服务端口（默认：3001）
PORT=3001

# DivineSense Webhook URL
# 开发环境：http://localhost:28081/api/v1/chat_apps/webhook
# 生产环境：https://your-domain.com/api/v1/chat_apps/webhook
DIVINESENSE_WEBHOOK_URL=http://localhost:28081/api/v1/chat_apps/webhook

# Baileys 认证文件路径（相对或绝对）
BAILEYS_AUTH_FILE=./baileys_auth_info.json
```

### 启动

```bash
npm start
```

首次启动会显示 QR 码：

```
==================================================
  QR Code - Scan with WhatsApp
  Settings → Linked Devices → Link a Device
==================================================

████████████████████████████████
████████████████████████████████
████ ▄▄▄▄▄ ▄▄ ▄▄▄▄▄▄▄ ▄▄▄▄▄▄▄ ████
████ █   █ █ █       █       █ ████
████ █▄▄▄█ █ █ █▄▄▄▄▄▄▄█ █▄▄▄▄▄▄▄█ ████
████     █ █       █       █ ████
████ ▄▄▄▄▄ █ █▄▄▄▄▄▄▄█ █▄▄▄▄▄▄▄█ ████
████                               ████
████████████████████████████████

Or visit: http://localhost:3001/info
```

### 扫码配对

1. 打开 WhatsApp 手机应用
2. 进入 **设置 → 已连接的设备 → 链接设备**
3. 扫描终端显示的 QR 码
4. 扫描成功后显示：`✅ WhatsApp connection opened successfully!`

---

## API 端点

### GET /health

健康检查端点。

**响应**：
```json
{
  "status": "ok",
  "connected": true,
  "timestamp": "2026-02-03T13:00:00.000Z"
}
```

### GET /info

获取连接信息，包括 QR 码（未连接时）和手机号（已连接时）。

**响应**（未连接）：
```json
{
  "connected": false,
  "qrcode": "4@otpCode...",
  "phone": null
}
```

**响应**（已连接）：
```json
{
  "connected": true,
  "qrcode": null,
  "phone": "8613800138000"
}
```

### POST /webhook

接收来自 Baileys 的内部 webhook（用于消息接收）。

**请求体**：
```json
{
  "key": {
    "remoteJid": "8613800138000@s.whatsapp.net",
    "fromMe": false,
    "id": "3EB0XXXX"
  },
  "message": {
    "conversation": "你好"
  },
  "messageType": "conversation"
}
```

### POST /send

发送消息到 WhatsApp。

**请求体**：
```json
{
  "jid": "8613800138000@s.whatsapp.net",
  "type": "conversation",
  "content": "Hello from DivineSense!"
}
```

**支持的 type**：
- `conversation` - 文本消息
- `imageMessage` - 图片消息
- `documentMessage` - 文档消息
- `videoMessage` - 视频消息
- `audioMessage` - 音频消息

**媒体消息示例**：
```json
{
  "jid": "8613800138000@s.whatsapp.net",
  "type": "imageMessage",
  "media": "https://example.com/image.jpg",
  "mimetype": "image/jpeg",
  "caption": "图片说明"
}
```

### GET /download?url=...

下载 WhatsApp 媒体文件。

**参数**：
- `url` - 媒体 URL（从消息中获取）

---

## 生产部署

### 使用 PM2

```bash
# 安装 PM2
npm install -g pm2

# 启动服务
pm2 start src/index.js --name baileys-bridge

# 保存进程列表
pm2 save

# 设置开机自启
pm2 startup
```

**配置文件** (`ecosystem.config.js`)：
```javascript
module.exports = {
  apps: [{
    name: 'baileys-bridge',
    script: './src/index.js',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '500M',
    env: {
      NODE_ENV: 'production',
      PORT: 3001,
      DIVINESENSE_WEBHOOK_URL: 'https://your-domain.com/api/v1/chat_apps/webhook'
    },
    error_file: '/var/log/baileys-bridge/error.log',
    out_file: '/var/log/baileys-bridge/out.log',
    log_date_format: 'YYYY-MM-DD HH:mm:ss Z'
  }]
};
```

### 使用 Systemd

创建服务文件 `/etc/systemd/system/baileys-bridge.service`：

```ini
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
sudo systemctl status baileys-bridge
```

### 使用 Docker

```dockerfile
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --only=production

COPY . .

EXPOSE 3001

CMD ["node", "src/index.js"]
```

构建并运行：
```bash
docker build -t baileys-bridge .
docker run -d \
  --name baileys-bridge \
  -p 3001:3001 \
  -e DIVINESENSE_WEBHOOK_URL=https://your-domain.com/api/v1/chat_apps/webhook \
  -v $(pwd)/baileys_auth_info.json:/app/baileys_auth_info.json \
  baileys-bridge
```

---

## 依赖

| 依赖 | 版本 | 说明 |
|:-----|:-----|:-----|
| `@whiskeysockets/baileys` | ^6.5.0 | WhatsApp Web API 库 |
| `express` | ^4.18.2 | HTTP 服务器 |
| `cors` | ^2.8.5 | CORS 中间件 |
| `dotenv` | ^16.3.1 | 环境变量加载 |
| `qrcode-terminal` | ^0.12.0 | QR 码终端显示 |

---

## 故障排查

### QR 码过期

**症状**：QR 码扫描后无响应或提示过期

**解决方案**：
```bash
# 删除认证文件
rm baileys_auth_info.json

# 重启服务
pm2 restart baileys-bridge
# 或
npm start
```

### 连接断开

**症状**：频繁显示 "Connection closed"

**解决方案**：
1. 检查网络连接
2. 检查 DivineSense Webhook URL 是否正确
3. 查看日志：
   ```bash
   pm2 logs baileys-bridge
   ```

### 消息发送失败

**症状**：DivineSense 发送消息但 WhatsApp 未收到

**检查项**：
1. JID 格式是否正确：`手机号@s.whatsapp.net`
2. 桥接服务是否正常运行：访问 `/health`
3. 查看 DivineSense 后端日志

---

## 开发

### 调试模式

```bash
# 启用详细日志
DEBUG=* npm start

# 或使用 Node.js 监视模式
npm run dev
```

### 测试

```bash
# 运行测试
npm test
```

### 项目结构

```
bridge/
├── src/
│   └── index.js          # 主入口
├── test/
│   └── test.js           # 测试文件
├── package.json
├── .env.example          # 环境变量模板
├── .env                  # 实际配置（不提交）
├── baileys_auth_info.json # 认证数据（自动生成）
└── README.md
```

---

## 安全建议

1. **认证文件保护**：`baileys_auth_info.json` 包含敏感会话信息，应设置适当权限
   ```bash
   chmod 600 baileys_auth_info.json
   ```

2. **环境变量**：不要将 `.env` 文件提交到版本控制

3. **HTTPS**：生产环境中使用 HTTPS 保护 DivineSense Webhook

4. **访问控制**：限制对 Bridge API 的访问（如使用防火墙或反向代理）

---

## 许可证

MIT

---

**最后更新**: 2026-02-03
**相关文档**: [Chat Apps 用户指南](../../../../docs/user-guides/CHAT_APPS.md) | [架构文档](../../../../docs/dev-guides/ARCHITECTURE.md)
