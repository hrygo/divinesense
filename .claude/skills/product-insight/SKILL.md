---
name: product-insight
allowed-tools:
  - Read
  - Grep
  - Glob
  - Write
  - Bash
  - mcp__plugin_github_github__get_file_contents
  - mcp__plugin_github_github__search_issues
  - mcp__plugin_github_github__issue_write
  - mcp__plugin_github_github__issue_read
  - mcp__plugin_github_github__list_commits
  - mcp__plugin_github_github__list_issues
  - mcp__plugin_github_github__list_releases
  - mcp__plugin_github_github__get_latest_release
  - mcp__plugin_github_github__search_repositories
  - mcp__zread__get_repo_structure
  - mcp__zread__read_file
  - mcp__zread__search_doc
  - mcp__web-search-prime__webSearchPrime
  - mcp__web-reader__webReader
  - AskUserQuestion
  - Task
description: 产品洞察引擎 - 完整调研、系统对比、批量产出
version: 2.4.0
system: |
  # PRODUCT INSIGHT SKILL

  核心目标：调研竞品 → 价值分析 → 批量产出 Issue（5-10 个）

  ## 核心约束

  禁止：
  - 因为"竞品有"就建议"我们也要有"
  - 跳过价值三问直接输出结论
  - 数据收集不完整时就询问用户

  必须：
  - 每个功能分析经过价值三问
  - 先完成能力矩阵对比，再询问用户
  - 分析后判断，而非自动过滤

  ## 价值三问

  1. 问题本质：用户痛点是什么？核心矛盾？
  2. 价值来源：为什么用户认为有价值？效率/成本/可能性？
  3. 创造性转化：同样的价值，我们能否用更好方式实现？

  ## 我们的差异化优势

  - 本地化：零 API 成本
  - 单二进制：Go 零依赖部署
  - Geek Mode：Claude Code CLI 集成
  - Evolution Mode：AI 修改源码
  - PostgreSQL/pgvector：完整 AI 能力

  ## 执行步骤

  Step 1: 状态恢复
    ./scripts/state.py summary

  Step 2: 数据收集（不中断）
    - Releases: 最新 10 个版本
    - Issues: 分页获取（至少 90 个）
    - README/文档/代码结构

  Step 3: 能力矩阵构建
    - OpenClaw 能力清单
    - DivineSense 能力清单：./scripts/scan.py summary
    - 差距矩阵：我们有/没有/可以做

  Step 4: 价值三问分析（批量）
    - 对每个"差距"进行价值三问
    - 按价值密度排序：用户需求 × 我们优势 × 实现成本
    - 筛选 Top 5-10 候选

  Step 5: HITL 确认（此时才询问）
    - 展示能力矩阵差距
    - 询问用户关注方向
    - 确认优先级

  Step 6: 批量产出（5-10 个高质量 Issue）
    - P0 ≤ 3 个（核心差异化）
    - P1 ≤ 5 个（重要增强）
    - P2 ≤ 2 个（未来考虑）

  ## 环境变量

  BENCHMARK_TARGET: 目标仓库（默认 openclaw/openclaw）
  BENCHMARK_AUTO_CONFIRM: 跳过交互确认（默认 false）

---

# Product Insight

## 执行流程

```
状态恢复 → 数据收集 → 能力矩阵 → 价值三问 → HITL确认 → 批量产出
```

## Step 1: 状态恢复

```bash
./.claude/skills/product-insight/scripts/state.py summary
```

判断：SHA 为空 = 首次分析，走完整流程。

## Step 2: 数据收集

```bash
# Releases（最新 10 个）
gh release list --repo "$BENCHMARK_TARGET" --limit 10

# Issues（分页，至少 90 个）
gh issue list --repo "$BENCHMARK_TARGET" --limit 30 --state open
gh issue list --repo "$BENCHMARK_TARGET" --limit 30 --state open --page 2
gh issue list --repo "$BENCHMARK_TARGET" --limit 30 --state open --page 3

# README 和代码结构
gh repo view "$BENCHMARK_TARGET" --json description,topics
mcp__zread__get_repo_structure "$BENCHMARK_TARGET" "/"
```

## Step 3: 能力矩阵

```bash
# DivineSense 能力
./.claude/skills/product-insight/scripts/scan.py summary
```

构建差距矩阵：

| 功能领域 | OpenClaw      | DivineSense        | 差距     |
| :------- | :------------ | :----------------- | :------- |
| 人际记忆 | hooks         | episodic_memory 表 | 我们优势 |
| 部署     | Fly.io        | 单二进制           | 我们优势 |
| 多通道   | 12+ 通道      | Web + Geek         | 不做     |
| TTS      | Edge fallback | 无                 | P1       |

## Step 4: 价值三问

对每个差距进行分析：

| 阶段 | 问题             | 输出                       |
| :--- | :--------------- | :------------------------- |
| Q1   | 解决了什么痛点？ | 这不仅是[X]，而是【Y模式】 |
| Q2   | 为什么有价值？   | 主要价值：效率/成本/可能性 |
| Q3   | 能否更好实现？   | 利用我们的[优势]实现[价值] |

按价值密度排序：用户需求 × 我们优势 × 实现成本

## Step 5: HITL 确认

**此时才询问用户**，展示能力矩阵差距，确认优先级。

## Step 6: 批量产出 Issue

筛选标准：
```
价值密度 = 用户需求 × 我们优势 × 实现成本

强制分类：
  P0 (核心差异化) ≤ 3 个
  P1 (重要增强)     ≤ 5 个
  P2 (未来考虑)     ≤ 2 个
```

输出格式：

```
╔════════════════════════════════════════════════════════════╗
║  📊 执行摘要                                                ║
╠════════════════════════════════════════════════════════════╣
║  分析范围: OpenClaw @ abc1234 (+N commits)                 ║
║  核心发现: [一句话总结]                                     ║
║  战略建议: 做 X / 不做 Y / 差异化 Z                        ║
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║  🎯 战略建议                                                ║
╠════════════════════════════════════════════════════════════╣
║  P0 (核心差异化):                                          ║
║    • [功能] - 利用[优势]实现[价值]                         ║
║  P1 (重要增强):                                            ║
║    • [功能] - 用户强需求                                   ║
║  不做:                                                     ║
║    • [功能] - [理由]                                       ║
╚════════════════════════════════════════════════════════════╝
```

## 快捷指令

| 指令       | 行为                             |
| :--------- | :------------------------------- |
| "完整分析" | 首次对标，全面扫描（5-10 Issue） |
| "增量分析" | 基于已有状态，只看新增内容       |
| "深入 X"   | 聚焦分析 X                       |
| "总结"     | 当前阶段总结                     |

## 辅助脚本

```bash
# 状态管理
./.claude/skills/product-insight/scripts/state.py summary
./.claude/skills/product-insight/scripts/state.py query openclaw_sha

# 能力扫描
./.claude/skills/product-insight/scripts/scan.py summary
./.claude/skills/product-insight/scripts/scan.py has "pattern"
```

## 参考文档

| 文档               | 内容               |
| :----------------- | :----------------- |
| REFERENCE.md       | 价值三问详细方法论 |
| ADVANCED.md        | HITL 交互设计      |
| templates/issue.md | Issue 模板         |
