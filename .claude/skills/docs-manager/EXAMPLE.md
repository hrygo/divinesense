# 文档管理技能示例 (docs-manager Examples)

本文档展示 `docs-manager` 技能的实际使用场景。

---

## 示例 1: 验证文档完整性

### 用户输入

```
/docs-check
```

### AI 执行过程

```python
# 1. 检查 docs/README.md
if not exists("docs/README.md"):
    issues.append("缺少 docs/README.md 总入口")

# 2. 验证 dev-guides/ 索引
expected_dev_guides = [
    "ARCHITECTURE.md", "BACKEND_DB.md", "FRONTEND.md",
    "QUICKSTART_AGENT.md", "UNEXPOSED_FEATURES.md"
]
for file in expected_dev_guides:
    if not exists(f"docs/dev-guides/{file}"):
        issues.append(f"缺少 {file}")

# 3. 验证 research/README.md 索引
research_index = parse_markdown("docs/research/README.md")
research_files = glob("docs/research/*.md")
for file in research_files:
    if basename(file) not in research_index:
        issues.append(f"research/ 中有未索引文件: {file}")

# 4. 验证 specs/INDEX.md
# ... 类似检查
```

### AI 输出

```
📋 文档检查报告 (2026-01-31)

✓ 通过的检查 (5/7)
  ✓ docs/README.md 存在且格式正确
  ✓ dev-guides/ 所有文档已索引
  ✓ research/README.md 索引完整 (12/12)
  ✓ specs/INDEX.md 存在
  ✓ 归档目录结构清晰

⚠ 警告 (1/7)
  ⚠ specs/INDEX.md 引用 P3-C002-review-system.md 但文件不存在

✗ 错误 (1/7)
  ✗ docs/dev-guides/TESTING.md 索引中引用但文件不存在

建议操作:
1. 创建缺失的 TESTING.md 或从索引中移除引用
2. 更新 specs/INDEX.md 移除 P3-C002 或创建该规格文档
```

---

## 示例 2: 创建新规格文档

### 用户输入

```
/docs-new spec reranker --phase=2 --team=c
```

### AI 执行过程

```python
# 1. 确定路径和文件名
phase = 2
team = "c"
name = "reranker"
spec_id = get_next_spec_id(phase, team)  # P2-C003
filename = f"P2-C003-reranker.md"
path = f"docs/specs/phase-2/team-c/{filename}"

# 2. 读取模板
template = read("docs/specs/SPEC_TEMPLATE.md")

# 3. 填充模板
content = template.replace(
    "{{SPEC_ID}}", spec_id
).replace(
    "{{TITLE}}", "重排服务优化"
).replace(
    "{{PHASE}}", "2"
).replace(
    "{{TEAM}}", "C"
)

# 4. 写入文件
write(path, content)

# 5. 更新索引
update_specs_index(spec_id, filename, "重排服务优化")
```

### 创建的文件

**docs/specs/phase-2/team-c/P2-C003-reranker.md**:

```markdown
# P2-C003: 重排服务优化

> **Phase**: 2 | **Team**: C | **状态**: Draft

---

## 概述

优化重排服务 (Reranker) 的性能和准确性。

## 背景

当前重排服务在处理大量结果时存在延迟问题...

## 目标

- [ ] 降低重排延迟 50%
- [ ] 提高排序准确性
- [ ] 支持批量重排

## 实施方案

...
```

### 更新的索引

**docs/specs/INDEX.md** (部分):

```markdown
### Phase 2 - Sprint 3

#### 团队 C

| Spec ID | 功能 | 投入 | 状态 |
|:---|:---|---:|:---:|
| [P2-C001](./phase-2/team-c/P2-C001-smart-tags.md) | 智能标签 | 7人天 | ✅ |
| [P2-C002](./phase-2/team-c/P2-C002-duplicate-detection.md) | 重复检测 | 9人天 | ✅ |
| [P2-C003](./phase-2/team-c/P2-C003-reranker.md) | 重排优化 | 5人天 | 📝 |
```

---

## 示例 3: 归档已完成规格

### 用户输入

```
/docs-archive docs/specs/phase-1/
```

### AI 执行过程

```python
# 1. 确认归档操作
print(f"即将归档 {count_files('docs/specs/phase-1/')} 个文件")
print("目标: docs/archived/specs/phase1_20260131/")

# 2. 创建归档目录
archive_dir = "docs/archived/specs/phase1_20260131"
mkdir(archive_dir)

# 3. 移动文件
move("docs/specs/phase-1/", f"{archive_dir}/phase-1/")

# 4. 创建归档 README
readme = f"""# Phase 1 规格归档

> **归档时间**: {today()}
> **状态**: 已完成实施

## 📁 归档内容

Phase 1 所有规格文档已实现并合并到主分支。

## 🔗 相关文档

当前活跃规格: [../../specs/](../../specs/)
"""
write(f"{archive_dir}/README.md", readme)

# 5. 更新索引
update_specs_index_remove("phase-1")
```

### AI 输出

```
📦 归档完成

源目录: docs/specs/phase-1/
目标目录: docs/archived/specs/phase-1-completed/  (2025-02-02)

归档文件:
  ✓ phase-1/team-a/P1-A001-memory-system.md
  ✓ phase-1/team-a/P1-A002-metrics-framework.md
  ... (共 9 个文件)

> 注: Phase-1 已完成规格已归档，此示例展示历史归档格式
```

---

## 示例 4: 更新索引

### 用户输入

```
/docs-index research/
```

### AI 执行过程

