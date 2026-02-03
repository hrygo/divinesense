# Chat Apps 集成用户指南

> 通过 Telegram 和钉钉机器人，随时随地与 DivineSense AI 代理对话

---

## 概述

DivineSense 支持将聊天平台（Telegram、钉钉）机器人接入到个人 AI 系统。配置后，你可以通过这些平台：

- **查询笔记** - 搜索和检索个人笔记内容
- **管理日程** - 创建、查看日程安排
- **AI 对话** - 与智能代理进行自然语言对话
- **获取提醒** - 接收重要事件通知

---

## 支持的平台

| 平台 | 状态 | 说明 |
|:-----|:-----|:-----|
| **Telegram** | ✅ 完全支持 | Bot API + Webhook，支持流式响应 |
| **钉钉** | ✅ 完全支持 | 群机器人 + Webhook，HMAC 签名验证 |
| **WhatsApp** | ✅ 完全支持 | Baileys Node.js 桥接服务，需单独部署 |

---

## 快速开始

### 步骤 1：准备聊天平台凭证

#### Telegram Bot

1. 访问 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 创建新机器人
3. 按提示设置名称和用户名
4. 保存生成的 **Bot Token**（格式：`1234567890:ABCDefGHIjklMNOpqrsTUVwxyz`）

#### 钉钉群机器人

