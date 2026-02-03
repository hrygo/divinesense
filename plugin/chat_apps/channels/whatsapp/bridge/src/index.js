/**
 * DivineSense Baileys WhatsApp Bridge Service
 *
 * This service acts as a bridge between DivineSense and WhatsApp Business API
 * using the Baileys library (a TypeScript/JavaScript WhatsApp Web API library).
 *
 * Features:
 * - Receive webhooks from WhatsApp and forward to DivineSense
 * - Send messages from DivineSense to WhatsApp
 * - Media upload/download support
 * - Health check endpoint
 *
 * Environment Variables:
 * - PORT: HTTP server port (default: 3001)
 * - DIVINESENSE_WEBHOOK_URL: DivineSense webhook URL
 * - BAILEYS_AUTH_FILE: Path to Baileys auth file (default: ./baileys_auth_info.json)
 * - BRIDGE_API_KEY: Optional API key for endpoint authentication (recommended for production)
 * - ALLOWED_ORIGINS: Comma-separated list of allowed CORS origins (default: *)
 */

import express from "express";
import { makeWASocket, DisconnectReason, useMultiFileAuthState, fetchLatestBaileysVersion } from "@whiskeysockets/baileys";
import cors from "cors";
import { fileURLToPath } from "url";
import dotenv from "dotenv";
import path from "path";
import { readFile, writeFile } from "fs/promises";
import qrcodeTerminal from "qrcode-terminal";

// Load environment variables
dotenv.config();

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PORT = process.env.PORT || 3001;
const DIVINESENSE_WEBHOOK_URL = process.env.DIVINESENSE_WEBHOOK_URL || "";
const BAILEYS_AUTH_FILE = process.env.BAILEYS_AUTH_FILE || path.join(__dirname, "baileys_auth_info.json");
const BRIDGE_API_KEY = process.env.BRIDGE_API_KEY || "";
const ALLOWED_ORIGINS = process.env.ALLOWED_ORIGINS || "*";

// Global state
let srv = null;
let qrCodeString = null;
let isConnected = false;
const divineSenseSessionId = null;

// Express app setup
const app = express();

// Configure CORS - restrict to specific origins if provided
const corsOptions = {
  origin: ALLOWED_ORIGINS === "*" ? "*" : ALLOWED_ORIGINS.split(","),
  credentials: true,
};
app.use(cors(corsOptions));

app.use(express.json());

// Optional API key middleware for sensitive endpoints
const requireApiKey = (req, res, next) => {
  if (!BRIDGE_API_KEY) {
    // No API key configured, skip authentication
    return next();
  }

  const apiKey = req.headers["x-bridge-api-key"];
  if (apiKey !== BRIDGE_API_KEY) {
    return res.status(401).json({ success: false, message: "Unauthorized: Invalid API key" });
  }
  next();
};

/**
 * Health check endpoint
 */
app.get("/health", (req, res) => {
  res.json({
    status: "ok",
    connected: isConnected,
    timestamp: new Date().toISOString(),
  });
});

/**
 * Receive webhook from Baileys and forward to DivineSense
 */
app.post("/webhook", async (req, res) => {
  try {
    const { body } = req;

    // Forward to DivineSense webhook
    if (DIVINESENSE_WEBHOOK_URL) {
      const response = await fetch(DIVINESENSE_WEBHOOK_URL, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          platform: "whatsapp",
          headers: {
            "X-Baileys-Signature": req.headers["x-baileys-signature"] || "",
          "Query-String": "",
          "X-Baileys-Timestamp": req.headers["x-baileys-timestamp"] || "",
          "X-Hub-Signature": req.headers["x-hub-signature"] || "",
          "X-Baileys-Instance-Id": req.headers["x-baileys-instance-id"] || "",
          "X-Baileys-Dispatch-Id": req.headers["x-baileys-dispatch-id"] || "",
          "X-Baileys-Dispatch-Device-Id": req.headers["x-baileys-dispatch-device-id"] || "",
        },
          payload: Buffer.from(JSON.stringify(body)),
        }),
      });

      if (response.ok) {
        res.json({ success: true, message: "Message received" });
      } else {
        res.status(500).json({ success: false, message: "Failed to forward message" });
      }
    } else {
      console.warn("DIVINESENSE_WEBHOOK_URL not configured, logging only");
      console.log("WhatsApp message received:", JSON.stringify(body, null, 2));
      res.json({ success: true, message: "Message logged (webhook not configured)" });
    }
  } catch (error) {
    console.error("Error processing webhook:", error);
    res.status(500).json({ success: false, message: "Internal error" });
  }
});

/**
 * Send message endpoint (called by DivineSense)
 * Protected by optional API key authentication
 */
app.post("/send", requireApiKey, async (req, res) => {
  try {
    const { jid, type, content, media, mime_type, file_name } = req.body;

    if (!jid || !type) {
      return res.status(400).json({ success: false, message: "Missing required fields" });
    }

    if (!srv) {
      return res.status(503).json({ success: false, message: "WhatsApp not connected" });
    }

    const message = { jid };

    // Text message
    if (type === "conversation" && content) {
      message.text = content;
    }

    // Media message
    if (media && mime_type) {
      message[type === "documentMessage" ? "document" : type] = {
        media: media,
        mimetype: mime_type,
        filename: file_name || "file",
      };
    }

    await srv.sendMessage(jid, message);

    res.json({ success: true, message: "Message sent" });
  } catch (error) {
    console.error("Error sending message:", error);
    res.status(500).json({ success: false, message: "Failed to send message" });
  }
});

/**
 * Download media endpoint
 * Protected by optional API key authentication
 */
