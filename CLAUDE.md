# CLAUDE.md

> DivineSense 项目开发纲领 — Claude Code 辅助开发的核心指导文档
>
> **保鲜状态**: ✅ 2026-02-08 v0.93.2 | **架构**: Go + React 单二进制 | **AI**: 五位鹦鹉代理

---

## 🎯 第一性原理

**DivineSense (神识)** = AI 代理驱动的个人「第二大脑」

```
核心使命：通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

技术本质：Go 后端 + React 前端的单二进制分发应用
架构核心：五位「鹦鹉」AI 代理 + 统一块模型 (Unified Block Model)
```

---

## 🔑 Critical Context

> **快速参考** - 避免常见陷阱的关键信息

### 项目结构

| 目录 | 说明 | 关键点 |
|:-----|:-----|:-------|
| `web/` | 前端根目录 | **始终从此处运行前端命令** |
| `ai/` | AI 核心模块 | Go 代码，一级模块 |
| `server/` | HTTP/gRPC 服务器 | 路由、服务层 |
| `store/` | 数据访问层 | PostgreSQL/SQLite |
| `proto/` | Protobuf 定义 | 修改后需重新生成 |

### 关键配置

| 配置 | 值 | 注意 |
|:-----|:---|:-----|
| **PostgreSQL 容器名** | `divinesense-postgres-dev` | **不是** `divinesense-postgres` |
| **前端端口** | 25173 | 开发环境 |
| **后端端口** | 28081 | API 服务 |
| **数据库端口** | 25432 | 开发环境 |

### i18n 完整性陷阱

> ⚠️ `make check-i18n` **检查 en.json 和 zh-Hans.json 的同步**

**注意路径差异**：组件 `ProgressIndicator.tsx` 使用 `ai.progress.phases.*`，确保所有语言文件在正确路径下有翻译。

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
| `Task(并行多代理)` | **2+个独立子任务**（详见下方并行代理策略） | 串行依赖任务 |
| `AskUserQuestion` | 架构决策、多个可行方案 | 技术细节选择 |
| `TaskCreate/Update` | 3+个子任务、>1小时工作 | 单一直接任务 |
| `Bash` | git操作、测试、构建 | 文件操作（用专用工具） |

**核心原则**：
- 优先使用专用工具（Glob > grep, Read > cat, Edit > sed）
- **并行调用独立工具减少延迟**（关键优化点）
- 探索性任务用Task工具，精确操作用专用工具

---

### 🚀 并行代理执行策略 (Parallel Agent Execution)

> **核心目标**：识别可并行化的独立子任务，通过多代理协作实现 2-4x 的加速效果。

#### 何时启动并行代理

**启动条件**（满足任一即考虑）：
- ✅ 存在 **2+ 个无依赖关系** 的子任务
- ✅ 子任务涉及 **不同代码区域**（前端/后端/数据库独立）
- ✅ 子任务需要 **不同专业能力**（分析/搜索/测试/代码生成）
- ✅ 任务总耗时预计 > 10 分钟

**不适用场景**：
- ❌ 子任务间有强依赖（B 的输入依赖 A 的输出）
- ❌ 单一文件的简单修改
- ❌ 需要用户频繁确认的交互式任务

#### Claude Code 并行代理实现模式

**模式 1：单次消息多并行调用**
```text
用户: "同时完成以下独立任务：
1. 分析前端 AIBlock 组件架构
2. 检查后端 ai/agent/ 目录结构
3. 搜索数据库迁移文件模式"
```
Claude 将在单个响应中并行调用多个 Task/Explore 代理。

**模式 2：显式指定并行执行**
```text
用户: "并行执行以下任务：
- 运行 pr-test-analyzer 检查测试覆盖
- 运行 comment-analyzer 检查注释质量
- 运行 code-explorer 分析架构"
```

#### 实际应用示例

**场景 1：代码重构分析**
```
任务：分析 universal parrot 重构影响
├─ Agent A: 分析 ai/agent/universal/ 依赖关系
├─ Agent B: 检查 server/router/api/v1/ai/ 调用点
└─ Agent C: 搜索配置文件 config/parrots/ 引用
```
**加速效果**：串行 ~8 分钟 → 并行 ~3 分钟

**场景 2：PR 审查**
```
任务：全面审查变更
├─ Agent A: pr-test-analyzer（测试覆盖）
├─ Agent B: code-simplifier（代码简化）
├─ Agent C: comment-analyzer（注释质量）
└─ Agent D: silent-failure-hunter（错误处理）
```
**加速效果**：串行 ~15 分钟 → 并行 ~5 分钟

**场景 3：跨模块功能开发**
```
任务：添加新的 AI 工具
├─ Agent A: 设计工具接口（proto/）
├─ Agent B: 实现后端逻辑（ai/agent/tools/）
└─ Agent C: 更新前端 UI（web/src/components/）
```
**加速效果**：串行 ~20 分钟 → 并行 ~8 分钟

#### 并行代理最佳实践

