# Reference - 竞品对标参考文档

> 动态功能发现方法、DivineSense 能力扫描、对标方法论。

---

## 动态功能发现

> **核心原则**：零硬编码，所有数据运行时从 GitHub API 动态获取。

### OpenClaw 实时数据获取

```bash
# 获取 OpenClaw 最新版本信息
gh repo view openclaw/openclaw --json name,description,defaultBranchRef,releases

# 获取最新 Release
gh release list --repo openclaw/openclaw --limit 1

# 获取最新 Commit
gh api repos/openclaw/openclaw/commits | jq -r '.sha'

# 获取目录结构（动态发现功能模块）
mcp__zread__get_repo_structure "openclaw/openclaw" "/"
```

### 功能分类动态推断

> 以下代码为**伪代码示例**，用于说明算法逻辑，实际使用时需根据 Skill 环境调整。

```python
# 基于目录路径推断功能分类
FUNCTION_CATEGORIES = {
    'session': ['src/sessions', 'src/memory', 'src/context'],
    'plugin': ['src/plugin-sdk', 'extensions/', 'src/plugins'],
    'agent': ['src/agents', 'src/parrot'],
    'command': ['src/commands', 'src/cli'],
    'media': ['src/media', 'src/image', 'src/pdf'],
    'channel': ['src/whatsapp', 'src/discord', 'src/telegram'],
}

def discover_functions(openclaw_structure):
    """从 OpenClaw 目录结构动态发现功能"""
    functions = []
    for path in openclaw_structure:
        category = infer_category(path)
        if category and category != 'channel':  # 过滤多渠道
            functions.append({
                'name': extract_feature_name(path),
                'category': category,
                'path': path,
            })
    return functions
```

### CHANGELOG 增量解析

```bash
# 获取上次对标后的 CHANGELOG 增量
LAST_DATE=$(cat docs/research/benchmark/state.jsonl | jq -r '.[-1].timestamp' | cut -dT -f1)

# 获取 CHANGELOG 并解析新增内容
gh api repos/openclaw/openclaw/contents/CHANGELOG.md | \
  jq -r '.content' | base64 -d | \
  awk "/$LAST_DATE/,0" | grep -E "^\-|\*"
```

---

## DivineSense 能力矩阵

### 动态扫描命令

> **脚本**: `scripts/benchmark/scan.sh`

```bash
# 方式一：使用脚本（推荐）
MATRIX=$(./scripts/benchmark/scan.sh matrix)
PARROTS=$(./scripts/benchmark/scan.sh parrots)
TOOLS=$(./scripts/benchmark/scan.sh tools)
PAGES=$(./scripts/benchmark/scan.sh pages)
TABLES=$(./scripts/benchmark/scan.sh tables)

# 列出名称
PARROT_NAMES=$(./scripts/benchmark/scan.sh parrot-names)
TOOL_NAMES=$(./scripts/benchmark/scan.sh tool-names)

# 显示摘要
./scripts/benchmark/scan.sh summary

# 方式二：手动扫描（兼容性）
PARROTS=$(find plugin/ai/agent -name "*_parrot.go" 2>/dev/null | wc -l)
TOOLS=$(find plugin/ai/agent/tools -name "*.go" 2>/dev/null | wc -l)
```

### 运行时能力矩阵生成

> 以下代码为**伪代码示例**，用于说明算法逻辑。

```python
def generate_capability_matrix():
    """生成 DivineSense 实时能力矩阵"""
    return {
        'agents': scan_parrots(),
        'tools': scan_tools(),
        'pages': scan_pages(),
        'tables': scan_tables(),
        'timestamp': datetime.utcnow().isoformat()
    }
```

---

## 对标方法论

### 技术契合度评分

| 评分 | 标准 | 示例 |
|:-----|:-----|:-----|
| **1.0** | Go 原生支持，可直接复用设计 | 会话修剪、压缩 |
| **0.7** | 需适配但有参考实现 | 插件 SDK（Go 版） |
| **0.4** | 需重构或使用不同技术 | Pi Agent（用 Parrot 替代） |
| **0.1** | TypeScript 特异性，难以迁移 | jiti 动态导入 |

### 用户价值评分

| 评分 | 标准 | 示例 |
|:-----|:-----|:-----|
| **1.0** | 高频使用，解决核心痛点 | 会话管理（每次 AI 交互） |
| **0.7** | 中频使用，显著提升体验 | 插件系统（扩展能力） |
| **0.4** | 低频使用，锦上添花 | 会话导出 |
| **0.1** | 边缘场景，用户需求弱 | 多渠道集成（不同定位） |

### 过滤规则

> 以下代码为**伪代码示例**，用于说明过滤逻辑。

```python
# 自动过滤条件
TS_SPECIFIC = ['jiti', 'tsx', 'oxlint', 'vitest', 'rollup', 'npm', 'pnpm']
CHANNELS = ['whatsapp', 'discord', 'telegram', 'signal', 'slack']

def should_filter(feature):
    if any(ts in feature.lower() for ts in TS_SPECIFIC):
        return True, 'TypeScript 特异性'
    if any(ch in feature.lower() for ch in CHANNELS):
        return True, '多渠道集成（不同定位）'
    return False, None
```

---

## 功能映射模板

### OpenClaw → DivineSense 映射（运行时生成）

| OpenClaw 功能 | DivineSense 对等 | 迁移复杂度 |
|:-------------|:----------------|:-----------|
| *动态发现* | *动态发现* | *动态评估* |

---

## 常用命令

```bash
# 状态管理
./scripts/benchmark/state.sh summary
./scripts/benchmark/state.sh query openclaw_sha

# 能力扫描
./scripts/benchmark/scan.sh summary
./scripts/benchmark/scan.sh has "pattern"

# 仓库信息（动态）
REPO=$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')

# 搜索现有 Issue
gh issue list --repo "$REPO" --search "<关键词>"

# 创建 Issue
gh issue create --repo "$REPO" --title "[feat] 功能" --body "..."
```

---

## 数据结构

### 状态记录（state.jsonl）

```json
{
  "timestamp": "2026-02-02T10:00:00Z",
  "openclaw_sha": "abc123",
  "divinesense_sha": "def456",
  "analyzed_features": ["会话修剪", "会话压缩"],
  "discovered_functions": [
    {"name": "会话修剪", "category": "session", "path": "src/sessions/pruning.ts"},
    {"name": "会话压缩", "category": "session", "path": "src/commands/compact.ts"}
  ],
  "created_issues": [30, 31]
}
```

### 功能差距记录

```typescript
interface FeatureGap {
  name: string;           // 功能名称
  category: string;       // 分类（动态推断）
  openclaw_path: string;  // OpenClaw 文件路径
  techFit: number;        // 技术契合度 0-1
  userValue: number;      // 用户价值 0-1
  priority: number;       // 优先级 = techFit*0.4 + userValue*0.6
  filterReason?: string;  // 过滤原因（如被过滤）
}
```

---

*文档版本：v1.2 | 最后更新：2026-02-02*
