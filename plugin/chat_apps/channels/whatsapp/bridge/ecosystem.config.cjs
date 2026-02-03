/**
 * PM2 Ecosystem Configuration for Baileys WhatsApp Bridge
 *
 * This file configures the PM2 process manager for the WhatsApp bridge service.
 * It provides automatic restart, logging, and resource management.
 *
 * Usage:
 *   pm2 start ecosystem.config.cjs
 *   pm2 save
 *   pm2 startup
 *
 * Monitoring:
 *   pm2 monit
 *   pm2 logs baileys-bridge
 *   pm2 show baileys-bridge
 */

module.exports = {
  apps: [
    {
      name: 'baileys-bridge',
      script: './src/index.js',
      cwd: '/opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge',

      // Instance management
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '500M',

      // Environment configuration
      env: {
        NODE_ENV: 'production',
        PORT: 3001,
        DIVINESENSE_WEBHOOK_URL: process.env.DIVINESENSE_WEBHOOK_URL || 'http://localhost:5230/api/v1/chat_apps/webhook',
        BRIDGE_API_KEY: process.env.BRIDGE_API_KEY || '',
        ALLOWED_ORIGINS: process.env.ALLOWED_ORIGINS || '*',
      },

      // Log configuration
      error_file: '/var/log/baileys-bridge/error.log',
      out_file: '/var/log/baileys-bridge/out.log',
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      merge_logs: true,

      // Process management
      min_uptime: '10s',
      max_restarts: 10,
      restart_delay: 4000,

      // Graceful shutdown
      shutdown_with_message: true,
      listen_timeout: 10000,

      // Health check (PM2 Plus)
      health_check: {
        script: ['curl', '-f', 'http://localhost:3001/health'],
        interval: 30000,
      },
    },
  ],
};
