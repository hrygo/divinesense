# 架构文档

> **保鲜状态**: ✅ 已更新 (2026-02-10) | **最后检查**: v0.97.0 (智能路由 + 性能优化)

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
| AI     | **DeepSeek**（对话 LLM），**SiliconFlow**（Embedding、意图分类、Reranker）                              |

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
│   ├── agent/           #   Parrot 代理（UniversalParrot 配置驱动系统 + GeekParrot、EvolutionParrot）
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
│   ├── tags/            #   标签建议
│   ├── duplicate/       #   重复检测
│   ├── habit/           #   习惯学习（日程增强）
│   ├── graph/           #   知识图谱
│   ├── schedule/        #   日程 AI
│   ├── aitime/          #   AI 时间解析
│   ├── timeout/         #   超时处理
│   ├── review/          #   审查服务（间隔重复）
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
│   │   ├── locales/     # i18n 翻译（en、zh-Hans）
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
   - **对话 LLM**：DeepSeek (`deepseek-chat`)
   - **Embedding**：SiliconFlow (`BAAI/bge-m3`, 1024 维)
   - **意图分类**：SiliconFlow (`Qwen/Qwen2.5-7B-Instruct`)
   - **Reranker**：SiliconFlow (`BAAI/bge-reranker-v2-m3`)
   - 所有 AI 功能可选（由 `DIVINESENSE_AI_ENABLED` 控制）
   - 包含从 `server/ai/` 和 `server/retrieval/` 迁移的嵌入和检索功能

3. **插件系统** (`plugin/`)：
   - 不再包含 AI 功能（已提升为一级模块）
   - 包含：调度器、存储适配器、身份提供商、Markdown、OCR、Webhook、聊天应用接入（Chat Apps）等

### 聊天应用集成 (Chat Apps)

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **最后检查**: v0.91.0

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

> **保鲜状态**: ✅ 已验证 (2026-02-10) | **覆盖范围**: `proto/api/v1/*.proto` | **最后检查**: v0.97.0

| 服务 | Proto 文件 | 描述 |
|:-----|:-----------|:-----|
| **ActivityService** | `activity_service.proto` | 用户活动记录 |
| **AttachmentService** | `attachment_service.proto` | 附件管理 |
| **AuthService** | `auth_service.proto` | 认证授权 |
| **AIService** | `ai_service.proto` | AI 聊天、嵌入、检索（含 Unified Block Model） |
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

> **保鲜状态**: ✅ 2026-02-07 | **智能检查策略**

DivineSense 使用 **智能 pre-commit + pre-push** hooks，根据修改内容自动选择检查项：

### 检查策略矩阵

| 修改类型 | pre-commit (~2-10s) | pre-push (~10-60s) |
|:---------|:---------------------|:-------------------|
| **仅后端** | `go fmt` + `go vet` | `go mod tidy` + `golangci-lint` + `go test` |
| **仅前端** | `pnpm lint:fix` | `pnpm lint` + `pnpm build` |
| **仅文档** | 跳过 | 跳过 |
| **混合** | 按需检查 | 按需检查 |

### 文件分类规则

| 分类 | 匹配模式 |
|:-----|:---------|
| 后端 | `*.go`, `go.mod`, `go.sum` |
| 前端 | `web/**`, `server/router/frontend/**` |
| 文档 | `docs/**`, `*.md` (不匹配上述) |

**注意**：Proto 文件变更会被归类为"后端"（因为修改后需要重新生成 Go 代码）。

### 安装与使用

```bash
# 安装 hooks
make install-hooks

# 本地 CI 检查（完整检查，不分类）
make ci-check
make ci-backend
make ci-frontend

# 跳过检查
git commit --no-verify -m "WIP"
git push --no-verify
```

### 示例输出

```
🔍 Pre-commit checks...

📦 Backend changes detected:
  → go.mod/go.sum tidy check...
    ✓ go.mod/go.sum tidy
  → go fmt...
    ✓ go fmt
  → go vet...
    ✓ go vet

🎨 Frontend changes detected:
  → pnpm lint:fix...
    ✓ pnpm lint:fix

✅ Pre-commit checks passed!
```

> **详细规范**：参见 [Git 工作流](../../.claude/rules/git-workflow.md)

---

## Parrot 代理架构

### 代理类型 (`ai/agent/`)

