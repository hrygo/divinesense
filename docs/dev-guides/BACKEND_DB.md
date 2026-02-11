# 后端与数据库指南

> **保鲜状态**: ✅ 已更新 (2026-02-11) | **最后检查**: v0.97.0 (Anthropic 默认 LLM)

## 数据库支持策略

### PostgreSQL（生产环境 - 完整支持）
- **状态**：生产环境主数据库
- **AI 功能**：完整支持（pgvector、混合搜索、重排、会话记忆）
- **推荐用途**：所有生产部署
- **维护状态**：积极维护和测试
- **端口**：25432（开发环境）
- **版本**：PostgreSQL 16+

### SQLite（开发环境 - 向量搜索支持 v0.97.0）

- **状态**：开发环境
- **AI 功能**：**部分支持** —— 仅向量搜索（语义检索）
- **向量搜索**：sqlite-vec 扩展（2026年1月新增）
- **全文搜索**：FTS5（可用）或 LIKE fallback
- **推荐用途**：本地开发、语义搜索功能测试
- **限制**：
  - **不**支持 AI 对话持久化（ai_block、ai_conversation）—— 需使用 PostgreSQL
  - **不**支持情景记忆（episodic_memory）—— 需使用 PostgreSQL
  - **不**支持用户偏好（user_preferences）—— 需使用 PostgreSQL
  - **不**支持智能路由（router_feedback、router_weight）—— 需使用 PostgreSQL
  - 向量搜索性能不如 PostgreSQL + pgvector
  - 大数据集（>10k 向量）性能下降
- **维护状态**：向量搜索功能已启用
- **注意**：完整 AI 功能（对话、记忆、偏好、路由）必须使用 PostgreSQL

### MySQL（已移除）
- **状态**：**不支持** —— 已移除所有 MySQL 支持
- **迁移**：生产环境使用 PostgreSQL，开发环境使用 SQLite
- **原因**：由于缺乏 AI 功能和维护负担，移除了 MySQL 支持

---

## 部署模式

### 开发环境（Docker Compose）
- **用途**：本地开发和测试
- **组件**：PostgreSQL 容器 + 后端开发服务器 + 前端开发服务器
- **命令**：`make start`
- **端口**：后端 28081，前端 25173，PostgreSQL 25432

### 生产环境（Docker 模式）
- **用途**：单服务器生产部署 + Docker 容器
- **组件**：PostgreSQL 容器 + DivineSense 容器
- **安装**：`deploy/aliyun/install.sh --mode=docker`（默认）
- **管理**：`cd /opt/divinesense && ./deploy.sh <command>`

### 生产环境（二进制模式）—— Geek Mode 推荐
- **用途**：原生 systemd 服务生产部署
- **组件**：Systemd 服务 + PostgreSQL（Docker 或系统级）
- **安装**：`deploy/aliyun/install.sh --mode=binary`
- **管理**：`/opt/divinesense/deploy-binary.sh <command>`
- **优势**：
  - 原生 Claude Code CLI 集成（Geek Mode）
  - 更快的启动速度，更低的资源开销
  - 更便捷的升级（二进制替换 + 校验和验证）
- **文档**：[部署指南](../deployment/BINARY_DEPLOYMENT.md)

**快速安装**：
```bash
# Docker 模式（默认）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash

# 二进制模式（Geek Mode 推荐）
curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | sudo bash -s -- --mode=binary
```

**目录结构（二进制模式）**：
```
/opt/divinesense/          # 安装根目录
├── bin/                   # 二进制
│   └── divinesense
├── data/                  # 工作目录（Geek Mode）
├── logs/                  # 日志
├── backups/               # 数据库备份
├── docker/                # PostgreSQL Docker 配置（可选）
│   ├── postgres.yml
│   └── .env
└── deploy-binary.sh      # 管理脚本

/etc/divinesense/          # 配置
└── config                 # 环境变量
└── .db_password          # 数据库密码（600 权限）

/etc/systemd/system/       # 服务
└── divinesense.service
```

---

## 后端开发

### 技术栈
- **语言**：Go 1.25+
- **框架**：Echo（HTTP）+ Connect RPC（gRPC-HTTP 转码）
- **日志**：`log/slog`
- **配置**：通过 `.env` 文件的环境变量

