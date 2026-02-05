# DivineSense 文档中心

> **更新时间**: 2025-02-02
> **文档版本**: v1.1

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
│   ├── evolution/               # 进化模式规格
│   ├── sprint-0/                # Sprint 0: 接口契约
│   ├── phase-1/                 # Phase 1: 基础稳定
│   ├── phase-2/                 # Phase 2: 智能进化
│   └── phase-3/                 # Phase 3: 极致体验
├── prompts/                     # AI 提示词
│   └── 202601301323.md
├── images/                      # 图片资源
└── archived/                    # 历史归档
    ├── cleanup_20260123/        # 早期归档 (2026-01-23)
    ├── research_cleanup_20260131/ # 研究归档 (2026-01-31)
    ├── research_20250202/       # 研究路线图归档 (2025-02-02)
    ├── specs/                   # 已完成规格归档
    │   └── phase-1-completed/    # Phase-1 已完成规格
    └── specs/                   # 已完成 AI/FE 规格
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

| 目录                                                                         | 归档时间   | 内容                       |
| :--------------------------------------------------------------------------- | :--------- | :------------------------- |
| [`archived/cleanup_20260123/`](archived/cleanup_20260123/)                   | 2026-01-23 | 早期实施计划、RAG 研究     |
| [`archived/research_cleanup_20260131/`](archived/research_cleanup_20260131/) | 2026-01-31 | 历史报告、方法论           |
| [`archived/research_20250202/`](archived/research_20250202/)                 | 2025-02-02 | 研究路线图（8 个文档）     |
| [`archived/specs/phase-1-completed/`](archived/specs/phase-1-completed/)     | 2025-02-02 | Phase-1 已完成规格（9 个） |
| [`archived/specs/`](archived/specs/)                                         | 2026-01-23 | 已完成的 AI/FE 规格        |

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