|  AgentType  | 鹦鹉名称 | 配置/文件              | 中文名 | 描述                             |
| :---------: | :------- | :--------------------- | :----- | :------------------------------- |
|   `AUTO`    | —        | —                      | —      | 路由标记（非鹦鹉），由后端三层路由决定使用哪只鹦鹉 |
|   `MEMO`    | 灰灰     | `config/parrots/memo.yaml` | 灰灰   | 笔记搜索和检索专家               |
| `SCHEDULE`  | 时巧     | `config/parrots/schedule.yaml` | 时巧 | 日程创建和管理                   |
|  `AMAZING`  | 折衷     | `config/parrots/amazing.yaml` | 折衷 | 综合助理（笔记 + 日程）          |
|   `GEEK`    | 极客     | `geek_parrot.go`       | 极客   | Claude Code CLI 通信层（零 LLM） |
| `EVOLUTION` | 进化     | `evolution_parrot.go`  | 进化   | 自我进化能力（源代码修改）       |

**说明**：
- **鹦鹉共五只**：MEMO、SCHEDULE、AMAZING（配置驱动）、GEEK、EVOLUTION（代码实现）
- **AUTO 不是鹦鹉**：它是前端发送给后端的特殊标记，表示"请后端路由系统决定使用哪只鹦鹉"
- 当 `AgentType == AUTO` 时，后端触发三层路由（规则匹配 → 历史感知 → LLM 降级）

### UniversalParrot 架构

> **实现状态**: ✅ 完成 (v0.97.0) | **位置**: `ai/agent/universal/`

**概述**：UniversalParrot 是配置驱动的通用代理系统，三只核心鹦鹉（MEMO、SCHEDULE、AMAZING）通过 YAML 配置文件定义，无需编写代码。

**配置目录**：`config/parrots/`

| 配置文件 | 代理名称 | 执行策略 |
|:--------|:--------|:---------|
| `memo.yaml` | MemoParrot | ReAct 循环（`react`） |
| `schedule.yaml` | ScheduleParrot | 原生工具调用（`direct`） |
| `amazing.yaml` | AmazingParrot | 两阶段规划 + 并发执行（`planning`） |

**核心组件**：

| 组件 | 文件 | 描述 |
|:-----|:-----|:-----|
| **UniversalParrot** | `universal_parrot.go` | 配置驱动的通用代理实现 |
| **ParrotFactory** | `parrot_factory.go` | 从配置创建代理的工厂 |
| **ParrotConfig** | `parrot_config.go` | 配置加载和验证 |
| **ExecutionStrategy** | `*_executor.go` | 执行策略接口（Direct/ReAct/Planning） |
| **ToolRegistry** | `registry/tool_registry.go` | 工具注册表（动态工具发现） |

**执行策略**：

| 策略 | 文件 | 特点 | 适用场景 |
|:-----|:-----|:-----|:---------|
| **DirectExecutor** | `direct_executor.go` | 原生 LLM 工具调用 | 简单工具调用 |
| **ReActExecutor** | `react_executor.go` | 思考-行动循环 | 复杂多步任务 |
| **PlanningExecutor** | `planning_executor.go` | 两阶段规划 + 并发 | 多工具协作 |

### 代理路由器

**位置**：`ai/agent/chat_router.go` + `ai/router/service.go`

ChatRouter 实现**四层**意图分类系统：

```
用户输入 → EvolutionMode? ─Yes→ EvolutionParrot（自我进化）
                  │
                  No
                  ↓
           GeekMode? ─Yes→ GeekParrot（Claude Code CLI）
                  │
                  No
                  ↓
           AgentType == AUTO?
                  │
           Yes ────────┴──── No（直接使用指定的鹦鹉）
                  ↓
           ChatRouter.Route()
                  ↓
           router.Service.ClassifyIntent()
                  ↓
    ┌─────────────────────────────────────┐
    │  Layer 0: Cache (LRU, 0ms)          │  → Hit? 返回缓存结果
    │  Layer 1: RuleMatcher (0ms)        │  → Match? 返回
    │  Layer 2: HistoryMatcher (~10ms)   │  → Match? 返回
    │  Layer 3: LLM Classifier (~400ms)   │  → 返回 JSON {intent, confidence}
    └─────────────────────────────────────┘
                  ↓
           路由结果（MEMO/SCHEDULE/AMAZING）
```

**Layer 0: Cache (LRU)**
- 容量：500 条
- TTL：规则匹配结果 5 分钟，LLM 结果 30 分钟
- 延迟：~0ms

**Layer 1: RuleMatcher (关键词匹配)**
- 时间词权重：2（今天、明天、后天、下周、本周、上午、下午、晚上、点）
- 核心关键词权重：2（日程、安排、会议、提醒、预约、开会）
- 快速路径：时间词 + 查询词 → `schedule_query` (如："明天有什么事情要做")
- 延迟：~0ms

