# DivineSense 文档中心

> **更新时间**: 2026-02-07
> **文档版本**: v1.2

---

## 📁 文档导航

```
docs/
├── README.md                    # 本文件 - 文档总入口
├── architecture/                # 系统架构 (核心)
│   ├── overview.md              # 系统总览
│   ├── summary.md               # 架构摘要
│   └── cc-runner.md             # CC Runner 架构
├── dev-guides/                  # 开发指南 (活跃)
│   ├── README.md                # 开发指南索引
│   ├── backend/                 # 后端开发
│   ├── frontend/                # 前端开发
│   ├── agent/                   # Agent 开发
│   ├── testing/                 # 测试指南
│   ├── deployment/              # 部署指南
│   ├── user-manuals/            # 用户手册 (开发参考)
│   └── workflow/                # 工作流与工具
├── research/                    # 研究文档 (活跃)
│   ├── README.md                # 研究文档索引
│   ├── BEST_PRACTICE_CLI_AGENT.md # CLI 最佳实践
│   └── DEBUG_LESSONS.md         # 调试经验
├── specs/                       # 规格文档 (活跃)
│   ├── INDEX.md                 # 规格总索引
│   ├── SPEC_TEMPLATE.md         # 规格模板
│   ├── block-design/            # 统一 Block 模型设计 (重要)
│   └── evolution/               # 进化模式规格
├── images/                      # 图片资源
└── archived/                    # 历史归档
    ├── agent-engineering/       # Agent 工程归档
    ├── design/                  # 设计文档归档
    ├── operations/              # 运维文档归档
    ├── plans/                   # 计划文档归档
    ├── prompts/                 # 提示词归档
    ├── refactoring/             # 重构文档归档
    ├── release/                 # 发布文档归档
    ├── reports/                 # 报告归档
    └── ...                      # 其他历史归档
```

---

## 🚀 快速开始

| 角色         | 入口文档                                                           | 说明             |
| :----------- | :----------------------------------------------------------------- | :--------------- |
| **新开发者** | [overview.md](architecture/overview.md)                            | 了解系统架构     |
| **后端开发** | [database.md](dev-guides/backend/database.md)                      | 数据库、API、AI  |
| **前端开发** | [overview.md](dev-guides/frontend/overview.md)                     | 布局、组件、样式 |
| **运维部署** | [BINARY_DEPLOYMENT.md](dev-guides/deployment/BINARY_DEPLOYMENT.md) | 部署与运维       |

---

## 📊 文档分类

### 活跃文档 (Active)

| 目录                             | 用途         | 状态     |
| :------------------------------- | :----------- | :------- |
| [`architecture/`](architecture/) | 系统架构     | ✅ 核心   |
| [`dev-guides/`](dev-guides/)     | 开发指南     | ✅ 维护中 |
| [`research/`](research/)         | 研究与路线图 | ✅ 维护中 |
| [`specs/`](specs/)               | 实施规格     | ✅ 维护中 |

### 归档文档 (Archived)

| 目录                                                         | 内容               |
| :----------------------------------------------------------- | :----------------- |
| [`archived/agent-engineering/`](archived/agent-engineering/) | Agent 工程历史文档 |
| [`archived/design/`](archived/design/)                       | 历史设计方案       |
| [`archived/plans/`](archived/plans/)                         | 历史实施计划       |
| [`archived/reports/`](archived/reports/)                     | 历史分析报告       |
| [`archived/operations/`](archived/operations/)               | 历史运维日志       |
| [`archived/specs/`](archived/specs/)                         | 已完成/过期的规格  |

---

## 🔍 常见问题

### Q: 如何添加新文档？

根据文档类型选择目录：

1. **架构设计** → `architecture/`
2. **开发指南** → `dev-guides/{category}/`
3. **规格文档** → `specs/phase-{1,2,3}/team-{a,b,c}/`
4. **研究报告** → `research/`

### Q: 如何归档旧文档？

1. 移动文件到 `archived/` 下的对应分类目录
2. 如果是成批归档，可以创建日期目录，如 `archived/research/20260216_archive/`

### Q: 历史规格在哪里？

已完成的历史规格已移至 [`archived/specs/`](archived/specs/)，按类型分类：
- `ai/` - AI 后端规格
- `frontend/` - 前端规格
- `general/` - 通用规格

---

## 📝 维护规范

### 文档命名

| 类型     | 格式                             | 示例                       |
| :------- | :------------------------------- | :------------------------- |
| 开发指南 | `kebab-case.md`                  | `frontend/overview.md`     |
| 架构文档 | `kebab-case.md`                  | `architecture/overview.md` |
| 研究报告 | `{name}-research.md`             | `assistant-research.md`    |
| 路线图   | `{name}-roadmap.md`              | `memo-roadmap.md`          |
| 规格     | `P{Phase}-T{Team}{ID}-{name}.md` | `P1-A001-memory-system.md` |

### 更新原则

1. **活跃文档** - 随产品演进持续更新
2. **完成规格** - 移至 `archived/specs/`
3. **过期报告** - 移至 `archived/research_cleanup_YYYYMMDD/`

---

> **维护**: 文档应随项目演进同步更新
> **反馈**: 请通过 Issue 报告文档问题