### API 设计模式

1. **协议优先**：修改 `proto/api/` 或 `proto/store/` 中的 `.proto` 文件
2. **生成代码**：运行 `make generate`（如果需要修改 proto）
3. **实现处理器**：在 `server/router/api/v1/` 添加实现
4. **存储层**：在 `store/` 添加接口 → 在 `store/db/{driver}/` 实现 → 添加迁移

### 命名约定

| 类型 | 约定 | 示例 |
|:-----|:-----|:-----|
| Go 文件 | `snake_case.go` | `memo_embedding.go` |
| 测试文件 | `*_test.go` | `universal_parrot_test.go` |
| Go 包 | 简单小写 | `ai`（例如 `ai/agent`，非 `ai_service`） |
| 脚本 | `kebab-case.sh` | `dev.sh` |
| 常量 | `PascalCase` | `DefaultCacheTTL` |

---

## 常用开发命令

### 服务控制
```bash
make start              # 启动所有服务（PostgreSQL + 后端 + 前端）
make stop               # 停止所有服务
make status             # 检查服务状态
make logs               # 查看所有日志
make logs-backend       # 查看后端日志
make logs-follow-backend # 实时后端日志
make run                # 仅启动后端（需先启动数据库）
make web                # 仅启动前端
```

### Docker（PostgreSQL）
```bash
make docker-up          # 启动数据库容器
make docker-down        # 停止数据库容器
make db-connect         # 连接到 PG shell
make db-reset           # 重置数据库模式（破坏性！）
make db-vector          # 验证 pgvector 扩展
```

### 测试
```bash
make test               # 运行所有测试
make test-ai            # 运行 AI 相关测试
make test-embedding     # 运行 embedding 测试
make test-runner        # 运行后台运行器测试
go test ./path/to/package -v  # 运行特定包测试
```

### 构建
```bash
make build              # 构建后端二进制
make build-web          # 构建前端静态资源
make build-all          # 同时构建前端和后端
```

### 依赖
```bash
make deps-all           # 安装所有依赖（后端、前端、AI）
make deps               # 仅安装后端依赖
make deps-web           # 仅安装前端依赖
make deps-ai            # 仅安装 AI 依赖
```

### 本地 CI 检查

```bash
make ci-check           # 模拟完整 CI 检查（与 GitHub Actions 一致）
make ci-backend         # 后端检查（golangci-lint + test）
make ci-frontend        # 前端检查（lint + build）
make lint               # 仅 golangci-lint
make vet                # 仅 go vet
```

---

## Git Hooks

DivineSense 使用 **pre-commit + pre-push** hooks 确保代码质量。

> **详细规范**：参见 [Git 工作流](../../.claude/rules/git-workflow.md)

---

## 配置（.env）

### 环境变量

**数据库**：
```bash
DIVINESENSE_DRIVER=postgres
DIVINESENSE_DSN=postgres://divinesense:divinesense@localhost:25432/divinesense?sslmode=disable
```

**AI（推荐 SiliconFlow + 智谱 Z.AI Claude）**：
```bash
DIVINESENSE_AI_ENABLED=true
DIVINESENSE_AI_EMBEDDING_PROVIDER=siliconflow
DIVINESENSE_AI_EMBEDDING_MODEL=BAAI/bge-m3
DIVINESENSE_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3
DIVINESENSE_AI_LLM_PROVIDER=anthropic
DIVINESENSE_AI_LLM_MODEL=opus
DIVINESENSE_AI_ANTHROPIC_API_KEY=your_key
DIVINESENSE_AI_ANTHROPIC_BASE_URL=https://open.bigmodel.cn/api/anthropic
DIVINESENSE_AI_SILICONFLOW_API_KEY=your_key
```

**Geek Mode（可选 —— Claude Code CLI 集成）**：
```bash
# 为代码相关任务启用 Geek Mode
DIVINESENSE_CLAUDE_CODE_ENABLED=true
```

**配置优先级**：
1. 系统环境变量（支持 direnv）
2. `.env` 文件
3. 代码默认值

---

## 核心组件

### AI 代理系统

所有 AI 聊天逻辑通过 `ai/agent/` 中的 `ChatRouter` 路由：

