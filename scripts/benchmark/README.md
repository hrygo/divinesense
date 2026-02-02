# Competitive Benchmark Scripts

竞品对标辅助脚本，用于 `competitive-benchmark` Skill 和手动对标分析。

---

## 文件说明

| 脚本 | 用途 |
|:-----|:-----|
| **state.sh** | 状态持久化管理 |
| **scan.sh** | DivineSense 能力矩阵扫描 |

---

## state.sh - 状态管理

### 功能

- `append_state()` - 追加新状态记录
- `get_latest_state()` - 获取最新状态
- `query_state()` - 查询状态字段
- `show_state_summary()` - 显示状态摘要

### 用法

```bash
# 方式一：source 后调用函数
source scripts/benchmark/state.sh
append_state "2026-02-02T10:00:00Z" "abc123" "def456" '["feat1"]' '[30]'
get_latest_state
query_state "openclaw_sha"

# 方式二：直接执行子命令
./scripts/benchmark/state.sh append "2026-02-02T10:00:00Z" "abc123" "def456" '["feat1"]' '[30]'
./scripts/benchmark/state.sh get
./scripts/benchmark/state.sh query "openclaw_sha"
./scripts/benchmark/state.sh summary
./scripts/benchmark/state.sh count
```

### 状态文件格式

状态保存在 `docs/research/benchmark/state.jsonl`：

```json
{"timestamp":"2026-02-02T10:00:00Z","openclaw_sha":"abc123","divinesense_sha":"def456","analyzed_features":["feat1"],"discovered_functions":[],"created_issues":[30]}
```

---

## scan.sh - 能力扫描

### 功能

- `scan_parrots()` - 扫描 AI 代理数量
- `scan_tools()` - 扫描工具数量
- `scan_pages()` - 扫描前端页面数量
- `scan_tables()` - 扫描数据库表数量
- `has_feature()` - 检查功能是否实现
- `generate_matrix()` - 生成 JSON 格式能力矩阵

### 用法

```bash
# 生成完整能力矩阵 (JSON)
./scripts/benchmark/scan.sh
./scripts/benchmark/scan.sh matrix

# 扫描单项
./scripts/benchmark/scan.sh parrots      # 代理数量
./scripts/benchmark/scan.sh tools        # 工具数量
./scripts/benchmark/scan.sh pages        # 页面数量
./scripts/benchmark/scan.sh tables       # 表数量

# 列出名称
./scripts/benchmark/scan.sh parrot-names
./scripts/benchmark/scan.sh tool-names

# 检查功能是否实现
./scripts/benchmark/scan.sh has "session.*prun"

# 显示人类可读摘要
./scripts/benchmark/scan.sh summary
```

### 输出示例

```bash
$ ./scripts/benchmark/scan.sh summary
DivineSense 能力矩阵:
  AI 代理: 5 个
  工具: 8 个
  前端页面: 12 个
  数据库表: 25 个

AI 代理列表:
  - amazing
  - evolution
  - geek
  - memo
  - schedule

工具列表:
  - memo_search
  - scheduler
  ...
```

---

## 环境依赖

| 工具 | 用途 | 安装 |
|:-----|:-----|:-----|
| **jq** | JSON 处理 | `brew install jq` |
| **gh** | GitHub CLI | `brew install gh` |
| **git** | 版本控制 | 系统内置 |

---

## 与 Skill 集成

在 `competitive-benchmark` Skill 中使用：

```bash
# 初始化状态
source scripts/benchmark/state.sh

# 获取上次对标状态
LAST_SHA=$(query_state "openclaw_sha")

# 扫描当前能力
PARROTS=$(./scripts/benchmark/scan.sh parrots)
TOOLS=$(./scripts/benchmark/scan.sh tools)

# 保存新状态
append_state "$TIMESTAMP" "$OC_SHA" "$DS_SHA" "$FEATURES" "$ISSUES"
```

---

## 状态文件位置

- 默认: `docs/research/benchmark/state.jsonl`
- 可通过环境变量覆盖: `STATE_FILE=/path/to/state.jsonl`

---

*版本: v1.2 | 最后更新: 2026-02-02*
