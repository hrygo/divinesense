# 架构文档

## 项目概述

DivineSense (神识) 是一款隐私优先、轻量级的笔记服务，通过 AI 驱动的「鹦鹉」代理增强用户体验。
- **核心架构**：Go 后端 (Echo/Connect RPC) + React 前端 (Vite/Tailwind)
- **数据存储**：PostgreSQL（生产环境，完整 AI 支持），SQLite（仅开发环境，**不支持 AI 功能**）
- **核心特性**：多代理 AI 系统、语义搜索、日程助理、自托管无遥测
- **端口**：后端 28081，前端 25173，PostgreSQL 25432（开发环境）

## 技术栈

| 领域 | 技术选型 |
|:-----|:--------|
| 后端 | Go 1.25, Echo, Connect RPC, pgvector |
| 前端 | React 18, Vite 7, TypeScript, Tailwind CSS 4, Radix UI, TanStack Query |
| 数据库 | PostgreSQL 16+（生产环境带 AI），SQLite（仅开发环境，**无 AI**） |
| AI | DeepSeek V3（LLM），SiliconFlow（Embedding、Reranker） |

---

## 项目架构

### 目录结构

```
divinesense/
├── cmd/divinesense/     # 主程序入口
├── server/              # HTTP/gRPC 服务器 & 路由
│   ├── router/          # API 处理器（v1 实现）
│   ├── queryengine/     # 查询路由 & 意图检测
│   ├── retrieval/       # 自适应检索（BM25 + 向量）
│   ├── runner/          # 后台任务运行器
│   ├── scheduler/       # 日程管理
│   └── service/         # 业务逻辑层
├── plugin/              # 插件系统
│   ├── ai/              # AI 能力
│   │   ├── agent/       # Parrot 代理（MemoParrot、ScheduleParrot、AmazingParrot）
│   │   ├── router/      # 三层意图路由
│   │   ├── vector/      # Embedding 服务
│   │   ├── memory/      # 情景记忆
│   │   ├── session/     # 对话持久化
│   │   ├── cache/       # LRU 缓存层
│   │   └── metrics/     # 代理性能追踪
│   ├── scheduler/       # 任务调度
│   ├── storage/         # 存储适配器（S3、本地）
│   └── idp/             # 身份提供商
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

1. **服务器初始化**：Profile → DB → Store → Server
   - 使用 Echo 框架 + Connect RPC（gRPC/HTTP 转码）
   - 启动时自动迁移

2. **插件系统** (`plugin/ai/`):
   - LLM 提供商：DeepSeek、OpenAI、Ollama
   - Embedding：SiliconFlow (BAAI/bge-m3)、OpenAI
   - Reranker：BAAI/bge-reranker-v2-m3
   - 所有 AI 功能可选（由 `DIVINESENSE_AI_ENABLED` 控制）

3. **后台运行器** (`server/runner/`):
   - 异步生成笔记 Embedding
   - AI 操作任务队列
   - AI 启用时自动运行

4. **存储层**：
   - 接口定义在 `store/`
   - 驱动特定实现在 `store/db/{postgres,sqlite}/`
   - 迁移系统在 `store/migration/`

5. **智能查询引擎** (`server/queryengine/`):
   - 自适应检索（BM25 + 向量搜索 + 选择性重排）
   - 智能查询路由（检测日程 vs. 搜索查询）
   - 自然语言日期解析
   - 带冲突检测的日程助理

---

## Parrot 代理架构

### 代理类型 (`plugin/ai/agent/`)

| AgentType | 鹦鹉名称 | 文件 | 中文名 | 描述 |
|:---------:|:--------|:-----|:-------|:-----|
| `MEMO` | 灰灰 | `memo_parrot.go` | 灰灰 | 笔记搜索和检索专家 |
| `SCHEDULE` | 金刚 | `schedule_parrot_v2.go` | 金刚 | 日程创建和管理 |
| `AMAZING` | 惊奇 | `amazing_parrot.go` | 惊奇 | 综合助理（笔记 + 日程） |
| `GEEK` | 极客 | `geek_parrot.go` | 极客 | Claude Code CLI 通信层（零 LLM） |
| `EVOLUTION` | 进化 | `evolution_parrot.go` | 进化 | 自我进化能力（源代码修改） |

### 代理路由器

**位置**：`plugin/ai/agent/chat_router.go`

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

**位置**：`plugin/ai/agent/tools/`

| 工具 | 文件 | 描述 |
|:-----|:-----|:-----|
| `memo_search` | `memo_search.go` | 语义笔记搜索 + RRF 融合 |
| `schedule_add` | `scheduler.go` | 创建新日程 |
| `schedule_query` | `scheduler.go` | 查询现有日程 |
| `schedule_update` | `scheduler.go` | 更新现有日程 |
| `find_free_time` | `scheduler.go` | 查找空闲时间段 |

### 日程代理 V2

**位置**：`plugin/ai/agent/scheduler_v2.go`

实现原生工具调用循环（现代 LLM 不需要 ReAct）：

**特性**：
- 带结构化参数的直接函数调用
- 默认 1 小时持续时间
- 自动冲突检测
- 时区感知日程安排

---

## AI 服务 (`plugin/ai/`)

### 服务概览

| 服务 | 包 | 描述 |
|:-----|:-----|:-----|
| Memory | `memory/` | 情景记忆 & 用户偏好 |
| Session | `session/` | 对话持久化（30 天保留） |
| Router | `router/` | 三层意图分类 & 路由 |
| Cache | `cache/` | 带 TTL 的 LRU 缓存（查询结果） |
| Metrics | `metrics/` | 代理 & 工具性能追踪（A/B 测试） |
| Vector | `vector/` | 多提供商 Embedding 服务 |

### 会话服务 (`plugin/ai/session/`)

为 AI 代理提供对话持久化：

**组件**：
- `store.go`：PostgreSQL 持久化 + 直写缓存（30min TTL）
- `recovery.go`：会话恢复工作流 + 滑动窗口（最多 20 条消息）
- `cleanup.go`：过期会话清理后台任务（默认：30 天）

**数据库**：`conversation_context` 表（JSONB 存储）

### 上下文构建器 (`plugin/ai/context/`)

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

## 检索系统 (`server/retrieval/`)

### AdaptiveRetriever

混合 BM25 + 向量搜索 + 智能融合：

| 策略 | 描述 |
|:-----|:-----|
| `BM25Only` | 仅关键词搜索（快，低质量） |
| `SemanticOnly` | 仅向量搜索（慢，语义） |
| `HybridStandard` | BM25 + 向量 + RRF 融合（平衡） |
| `FullPipeline` | 混合 + 重排器（最高质量，最慢） |

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

| 路径 | 组件 | 布局 | 用途 |
|:-----|:-----|:-----|:-----|
| `/` | `Home.tsx` | MainLayout | 主时间线 + 笔记编辑器 |
| `/explore` | `Explore.tsx` | MainLayout | 搜索和探索内容 |
| `/archived` | `Archived.tsx` | MainLayout | 已归档笔记 |
| `/chat` | `AIChat.tsx` | AIChatLayout | AI 聊天界面 + 自动路由 |
| `/schedule` | `Schedule.tsx` | ScheduleLayout | 日历视图（FullCalendar） |
| `/review` | `Review.tsx` | MainLayout | 每日回顾 |
| `/setting` | `Setting.tsx` | MainLayout | 用户设置 |
| `/u/:username` | `UserProfile.tsx` | MainLayout | 公开用户资料 |

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

| 表名 | 用途 |
|:-----|:-----|
| `memo_embedding` | 向量嵌入（1024 维）用于语义搜索 |
| `conversation_context` | 会话持久化（30 天保留） |
| `episodic_memory` | 长期用户记忆和学习 |
| `user_preferences` | 用户沟通偏好 |
| `agent_metrics` | A/B 测试指标（prompt 版本、延迟、成功率） |

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