**Layer 2: HistoryMatcher (对话历史)**
- 基于用户历史对话的向量相似度匹配
- 存储表：`conversation_context`
- 延迟：~10ms

**Layer 3: LLM Classifier**
- Provider: SiliconFlow
- Model: `Qwen/Qwen2.5-7B-Instruct`
- Token: 50, Temperature: 0
- 输出格式：JSON Schema `{intent, confidence}`
- 延迟：~400ms

**智能路由反馈（v0.97.0 新增）**：
- 收集用户对路由结果的反馈
- 存储表：`router_feedback`（predicted_intent, actual_intent, confidence）
- 用于优化关键词权重和 LLM 分类器
- 支持 A/B 测试不同路由策略
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

### AI 模型策略总览

| 功能 | 提供商 | 模型 | 用途 |
|:-----|:-------|:-----|:-----|
| **对话 LLM** | DeepSeek | `deepseek-chat` | 主对话生成 |
| **向量 Embedding** | SiliconFlow | `BAAI/bge-m3` | 语义搜索（1024维） |
| **意图分类** | SiliconFlow | `Qwen/Qwen2.5-7B-Instruct` | 路由意图分类 |
| **重排 Rerank** | SiliconFlow | `BAAI/bge-reranker-v2-m3` | 检索结果精炼 |

**策略说明**：
- **意图分类独立模型**：使用轻量级 Qwen2.5-7B-Instruct（而非主对话 LLM），实现快速、低成本的分类
- **成本优化**：意图分类 Token 限制为 50，Temperature 0（确定性输出）
- **输出格式**：JSON Schema `{intent, confidence}` 确保结构化响应

### DeepSeek 上下文缓存

> **保鲜状态**: ✅ 已验证 (2026-02-07)

DeepSeek API 提供自动上下文缓存（Prompt Caching），降低多轮对话成本。

**缓存机制**：
- **缓存粒度**：64 token 块
- **工作原理**：相同会话前缀的后续请求自动命中缓存
- **命中识别**：API 响应返回 `prompt_cache_hit_tokens` 字段

**数据流**：
```
DeepSeek API Response
    ↓ prompt_cache_hit_tokens
go-openai 库映射
    ↓ PromptTokensDetails.CachedTokens
ai/llm.go:206
    ↓ CacheReadTokens: resp.Usage.PromptTokensDetails.CachedTokens
LLMCallStats.CacheReadTokens
    ↓ SessionStats.CacheReadTokens
数据库 ai_block.token_usage.cache_read_tokens
```

**成本优化效果**：
| 轮次 | Prompt Tokens | Cache Hit | 缓存率 |
|:-----|:--------------|:---------|:-------|
| 第1轮 | ~5000 | 0 | 0% (冷启动) |
| 第2轮 | ~6000 | ~5000 | ~83% |
| 第3轮 | ~8000 | ~5760 | ~72% |

**最佳实践**：
- 保持系统提示词稳定，提升缓存命中率
- 避免频繁修改会话前缀
- 监控 `cache_write_tokens` 与 `cache_read_tokens` 比例

### 代理工具

**位置**：`ai/agent/tools/`

| 工具              | 文件             | 描述                    |
| :---------------- | :--------------- | :---------------------- |
| `memo_search`     | `memo_search.go` | 语义笔记搜索 + RRF 融合 |
| `schedule_add`    | `scheduler.go`   | 创建新日程              |
| `schedule_query`  | `scheduler.go`   | 查询现有日程            |
| `schedule_update` | `scheduler.go`   | 更新现有日程            |
| `find_free_time`  | `scheduler.go`   | 查找空闲时间段          |

### 流式工具调用

> **实现状态**: ✅ 完成 (v0.97.0)

**概述**：所有执行策略支持流式事件回调，前端可实时显示工具执行进度。

**事件类型**：

| 事件 | 描述 | 元数据 |
|:-----|:-----|:-------|
| `thinking` | 思考中 | tokens, duration |
| `tool_use` | 工具调用 | toolName, input, toolId |
| `tool_result` | 工具结果 | toolName, status, error |
| `phase_change` | 阶段切换 | currentStep, totalSteps |
| `answer` | 最终回答 | — |

**前端组件**：
- `EventBadge` - 事件类型徽章
- `CompactToolCall` - 轻量级工具调用卡片
- `UnifiedMessageBlock` - 统一消息块（集成工具展示）

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
| Filter  | `filter/`  | 敏感信息过滤器（<1ms 响应）     |
| Preload | `preload/` | 预测性缓存预加载                |
| Tracing | `tracing/` | 分布式链路追踪                  |

### 新增 AI 模块 (v0.97.0)

