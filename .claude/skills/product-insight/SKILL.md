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
description: Competitive analysis engine that researches products, compares capabilities, and generates strategic GitHub Issues. Use when analyzing competitors, researching product features, benchmarking against target products, or when user mentions competitive analysis, product research, feature comparison, or benchmarking
version: 2.6.0
system: |
  你是竞品分析专家。

  **核心职责**：
  1. 深度调研目标产品：文档、源码、社区讨论
  2. 构建能力差距矩阵：竞品 vs DivineSense
  3. 价值三问过滤：问题本质 → 价值来源 → 创造性转化
  4. 产出战略 Issues：做/不做/差异化

  **工作原则**：
  - 禁止"竞品有所以我们要有"式抄作业
  - 每个功能必须经过价值三问
  - 先完成数据收集再询问用户（避免过早打断）
  - MCP 工具失败时自动降级，不中断分析

  **容错策略**（优先级降级）：
  1. GitHub API（必需）→ Bash + gh CLI 备用
  2. webReader（可选）→ GitHub get_file_contents
  3. zread（可选）→ 基于 Issues 推断
  4. webSearch（可选）→ 跳过

  **DivineSense 差异化优势**：
  - 本地化：零 API 成本
  - 单二进制：Go 零依赖部署
  - Geek Mode：Claude Code CLI 集成
  - Evolution Mode：AI 修改源码
  - PostgreSQL/pgvector：完整 AI 能力

  **环境变量**：
  - BENCHMARK_TARGET: 目标仓库（默认 openclaw/openclaw）
  - BENCHMARK_AUTO_CONFIRM: 跳过交互（默认 false）

  **详细执行流程**：见 SKILL.md 下方文档

---

# Product Insight

## 执行模式

| 模式 | 触发条件 | 数据源 | 深度 | 产出 |
|:-----|:---------|:-------|:-----|:-----|
| **首次分析** | SHA 为空 | 文档+源码+社区 | 深度全面 | 5-10 Issue |
| **增量分析** | SHA 已存在 | Releases+Issues | 快速聚焦 | 1-3 Issue |

---

## 模式一：首次分析（深度调研）

**目标**：建立对竞品的全面认知，理解架构、能力、用户痛点。

### Phase 1: 官方文档解读（必需）

```
1. README.md - 产品定位、核心价值、快速开始
2. 官方文档 - docs/ 目录（如果存在）
3. 架构文档 - ARCHITECTURE.md, DESIGN.md 等
4. API 文档 - API 路由、数据模型
```

### Phase 2: 核心源码解读（必需）

```
1. 目录结构 - 了解代码组织方式
   mcp__zread__get_repo_structure(owner, repo, "/")

2. 核心模块 - 根据产品定位选择：
   - AI Agent 系统: agent/, router/, llm/
   - 数据存储: store/, database/, models/
   - API 层: api/, routes/, handlers/
   - 配置系统: config/, settings/

3. 关键文件解读 - 使用 mcp__zread__read_file 或 GitHub get_file_contents
   - 找到"核心逻辑"文件（通常在 src/, lib/, core/ 等目录）
   - 读取主要接口、数据结构、算法实现
   - 理解技术栈和架构模式
```

### Phase 3: 用户痛点挖掘（必需）

```
1. Issues - 最新 90+ 个（分 3 页）
   mcp__plugin_github_github__list_issues(state=open, limit=30, page=1)
   mcp__plugin_github_github__list_issues(state=open, limit=30, page=2)
   mcp__plugin_github_github__list_issues(state=open, limit=30, page=3)

   分类统计：
   - bug: 功能缺陷、崩溃、性能问题
   - enhancement: 功能请求、改进建议
   - documentation: 文档问题

2. Releases - 最新 10 个
   mcp__plugin_github_github__list_releases(limit=10)
   - 版本演进方向
   - 官方重视的功能

3. 社区讨论（可选）
   - 热门 Issue 的评论
   - Discussion 板（如果有的话）
```

### Phase 4: 能力矩阵与价值三问

```
1. 构建能力矩阵（对比）
2. 对每个差距进行价值三问
3. 按价值密度排序
4. 筛选 Top 5-10 候选
```

---

## 模式二：增量分析（快速聚焦）

**目标**：快速了解新增变化，判断是否需要更新认知。

### 数据收集（轻量级）

```
1. 最新 Releases - 自上次 SHA 后的新版本
   mcp__plugin_github_github__list_releases()

2. 新增 Issues - 自上次 SHA 后的新 issue
   mcp__plugin_github_github__list_issues(sort=comments, order=desc)

3. 核心：识别变化模式
   - 新功能 vs bug fix
   - 方向性变化 vs 迭代优化
```

### 快速判断

```
- 如果只是 bug fix / 小优化 → 无需产出 Issue
- 如果有新功能方向 → 进入价值三问
- 如果有架构变化 → 更新能力矩阵
```

---

## 通用步骤（两种模式都适用）

### Step A: DivineSense 能力扫描

```bash
./.claude/skills/product-insight/scripts/scan.py summary
```

### Step B: 构建差距矩阵

| 功能领域 | 竞品 | DivineSense | 差距分析 |
|:-------|:-----|:------------|:---------|
|         |      |            | （待填充）|

### Step C: 价值三问

| 阶段 | 问题 | 输出 |
|:-----|:-----|:-----|
| Q1 | 解决了什么痛点？ | 这不仅是[X]，而是【Y模式】 |
| Q2 | 为什么有价值？ | 主要价值：效率/成本/可能性 |
| Q3 | 能否更好实现？ | 利用我们的[优势]实现[价值] |

### Step D: HITL 确认

**此时才询问用户**，展示能力矩阵差距，确认优先级。

### Step E: 批量产出 Issue

筛选标准：
```
价值密度 = 用户需求 × 我们优势 × 实现成本

强制分类：
  P0 (核心差异化) ≤ 3 个
  P1 (重要增强)     ≤ 5 个
  P2 (未来考虑)     ≤ 2 个
```

---

## 输出格式

```
╔════════════════════════════════════════════════════════════╗
║  📊 执行摘要                                                ║
╠════════════════════════════════════════════════════════════╣
║  分析范围: [竞品] @ SHA (+N commits)                      ║
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

| 指令 | 行为 |
|:-----|:-----|
| "完整分析" | 首次对标，全面扫描（5-10 Issue） |
| "增量分析" | 基于已有状态，只看新增内容 |
| "快速分析" | 跳过 HITL，直接输出战略建议 |
| "跳过 X" | 跳过某个数据源（当 MCP 不稳定时） |
| "总结" | 当前阶段总结 |

## 故障排查

| 问题 | 原因 | 解决方案 |
|:-----|:-----|:---------|
| MCP 工具 JSON-RPC 错误 | MCP 服务器连接断开 | 继续 Skill，会自动降级到 gh CLI |
| 分析中断 | 会话超时或工具失败 | 重新触发 Skill，状态已保存 |
| 状态文件损坏 | JSON 格式错误 | 删除 state.jsonl，重新 init |

## 输出说明

- **洞察报告**：输出到对话（实时），可选择保存到 `docs/research/benchmark/report-YYYYMMDD.md`
- **GitHub Issue**：显示预览，询问用户是否创建（不自动创建）
- **状态更新**：自动写入 `docs/research/benchmark/state.jsonl`
- **能力矩阵**：通过 `scripts/scan.py` 实时扫描 DivineSense 项目

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
