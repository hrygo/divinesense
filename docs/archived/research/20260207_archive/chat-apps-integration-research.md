# Chat Apps 接入调研报告

> **调研日期**: 2026-02-03
> **Issue**: [#53](https://github.com/hrygo/divinesense/issues/53)
> **状态**: ✅ 已完成

---

## 执行摘要

为 DivineSense 添加多平台 Chat Apps 接入能力，支持用户通过 Telegram、WhatsApp、钉钉机器人与 AI 代理交互。

| 维度 | 评估 |
|:-----|:-----|
| **技术可行性** | ✅ 高 — DivineSense 已有完整 AI 代理架构 |
| **用户价值** | ✅ 高 — 随时随地 AI 助手 |
| **实现复杂度** | ⚠️ 中高 — 多平台协议 + 多媒体处理 |
| **预估工作量** | 3-4 人周 |

---

## 1. 目标平台分析

### 1.1 Telegram Bot API

| 特性 | 评估 |
|:-----|:-----|
| **开发难度** | ⭐ 低 — 标准 HTTP Bot API |
| **消息类型** | 文本、图片、语音、视频、文档、贴纸 |
| **文件上传** | 默认 50MB，本地服务器可达 2GB |
| **语音识别** | 需自行集成 Whisper |
| **文档** | [Telegram Bot API](https://core.telegram.org/bots/api) |

### 1.2 WhatsApp Business API

| 特性 | 评估 |
|:-----|:-----|
| **开发难度** | ⭐⭐⭐ 中 — 需 Business API 或 Baileys 库 |
| **实现方式** | 推荐 Baileys (Node.js) 桥接 |
| **消息类型** | 文本、图片、语音、视频、文档、位置 |
| **文档** | [WhatsApp Media API](https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/) |

### 1.3 钉钉机器人

| 特性 | 评估 |
|:-----|:-----|
| **开发难度** | ⭐⭐ 中 — 回调模式 + 签名验证 |
| **消息类型** | 文本、图片、语音、视频、文件、富文本 |
| **文件下载** | 通过 `downloadCode` 临时下载码 |
| **文档** | [钉钉机器人接收消息](https://open.dingtalk.com/document/dingstart/robot-receive-message)、[下载文件内容](https://open.dingtalk.com/document/development/download-the-file-content-of-the-robot-receiving-message) |

---

## 2. DivineSense 现有资产

| 组件 | 位置 | 复用方式 |
|:-----|:-----|:---------|
| **AI Chat 流式** | `AIService.Chat` + SSE | 直接复用 |
| **Parrot 代理** | `plugin/ai/agent/*_parrot.go` | 无需修改 |
| **会话管理** | `conversation_context` 表 | 扩展 `channel_type` 字段 |
| **用户设置** | `user_setting` 表 | 扩展存储 Chat App 凭证 |
| **通知通道** | `plugin/ai/reminder/channels.go` | 扩展为双向通道 |

**关键发现**：数据库中已存在 `USER_SETTING_TELEGRAM_USER_ID` 配置项（历史遗留）

---

## 3. Nanobot 借鉴要点

| 设计模式 | Nanobot 实现 | DivineSense 应用 |
|:---------|:-------------|:-----------------|
| **通道基类** | `BaseChannel` 抽象接口 | 定义 `ChatChannel` Go 接口 |
| **消息总线** | `MessageBus` + 事件消息 | 复用现有 SSE 架构 |
| **权限控制** | `allow_from` 白名单 | 关联 `user_setting` 表验证 |
| **多媒体处理** | 消息类型提取 + 文件下载 | 实现 MediaHandler 组件 |

**Nanobot 项目链接**: [HKUDS/nanobot](https://github.com/HKUDS/nanobot)

---

## 4. 技术方案

### 4.1 架构图

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
                  │  (用户验证 + 消息分发)│
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   MediaHandler      │
                  │ (图片OCR/语音转文字) │
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   ChatRouter        │
                  │  (现有 AI 代理路由)   │
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   Parrot Agents     │
                  │ (Memo/Schedule/...) │
                  └─────────────────────┘
```

### 4.2 数据库设计

```sql
-- 新增表：chat_app_credential
CREATE TABLE chat_app_credential (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,              -- 'telegram', 'whatsapp', 'dingtalk'
  platform_user_id TEXT NOT NULL,      -- 平台用户ID
  access_token TEXT,                   -- 加密存储
  webhook_url TEXT,                    -- 回调地址（钉钉用）
  enabled BOOLEAN DEFAULT true,
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  UNIQUE(user_id, platform)
);

-- 扩展 conversation_context 表
ALTER TABLE conversation_context
ADD COLUMN channel_type TEXT;         -- 'web', 'telegram', 'whatsapp', 'dingtalk'
```

### 4.3 组件划分

```
plugin/chat_apps/
├── channels/
│   ├── base.go              # ChatChannel 接口定义
│   ├── telegram/
│   │   ├── telegram.go      # Telegram Bot 实现
│   │   └── webhook.go       # Webhook 处理
│   ├── whatsapp/
│   │   └── bridge.go        # WhatsApp Baileys 桥接
│   └── dingtalk/
│       ├── dingtalk.go      # 钉钉机器人实现
│       └── crypto.go        # 签名验证
├── media/
│   ├── handler.go           # 多媒体处理器接口
│   ├── whisper.go           # 语音转文字
│   └── ocr.go              # 图片 OCR
├── router.go                # 通道路由器
└── store.go                 # 凭证存储
```

---

## 5. 实施计划

| Phase | 内容 | 工作量 |
|:-----|:-----|:-------|
| **Phase 1** | Telegram Bot 接入 | 1 人周 |
| **Phase 2** | WhatsApp Baileys 桥接 | 1 人周 |
| **Phase 3** | 钉钉机器人接入 | 1 人周 |
| **Phase 4** | 多媒体处理增强 (OCR/Whisper) | 0.5 人周 |

---

## 6. 风险与缓解

| 风险 | 影响 | 措施 |
|:-----|:-----|:-----|
| **平台协议变更** | 高 | 版本锁定 + 抽象层隔离 |
| **多媒体文件处理** | 中 | 异步队列 + 大小限制 |
| **Token 泄露** | 高 | 数据库加密存储 |
| **WhatsApp 稳定性** | 中 | 提供 Business API 备选 |

---

## 7. 参考资源

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [WhatsApp Business API - Media](https://developers.facebook.com/documentation/business-messaging/whatsapp/business-phone-numbers/media/)
- [钉钉机器人接收消息](https://open.dingtalk.com/document/dingstart/robot-receive-message)
- [钉钉下载机器人接收消息的文件内容](https://open.dingtalk.com/document/development/download-the-file-content-of-the-robot-receiving-message)
- [HKUDS/nanobot](https://github.com/HKUDS/nanobot)
- [OpenClaw 架构调研](https://github.com/hrygo/divinesense/blob/main/docs/archived/research_cleanup_20260131/reports/OPENCLAW_RESEARCH.md)

---

## 8. 相关 Issue

- [#53 [feat] Chat Apps 接入支持 (Telegram/WhatsApp/钉钉)](https://github.com/hrygo/divinesense/issues/53)

---

**调研人员**: Claude Opus 4.5
**文档版本**: v1.0
**更新日期**: 2026-02-03