| 模块 | 功能 | 性能指标 |
|:-----|:-----|:---------|
| **ai/filter/** | 敏感信息过滤（手机号、身份证、邮箱、银行卡、IP） | <1ms 响应时间 |
| **ai/preload/** | 基于用户行为模式的智能预加载 | 命中率 >60% |
| **ai/stats/** | 告警持久化、指标存储 | 实时聚合 |
| **ai/tracing/** | 分布式追踪（OpenTelemetry 兼容） | <5% 开销 |
| **ai/agent/registry/** | 动态工具发现、执行策略注册 | 热加载 |

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
| `/review`      | `Review.tsx`      | GeneralLayout  | 每日回顾                 |
| `/setting`     | `Setting.tsx`     | GeneralLayout  | 用户设置                 |
| `/u/:username` | `UserProfile.tsx` | MemoLayout     | 公开用户资料             |
| `/memos/:uid`  | `MemoDetail.tsx`  | GeneralLayout  | 笔记详情页               |
| `/m/:uid`      | `MemoDetailRedirect` | GeneralLayout | 笔记详情重定向           |
| `/403`         | `PermissionDenied.tsx` | GeneralLayout | 权限拒绝                 |
| `/404`         | `NotFound.tsx`    | GeneralLayout  | 404 页面                 |

### 布局层级

```
RootLayout（全局导航 + 认证）
    │
    ├── MemoLayout（可折叠侧边栏：MemoExplorer）
    │   └── /home, /explore, /archived, /u/:username
    │
    ├── AIChatLayout（固定侧边栏：AIChatSidebar）
    │   └── /chat
    │
    ├── ScheduleLayout（固定侧边栏：ScheduleCalendar）
    │   └── /schedule
    │
    └── GeneralLayout（无侧边栏，全宽内容）
        └── /knowledge-graph, /inbox, /attachments, /setting, /memos/:uid, /review, /403, /404
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

## Unified Block Model (统一块模型)

> **实现状态**: ✅ 完成 (Issue #71) | **版本**: v0.97.0

**概述**：Unified Block Model 是一种新的 AI 聊天对话持久化方案，替代原有的 ChatItem[] 结构。

### 核心概念

**Block (块)**：一个聊天轮次的完整数据单元
```
┌─────────────────────────────────────────────────────────┐
│                      AIBlock                          │
├─────────────────────────────────────────────────────────┤
│  id, uid, conversation_id, round_number               │
│  mode: normal | geek | evolution                        │
│  user_inputs[]       // 用户输入（支持多轮补充）       │
│  assistant_content   // AI 回复内容                    │
│  event_stream[]      // 流式事件（thinking/tool_use）  │
│  session_stats       // 会话统计（tokens/cost）         │
│  status: pending | streaming | completed | error       │
└─────────────────────────────────────────────────────────┘
```

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│                  Frontend (React)                        │
│  ChatMessages ──▶ AIBlock[] ──▶ UnifiedMessageBlock    │
└────────────────────┬────────────────────────────────────┘
                     │ gRPC Stream
┌────────────────────▼────────────────────────────────────┐
│                 Backend (Go)                             │
│  Chat Handler ──▶ BlockManager ──▶ Store.AIBlock       │
│                      │                                 │
│                      ├─ CreateBlockForChat()           │
│                      ├─ AppendEvent() (async)         │
│                      ├─ CompleteBlock()               │
│                      └─ MarkBlockError()              │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│              PostgreSQL (ai_block 表)                   │
│  - id, uid, conversation_id, round_number              │
│  - mode (normal/geek/evolution)                         │
│  - user_inputs[], event_stream[], session_stats         │
│  - status (pending/streaming/completed/error)           │
└─────────────────────────────────────────────────────────┘
```

### 核心组件

| 组件 | 位置 | 描述 |
|:-----|:-----|:-----|
| **BlockManager** | `server/router/api/v1/ai/block_manager.go` | Block 生命周期管理 |
| **ChatMessages** | `web/src/components/AIChat/ChatMessages.tsx` | 前端 Block 渲染 |
| **AIChatContext** | `web/src/contexts/AIChatContext.tsx` | Block 状态管理 |
| **Block Queries** | `web/src/hooks/useBlockQueries.ts` | React Query 集成 |

### 数据库表：`ai_block`

| 字段 | 类型 | 描述 |
|:-----|:-----|:-----|
| `id` | BIGINT | 主键 |
| `uid` | VARCHAR(64) | 唯一标识符 |
| `conversation_id` | INTEGER | 所属会话（外键） |
| `round_number` | INTEGER | 轮次号（会话内自增） |
| `mode` | TEXT | 模式：normal/geek/evolution |
| `user_inputs` | JSONB | 用户输入数组 |
| `assistant_content` | TEXT | AI 回复内容 |
| `event_stream` | JSONB | 流式事件数组 |
| `session_stats` | JSONB | 会话统计信息 |
| `cc_session_id` | VARCHAR(64) | CC Runner 会话 ID |
| `status` | TEXT | 状态：pending/streaming/completed/error |
| `metadata` | JSONB | 元数据 |
| `created_ts` | BIGINT | 创建时间戳 |
| `updated_ts` | BIGINT | 更新时间戳 |

**索引**：
- `idx_ai_block_conversation`：`(conversation_id, round_number)`
- `idx_ai_block_status`：`(status)`
- `idx_ai_block_cc_session`：`(cc_session_id)`

**触发器**：
- `ai_block_round_number_trigger`：自动设置 `round_number`

### Block 状态流转

```
pending ──▶ streaming ──▶ completed
    │             │
    │             └──▶ error
    └──▶ error
```

### BlockMode 映射

| BlockMode | ParrotAgentType | 用途 |
|:----------|:----------------|:-----|
| `normal` | `AUTO` | 普通模式，由后端三层路由决定使用哪只鹦鹉（MEMO/SCHEDULE/AMAZING） |
| `geek` | `GEEK` | 极客模式，Claude Code CLI 代码执行 |
| `evolution` | `EVOLUTION` | 进化模式，系统自我进化 |

### API 端点

| RPC | 方法 | 描述 |
|:-----|:-----|:-----|
| `AIService` | `ListBlocks` | 列出会话的所有 Blocks |
| `AIService` | `GetBlock` | 获取单个 Block 详情 |
| `AIService` | `CreateBlock` | 创建新 Block |
| `AIService` | `UpdateBlock` | 更新 Block |
| `AIService` | `DeleteBlock` | 删除 Block |
| `AIService` | `AppendEvent` | 追加事件到流 |
| `AIService` | `AppendUserInput` | 追加用户输入 |

**详细规格**：[Unified Block Model 规格](../specs/block-design/unified-block-model.md)

**界面设计**：[AI Chat 界面架构](AI_CHAT_INTERFACE.md) - 包含完整的 UI 布局、组件层级和交互设计

---

## AI 数据库架构（PostgreSQL）

### 核心表

| 表名                   | 用途                                      | 版本    |
| :--------------------- | :---------------------------------------- | :------ |
| `ai_block`            | **统一块模型**：AI 聊天对话持久化 (#71)     | v0.97.0 |
| `ai_conversation`     | AI 对话会话                              | v0.97.0 |
| `memo_embedding`       | 向量嵌入（1024 维）用于语义搜索           | v0.97.0 |
| `conversation_context` | 会话持久化（多渠道支持）                  | v0.97.0 |
| `episodic_memory`      | 长期用户记忆和学习                        | -       |
| `user_preferences`     | 用户沟通偏好                              | -       |

### 增强功能表

| 表名                   | 用途                                      | 版本    |
| :--------------------- | :---------------------------------------- | :------ |
| `agent_session_stats`  | 会话统计（成本追踪）                       | v0.97.0 |
| `user_cost_settings`   | 用户成本预算设置                          | v0.97.0 |
| `agent_security_audit` | 安全审计（高风险操作记录）                 | v0.97.0 |

### 智能路由表（v0.97.0 新增）

| 表名                   | 用途                                      | 功能     |
| :--------------------- | :---------------------------------------- | :------- |
| `router_feedback`      | 路由反馈收集                              | 意图分类优化 |
| `router_weight`        | 动态权重存储                              | 个性化路由 |

---

## 环境配置

### 关键变量

```bash
# 数据库
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable

# AI 开关
DIVINESENSE_AI_ENABLED=true

# 对话 LLM (DeepSeek)
DIVINESENSE_AI_LLM_PROVIDER=deepseek
DIVINESENSE_AI_LLM_MODEL=deepseek-chat
DIVINESENSE_AI_DEEPSEEK_API_KEY=your_key

# 向量 Embedding (SiliconFlow)
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
DIVINESENSE_AI_OPENAI_BASE_URL=https://api.siliconflow.cn/v1

# 意图分类 (SiliconFlow + Qwen)
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
DIVINESENSE_AI_OPENAI_BASE_URL=https://api.siliconflow.cn/v1

# 重排 Reranker (SiliconFlow)
DIVINESENSE_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
DIVINESENSE_AI_OPENAI_BASE_URL=https://api.siliconflow.cn/v1
```
