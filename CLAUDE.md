# CLAUDE.md

> DivineSense 项目开发纲领 — Claude Code 辅助开发的核心指导文档
>
> **保鲜状态**: ✅ 2026-02-10 v0.97.0 | **架构**: Go + React 单二进制 | **AI**: 五位鹦鹉代理

---

## 🎯 项目本质

**DivineSense (神识)** = AI 代理驱动的个人「第二大脑」

```
技术栈：Go 后端 + React 前端（单二进制分发）
核心架构：五位「鹦鹉」AI 代理 + Unified Block Model
```

---

## 🔑 Critical Context（必读）

**详细内容**：@docs/essentials/CRITICAL_CONTEXT.md

### 目录速览
| 目录      | 说明                                    |
| :-------- | :-------------------------------------- |
| `web/`    | 前端根目录 — **始终从此处运行前端命令** |
| `ai/`     | AI 核心模块（Go 一级模块）              |
| `server/` | HTTP/gRPC 服务器                        |
| `store/`  | 数据访问层                              |
| `proto/`  | Protobuf 定义（修改后需重新生成）       |

### 关键配置
| 配置              | 值                         |
| :---------------- | :------------------------- |
| PostgreSQL 容器名 | `divinesense-postgres-dev` |
| 前端端口          | 25173                      |
| 后端端口          | 28081                      |
| 数据库端口        | 25432                      |

### 常见陷阱
| 陷阱                | 说明                                           |
| :------------------ | :--------------------------------------------- |
| `max-w-md` 等语义类 | Tailwind v4 解析为 ~16px，用 `max-w-[24rem]`   |
| i18n 不同步         | `make check-i18n` 检查 en.json 和 zh-Hans.json |
| 服务重启            | 修改后端代码后通知用户手动 `make restart`      |
| SQLite 无 AI        | 生产 AI 功能必须用 PostgreSQL                  |

---

## 🧠 Agent 工作范式

**工作协议**：@docs/dev-guides/AGENT_WORKFLOW.md

### 思考协议
```
任务 → 分析 → 方案 → 执行 → 验证
         ↑         ↓
         └── 修订 ──┘
```

### 工具选择
| 任务           | 工具            |
| :------------- | :-------------- |
| 理解代码库结构 | `Task(Explore)` |
| 实现方案设计   | `Task(Plan)`    |
| 查找具体文件   | `Glob`          |
| 搜索代码内容   | `Grep`          |
| 读取文件       | `Read`          |
| 编辑文件       | `Edit` / `Write` |

> **文件编辑**：连续 3 次 Edit 失败时，改用 `Read 完整文件 → Write 整体重写`。详见 @.claude/rules/file-editing.md

---

## 🏗️ 架构速览

**架构详情**：@docs/dev-guides/ARCHITECTURE_SUMMARY.md

### 五位鹦鹉
| 代理 | 角色 |
|:-----|:-----|
| MemoParrot (灰灰) | 笔记搜索 |
| ScheduleParrot (时巧) | 日程管理 |
| AmazingParrot (折衷) | 综合助理 |
| GeekParrot (极客) | Claude Code CLI |
| EvolutionParrot (进化) | 自我进化 |

### 路由四层
```
用户输入 → Cache → Rule → History → LLM (~400ms)
```

### 核心概念
- **Block**：用户-AI 交互轮次
- **Agent**：AI 代理处理器
- **Router**：意图路由系统

---

## 🔄 工作流

**工作规范**：@docs/dev-guides/WORKFLOW.md

### 开发命令
| 阶段   | 命令                          |
| :----- | :---------------------------- |
| 启动   | `make start`                  |
| 前端   | `make web` / `make build-web` |
| 数据库 | `make db-shell`               |
| 检查   | `make check-all`              |

### 提交流程
```
make check-all → feat/fix 分支 → PR → 合并
```
详细规范：@.claude/rules/git-workflow.md

---

## 📐 编码规范

**代码风格**：@.claude/rules/code-style.md

### 核心原则
> **减法 > 加法**：删除重复代码、合并相似功能

### 语言规范
| Go              | React/TS         | Tailwind v4        |
| :-------------- | :--------------- | :----------------- |
| `snake_case.go` | `PascalCase.tsx` | `max-w-[24rem]`    |
| `log/slog`      | `use` 前缀       | 显式值避免解析错误 |
| 检查错误        | `t("key")`       |                    |

---

## 📚 文档导航

| 任务       | 文档                                  |
| :--------- | :------------------------------------ |
| 理解架构   | @docs/dev-guides/ARCHITECTURE.md      |
| 后端开发   | @docs/dev-guides/BACKEND_DB.md        |
| 前端开发   | @docs/dev-guides/FRONTEND.md          |
| 部署       | @docs/deployment/BINARY_DEPLOYMENT.md |
| 调试问题   | @docs/research/DEBUG_LESSONS.md       |
| 数据库迁移 | @store/migration/postgres/CLAUDE.md   |

---

*本文档随项目演进自动更新。*
