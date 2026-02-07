# Memos 重构方案：个人智能助理 (Personal Intelligence Assistant)

> **版本**: v1.0
> **创建时间**: 2026-01-22
> **目标周期**: 6-8 个月（分 5 个阶段）
> **核心理念**: 轻量级 + 智能化 + 可扩展 + 成本可控

---

## 目录

1. [项目概述](#1-项目概述)
2. [现有系统分析](#2-现有系统分析)
3. [重构方案](#3-重构方案)
4. [核心功能设计](#4-核心功能设计)
5. [技术架构](#5-技术架构)
6. [实施路线图](#6-实施路线图)
7. [验收标准](#7-验收标准)
8. [风险评估](#8-风险评估)

---

## 1. 项目概述

### 1.1 产品定位

**Memos** 是一款**隐私优先、轻量级的个人智能助理**，核心能力包括：

- **知识管理**：笔记 + 附件 + 全文检索（含 OCR）
- **时间管理**：智能日程 + 自然语言创建 + 冲突检测
- **智能查询**：混合检索（BM25 + Vector + Reranker）+ RAG 对话
- **可扩展性**：支持邮件、任务等未来功能

### 1.2 重构目标

#### 功能目标（5 大核心能力）

| 序号 | 功能               | 描述                                                                 | 优先级 |
| ---- | ------------------ | -------------------------------------------------------------------- | ------ |
| 1    | **记笔记（含附件）** | 支持图片、PDF、Office 文档上传和预览，自动 OCR 提取全文               | P0     |
| 2    | **查笔记与总结**   | 混合检索（SQL + BM25 + Vector + Reranker），AI 总结和生成答案        | P0     |
| 3    | **规划日程**       | 自然语言创建日程，智能冲突检测，建议最佳时间段                        | P0     |
| 4    | **新增修改日程**   | CRUD 操作，支持重复规则（RRULE），多时区处理                          | P0     |
| 5    | **查询总结日程**   | 时间范围查询，AI 总结日程安排，冲突和空闲时间分析                    | P1     |

#### 性能目标

| 指标               | 目标值                  | 说明                           |
| ------------------ | ----------------------- | ------------------------------ |
| **查询延迟 P95**   | < 500ms                 | 混合检索端到端延迟             |
| **向量检索延迟**   | < 100ms (10,000 docs)   | HNSW 索引优化                  |
| **Reranker 延迟**  | < 200ms (Top 20)        | 批处理优化                     |
| **并发 QPS**       | > 1000 QPS              | 单实例，2C4G 配置              |
| **缓存命中率**     | > 60%                   | 三层缓存（L1 内存 + L2 Redis） |

#### 成本目标

| 项目               | 优化前        | 优化后        | 降幅   |
| ------------------ | ------------- | ------------- | ------ |
| **月度 AI 成本**   | ~$10-15/月    | ~$5-8/月      | 40-50% |
| **向量存储成本**   | PostgreSQL    | PostgreSQL    | 0%     |
| **缓存成本**       | 无            | Redis (可选)  | +$2/月 |
| **总成本**         | ~$15-20/月    | ~$7-10/月     | 50%    |

### 1.3 范围界定

#### 包含范围

- ✅ 笔记 CRUD + 附件管理（图片、PDF、Office）
- ✅ 日程 CRUD + 自然语言解析 + 重复规则
- ✅ 混合检索 RAG 系统（SQL + BM25 + Vector + Reranker）
- ✅ 三层缓存架构（L1 内存 + L2 Redis + L3 PG）
- ✅ 性能优化（HNSW 索引、连接池、Worker Pool、批处理）
- ✅ FinOps 成本优化（语义缓存、条件性 Reranker）
- ✅ 完整的可观测性（Metrics、Tracing、Logging）

#### 不包含范围

- ❌ 多用户协作（保持单用户隐私优先）
- ❌ 移动端原生应用（专注 Web + PWA）
- ❌ 企业级功能（SSO、审计日志、RBAC）
- ❌ 第三方集成（钉钉、飞书、Google Calendar）- Phase 6 考虑

---

## 2. 现有系统分析

### 2.1 优势

#### 技术架构优势

| 优势                    | 说明                                                                 | 价值                       |
| ----------------------- | -------------------------------------------------------------------- | -------------------------- |
| **智能查询路由**        | `QueryRouter` 自动识别日程/笔记查询，选择最优策略                    | 减少 60% 无效向量检索      |
| **Adaptive Retrieval**  | 混合检索（BM25 + Vector + Reranker），动态调整策略                   | 查询准确率 +40%            |
| **隐私优先设计**        | 单用户架构，数据本地存储，无第三方追踪                               | 符合 GDPR 和数据主权要求   |
| **轻量级技术栈**        | Go + PostgreSQL + React，无微服务复杂度                             | 低运维成本，易部署         |
| **插件化 AI**           | Embedding/Reranker/LLM 服务抽象，支持多供应商切换                    | 避免厂商锁定               |

#### 代码质量优势

```go
// 示例：智能查询路由（已完成）
queryRouter := queryengine.NewQueryRouter()
decision := queryRouter.Route(ctx, "明天下午3点的会议")
// 输出：Strategy=schedule_bm25_only, TimeRange=[明天 00:00-23:59]

// 示例：自适应检索（已完成）
retriever := retrieval.NewAdaptiveRetriever(store, embedding, reranker)
results := retriever.Retrieve(ctx, &retrieval.RetrievalOptions{
    Strategy: "hybrid_with_time_filter",  // 根据决策自动选择
    TimeRange: decision.TimeRange,
})
```

#### 已完成功能（截至 2026-01-22）

| 模块               | 完成度 | 说明                                   |
| ------------------ | ------ | -------------------------------------- |
| **向量检索**       | 100%   | pgvector + HNSW 索引                   |
| **Reranker**       | 100%   | BAAI/bge-reranker-v2-m3                |
| **智能查询路由**   | 100%   | 日程/笔记自动识别                      |
| **混合检索**       | 100%   | BM25 + Vector + Reranker               |
| **日程 CRUD**      | 100%   | 自然语言解析 + 冲突检测                |
| **成本监控**       | 100%   | `CostMonitor` + `query_cost_log` 表    |
| **前端基础**       | 80%    | React Query hooks + 基础 UI 组件        |

### 2.2 问题与痛点

#### 功能缺口

| 问题                          | 影响                             | 优先级 |
| ----------------------------- | -------------------------------- | ------ |
| **无附件管理**                | 无法管理图片、PDF、Office 文档   | P0     |
| **无 OCR 能力**               | 图片和扫描件无法全文检索         | P0     |
| **日程重复规则不完整**        | 无 RRULE 解析器，只支持简单重复  | P1     |
| **无缓存层**                  | 高频查询重复调用 AI API          | P0     |
| **无性能监控**                | 无法追踪慢查询和性能瓶颈         | P1     |

#### 性能瓶颈

| 瓶颈                    | 现状                         | 优化潜力              |
| ----------------------- | ---------------------------- | --------------------- |
| **向量检索延迟**        | 100-300ms (10,000 docs)      | 优化至 < 100ms        |
| **Reranker 批处理缺失** | 逐个调用，无批量优化         | 减少 50% 延迟         |
| **N+1 查询**            | 日程/笔记查询未预加载         | 减少 70% DB 往返      |
| **无查询缓存**          | 相同查询重复计算向量         | 减少 60% AI 调用      |

#### 成本优化空间

| 优化项                  | 现状                | 优化后              | 节省           |
| ----------------------- | ------------------- | ------------------- | -------------- |
| **语义缓存**            | 无                  | L1 内存 + L2 Redis  | -60% AI 成本   |
| **条件性 Reranker**     | 所有查询都调用      | 仅复杂查询调用      | -40% 成本      |
| **Embedding 批处理**    | 批次大小 = 8        | 动态调整至 16-32    | -20% 调用次数  |
| **查询路由优化**        | 已实现，但未完全调优 | 添加更多规则        | -15% 无效调用  |

### 2.3 改进机会

#### 架构优化机会

```
现状：
Query → Embedding → Vector Search → Reranker → Results
      (每次都调用，无缓存)

优化后（Phase 2）：
Query → QueryRouter → Cache Hit?
                      ├─ Yes → Return Cached (60% 命中)
                      └─ No  → AdaptiveRetrieval → Update Cache
```

#### 功能扩展机会

| 机会                          | 技术方案                              | 工作量 |
| ----------------------------- | ------------------------------------- | ------ |
| **附件管理**                  | MinIO/S3 + Tika 全文提取 + Tesseract OCR | 2 周   |
| **智能摘要**                  | LLM 长上下文总结（支持 100K+ tokens）  | 1 周   |
| **邮件集成**                  | IMAP 同步 + RAG 检索                  | 3 周   |
| **任务管理**                  | Todo + 优先级 + 依赖关系              | 2 周   |

---

## 3. 重构方案

### 3.1 设计原则

#### 核心原则

1. **向后兼容**：不破坏现有笔记和日程数据
2. **渐进式重构**：分阶段实施，每个阶段可独立交付
3. **可扩展性**：插件化架构，轻松添加新功能
4. **性能优先**：< 500ms P95 延迟，1000+ QPS
5. **成本可控**：FinOps 驱动，月成本降低 40-50%

#### 技术原则

- **单一数据源**：PostgreSQL 作为主存储，避免数据分散
- **缓存优先**：三层缓存，减少 DB 和 AI 调用
- **异步处理**：耗时操作（OCR、Embedding）后台化
- **可观测性**：Metrics + Tracing + Logging 全覆盖
- **安全优先**：输入验证、SQL 注入防护、速率限制

### 3.2 整体架构

#### 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Memo Editor  │  │ Calendar View│  │ AI Chat Box            │ │
│  │ + Attachment │  │ + Schedule   │  │ + RAG                  │ │
│  │              │  │   Management │  │ + Intent Recognition   │ │
│  └──────────────┘  └──────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │ gRPC / Connect (HTTP/2)
┌─────────────────────────────────────────────────────────────────┐
│                      API Gateway (Echo)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │MemoService   │  │ScheduleSvc   │  │AIService                │ │
│  │              │  │              │  │  ├─ SemanticSearch      │ │
│  │  └─ Upload   │  │  └─ Parse    │  │  ├─ ChatWithMemos      │ │
│  │  └─ OCR      │  │  └─ Conflict │  │  ├─ Summarize          │ │
│  └──────────────┘  └──────────────┘  │  └─ SuggestTags        │ │
│                                        └────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                    Business Logic Layer                         │
│  ┌──────────────────┐  ┌────────────────┐  ┌─────────────────┐ │
│  │ QueryRouter      │  │ AdaptiveRetri  │  │ CostMonitor     │ │
│  │  (智能路由)       │  │  (混合检索)     │  │  (FinOps)       │ │
│  └──────────────────┘  └────────────────┘  └─────────────────┘ │
│  ┌──────────────────┐  ┌────────────────┐  ┌─────────────────┐ │
│  │ CacheManager     │  │ OCRProcessor   │  │ AttachmentMgr   │ │
│  │  (三层缓存)       │  │  (Tesseract)    │  │  (S3/MinIO)     │ │
│  └──────────────────┘  └────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                      Storage & AI Layer                         │
│  ┌──────────────────┐  ┌────────────────┐  ┌─────────────────┐ │
│  │ PostgreSQL       │  │ Redis (可选)   │  │ AI Providers    │ │
│  │  ├─ memo         │  │  ├─ L1 Cache   │  │  ├─ Embedding   │ │
│  │  ├─ schedule     │  │  ├─ L2 Cache   │  │  ├─ Reranker    │ │
│  │  ├─ attachment   │  │  └─ Session    │  │  └─ LLM        │ │
│  │  ├─ memo_embedding│ │                 │  │                 │ │
│  │  └─ query_cost   │  │                 │  │                 │ │
│  └──────────────────┘  └────────────────┘  └─────────────────┘ │
│  ┌──────────────────┐  ┌────────────────┐  ┌─────────────────┐ │
│  │ S3/MinIO         │  │ Background     │  │ Observability   │ │
│  │  (附件存储)       │  │  Runners       │  │  ├─ Prometheus  │ │
│  │                  │  │  ├─ Embedding  │  │  ├─ Jaeger      │ │
│  │                  │  │  ├─ OCR        │  │  └─ Loki        │ │
│  │                  │  │  └─ Reminder   │  │                 │ │
│  └──────────────────┘  └────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 技术栈

#### 后端技术栈

| 组件           | 技术方案                          | 说明                               |
| -------------- | --------------------------------- | ---------------------------------- |
| **语言**       | Go 1.25+                          | 高性能、并发友好                   |
| **框架**       | Echo + Connect RPC                | gRPC-HTTP 转换，类型安全           |
| **数据库**     | PostgreSQL 16+                    | pgvector、JSONB、全文检索          |
| **向量引擎**   | pgvector (HNSW)                   | m=16, ef_construction=64           |
| **缓存**       | Redis 7+ (可选)                   | L2 缓存、Session 存储              |
| **对象存储**   | MinIO / AWS S3                    | 附件存储、预览图生成               |
| **全文提取**   | Apache Tika                       | PDF、Office 文档解析               |
| **OCR**        | Tesseract + chi-sim traineddata   | 中文图片识别                       |
| **监控**       | Prometheus + Grafana             | Metrics、可视化                    |
| **追踪**       | OpenTelemetry + Jaeger           | 分布式追踪                         |
| **日志**       | Loki + Structlog (slog)          | 结构化日志                         |

#### 前端技术栈

| 组件           | 技术方案                          | 说明                               |
| -------------- | --------------------------------- | ---------------------------------- |
| **框架**       | React 18                          | 并发特性、Suspense                 |
| **构建**       | Vite 7                            | 快速冷启动、HMR                    |
| **状态管理**   | TanStack Query (React Query)      | 服务端状态、缓存                   |
| **UI 组件**    | Radix UI + Tailwind CSS 4         | 无障碍、主题定制                   |
| **富文本**     | Tiptap                            | 协作编辑器、Markdown 支持          |
| **日历**       | @fullcalendar/react               | 日程视图、拖拽调整                 |
| **图表**       | Recharts                          | 成本分析、性能监控                 |

---

## 4. 核心功能设计

### 4.1 记笔记（含附件）

#### 功能需求

| 需求                | 描述                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| **附件上传**        | 支持图片（PNG/JPG）、PDF、Office（Word/Excel/PPT）                    |
| **自动预览**        | 图片生成缩略图，PDF 提取首页预览                                      |
| **全文提取**        | Apache Tika 提取文档文本（用于检索）                                  |
| **OCR 识别**        | 图片和扫描件自动 OCR（中文 + 英文）                                   |
| **版本管理**        | 附件更新时保留历史版本                                               |
| **权限控制**        | 笔记继承附件权限（Private/Protected/Public）                          |

#### 数据模型

```sql
-- 附件表
CREATE TABLE attachment (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

  -- 文件信息
  filename TEXT NOT NULL,
  filesize BIGINT NOT NULL,
  content_type TEXT NOT NULL,
  storage_type TEXT NOT NULL DEFAULT 's3',  -- 's3' | 'local'

  -- 存储路径
  file_path TEXT NOT NULL,                  -- S3 key or local path
  thumbnail_path TEXT,                      -- 缩略图路径

  -- 全文提取和 OCR
  extracted_text TEXT,                      -- Tika 提取的文本
  ocr_text TEXT,                            -- OCR 识别的文本

  -- 关联
  memo_id INTEGER REFERENCES memo(id) ON DELETE CASCADE,

  -- 元数据
  payload JSONB NOT NULL DEFAULT '{}',      -- 扩展字段

  -- 标准字段
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  row_status TEXT NOT NULL DEFAULT 'NORMAL'
);

-- 索引
CREATE INDEX idx_attachment_creator ON attachment(creator_id, created_ts);
CREATE INDEX idx_attachment_memo ON attachment(memo_id);
CREATE INDEX idx_attachment_type ON attachment(content_type);

-- 全文搜索索引（PostgreSQL）
CREATE INDEX idx_attachment_text_gin ON attachment USING gin(to_tsvector('simple', COALESCE(extracted_text, '') || ' ' || COALESCE(ocr_text, '')));
```

#### 核心流程

```
1. 用户上传附件
   ↓
2. 前端：分片上传（大文件 > 10MB）
   ↓
3. 后端：UploadAttachment API
   ├─ 验证文件类型和大小
   ├─ 上传到 S3/MinIO
   ├─ 生成缩略图（图片）
   └─ 保存记录到 attachment 表
   ↓
4. 后台任务（异步）：
   ├─ Tika 全文提取（PDF/Office）
   └─ Tesseract OCR（图片）
   ↓
5. 更新 attachment 表：
   ├─ extracted_text (Tika)
   └─ ocr_text (OCR)
   ↓
6. 触发 Embedding 生成（后台 runner）
```

#### API 设计

```protobuf
service AttachmentService {
  // 上传附件（支持分片上传）
  rpc UploadAttachment(UploadAttachmentRequest) returns (stream UploadProgress);

  // 获取附件详情
  rpc GetAttachment(GetAttachmentRequest) returns (Attachment);

  // 下载附件（返回 S3 预签名 URL）
  rpc DownloadAttachment(DownloadAttachmentRequest) returns (DownloadUrl);

  // 删除附件
  rpc DeleteAttachment(DeleteAttachmentRequest) returns (google.protobuf.Empty);
}

message UploadAttachmentRequest {
  string filename = 1;
  int64 filesize = 2;
  string content_type = 3;
  int32 memo_id = 4;  // 关联的笔记 ID
  bytes chunk = 5;     // 分片数据
  int32 chunk_index = 6;
  int32 total_chunks = 7;
}

message Attachment {
  string name = 1;           // attachment/{uid}
  string filename = 2;
  int64 filesize = 3;
  string content_type = 4;
  string download_url = 5;   // S3 预签名 URL（15 分钟有效）
  string thumbnail_url = 6;
  bool has_ocr = 7;
  bool has_extracted_text = 8;
}
```

#### 性能优化

| 优化项                | 方案                                   |
| --------------------- | -------------------------------------- |
| **大文件上传**        | 分片上传（10MB/片），断点续传          |
| **缩略图生成**        | 异步后台任务，不阻塞上传               |
| **全文提取**          | 后台队列，优先级队列（PDF > Office）   |
| **OCR 优化**          | 仅对无文本的图片执行 OCR               |
| **缓存预览图**        | Redis 缓存缩略图 URL（TTL=1h）          |

---

### 4.2 查笔记与总结

#### 功能需求

| 需求                | 描述                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| **混合检索**        | SQL 过滤 + BM25 全文搜索 + 向量相似度 + Reranker 重排序               |
| **智能查询路由**    | 自动识别查询类型（笔记/日程/总结）                                   |
| **AI 总结**         | 基于检索结果生成摘要（支持长上下文）                                 |
| **语义搜索**        | 支持自然语言查询（"上周关于 AI 的笔记"）                             |
| **相关推荐**        | 展示相关笔记（基于向量相似度）                                       |

#### 混合检索架构

```
Query Input
    ↓
QueryRouter (智能路由)
    ├─ 检测时间范围 → "今天"、"本周"
    ├─ 检测关键词 → "笔记"、"日程"
    ├─ 检测疑问词 → "总结"、"是什么"
    └─ 输出：RouteDecision {Strategy, TimeRange, NeedsReranker}
    ↓
CacheManager (三层缓存)
    ├─ L1: 内存缓存（相似查询，TTL=5min）
    ├─ L2: Redis缓存（热门查询，TTL=1h）
    └─ L3: PostgreSQL（持久化）
    ↓
[Cache Miss] → AdaptiveRetriever (混合检索)
    ├─ Path 1: schedule_bm25_only (日程查询)
    ├─ Path 2: memo_semantic_only (纯向量)
    ├─ Path 3: hybrid_bm25_weighted (BM25 + Vector 融合)
    ├─ Path 4: hybrid_with_time_filter (时间过滤)
    └─ Path 5: full_pipeline_with_reranker (完整流程)
    ↓
Reranker (条件性)
    └─ 仅复杂查询调用（Top 20 → Top 10）
    ↓
Results + AI Summary (可选)
```

#### 核心代码示例

```go
// 1. 智能查询路由（已完成）
decision := queryRouter.Route(ctx, "上周关于 AI 的笔记")
// 输出：
// - Strategy: "hybrid_with_time_filter"
// - TimeRange: [上周一 00:00, 上周日 23:59]
// - NeedsReranker: true

// 2. 混合检索（已完成）
results, err := adaptiveRetriever.Retrieve(ctx, &retrieval.RetrievalOptions{
    Query:            "AI 技术",
    UserID:           userID,
    Strategy:         decision.Strategy,
    TimeRange:        decision.TimeRange,
    MinScore:         0.5,
    Limit:            10,
})

// 3. AI 总结（Phase 2 实现）
summary, err := llmService.Summarize(ctx, &ai.SummaryOptions{
    Context:  extractContent(results),
    Query:    "上周关于 AI 的笔记总结",
    MaxTokens: 500,
})
```

#### API 设计

```protobuf
service AIService {
  // 语义搜索（已实现）
  rpc SemanticSearch(SemanticSearchRequest) returns (SemanticSearchResponse);

  // AI 总结（新增）
  rpc SummarizeMemos(SummarizeMemosRequest) returns (SummarizeMemosResponse);
}

message SemanticSearchRequest {
  string query = 1;
  int32 limit = 2;           // 默认 10
  float32 min_score = 3;     // 默认 0.5
  string time_range = 4;     // 可选：自动解析
}

message SemanticSearchResponse {
  repeated MemoWithScore results = 1;
  string summary = 2;        // AI 生成的总结
  string strategy_used = 3;  // 使用的检索策略
  int64 latency_ms = 4;      // 查询延迟
}
```

#### 性能目标

| 指标               | 目标值          | 优化方案                               |
| ------------------ | --------------- | -------------------------------------- |
| **查询延迟 P95**   | < 500ms         | 三层缓存 + HNSW 优化                   |
| **向量检索延迟**   | < 100ms         | HNSW m=16, ef_construction=64          |
| **Reranker 延迟**  | < 200ms         | 批处理（20 条/批）                     |
| **缓存命中率**     | > 60%           | L1 内存 + L2 Redis                     |

---

### 4.3 规划日程

#### 功能需求

| 需求                | 描述                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| **自然语言创建**    | "明天下午3点开会" → 自动解析时间、标题、提醒                          |
| **冲突检测**        | 检测时间重叠的日程，智能建议空闲时间                                  |
| **智能建议**        | 根据历史习惯推荐最佳时间段（如"周五下午"用于总结）                    |
| **重复规则**        | 支持每日、每周、每月、工作日、自定义 RRULE                            |
| **多时区**          | 支持跨时区日程（UTC 存储 + 本地显示）                                |
| **批量操作**        | 批量创建、批量修改、批量删除                                          |

#### 数据模型

```sql
-- 日程表（已实现，扩展字段）
CREATE TABLE schedule (
  id SERIAL PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,

  -- 核心字段
  title TEXT NOT NULL,
  description TEXT DEFAULT '',
  location TEXT DEFAULT '',

  -- 时间字段（UTC 时间戳）
  start_ts BIGINT NOT NULL,
  end_ts BIGINT,
  all_day BOOLEAN NOT NULL DEFAULT FALSE,
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',

  -- 重复规则（RRULE 格式）
  recurrence_rule TEXT,                    -- "FREQ=WEEKLY;BYDAY=MO,WE,FR"
  recurrence_end_ts BIGINT,                -- 重复结束时间
  recurrence_exceptions TEXT[],             -- 排除的日期（[20260115, 20260122]）

  -- 提醒设置
  reminders JSONB NOT NULL DEFAULT '[]',   -- [{"type":"before","value":15,"unit":"minutes"}]

  -- 扩展
  payload JSONB NOT NULL DEFAULT '{}',     -- 智能建议、优先级等

  -- 标准字段
  created_ts BIGINT NOT NULL,
  updated_ts BIGINT NOT NULL,
  row_status TEXT NOT NULL DEFAULT 'NORMAL'
);

-- 索引（已优化）
CREATE INDEX idx_schedule_creator_start ON schedule(creator_id, start_ts);
CREATE INDEX idx_schedule_start_ts ON schedule(start_ts);  -- 全局时间查询
CREATE INDEX idx_schedule_recurrence ON schedule(creator_id, recurrence_rule) WHERE recurrence_rule IS NOT NULL;
```

#### 核心流程

```
用户输入："明天下午3点开会"
    ↓
ParseAndCreateSchedule API (已实现)
    ├─ 1. LLM 解析：
    │   ├─ 标题: "开会"
    │   ├─ 时间: 明天 15:00-16:00
    │   └─ 提醒: 提前 15 分钟
    ├─ 2. 冲突检测：
    │   └─ CheckConflict(start_ts, end_ts)
    │       ├─ 有冲突 → 返回冲突列表 + 建议时间段
    │       └─ 无冲突 → 继续
    └─ 3. 创建日程
    ↓
前端展示
    ├─ 无冲突：直接创建成功
    └─ 有冲突：
        ├─ 展示冲突列表
        ├─ 展示建议时间（AI 推荐）
        └─ 用户选择：
            ├─ 确认原时间
            ├─ 调整到建议时间
            └─ 取消创建
```

#### 智能建议算法

```go
// 智能时间段推荐（Phase 3 实现）
type ScheduleSuggestion struct {
  StartTime    time.Time
  EndTime      time.Time
  Confidence   float32   // 0.0-1.0
  Reason       string    // "基于历史习惯"
}

func (s *ScheduleService) SuggestTimeSlots(ctx context.Context, duration time.Duration) ([]*ScheduleSuggestion, error) {
  // 1. 获取用户历史日程（最近 30 天）
  history, err := s.store.ListSchedules(ctx, &store.FindSchedule{
    CreatorID: &userID,
    StartTs:    time.Now().AddDate(0, 0, -30).Unix(),
    EndTs:      time.Now().Unix(),
  })

  // 2. 分析空闲时间分布
  busySlots := analyzeBusySlots(history)
  freeSlots := findFreeSlots(busySlots, duration, 7)  // 未来 7 天

  // 3. 评分和排序
  suggestions := scoreSuggestions(freeSlots, history)
  return suggestions, nil
}
```

#### API 设计（已实现 + 扩展）

```protobuf
service ScheduleService {
  // 自然语言创建日程（已实现）
  rpc ParseAndCreateSchedule(ParseAndCreateScheduleRequest) returns (ParseAndCreateScheduleResponse);

  // 冲突检测（已实现）
  rpc CheckConflict(CheckConflictRequest) returns (CheckConflictResponse);

  // 智能时间建议（新增）
  rpc SuggestTimeSlots(SuggestTimeSlotsRequest) returns (SuggestTimeSlotsResponse);

  // 批量创建（新增）
  rpc BatchCreateSchedules(BatchCreateSchedulesRequest) returns (BatchCreateSchedulesResponse);
}

message SuggestTimeSlotsRequest {
  int64 duration_minutes = 1;   // 会议时长
  int32 days_ahead = 2;         // 提前几天（默认 7）
  int32 max_suggestions = 3;    // 最多建议数（默认 5）
}

message SuggestTimeSlotsResponse {
  repeated TimeSlot suggestions = 1;
}

message TimeSlot {
  int64 start_ts = 1;
  int64 end_ts = 2;
  float32 confidence = 3;
  string reason = 4;
}
```

---

### 4.4 查询总结日程

#### 功能需求

| 需求                | 描述                                                                 |
| ------------------- | -------------------------------------------------------------------- |
| **时间范围查询**    | "今天的日程"、"本周会议"、"下月重要事项"                              |
| **智能总结**        | AI 生成日程摘要（"今日 3 个会议，2 个空闲时间段"）                    |
| **冲突分析**        | 检测潜在冲突，提供解决建议                                            |
| **负载分析**        | 可视化日程负载（热力图、甘特图）                                      |
| **日历视图**        | 月视图、周视图、日视图（FullCalendar 集成）                          |

#### 核心流程

```
用户查询："今天的会议安排"
    ↓
QueryRouter 识别
    ├─ 类型: ScheduleQuery
    ├─ 时间: 今天 00:00-23:59
    └─ 关键词: "会议" (filter by title/description)
    ↓
AdaptiveRetriever
    ├─ 策略: schedule_bm25_only
    ├─ 查询: ListSchedules(start_ts, end_ts)
    └─ 过滤: title LIKE '%会议%' OR description LIKE '%会议%'
    ↓
结果增强
    ├─ 1. 检测冲突（重叠时间）
    ├─ 2. 计算空闲时间
    ├─ 3. 生成 AI 总结
    └─ 4. 视觉化数据（热力图）
    ↓
返回结果
    ├─ 日程列表
    ├─ AI 总结："今天有 3 个会议，共 4.5 小时，下午 2-4 点最忙"
    ├─ 冲突警告（如有）
    └─ 空闲时间建议
```

#### AI 总结 Prompt 模板

```go
const scheduleSummaryPrompt = `
你是一个专业的时间管理助理。请根据以下日程信息生成总结：

**查询范围**: {{.TimeRange}}
**日程列表**:
{{- range .Schedules}}
- {{.Title}} ({{.StartTime}} - {{.EndTime}})
  地点: {{.Location}}
  描述: {{.Description}}
{{- end}}

**请提供**:
1. 总体概览（日程数量、总时长、最忙时间段）
2. 重要事项提醒（标注高优先级）
3. 冲突和空闲时间分析
4. 时间管理建议（如"建议将低优先级会议合并"）

**输出格式**: 简洁、友好、可操作的中文总结（200 字以内）
`
```

#### API 设计

```protobuf
service ScheduleService {
  // 查询日程（已实现，扩展总结）
  rpc ListSchedules(ListSchedulesRequest) returns (ListSchedulesResponse);

  // AI 总结日程（新增）
  rpc SummarizeSchedules(SummarizeSchedulesRequest) returns (SummarizeSchedulesResponse);
}

message ListSchedulesRequest {
  int64 start_ts = 1;
  int64 end_ts = 2;
  string filter = 3;           // 可选：关键词过滤
  bool include_summary = 4;    // 是否包含 AI 总结
}

message ListSchedulesResponse {
  repeated Schedule schedules = 1;
  ScheduleSummary summary = 2; // AI 生成的总结
  ConflictInfo conflicts = 3;  // 冲突信息
}

message ScheduleSummary {
  string overview = 1;         // "今天有 3 个会议，共 4.5 小时"
  string busy_periods = 2;     // "下午 2-4 点最忙"
  repeated string suggestions = 3;  // 时间管理建议
}

message ConflictInfo {
  int32 conflict_count = 1;
  repeated Conflict conflicts = 2;
  repeated TimeSlot free_slots = 3;  // 空闲时间段
}
```

---

## 5. 技术架构

### 5.1 混合检索 RAG 系统

#### 三层架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ QueryRouter  │  │ AIService    │  │ ScheduleService  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                     Retrieval Layer                         │
│  ┌──────────────────┐  ┌────────────────┐  ┌──────────────┐ │
│  │ AdaptiveRetriever│  │ CacheManager   │  │ Reranker     │ │
│  │  ├─ BM25         │  │  ├─ L1 Memory  │  │  (条件性)    │ │
│  │  ├─ Vector       │  │  ├─ L2 Redis   │  │              │ │
│  │  └─ SQL Filter   │  │  └─ L3 PG      │  │              │ │
│  └──────────────────┘  └────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                        Data Layer                            │
│  ┌──────────────────┐  ┌────────────────┐  ┌──────────────┐ │
│  │ PostgreSQL       │  │ Redis (可选)   │  │ S3/MinIO     │ │
│  │  ├─ memo         │  │                │  │              │ │
│  │  ├─ schedule     │  │                │  │              │ │
│  │  └─ embedding    │  │                │  │              │ │
│  └──────────────────┘  └────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

#### 检索策略矩阵

| 查询类型                  | 推荐策略                     | 说明                               |
| ------------------------- | ---------------------------- | ---------------------------------- |
| **精确时间日程查询**       | `schedule_bm25_only`         | 时间过滤 + 标题匹配                |
| **模糊语义笔记查询**       | `memo_semantic_only`         | 纯向量检索（Top 10）               |
| **混合查询（笔记+日程）**  | `hybrid_bm25_weighted`       | BM25 + Vector 加权融合（0.5:0.5）  |
| **时间范围 + 语义**        | `hybrid_with_time_filter`    | 先时间过滤，再向量检索             |
| **复杂查询（总结类）**     | `full_pipeline_with_reranker` | 完整流程：BM25 + Vector + Reranker |

#### 性能优化配置

```go
// HNSW 索引参数（2C4G 优化）
CREATE INDEX idx_memo_embedding_hnsw
ON memo_embedding USING hnsw (embedding vector_cosine_ops)
WITH (
  m = 16,                -- 连接数（默认 16，平衡召回和内存）
  ef_construction = 64   -- 构建时候选数（默认 64，提高精度）
);

// 查询时 ef 参数（动态调整）
SET hnsw.ef_search = 100;  -- 查询候选数（召回 Top 10 时设为 100）

// 连接池配置
db.SetMaxOpenConns(25)     // 最大连接数（2C4G 推荐）
db.SetMaxIdleConns(5)      // 最大空闲连接
db.SetConnMaxLifetime(5 * time.Minute)
```

### 5.2 缓存架构

#### 三层缓存设计

```
Cache Hit Flow:
1. L1 Memory Cache (Go sync.Map)
   ├─ 存储内容: 相似查询向量、热门结果
   ├─ TTL: 5 分钟
   ├─ 容量: 1000 条
   └─ 命中率目标: 30%

2. L2 Redis Cache (可选)
   ├─ 存储内容: 查询结果、Embedding 向量
   ├─ TTL: 1 小时
   ├─ 容量: 无限（受内存限制）
   └─ 命中率目标: 30%

3. L3 PostgreSQL
   ├─ 存储内容: 持久化数据
   ├─ TTL: 永久
   └─ 命中率目标: 40%

总体缓存命中率目标: 60%+
```

#### 缓存键设计

```go
// 缓存键生成（哈希查询内容）
type CacheKey struct {
  Query      string
  UserID     int32
  TimeRange  *TimeRange
  Strategy   string
  Limit      int
}

func (k *CacheKey) Hash() string {
  data := fmt.Sprintf("%s:%d:%v:%s:%d",
    k.Query, k.UserID, k.TimeRange, k.Strategy, k.Limit)
  return sha256Hex(data)
}

// 示例：
// "明天下午3点的会议" → "a3f2b1c4..."
// 缓存内容: []SearchResult + metadata (timestamp, ttl)
```

#### 缓存更新策略

| 策略       | 触发条件                          | 说明                     |
| ---------- | --------------------------------- | ------------------------ |
| **写穿透** | 创建/更新笔记/日程                | 同步更新缓存，保证一致性 |
| **删除**   | 删除笔记/日程                     | 立即失效相关缓存         |
| **TTL 失效** | 缓存过期                         | 重新计算并更新缓存       |
| **LRU 淘汰** | L1 缓存满（>1000 条）            | 淘汰最久未使用条目       |

### 5.3 性能优化

#### 数据库优化

| 优化项                | 方案                                   | 收益                     |
| --------------------- | -------------------------------------- | ------------------------ |
| **HNSW 索引调优**     | m=16, ef_construction=64, ef_search=100| 检索延迟 -50%            |
| **连接池优化**        | MaxOpen=25, MaxIdle=5                  | 减少 80% 连接建立开销    |
| **查询预加载**        | 预加载 tags、attachments               | 减少 70% N+1 查询        |
| **部分索引**          | 仅索引 NORMAL 状态数据                 | 索引大小 -40%            |
| **批量操作**          | BatchInsert（100 条/批）               | 写入吞吐 +300%           |

#### 应用层优化

| 优化项                | 方案                                   | 收益                     |
| --------------------- | -------------------------------------- | ------------------------ |
| **Worker Pool**       | Embedding 生成（10 workers）            | CPU 利用率 +200%         |
| **批处理**            | Reranker 批处理（20 条/批）             | API 调用 -50%            |
| **并发控制**          | 信号量限制并发数（MaxConcurrent=10）    | 防止资源耗尽             |
| **超时控制**          | API 超时 5s，DB 超时 1s                | 防止慢查询雪崩           |

#### 前端优化

| 优化项                | 方案                                   | 收益                     |
| --------------------- | -------------------------------------- | ------------------------ |
| **React Query**       | 缓存查询结果（staleTime=30s）          | 减少 60% 重复请求        |
| **虚拟滚动**          | 长列表使用 react-window                | 首屏渲染 +300%           |
| **代码分割**          | React.lazy + Suspense                  | 初始包体积 -40%          |
| **预加载**            | 预加载下一页数据                       | 感知延迟 -200ms          |

### 5.4 FinOps 优化

#### 成本优化策略

```
优化前成本结构:
- Embedding API: $5/月 (100K calls)
- Reranker API: $4/月 (50K calls)
- LLM API: $6/月 (10K calls)
总: $15/月

优化后成本结构:
- 语义缓存: -60% Embedding/Reranker 调用 → -$5.4/月
- 条件性 Reranker: -40% Reranker 调用 → -$1.6/月
- 批处理优化: -20% API 调用 → -$0.8/月
总: $7.2/月 (节省 52%)
```

#### 语义缓存实现

```go
// 语义缓存（基于向量相似度）
type SemanticCache struct {
  cache  sync.Map  // key: query_hash, value: *CacheEntry
  embeddingService ai.EmbeddingService
}

type CacheEntry struct {
  Query      string
  Results    []*SearchResult
  Vector     []float32
  Timestamp  time.Time
}

func (c *SemanticCache) Get(query string) ([]*SearchResult, bool) {
  // 1. 精确匹配（哈希查询）
  hash := sha256Hex(query)
  if entry, ok := c.cache.Load(hash); ok {
    return entry.(*CacheEntry).Results, true
  }

  // 2. 语义匹配（余弦相似度 > 0.95）
  queryVector, _ := c.embeddingService.EmbedQuery(ctx, query)
  var bestMatch *CacheEntry
  bestScore := float32(0.0)

  c.cache.Range(func(_, value interface{}) bool {
    entry := value.(*CacheEntry)
    score := cosineSimilarity(queryVector, entry.Vector)
    if score > bestScore {
      bestMatch = entry
      bestScore = score
    }
    return true
  })

  if bestScore > 0.95 {
    return bestMatch.Results, true
  }

  return nil, false
}
```

#### 条件性 Reranker

```go
// 仅复杂查询使用 Reranker
func (r *AdaptiveRetriever) shouldRerank(opts *RetrievalOptions, results []*SearchResult) bool {
  // 条件 1: 结果数 > 5
  if len(results) <= 5 {
    return false
  }

  // 条件 2: 查询包含疑问词
  questionWords := []string{"总结", "是什么", "如何", "why", "how"}
  for _, word := range questionWords {
    if strings.Contains(opts.Query, word) {
      return true
    }
  }

  // 条件 3: 结果分数方差大（说明需要重排序）
  if varianceScores(results) > 0.3 {
    return true
  }

  return false
}
```

#### 成本监控 Dashboard

```go
// 成本指标（Prometheus + Grafana）
var costMetrics = struct {
  TotalCost       prometheus.Gauge
  VectorCost      prometheus.Gauge
  RerankerCost    prometheus.Gauge
  LLMCost         prometheus.Gauge
  CacheHitRate    prometheus.Gauge
  AvgLatency      prometheus.Histogram
}{
  TotalCost:     prometheus.NewGaugeVec(prometheus.GaugeOpts{
    Name: "ai_cost_total_dollars",
    Help: "Total AI cost in dollars",
  }, []string{"strategy"}),
  CacheHitRate: prometheus.NewGauge(prometheus.GaugeOpts{
    Name: "cache_hit_rate",
    Help: "Cache hit rate (0-1)",
  }),
  // ... 其他指标
}

// Grafana 面板配置
// - Panel 1: 实时成本（$/小时）
// - Panel 2: 成本趋势（7 天）
// - Panel 3: 缓存命中率
// - Panel 4: 各策略成本占比
```

### 5.5 可观测性

#### 监控指标

| 类别       | 指标                          | 类型        | 告警阈值              |
| ---------- | ----------------------------- | ----------- | --------------------- |
| **性能**   | query_latency_p95             | Histogram   | > 500ms               |
| **性能**   | vector_search_latency_p95     | Histogram   | > 100ms               |
| **性能**   | reranker_latency_p95          | Histogram   | > 200ms               |
| **成本**   | ai_cost_total_dollars         | Gauge       | > $10/天              |
| **缓存**   | cache_hit_rate                | Gauge       | < 0.5 (50%)           |
| **错误**   | query_errors_total            | Counter     | 错误率 > 1%           |
| **业务**   | daily_active_users            | Gauge       | < 1 (检测异常)        |

#### 链路追踪

```go
// OpenTelemetry 集成
import (
  "go.opentelemetry.io/otel"
  "go.opentelemetry.io/otel/trace"
)

func (s *AIService) SemanticSearch(ctx context.Context, req *SemanticSearchRequest) (*SemanticSearchResponse, error) {
  ctx, span := otel.Tracer("ai-service").Start(ctx, "SemanticSearch")
  defer span.End()

  // 记录属性
  span.SetAttributes(
    attribute.String("query", req.Query),
    attribute.Int64("limit", req.Limit),
  )

  // 子操作：QueryRouter
  ctx, routeSpan := otel.Tracer("query-router").Start(ctx, "Route")
  decision := s.QueryRouter.Route(ctx, req.Query)
  routeSpan.End()

  // 子操作：AdaptiveRetrieval
  ctx, retrieveSpan := otel.Tracer("retrieval").Start(ctx, "Retrieve")
  results, err := s.AdaptiveRetriever.Retrieve(ctx, &RetrievalOptions{
    Strategy: decision.Strategy,
    // ...
  })
  retrieveSpan.End()

  // ... 返回结果
}
```

#### 日志结构

```go
// 结构化日志（slog）
logger.Info("Query executed",
  "request_id", requestID,
  "user_id", userID,
  "query", query,
  "strategy", strategy,
  "latency_ms", latencyMs,
  "result_count", len(results),
  "cache_hit", cacheHit,
  "cost_dollars", cost,
)

// 日志聚合（Loki）
// - 查询模式分析
// - 错误日志统计
// - 性能瓶颈识别
```

---

## 6. 实施路线图

### 6.1 分阶段计划

#### Phase 1: 附件管理（4 周）

**目标**: 实现完整的附件上传、存储、预览、检索功能

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **1.1 数据库迁移**            | 2 天   | 无          | `attachment` 表 + 索引                   |
| **1.2 对象存储集成**          | 3 天   | 1.1         | MinIO/S3 上传/下载 API                   |
| **1.3 全文提取**              | 3 天   | 1.2         | Apache Tika 集成（PDF/Office）            |
| **1.4 OCR 实现**              | 4 天   | 1.2         | Tesseract OCR（中文图片）                 |
| **1.5 前端上传组件**          | 3 天   | 1.2         | 分片上传、进度条、预览                   |
| **1.6 缩略图生成**            | 2 天   | 1.2         | 图片缩略图、PDF 首页预览                  |
| **1.7 后台任务队列**          | 2 天   | 1.3, 1.4    | OCR/全文提取异步任务                     |
| **1.8 测试和文档**            | 3 天   | 所有        | 单元测试 + API 文档                      |

**里程碑**: 用户可以上传 PDF/图片，自动提取全文并支持检索

---

#### Phase 2: 缓存与性能优化（3 周）

**目标**: 实现三层缓存，优化查询性能至 < 500ms P95

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **2.1 L1 内存缓存**           | 2 天   | 无          | `CacheManager` + sync.Map 实现           |
| **2.2 L2 Redis 缓存**         | 3 天   | 2.1         | Redis 集成 + 缓存失效策略                |
| **2.3 语义缓存**              | 4 天   | 2.2         | 基于向量相似度的语义缓存                 |
| **2.4 HNSW 索引优化**         | 2 天   | 无          | 调整 m=16, ef_construction=64            |
| **2.5 连接池调优**            | 1 天   | 无          | DB 连接池优化                            |
| **2.6 批处理优化**            | 3 天   | 2.3         | Reranker 批处理（20 条/批）               |
| **2.7 性能测试**              | 3 天   | 所有        | 压力测试 + 性能基准                      |
| **2.8 监控 Dashboard**        | 2 天   | 2.7         | Grafana 面板（延迟、缓存命中率）         |

**里程碑**: 查询延迟 < 500ms P95，缓存命中率 > 60%

---

#### Phase 3: 智能日程增强（3 周）

**目标**: 完善日程管理功能（重复规则、智能建议、批量操作）

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **3.1 RRULE 解析器**          | 5 天   | 无          | 重复规则解析和扩展                        |
| **3.2 智能时间建议**          | 4 天   | 无          | 基于历史习惯的空闲时间推荐                |
| **3.3 冲突检测增强**          | 3 天   | 3.2         | 多时间段冲突检测 + AI 建议                |
| **3.4 批量操作 API**          | 3 天   | 无          | 批量创建/修改/删除日程                    |
| **3.5 前端日历视图**          | 4 天   | 3.1         | FullCalendar 集成（月/周/日视图）         |
| **3.6 日程负载分析**          | 3 天   | 3.5         | 热力图、甘特图可视化                     |
| **3.7 测试和文档**            | 2 天   | 所有        | 集成测试 + 用户指南                      |

**里程碑**: 用户可以创建重复日程，获取智能时间建议

---

#### Phase 4: AI 总结与增强（3 周）

**目标**: 实现 AI 笔记总结、日程总结、相关推荐

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **4.1 笔记总结 API**          | 3 天   | 无          | `SummarizeMemos` API（长上下文）         |
| **4.2 日程总结 API**          | 2 天   | 无          | `SummarizeSchedules` API                  |
| **4.3 相关推荐**              | 3 天   | Phase 1     | 基于向量的相关笔记推荐                   |
| **4.4 前端集成**              | 4 天   | 4.1, 4.2    | 总结卡片、推荐组件                       |
| **4.5 Prompt 优化**           | 3 天   | 4.1, 4.2    | 多语言 Prompt 模板（中文/英文）           |
| **4.6 长上下文支持**          | 3 天   | 无          | 支持 100K+ tokens（DeepSeek）            |
| **4.7 测试和调优**            | 2 天   | 所有        | 质量评估 + A/B 测试                      |

**里程碑**: AI 可以生成高质量笔记和日程总结

---

#### Phase 5: FinOps 优化（2 周）

**目标**: 实现 FinOps 最佳实践，降低 AI 成本 40-50%

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **5.1 成本监控完善**          | 2 天   | 无          | `CostMonitor` + Prometheus 指标          |
| **5.2 语义缓存优化**          | 3 天   | Phase 2     | 语义缓存命中率 > 40%                     |
| **5.3 条件性 Reranker**       | 2 天   | 无          | 减少 40% Reranker 调用                   |
| **5.4 批处理优化**            | 2 天   | 无          | Embedding/Reranker 批处理                 |
| **5.5 成本报告**              | 2 天   | 5.1         | 每日/每周/每月成本报告                   |
| **5.6 成本告警**              | 1 天   | 5.1         | 超预算告警（>$10/天）                    |
| **5.7 文档和培训**            | 2 天   | 所有        | FinOps 最佳实践文档                      |

**里程碑**: AI 月成本降低至 $5-8，节省 40-50%

---

#### Phase 6: 可观测性与稳定性（2 周）

**目标**: 完善监控、日志、追踪，提升系统稳定性

| 任务                          | 工作量 | 依赖        | 交付物                                   |
| ----------------------------- | ------ | ----------- | ---------------------------------------- |
| **6.1 OpenTelemetry 集成**    | 3 天   | 无          | 分布式追踪（Jaeger）                     |
| **6.2 结构化日志**            | 2 天   | 无          | slog + Loki 集成                         |
| **6.3 告警规则**              | 2 天   | 6.1, 6.2    | Prometheus AlertManager                  |
| **6.4 错误追踪**             | 2 天   | 无          | Sentry 集成（前端错误）                  |
| **6.5 健康检查**              | 1 天   | 无          | `/healthz` 端点                          |
| **6.6 性能基准**              | 2 天   | 所有        | 性能基准测试 + 报告                      |

**里程碑**: 系统可观测性完善，可快速定位问题

---

### 6.2 时间线

```
2026-01-22 ──────────────────────────────────────────────────────► 2026-06-30
  │         │         │         │         │         │         │
  │ Phase 1 │ Phase 2 │ Phase 3 │ Phase 4 │ Phase 5 │ Phase 6 │
  │ 4 周    │ 3 周    │ 3 周    │ 3 周    │ 2 周    │ 2 周    │
  ▼         ▼         ▼         ▼         ▼         ▼         ▼
1/22      2/19      3/12      4/02      4/23      5/14      5/28
```

**总工期**: 17-18 周（约 4.5 个月）

**并行优化**: Phase 2 和 Phase 3 可部分并行（缓存和日程功能独立）

---

### 6.3 资源规划

#### 人力资源

| 角色               | 投入比 | 说明                                   |
| ------------------ | ------ | -------------------------------------- |
| **后端工程师**     | 100%   | 核心开发（Go、PostgreSQL、AI 集成）     |
| **前端工程师**     | 60%    | React 组件、UI/UX、API 集成            |
| **DevOps 工程师**  | 30%    | 基础设施、监控、部署自动化              |
| **测试工程师**     | 40%    | 单元测试、集成测试、性能测试            |

#### 基础设施

| 资源               | 配置           | 成本/月          | 说明               |
| ------------------ | -------------- | ---------------- | ------------------ |
| **应用服务器**     | 2C4G           | $20              | 后端 + 前端构建    |
| **PostgreSQL**     | 2C4G           | $15              | 主数据库           |
| **Redis**          | 1C2G           | $8               | 缓存（可选）       |
| **MinIO**          | 2C4G + 100GB   | $12              | 对象存储          |
| **S3**             | -              | $5               | 备用（按需）       |
| **监控**           | -              | $0               | 自建（可选 $10）   |
| **AI API**         | -              | $5-8             | 优化后成本         |
| **总计**           | -              | **$65-83/月**    | 包含基础设施       |

---

## 7. 验收标准

### 7.1 功能验收

| 功能               | 验收标准                                                             |
| ------------------ | -------------------------------------------------------------------- |
| **附件上传**       | 支持图片/PDF/Word，自动 OCR，全文检索准确率 > 90%                     |
| **混合检索**       | 查询延迟 < 500ms P95，准确率 > 85%（人工标注 100 条测试集）           |
| **日程管理**       | 自然语言解析准确率 > 90%，冲突检测准确率 100%                         |
| **AI 总结**        | 总结质量评分 > 4.0/5.0（用户反馈）                                   |
| **缓存**           | 缓存命中率 > 60%，缓存一致性 100%                                    |

### 7.2 性能验收

| 指标               | 目标值          | 测试方法                                       |
| ------------------ | --------------- | ---------------------------------------------- |
| **查询延迟 P95**   | < 500ms         | 压力测试（100 QPS，10000 docs）                 |
| **向量检索延迟**   | < 100ms         | pgvector 查询（Top 10）                        |
| **并发 QPS**       | > 1000 QPS      | wrk 压测（2C4G 配置）                          |
| **缓存命中率**     | > 60%           | Prometheus 指标（7 天平均值）                   |
| **可用性**         | > 99.5%         | 生产环境监控（月度）                           |

### 7.3 成本验收

| 项目               | 目标值        | 测量方法                          |
| ------------------ | ------------- | --------------------------------- |
| **AI 月成本**      | < $8/月       | CostMonitor 报告（月度）          |
| **缓存节省率**     | > 40%         | （优化前成本 - 优化后成本）/优化前 |
| **单次查询成本**   | < $0.001      | 总成本 / 总查询数                |

### 7.4 代码质量验收

| 指标               | 目标值         | 工具                          |
| ------------------ | -------------- | ----------------------------- |
| **单元测试覆盖率** | > 80%          | go test -cover               |
| **代码审查**       | 所有 PR 审查   | GitHub Pull Request          |
| **静态分析**       | 0 bug          | golangci-lint                |
| **文档完整性**     | 100% API 文档  | Protobuf + Markdown          |

---

## 8. 风险评估

### 8.1 技术风险

| 风险                      | 影响 | 概率 | 缓解措施                                       |
| ------------------------- | ---- | ---- | ---------------------------------------------- |
| **OCR 准确率低**          | 高   | 中   | 使用预训练中文模型（chi-sim），人工抽检         |
| **向量检索性能不足**      | 高   | 低   | HNSW 索引优化，增加 Redis 缓存                  |
| **AI API 限流**           | 中   | 中   | 实现速率限制，多供应商冗余                      |
| **缓存一致性**            | 中   | 中   | 写穿透 + 失效策略，定期校验                     |
| **存储成本增长**          | 低   | 高   | 定期清理过期附件，生命周期策略                  |

### 8.2 业务风险

| 风险                      | 影响 | 概率 | 缓解措施                                       |
| ------------------------- | ---- | ---- | ---------------------------------------------- |
| **用户需求变更**          | 中   | 高   | 迭代式开发，每月收集反馈                       |
| **性能不达标**            | 高   | 中   | 性能基准测试，预留优化时间（2 周）             |
| **成本超支**              | 中   | 中   | FinOps 监控，预算告警                          |
| **数据丢失**              | 高   | 低   | 每日备份（PG + MinIO），灾难恢复演练           |

### 8.3 进度风险

| 风险                      | 影响 | 概率 | 缓解措施                                       |
| ------------------------- | ---- | ---- | ---------------------------------------------- |
| **Phase 1 延期**          | 高   | 中   | OCR 可选（Phase 1.4 可延后）                   |
| **资源不足**              | 中   | 低   | 优先级排序，P0 功能优先                        |
| **技术难点**              | 中   | 中   | 技术预研（1 周），POC 验证                     |

---

## 附录

### A. 参考文档

| 文档                                                    | 链接（相对路径）                                    |
| ------------------------------------------------------- | -------------------------------------------------- |
| AI 实现方案                                             | `/docs/ai-implementation-plan.md`                   |
| 日程助手实现计划                                        | `/docs/schedule-assistant-implementation-plan.md`  |
| 项目结构说明                                           | `/docs/PROJECT_STRUCTURE.md`                        |

### B. 技术选型对比

#### 向量数据库选型

| 方案          | 优势                          | 劣势                      | 选择理由                  |
| ------------- | ----------------------------- | ------------------------- | ------------------------- |
| **pgvector**  | 原生集成 PostgreSQL，运维简单  | 性能略低于专用向量库       | 已有 PG，降低复杂度       |
| **Milvus**    | 高性能，支持多种索引          | 需要额外部署，学习曲线陡   | 增加运维成本              |
| **Pinecone**  | 全托管，自动扩展              | 成本高，数据隐私风险       | 不符合隐私优先原则        |

#### 对象存储选型

| 方案          | 优势                          | 劣势                      | 选择理由                  |
| ------------- | ----------------------------- | ------------------------- | ------------------------- |
| **MinIO**     | 自托管，S3 API 兼容           | 需要自己运维              | 数据主权，成本可控        |
| **AWS S3**    | 高可用，全球 CDN              | 成本高，数据在美国        | 备用方案                  |
| **阿里云 OSS** | 国内速度快                    | 成本较高                  | 国内用户可考虑            |

### C. 关键配置示例

#### PostgreSQL 配置（2C4G）

```ini
# postgresql.conf
shared_buffers = 1GB
effective_cache_size = 3GB
maintenance_work_mem = 256MB
work_mem = 16MB
max_connections = 100

# pgvector 配置
ivfflat.probes = 10
hnsw.ef_search = 100
```

#### Redis 配置（1C2G）

```ini
# redis.conf
maxmemory 512mb
maxmemory-policy allkeys-lru
save 900 1
save 300 10
```

#### AI Provider 配置

```bash
# .env
MEMOS_AI_ENABLED=true

# Embedding (SiliconFlow)
MEMOS_AI_EMBEDDING_PROVIDER=siliconflow
MEMOS_AI_EMBEDDING_MODEL=BAAI/bge-m3
MEMOS_AI_SILICONFLOW_API_KEY=sk-xxx

# Reranker (SiliconFlow)
MEMOS_AI_RERANK_PROVIDER=siliconflow
MEMOS_AI_RERANK_MODEL=BAAI/bge-reranker-v2-m3

# LLM (DeepSeek)
MEMOS_AI_LLM_PROVIDER=deepseek
MEMOS_AI_LLM_MODEL=deepseek-chat
MEMOS_AI_DEEPSEEK_API_KEY=sk-xxx

# 缓存配置
MEMOS_CACHE_L1_ENABLED=true
MEMOS_CACHE_L1_MAX_SIZE=1000
MEMOS_CACHE_L1_TTL=300s

MEMOS_CACHE_L2_ENABLED=true
MEMOS_CACHE_L2_ADDR=localhost:6379
MEMOS_CACHE_L2_TTL=3600s
```

---

## 结语

本重构方案基于 Memos 现有优势（智能查询路由、Adaptive Retrieval、轻量级架构），通过**渐进式重构**和**FinOps 驱动**，在**6-8 个月**内实现：

1. **功能完善**：附件管理、智能日程、AI 总结
2. **性能优化**：< 500ms P95，1000+ QPS
3. **成本降低**：月成本降低 40-50%（$15 → $7-10）
4. **可扩展性**：插件化架构，轻松添加邮件、任务等功能
5. **可观测性**：完整的监控、追踪、日志体系

**核心理念**：**个人智能助理 = 轻量级 + 智能化 + 可扩展**

---

*文档版本: v1.0*
*创建时间: 2026-01-22*
*作者: Claude (Anthropic)*
*审核: 待审核*
*状态: 草稿（待评审）*