1. 访问 [钉钉开放平台](https://open.dingtalk.com/)
2. 登录后进入「应用开发」→「企业内部应用」
3. 创建应用或选择现有应用
4. 在「应用凭证」页面获取：
   - **AppKey**（企业 ID）
   - **AppSecret**（应用密钥）
5. 在「消息推送」中配置：
   - **消息接收地址**：`https://your-domain.com/api/v1/chat_apps/webhook?platform=dingtalk`
   - 保存生成的 **AccessToken**

#### WhatsApp (Baileys 桥接服务)

WhatsApp 需要单独部署 Baileys Node.js 桥接服务：

1. **部署桥接服务**：
   ```bash
   cd plugin/chat_apps/channels/whatsapp/bridge
   npm install
   cp .env.example .env
   # 编辑 .env 配置 DivineSense webhook URL
   npm start
   ```

2. **配对 WhatsApp**：
   - 访问 `http://localhost:3001/info` 获取 QR 码
   - 打开 WhatsApp → 设置 → 关联设备 → 关联设备
   - 扫描 QR 码完成配对

3. **获取凭证**：
   - **Bridge URL**: 桥接服务地址（如 `http://localhost:3001`）
   - 配置后 DivineSense 将自动与桥接服务通信

**生产部署**：建议使用 PM2 或 systemd 管理桥接服务：
```bash
pm2 start src/index.js --name baileys-bridge
pm2 save
```

### 步骤 2：配置 DivineSense

1. 登录 DivineSense Web 界面
2. 进入「设置」→「Chat Apps」
3. 选择平台并填入凭证：

| 字段 | Telegram | 钉钉 | WhatsApp |
|:-----|:---------|:-----|:---------|
| **Bot Token / AppKey** | `1234567890:ABC...` | `dingxxxxx` | - |
| **App Secret** | - | `SEC...` | - |
| **Webhook URL / Bridge URL** | 自动生成 | 自动生成 | `http://localhost:3001` |

4. 点击「保存」

### 步骤 3：配置 Webhook

#### Telegram Bot

1. 向 Bot 发送 `/setwebhook`
2. 发送 webhook URL：`https://your-domain.com/api/v1/chat_apps/webhook?platform=telegram`

#### 钉钉群机器人

1. 在群设置中添加自定义机器人
2. 搜索你的应用并添加
3. 完成！

#### WhatsApp

1. 确保 Baileys 桥接服务正在运行
2. 桥接服务会自动将消息转发到 DivineSense
3. 完成！（无需额外配置）

---

## 使用指南

### Telegram Bot 使用

```
/start          # 开始使用（可选）
<任意消息>      # 直接发送问题
```

**示例对话**：
```
你: 搜索关于"Python asyncio"的笔记
Bot: [返回相关笔记内容...]

你: 创建明天下午3点的会议
Bot: 好的，已为您创建明天15:00的会议...
```

### 钉钉群机器人使用

在群中直接 @机器人：

```
@机器人 搜索上周的会议记录
@机器人 帮我创建周五下午的提醒
```

### WhatsApp 使用

直接发送消息给 DivineSense 联系的号码：

```
你: 搜索关于"Python asyncio"的笔记
Bot: [返回相关笔记内容...]

你: 创建明天下午3点的会议
Bot: 好的，已为您创建明天15:00的会议...
```

---

## 功能详解

### 1. 笔记查询

发送关键词或自然语言查询：

- `"搜索关于Golang的笔记"`
- `"查找昨天关于AI的笔记"`
- `"有什么TODO待办事项吗"`

### 2. 日程管理

- 自然语言创建：`明天下午3点开会`、`下周一下午提醒我`
- 查询日程：`今天的安排`、`本周会议`
- 冲突检测：自动检测时间冲突

### 3. AI 对话

自动路由到合适的 AI 代理：

- **灰灰**（笔记）→ 搜索和知识检索
- **时巧**（日程）→ 日程管理
- **折衷**（综合）→ 组合查询

---

## 安全说明

### Token 加密

所有敏感凭证（Bot Token、App Secret）使用 **AES-256-GCM** 加密存储：

- 密钥长度：必须 32 字节
- 加密算法：AES-256-GCM（带认证）
- 存储：数据库加密字段

### 环境变量配置

```bash
# 必需：32字节加密密钥
DIVINESENSE_CHAT_APPS_SECRET_KEY=<your-32-byte-key>

# 实例 URL（用于 Webhook 设置）
DIVINESENSE_INSTANCE_URL=https://your-domain.com
```

**生成加密密钥**：
```bash
# Linux/Mac
openssl rand -base64 32 | head -c 32

# Python
python3 -c "import secrets; print(secrets.token_urlsafe(32)[:32])"
```

---

## Webhook 配置说明

### Telegram

**Webhook URL**：
```
https://your-domain.com/api/v1/chat_apps/webhook?platform=telegram
```

**验证方式**：Bot Token 验证

### 钉钉

**Webhook URL**：
```
https://your-domain.com/api/v1/chat_apps/webhook?platform=dingtalk
```

**验证方式**：HMAC-SHA256 签名

**签名计算**（钉钉要求）：
```
stringToSign = timestamp + "\n" + appSecret
signature = base64(hmac_sha256(stringToSign))
```

### WhatsApp

**架构模式**：Baileys Node.js 桥接服务

**验证方式**：通过桥接服务内部验证

**工作原理**：
```
WhatsApp ──→ Baileys Bridge ──→ DivineSense Webhook
                     ↑
              (Node.js 服务)
```

**桥接服务 API**：
- `GET /health` - 健康检查
- `GET /info` - 获取连接状态和 QR 码
- `POST /send` - 发送消息到 WhatsApp
- `GET /download?url=...` - 下载媒体

---

## API 端点

### 注册凭证

```bash
curl -X POST https://your-domain.com/api/v1/chat_apps/credentials \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "PLATFORM_TELEGRAM",
    "platform_user_id": "telegram_user_123",
    "access_token": "bot_token_here"
  }'
```

### 列出凭证

```bash
curl https://your-domain.com/api/v1/chat_apps/credentials \
  -H "Authorization: Bearer <token>"
```

### 删除凭证

```bash
curl -X DELETE https://your-domain.com/api/v1/chat_apps/credentials/123 \
  -H "Authorization: Bearer <token>"
```

---

## 故障排查

### 常见问题

**Q: Telegram Bot 返回 "Bad Request"**
- 检查 Bot Token 格式是否正确
- 确认 Webhook URL 可访问
- 查看服务器日志

**Q: 钉钉签名验证失败**
- 检查 AppSecret 是否正确
- 确认服务器时间与钉钉服务器同步
- 检查 Webhook URL 配置

**Q: AI 无响应**
- 确认 AI 服务已启用
- 检查环境变量配置
- 查看服务器日志

**Q: 消息发送失败**
- 检查聊天 ID 是否正确
- 确认机器人有权限发送消息
- 查看平台限制（如频率限制）

### 调试日志

查看服务器日志：
```bash
# 二进制模式
sudo journalctl -u divinesense -f

# Docker 模式
docker logs divinesense -f
```

相关日志标签：
- `platform=telegram`
- `platform=dingtalk`
- `user_id=<your-id>`

---

## 隐私说明

- 所有聊天消息仅在处理时临时存储
- AI 对话上下文保留 30 天后自动清理
- 敏感凭证加密存储，无法从日志中获取
- 用户可随时删除关联的聊天平台

---

## 下一步

- [ ] 配置 Telegram Bot
- [ ] 配置钉钉群机器人
- [ ] 测试 AI 对话功能
- [ ] 设置日程提醒
- [ ] 查看个人笔记

---

**最后更新**: 2026-02-03
**相关文档**:
- [系统架构](../dev-guides/ARCHITECTURE.md)
- [后端开发](../dev-guides/BACKEND_DB.md)
- [功能规格](../specs/chat-apps-integration.md)
