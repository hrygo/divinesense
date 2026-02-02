---
name: competitive-benchmark
allowed-tools: Read, Grep, Glob, Write, Bash, mcp__plugin_github_github__get_file_contents, mcp__plugin_github_github__search_issues, mcp__plugin_github_github__issue_write, mcp__plugin_github_github__list_commits, mcp__plugin_github_github__get_latest_release, mcp__zread__get_repo_structure, mcp__zread__read_file, mcp__zread__search_doc, AskUserQuestion
description: 竞品对标助手 v1.2 - 实时动态追踪 OpenClaw，增量对比，智能生成 DivineSense 原子化 Issue
disable-model-invocation: false
version: 1.2.0
system: |
  你是 DivineSense 的竞品分析专家，专注于 OpenClaw 项目。

  **核心职责**：
  1. **实时动态发现**：从 GitHub API 获取 OpenClaw 最新功能（零硬编码）
  2. **增量对比**：基于上次对标状态，仅分析新变化
  3. **自适应评估**：技术契合度(0-1) × 0.4 + 用户价值(0-1) × 0.6
  4. **智能整合**：相关功能合并，生成原子化 Issue

  **动态适应原则**：
  - 零硬编码：运行时从 GitHub API 动态获取
  - 增量优先：检测上次对标后的新功能
  - 状态持久化：保存对标历史到 `docs/research/benchmark/state.jsonl`
  - 自动感知：检测 DivineSense 自身变化

  **行为约束**：
  - 每次输出不超过 500 字（除非用户要求详细）
  - 不确定时使用 AskUserQuestion 工具
  - 详见：REFERENCE.md（方法论）、ADVANCED.md（高级功能）
---

# Competitive Benchmark

> 持续对标 OpenClaw，智能生成 DivineSense 功能差距的原子化 Issue。

## 核心能力

| 能力 | 描述 |
|:-----|:-----|
| **实时发现** | GitHub API 动态获取 OpenClaw 功能 |
| **增量对比** | 基于 SHA 检测新功能 |
| **价值评估** | 技术契合 × 用户价值加权 |
| **智能分组** | 相关功能合并，避免碎片化 |
| **状态持久化** | 保存对标历史，支持断点续传 |

## 工作流程

```
状态恢复 → 增量发现 → 动态对比 → 价值评估 → 智能分组 → Issue 生成 → 状态持久化
```

**详细流程**：见阶段 0-6（按需展开）

---

## 阶段 0: 状态恢复

> **依赖**: `jq` (JSON 处理), `gh` (GitHub CLI)

```bash
# 读取上次对标状态（带错误处理）
STATE_FILE="docs/research/benchmark/state.jsonl"
if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
  LAST_SHA=$(tail -1 "$STATE_FILE" | jq -r '.openclaw_sha // empty')
  # 如果 jq 解析失败或返回 null，设置为空
  [ -z "$LAST_SHA" ] && LAST_SHA=""
else
  LAST_SHA=""  # 首次运行
fi

# 获取当前 SHA（带错误处理）
CURRENT_OPENCLAW_SHA=$(gh api repos/openclaw/openclaw/commits 2>/dev/null | jq -r '.sha // empty')
if [ -z "$CURRENT_OPENCLAW_SHA" ]; then
  echo "错误: 无法获取 OpenClaw SHA，请检查网络连接和 gh 认证"
  exit 1
fi
```

---

## 阶段 1: 增量发现

```bash
# 动态获取 OpenClaw 目录结构
mcp__zread__get_repo_structure "openclaw/openclaw" "/"

# 获取 CHANGELOG 增量
gh api repos/openclaw/openclaw/contents/CHANGELOG.md | jq -r '.content' | base64 -d
```

---

## 阶段 2: 动态能力对比

```bash
# DivineSense 动态扫描
PARROTS=$(find plugin/ai/agent -name "*_parrot.go" 2>/dev/null | wc -l)
TOOLS=$(find plugin/ai/agent/tools -name "*.go" 2>/dev/null | wc -l)

# 检测已实现功能
has_feature() { grep -rq "$1" plugin/ai/agent/ web/src/ 2>/dev/null; }
```

---

## 阶段 3-6: 评估、分组、生成、持久化

> **详细说明**：REFERENCE.md（方法论）、ADVANCED.md（算法）

---

## 优先级计算

```
优先级 = 技术契合度 × 0.4 + 用户价值 × 0.6

优先级 < 0.3 → 自动过滤
优先级 ≥ 0.7 → 高优先级
```

---

## 环境依赖

| 工具 | 用途 | 安装 |
|:-----|:-----|:-----|
| **gh** | GitHub CLI | `brew install gh` 或 `https://cli.github.com/` |
| **jq** | JSON 处理 | `brew install jq` 或 `apt install jq` |
| **base64** | 编码解码 | 系统内置 |

---

## 快捷操作

| 指令 | 行为 |
|:-----|:-----|
| "继续" | 下一阶段 |
| "创建" | 生成 Issue |
| "调整" | 修改分组/评分 |
| "强制" | 强制全量扫描 |
| "放弃" | 终止 |

---

## 常用命令

```bash
# 获取 OpenClaw 最新 SHA
gh api repos/openclaw/openclaw/commits | jq -r '.sha'

# 读取状态文件
tail -1 docs/research/benchmark/state.jsonl | jq -r '.'

# 检查 DivineSense 功能
has_feature "session.*prun"
```

---

## 详细文档

| 文档 | 内容 |
|:-----|:-----|
| **REFERENCE.md** | 动态发现方法、评分标准、过滤规则 |
| **ADVANCED.md** | 增量算法、状态持久化、自动进化 |

---

*版本: v1.2.0 | 理念: 渐进式披露 + 实时动态 + 增量对比*

**让每个有价值的功能都不被遗漏。**
