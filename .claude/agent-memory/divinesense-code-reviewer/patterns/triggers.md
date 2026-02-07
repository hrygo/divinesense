# Code Review Trigger Patterns

> **Version**: 1.0.0
> **Purpose**: 记录触发 code-review 的话语模式和场景

---

## 用户话语模式

### 中文触发词

| 模式 | 示例 | 置信度 |
|:-----|:-----|:-------|
| `审查代码` | "帮我审查代码" / "请审查这段代码" | 95 |
| `代码审查` | "做个代码审查" / "需要进行代码审查" | 95 |
| `检查代码` | "检查这段代码有没有问题" / "帮我检查代码" | 90 |
| `看看代码` | "帮我看看这段代码" / "看看有什么问题" | 85 |
| `Review` | "Review the commit" / "Review my changes" | 95 |
| `Code review` | "Do a code review" / "Need code review" | 95 |
| `CR` | "CR this PR" / "CR please" | 90 |

### 场景触发词

| 场景 | 触发词 | 示例 |
|:-----|:-------|:-----|
| **Commit 后** | "review" / "check" | "commit and review" / "提交后审查" |
| **PR 创建** | "review PR" | "create PR and review" / "PR 创建后审查" |
| **Push 前** | "review before push" | "push 前帮我看看" / "推送前审查" |
| **大改动** | "深度审查" / "thorough review" | "大改动需要深度审查" |
| **质量检查** | "质量检查" / "质量分析" | "检查代码质量" / "代码质量分析" |

### 隐式触发

用户以下意图时也应考虑触发 code review：
- 提到 "bug" / "error" / "issue" 时询问代码
- 提到 "优化" / "refactor" 后询问代码质量
- 提到 "合并" / "merge" / "PR" 时
- 代码出现编译错误、测试失败时

---

## Git 操作关联

| Git 操作 | 触发时机 | 模式 |
|:---------|:---------|:-----|
| `git commit` | Hook: pre-commit（如果启用） | pre-commit |
| `git push` | Hook: pre-push（如果启用） | incremental |
| `gh pr create` | 手动触发或自动 | pr |
| `gh pr merge` | 手动触发 | pr |

---

## 技术决策记录

### 2026-02-07: 优先级策略

**问题**: 多个 code-review agents 存在，选择哪个？

**决策**:
1. DivineSense 项目使用 `divinesense-code-reviewer`（内置项目特定知识）
2. 通用 PR 使用 `pr-review-toolkit:code-reviewer`
3. Feature 开发使用 `feature-dev:code-reviewer`

**理由**:
- `divinesense-code-reviewer` 有项目特定的 pattern 和 decision 记忆
- 熟悉 DivineSense 的架构陷阱（Tailwind v4、Go embed、AI 路由等）
- 能提供更精准的建议和更少的误报

### 2026-02-07: 自动触发阈值

**问题**: 何时自动触发 code review？

**决策**:
- 单次变更 >500 行时自动触发
- 涉及以下模块时建议触发：
  - `ai/agent/` (AI 代理核心逻辑)
  - `server/router/` (API 路由)
  - `store/migration/` (数据库迁移)
- PR 打开/更新时触发

**理由**:
- 大改动风险更高，需要深度审查
- 核心模块变更影响面广
- PR 是代码合入前的最后防线

### 2026-02-07: 信心度阈值

**问题**: 设置多少信心度阈值？

**决策**: 默认 80，可调整

| 阈值 | 适用场景 | 说明 |
|:-----|:---------|:-----|
| 100 | 生产环境、安全相关 | 只报告确定性问题 |
| 90-95 | 关键路径、核心模块 | 过滤掉大部分 nitpick |
| 80-85 | 日常开发、新功能 | 平衡噪音和信号 |
| <80 | 学习阶段、实验性代码 | 报告所有问题 |

**理由**:
- 80 是"建议修复"的门槛，平衡发现问题和效率
- 太高会漏掉重要问题，太低会产生太多噪音
- 可根据项目阶段调整（早期开发用 70，稳定期用 90）
