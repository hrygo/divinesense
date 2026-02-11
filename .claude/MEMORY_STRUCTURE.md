# Memory 分层结构配置

> **优化策略**: 分层加载 | **基线 Token**: ~5k | **版本**: v1.0

---

## 🎯 设计原则

1. **核心常驻**: 最高频信息始终加载 (~2k tokens)
2. **按需加载**: 详细文档通过 `@path` 引用加载
3. **摘要优先**: 大文件先提供摘要，按需展开
4. **单一来源**: 每个知识点只在一个地方维护

---

## 📊 分层结构

### Tier 1: 核心常驻 (~2k tokens)

| 文件 | 大小 | 说明 |
|:-----|:-----|:-----|
| `~/.claude/CLAUDE.md` | ~1k | 用户全局指令 |
| `CLAUDE.md` | ~4k | 项目根文档（已精简） |
| `.claude/rules/code-style.md` | ~300 | 编码规范 |

**总计**: ~2.5k tokens（从 11k 压缩至 2.5k）

---

### Tier 2: 重要按需 (~5k tokens 按需加载)

触发条件：架构讨论、后端开发、前端开发、部署

| 文件 | 大小 | 触发场景 |
|:-----|:-----|:---------|
| `docs/dev-guides/ARCHITECTURE.md` | 44k | 架构设计/重构 |
| `docs/dev-guides/BACKEND_DB.md` | 19k | 后端/数据库开发 |
| `docs/dev-guides/FRONTEND.md` | 16k | 前端开发 |
| `docs/deployment/BINARY_DEPLOYMENT.md` | 11k | 部署相关 |
| `docs/research/DEBUG_LESSONS.md` | 8k | 调试问题 |

**加载方式**: 通过 `@path` 引用时自动加载

---

### Tier 3: 参考归档 (仅在明确请求时加载)

| 类型 | 说明 |
|:-----|:-----|
| `docs/archived/` | 历史研究/计划 |
| `docs/specs/` | 详细规格文档 |
| `.claude/skills/` | Skill 定义（需时加载） |

---

## 🔧 使用模式

### 模式 1: 引用加载（推荐）
```markdown
# 不直接嵌入内容，使用引用
架构详情：@docs/dev-guides/ARCHITECTURE.md
后端开发：@docs/dev-guides/BACKEND_DB.md
前端开发：@docs/dev-guides/FRONTEND.md
```

### 模式 2: 摘要 + 引用
```markdown
## Git 工作流

**核心**: Conventional Commits + PR 流程
详细规范: @.claude/rules/git-workflow.md

### 快速命令
```bash
gh pr create    # 创建 PR
gh pr merge    # 合并 PR
```
```

### 模式 3: 索引式引用
```markdown
## 常见问题

| 问题 | 参考 |
|:-----|:-----|
| 布局宽度不统一 | @docs/research/layout-spacing-unification.md |
| 流式事件缺失 | @docs/research/DEBUG_LESSONS.md → 流式渲染事件缺失 |
| Tailwind v4 陷阱 | @docs/dev-guides/FRONTEND.md → Tailwind CSS 4 陷阱 |
```

---

## 📏 Token 预算

| 场景 | 预估用量 |
|:-----|:---------|
| 冷启动 | ~2.5k (仅 Tier 1) |
| 后端任务 | ~7k (Tier 1 + BACKEND_DB) |
| 前端任务 | ~6.5k (Tier 1 + FRONTEND) |
| 架构设计 | ~9k (Tier 1 + ARCHITECTURE) |
| 全栈开发 | ~12k (Tier 1 + Tier 2 全部) |

**优化前**: 冷启动 ~11k，全栈 ~40k+
**优化后**: 冷启动 ~2.5k，全栈 ~12k
**节省**: ~77% (冷启动), ~70% (全栈)

---

## ✅ 检查清单

- [ ] 避免在 CLAUDE.md 中嵌入大段内容
- [ ] 使用 `@path` 引用而非复制粘贴
- [ ] 新增文档考虑放入 Tier 2/3
- [ ] 定期归档过时内容到 `docs/archived/`
- [ ] 技能讨论使用 Task(Explore) 而非预加载

---

*本文档控制 Memory 文件的加载策略。*
