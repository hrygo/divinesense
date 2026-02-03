# DivineSense 聊天应用接入指南

> **零门槛入门** — 将 DivineSense AI 连接到您常用的聊天应用，随时随地享受智能助手服务。

---

## 📖 目录

1. [功能介绍](#功能介绍)
2. [支持的平台](#支持的平台)
3. [快速开始](#快速开始)
4. [Telegram 配置指南](#telegram-配置指南)
5. [WhatsApp 配置指南](#whatsapp-配置指南)
6. [钉钉配置指南](#钉钉配置指南)
7. [使用方法](#使用方法)
8. [常见问题](#常见问题)
9. [故障排查](#故障排查)

---

## 功能介绍

**DivineSense 聊天应用接入** 功能允许您将 AI 助手连接到日常使用的聊天应用中。配置完成后，您可以直接在 Telegram、WhatsApp 或钉钉中与 DivineSense AI 对话，无需打开网页或特殊应用。

### 核心功能

- ✅ **双向对话** — 在聊天应用中直接发送消息，AI 回复会自动推送回来
- ✅ **多平台支持** — 同时支持 Telegram、WhatsApp、钉钉
- ✅ **媒体处理** — 支持发送和接收图片、文件等媒体内容
- ✅ **安全加密** — 所有访问令牌使用 AES-256 加密存储
- ✅ **即时生效** — 启用/禁用开关，随时控制消息接收

### 使用场景

| 场景 | 描述 |
|:-----|:-----|
| **移动办公** | 出门在外，用手机通过 Telegram 快速查询笔记或日程 |
| **团队协作** | 在钉钉群中直接询问 AI 问题，提高团队效率 |
| **海外沟通** | 通过 WhatsApp 与海外客户沟通时，AI 实时辅助 |
| **个人助理** | 随时随地让 AI 帮您记录灵感、提醒事项 |

---

## 支持的平台

| 平台 | 类型 | 推荐场景 | 特点 |
|:-----|:-----|:---------|:-----|
| **Telegram** | Bot | 个人用户、海外用户 | 稳定、功能丰富、支持长消息 |
| **WhatsApp** | Baileys | 海外用户、个人联系 | 使用广泛、无需额外应用 |
| **钉钉** | 企业机器人 | 国内企业用户 | 与工作流集成、支持群聊 |

---

## 快速开始

### 前置条件

1. ✅ 您已经有一个可以登录的 DivineSense 账号
2. ✅ 您有 DivineSense 服务器的访问权限（如果是自托管）
3. ✅ 您的 DivineSense 服务器已配置好公网访问（或使用内网穿透）

### 步骤概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  1. 选择平台    │ -> │  2. 获取凭证    │ -> │  3. 配置 DivineSense │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                                                        v
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  5. 开始对话！   │ <- │  4. 测试连接    │ <- │  4. 设置 Webhook  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

---

## Telegram 配置指南

Telegram 是最推荐的入门平台，配置简单，功能稳定。

### 第一步：创建 Telegram Bot

1. **打开 Telegram**，在搜索框中输入 `@BotFather`

2. **点击开始对话**，发送 `/newbot` 命令

3. **按照提示操作**：
   ```
   BotFather: 好的，请给您的机器人起个名字
   您: 我的AI助手

   BotFather: 很好！现在请给机器人一个用户名（必须以 bot 结尾）
   您: MyDivineSenseBot
   ```

4. **保存 Bot Token**

   BotFather 会返回一个类似这样的 Token：
   ```
   1234567890:ABCDefGhIJKlMnOPqrSTUvwxYZ-1234567890
   ```

   ⚠️ **请妥善保管此 Token**，不要泄露给他人！

### 第二步：获取您的 Telegram User ID

1. **搜索并打开** `@userinfobot`

2. **点击** `Start` 按钮

3. **机器人会返回您的 User ID**，例如：
   ```
   Your ID: 123456789
   ```

4. **记下这个数字**，后面配置时会用到

### 第三步：在 DivineSense 中配置

1. **登录 DivineSense**，进入 **设置** 页面

2. **点击左侧菜单** 中的 **「聊天应用」**

3. **点击右上角** 「添加账号」按钮

4. **填写表单**：

   | 字段 | 填写内容 | 示例 |
   |:-----|:---------|:-----|
   | 平台 | 选择 `Telegram` | — |
   | 平台用户 ID | 您的 Telegram User ID | `123456789` |
   | 访问令牌 | Bot Token（从 BotFather 获取） | `1234567890:ABCDef...` |

5. **点击** 「确认」完成配置

### 第四步：设置 Webhook（可选但推荐）

1. **在 DivineSense 设置页面**，找到刚添加的 Telegram 账号

2. **点击** 链接图标 🔗

3. **复制显示的 Webhook URL**，格式类似：
   ```
   https://your-domain.com/api/v1/chat-apps/webhook/telegram/YOUR_TOKEN
   ```

4. **回到 Telegram**，打开 `@BotFather` 对话

5. **发送命令**：
   ```
   /setwebhook
   ```

6. **选择您的机器人**

7. **粘贴 Webhook URL** 并发送

8. **成功提示**：
   ```
   Webhook was set!
   ```

### 第五步：测试连接

1. **打开 Telegram**，找到您创建的机器人

2. **点击** `Start` 按钮或发送 `/start`

3. **发送任意消息**，例如：
   ```
   你好
   ```

4. **如果配置正确**，DivineSense AI 会回复您！

---

## WhatsApp 配置指南

WhatsApp 使用 Baileys 桥接服务，需要在服务器上运行一个 Node.js 程序。

### 第一步：准备服务器环境

⚠️ **注意**：此步骤需要服务器访问权限，如果您使用的是 DivineSense 云服务，请联系管理员。

```bash
# SSH 登录到服务器
ssh user@your-server

# 进入 WhatsApp 桥接目录
cd /opt/divinesense/plugin/chat_apps/channels/whatsapp/bridge

# 安装依赖
npm install
```

### 第二步：配置桥接服务

1. **创建配置文件** `.env`：
   ```bash
   # HTTP 服务端口（默认 3001）
   PORT=3001

   # DivineSense Webhook URL
   # 请将 your-domain.com 替换为您的域名
   DIVINESENSE_WEBHOOK_URL=https://your-domain.com/api/v1/chat-apps/webhook

   # Baileys 认证文件路径
   BAILEYS_AUTH_FILE=./baileys_auth_info.json
   ```

2. **启动桥接服务**：
   ```bash
   npm start
   ```

3. **查看日志**，确认服务正常运行：
   ```
   Baileys bridge server listening on port 3001
   Health check: http://localhost:3001/health
   ```

### 第三步：获取 WhatsApp QR 码

1. **在服务器上**，访问桥接服务的信息端点：
   ```bash
   curl http://localhost:3001/info
   ```

2. **会返回 QR 码**（在终端中显示）

3. **或者在 DivineSense 设置页面**中添加 WhatsApp 账号后，系统会显示 QR 码

### 第四步：扫码绑定

1. **打开 WhatsApp** 手机应用

2. **进入设置**：
   - Android: ⋮ 菜单 → 设置
   - iOS: 设置 ⚙️

3. **点击**「已连接的设备」或「链接设备」

4. **点击**「链接设备」或「连接设备」

5. **扫描 QR 码**：

   ![扫码流程](https://faq.whatsapp.com/general/attachments-and-photos/29760017)

6. **扫描成功后**，桥接服务会显示：
   ```
   WhatsApp connection opened
   ```

### 第五步：在 DivineSense 中配置

1. **登录 DivineSense**，进入 **设置 → 聊天应用**

2. **点击** 「添加账号」

3. **填写表单**：

   | 字段 | 填写内容 |
   |:-----|:---------|
   | 平台 | 选择 `WhatsApp` |
   | 平台用户 ID | 您的 WhatsApp 手机号（可选，用于标识） |

4. **点击** 「确认」

### 第六步：获取您的 WhatsApp JID

1. **给任意联系人**（包括您自己）发送消息

2. **在桥接服务日志中**，会显示发送者的 JID：
   ```
   Message from: 1234567890@s.whatsapp.net
   ```

3. **复制这个 JID**，填入 DivineSense 配置中的「平台用户 ID」字段

---

## 钉钉配置指南

钉钉机器人适合国内企业用户，可以集成到钉钉群聊中。

### 第一步：创建钉钉应用

1. **登录** [钉钉开放平台](https://open.dingtalk.com/)

2. **进入**「应用开发」→「企业内部开发」→「H5微应用」或「小程序」

3. **点击**「创建应用」，填写基本信息：
   - 应用名称：`DivineSense AI`
   - 应用描述：`智能助手服务`

4. **创建完成后**，记录以下信息：
   - **AppKey**（也叫 ClientId）
   - **AppSecret**（也叫 ClientSecret）

### 第二步：获取企业内部机器人 Webhook

钉钉支持两种机器人方式，推荐使用「企业内部机器人」。

#### 方式一：群机器人（简单模式）

1. **在钉钉群聊**中，点击右上角 `...`

2. **选择**「机器人」→「添加机器人」→「自定义」

3. **填写机器人信息**：
   - 机器人名字：`DivineSense AI`
   - 安全设置：选择「加签」或「关键词」（推荐「加签」）

4. **点击完成**，会获得：
   - **Webhook URL**，格式：
     ```
     https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxx
     ```

5. **记录 Webhook URL**，后面配置时使用

#### 方式二：企业内部机器人（高级模式）

1. **在钉钉开放平台**，进入您创建的应用

2. **进入**「消息推送」→「机器人」

3. **配置**消息推送地址：
   ```
   https://your-domain.com/api/v1/chat-apps/webhook/dingtalk
   ```

4. **获取** AppKey 和 AppSecret

### 第三步：获取您的钉钉 User ID

1. **在钉钉PC版**中，点击个人头像

2. **查看**「个人信息」→「成员信息」

3. **记录**您的「工号」或「Union ID」

### 第四步：在 DivineSense 中配置

1. **登录 DivineSense**，进入 **设置 → 聊天应用**

2. **点击** 「添加账号」

3. **填写表单**：

   | 字段 | 填写内容 | 示例 |
   |:-----|:---------|:-----|
   | 平台 | 选择 `钉钉` | — |
   | 平台用户 ID | 您的钉钉工号或 Union ID | `manager1234` |
   | 访问令牌 | AppKey | `dingxxxxx` |
   | Webhook URL | 群机器人 Webhook URL | `https://oapi.dingtalk.com/...` |

4. **点击** 「确认」

---

## 使用方法

### 发送消息

配置完成后，您可以在聊天应用中直接发送消息：

```
你今天有什么安排？
```

DivineSense AI 会解析您的消息，查询相关数据，并直接回复到聊天应用中。

### 支持的命令类型

| 类型 | 示例命令 | AI 功能 |
|:-----|:---------|:---------|
| **笔记查询** | `搜索关于 Python 的笔记` | 语义搜索笔记 |
| **日程查询** | `今天有什么会议？` | 查询日程安排 |
| **日程创建** | `明天下午3点提醒我开会` | 创建新日程 |
| **综合查询** | `总结这周的工作` | 组合查询 |

### AI 代理智能路由

DivineSense 会根据您的问题自动选择最合适的 AI 代理：

- 📝 **灰灰（笔记专家）** — 笔记搜索、知识查询
- 📅 **时巧（日程助理）** — 日程管理、时间提醒
- 🤖 **折衷（综合助理）** — 复杂任务、多数据源查询

---

## 常见问题

### Q1: 为什么收不到 AI 的回复？

**可能原因和解决方案**：

| 原因 | 解决方案 |
|:-----|:---------|
| Webhook 未设置 | 参考「设置 Webhook」步骤 |
| 凭证被禁用 | 在设置中检查账号状态，确保「已启用」 |
| 网络问题 | 检查服务器网络连接和防火墙设置 |
| Bot Token 错误 | 重新验证 Token 是否正确 |

### Q2: Telegram 提示 "Bot was blocked by the user"

**解决方法**：

1. 打开与机器人的对话
2. 点击 `Start` 或发送 `/start`
3. 如果还不行，点击 `Stop` 后重新 `Start`

### Q3: WhatsApp 二维码过期怎么办？

**解决方法**：

1. 删除 `baileys_auth_info.json` 文件
2. 重启桥接服务：`npm start`
3. 重新扫描新的 QR 码

### Q4: 钉钉机器人只返回配置信息，不回复我的消息？

**解决方法**：

1. 检查「关键词」设置，确保消息中包含关键词
2. 或使用「加签」方式，更安全可靠
3. 确认 DivineSense 服务器能接收外网请求

### Q5: 如何同时使用多个平台？

**答**：完全可以！您可以为同一个 DivineSense 账号配置多个平台的凭证。AI 会根据消息来源平台进行回复。

---

## 故障排查

### 检查清单

```
□ DivineSense 服务运行正常
□ 聊天应用账号已启用
□ Webhook URL 正确配置
□ 网络连接正常
□ Token/密钥 未过期
□ 防火墙允许外部访问
```

### 调试模式

启用 DivineSense 调试日志：

```bash
# 在服务器上
export DIVINESENSE_LOG_LEVEL=debug
systemctl restart divinesense
journalctl -u divinesense -f
```

### 日志查看

```bash
# 查看 DivineSense 日志
tail -f /var/log/divinesense/app.log

# 查看 WhatsApp 桥接服务日志
tail -f /var/log/baileys-bridge/output.log
```

### 常见错误码

| 错误 | 含义 | 解决方案 |
|:-----|:-----|:---------|
| `401 Unauthorized` | Token 无效或过期 | 重新获取 Token |
| `403 Forbidden` | 用户被禁用 | 检查账号状态 |
| `404 Not Found` | API 端点不存在 | 检查 URL 配置 |
| `500 Internal Server Error` | 服务器错误 | 查看服务器日志 |
| `502 Bad Gateway` | 桥接服务不可用 | 检查桥接服务状态 |

---

## 高级配置

### 自定义 AI 代理

您可以为不同平台配置不同的 AI 代理：

```go
// 在 channel 配置中
type ChannelConfig struct {
    AgentType string  // "memo", "schedule", "amazing"
    Platform  string  // "telegram", "whatsapp", "dingtalk"
}
```

### 消息过滤

配置消息过滤规则，避免处理不相关的消息：

```bash
# 环境变量
DIVINESENSE_CHAT_APPS_MESSAGE_FILTER=true
DIVINESENSE_CHAT_APPS_MIN_LENGTH=2
```

### 速率限制

防止消息轰炸，配置速率限制：

```bash
DIVINESENSE_CHAT_APPS_RATE_LIMIT=10
# 每分钟最多处理 10 条消息
```

---

## 安全建议

1. 🔒 **定期更换 Token** — 建议每 3-6 个月更换一次
2. 🔒 **使用加密** — 确保 `DIVINESENSE_CHAT_APPS_SECRET_KEY` 已设置
3. 🔒 **限制访问** — Webhook URL 不要泄露给无关人员
4. 🔒 **监控日志** — 定期检查异常访问记录
5. 🔒 **备份凭证** — 定期备份 `baileys_auth_info.json` 文件

---

## 获取帮助

如果遇到问题：

1. 📖 查看本文档的「常见问题」和「故障排查」部分
2. 💬 访问 [GitHub Issues](https://github.com/hrygo/divinesense/issues)
3. 📧 联系技术支持：support@divinesense.io

---

**祝您使用愉快！🎉**

*最后更新：2026年2月*