app.get("/download", requireApiKey, async (req, res) => {
  try {
    const { url } = req.query;

    if (!url) {
      return res.status(400).json({ success: false, message: "URL required" });
    }

    if (!srv) {
      return res.status(503).json({ success: false, message: "WhatsApp not connected" });
    }

    const result = await srv.downloadMedia(url, {});

    // Set appropriate headers
    res.setHeader("Content-Type", result.mimetype || "application/octet-stream");
    res.setHeader("Content-Disposition", `attachment; filename="${result.filename || "file"}"`);

    res.send(Buffer.from(result.buffer));
  } catch (error) {
    console.error("Error downloading media:", error);
    res.status(500).json({ success: false, message: "Failed to download media" });
  }
});

/**
 * Get connection info (QR code for pairing)
 */
app.get("/info", (req, res) => {
  res.json({
    connected: isConnected,
    qrcode: qrCodeString,
    phone: srv?.user?.id || null,
  });
});

/**
 * Start the WhatsApp connection
 */
async function connectToWhatsApp() {
  const { version } = await fetchLatestBaileysVersion();
  console.log(`Using Baileys version ${version}`);

  const { state, saveCreds } = await useMultiFileAuthState(BAILEYS_AUTH_FILE);
  srv = makeWASocket({
    version,
    defaultQueryTimeoutMs: 60000,
    printQRInTerminal: false,
    auth: state,
  });

  // Store authentication state
  srv.ev.on("creds.update", saveCreds);

  // Connection events - handle QR code and connection status
  srv.ev.on("connection.update", (data) => {
    const { connection, lastDisconnect, qr } = data;

    // QR code received
    if (qr) {
      qrCodeString = qr;
      console.log("\n" + "=".repeat(50));
      console.log("  QR Code - Scan with WhatsApp");
      console.log("  Settings → Linked Devices → Link a Device");
      console.log("=".repeat(50) + "\n");

      // Display QR code in terminal
      qrcodeTerminal.generate(qr, { small: true });

      console.log("\nOr visit: http://localhost:3001/info\n");
    }

    // Connection opened
    if (connection === "open") {
      isConnected = true;
      qrCodeString = null;
      console.log("\n" + "=".repeat(50));
      console.log("  ✅ WhatsApp connection opened successfully!");
      console.log("=".repeat(50) + "\n");
    }

    // Connection closed
    if (connection === "close") {
      isConnected = false;
      const shouldReconnect = lastDisconnect?.error?.output?.statusCode !== DisconnectReason.loggedOut;
      console.log("Connection closed. Reconnect:", shouldReconnect);

      if (shouldReconnect) {
        setTimeout(() => connectToWhatsApp(), 5000);
      }
    }
  });

  // Message handler
  srv.ev.on("messages.upsert", async ({ messages, type }) => {
    if (type !== "notify") return;

    for (const msg of messages) {
      if (!msg.message) continue;

      // Forward to DivineSense webhook
      if (DIVINESENSE_WEBHOOK_URL) {
        await forwardMessageToDivineSense(msg);
      }
    }
  });

  // Baileys automatically connects when socket is created
}

/**
 * Forward a WhatsApp message to DivineSense
 */
async function forwardMessageToDivineSense(msg) {
  try {
    const message = msg.message;
    const messageType = Object.keys(message)[0]; // conversation, imageMessage, etc.

    const webhookPayload = {
      key: {
        remoteJid: msg.key.remoteJid,
        fromMe: msg.key.fromMe,
        id: msg.key.id,
      },
      message: {
        conversation: message.conversation || "",
      },
      messageType: messageType,
    };

    // Add media info if present
    if (message.imageMessage) {
      webhookPayload.message.imageMessage = {
        mediaId: message.imageMessage?.url,
        mimetype: message.imageMessage?.mimetype,
      };
    } else if (message.documentMessage) {
      webhookPayload.message.documentMessage = {
        mediaId: message.documentMessage?.url,
        mimetype: message.documentMessage?.mimetype,
        fileName: message.documentMessage?.fileName,
      };
    } else if (message.videoMessage) {
      webhookPayload.message.videoMessage = {
        mediaId: message.videoMessage?.url,
        mimetype: message.videoMessage?.mimetype,
      };
    } else if (message.audioMessage) {
      webhookPayload.message.audioMessage = {
        mediaId: message.audioMessage?.url,
        mimetype: message.audioMessage?.mimetype,
      };
    }

    await sendToDivineSense(webhookPayload);
  } catch (error) {
    console.error("Error forwarding message to DivineSense:", error);
  }
}

/**
 * Send payload to DivineSense webhook
 */
async function sendToDivineSense(payload) {
  if (!DIVINESENSE_WEBHOOK_URL) {
    console.log("DivineSense webhook not configured, payload:", JSON.stringify(payload, null, 2));
    return;
  }

  try {
    const response = await fetch(DIVINESENSE_WEBHOOK_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        platform: "whatsapp",
        headers: {},
        payload: Buffer.from(JSON.stringify(payload)),
      }),
    });

    if (!response.ok) {
      console.error("Failed to send to DivineSense:", await response.text());
    }
  } catch (error) {
    console.error("Error sending to DivineSense:", error);
  }
}

/**
 * Start the HTTP server
 */
function startServer() {
  app.listen(PORT, () => {
    console.log(`Baileys bridge server listening on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
    console.log(`DivineSense webhook: ${DIVINESENSE_WEBHOOK_URL || "(not configured)"}`);

    // Start WhatsApp connection
    connectToWhatsApp().catch((error) => {
      console.error("Failed to connect to WhatsApp:", error);
      process.exit(1);
    });
  });
}

// Start the server
startServer();

// Graceful shutdown
process.on("SIGINT", async () => {
  console.log("\nShutting down gracefully...");

  if (srv) {
    await srv.logout();
  }

  process.exit(0);
});