| 实践 | 说明 |
|:-----|:-----|
| **明确边界** | 每个代理的任务边界清晰，无重叠 |
| **独立状态** | 避免共享可变状态，使用只读输入 |
| **结果聚合** | 指定主代理负责汇总各代理结果 |
| **超时控制** | 每个代理设置合理超时（默认 5-10 分钟） |
| **错误隔离** | 单个代理失败不影响其他代理执行 |

#### Claude Code 技能参考

DivineSense 项目已集成以下支持并行执行的技能：
- `superpowers:dispatching-parallel-agents` - 并行代理调度框架
- `pr-review-toolkit:review-pr` - 多代理并行 PR 审查
- `feature-dev:code-explorer` - 架构分析代理
- `pr-review-toolkit:code-simplifier` - 代码简化代理

**参考资源**：
- [Claude Code 并行代理指南](https://www.marc0.dev/en/blog/claude-code-agent-teams-multiple-ai-agents-working-in-parallel-setup-guide-1770317684454)
- [Anthropic 多代理研究系统](https://www.anthropic.com/engineering/multi-agent-research-system)
- [并行编码代理生活方式](https://simonwillison.net/2025/Oct/5/parallel-coding-agents/)

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

### 服务重启规范

> **⚠️ 关键规则：修改后端代码后，需要通知用户手动重启服务**
>
> **禁止直接执行** `make stop`、`make start`、`make run` 等服务启停命令。

**需要重启服务的场景**：
| 修改类型 | 是否需要重启 | 说明 |
|:---------|:-------------|:-----|
| **后端 Go 代码** | ✅ 是 | 任何 `*.go` 文件修改 |
| **前端代码** | ✅ 否 | Vite HMR 自动更新 |
| **数据库迁移** | ✅ 是 | 新增/修改 SQL 文件 |
| **配置文件** | ✅ 是 | `.env` 或系统配置 |
| **文档** | ❌ 否 | 不影响运行状态 |

**正确操作流程**：
1. 完成代码修改和构建
2. **通知用户**："后端代码已修改，请手动重启服务：`make restart`"
3. 等待用户确认重启
4. 继续后续工作

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

### 语言特定注意事项

#### TypeScript/React
- **Streaming UI**: 组件必须主动消费 `eventStream` - 优化 React Query 缓存前先验证数据流
- **State updates**: 使用 React 18 自动批处理，而非 `queryClient.batch`（不存在）
- **Block rendering**: "initializing" 卡住状态？检查流式内容是否实时更新 `assistantContent`

#### Go Backend
- **Error definitions**: 跨服务共享 - 删除前检查所有引用
- **Database queries**: 优化延迟时注意 N+1 模式
- **Migrations**: 确保所有表存在（参见 @docs/research/DEBUG_LESSONS.md → "7 个缺失表曾导致冷启动延迟"）

#### Go Lint 常见陷阱 (golangci-lint)

> ⚠️ **必须遵守的模式** - 这些问题在 pre-push 时会被拦截

| 问题 | ❌ 错误写法 | ✅ 正确写法 |
|:-----|:-----------|:-----------|
| **类型断言** | `v := x.(T)` | `v, ok := x.(T)` (comma-ok) |
| **defer 错误** | `defer resp.Body.Close()` | `defer func() { if err := resp.Body.Close(); err != nil { slog.Error(...) } }()` |
| **错误比较** | `err != expectedErr` | `errors.Is(err, expectedErr)` |
| **HTTP nil body** | `NewRequest("GET", url, nil)` | `NewRequest("GET", url, http.NoBody)` |
| **error 变量名** | `var testErr` | `var errTest` (以 `err` 开头) |
| **正则简化** | `[^\s]*` | `\S*` |
| **sync.Pool** | `pool.Put(slice)` | ❌ 不放 slice，只放指针 |

**核心原则**：
1. **所有错误返回值必须检查** - 不能用 `_` 忽略
2. **类型断言用 comma-ok** - 避免 panic
3. **defer 也要检查错误** - 即使是 Close()

---

## 🚫 Code Change Boundaries

> **边界约束** - 避免破坏性变更的规则

| 约束 | 说明 |
|:-----|:-----|
| **避免过度删除** | 删除代码前验证错误定义、类型导出、共享工具的所有使用 |
| **测试后提交** | 推送前从 `web/` 运行 `npm test` - pre-push 会捕获缺失依赖 |
| **Proto/Schema 变更** | Proto 定义变更后始终重新生成前后端绑定 |
| **批量重构** | 跨多文件变更 API 时，分阶段提交而非一次性大改 |

### Critical Workflow Patterns

1. **结构性变更前**：检查文件是否存在于多个位置（如 `web/src` vs `src`）避免目录错误
2. **删除代码时**：验证没有关键错误定义或类型依赖，先进行死代码分析
3. **流式/实时特性**：确保组件实际消费 `eventStream` 和 `data` - React Query 缓存问题常被误诊
4. **优化方法**：优先简单方案（CSS 移除、标志切换）而非架构变更

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
