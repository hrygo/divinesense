# 🤝 Contributing to DivineSense

感谢你对 DivineSense 项目的关注！我们要打造的是一个 premium、aesthetic 且充满活力的 AI Native 应用。
本文档旨在帮助你以 minimal friction (零摩擦) 的方式开始贡献。

---

## 🚀 Quick Start (零基础入门)

### 1. 环境准备 (Prerequisites)

确保你的开发环境已安装以下工具：

- **Go**: >= 1.22
- **Node.js**: >= 20 (推荐使用 `fnm` 或 `nvm` 管理)
- **pnpm**: >= 9 (`npm install -g pnpm`)
- **Docker**: 用于运行本地数据库和 AI 服务
- **Make**: 构建工具 (Windows 用户请使用 WSL2 或 Git Bash)

### 2. 启动项目 (Setup & Run)

我们封装了完善的 `Makefile` 指令，让你一键启动环境。

```bash
# 1. 克隆项目
git clone https://github.com/hrygo/divinesense.git
cd divinesense

# 2. 安装所有依赖 (Backend + Frontend)
make deps-all

# 3. 安装 Git Hooks (Required) ⚠️
# 这将安装 pre-commit 和 pre-push 钩子，确保你的提交符合规范
make install-hooks

# 4. 启动基础设施 (PostgreSQL Docker)
make docker-up

# 5. 启动开发服务 (后端 + 前端)
# 访问 http://localhost:25173
make start
```

> **Tip**: 如果你需要实时查看详细的合并日志，可以使用 `make dev-logs-follow`。

---

## 📂 Project Structure (项目地图)

熟悉项目结构有助于你快速定位代码：

- **`cmd/`**: 应用程序入口 (Server)。
- **`web/`**: 前端 React 应用 (Vite + TailwindCSS + Radix UI)。
- **`internal/`**: 私有业务逻辑。
- **`store/`**: 数据持久层 & 数据库迁移 (`store/migration/`).
- **`plugin/`**: 插件系统 (Go Plugin).
- **`.agent/`**: AI Agent 技能与工作流定义 (Workflows & Skills)。
- **`.claude/`**: AI 助手配置 (Rules & Skills)。
  - **`rules/`**: AI 行为准则 (e.g., `git-workflow.md`, `i18n.md`, `code-style.md`)。
  - **`skills/`**: 增强能力 (e.g., `docs-manager` 文档管理, `idea-researcher` 创意调研)。
- **`deploy/`**: 部署脚本与 Docker 配置。

---

## 🛠 Development Workflow (开发工作流)

### 1. 分支策略 (Branching)

- **`main`**: 主分支，保持随时可发布状态。
- **`feature/<name>`**: 新功能开发。
- **`fix/<name>`**: Bug 修复。
- **`refactor/<name>`**: 代码重构。

### 2. 提交规范 (Commits)

我们遵循 **Conventional Commits** 规范，并要求提交**原子化**。

格式：`<type>(<scope>): <subject>`

- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `revert`.
- **Scope**: `web`, `server`, `store`, `api`, `ai` 等。
- **Example**: `feat(web): add dark mode support to settings page`

> **Note**: `make install-hooks` 会安装 Git Hooks 帮助你检查提交格式。

### 3. Agentic Workflows (AI 辅助)

本项目集成了 Agentic Workflow，你可以在 `.agent/workflows` 中找到定义好的工作流。

- **Upstream Analysis**: 使用 `/git-upstream-analysis` 可以自动分析上游代码变更（如果你在维护 Fork）。
- **Documentation**: 使用 `make docs-check` 调用 `docs-manager` 技能来维护文档一致性。

---

### 💎 Coding Standards (代码规范)

#### 🌐 Frontend (Web)

- **Stack**: React, TypeScript, TailwindCSS (v4), Radix UI.
- **Aesthetics**: 追求 Premium 设计。避免默认颜色，使用 HSL 定制色板。
- **Linting**: 
  ```bash
  cd web
  pnpm lint && pnpm format
  ```
- **Specific Rules (坑点注意)**:
  - **Tailwind v4**: 禁用 `max-w-md` 等语义化宽度，统一使用显式值如 `max-w-[24rem]` 以避免布局塌陷。
  - **Components**: 组件名必须 PascalCase。Hooks 必须以 `use` 开头。

#### 🔙 Backend (Go)

- **Style**: 遵循 Effective Go。文件命名使用 `snake_case.go`。
- **Linting**:
  ```bash
  make lint
  ```
- **Testing**:
  ```bash
  make test       # 运行所有测试
  make test-ai    # 仅运行 AI 插件测试
  ```

#### 🌍 Internationalization (i18n) - **CRITICAL**

**所有 UI 文本必须双语支持 (English & Simplified Chinese)。**

1.  **文件位置**: `web/src/locales/en.json` 和 `zh-Hans.json`。
2.  **流程**:
    -   在 `en.json` 添加 Key。
    -   在 `zh-Hans.json` 添加对应翻译。
    -   运行检查: `make check-i18n`。
3.  **禁止硬编码**: 前端代码中禁止直接写中/英文字符串，必须使用 `useTranslate` 或 `t()`。

#### 🗄 Database Strategy

- **Development**: 默认可以使用 PostgreSQL (推荐) 或 SQLite。
  - **PostgreSQL**: 生产环境标准，支持完整 AI 功能 (pgvector)。
  - **SQLite**: 仅限轻量开发，**不支持 AI 向量检索功能**。
- **Migrations**: 位于 `store/migration/postgres`。
  - 新增迁移需同时包含 `up` 和 `down` 逻辑。

---

## 📚 Documentation Index (进阶阅读)

项目中包含详细的开发文档，建议深入阅读：

- **项目首页**: [`README.md`](README.md) (产品愿景、功能特性)
- **架构设计**: [`docs/dev-guides/ARCHITECTURE.md`](docs/dev-guides/ARCHITECTURE.md)
- **后端指南**: [`docs/dev-guides/BACKEND_DB.md`](docs/dev-guides/BACKEND_DB.md) (API, DB, Docker)
- **前端指南**: [`docs/dev-guides/FRONTEND.md`](docs/dev-guides/FRONTEND.md) (Layouts, Components)
- **常见任务**: [`docs/dev-guides/COMMON_TASKS.md`](docs/dev-guides/COMMON_TASKS.md)

---

## ✅ Pull Request Process

在提交 PR 之前，请运行以下 "Quality Gate" 命令确保 CI 会通过：

```bash
# 运行完整的本地 CI 检查 (Backend + Frontend + Lint + Test + i18n)
make ci-check
```

1.  **Self-Review**: 检查代码风格，确保没有 debug print。
2.  **Screenshots**: 如果是 UI 变更，请在 PR 中附带 **截图或录屏**。
3.  **Description**: 清晰描述变更的 **Why** 和 **How**。

---

## 🧰 Makefile Cheat Sheet

| 命令                   | 描述             |
| :--------------------- | :--------------- |
| `make help`            | 显示所有可用命令 |
| `make deps-all`        | 安装所有依赖     |
| `make docker-up`       | 启动数据库容器   |
| `make start`           | 同时启动前后端   |
| `make test`            | 运行后端测试     |
| `make check-i18n`      | 检查多语言一致性 |
| `make ci-check`        | 运行全量 CI 检查 |
| `make dev-logs-follow` | 实时查看聚合日志 |

Happy Coding! 🚀
