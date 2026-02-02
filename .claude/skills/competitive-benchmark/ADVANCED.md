# Advanced - 竞品对标高级功能

> 增量对比算法、状态持久化、自动进化机制。

---

## 增量对比算法

### 核心思想

**增量对比**：基于上次对标状态，仅分析 OpenClaw 的新增功能。

> 以下代码为**伪代码示例**，用于说明算法逻辑，`get_latest_commit` 等函数需通过 MCP 工具实现。

```python
def incremental_benchmark(last_state):
    """增量对比算法"""
    # 1. 获取当前状态
    current_openclaw_sha = get_latest_commit("openclaw/openclaw")  # MCP: gh api
    current_divinesense_sha = get_latest_commit("divinesense")     # MCP: git rev-parse

    # 2. 检测是否有新提交
    if current_openclaw_sha == last_state["openclaw_sha"]:
        return {
            "has_changes": False,
            "message": f"OpenClaw 无变化，上次对标: {last_state['timestamp']}"
        }

    # 3. 获取变更文件列表
    changed_files = get_changed_files(
        "openclaw/openclaw",
        since=last_state["openclaw_sha"]
    )

    # 4. 过滤相关文件
    relevant_files = filter_relevant_files(changed_files)

    # 5. 提取新功能
    new_features = extract_features(relevant_files)

    # 6. 排除已分析功能
    new_features = [
        f for f in new_features
        if f["name"] not in last_state["analyzed_features"]
    ]

    return {
        "has_changes": True,
        "new_features": new_features,
        "total_changes": len(changed_files)
    }
```

### 变更检测

```bash
# 获取两个 SHA 之间的文件变更
gh api repos/openclaw/openclaw/compare/$LAST_SHA...$CURRENT_SHA | \
  jq -r '.files[].filename'

# 过滤核心功能目录
grep -E "^(src/sessions|src/plugin-sdk|src/commands|extensions/)"
```

### CHANGELOG 增量解析

```bash
# 获取上次对标日期后的 CHANGELOG 条目
LAST_DATE=$(cat docs/research/benchmark/state.jsonl | \
  jq -r '.[-1].timestamp' | cut -dT -f1)

# 解析 CHANGELOG 格式（OpenClaw 使用日期格式）
parse_changelog_since() {
    local since_date=$1
    gh api repos/openclaw/openclaw/contents/CHANGELOG.md | \
      jq -r '.content' | base64 -d | \
      awk -v date="$since_date" '
      /^## / {
        if (match($0, /^## ([0-9]{4}\.[0-9]+\.[0-9]+)/, m)) {
          current_date = m[1]
          next
        }
      }
      current_date >= date { print }
      '
}
```

---

## 状态持久化

### 状态文件格式

```json
// docs/research/benchmark/state.jsonl (JSONL 格式，每行一条记录)
{"timestamp":"2026-02-01T10:00:00Z","openclaw_sha":"abc123","divinesense_sha":"def456","analyzed_features":[],"discovered_functions":[],"created_issues":[]}
{"timestamp":"2026-02-02T10:00:00Z","openclaw_sha":"xyz789","divinesense_sha":"ghi012","analyzed_features":["会话修剪"],"discovered_functions":[{"name":"会话修剪","category":"session","path":"src/sessions/pruning.ts"}],"created_issues":[30]}
```

### 状态操作命令

> **脚本**: `scripts/state.sh`

```bash
# 方式一：直接执行脚本
./.claude/skills/competitive-benchmark/scripts/state.sh append "2026-02-02T10:00:00Z" "abc123" "def456" '["feat1"]' '[30]'
./.claude/skills/competitive-benchmark/scripts/state.sh get
./.claude/skills/competitive-benchmark/scripts/state.sh query "openclaw_sha"
./.claude/skills/competitive-benchmark/scripts/state.sh summary

# 方式二：source 后使用（更灵活）
source .claude/skills/competitive-benchmark/scripts/state.sh
append_state "$timestamp" "$oc_sha" "$ds_sha" "$features" "$issues"
get_latest_state
query_state "openclaw_sha"
show_state_summary
```

