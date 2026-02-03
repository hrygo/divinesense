# 后端与数据库指南

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **最后检查**: v6.1 (AI Core 重构)

## 数据库支持策略

### PostgreSQL（生产环境 - 完整支持）
- **状态**：生产环境主数据库
- **AI 功能**：完整支持（pgvector、混合搜索、重排、会话记忆）
- **推荐用途**：所有生产部署
- **维护状态**：积极维护和测试
- **端口**：25432（开发环境）
- **版本**：PostgreSQL 16+

### SQLite（仅开发环境 - 无 AI 功能）

- **状态**：仅限开发和测试
- **AI 功能**：**不支持** —— 向量搜索、对话持久化、重排均已禁用
- **推荐用途**：仅限非 AI 功能的本地开发
- **限制**：
  - 无 AI 对话持久化（AI 功能需使用 PostgreSQL）
  - 无向量搜索、BM25、混合搜索或重排
  - 无并发写入支持
  - 无全文搜索（FTS5 不保证）
- **维护状态**：仅对非 AI 功能尽力维护

> 💡 **SQLite AI 支持研究**：详见 [#9](https://github.com/hrygo/divinesense/issues/9) - 探索向量搜索可行性及替代方案

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
| 测试文件 | `*_test.go` | `memo_parrot_test.go` |
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

**AI（推荐 SiliconFlow/DeepSeek）**：
```bash
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

| 代理 | 文件 | 用途 | 工具 |
|:-----|:-----|:-----|:-----|
| **MemoParrot** | `memo_parrot.go` | 笔记搜索和检索 | `memo_search` |
| **ScheduleParrotV2** | `schedule_parrot_v2.go` | 日程管理 | `schedule_add`、`schedule_query`、`schedule_update`、`find_free_time` |
| **AmazingParrot** | `amazing_parrot.go` | 组合笔记 + 日程 | 所有工具 + 并发执行 |

**聊天路由流程**（`chat_router.go`）：
```
输入 → 规则匹配（0ms）→ 历史感知（~10ms）→ LLM 降级（~400ms）
       ↓                ↓                   ↓
    关键词         对话上下文          语义理解
```

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

| 表名 | 用途 | 关键列 |
|:-----|:-----|:-----|
| `memo_embedding` | 语义搜索的向量嵌入 | `memo_id`、`embedding`（vector(1024)） |
| `conversation_context` | AI 代理的会话持久化 | `session_id`、`user_id`、`context_data`（JSONB） |
| `episodic_memory` | 长期用户记忆 | `user_id`、`summary`、`embedding`（vector） |
| `user_preferences` | 用户沟通偏好 | `user_id`、`preferences`（JSONB） |
| `agent_metrics` | 代理性能追踪 | `agent_type`、`prompt_version`、`success_rate`、`avg_latency` |

### conversation_context 结构

```sql
CREATE TABLE conversation_context (
  id            SERIAL PRIMARY KEY,
  session_id    VARCHAR(64) NOT NULL UNIQUE,
  user_id       INTEGER NOT NULL REFERENCES "user"(id),
  agent_type    VARCHAR(20) NOT NULL,  -- 'memo', 'schedule', 'amazing'
  context_data  JSONB NOT NULL,         -- 消息 + 元数据
  created_ts    BIGINT NOT NULL,
  updated_ts    BIGINT NOT NULL
);

-- 索引
CREATE INDEX idx_conversation_context_user ON conversation_context(user_id);
CREATE INDEX idx_conversation_context_updated ON conversation_context(updated_ts DESC);
```

**context_data 结构**：
```json
{
  "messages": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "metadata": {"topic": "...", ...}
}
```

**保留期**：会话在 30 天后自动过期（可通过清理任务配置）。

### agent_metrics 结构

```sql
CREATE TABLE agent_metrics (
  id             SERIAL PRIMARY KEY,
  agent_type     VARCHAR(20) NOT NULL,
  prompt_version VARCHAR(20) NOT NULL,  -- A/B 测试
  success_count  INTEGER DEFAULT 0,
  failure_count  INTEGER DEFAULT 0,
  avg_latency_ms BIGINT DEFAULT 0,
  updated_ts     BIGINT NOT NULL
);
```

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
| `store/` | 数据访问层接口 |
| `store/db/postgres/` | PostgreSQL 实现 |
| `store/migration/postgres/` | 数据库迁移 |
| `proto/api/v1/` | Connect RPC 协议定义 |
| `proto/store/` | Store 协议定义 |
| `web/` | 前端（React + Vite） |