| 代理 | 配置文件 | 用途 | 工具 |
|:-----|:---------|:-----|:-----|
| **MemoParrot** | `config/parrots/memo.yaml` | 笔记搜索和检索 | `memo_search` |
| **ScheduleParrot** | `config/parrots/schedule.yaml` | 日程管理 | `schedule_add`、`schedule_query`、`schedule_update`、`find_free_time` |
| **AmazingParrot** | `config/parrots/amazing.yaml` | 组合笔记 + 日程 | 所有工具 + 并发执行 |

> **实现**: 所有三种代理由 `ai/agent/universal/` 中的 **UniversalParrot** 配置驱动系统实现。

**聊天路由流程**（`chat_router.go`）：
```
输入 → 规则匹配（0ms）→ 历史感知（~10ms）→ 权重匹配（~5ms）→ LLM 分类（~400ms）
       ↓                ↓                  ↓                    ↓
    关键词         对话上下文          个性化路由          语义理解
```

**五层路由系统（v0.97.0）**：
- Layer 0: Cache (LRU, 0ms)
- Layer 1: RuleMatcher (关键词匹配, 0ms)
- Layer 2: HistoryMatcher (对话历史, ~10ms)
- Layer 3: WeightMatcher (动态权重, ~5ms) — 新增
- Layer 4: LLM Classifier (~400ms)

### 查询引擎

位于 `server/queryengine/`：
- 意图检测和路由
- 基于时间关键词的智能查询策略
- 自适应检索选择

### 检索系统

位于 `ai/core/retrieval/`：
- 混合 BM25 + 向量搜索（`AdaptiveRetriever`）
- 重排管道
- 查询结果的 LRU 缓存层

---

## AI 数据库架构（PostgreSQL）

### 核心 AI 表

| 表名 | 用途 | 版本 | 关键列 |
|:-----|:-----|:-----|:-----|
| `ai_conversation` | AI 对话会话 | v0.97.0 | `id`, `creator_id`, `title` |
| `ai_block` | **统一块模型**：对话持久化 | v0.97.0 | `conversation_id`, `round_number`, `mode` |
| `memo_embedding` | 语义搜索的向量嵌入 | v0.97.0 | `memo_id`、`embedding`（vector(1024)） |
| `conversation_context` | 会话上下文（多渠道） | v0.97.0 | `session_id`、`channel_type`、`context_data`（JSONB） |
| `episodic_memory` | 长期用户记忆 | - | `user_id`、`summary`、`importance` |

### 增强功能表

| 表名 | 用途 | 版本 | 关键列 |
|:-----|:-----|:-----|:-----|
| `user_preferences` | 用户沟通偏好 | - | `user_id`、`preferences`（JSONB） |
| `agent_session_stats` | 会话统计（成本追踪） | v0.97.0 | `session_id`、`token_usage`、`cost_estimate` |
| `user_cost_settings` | 用户成本预算设置 | v0.97.0 | `user_id`、`daily_budget` |
| `agent_security_audit` | 安全审计日志 | v0.97.0 | `user_id`、`risk_level`、`operation` |

### 智能路由表（v0.97.0 新增）

| 表名 | 用途 | 关键列 |
|:-----|:-----|:-----|
| `router_feedback` | 路由反馈收集 | `predicted_intent`, `actual_intent`, `confidence` |
| `router_weight` | 动态权重存储 | `user_id`, `keyword`, `weight` |

### 集成表

| 表名 | 用途 | 关键列 |
|:-----|:-----|:-----|
| `chat_app_credential` | 聊天应用接入凭证 | `platform`、`platform_user_id`、`access_token`（AES-256-GCM 加密） |

---

### ai_block 结构（统一块模型）

