# Competitive Benchmark Skill

> 竞品洞察引擎 v2.1 — LLM 优化的执行协议，HITL 协作分析

## 简介

`competitive-benchmark` 是一个 Claude Code Skill，从 "Issue 生成器" 升级为 "洞察引擎"：

1. **洞察驱动** — 价值三问框架，深入理解"为什么"
2. **HITL 协作** — 关键决策点与人类交互，共同思考
3. **战略输出** — 做/不做/差异化，而非简单追赶
4. **模式抽象** — 提取可迁移的设计模式，跨技术栈
5. **LLM 优化** — 内嵌执行协议，明确 guardrails，结构化输出

## 使用方法

```bash
# 触发竞品分析
/competitive-benchmark

# 或使用 Skill 工具
Skill "competitive-benchmark" "执行 OpenClaw 洞察分析"
```

## 独立脚本

| 脚本 | 用途 |
|:-----|:-----|
| **scripts/state.sh** | 状态持久化管理 |
| **scripts/scan.sh** | DivineSense 能力矩阵扫描 |

详见: [scripts/README.md](./scripts/README.md)

## 工作流程

```
多源收集 → 价值三问 → 模式抽象 → 创造性转化 → 洞察产出
   ↑_________HITL_交互点__________↑
```

## 核心变化：v1.3 → v2.0

| 维度 | v1.3 | v2.0 |
|:-----|:-----|:-----|
| **定位** | OpenClaw 追随者 | 赛道洞察专家 |
| **目标** | 发现功能差距 | 理解成功模式 |
| **方法** | 技术契合度评分 | 价值三问框架 |
| **产出** | Issue 列表 | 洞察+战略+Issue |
| **交互** | 无 | HITL 协作 |

## 价值三问框架

```
1. 问题本质：这个功能解决了用户的什么痛点？
2. 价值来源：为什么用户认为它有价值？
3. 创造性转化：同样的价值，我们能否用更好方式实现？
```

## 过滤规则 2.0

不再"自动过滤"，而是"分析后判断"：

| 类型 | v1.3 | v2.0 |
|:-----|:-----|:-----|
| TypeScript 特异性 | 自动过滤 | 分析问题，判断能否 Go 实现 |
| 多渠道集成 | 自动过滤 | 分析价值，判断是否符合个人场景 |
| 外部依赖 | 自动过滤 | 分析必要性，判断能否本地化 |

## 文件结构

```
.claude/skills/competitive-benchmark/
├── SKILL.md          # 核心（精简，5阶段工作流）
├── REFERENCE.md      # 方法论（价值三问框架）
├── ADVANCED.md       # HITL 交互设计
├── INSIGHT.md        # 洞察报告模板
├── README.md         # 本文件
├── templates/        # Issue/Report 模板
│   ├── issue.md      # Issue 模板
│   └── report.md     # 报告模板
└── scripts/          # 对标脚本
    ├── benchmark.sh  # 主入口
    ├── state.sh      # 状态管理
    ├── scan.sh       # 能力扫描
    └── README.md     # 脚本说明
```

## 版本历史

| 版本 | 日期 | 变更内容 |
|:-----|:-----|:---------|
| v2.1 | 2026-02-02 | **LLM 优化**：内嵌执行协议、明确 guardrails、结构化输出 |
| v2.0 | 2026-02-02 | **洞察引擎**：HITL 协作、价值三问、战略输出 |
| v1.3 | 2026-02-02 | Agent 对标：Parrot 扫描增强、Pi Agent 架构对比 |
| v1.2 | 2026-02-02 | 完善文档：统一版本号、独立脚本 |
| v1.1 | 2026-02-02 | 实时动态：零硬编码、增量对比、状态持久化 |
| v1.0 | 2026-02-02 | 初始版本 |

## 相关文档

- [Idea Researcher](../idea-researcher/) — 创意调研 Skill
- [CLAUDE.md](../../../CLAUDE.md) — 项目指南
- [ARCHITECTURE.md](../../../docs/dev-guides/ARCHITECTURE.md) — 系统架构
