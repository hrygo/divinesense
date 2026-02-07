# DivineSense 文档中心

> **更新时间**: 2026-02-07
> **文档版本**: v1.2

---

## 📁 文档导航

```
docs/
├── README.md                    # 本文件 - 文档总入口
├── dev-guides/                  # 开发指南 (活跃)
│   ├── ARCHITECTURE.md          # 系统架构
│   ├── BACKEND_DB.md            # 后端与数据库
│   ├── FRONTEND.md              # 前端开发
│   ├── QUICKSTART_AGENT.md      # Agent 快速开始
│   └── UNEXPOSED_FEATURES.md    # 未暴露功能
├── deployment/                  # 部署文档
│   └── BINARY_DEPLOYMENT.md     # 二进制部署指南
├── research/                    # 研究文档
│   ├── BEST_PRACTICE_CLI_AGENT.md # CLI 最佳实践
│   ├── cc-runner-async-upgrade.md # CC Runner 异步架构
│   └── DEBUG_LESSONS.md         # 调试经验
├── specs/                       # 规格文档 (活跃)
│   ├── INDEX.md                 # 规格总索引
│   ├── SPEC_TEMPLATE.md         # 规格模板
│   ├── block-design/            # 统一 Block 模型设计 (重要)
│   └── evolution/               # 进化模式规格
├── prompts/                     # AI 提示词
│   └── 202601301323.md
├── images/                      # 图片资源
└── archived/                    # 历史归档
    ├── specs/                   # 规格文档归档
    │   ├── 20260207_archive/    # 2026-02-07 归档
    │   └── phase-1-completed/   # Phase-1 已完成规格
    ├── research/                # 研究文档归档
    │   ├── 20260207_archive/    # 2026-02-07 归档
    │   └── 20260131_archive/    # 2026-01-31 归档
    ├── projects/                # 项目专题归档 (Parrot 等)
    ├── reviews/                 # 代码评审与审计
    ├── refactor-plans/          # 重构与集成计划
    └── misc/                    # 启动计划与 ROI 分析
```

---

## 🚀 快速开始

| 角色           | 入口文档                                                | 说明             |
| :------------- | :------------------------------------------------------ | :--------------- |
| **新开发者**   | [ARCHITECTURE.md](dev-guides/ARCHITECTURE.md)           | 了解系统架构     |
| **后端开发**   | [BACKEND_DB.md](dev-guides/BACKEND_DB.md)               | 数据库、API、AI  |
| **前端开发**   | [FRONTEND.md](dev-guides/FRONTEND.md)                   | 布局、组件、样式 |
| **Agent 开发** | [QUICKSTART_AGENT.md](dev-guides/QUICKSTART_AGENT.md)   | AI 代理开发      |
| **运维部署**   | [BINARY_DEPLOYMENT.md](deployment/BINARY_DEPLOYMENT.md) | 部署与运维       |

---

## 📊 文档分类

### 活跃文档 (Active)

| 目录                         | 用途         | 状态     |
| :--------------------------- | :----------- | :------- |
| [`dev-guides/`](dev-guides/) | 开发指南     | ✅ 维护中 |
| [`deployment/`](deployment/) | 部署文档     | ✅ 维护中 |
| [`research/`](research/)     | 研究与路线图 | ✅ 维护中 |
| [`specs/`](specs/)           | 实施规格     | ✅ 维护中 |

### 归档文档 (Archived)

| 目录                                                                         | 归档时间   | 内容                     |
| :--------------------------------------------------------------------------- | :--------- | :----------------------- |
| [`archived/specs/20260207_archive/`](archived/specs/20260207_archive/)       | 2026-02-07 | Sprint 0/Phase 2/3 规格  |
| [`archived/research/20260207_archive/`](archived/research/20260207_archive/) | 2026-02-07 | 历史研究报告 (Agent/UBM) |
| [`archived/research/20260131_archive/`](archived/research/20260131_archive/) | 2026-01-31 | 历史报告、方法论         |
| [`archived/projects/parrot/`](archived/projects/parrot/)                     | 2026-01-29 | Parrot 专题文档          |
| [`archived/reviews/`](archived/reviews/)                                     | -          | 代码评审、审计报告       |
| [`archived/refactor-plans/`](archived/refactor-plans/)                       | -          | 重构计划、集成设计       |
| [`archived/specs/phase-1-completed/`](archived/specs/phase-1-completed/)     | 2025-02-02 | Phase-1 已完成规格       |

---

## 🔍 常见问题

### Q: 如何添加新文档？

根据文档类型选择目录：

1. **开发指南** → `dev-guides/`
2. **规格文档** → `specs/phase-{1,2,3}/team-{a,b,c}/`
3. **研究报告** → `research/`
4. **部署文档** → `deployment/`

### Q: 如何归档旧文档？

1. 在 `archived/` 下创建带日期的目录
2. 移动文件并添加 `README.md` 说明
3. 更新原目录的索引文件

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
| 开发指南 | `UPPER_CASE.md`                  | `ARCHITECTURE.md`          |
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
