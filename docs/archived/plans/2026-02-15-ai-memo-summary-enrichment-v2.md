# AI 摘要生成与 Memo 内容增强统一架构方案 v2.0

> **版本**: v2.0（重新规划）
> **日期**: 2026-02-15
> **状态**: 草案，待评审
> **关联**: [ai-solid-refactoring-plan.md](./ai-solid-refactoring-plan.md)

---

## 目录

1. [背景与目标](#1-背景与目标)
2. [现状分析](#2-现状分析)
3. [架构设计](#3-架构设计)
4. [实施计划](#4-实施计划)
5. [风险与验证](#5-风险与验证)

---

## 1. 背景与目标

### 1.1 需求背景

Memos 笔记内容可能很长（数千字），在便签纸风格的 UI 上直接展示全文会造成 **视觉沙漠**（Visual Desert）——用户无法快速扫描和定位笔记。

### 1.2 目标

- **AI 摘要**: 由 LLM 生成 ≤200 字的笔记摘要，用于便签卡片 UI 展示
- **异步运行**: 摘要生成不阻塞笔记创建/编辑流程
- **Fallback 策略**: 在 AI 摘要生成前，提供优雅的降级展示
- **统一架构**: 与现有标签生成、标题生成能力统筹设计，符合 DRY + SOLID

### 1.3 与原方案 (v1.0) 的差异

| 方面 | v1.0（草案） | v2.0（本次） |
|------|-------------|-------------|
| 配置加载器 | 新建 `ai/configloader` | 重构现有 `orchestrator/prompts.go` |
| **Format 功能** | 本期实现 | ✅ 本期实现 |
| 标签功能 | 接入 Pipeline | **双轨制**：保留现有按钮 + Pipeline 自动增强 |
| Title Generator | 新建 `ai/title/` 包 | **渐进式重构**：先适配接口，再考虑拆包 |

---

## 2. 现状分析

### 2.1 现有 AI 能力

| 能力 | 位置 | 触发方式 | LLM 依赖 |
|------|------|---------|---------|
| **标签建议** | `ai/tags/` | 用户点击按钮 | 可选 |
| **标题生成** | `ai/title_generator.go` | 对话结束异步 | 必须 |
| **意图识别** | `ai/agents/llm_intent_classifier.go` | 路由时同步 | 必须 |
| **摘要生成** | ❌ 不存在 | — | — |

### 2.2 现有代码结构

```
ai/
├── core/llm/service.go      # LLM Service 接口（核心）
├── tags/                    # 标签建议（三层渐进式）
│   ├── suggester.go
│   ├── suggester_impl.go
│   ├── layer1_statistics.go
│   ├── layer2_rules.go
│   └── layer3_llm.go
├── title_generator.go       # 标题生成（单一文件）
├── agents/
│   ├── orchestrator/prompts.go  # YAML 配置加载
│   └── universal/parrot_factory.go
```

### 2.3 现有 API 端点

| 端点 | 触发方式 | 说明 |
|------|---------|------|
| `/api/v1/ai/suggest-tags` | 用户点击 | 现有标签推荐 |
| `/api/v1/ai/generate-title` | 对话结束 | 现有标题生成 |

### 2.4 架构痛点

```
问题：
1. TitleGenerator 在根包，tags 在子包，无统一抽象
2. 配置加载逻辑分散（orchestrator + parrot_factory）
3. truncate/snippet 逻辑在多处重复
4. 新增 Summary 会加剧碎片化
```

---

## 3. 架构设计

### 3.1 整体架构

```
用户创建/编辑 Memo
         │
         ▼
┌─────────────────────┐
│   Memo 保存到 DB     │
└──────────┬──────────┘
           │
           ▼ 异步触发
┌─────────────────────────────────────┐
│       Enrichment Pipeline            │
│  ┌─────────┐ ┌─────┐ ┌─────────┐  │
│  │ Summary │ │Tags │ │ Title   │  │
│  └────┬────┘ └──┬──┘ └────┬────┘  │
│       └─────────┼─────────┘        │
│          并行执行                    │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┐
        ▼             ▼
   memo_summary   memo_tags
     (新表)       (新表)
```

### 3.2 双轨制：标签增强

```
┌─────────────────────────────────────────────────────┐
│                   标签增强双轨制                     │
├─────────────────────────────────────────────────────┤
│                                                     │
│  现有模式（保留）           Pipeline 新增            │
│  ────────────────           ──────────────         │
│  用户点击按钮触发     VS     保存后自动触发          │
│  插入到内容中        VS     存储为建议元数据        │
│  用户主动选择        VS     侧边栏静默展示          │
└─────────────────────────────────────────────────────┘
```

### 3.3 包结构设计

```
ai/
├── enrichment/              # [NEW] 统一内容增强包
│   ├── enricher.go          # Enricher 接口 + EnrichmentResult
│   ├── pipeline.go          # Pipeline 编排器
│   └── pipeline_test.go
├── summary/                  # [NEW] 摘要生成
│   ├── summarizer.go        # Summarizer 接口
│   ├── summarizer_impl.go   # LLM 实现
│   ├── fallback.go          # Fallback 策略
│   └── summarizer_test.go
├── format/                   # [NEW] 内容格式化
│   ├── formatter.go         # Formatter 接口
│   ├── formatter_impl.go    # LLM 实现
│   └── formatter_test.go
├── tags/
│   ├── suggester.go         # [现有]
│   ├── suggester_impl.go    # [现有]
│   └── enricher_adapter.go  # [NEW] 适配 Pipeline
├── title_generator.go       # [重构] 保留文件，添加适配接口
└── configloader/            # [NEW] 统一配置加载器
    └── loader.go
```

### 3.4 核心接口

```go
// ai/enrichment/enricher.go
type EnrichmentType string
type Phase string

const (
    // Pre-save（同步，用户触发）
    EnrichmentFormat  EnrichmentType = "format"

    // Post-save（异步，自动触发）
    EnrichmentSummary EnrichmentType = "summary"
    EnrichmentTags    EnrichmentType = "tags"
    EnrichmentTitle   EnrichmentType = "title"
)

const (
    PhasePre  Phase = "pre_save"   // 同步，保存前
    PhasePost Phase = "post_save"  // 异步，保存后
)

// MemoContent 是增强器的统一输入
type MemoContent struct {
    MemoID  string
    Content string
    Title   string
    UserID  int32
}

// EnrichmentResult 是单个增强器的输出
type EnrichmentResult struct {
    Type    EnrichmentType
    Success bool
    Data    any
    Error   error
    Latency time.Duration
}

// Enricher 是内容增强器的统一接口
type Enricher interface {
    Type() EnrichmentType
    Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult
}
```

### 3.5 Pipeline 编排器

```go
// ai/enrichment/pipeline.go
type Pipeline struct {
    enrichers []Enricher
    timeout   time.Duration
}

// EnrichAll 并行执行所有增强器
func (p *Pipeline) EnrichAll(ctx context.Context, content *MemoContent) map[EnrichmentType]*EnrichmentResult
```

### 3.6 统一配置加载器

```go
// ai/configloader/loader.go
// 复用 orchestrator/prompts.go 的 readFileWithFallback
// 融合 parrot_factory.go 的目录遍历逻辑

type Loader struct {
    baseDir string
    cache   sync.Map
}

func (l *Loader) Load(subPath string, target any) error
func (l *Loader) LoadCached(subPath string, factory func() any) (any, error)
func (l *Loader) LoadDir(subDir string, factory func(path string) (any, error)) (map[string]any, error)
```

### 3.7 摘要服务

```go
// ai/summary/summarizer.go
type Summarizer interface {
    Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error)
}

type SummarizeRequest struct {
    MemoID  string
    Content string
    Title   string
    MaxLen  int    // 默认 200
}

type SummarizeResponse struct {
    Summary string
    Source  string  // "llm" | "fallback_first_para" | "fallback_truncate"
    Latency time.Duration
}
```

### 3.8 Fallback 三级降级

| 级别 | 策略 | 效果 |
|------|------|------|
| L1 | 首段提取 | ⭐⭐⭐ 语义完整 |
| L2 | 首句提取 | ⭐⭐ 主题明确 |
| L3 | Rune 截断 | ⭐ 保底展示 |

### 3.9 存储方案

```sql
-- memo_summary 表
CREATE TABLE memo_summary (
    memo_id    INTEGER PRIMARY KEY REFERENCES memo(id) ON DELETE CASCADE,
    summary    TEXT NOT NULL,
    source     VARCHAR(32) NOT NULL DEFAULT 'fallback_truncate',
    version    INTEGER NOT NULL DEFAULT 1,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- memo_tags 表（Pipeline 自动增强用）
CREATE TABLE memo_tags (
    id         SERIAL PRIMARY KEY,
    memo_id    INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
    tag        VARCHAR(64) NOT NULL,
    source     VARCHAR(32) NOT NULL DEFAULT 'pipeline',
    confidence FLOAT,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(memo_id, tag)
);
```

### 3.10 Store 层接口扩展

需要在 `store/driver.go` 添加以下方法：

```go
// memo_summary 需要
UpsertMemoSummary(ctx context.Context, upsert *UpsertMemoSummary) error
GetMemoSummary(ctx context.Context, memoID int32) (*MemoSummary, error)
BatchGetMemoSummaries(ctx context.Context, memoIDs []int32) (map[int32]*MemoSummary, error)

// memo_tags 需要
UpsertMemoTags(ctx context.Context, upsert *UpsertMemoTags) error
ListMemoTags(ctx context.Context, memoID int32) ([]*MemoTag, error)
DeleteMemoTags(ctx context.Context, memoID int32) error
```

新增文件：
- `store/memo_summary.go` - Summary CRUD 实现
- `store/memo_tags.go` - Tags CRUD 实现

---

## 4. 实施计划

### 4.1 阶段划分

```
┌─────────────────────────────────────────────────────────────┐
│  阶段 1: 基础设施（配置加载器 + Pipeline 接口）             │
├─────────────────────────────────────────────────────────────┤
│  S1.1: 创建 ai/configloader/loader.go                       │
│  S1.2: 迁移 orchestrator/prompts.go 使用 configloader      │
│  S1.3: 创建 ai/enrichment/enricher.go 接口                 │
│  S1.4: 创建 ai/enrichment/pipeline.go 编排器               │
├─────────────────────────────────────────────────────────────┤
│  阶段 2: 摘要功能                                           │
├─────────────────────────────────────────────────────────────┤
│  S2.1: 创建 ai/summary/summarizer.go 接口                  │
│  S2.2: 创建 ai/summary/fallback.go 三级降级                │
│  S2.3: 创建 ai/summary/summarizer_impl.go LLM 实现         │
│  S2.4: 创建 config/prompts/summary.yaml                    │
├─────────────────────────────────────────────────────────────┤
│  阶段 2b: 格式化功能                                        │
├─────────────────────────────────────────────────────────────┤
│  S2b.1: 创建 ai/format/formatter.go 接口                  │
│  S2b.2: 创建 ai/format/formatter_impl.go LLM 实现         │
│  S2b.3: 创建 config/prompts/format.yaml                   │
│  S2b.4: 复用现有 AIFormatButton，集成 API                 │
│  S2b.5: Format API 端点 (POST /api/v1/ai/format)          │
├─────────────────────────────────────────────────────────────┤
│  阶段 3: 存储层                                             │
├─────────────────────────────────────────────────────────────┤
│  S3.1: 创建 DB 迁移 (memo_summary)                          │
│  S3.2: 创建 store/memo_summary.go CRUD                     │
├─────────────────────────────────────────────────────────────┤
│  阶段 4: Pipeline 集成                                       │
├─────────────────────────────────────────────────────────────┤
│  S4.1: 创建 ai/tags/enricher_adapter.go 适配器             │
│  S4.2: 重构 title_generator.go 添加适配接口                 │
│  S4.3: 创建 server/ 异步触发逻辑                            │
│  S4.4: API 集成 (GET /api/v1/memos/{id}/summary)          │
├─────────────────────────────────────────────────────────────┤
│  阶段 5: 标签双轨制                                         │
├─────────────────────────────────────────────────────────────┤
│  S5.1: 创建 DB 迁移 (memo_tags)                            │
│  S5.2: 创建 store/memo_tags.go CRUD                        │
│  S5.3: 前端侧边栏展示推荐标签                              │
├─────────────────────────────────────────────────────────────┤
│  阶段 6: 测试验收                                           │
├─────────────────────────────────────────────────────────────┤
│  S6.1: 单元测试                                             │
│  S6.2: 集成测试                                             │
│  S6.3: E2E 测试                                            │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 工作量估算

| 任务 | 预估 | 依赖 |
|------|------|------|
| S1.1 配置加载器 | 0.5d | 无 |
| S1.2 迁移 orchestrator | 0.5d | S1.1 |
| S1.3 Enricher 接口 | 0.25d | 无 |
| S1.4 Pipeline 编排器 | 0.25d | S1.3 |
| S2.1 Summary 接口 | 0.25d | S1.4 |
| S2.2 Fallback | 0.25d | S2.1 |
| S2.3 LLM 实现 | 0.5d | S2.2 |
| S2.4 Prompt 配置 | 0.25d | S1.1 |
| S2b.1 Format 接口 | 0.25d | S1.3 |
| S2b.2 Format LLM 实现 | 0.5d | S2b.1 |
| S2b.3 Format Prompt | 0.25d | S1.1 |
| S2b.4 前端按钮集成 | 0.5d | 无 |
| S2b.5 Format API | 0.25d | S2b.2 |
| S3.1 DB 迁移 | 0.25d | 无 |
| S3.2 Store 层 | 0.5d | S3.1 |
| S4.1 Tags 适配器 | 0.25d | S1.3 |
| S4.2 Title 重构 | 0.25d | S1.3 |
| S4.3 异步触发 | 0.5d | S2.3, S3.2 |
| S4.4 API 集成 | 0.5d | S4.3 |
| S5.1 DB 迁移 | 0.25d | 无 |
| S5.2 Store 层 | 0.25d | S5.1 |
| S5.3 前端 | 1d | S5.2 |
| S6 测试 | 1.5d | 以上全部 |

**总计**: ~9.5 工作日（包含 Format 功能）

---

## 5. 风险与验证

### 5.1 技术风险

| 风险 | 概率 | 缓解措施 |
|------|------|---------|
| LLM 延迟过高 | 中 | 异步执行 + Fallback |
| 大量 Memo 同时触发 | 中 | 限流 + 排队 |
| 合并冲突 | 低 | 按文件边界分派 |

### 5.2 验证命令

```bash
# 编译
go build ./...

# 单元测试
go test ./ai/enrichment/... -v
go test ./ai/summary/... -v

# Lint
go vet ./ai/...

# 全量测试
go test ./... -count=1
```

---

## 附录：与现有功能的关系

| 功能 | 处理方式 |
|------|---------|
| AI 标签按钮 | 完全保留，功能不变 |
| 三层标签推荐 | 保留，现有触发方式不变 |
| 标题生成 | 适配 Pipeline 接口，继续工作 |
| **格式按钮** | ✅ 本期实现 |

## 附录：Format 格式化功能设计

### 功能定位

用户点击编辑器工具栏中的"格式化"按钮，将随意文本转换为标准 Markdown 格式。

### 与 Summary 的对比

| 维度 | Format（格式化） | Summary（摘要） |
|------|----------------|----------------|
| **触发方式** | 用户手动点击按钮 | 保存后自动触发 |
| **执行方式** | 同步，实时返回 | 异步执行 |
| **修改对象** | 编辑器内容（未持久化） | 附加元数据 |
| **失败策略** | 提示用户，保留原文 | Fallback 三级降级 |
| **温度** | 0.1（保守，忠实原文） | 0.3（适度创意） |

### API 设计

```
POST /api/v1/ai/format
Request: { "content": "用户输入的原始文本" }
Response: { "formatted": "格式化后的 Markdown", "changed": true/false }
```

### 前端交互

**现有组件**：`web/src/components/MemoEditor/components/AIFormatButton.tsx`（已存在）

1. 用户在编辑器中输入内容
2. 点击工具栏"格式化"按钮（AIFormatButton）
3. 同步调用 API，返回格式化结果
4. 替换编辑器内容，用户可预览
5. 用户决定是否保存

### 侧边栏推荐标签（Pipeline 新增）

需要在 `MemoDetail` 或侧边栏组件中新增展示区域：
- 读取 `memo_tags` 表数据
- 展示 Pipeline 自动生成的标签建议
- 用户可点击采纳或忽略

---

*文档状态：已通过技术复核，待评审*
