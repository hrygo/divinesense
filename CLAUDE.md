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

### 项目结构
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

### 思考协议
```
任务 → 分析 → 方案 → 执行 → 验证
         ↑         ↓
         └── 修订 ──┘
```

**何时显式思考**：架构变更、陌生 API、需用户确认的方案
**何时直接执行**：单文件修改、明确 lint 错误

### 工具选择
| 任务           | 工具            |
| :------------- | :-------------- |
| 理解代码库结构 | `Task(Explore)` |
| 实现方案设计   | `Task(Plan)`    |
| 查找具体文件   | `Glob`          |
| 搜索代码内容   | `Grep`          |
| 读取文件       | `Read`          |
| 编辑文件       | `Edit`          |

**核心原则**：专用工具 > 通用工具，并行调用独立工具

### 并行代理策略
**启动条件**（满足任一）：
- 2+ 个无依赖关系的子任务
- 子任务涉及不同代码区域
- 任务预计 > 10 分钟

**不适用**：强依赖任务、单文件修改、频繁交互

---

## 🏗️ 架构速览

### 五位鹦鹉（内部代理）
| 鹦鹉                   | 领域                 |
| :--------------------- | :------------------- |
| MemoParrot (灰灰)      | 笔记搜索             |
| ScheduleParrot (时巧)  | 日程管理             |
| AmazingParrot (折衷)   | 综合助理             |
| GeekParrot (极客)      | Claude Code CLI 桥接 |
| EvolutionParrot (进化) | 自我进化             |

**路由四层**：Cache → Rule → History → LLM（~400ms）

### 关键概念
| 概念 | 实体             | 说明                 |
| :--- | :--------------- | :------------------- |
| 对话 | `AIConversation` | 包含多个 Block       |
| 块   | `AIBlock`        | 一个用户-AI 交互轮次 |
| 代理 | `ParrotAgent`    | 处理请求的 AI 实体   |
| 路由 | `ChatRouter`     | 决定使用哪只鹦鹉     |

**常混淆**：
- `BlockMode` vs `AgentType`：两者独立，Mode 是结构模式，AgentType 是处理者
- `AUTO` 不是鹦鹉：是"请后端路由"的标记

---

## 🔄 工作流

### 多任务管理
**何时创建 TODO LIST**：3+ 个优化点、任务 > 1 小时
```
TaskCreate → TaskList → TaskUpdate(in_progress) → TaskUpdate(completed)
```

### 开发命令
| 阶段   | 命令                          |
| :----- | :---------------------------- |
| 启动   | `make start`                  |
| 前端   | `make web` / `make build-web` |
| 数据库 | `make db-shell`               |
| 检查   | `make check-all`              |
| CI     | `make ci-check`               |

### 服务重启规范
**⚠️ 禁止直接执行启停命令**，修改后端代码后通知用户手动 `make restart`

### 提交流程
```
make check-all → feat/fix 分支 → PR → 合并
```
详细规范：@.claude/rules/git-workflow.md

---

## 📐 编码规范

### 核心原则
> **减法 > 加法**：优先删除重复代码、合并相似功能

| 原则    | 实践               |
| :------ | :----------------- |
| DRY     | 提取公共逻辑       |
| SOLID-S | 单一职责           |
| SOLID-O | 扩展开放，修改封闭 |
| SOLID-D | 依赖接口           |

### 语言规范
| Go              | React/TS         | Tailwind v4        |
| :-------------- | :--------------- | :----------------- |
| `snake_case.go` | `PascalCase.tsx` | `max-w-[24rem]`    |
| `log/slog`      | `use` 前缀       | 显式值避免解析错误 |
| 检查错误        | `t("key")`       |                    |

### Go Lint 必过
| 问题          | 正确写法                      |
| :------------ | :---------------------------- |
| 类型断言      | `v, ok := x.(T)`              |
| defer 错误    | 检查 Close() 返回值           |
| 错误比较      | `errors.Is(err, expectedErr)` |
| HTTP nil body | `http.NoBody`                 |
| error 变量名  | `var errTest`                 |

---

## 🚫 变更边界

| 约束         | 说明               |
| :----------- | :----------------- |
| 避免过度删除 | 删除前验证所有引用 |
| 测试后提交   | 推送前 `npm test`  |
| Proto 变更   | 重新生成前后端绑定 |
| 批量重构     | 分阶段提交         |

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
