# DivineSense Baileys WhatsApp Bridge

This Node.js service acts as a bridge between DivineSense and WhatsApp Business API using the [Baileys](https://github.com/adiwajshing/baileys) library.

## Features

- Receive WhatsApp messages via webhook and forward to DivineSense
- Send messages from DivineSense to WhatsApp users
- Media upload/download support
- Health check endpoint
- Automatic reconnection

## Prerequisites

- Node.js >= 18.0.0
- A Meta for Developers account with WhatsApp Business API app
- WhatsApp Business phone number

## Installation

```bash
cd plugin/chat_apps/channels/whatsapp/bridge
npm install
```

## Configuration

Create a `.env` file in the bridge directory:

```bash
# HTTP server port (default: 3001)
PORT=3001

# DivineSense webhook URL
DIVINESENSE_WEBHOOK_URL=https://your-domain.com/api/v1/chat-apps/webhook/whatsapp

# Path to Baileys auth file (relative to bridge directory)
BAILEYS_AUTH_FILE=./baileys_auth_info.json
```

## Usage

Start the bridge:

```bash
npm start
```

The bridge will:
1. Start an HTTP server on the configured port
2. Connect to WhatsApp using Baileys
3. Display pairing QR code in console on first run
4. Forward received messages to DivineSense webhook
5. Listen for send requests from DivineSense

## API Endpoints

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "connected": true,
  "timestamp": "2026-02-03T13:00:00.000Z"
}
```

### GET /info
Get connection info including QR code for pairing.

**Response:**
```json
{
  "connected": false,
  "qrcode": "XXXX",
  "phone": null
}
```

### POST /webhook
Receive webhooks from Baileys (internal use).

### POST /send
Send a message to WhatsApp.

**Request:**
```json
{
  "jid": "1234567890@s.whatsapp.net",
  "type": "conversation",
  "content": "Hello from DivineSense!"
}
```

### GET /download?url=...
Download media from WhatsApp.

## Pairing with WhatsApp

1. Start the bridge service: `npm start`
2. GET `/info` to get QR code (or check console)
3. Open WhatsApp on your phone
4. Settings → Linked Devices → Link a Device
5. Scan the QR code displayed
6. Connection will be established automatically

## Production Deployment

For production, consider:
- Using process manager (PM2, systemd)
- Configuring HTTPS reverse proxy
- Setting up monitoring and logging
- Configuring auto-restart

### PM2 Example

```bash
npm install -g pm2
pm2 start src/index.js --name baileys-bridge
pm2 save
pm2 startup
```

### Systemd Service

```ini
[Unit]
Description=DivineSense Baileys Bridge
After=network.target

[Service]
Type=simple
User=divinesense
WorkingDirectory=/opt/baileys-bridge
ExecStart=/usr/bin/node src/index.js
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Integration with DivineSense

The Go code in `bridge.go` communicates with this bridge service:
- Health check on startup
- Send messages via POST /send
- Forward webhooks via POST /webhook

The bridge URL must be configured when creating the WhatsApp channel in DivineSense.
