# CLAUDE.md

> DivineSense 项目开发纲领 — Claude Code 辅助开发的核心指导文档

## 第一性原理

**DivineSense (神识)** = AI 代理驱动的个人「第二大脑」

```
核心使命：通过智能代理自动化任务、过滤高价值信息、以技术杠杆提升生产力

技术本质：Go 后端 + React 前端的单二进制分发应用
架构核心：五位「鹦鹉」AI 代理 + 统一块模型 (Unified Block Model)
```

---

## 架构原则

### 核心概念映射

| 概念     | 实体             | 关系                   |
| :------- | :--------------- | :--------------------- |
| **对话** | `AIConversation` | 包含多个 Block         |
| **块**   | `AIBlock`        | 一个用户-AI 交互轮次   |
| **代理** | `ParrotAgent`    | 处理用户请求的 AI 实体 |
| **路由** | `ChatRouter`     | 决定使用哪只鹦鹉       |

### 关键架构决策

**1. BlockMode ≠ ParrotAgentType** （最常混淆）
- `BlockMode.NORMAL/GEEK/EVOLUTION` — 消息块的结构模式
- `ParrotAgentType.AUTO/MEMO/SCHEDULE/...` — 哪只鹦鹉处理请求
- **无映射关系**：不要在代码中相互转换

**2. AUTO 不是鹦鹉**
- `AUTO` 是「请后端决定」的标记
- 后端三层路由：规则匹配 → 历史感知 → LLM 降级

**3. 数据库选择影响功能**
- PostgreSQL → 完整 AI 功能（向量搜索、对话持久化）
- SQLite → 仅开发环境，AI 功能禁用

---

## 工作流

### 开发前
```bash
make deps-all      # 安装依赖
make start         # 启动全栈
```

### 开发中
```bash
make check-all     # 提交前检查
make ci-check      # 模拟 CI
```

### 提交流程
1. `make check-all` 通过
2. 分支命名：`feat/xxx`、`fix/xxx`、`evolution/xxx`
3. 禁止直接 push 到 main
4. 通过 PR 合并

详细规范：@.claude/rules/git-workflow.md

---

## 编码规范

### Go
- 文件：`snake_case.go`
- 日志：`log/slog`
- 错误：始终检查并处理

### React/TypeScript
- 组件：`PascalCase.tsx`
- Hooks：`use` 前缀
- 文本：`t("key")` 国际化

### Tailwind v4
- ❌ `max-w-md/lg/xl` → 解析错误 (~16px)
- ✅ `max-w-[24rem]` → 显式值

---

## 导航索引

| 任务         | 文档                                  |
| :----------- | :------------------------------------ |
| **理解架构** | @docs/dev-guides/ARCHITECTURE.md      |
| **后端开发** | @docs/dev-guides/BACKEND_DB.md        |
| **前端开发** | @docs/dev-guides/FRONTEND.md          |
| **部署**     | @docs/deployment/BINARY_DEPLOYMENT.md |
| **调试问题** | @docs/research/DEBUG_LESSONS.md       |

---

## 产品能力边界

| 功能           | 状态                                 |
| :------------- | :----------------------------------- |
| 笔记           | ✅ Markdown + 语义搜索                |
| 日程           | ✅ 自然语言 + 冲突检测                |
| AI 代理        | ✅ 五位鹦鹉协同                       |
| Geek Mode      | ✅ Claude Code CLI 集成，用于复杂任务 |
| Evolution Mode | ✅ Claude Code CLI 集成，系统自我进化 |

---

*本文档随项目演进自动更新。新增功能时同步更新架构原则和导航索引。*
