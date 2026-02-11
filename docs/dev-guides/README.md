# DivineSense 开发者指南索引

> **版本**: v0.97.0 | **更新时间**: 2026-02-11
>
> 本文档目录为 DivineSense 开发者提供全面的技术指南和架构文档。

---

## 📖 目录

### 核心架构文档

| 文档 | 描述 | 面向读者 |
|:-----|:-----|:---------|
| **[架构摘要](ARCHITECTURE_SUMMARY.md)** | 五位鹦鹉、关键概念、目录结构速查 | 所有开发者 |
| **[架构详细文档](ARCHITECTURE.md)** | 项目整体架构、技术栈、系统设计 | 所有开发者 |
| **[后端与数据库指南](BACKEND_DB.md)** | Go 后端开发、数据库设计、API 设计 | 后端开发者 |
| **[前端开发指南](FRONTEND.md)** | React 组件、布局系统、i18n、状态管理 | 前端开发者 |

### 工作流与规范

| 文档 | 描述 | 面向读者 |
|:-----|:-----|:---------|
| **[开发工作流](WORKFLOW.md)** | 开发命令、Git 提交流程、常见任务 | 所有开发者 |
| **[Agent 工作流](AGENT_WORKFLOW.md)** | Agent 开发思考协议、工具选择 | Agent 开发者 |
| **[Agent 测试](AGENT_TESTING.md)** | Agent 三种测试方式、验证清单 | Agent 开发者 |

### Agent 开发

| 文档 | 描述 | 面向读者 |
|:-----|:-----|:---------|
| **[Agent 快速开始](QUICKSTART_AGENT.md)** | UniversalParrot 架构、配置驱动开发 | 新手开发者 |

### 系统模块文档

| 文档 | 描述 | 面向读者 |
|:-----|:-----|:---------|
| **[AI Chat 界面](AI_CHAT_INTERFACE.md)** | Unified Block Model、组件架构 | 前端开发者 |
| **[CC Runner 架构](CC_RUNNER_ARCH.md)** | Geek Mode 核心、Claude Code CLI 集成 | 后端开发者 |
| **[Stats 系统](STATS_SYSTEM.md)** | 成本追踪、指标收集、告警 | 后端开发者 |
| **[SQLite 向量使用](SQLITE_VEC_USAGE_GUIDE.md)** | sqlite-vec 功能范围、本地开发 | 后端开发者 |

---

## 🚀 快速导航

### 我想了解...

**项目整体架构**
→ 从 [架构摘要](ARCHITECTURE_SUMMARY.md) 开始，了解五位鹦鹉和目录结构
→ 深入阅读 [架构详细文档](ARCHITECTURE.md)

**开发环境搭建**
→ 参考 [贡献指南](../CONTRIBUTING.md) 的"开发环境搭建"章节

**后端开发**
→ 阅读 [后端与数据库指南](BACKEND_DB.md)，了解 API 设计、数据库架构

**前端开发**
→ 阅读 [前端开发指南](FRONTEND.md)，了解布局架构、组件模式、i18n 规范

**AI 代理开发**
→ 从 [Agent 快速开始](QUICKSTART_AGENT.md) 入门
→ 参考 [Agent 工作流](AGENT_WORKFLOW.md) 了解开发范式

**开发命令和工作流**
→ 查看 [开发工作流](WORKFLOW.md)，了解常用命令和提交流程

---

## 📁 文档组织结构

```
docs/
├── dev-guides/           # 开发者指南（当前目录）
│   ├── README.md                  # 本索引文件
│   │
│   ├── 核心架构
│   ├── ARCHITECTURE_SUMMARY.md    # 架构速查
│   ├── ARCHITECTURE.md            # 详细架构
│   ├── BACKEND_DB.md              # 后端与数据库
│   ├── FRONTEND.md                # 前端开发
│   │
│   ├── 工作流与规范
│   ├── WORKFLOW.md                # 开发工作流
│   ├── AGENT_WORKFLOW.md          # Agent 工作范式
│   ├── AGENT_TESTING.md           # Agent 测试
│   │
│   ├── Agent 开发
│   ├── QUICKSTART_AGENT.md        # Agent 快速开始
│   │
│   └── 系统模块
│       ├── AI_CHAT_INTERFACE.md   # AI Chat 界面
│       ├── CC_RUNNER_ARCH.md      # CC Runner 架构
│       ├── STATS_SYSTEM.md        # 统计系统
│       └── SQLITE_VEC_USAGE_GUIDE.md  # SQLite 向量
│
├── essentials/           # 核心上下文
│   └── CRITICAL_CONTEXT.md
│
├── deployment/           # 部署指南
│   └── BINARY_DEPLOYMENT.md
│
├── research/             # 研究文档
│   └── DEBUG_LESSONS.md
│
└── specs/                # 技术规格
    ├── UNEXPOSED_FEATURES.md
    └── block-design/
```

---

## 🔗 相关资源

### 规范文档

| 文档 | 位置 | 描述 |
|:-----|:-----|:-----|
| **Memory 分层结构** | `../../.claude/MEMORY_STRUCTURE.md` | 文档加载策略 |
| **Git 工作流** | `../../.claude/rules/git-workflow.md` | 分支管理、PR 规范 |
| **代码风格** | `../../.claude/rules/code-style.md` | Go/React 代码规范 |
| **国际化规范** | `../../.claude/rules/i18n.md` | i18n 最佳实践 |
| **API 设计规范** | `../../.claude/rules/api-design.md` | API 设计原则 |
| **测试规范** | `../../.claude/rules/testing.md` | 测试指南 |

### 项目根文档

| 文档 | 描述 |
|:-----|:-----|
| [README.md](../../README.md) | 项目概述、快速开始 |
| [CONTRIBUTING.md](../../CONTRIBUTING.md) | 贡献指南、开发规范 |
| [CHANGELOG.md](../../CHANGELOG.md) | 版本更新记录 |

---

## 📝 文档维护

### 保鲜状态

所有文档应保持"保鲜状态"：
- 标注最后检查版本和日期
- 重大变更时同步更新文档
- 发现过时内容及时修正

### 更新规范

1. **版本同步**：文档中引用的版本号应与 git tag 一致
2. **交叉引用**：使用相对路径维护文档间的链接
3. **示例代码**：所有代码示例应可实际运行
4. **避免重复**：新内容优先放入现有文档，而非创建新文档

---

*最后更新：2026-02-11*
