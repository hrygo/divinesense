# Product Insight Scripts

产品洞察辅助脚本。

## 文件说明

| 脚本 | 用途 |
|:-----|:-----|
| **core.py** | 主入口：init/run/status 命令 |
| **state.py** | 状态持久化管理 + StateManager 类 |
| **scan.py** | 能力矩阵扫描 + CapabilityScanner 类 |

所有脚本位于 `scripts/` 目录，可独立运行或作为模块导入。

## 快速开始

```bash
# 初始化洞察状态
python core.py init

# 查看当前状态和能力
python core.py status

# 运行全量洞察分析
python core.py run
```

## core.py - 主入口

```bash
python .claude/skills/product-insight/scripts/core.py init    # 初始化
python .claude/skills/product-insight/scripts/core.py run     # 运行分析
python .claude/skills/product-insight/scripts/core.py status  # 查看状态
```

## state.py - 状态管理

```bash
python .claude/skills/product-insight/scripts/state.py get
python .claude/skills/product-insight/scripts/state.py query "openclaw_sha"
python .claude/skills/product-insight/scripts/state.py summary
```

## scan.py - 能力扫描

```bash
python .claude/skills/product-insight/scripts/scan.py matrix   # 完整矩阵 (JSON)
python .claude/skills/product-insight/scripts/scan.py summary  # 人类可读摘要
python .claude/skills/product-insight/scripts/scan.py has "pattern"  # 检查功能
```

## 环境依赖

Python 3.10+ required

| 外部工具 | 用途 |
|:--------|:-----|
| **gh** | GitHub CLI |
| **git** | 版本控制 |
