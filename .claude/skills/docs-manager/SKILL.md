---
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, TodoWrite, AskUserQuestion
description: 管理项目文档，自动追踪和更新引用
version: 2.0
system: |
  你是 DivineSense 的文档管理员。

  **核心原则**：安全第一、引用完整、可追溯

  **引用格式**：Markdown 链接、@语法、代码注释、绝对 URL

  **执行前**：扫描 → 构建引用图 → 分析影响 → 用户确认 → 执行
---

# 文档管理技能 (docs-manager)

> 管理文档系统，**自动追踪和更新引用**。

## 🚀 快速开始

```bash
/docs-check      # 检查断链和索引完整性
/docs-ref        # 查看文档引用关系
/docs-new spec feature --phase=2 --team=a    # 创建规格
/docs-archive docs/old.md               # 归档并更新引用
```

---

## 🔗 核心能力：引用追踪

文档移动/归档时，**自动更新所有引用**。

### 支持的引用格式

| 格式 | 示例 | 更新方式 |
|:-----|:-----|:---------|
| Markdown | `[文字](docs/xxx.md)` | 更新路径 |
| @ 语法 | `@docs/xxx.md` | 更新路径 |
| 代码注释 | `详见 docs/xxx.md` | 更新路径 |
| 绝对 URL | `https://.../docs/xxx.md` | 更新路径 |

### 引用更新流程

```
1. 构建引用图 → 2. 找反向引用 → 3. 显示影响 → 4. 用户确认 → 5. 执行
```

---

## 📋 命令

### `/docs-check`

检查文档结构和链接有效性。

```bash
# 检查所有文档
/docs-check

# 使用辅助脚本
python .claude/skills/docs-manager/docs_helper.py check
```

### `/docs-ref [target]`

查看文档引用关系。

```bash
/docs-ref                                    # 完整引用图
/docs-ref ARCHITECTURE.md                    # 单个文档引用
python .../docs_helper.py refs --json       # JSON 输出
```

### `/docs-new <type> <name> [options]`

创建新文档。

| 类型 | 位置 | 示例 |
|:-----|:-----|:-----|
| `dev-guide` | `dev-guides/` | `cache-guide` |
| `research` | `research/` | `vector-search` |
| `roadmap` | `research/` | `feature-roadmap` |
| `spec` | `specs/phase-X/team-Y/` | `memory` |

```bash
/docs-new spec memory --phase=2 --team=a    # P2-A001-memory-system.md
/docs-new dev-guide cache-guide            # dev-guides/CACHE_GUIDE.md
```

<details>
<summary>高级选项</summary>

- `--dry-run`: 预览不执行
- `--template`: 指定模板文件

</details>

### `/docs-archive <files...>`

归档文档，**自动更新所有引用**。

```bash
/docs-archive docs/old.md                      # 单文件
/docs-archive docs/specs/phase-1/               # 目录
/docs-archive "research/*_REPORT.md" --target=reports_20260131
```

**执行前显示**：
```
⚠️ 即将归档: old.md
🔗 受影响的引用 (3 处):
   CLAUDE.md:82 → @docs/old.md
   README.md:66 → docs/old.md
👉 是否继续? [Yes/No]
```

<details>
<summary>引用更新策略</summary>

| 原路径 | 新路径 (归档后) |
|:-------|:---------------|
| `docs/old.md` | `docs/archived/cleanup_YYYYMMDD/old.md` |
| `@docs/old.md` | `@docs/archived/.../old.md` |

</details>

### `/docs-index <dir>`

更新索引文件。

```bash
/docs-index research               # 更新 research/README.md
/docs-index specs --force         # 完全重建
```

### `/docs-tidy`

整理文档，提供建议。

```bash
/docs-tidy                          # 检查命名、重复内容
python .../docs_helper.py duplicates  # 仅检测重复
```

### `/docs-tree`

显示文档结构。

```bash
/docs-tree                           # 树形图
python .../docs_helper.py tree       # 详细视图
```

---

## 📁 文档结构

```
docs/
├── README.md              # 总入口
├── dev-guides/            # 开发指南
├── deployment/            # 部署文档
├── research/              # 研究文档
├── specs/                 # 规格文档
│   ├── evolution/         # 进化模式
│   └── phase-{1,2,3}/    # Sprint 规格
└── archived/              # 历史归档
```

---

## 📝 命名规范

| 类型 | 格式 | 示例 |
|:-----|:-----|:-----|
| 开发指南 | `UPPER_CASE.md` | `ARCHITECTURE.md` |
| 研究报告 | `{name}-research.md` | `assistant-research.md` |
| 路线图 | `{name}-roadmap.md` | `memo-roadmap.md` |
| 规格 | `P{X}-{Y}{ZZZ}-{name}.md` | `P1-A001-memory-system.md` |

---

## ⚙️ 执行标准

| 命令 | 成功标准 |
|:-----|:---------|
| `/docs-new` | 文件创建 + 索引更新 + 验证通过 |
| `/docs-archive` | 文件移动 + 引用更新 + Git 正常 |
| `/docs-check` | 扫描完成 + 报告输出 |

---

## 🔧 辅助工具

```bash
# 核心功能
python .claude/skills/docs-manager/docs_helper.py check       # 检查断链
python .claude/skills/docs-manager/docs_helper.py refs       # 引用图
python .claude/skills/docs-manager/docs_helper.py next-spec  # Spec ID
python .claude/skills/docs-manager/docs_helper.py duplicates # 重复内容

# JSON 输出 (AI 友好)
python .../docs_helper.py refs --json
python .../docs_helper.py next-spec --json
```

---

## 📚 相关文档

- 开发指南：`@docs/dev-guides/ARCHITECTURE.md`
- 研究文档：`@docs/research/00-master-roadmap.md`
- Git 工作流：`@.claude/rules/git-workflow.md`

---

> **版本**: v2.0 | **更新**: 2026-01-31