### 状态查询

```bash
# 显示完整摘要
./.claude/skills/competitive-benchmark/scripts/state.sh summary

# 查询特定字段
last_run=$(./.claude/skills/competitive-benchmark/scripts/state.sh query timestamp)
analyzed_count=$(./.claude/skills/competitive-benchmark/scripts/state.sh get | jq -r '.analyzed_features | length')
issues_count=$(./.claude/skills/competitive-benchmark/scripts/state.sh get | jq -r '.created_issues | length')
```

---

## 自动化进化

### 自我更新触发条件

> 以下代码为**伪代码示例**，用于说明检测逻辑。

```python
def should_self_update():
    """检测是否需要自我更新"""
    last_state = get_latest_state()
    current = scan_current_state()

    triggers = {
        "openclaw_updated": last_state["openclaw_sha"] != current["openclaw_sha"],
        "divinesense_updated": last_state["divinesense_sha"] != current["divinesense_sha"],
        "new_parrot": current["parrot_count"] > last_state.get("parrot_count", 0),
        "new_tool": current["tool_count"] > last_state.get("tool_count", 0),
    }

    return any(triggers.values()), triggers
```

### 进化建议生成

```python
def generate_evolution_suggestion(triggers):
    """生成进化建议"""
    suggestions = []

    if triggers["openclaw_updated"]:
        suggestions.append({
            "type": "info",
            "message": f"OpenClaw 有新提交，建议运行增量对标"
        })

    if triggers["divinesense_updated"]:
        suggestions.append({
            "type": "action",
            "message": f"DivineSense 有新代码，建议更新能力矩阵",
            "action": "update_capability_matrix"
        })

    if triggers["new_parrot"]:
        suggestions.append({
            "type": "info",
            "message": f"发现新代理，可能需要更新对标策略"
        })

    return suggestions
```

---

## 智能分组算法

### 分组策略

> 以下代码为**伪代码示例**，用于说明算法逻辑。

```python
def group_features(features: List[FeatureGap]) -> List[FeatureGroup]:
    """智能分组算法"""
    groups = {}

    for feature in features:
        category = feature["category"]
        if category not in groups:
            groups[category] = {
                "name": CATEGORY_NAMES.get(category, category),
                "features": [],
                "techFit": [],
                "userValue": [],
                "paths": []
            }

        groups[category]["features"].append(feature["name"])
        groups[category]["techFit"].append(feature["techFit"])
        groups[category]["userValue"].append(feature["userValue"])
        groups[category]["paths"].append(feature.get("path", ""))

    # 计算分组优先级和估算
    result = []
    for category, data in groups.items():
        avg_tech = sum(data["techFit"]) / len(data["techFit"])
        avg_value = sum(data["userValue"]) / len(data["userValue"])
        priority = avg_tech * 0.4 + avg_value * 0.6

        result.append({
            "name": data["name"],
            "features": data["features"],
            "paths": data["paths"],
            "priority": priority,
            "estimatedEffort": estimate_effort(category, len(data["features"])),
            "riskLevel": assess_risk(avg_tech, avg_value)
        })

    return sorted(result, key=lambda x: x["priority"], reverse=True)
```

### 工作量估算

> 以下代码为**伪代码示例**，用于说明估算逻辑。

```python
def estimate_effort(category: str, feature_count: int) -> int:
    """估算工作量（人周）"""
    BASE_EFFORT = {
        "session": 1,      # 会话管理：1-2 周
        "plugin": 2,       # 插件系统：2-3 周
        "agent": 1,        # 代理功能：1-2 周
        "media": 2,        # 媒体处理：2-3 周
        "ui": 1,           # UI 组件：1 周
    }

    base = BASE_EFFORT.get(category, 1)
    return base + feature_count * 0.5  # 每个额外功能 +0.5 周
```

---

## 重复检测

