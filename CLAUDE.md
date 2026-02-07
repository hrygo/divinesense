# CLAUDE.md

> DivineSense 项目开发纲领 — Claude Code 辅助开发的核心指导文档
>
> **保鲜状态**: ✅ 2026-02-07 v0.93.1 | **架构**: Go + React 单二进制 | **AI**: 五位鹦鹉代理

---

## 🎯 第一性原理

**DivineSense (神识)** = AI 代理驱动的个人「第二大脑」

```
核心使命：通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

技术本质：Go 后端 + React 前端的单二进制分发应用
架构核心：五位「鹦鹉」AI 代理 + 统一块模型 (Unified Block Model)
```

---

## 🧠 SOTA Agent 工程实践

> **本节定义大模型 Agent 的核心工作范式，确保与 SOTA 能力对齐**

### 思考协议 (Thinking Protocol)

**显式思考 > 隐式推理**：在复杂决策前，先输出思考过程。

```
任务 → 分析 → 方案 → 执行 → 验证
         ↑         ↓
         └── 修订 ──┘
```

**何时显式思考**：
- ✅ 架构变更、影响多个模块
- ✅ 陌生领域或不确定的 API
- ✅ 需要用户确认的方案
- ❌ 单一文件的简单修改
- ❌ 明确的 lint 错误修复

### 工具使用策略

| 工具 | 使用场景 | 避免使用 |
|:-----|:---------|:---------|
| `Task(Explore)` | 理解代码库结构、寻找文件模式 | 查找具体文件（用Glob） |
| `Task(Plan)` | 实现方案设计、多步骤任务 | 单一bug修复 |
| `AskUserQuestion` | 架构决策、多个可行方案 | 技术细节选择 |
| `TaskCreate/Update` | 3+个子任务、>1小时工作 | 单一直接任务 |
| `Bash` | git操作、测试、构建 | 文件操作（用专用工具） |

**核心原则**：
- 优先使用专用工具（Glob > grep, Read > cat, Edit > sed）
- 并行调用独立工具减少延迟
- 探索性任务用Task工具，精确操作用专用工具

### 元认知：卡住时的应对

```
┌─────────────────────────────────────────┐
│  遇到问题？遵循此流程（自愈协议）        │
├─────────────────────────────────────────┤
│  1. 重读问题 → 确保理解正确             │
│  2. 画图/拆解 → 可视化关系              │
│  3. 澄清歧义 → AskUserQuestion          │
│  4. 展示置信度 → 不确定时说明            │
│  5. 记录学习点 → 更新文档                │
└─────────────────────────────────────────┘
```

### SOTA 推理模式

| 模式 | 适用场景 | 实现 |
|:-----|:---------|:-----|
| **Chain-of-Thought** | 复杂逻辑推理 | 先输出分析步骤，再给结论 |
| **ReAct** | 工具调用任务 | Thought → Action → Observation 循环 |
| **Self-Refinement** | 代码生成 | 初稿 → 自审 → 修正 |
| **Few-Shot** | 格式化输出 | 给出2-3个示例 |

---

## 🏗️ 架构原则

### 核心概念映射

| 概念     | 实体             | 关系                   |
| :------- | :--------------- | :--------------------- |
| **对话** | `AIConversation` | 包含多个 Block         |
| **块**   | `AIBlock`        | 一个用户-AI 交互轮次   |
| **代理** | `ParrotAgent`    | 处理用户请求的 AI 实体 |
| **路由** | `ChatRouter`     | 决定使用哪只鹦鹉       |

### 关键架构决策（常混淆）

| 决策点 | 误区 | 正确理解 |
|:-------|:-----|:---------|
| **BlockMode vs AgentType** | 认为有映射关系 | 两者独立：Mode是结构模式，AgentType是处理者 |
| **AUTO 的本质** | 是一只鹦鹉 | 是"请后端路由"的标记，非鹦鹉 |
| **数据库选择** | SQLite可用于生产 | SQLite仅开发，生产需PostgreSQL |

**路由四层**（v0.93.1）：
```
用户输入 → Cache (0ms) → Rule (0ms) → History (~10ms) → LLM (~400ms)
           ↓              ↓            ↓               ↓
        LRU命中        关键词       对话上下文      Qwen2.5-7B
```

