# Product Insight Skill

> 产品洞察引擎 — 完整调研、系统对比、批量产出

**版本**: v2.6.0

## 简介

`product-insight` 是一个 Claude Code Skill，用于产品洞察分析：

1. **洞察驱动** — 价值三问框架，深入理解"为什么"
2. **HITL 协作** — 关键决策点与人类交互
3. **战略输出** — 做/不做/差异化，而非简单追赶
4. **模式抽象** — 提取可迁移的设计模式
5. **批量产出** — 一次分析生成 5-10 个高质量 Issue
6. **容错机制** — MCP 工具失败时自动降级，不中断分析
7. **双模式执行** — 首次深度调研 vs 增量快速聚焦

## 使用方法

```bash
# 触发产品洞察分析
/product-insight

# 或使用 Skill 工具
Skill "product-insight" "执行目标产品洞察分析"
```

## v2.6.0 更新内容

- **双模式执行**：区分首次分析（深度）和增量分析（快速）
  - 首次：文档+源码+社区全面调研
  - 增量：Releases+Issues 快速聚焦
- 官方文档解读流程
- 核心源码解读流程
- 用户痛点挖掘流程

## v2.5.0 更新内容

- 添加 MCP 工具容错策略
- 添加数据源降级机制
- 添加 subprocess 超时保护
- 添加故障排查指南
- 添加输出说明

## 独立脚本

| 脚本 | 用途 |
|:-----|:-----|
| **scripts/core.py** | 主入口：init/run/status 命令 |
| **scripts/state.py** | 状态持久化管理 |
| **scripts/scan.py** | DivineSense 能力矩阵扫描 |

详见: [scripts/README.md](./scripts/README.md)

## 工作流程

```
多源收集 → 价值三问 → 模式抽象 → 创造性转化 → 洞察产出
   ↑_________HITL_交互点__________↑
```

## 价值三问框架

```
1. 问题本质：这个功能解决了用户的什么痛点？
2. 价值来源：为什么用户认为它有价值？
3. 创造性转化：同样的价值，我们能否用更好方式实现？
```

## 过滤规则

不再"自动过滤"，而是"分析后判断"：

| 类型 | 处理方式 |
|:-----|:---------|
| TypeScript 特异性 | 分析问题，判断能否 Go 实现 |
| 多渠道集成 | 分析价值，判断是否符合个人场景 |
| 外部依赖 | 分析必要性，判断能否本地化 |

## 文件结构

```
.claude/skills/product-insight/
├── SKILL.md          # 核心（执行协议）
├── REFERENCE.md      # 方法论（价值三问框架）
├── ADVANCED.md       # HITL 交互设计
├── INSIGHT.md        # 洞察报告模板
├── README.md         # 本文件
├── templates/        # Issue 模板
│   └── issue.md
└── scripts/          # Python 脚本
    ├── core.py       # 主入口
    ├── state.py     # 状态管理
    ├── scan.py      # 能力扫描
    └── README.md
```

## 环境变量

| 变量 | 说明 | 默认值 |
|:-----|:-----|:-------|
| `BENCHMARK_TARGET` | 目标仓库 | `openclaw/openclaw` |
| `BENCHMARK_AUTO_CONFIRM` | 跳过交互确认 | `false` |

## 相关文档

- [Idea Researcher](../idea-researcher/) — 深度技术调研与方案设计（可对 product-insight 发现的功能进行细化）
- [CLAUDE.md](../../../CLAUDE.md) — 项目指南
- [ARCHITECTURE.md](../../../docs/dev-guides/ARCHITECTURE.md) — 系统架构