```sql
CREATE TABLE ai_block (
  id                  BIGSERIAL PRIMARY KEY,
  uid                 VARCHAR(64) UNIQUE,
  conversation_id     INTEGER NOT NULL REFERENCES ai_conversation(id) ON DELETE CASCADE,
  round_number        INTEGER DEFAULT 0,
  block_type          TEXT DEFAULT 'message',
  mode                TEXT DEFAULT 'normal',  -- normal | geek | evolution

  -- 数据存储
  user_inputs         JSONB DEFAULT '[]',
  assistant_content   TEXT,
  event_stream        JSONB DEFAULT '[]',
  session_stats       JSONB,
  cc_session_id       VARCHAR(64),

  -- 状态管理
  status              TEXT DEFAULT 'pending',  -- pending | streaming | completed | error
  parent_block_id     BIGINT DEFAULT 0,
  branch_path         TEXT,
  regeneration_count  INTEGER DEFAULT 0,
  archived_at         BIGINT,

  -- 成本追踪
  token_usage         JSONB,
  cost_estimate       BIGINT DEFAULT 0,
  model_version       TEXT,

  -- 元数据
  metadata            JSONB,
  created_ts          BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000,
  updated_ts          BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000
);

-- 索引
CREATE INDEX idx_ai_block_conversation ON ai_block(conversation_id, round_number);
CREATE INDEX idx_ai_block_status ON ai_block(status);
CREATE INDEX idx_ai_block_cc_session ON ai_block(cc_session_id);
CREATE INDEX idx_ai_block_mode ON ai_block(mode);
CREATE INDEX idx_ai_block_created ON ai_block(created_ts DESC);
```

---

### conversation_context 结构（多渠道支持）

```sql
CREATE TABLE conversation_context (
  id            SERIAL PRIMARY KEY,
  session_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id       INTEGER NOT NULL REFERENCES "user"(id),
  agent_type    VARCHAR(20) NOT NULL,  -- 'memo', 'schedule', 'amazing', 'auto'
  channel_type  VARCHAR(20) DEFAULT 'web',  -- 'web', 'telegram', 'whatsapp', 'dingtalk'
  context_data  JSONB NOT NULL DEFAULT '{}',
  created_ts    BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000,
  updated_ts    BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000
);

-- 索引
CREATE INDEX idx_conversation_context_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_context_updated ON conversation_context(updated_ts DESC);
CREATE INDEX idx_conversation_context_channel_type ON conversation_context(channel_type);
```

**context_data 结构**：
```json
{
  "messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "metadata": {
    "topic": "...",
    "channel": "web",
    "platform_user_id": "..."
  }
}
```

**保留期**：会话在 30 天后自动过期（可通过清理任务配置）。

---

### agent_session_stats 结构（v0.97.0）

```sql
CREATE TABLE agent_session_stats (
  id                      BIGSERIAL PRIMARY KEY,
  user_id                 INTEGER NOT NULL REFERENCES "user"(id),
  session_id              VARCHAR(64) NOT NULL,
  agent_type              VARCHAR(20) NOT NULL,
  parrot_id               VARCHAR(20),

  -- Token 统计
  prompt_tokens           INTEGER DEFAULT 0,
  completion_tokens       INTEGER DEFAULT 0,
  cache_read_tokens       INTEGER DEFAULT 0,
  cache_write_tokens      INTEGER DEFAULT 0,
  total_tokens            INTEGER DEFAULT 0,

  -- 成本统计（毫美分）
  prompt_cost             BIGINT DEFAULT 0,
  completion_cost         BIGINT DEFAULT 0,
  total_cost              BIGINT DEFAULT 0,

  -- 性能指标
  latency_ms              BIGINT DEFAULT 0,
  tool_calls              INTEGER DEFAULT 0,
  thinking_time_ms        BIGINT DEFAULT 0,

  -- 状态
  status                  VARCHAR(20) DEFAULT 'success',

  created_ts              BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000
);
```

---

### user_cost_settings 结构（v0.97.0）

```sql
CREATE TABLE user_cost_settings (
  id              SERIAL PRIMARY KEY,
  user_id         INTEGER NOT NULL UNIQUE REFERENCES "user"(id),
  daily_budget    BIGINT DEFAULT 100000,  -- 每日预算（毫美分）
  alert_threshold REAL DEFAULT 0.8,       -- 告警阈值（百分比）
  cost_alerts_enabled BOOLEAN DEFAULT true,
  created_ts      BIGINT NOT NULL,
  updated_ts      BIGINT NOT NULL
);
```

---

### router_feedback 结构（v0.97.0 新增）

