# CLAUDE.md

> Claude Code 辅助开发的主要上下文文档。

## 产品愿景

**DivineSense (神识)**：AI 代理驱动的个人「第二大脑」—— 通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力。

---

## 快速开始

**默认端口**：`make start` → localhost:25173 (前端) / 28081 (后端) / 25432 (PostgreSQL)

| 命令             | 作用                                  |
| :--------------- | :------------------------------------ |
| `make start`     | 启动全栈 (PostgreSQL + 后端 + 前端)   |
| `make stop`      | 停止所有服务                          |
| `make test`      | 运行后端测试                          |
| `make build-all` | 构建二进制 + 静态资源                 |
| `make check-all` | 运行所有预提交检查 (构建、测试、i18n) |

**技术栈**：Go 1.25 + React 18 (Vite/Tailwind 4) + PostgreSQL (生产) / SQLite (开发)

---

## 核心规则

### 1. Git 工作流 — 严格执行

- **强制分支开发**：禁止直接在 `main` 修改，分支命名: `feat/`, `fix/`, `evolution/`
- **强制 PR**：所有变更通过 Pull Request 合并，禁止 `git push origin main`
- **提交前检查**：`make check-all` + `golangci-lint run` 通过后才允许提交

详细规范 → @.claude/rules/git-workflow.md

### 2. 国际化 (i18n)

所有 UI 文本必须使用 `t("key")`，禁止硬编码。验证: `make check-i18n`

详细规范 → @.claude/rules/i18n.md

### 3. 代码风格

- **Go**：`snake_case.go`，使用 `log/slog`
- **React**：PascalCase 组件，`use` 前缀 Hooks
- **Tailwind v4 陷阱**：禁用 `max-w-md`，使用 `max-w-[24rem]`

详细规范 → @.claude/rules/code-style.md

### 4. Lint 工作流

**前端** (在 `web/` 目录下运行):
```bash
pnpm lint       # 检查 TypeScript + Biome 格式
pnpm lint:fix   # 自动修复格式问题
```

**后端**:
```bash
go fmt ./...    # 格式化 Go 代码
go vet ./...    # 静态分析
```

**Pre-commit Hook**:
- 已配置 `.git/hooks/pre-commit` 自动运行格式检查
- 提交前自动执行：`go fmt` + `go vet` + `pnpm lint:fix`
- 如格式问题被自动修复，请 `git add` 变更后重新提交

### 5. 数据库策略

- **PostgreSQL**：生产环境，完整 AI 支持 (pgvector)
- **SQLite**：开发环境，**不支持 AI 功能**

---

## 文档索引

| 领域     | 文件                                  | 参考时机              |
| :------- | :------------------------------------ | :-------------------- |
| **后端** | @docs/dev-guides/BACKEND_DB.md        | API、数据库、Docker   |
| **前端** | @docs/dev-guides/FRONTEND.md          | 布局、Tailwind、组件  |
| **架构** | @docs/dev-guides/ARCHITECTURE.md      | 项目结构、AI 代理     |
| **部署** | @docs/deployment/BINARY_DEPLOYMENT.md | 二进制部署、Geek Mode |
| **路径** | @docs/dev-guides/PROJECT_PATHS.md     | 项目目录结构速查      |
| **任务** | @docs/dev-guides/COMMON_TASKS.md      | 常见开发任务步骤      |
| **环境** | @.env.example                         | 环境变量配置          |

---

## 元认知系统

DivineSense 对自身知识状态、检索质量、代理决策的监控与反思机制。

详细内容 → @docs/specs/META_COGNITION.md

**CLAUDE.md 自我进化原则**：本文档应随项目演进自动更新：
- 新增 AI 代理 → 更新文档索引
- 新增 Make 命令 → 更新快速开始
- 架构变更 → 同步更新相关章节

---

## 调试经验

<details>
<summary>📋 历史调试记录 (点击展开)</summary>

记录开发过程中遇到的典型问题和解决方案，避免重复踩坑：

- **Evolution Mode 路由失败** (2025-01): Protobuf JSON 序列化导致 `evolutionMode` 丢失
- **前端布局宽度不统一** (2025-01): Tailwind v4 语义化类名陷阱、组件内部宽度限制
- **Biome 格式检查失败** (2025-01): Import 顺序要求 react 第三方库优先、CSS 选择器逗号后需换行

详细内容 → @docs/research/DEBUG_LESSONS.md

</details>

---

## 产品功能

- **笔记**：Markdown 编辑 (KaTeX/Mermaid)、语义搜索、AI 标签
- **日程**：自然语言创建、冲突检测、多视图日历
- **AI 代理**：灰灰 (笔记) / 金刚 (日程) / 惊奇 (综合)
- **Geek Mode**：Claude Code CLI 直接集成

详细功能 → @README.md
