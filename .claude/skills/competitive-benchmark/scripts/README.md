# Competitive Benchmark Scripts

竞品对标辅助脚本，用于 `competitive-benchmark` Skill 和手动对标分析。

---

## 文件说明

| 脚本 | 用途 |
|:-----|:-----|
| **benchmark.sh** | 主入口：init/run/status 命令 |
| **state.sh** | 状态持久化管理 |
| **scan.sh** | DivineSense 能力矩阵扫描 |

---

## 快速开始

### 首次使用

```bash
# 1. 初始化对标状态
./scripts/benchmark/benchmark.sh init

# 2. 查看当前状态和能力
./scripts/benchmark/benchmark.sh status

# 3. 运行全量对标分析
./scripts/benchmark/benchmark.sh run

# 4. 或通过 Skill 执行对标
/competitive-benchmark
```

---

## benchmark.sh - 主入口

```bash
./scripts/benchmark/benchmark.sh init    # 初始化对标状态
./scripts/benchmark/benchmark.sh run     # 运行全量对标分析
./scripts/benchmark/benchmark.sh status  # 查看对标状态和能力
```

---

## state.sh - 状态管理

### 功能

- `append_state()` - 追加新状态记录
- `get_latest_state()` - 获取最新状态
- `query_state()` - 查询状态字段
- `show_state_summary()` - 显示状态摘要
- `init_state()` - 初始化状态文件

### 用法

```bash
# 方式一：source 后调用函数
source scripts/benchmark/state.sh
append_state "2026-02-02T10:00:00Z" "abc123" "def456" '["feat1"]' '[30]'
get_latest_state
query_state "openclaw_sha"

# 方式二：直接执行子命令
./scripts/benchmark/state.sh append "..." "..." "..." '[]' '[]'
./scripts/benchmark/state.sh get
./scripts/benchmark/state.sh query "openclaw_sha"
./scripts/benchmark/state.sh summary
./scripts/benchmark/state.sh count
./scripts/benchmark/state.sh init
```

### 安全特性

| 特性 | 描述 |
|:-----|:-----|
| **路径验证** | 状态文件必须在项目目录内 |
| **输入验证** | 时间戳、SHA、JSON 数组格式验证 |
| **文件锁** | 使用 flock 防止并发写入 |
| **安全构建** | 使用 jq 构建 JSON，防止注入 |

---

## scan.sh - 能力扫描

### 功能

- `scan_parrots()` - 扫描 AI 代理数量
- `scan_tools()` - 扫描工具数量（排除测试文件）
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

# 检查功能是否实现（固定字符串搜索，非正则）
./scripts/benchmark/scan.sh has "session.prun"

# 显示人类可读摘要
./scripts/benchmark/scan.sh summary
```

### 安全特性

| 特性 | 描述 |
|:-----|:-----|
| **输入验证** | has 模式只允许安全字符 |
| **固定搜索** | 使用 grep -F 禁用正则，防止注入 |
| **测试排除** | 工具扫描排除 *_test.go 文件 |
| **路径常量** | 提取硬编码路径，便于维护 |

---

## 输出示例

```bash
$ ./scripts/benchmark/benchmark.sh status
[2026-02-02 13:50:00][INFO] 查看对标状态...

对标状态摘要:
  上次对标: 2026-02-02T10:00:00Z
  OpenClaw SHA: abc1234
  DivineSense SHA: def5678
  已分析功能: 2 个
  已创建 Issue: 1 个

DivineSense 能力矩阵:
  AI 代理: 5 个
  工具: 8 个
  前端页面: 12 个
  数据库表: 25 个
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
init_state

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

## 安全审计历史

| 版本 | 日期 | 修复内容 |
|:-----|:-----|:---------|
| v1.4 | 2026-02-02 | **功能增强**：Parrot 扫描支持 v2 变体（schedule_parrot_v2.go） |
| v1.3 | 2026-02-02 | **安全修复**：输入验证、文件锁、jq 构建 JSON |
| v1.2 | 2026-02-02 | 初始版本 |

### 修复的问题

**P0 (严重)**:
- 正则表达式注入 → 使用 grep -F + 输入验证
- JSON 注入 → 使用 jq 构建而非 HERE 文档

**P1 (高优先级)**:
- 路径遍历 → 验证路径在项目目录内
- 并发竞态 → 使用 flock 文件锁
- 空文件处理 → 验证 JSON 格式
- 参数缺失 → 添加参数验证

**P2 (中优先级)**:
- 测试文件计入 → 排除 *_test.go
- 路径硬编码 → 提取为 readonly 常量
- 重复扫描 → 添加 SCAN_CACHE 缓存

---

*版本: v1.4 | 最后更新: 2026-02-02*