```sql
CREATE TABLE router_feedback (
  id                SERIAL PRIMARY KEY,
  user_id           INTEGER NOT NULL REFERENCES "user"(id),
  session_id        VARCHAR(64),
  query_text        TEXT NOT NULL,
  predicted_intent  VARCHAR(20) NOT NULL,
  actual_intent     VARCHAR(20),
  confidence        REAL,
  feedback_type     VARCHAR(20),  -- 'auto', 'manual', 'correction'
  created_ts        BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()) * 1000
);

-- 索引
CREATE INDEX idx_router_feedback_user ON router_feedback(user_id);
CREATE INDEX idx_router_feedback_intent ON router_feedback(predicted_intent);
CREATE INDEX idx_router_feedback_created ON router_feedback(created_ts DESC);
```

---

### router_weight 结构（v0.97.0 新增）

```sql
CREATE TABLE router_weight (
  id            SERIAL PRIMARY KEY,
  user_id       INTEGER NOT NULL REFERENCES "user"(id),
  keyword       VARCHAR(100) NOT NULL,
  weight        REAL DEFAULT 1.0,
  intent_type   VARCHAR(20),
  created_ts    BIGINT NOT NULL,
  updated_ts    BIGINT NOT NULL,
  UNIQUE (user_id, keyword)
);

-- 索引
CREATE INDEX idx_router_weight_user ON router_weight(user_id);
CREATE INDEX idx_router_weight_keyword ON router_weight(keyword);
```

### chat_app_credential 结构

```sql
CREATE TABLE chat_app_credential (
  id               SERIAL PRIMARY KEY,
  creator_id       INTEGER NOT NULL REFERENCES "user"(id),
  platform         VARCHAR(20) NOT NULL,  -- 'telegram', 'whatsapp', 'dingtalk'
  platform_user_id VARCHAR(255) NOT NULL,
  access_token     TEXT NOT NULL,         -- AES-256-GCM 加密存储
  app_secret       TEXT,                  -- 钉钉 AppSecret（加密）
  created_ts       BIGINT NOT NULL,
  updated_ts       BIGINT NOT NULL,
  UNIQUE (creator_id, platform, platform_user_id)
);

-- 索引
CREATE INDEX idx_chat_app_credential_creator ON chat_app_credential(creator_id);
CREATE INDEX idx_chat_app_credential_platform ON chat_app_credential(platform);
CREATE UNIQUE INDEX idx_chat_app_credential_unique ON chat_app_credential(creator_id, platform, platform_user_id);
```

**支持的平台**：
- `telegram` — Telegram Bot（Bot Token）
- `dingtalk` — 钉钉群机器人（AppKey + AppSecret）
- `whatsapp` — WhatsApp（预留，需 Baileys Node.js 服务桥接）

**安全说明**：
- 所有敏感凭证使用 AES-256-GCM 加密存储
- 加密密钥通过环境变量 `DIVINESENSE_CHAT_APPS_SECRET_KEY` 提供（必须 32 字节）
- **启动验证**：服务启动时验证密钥存在性和长度，失败则快速报错
- **输入验证**：平台白名单验证 + 字段长度限制（user_id: 255, token: 2048）
- **Webhook 安全**：
  - 钉钉：HMAC-SHA256 签名 + 时间戳验证（5分钟窗口，防重放攻击）
  - Telegram：Bot Token 匹配验证
  - WhatsApp：桥接服务连接状态检查
- 详见：[Chat Apps 用户指南](../user-guides/CHAT_APPS.md#安全建议)

---

## 目录结构

| 路径 | 用途 |
|:-----|:-----|
| `cmd/divinesense/` | 主程序入口 |
| `server/router/api/v1/` | REST/Connect RPC API 处理器 |
| `server/service/` | 业务逻辑层 |
| `ai/core/retrieval/` | 混合搜索（BM25 + 向量） |
| `server/queryengine/` | 查询分析和路由 |
| `ai/agent/` | AI 代理（MemoParrot、ScheduleParrot、AmazingParrot） |
| `ai/router/` | 三层意图路由 |
| `ai/vector/` | Embedding 服务 |
| `plugin/chat_apps/` | 聊天应用接入（Telegram/钉钉/WhatsApp） |
| `store/` | 数据访问层接口 |
| `store/db/postgres/` | PostgreSQL 实现 |
| `store/migration/postgres/` | 数据库迁移 |
| `proto/api/v1/` | Connect RPC 协议定义 |
| `proto/store/` | Store 协议定义 |
| `web/` | 前端（React + Vite） |
