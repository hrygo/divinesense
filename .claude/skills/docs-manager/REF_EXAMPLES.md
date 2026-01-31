# 引用追踪示例 (docs-manager Reference Tracking)

本文档展示 `docs-manager` 技能的引用追踪和更新能力。

---

## 场景 1: 查看文档引用关系

### 用户输入

```
/docs-ref ARCHITECTURE.md
```

### AI 执行

```python
# 1. 调用辅助脚本构建引用图
graph = build_reference_graph()

# 2. 查询目标文档
target = "dev-guides/ARCHITECTURE.md"
node = graph[target]

# 3. 格式化输出
```

### AI 输出

```
🔗 ARCHITECTURE.md 引用关系

┌─ 引用 (5 个)
│  ├─ BACKEND_DB.md
│  ├─ FRONTEND.md
│  ├─ ../specs/INDEX.md
│  ├─ ../research/00-master-roadmap.md
│  └─ ../deployment/BINARY_DEPLOYMENT.md
│
└─ 被引用 (8 处)
   ├─ CLAUDE.md:82 (@docs/dev-guides/ARCHITECTURE.md)
   ├─ README.md:232 (docs/dev-guides/ARCHITECTURE.md)
   ├─ .claude/rules/code-style.md:25 (详见 docs/dev-guides/FRONTEND.md)
   ├─ docs/README.md:7
   ├─ docs/dev-guides/BACKEND_DB.md:10
   ├─ docs/dev-guides/FRONTEND.md:5
   ├─ .claude/skills/docs-manager/SKILL.md:49
   └─ README.md (根目录):68
```

---

## 场景 2: 归档时自动更新引用

### 用户输入

```
/docs-archive docs/research/PROJECT_COMPLETION_REPORT.md
```

### AI 执行流程

```
1. 构建引用图
   ↓
2. 查找反向引用
   ↓
3. 显示影响预览
   ⚠️ 即将归档: PROJECT_COMPLETION_REPORT.md
   📦 目标: archived/research_cleanup_20260131/reports/
   🔗 受影响的引用 (1 处):
      └─ METHODOLOGY_REPORT.md:267 → specs/PROJECT_COMPLETION_REPORT.md
   👉 是否更新引用? [Yes/No/Show diff]
   ↓
4. 用户确认后更新
```

### 引用更新示例

**更新前** (`METHODOLOGY_REPORT.md:267`):
```markdown
详见 [项目完成报告](specs/PROJECT_COMPLETION_REPORT.md)
```

**更新后**:
```markdown
详见 [项目完成报告](archived/research_cleanup_20260131/reports/PROJECT_COMPLETION_REPORT.md) *(已归档)*
```

---

## 场景 3: 检测断链

### 用户输入

```
/docs-check
```

### AI 输出

```
📋 文档检查报告 (2026-01-31)

✓ 通过的检查 (5/7)
  ✓ docs/README.md 存在
  ✓ docs/dev-guides/ 所有文档已索引
  ✓ docs/research/README.md 索引完整
  ✓ docs/specs/INDEX.md 存在
  ✓ 无孤立的归档文档

⚠ 警告 (1/7)
  ⚠ specs/SPEC_TEMPLATE.md:130 → dev-guides/XXX.md (模板占位符)

✗ 断链 (47 个)
  ✗ research/SESSION_EVOLUTION_ROADMAP.md:305 → research/SESSION_MANAGEMENT_REPORT.md
  ✗ archived/... (归档文档内部引用，预期行为)

建议操作:
1. 归档 SESSION_MANAGEMENT_REPORT.md 或更新引用
2. SPEC_TEMPLATE.md 占位符可忽略
```

---

## 场景 4: 批量归档 Phase 1 规格

### 用户输入

```
/docs-archive docs/specs/phase-1/
```

### AI 执行

```
1. 扫描 phase-1/ 目录: 12 个文件
2. 构建引用图
3. 查找所有反向引用

⚠️ 即将归档 Phase 1 规格 (12 个文件)
📦 目标: archived/specs/phase1_20260131/

📋 文件列表:
   - phase-1/team-a/P1-A001-memory-system.md
   - phase-1/team-a/P1-A002-metrics-framework.md
   - ... (共 12 个)

🔗 受影响的引用 (5 处):
   └─ docs/specs/INDEX.md
      ├─ Sprint 0 链接
      ├─ Phase 1 链接
      └─ 3 个团队链接

👉 操作选项:
   [1] 归档并更新引用 (推荐)
   [2] 仅归档，手动更新引用
   [3] 取消
```

### 更新后的 specs/INDEX.md

```markdown
## Phase 1: 基础稳定 (已完成)

> ✅ 已完成实施，归档至 [archived/specs/phase1_20260131/](../archived/specs/phase1_20260131/)

所有 Phase 1 规格已实现并合并。

查看历史规格:
- [Sprint 0](../archived/specs/phase1_20260131/sprint-0/)
- [Team A](../archived/specs/phase1_20260131/team-a/)
- [Team B](../archived/specs/phase1_20260131/team-b/)
- [Team C](../archived/specs/phase1_20260131/team-c/)
```

---

## 引用格式处理

| 引用格式 | 示例 | 更新策略 |
|:---------|:-----|:---------|
| Markdown 链接 | `[文字](docs/old.md)` | 更新路径: `[文字](docs/archived/.../old.md)` |
| @ 语法 | `@docs/old.md` | 更新路径: `@docs/archived/.../old.md` |
| 代码注释 | `详见 docs/old.md` | 更新路径: `详见 docs/archived/.../old.md` |
| 绝对 URL | `https://.../docs/old.md` | 更新为归档路径或添加重定向 |

---

## 工具命令

```bash
# 检查链接
python .claude/skills/docs-manager/docs_helper.py check

# 显示引用图
python .claude/skills/docs-manager/docs_helper.py refs

# 获取下一个 Spec ID
python .claude/skills/docs-manager/docs_helper.py next-spec 2 a

# 检测重复内容
python .claude/skills/docs-manager/docs_helper.py duplicates
```

---

> **更新时间**: 2026-01-31
> **版本**: v1.2
