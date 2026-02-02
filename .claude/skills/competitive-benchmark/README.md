# Competitive Benchmark Skill

> 竞品对标助手 v1.3 — 实时动态追踪 OpenClaw，增量对比，智能生成 DivineSense 原子化 Issue

## 简介

`competitive-benchmark` 是一个 Claude Code Skill，用于：

1. **实时动态发现** — 从 GitHub API 获取 OpenClaw 最新功能
2. **增量对比** — 仅分析上次对标后的新变化
3. **价值评估** — 技术契合度 × 用户价值加权评分
4. **智能分组** — 相关功能合并，避免 Issue 碎片化
5. **状态持久化** — 保存对标历史，支持断点续传

## 使用方法

```bash
# 触发对标分析
/competitive-benchmark

# 或使用 Skill 工具
Skill "competitive-benchmark" "执行 OpenClaw 增量对标"
```

## 独立脚本

除了 Skill 文档中的示例代码，项目还提供了可独立执行的脚本：

| 脚本 | 用途 |
|:-----|:-----|
| **scripts/state.sh** | 状态持久化管理 |
| **scripts/scan.sh** | DivineSense 能力矩阵扫描 |

详见: [scripts/README.md](./scripts/README.md)

## 工作流程

```
状态恢复 → 增量发现 → 动态对比 → 价值评估 → 智能分组 → Issue 生成 → 状态持久化
```

## 核心特性

| 特性 | v1.0 | v1.1 | v1.2 | v1.3 |
|:-----|:-----|:-----|:-----|:-----|
| 实时动态发现 | ❌ | ✅ GitHub API | ✅ | ✅ |
| 增量对比 | ❌ | ✅ 基于 SHA | ✅ | ✅ |
| 状态持久化 | ❌ | ✅ state.jsonl | ✅ | ✅ |
| 独立脚本 | ❌ | ❌ | ✅ scripts/ | ✅ |
| 智能过滤 | ✅ | ✅ 增强规则 | ✅ | ✅ |
| 自动关闭已实现 | ❌ | ✅ 动态检测 | ✅ | ✅ |
| Parrot 扫描增强 | ❌ | ❌ | ❌ | ✅ v2 支持 |
| Agent 架构对比 | ❌ | ❌ | ❌ | ✅ Pi Agent vs Parrot |
| Skills 维度对标 | ❌ | ❌ | ❌ | ✅ 40+ 技能分析 |

## 对标策略

### 技术契合度 (0-1)

- `1.0` — Go 原生支持，可直接复用设计
- `0.7` — 需适配但有参考实现
- `0.4` — 需重构或使用不同技术
- `0.1` — TypeScript 特异性，难以迁移

### 用户价值 (0-1)

- `1.0` — 高频使用，解决核心痛点
- `0.7` — 中频使用，显著提升体验
- `0.4` — 低频使用，锦上添花
- `0.1` — 边缘场景，用户需求弱

### 优先级计算

```
优先级 = 技术契合度 × 0.4 + 用户价值 × 0.6

优先级 < 0.3 → 自动过滤
优先级 ≥ 0.7 → 高优先级
```

## 过滤规则

自动过滤以下功能：

- **TypeScript 特异性**：jiti、tsx、oxlint、vitest 等
- **多渠道集成**：WhatsApp、Discord、Telegram 等（不同产品定位）
- **外部服务依赖**：npm registry、GitHub API 等

## 文件结构

```
.claude/skills/competitive-benchmark/
├── SKILL.md          # 核心（6 阶段状态机）
├── REFERENCE.md      # 参考（动态发现方法）
├── ADVANCED.md       # 高级（增量算法、状态持久化）
├── README.md         # 本文件
├── templates/        # Issue/Report 模板
│   ├── issue.md      # Issue 模板
│   └── report.md     # 报告模板
└── scripts/          # 对标脚本（自包含）
    ├── benchmark.sh  # 主入口：init/run/status
    ├── state.sh      # 状态持久化管理
    ├── scan.sh       # 能力矩阵扫描
    └── README.md     # 脚本使用说明
```

## 状态文件

对标历史保存在 `docs/research/benchmark/state.jsonl`：

```json
{"timestamp":"2026-02-02T10:00:00Z","openclaw_sha":"abc123","divinesense_sha":"def456","analyzed_features":["会话修剪"],"discovered_functions":[],"created_issues":[30]}
```

## 版本历史

| 版本 | 日期 | 变更内容 |
|:-----|:-----|:---------|
| v1.3 | 2026-02-02 | **Agent 对标**：Parrot 扫描增强、Pi Agent 架构对比、Skills 维度对标 |
| v1.2 | 2026-02-02 | **完善文档**：统一版本号、添加错误处理、提取独立脚本 |
| v1.1 | 2026-02-02 | **实时动态**：零硬编码、增量对比、状态持久化 |
| v1.0 | 2026-02-02 | 初始版本 |

## 相关文档

- [Idea Researcher](../idea-researcher/) — 创意调研 Skill
- [CLAUDE.md](../../../CLAUDE.md) — 项目指南
- [ARCHITECTURE.md](../../../docs/dev-guides/ARCHITECTURE.md) — 系统架构