### Issue 重复检测

```bash
# 搜索关键词组合
check_duplicate_issue() {
    local feature_name=$1
    local keywords=($(echo "$feature_name" | tr ' ' '\n'))

    for kw in "${keywords[@]}"; do
        results=$(gh issue list --repo "$REPO" --search "$kw" --state all --limit 5)
        if [ -n "$results" ]; then
            echo "可能重复: $kw"
        fi
    done
}
```

### 功能去重

> 以下代码为**伪代码示例**，用于说明去重逻辑。

```python
def is_duplicate_feature(feature_name: str, analyzed_features: List[str]) -> bool:
    """检测功能是否已分析"""
    # 精确匹配
    if feature_name in analyzed_features:
        return True

    # 模糊匹配
    for analyzed in analyzed_features:
        similarity = calculate_similarity(feature_name, analyzed)
        if similarity > 0.8:
            return True

    return False
```

---

## 批量 Issue 创建

### 创建策略

> 以下代码为**伪代码示例**，实际使用 MCP GitHub 工具创建 Issue。

```python
def create_issues(groups: List[FeatureGroup], repo: str) -> List[int]:
    """批量创建 Issue"""
    created_issues = []

    for group in groups:
        if group["priority"] < 0.3:
            continue  # 跳过低优先级

        # 检查重复
        if is_duplicate_group(group, repo):
            print(f"跳过重复: {group['name']}")
            continue

        # 创建 Issue
        issue_number = create_github_issue(
            repo=repo,
            title=format_issue_title(group),
            body=format_issue_body(group)
        )

        created_issues.append(issue_number)
        print(f"创建 Issue: #{issue_number} - {group['name']}")

    return created_issues
```

---

## Skill 进化记录

| 版本 | 日期 | 变更内容 |
|:-----|:-----|:---------|
| v1.3 | 2026-02-02 | **Agent 对标**：Parrot 扫描增强、Pi Agent 架构对比、Skills 维度对标 |
| v1.2 | 2026-02-02 | **完善文档**：统一版本号、添加错误处理、改进模板 |
| v1.1 | 2026-02-02 | **实时动态**：零硬编码、增量对比、状态持久化 |
| v1.0 | 2026-02-02 | 初始版本：全面对标、价值评估、智能分组 |

### 未来方向

- [x] v1.1: 增量对比模式（基于 CHANGELOG）
- [x] v1.2: 语义相似度重复检测（基础实现）
- [x] v1.3: Agent/Pi Agent 架构对标
- [ ] v1.4: 自动化触发（GitHub Webhook）
- [ ] v2.0: 多竞品支持（Memos、Obsidian 等）

---

## 文件结构

```
.claude/skills/competitive-benchmark/
├── SKILL.md          # 核心（6 阶段状态机）
├── REFERENCE.md      # 参考（动态发现方法）
├── ADVANCED.md       # 高级（本文档）
├── README.md         # 介绍
├── templates/        # Issue/Report 模板
│   ├── issue.md      # Issue 模板
│   └── report.md     # 报告模板
└── scripts/          # 对标脚本（自包含）
    ├── benchmark.sh  # 主入口：init/run/status
    ├── state.sh      # 状态持久化管理
    ├── scan.sh       # 能力矩阵扫描
    └── README.md     # 脚本使用说明
```

---

## 快速参考

### 核心命令

```bash
# 获取 OpenClaw 最新 SHA
gh api repos/openclaw/openclaw/commits | jq -r '.sha'

# 获取 DivineSense 当前 SHA
git rev-parse HEAD

# 读取状态文件
tail -1 docs/research/benchmark/state.jsonl | jq -r '.'

# 扫描 DivineSense 代理（使用脚本）
./.claude/skills/competitive-benchmark/scripts/scan.sh parrot-names

# 扫描 DivineSense 工具（使用脚本）
find plugin/ai/agent/tools -name "*.go"
```

---

*文档版本：v1.3 | 最后更新：2026-02-02*
