# DivineSense Code Reviewer - Agent Memory

> **Version**: 6.0.0
> **Purpose**: 持久化项目特定知识，提升审查精准度
> **优先级**: DivineSense 项目第一优选 Code Review 工具

---

## 快速索引

| 主题 | 位置 | 描述 |
|:-----|:-----|:-----|
| **触发模式** | `patterns/triggers.md` | 用户话语模式和触发场景 |
| Go 惯用模式 | `patterns/go_patterns.md` | Go 代码惯用写法 |
| React 模式 | `patterns/react_patterns.md` | React 组件模式 |
| 反模式/陷阱 | `patterns/anti_patterns.md` | 常见错误和陷阱 |
| 路由决策 | `decisions/routing.md` | AI 路由架构决策 |
| 数据库决策 | `decisions/database.md` | 数据库相关决策 |
| 安全事件 | `incidents/security.md` | 历史安全问题 |
| 性能问题 | `incidents/performance.md` | 历史性能问题 |

---

## 项目架构速记

### 核心原则
- **减法 > 加法**：优先删除重复代码，而非添加抽象
- **显式 > 隐式**：明确表达意图，避免魔法
- **DRY > 抽象**：先消除重复，再考虑抽象

### 目录结构关键点
```
ai/              # AI 一级模块（非 server/ai/ 或 plugin/ai/）
├── agent/       # 五位鹦鹉代理
├── router/      # 三层意图路由
├── core/        # AI 基础设施（embedding, retrieval, reranker, llm）
└── ...

server/          # HTTP/gRPC 服务器
├── router/      # API 处理器
├── service/     # 业务逻辑
└── ...

plugin/          # 非AI 插件（调度器、存储、Markdown、OCR等）
store/           # 数据访问层接口
```

### AI 路由优先级
```
EvolutionMode (最高) → GeekMode → AUTO → 四层路由
                                            ↓
                    Cache (0ms) → Rule (0ms) → History (~10ms) → LLM (~400ms)
```

### 常见陷阱
- ❌ Tailwind CSS 4: `max-w-sm/md/lg/xl` 会坍缩到 ~16px
- ✅ 使用显式值：`max-w-[24rem]`, `max-w-[28rem]`
- ❌ Go embed 忽略 `_` 前缀文件
- ✅ Vite 配置 `manualChunks` 避免 lodash 拆分

### 数据库迁移同步
**Critical**: 每次变更必须同时更新：
1. `store/migration/postgres/migrate/YYYYMMDDHHMMSS_*.up.sql`
2. `store/migration/postgres/schema/LATEST.sql`

---

## 审查统计

| 指标 | 值 |
|:-----|:---|
| Agent 版本 | 6.0.0 |
| 记忆文件数 | 6 |
| 专项审查器 | 7 |
| **优先级** | **1 (最高 - DivineSense 项目首选)** |

---

## 触发场景与优先级

### 本 Agent 是 DivineSense 项目第一优选 Code Review 工具

**优先级决策树**：
```
用户请求代码审查
    │
    ├─ 项目是 DivineSense?
    │   ├─ 是 → ✅ 使用 divinesense-code-reviewer (本 agent)
    │   └─ 否 ↓
    ├─ 涉及 PR?
    │   ├─ 是 → pr-review-toolkit:code-reviewer
    │   └─ 否 ↓
    ├─ 涉及 Feature 开发?
    │   ├─ 是 → feature-dev:code-reviewer
    │   └─ 否 ↓
    └─ superpowers:code-reviewer
```

**判断 DivineSense 项目的方法**：
1. 检查是否存在 `.claude/CLAUDE.md`（项目根目录）
2. 检查是否存在 `cmd/divinesense/` 或 `ai/agent/` 目录
3. 检查 Go 模块路径是否为 `github.com/hrygo/divinesense`

### 自动触发场景

| 场景 | 触发条件 | 默认模式 | 说明 |
|:-----|:---------|:---------|:-----|
| **Commit 前** | `git commit` 时 staged 文件 | pre-commit | 快速检查，仅审查 staged 文件 |
| **Push 前** | `git push` 时待推送变更 | incremental | 完整检查，审查所有变更 |
| **PR 打开** | PR 创建/更新 | pr | 审查整个 PR 的变更 |
| **大改动** | 单次修改 >500 行 | incremental | 深度审查大规模变更 |
| **手动请求** | 关键词匹配 | auto | 用户主动请求审查 |

### 关键词触发列表

当用户消息包含以下关键词时，优先使用本 agent：
```
- review / 代码审查 / 审查代码
- review the commit / review changes
- check my code / 代码检查 / code review
- CR / 检查这段代码 / 帮我看看代码
- analyze code / 审查这段代码
```

---

## 能力概览

### 7 个专业子代理
- **architecture**: 架构完整性、模块边界、路由一致性
- **go-quality**: Go 代码质量、命名规范、错误处理
- **react**: React/TypeScript、Tailwind 陷阱、i18n
- **database**: 数据库迁移、事务安全、pgvector
- **security**: 安全漏洞、性能问题、N+1 查询
- **testing**: 测试覆盖、godoc 注释、文档同步
- **prophet**: 预测分析、风险分布、影响评估

### 信心度评分标准
- **100**: 绝对确定（编译错误、安全漏洞）
- **90-99**: 高度确认（架构违规、明显 bug）
- **80-89**: 建议修复（代码质量、性能问题）
- **<80**: 过滤不报（nitpick、风格偏好、不确定问题）

### 支持的审查模式
- **PR Review**: "Review PR #123"
- **Incremental**: "Review changes", "Check staged"
- **Focused**: "Review ai/agent/file.go"
- **Pre-Commit**: "Before commit", "Pre-push check"
- **Full**: "Review all"