---

## 🔄 工作流

### 多任务管理（TODO LIST）

> **原则**：始终使用 TODO LIST 跟踪多任务状态，避免"失忆"或迷失方向。

**何时创建**：
- 发现**3+**个优化点
- 用户要求"逐个击破"
- 任务预计 > 1 小时

**操作流程**：
```
TaskCreate("标题", "描述") → TaskList → TaskUpdate(id, in_progress)
                                                        ↓
                                               TaskUpdate(id, completed)
```

**状态流转**：
```
pending → in_progress → completed
    ↓                      ↓
  (开始)                (完成)
```

### 开发命令速查

> **⚠️ 重要：始终优先使用 `make` 命令**
>
> | 错误操作 | 正确操作 | 原因 |
> |:---------|:---------|:-----|
> | `pnpm build`（根目录）| `make build-web` | `package.json` 在 `web/` 下 |
> | `docker exec divinesense-postgres` | `make db-shell` | 容器名自动检测 |
> | `cd web && pnpm dev` | `make web` | Makefile 处理目录切换 |

| 阶段 | 命令 | 说明 |
|:-----|:-----|:-----|
| **启动** | `make start` | 全栈服务（DB + 后端 + 前端） |
| **前端** | `make web` / `make build-web` | 启动 dev server / 构建 |
| **数据库** | `make db-shell` / `make db-connect` | 连接 PostgreSQL（自动检测容器） |
| **检查** | `make check-all` | 提交前完整检查 |
| **CI** | `make ci-check` | 模拟 CI 环境 |
| **测试** | `make test-ai` | AI 相关测试 |

### 提交流程
```
1. make check-all 通过
2. 分支命名：feat/xxx、fix/xxx、evolution/xxx
3. 禁止直接 push 到 main
4. 通过 PR 合并
```
详细规范：@.claude/rules/git-workflow.md

---

## 📐 编码规范

### 核心原则

> **减法 > 加法**：优先通过删除重复代码、合并相似功能来优化架构，而非添加新的抽象层。

| 原则 | 简记 | 实践 |
|:-----|:-----|:-----|
| **DRY** | 不重复 | 提取公共逻辑，v0.93.1删除492行重复代码 |
| **SOLID-S** | 单一职责 | 每个模块只做一件事 |
| **SOLID-O** | 开闭原则 | 扩展开放，修改封闭 |
| **SOLID-D** | 依赖倒置 | 依赖接口而非实现 |

```go
// ✅ DIP + ISP：依赖抽象，接口隔离
type LLMClient interface {
    Complete(ctx, prompt, config) (string, error)
}

// 可替换的实现
type routerLLMClient struct{ llm LLMService }
type routerIntentLLMClient struct{ apiKey, baseURL, model string }
```

### 语言规范

| Go | React/TypeScript | Tailwind v4 |
|:---|:-----------------|:------------|
| `snake_case.go` | `PascalCase.tsx` | ❌ `max-w-md` → ✅ `max-w-[24rem]` |
| `log/slog` | `use` 前缀 | 显式值避免~16px解析错误 |
| 始终检查错误 | `t("key")` 国际化 | |

---

## 📚 导航索引

| 任务 | 文档 |
|:-----|:-----|
| **理解架构** | @docs/dev-guides/ARCHITECTURE.md |
| **后端开发** | @docs/dev-guides/BACKEND_DB.md |
| **前端开发** | @docs/dev-guides/FRONTEND.md |
| **部署** | @docs/deployment/BINARY_DEPLOYMENT.md |
| **调试问题** | @docs/research/DEBUG_LESSONS.md |
| **数据库迁移** | @store/migration/postgres/CLAUDE.md |

---

## 🎯 产品能力边界

| 功能 | 状态 |
|:-----|:-----|
| 笔记 | ✅ Markdown + 语义搜索 |
| 日程 | ✅ 自然语言 + 冲突检测 |
| AI 代理 | ✅ 五位鹦鹉协同 |
| Geek Mode | ✅ Claude Code CLI 集成 |
| Evolution Mode | ✅ 系统自我进化 |

---

*本文档随项目演进自动更新。新增功能时同步更新架构原则和导航索引。*