```python
# 1. 扫描目录
files = glob("docs/research/*.md")
files = [f for f in files if f != "docs/research/README.md"]

# 2. 分类文档
categories = {
    "core": [],      # 00-master-roadmap.md
    "assistant": [], # assistant-*.md
    "memo": [],      # memo-*.md
    "schedule": [],  # schedule-*.md
    "practice": [],  # BEST_PRACTICE_*.md
}

for file in files:
    name = basename(file)
    if name.startswith("00-"):
        categories["core"].append(name)
    elif name.startswith("assistant-"):
        categories["assistant"].append(name)
    # ... 其他分类

# 3. 生成索引
index = generate_index(categories)

# 4. 写入文件
write("docs/research/README.md", index)
```

### 生成的索引

**docs/research/README.md** (部分):

```markdown
# 研究文档 (Research)

## 📁 当前结构

```
research/
├── 00-master-roadmap.md           # 统一实施路线图
├── assistant-research.md          # 智能助理架构调研
├── assistant-roadmap.md           # 助理升级路径
├── BEST_PRACTICE_CLI_AGENT.md     # CLI Agent 最佳实践
└── ...
```

## 📊 文档分类

### 核心路线图

| 文档 | 说明 |
|:---|:---|
| [00-master-roadmap.md](./00-master-roadmap.md) | 统一实施路线图 |

### 领域调研

| 领域 | 调研 | 路线图 |
|:-----|:-----|:-------|
| 智能助理 | [assistant-research.md](./assistant-research.md) | [assistant-roadmap.md](./assistant-roadmap.md) |
...
```

---

## 示例 5: 整理文档

### 用户输入

```
/docs-tidy
```

### AI 执行过程

```python
# 1. 检查命名规范
violations = []
for file in all_docs_files():
    if not follows_naming_convention(file):
        violations.append((file, suggest_new_name(file)))

# 2. 检查过期内容
old_files = find_files_not_modified_in(days=180)
archivable = [f for f in old_files if is_completed_spec(f)]

# 3. 检查重复内容
duplicates = find_duplicate_content()

# 4. 生成建议
suggestions = {
    "rename": violations,
    "archive": archivable,
    "merge": duplicates
}
```

### AI 输出

```
🧹 文档整理建议

命名规范问题 (2):
  ⚠ docs/research/MemoResearch.md
     → 建议重命名为: memo-research.md
  ⚠ docs/specs/P1-A001.md
     → 建议重命名为: phase-1/team-a/P1-A001-memory-system.md

建议归档 (5):
  📦 docs/specs/phase-1/ (已完成实施)
  📦 docs/research/PROJECT_COMPLETION_REPORT.md (历史报告)

重复内容 (1):
  🔀 docs/dev-guides/CACHE.md 和 docs/research/cache-research.md
     内容重叠 60%，建议合并或交叉引用

操作建议:
1. 运行 /docs-archive 归档已完成规格
2. 手动重命名不符合规范的文件
3. 检查重复内容并决定合并策略
```

---

## 示例 6: 显示文档树

### 用户输入

```
/docs-tree
```

### AI 输出

```
📂 docs/
├── 📄 README.md ✅ (索引完整)
├── 📂 dev-guides/
│   ├── 📄 ARCHITECTURE.md ✅
│   ├── 📄 BACKEND_DB.md ✅
│   ├── 📄 FRONTEND.md ✅
│   ├── 📄 QUICKSTART_AGENT.md ✅
│   └── 📄 UNEXPOSED_FEATURES.md ✅
├── 📂 deployment/
│   └── 📄 BINARY_DEPLOYMENT.md ✅
├── 📂 research/
│   ├── 📄 README.md ✅
│   ├── 📄 00-master-roadmap.md ✅
│   ├── 📄 assistant-research.md ✅
│   ├── 📄 assistant-roadmap.md ✅
│   ├── 📄 assistant-roadmap-industry.md 📦 (建议归档)
│   ├── 📄 BEST_PRACTICE_CLI_AGENT.md ✅
│   ├── 📄 DEBUG_LESSONS.md ✅
│   ├── 📄 memo-research.md ✅
│   ├── 📄 memo-roadmap.md ✅
│   ├── 📄 schedule-research.md ✅
│   └── 📄 schedule-roadmap.md ✅
├── 📂 specs/
│   ├── 📄 INDEX.md ✅
│   ├── 📄 SPEC_TEMPLATE.md ✅
│   ├── 📂 evolution/
│   │   └── 📄 EVOLUTION_MODE_SPEC.md ✅
│   ├── 📂 sprint-0/
│   │   └── 📄 S0-interface-contract.md ✅
│   ├── 📂 phase-1/ 📦 (已完成并归档至 archived/specs/phase-1-completed/)
│   ├── 📂 phase-2/ 🔄 (进行中)
│   └── 📂 phase-3/ ⏸️ (搁置)
└── 📂 archived/
    ├── 📂 cleanup_20260123/ 📦
    ├── 📂 research_cleanup_20260131/ 📦
    └── 📂 specs/ 📦

图例: ✅ 正常  🔄 进行中  ⏸️ 搁置  📦 归档
```

---

## 命令速查表

| 命令 | 功能 | 频率 |
|:-----|:-----|:-----|
| `/docs-check` | 验证文档完整性 | 每周 |
| `/docs-new <type> <name>` | 创建新文档 | 按需 |
| `/docs-archive <files>` | 归档文档 | 每月 |
| `/docs-index <dir>` | 更新索引 | 按需 |
| `/docs-tidy` | 整理建议 | 每月 |
| `/docs-tree` | 显示结构树 | 按需 |

---

> **提示**: 本技能可与其他技能组合使用，如与 `/commit` 配合完成文档更新的提交。
