# 架构文档

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **最后检查**: v6.1 (AI Core 重构)

## 项目概述

DivineSense (神识) 是一款隐私优先、轻量级的笔记服务，通过 AI 驱动的「鹦鹉」代理增强用户体验。
- **核心架构**：Go 后端 (Echo/Connect RPC) + React 前端 (Vite/Tailwind) —— **单二进制分发**
- **数据存储**：PostgreSQL（生产环境，完整 AI 支持），SQLite（仅开发环境，**无 AI**）详见 [#9](https://github.com/hrygo/divinesense/issues/9)
- **核心特性**：多代理 AI 系统、语义搜索、日程助理、自托管无遥测
- **端口**：后端 28081，前端 25173，PostgreSQL 25432（开发环境）

## 技术栈

| 领域   | 技术选型                                                                                             |
| :----- | :--------------------------------------------------------------------------------------------------- |
| 后端   | Go 1.25, Echo, Connect RPC, pgvector                                                                 |
| 前端   | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, TanStack Query                               |
| 数据库 | PostgreSQL 16+（生产），SQLite（开发，**无 AI**）[#9](https://github.com/hrygo/divinesense/issues/9) |
| AI     | DeepSeek V3（LLM），SiliconFlow（Embedding、Reranker）                                               |

---

## 项目架构

### 目录结构

```
divinesense/
├── cmd/divinesense/     # 主程序入口
├── server/              # HTTP/gRPC 服务器 & 路由
│   ├── router/          # API 处理器（v1 实现）
│   ├── queryengine/     # 查询路由 & 意图检测
│   ├── runner/          # 后台任务运行器
│   ├── scheduler/       # 日程管理
│   └── service/         # 业务逻辑层
├── ai/                  # 🔴 AI 核心模块（一级模块）
│   ├── agent/           #   Parrot 代理（MemoParrot、ScheduleParrot、AmazingParrot、GeekParrot、EvolutionParrot）
│   ├── router/          #   三层意图路由
│   ├── vector/          #   Embedding 服务
│   ├── memory/          #   情景记忆
│   ├── session/         #   对话持久化
│   ├── cache/           #   LRU 缓存层
│   ├── metrics/         #   代理性能追踪
│   ├── core/            #   AI 基础设施
│   │   ├── embedding/   #     嵌入服务（从 server/ai/ 迁移）
│   │   ├── retrieval/   #     检索系统（从 server/retrieval/ 迁移）
│   │   ├── reranker/    #     重排服务
│   │   └── llm/         #     LLM 客户端
│   ├── rag/             #   RAG 高级功能
│   ├── tags/            #   标签建议
│   ├── duplicate/       #   重复检测
│   ├── habit/           #   习惯学习
│   ├── genui/           #   生成式 UI
│   ├── graph/           #   知识图谱
│   ├── prediction/      #   预测引擎
│   ├── reminder/        #   提醒系统
│   ├── schedule/        #   日程 AI
│   ├── aitime/          #   AI 时间解析
│   ├── timeout/         #   超时处理
│   ├── review/          #   审查服务
│   ├── context/         #   上下文构建
│   └── config.go        #   AI 配置
├── plugin/              # 其他可选插件（非 AI）
│   ├── scheduler/       # 任务调度
│   ├── storage/         # 存储适配器（S3、本地）
│   ├── idp/             # 身份提供商
│   ├── markdown/        # Markdown 插件
│   ├── ocr/             # OCR 插件
│   ├── webhook/         # Webhook 插件
│   └── chat_apps/       # 聊天应用接入（Telegram/钉钉/WhatsApp）
├── store/               # 数据存储层
│   ├── db/              # 数据库实现
│   │   ├── postgres/    # PostgreSQL with pgvector
│   │   └── sqlite/      # SQLite（仅开发环境，无 AI）
│   └── [interfaces]     # 存储抽象
├── proto/               # Protobuf 定义（API 契约）
│   ├── api/v1/          # API 服务定义
│   └── store/           # Store 服务定义
├── web/                 # React 前端应用
│   ├── src/
│   │   ├── pages/       # 页面组件
│   │   ├── layouts/     # 布局组件
│   │   ├── components/  # UI 组件
│   │   ├── locales/     # i18n 翻译（en、zh-Hans、zh-Hant）
│   │   └── hooks/       # React hooks
│   └── package.json
├── docs/                # 文档
├── scripts/             # 开发和构建脚本
└── docker/              # Docker 配置
```

### 核心组件

1. **单二进制构建 (Single Binary)**：
   - **前端集成**：使用 `go:embed` 将 `web/dist` 打包进 Go 二进制文件。
   - **数据库迁移**：SQL 脚本同样通过 `embed` 嵌入，启动时自动执行 `store/migrator.go` 进行架构升级。
   - **优势**：分发无需 Node.js/Nginx，直接运行全栈服务。

2. **服务器初始化**：Profile → DB → Store → Server
   - 使用 Echo 框架 + Connect RPC（gRPC/HTTP 转码）
   - 静态资源服务支持 Gzip 压缩、SPA 路由回退及强缓存优化。

2. **AI 核心模块** (`ai/`)：
   - LLM 提供商：DeepSeek、OpenAI、Ollama
   - Embedding：SiliconFlow (BAAI/bge-m3)、OpenAI
   - Reranker：BAAI/bge-reranker-v2-m3
   - 所有 AI 功能可选（由 `DIVINESENSE_AI_ENABLED` 控制）
   - 包含从 `server/ai/` 和 `server/retrieval/` 迁移的嵌入和检索功能

3. **插件系统** (`plugin/`)：
   - 不再包含 AI 功能（已提升为一级模块）
   - 包含：调度器、存储适配器、身份提供商、Markdown、OCR、Webhook、聊天应用接入（Chat Apps）等

### 聊天应用集成 (Chat Apps)

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **最后检查**: v6.1

**架构概览**：将 DivineSense AI 连接到 Telegram、WhatsApp、钉钉等聊天平台，实现多渠道智能助手服务。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Chat Apps Gateway                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────────┐ │
│  │   Telegram   │  │   WhatsApp   │  │         DingTalk             │ │
│  │   Channel    │  │   Bridge     │  │         Channel              │ │
│  │              │  │              │  │                              │ │
│  │ BotToken     │  │ Baileys      │  │ HMAC-SHA256 + Timestamp      │ │
│  │ Webhook      │  │ HTTP API     │  │ Webhook                      │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────────────┘ │
└─────────┼──────────────────┼───────────────────────┼────────────────────┘
          │                  │                       │
          └──────────────────┼───────────────────────┘
                             ▼
                  ┌─────────────────────┐
                  │  ChatChannel Router  │
                  │  (user verification) │
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │  AI Agent Router     │
                  │  (AUTO: memo/schedule)│
                  └──────────┬──────────┘
                             ▼
                  ┌─────────────────────┐
                  │   Parrot Agents      │
                  │  (灰灰/时巧/折衷)     │
                  └─────────────────────┘
```

**目录结构**：
```
plugin/chat_apps/
├── channels/
│   ├── base.go                    # ChatChannel 接口定义
│   ├── router.go                  # 通道路由器（用户验证）
│   ├── telegram/
│   │   ├── telegram.go            # Telegram Bot 实现
│   │   └── webhook.go             # Webhook 处理
│   ├── whatsapp/
│   │   └── bridge.go              # Baileys 桥接客户端
│   └── dingtalk/
│       ├── dingtalk.go            # 钉钉机器人实现
│       └── crypto.go              # 签名验证
├── store/
│   ├── db.go                      # 数据库操作
│   └── crypto.go                  # Token 加密（AES-256-GCM）
├── metrics/
│   └── metrics.go                 # Webhook 指标收集
└── types.go                       # 通用类型定义
```

**安全机制**：
- Token 加密：AES-256-GCM，32 字节密钥
- 输入验证：平台白名单 + 长度限制
- Webhook 签名：HMAC-SHA256 + 时间戳（5分钟窗口，防重放）
- 并发安全：单一 Mutex 防止 Token 缓存竞态

**数据库表**：`chat_app_credential`

| 字段 | 类型 | 描述 |
|:-----|:-----|:-----|
| `id` | SERIAL | 主键 |
| `user_id` | INTEGER | 所属用户（外键） |
| `platform` | TEXT | 平台名称（telegram/whatsapp/dingtalk） |
| `platform_user_id` | TEXT | 平台用户 ID |
| `access_token` | TEXT | 加密存储的访问令牌 |
| `app_secret` | TEXT | 加密存储的应用密钥（钉钉） |
| `webhook_url` | TEXT | Webhook URL |
| `enabled` | BOOLEAN | 启用状态 |

**详细文档**：
- [用户指南](../user-guides/CHAT_APPS.md)
- [开发者指南](../guides/CHAT_APPS.md)
- [技术规格](../specs/chat-apps-integration.md)

4. **后台运行器** (`server/runner/`):
   - 异步生成笔记 Embedding
   - AI 操作任务队列
   - AI 启用时自动运行

5. **存储层**：
   - 接口定义在 `store/`
   - 驱动特定实现在 `store/db/{postgres,sqlite}/`
   - 迁移系统在 `store/migration/`

6. **智能查询引擎** (`server/queryengine/`):
   - 自适应检索（BM25 + 向量搜索 + 选择性重排）
   - 智能查询路由（检测日程 vs. 搜索查询）
   - 自然语言日期解析
   - 带冲突检测的日程助理

### API 服务

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **覆盖范围**: `proto/api/v1/*.proto` | **最后检查**: v6.1

| 服务 | Proto 文件 | 描述 |
|:-----|:-----------|:-----|
| **ActivityService** | `activity_service.proto` | 用户活动记录 |
| **AttachmentService** | `attachment_service.proto` | 附件管理 |
| **AuthService** | `auth_service.proto` | 认证授权 |
| **ChatAppService** | `chat_app_service.proto` | 聊天应用接入（Telegram/钉钉/WhatsApp） |
| **IdpService** | `idp_service.proto` | 身份提供商集成 |
| **InstanceService** | `instance_service.proto` | 实例配置 |
| **MemoService** | `memo_service.proto` | 笔记 CRUD |
| **ScheduleService** | `schedule_service.proto` | 日程管理 |
| **ShortcutService** | `shortcut_service.proto` | 快捷方式 |
| **UserService** | `user_service.proto` | 用户管理 |
| **Common** | `common.proto` | 通用类型定义 |
| **AIService** | `ai_service.proto` | AI 聊天、嵌入、检索 |

---

## 🔒 Git Hooks 工作流

DivineSense 使用 **pre-commit + pre-push** hooks 确保代码质量：

| Hook | 检查内容 | 速度 | 触发时机 |
|:-----|:---------|:-----|:---------|
| **pre-commit** | `go fmt` + `go vet` + `pnpm lint:fix` | ~5秒 | 每次 `git commit` |
| **pre-push** | `golangci-lint` + `go test` + `pnpm build` | ~1分钟 | 每次 `git push` |

### 安装与使用

```bash
# 安装 hooks
make install-hooks

# 本地 CI 检查（与 GitHub Actions 一致）
make ci-check
make ci-backend
make ci-frontend

# 跳过检查
git commit --no-verify -m "WIP"
git push --no-verify
```

> **详细规范**：参见 [Git 工作流](../../.claude/rules/git-workflow.md)

---

## Parrot 代理架构

### 代理类型 (`ai/agent/`)

|  AgentType  | 鹦鹉名称 | 文件                    | 中文名 | 描述                             |
| :---------: | :------- | :---------------------- | :----- | :------------------------------- |
|   `MEMO`    | 灰灰     | `memo_parrot.go`        | 灰灰   | 笔记搜索和检索专家               |
| `SCHEDULE`  | 时巧     | `schedule_parrot_v2.go` | 时巧   | 日程创建和管理                   |
|  `AMAZING`  | 折衷     | `amazing_parrot.go`     | 折衷   | 综合助理（笔记 + 日程）          |
|   `GEEK`    | 极客     | `geek_parrot.go`        | 极客   | Claude Code CLI 通信层（零 LLM） |
| `EVOLUTION` | 进化     | `evolution_parrot.go`   | 进化   | 自我进化能力（源代码修改）       |

### 代理路由器

**位置**：`ai/agent/chat_router.go`

ChatRouter 实现**三层**意图分类系统：

```
用户输入 → EvolutionMode? ─Yes→ EvolutionParrot（自我进化）
                  │
                  No
                  ↓
           GeekMode? ─Yes→ GeekParrot（Claude Code CLI）
                  │
                  No
                  ↓
           ChatRouter.Route()
                  ↓
           routerService? ─Yes→ 三层路由
                  │          (规则 + 历史 + LLM)
                  │
                  No（向后兼容）
                  ↓
           routeByRules()     ← 快速路径（0ms）
                  ↓
         匹配成功? ─Yes→ 返回（置信度 ≥0.80）
                  │
                  No
                  ↓
           routeByLLM()       ← 慢速路径（~400ms）
                  ↓
         Qwen2.5-7B-Instruct
         （严格 JSON Schema）
                  ↓
           路由结果
```

**EvolutionMode 最高优先级路由**：
- 当 `EvolutionMode=true` 时，**绕过所有路由**，直接创建 EvolutionParrot
- **工作目录**: DivineSense 源代码根目录
- **产出物**: 强制 GitHub PR，需人工 Review 后合并
- **安全等级**: 高（需管理员权限 + 环境变量启用 + PR 审核）
- 仅限管理员使用
- 实现：`server/router/api/v1/ai/handler.go` 中的 `handleEvolutionMode()`

**GeekMode 优先路由**：
- 当 `GeekMode=true` 时（且 EvolutionMode=false），**绕过所有路由**，直接创建 GeekParrot
- **工作目录**: `~/.divinesense/claude/user_{id}`（用户沙箱）
- **产出物**: 用户可浏览/下载的代码产物
- **安全等级**: 中（沙箱隔离）
- 所有用户可用
- 实现：`server/router/api/v1/ai/handler.go` 中的 `handleGeekMode()`

**三层路由**（当 `router.Service` 已配置时）：
1. **规则匹配**（0ms）：常见模式的关键词匹配
2. **历史感知**（~10ms）：对话上下文匹配
3. **LLM 降级**（~400ms）：模糊输入的语义理解

**规则匹配**：
- 日程关键词：`日程`、`schedule`、`会议`、`meeting`、`提醒`、`remind`、时间词（`今天`、`明天`、`周X`、`点`、`分`）
- 笔记关键词：`笔记`、`memo`、`note`、`搜索`、`search`、`查找`、`find`、`写过`、`关于`
- Amazing 关键词：`综合`、`总结`、`summary`、`本周工作`、`周报`

**LLM 降级**：
- 模型：`Qwen/Qwen2.5-7B-Instruct`（通过 SiliconFlow）
- 最大 token：30（最小化响应）
- 严格 JSON Schema：`{"route": "memo|schedule|amazing", "confidence": 0.0-1.0}`

### 代理工具

**位置**：`ai/agent/tools/`

| 工具              | 文件             | 描述                    |
| :---------------- | :--------------- | :---------------------- |
| `memo_search`     | `memo_search.go` | 语义笔记搜索 + RRF 融合 |
| `schedule_add`    | `scheduler.go`   | 创建新日程              |
| `schedule_query`  | `scheduler.go`   | 查询现有日程            |
| `schedule_update` | `scheduler.go`   | 更新现有日程            |
| `find_free_time`  | `scheduler.go`   | 查找空闲时间段          |

### 日程代理 V2

**位置**：`ai/agent/scheduler_v2.go`

实现原生工具调用循环（现代 LLM 不需要 ReAct）：

**特性**：
- 带结构化参数的直接函数调用
- 默认 1 小时持续时间
- 自动冲突检测
- 时区感知日程安排

---

## CC Runner 异步架构 (Geek Mode 核心)

**规格文档**：[CC Runner 异步架构说明书](../specs/cc_runner_async_arch.md) (v1.2)

**概述**：Geek Mode 从一次性执行（One-shot）升级为**全双工持久化**（Full-Duplex Persistent）架构。

### 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (React)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │  EventBadge  │  │ ToolCallCard │  │  SessionSummaryPanel │ │
│  └──────────────┘  └──────────────┘  └──────────────────────┘ │
│                              │                                  │
│                        WebSocket (SSE)                         │
└──────────────────────────────┼──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                     Backend (Go)                                │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Session Mgr │◄─┤   Streamer  │◄─┤  DangerDetector      │  │
│  │  (30min)    │  │ (Bidirect)  │  │  (rm -rf, format)    │  │
│  └─────────────┘  └──────────────┘  └──────────────────────┘  │
└──────────────────────────────┼──────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────┐
│                   Claude Code CLI (OS Process)                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  --session-id <UUID> --output-format stream-json          │ │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │ │
│  │  │    CLI      │  │  In-Memory   │  │  Skills & MCP    │  │ │
│  │  │   Engine    │◄─┤   Context    │  │    Registry      │  │ │
│  │  └─────────────┘  └──────────────┘  └──────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### 核心组件

**位置**：`ai/agent/cc_runner/`

| 组件 | 文件 | 描述 |
|:-----|:-----|:-----|
| **SessionManager** | `session_manager.go` | 会话生命周期管理（30min 空闲超时） |
| **Streamer** | `streamer.go` | 双向流式转换（HTTP ⇄ CLI JSON Stream） |
| **DangerDetector** | `danger_detector.go` | 危险命令检测（rm -rf, mkfs, etc.） |
| **SessionStats** | `session_stats.go` | 实时指标收集（thinking, tokens, tools） |

### 会话映射模型

```
前端 ConversationID (数据库 ID)
         │
         ▼ UUID v5 定向哈希
         │
    SessionID (UUID)
         │
         ▼
Claude Code CLI Process
```

- **确定性映射**：`UUID v5(Namespace, "divinesense:conversation:{ID}")`
- **状态恢复**：CLI 自动从 `~/.claude/sessions/` 恢复上下文
- **物理隔离**：每个会话独立 OS 进程，互不干扰

### 交互协议

**Client → Server (WebSocket Events)**:

| Event | Payload | 描述 |
|:-----|:--------|:-----|
| `session.start` | `{config}` | 启动新会话 |
| `input.send` | `{text}` | 发送用户输入 |
| `session.stop` | `{}` | 强制停止 |

**Server → Client (Stream Events)**:

| Event | Meta | 描述 |
|:-----|:-----|:-----|
| `thinking` | — | 思考过程（增量） |
| `tool_use` | `{name, input, id}` | 工具调用 |
| `tool_result` | `{is_error}` | 工具结果 |
| `answer` | — | 最终回答（增量） |
| `error` | — | 系统级错误 |

### 安全与风控

- **Permission Bypass**: 使用 `--permission-mode bypassPermissions`
- **前端确认**: 对关键操作（如 `rm -rf`）进行 Regex 拦截
- **Git 恢复**: 强制在 Git 仓库内运行，确保可回滚
- **超时保护**: 30 分钟空闲自动 Kill

### API 端点

| RPC | 方法 | 描述 |
|:-----|:-----|:-----|
| `ChatService` | `StreamChat` | 流式聊天（SSE） |
| `ChatService` | `StopChat` | 停止会话（所有权验证） |

---

## AI 服务 (`ai/`)

### 服务概览

| 服务    | 包         | 描述                            |
| :------ | :--------- | :------------------------------ |
| Memory  | `memory/`  | 情景记忆 & 用户偏好             |
| Session | `session/` | 对话持久化（30 天保留）         |
| Router  | `router/`  | 三层意图分类 & 路由             |
| Cache   | `cache/`   | 带 TTL 的 LRU 缓存（查询结果）  |
| Metrics | `metrics/` | 代理 & 工具性能追踪（A/B 测试） |
| Vector  | `vector/`  | 多提供商 Embedding 服务         |

### 会话服务 (`ai/session/`)

为 AI 代理提供对话持久化：

**组件**：
- `store.go`：PostgreSQL 持久化 + 直写缓存（30min TTL）
- `recovery.go`：会话恢复工作流 + 滑动窗口（最多 20 条消息）
- `cleanup.go`：过期会话清理后台任务（默认：30 天）

**数据库**：`conversation_context` 表（JSONB 存储）

### 上下文构建器 (`ai/context/`)

组装 LLM 上下文，智能分配 token 预算：

```
Token 预算分配（带检索）
┌─────────────────────────────────────────┐
│ System Prompt      │ 500 tokens（固定） │
│ User Preferences   │ 10%                │
│ Short-term Memory  │ 40%                │
│ Long-term Memory   │ 15%                │
│ Retrieval Results  │ 45%                │
└─────────────────────────────────────────┘
```

---

## 检索系统 (`ai/core/retrieval/`)

### AdaptiveRetriever

混合 BM25 + 向量搜索 + 智能融合：

| 策略             | 描述                            |
| :--------------- | :------------------------------ |
| `BM25Only`       | 仅关键词搜索（快，低质量）      |
| `SemanticOnly`   | 仅向量搜索（慢，语义）          |
| `HybridStandard` | BM25 + 向量 + RRF 融合（平衡）  |
| `FullPipeline`   | 混合 + 重排器（最高质量，最慢） |

### RRF 融合

用于合并 BM25 和向量结果的倒数排名融合：
```
score = Σ weight_i / (60 + rank_i)
```

### 重排器

BAAI/bge-reranker-v2-m3 用于结果精炼（可通过策略配置）。

---

## 前端架构 (`web/src/`)

### 页面组件

| 路径           | 组件              | 布局           | 用途                     |
| :------------- | :---------------- | :------------- | :----------------------- |
| `/`            | `Home.tsx`        | MainLayout     | 主时间线 + 笔记编辑器    |
| `/explore`     | `Explore.tsx`     | MainLayout     | 搜索和探索内容           |
| `/archived`    | `Archived.tsx`    | MainLayout     | 已归档笔记               |
| `/chat`        | `AIChat.tsx`      | AIChatLayout   | AI 聊天界面 + 自动路由   |
| `/schedule`    | `Schedule.tsx`    | ScheduleLayout | 日历视图（FullCalendar） |
| `/review`      | `Review.tsx`      | MainLayout     | 每日回顾                 |
| `/setting`     | `Setting.tsx`     | MainLayout     | 用户设置                 |
| `/u/:username` | `UserProfile.tsx` | MainLayout     | 公开用户资料             |

### 布局层级

```
RootLayout（全局导航 + 认证）
    │
    ├── MainLayout（可折叠侧边栏：MemoExplorer）
    │   └── /, /explore, /archived, /u/:username
    │
    ├── AIChatLayout（固定侧边栏：AIChatSidebar）
    │   └── /chat
    │
    └── ScheduleLayout（固定侧边栏：ScheduleCalendar）
        └── /schedule
```

### 静态资源优化 (Static Asset Optimization)

为了在单二进制分发中保持极致的 Web 性能，`FrontendService` 实现了以下优化策略：

| 策略                 | 实现细节                                 | 目标                                      |
| :------------------- | :--------------------------------------- | :---------------------------------------- |
| **Gzip 压缩**        | `middleware.Gzip(Level: 5)`              | 减少二进制嵌入产物的传输大小（约 70%）    |
| **强缓存 (Vite)**    | `/assets/*` 匹配 `immutable, max-age=1y` | 针对 Vite 哈希资源实现“零请求”重复访问    |
| **入口防缓存**       | `index.html` 强制 `no-cache, no-store`   | 确保版本迭代后用户立刻获取最新 JS 引用    |
| **Geek 工作区 Host** | `/file/geek/:userID/*` 实时 Host         | 极客模式产生的网页/产物可在浏览器实时预览 |
| **安全加固**         | `X-Content-Type-Options: nosniff`        | 增强针对嵌入式静态资源的安全防御          |

---

## 数据流

### AI 聊天流程

```
前端（AIChat.tsx）
    │（WebSocket / SSE）
    ↓
后端（ai_service_chat.go）
    │
    ↓ GeekMode?
    │   Yes → GeekParrot（Claude Code CLI，零 LLM）
    │   No  ↓ ChatRouter.Route()
    │       → 规则匹配（0ms）
    │       → 历史感知（~10ms）
    │       → LLM 降级（~400ms）
    ↓
代理执行
    │   → GeekParrot（Claude Code CLI）
    │   → MemoParrot（memo_search 工具）
    │   → ScheduleParrotV2（scheduler 工具）
    │   → AmazingParrot（并发工具）
    ↓
响应流式传输
    │   → 事件类型：thinking、tool_use、tool_result、answer
    ↓
前端 UI 更新
```

---

## AI 数据库架构（PostgreSQL）

### 核心表

| 表名                   | 用途                                      |
| :--------------------- | :---------------------------------------- |
| `memo_embedding`       | 向量嵌入（1024 维）用于语义搜索           |
| `conversation_context` | 会话持久化（30 天保留）                   |
| `episodic_memory`      | 长期用户记忆和学习                        |
| `user_preferences`     | 用户沟通偏好                              |
| `agent_metrics`        | A/B 测试指标（prompt 版本、延迟、成功率） |

---

## 环境配置

### 关键变量

```bash
# 数据库
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable

# AI
DIVINESENSE_AI_ENABLED=true
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3
DIVINESENSE_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
DIVINESENSE_AI_LLM_PROVIDER=deepseek
DIVINESENSE_AI_LLM_MODEL=deepseek-chat
DIVINESENSE_AI_DEEPSEEK_API_KEY=your_key
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
DIVINESENSE_AI_OPENAI_BASE_URL=https://api.siliconflow.cn/v1
```
