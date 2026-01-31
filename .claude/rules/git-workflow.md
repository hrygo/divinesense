# Git 工作流 (严格执行)

**所有开发与进化模式均需遵守以下流程：**

---

## 工作流概览

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  创建 Issue  │ → │  创建分支    │ → │  开发提交    │ → │  发起 PR     │ → │  审核合并    │
│  (gh issue) │    │  (git checkout -b)│ │  (git commit)│  │  (gh pr create)│ │  (gh pr merge)│
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

---

## 1. 创建 Issue

每个任务/功能/修复都应先创建 Issue 进行追踪。

```bash
# 创建新 Issue
gh issue create --title "标题" --body "描述内容"

# 或交互式创建
gh issue create
```

**Issue 标题格式**：
- 功能：`[feat] 功能描述`
- 修复：`[fix] 问题描述`
- 重构：`[refactor] 重构描述`

**Issue 模板**：
```markdown
## 问题描述
<!-- 清晰描述要解决的问题或要实现的功能 -->

## 当前行为
<!-- 描述当前的行为（如果是 bug） -->

## 期望行为
<!-- 描述期望的行为 -->

## 解决方案
<!-- 提出解决方案或实现思路 -->

## 验收标准
- [ ] 标准 1
- [ ] 标准 2
```

记录 Issue 编号（如 #123），后续分支命名需要引用。

---

## 2. 创建分支

**禁止直接在 `main` 分支修改**。为每个 Issue 创建独立分支。

```bash
# 确保本地 main 是最新的
git checkout main
git pull origin main

# 创建功能分支 (引用 Issue 编号)
git checkout -b feat/123-add-ai-router
git checkout -b fix/456-session-cleanup
git checkout -b refactor/789-remove-dead-code
```

**分支命名规范**：

| 类型  | 格式                          | 示例                          |
| :---- | :---------------------------- | :---------------------------- |
| 功能  | `feat/<issue-id>-简短描述`     | `feat/123-add-ai-router`      |
| 修复  | `fix/<issue-id>-简短描述`      | `fix/456-session-cleanup`     |
| 重构  | `refactor/<issue-id>-简短描述` | `refactor/789-remove-dead-code`|
| 进化  | `evolution/<issue-id>-任务名`  | `evolution/200-self-improve`  |

**描述使用小写、连字符分隔**。

---

## 3. 开发与提交

### 3.1 提交前检查

```bash
make check-all      # 格式、vet、测试、i18n 检查
golangci-lint run   # 后端 lint (如果需要)
```

### 3.2 约定式提交

| 类型     | 范围    | 示例                                |
| :------- | :------ | :---------------------------------- |
| `feat`   | 功能区域| `feat(ai): 添加意图路由器`          |
| `fix`    | Bug 区域| `fix(db): 修复竞态条件`             |
| `refactor`| 代码区域| `refactor(frontend): 提取 hooks`    |
| `perf`   | N/A     | `perf(query): 优化向量搜索`         |
| `docs`   | N/A     | `docs(readme): 更新快速开始`        |
| `test`   | N/A     | `test(ai): 添加代理测试用例`        |
| `chore`  | N/A     | `chore(deps): 升级依赖版本`          |

**格式**：`<type>(<scope>): <description>`

**示例**：
```bash
git commit -m "$(cat <<'EOF'
feat(ai): add intent router for multi-agent system

- Add ChatRouter for routing user queries to appropriate agents
- Implement rule-based fast path (0ms)
- Add LLM fallback for ambiguous queries (~400ms)
- Add unit tests for router logic

Refs #123

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

**必须包含**：
- `Refs #<issue-id>` - 关联 Issue
- `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>`

### 3.3 推送分支

```bash
git push -u origin feat/123-add-ai-router
```

---

## 4. 发起 Pull Request

### 4.1 创建 PR

```bash
# 基于分支创建 PR
gh pr create --title "feat(ai): add intent router" --body "详细描述..."

# 或使用交互式创建
gh pr create --web
```

### 4.2 PR 模板

```markdown
## 概述
<!-- 简短描述此 PR 的目的 -->

## 变更内容
- [ ] 变更点 1
- [ ] 变更点 2
- [ ] 变更点 3

## 关联 Issue
Resolves #123

## 测试计划
- [ ] 本地测试通过
- [ ] 单元测试新增/更新
- [ ] `make check-all` 通过
- [ ] 手动测试场景描述

## 截图/演示
<!-- 如果是 UI 变更，添加截图或 GIF -->

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 自我审查代码
- [ ] 注释说明了复杂逻辑
- [ ] 文档已更新（如需要）
- [ ] 无合并冲突

---

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

### 4.3 PR 标题格式

与 commit message 保持一致：
- `feat(ai): add intent router`
- `fix(session): resolve zombie cleanup job`
- `refactor(ai): remove fixed conversation mechanism`

---

## 5. 审核与合并

### 5.1 PR 审核

**审核检查项**：
- [ ] 代码逻辑正确
- [ ] 没有引入新的 bug
- [ ] 测试覆盖充分
- [ ] 文档已同步更新
- [ ] 符合代码风格

**要求**：至少一名维护者审核通过后方可合并。

### 5.2 合并 PR

```bash
# 查看 PR 状态
gh pr status

# 合并 PR (squash merge 推荐)
gh pr merge <pr-number> --squash --delete-branch

# 或使用 Web UI 点击合并按钮
```

**合并方式**：
- **Squash Merge**（推荐）：将多个 commit 压缩为一个，保持历史清洁
- **Merge Commit**：保留分支历史
- **Rebase Merge**：线性历史，不推荐用于多人协作

### 5.3 合并后清理

```bash
# 删除已合并的本地分支
git branch -d feat/123-add-ai-router

# 同步远程分支列表
git remote prune origin
```

---

## 6. 分支保护规则

**`main` 分支受保护**：

| 规则                    | 状态 | 说明 |
| :---------------------- | :--- | :--- |
| 禁止直接推送            | ✅   | 所有用户必须通过 PR |
| 需要 PR 审核才能合并     | ✅   | 普通用户需要他人批准 |
| 需要 1 个审核批准        | ✅   | 普通用户 |
| 线性历史要求            | ✅   | 使用 squash merge |
| 禁止强制推送            | ✅   | 保护历史完整性 |
| 禁止删除分支            | ✅   | 防止误删 |

**权限差异**：

| 用户类型 | 审核要求 | 合并方式 |
|:---------|:--------|:---------|
| **管理员** | 可自己批准 | 直接合并或自己审批后合并 |
| **普通用户** | 需他人批准 | 等待维护者审核通过后合并 |

**管理员自审流程**：
```bash
# 1. 创建 PR
gh pr create

# 2. 管理员可以直接批准并合并
gh pr merge --squash --delete-branch --admin
```

---

## 7. 常用命令速查

```bash
# Issue 管理
gh issue list              # 列出所有 Issue
gh issue view 123          # 查看 Issue 详情
gh issue close 123         # 关闭 Issue

# 分支管理
git branch -a              # 列出所有分支
git checkout -b feat/new   # 创建并切换到新分支
git push -u origin feat/new # 推送并跟踪分支

# PR 管理
gh pr list                 # 列出所有 PR
gh pr view 456             # 查看 PR 详情
gh pr diff 456             # 查看 PR diff
gh pr checks 456           # 查看 PR CI 状态
gh pr merge 456            # 合并 PR
gh pr close 456            # 关闭 PR

# 快速流程 (结合)
gh issue create --title "Fix bug" --body "Description"
# 记录 Issue ID (如 #123)
git checkout -b fix/123-bug
# ... 开发 ...
git commit -m "fix: resolve bug (Refs #123)"
git push -u origin fix/123-bug
gh pr create --title "fix: resolve bug" --body "Resolves #123"
```

---

## 8. 故障排查

### Q: 推送被拒绝，提示 "Changes must be made through a pull request"

**A**: 你正在尝试直接推送到 `main`。创建功能分支并提交 PR。

### Q: PR 检查失败

**A**: 运行 `make check-all` 修复问题，或推送新 commit 触发重新检查。

### Q: 合并冲突

**A**:
```bash
git checkout main
git pull origin main
git checkout feat/123-feature
git rebase main
# 解决冲突
git rebase --continue
git push --force-with-lease
```

### Q: 错误的 commit 信息

**A**:
```bash
# 修改最后一次 commit
git commit --amend

# 修改历史多个 commit (危险！)
git rebase -i HEAD~n
```

---

## 9. 最佳实践

1. **小步快跑**：每个 PR 只做一件事，保持改动范围小
2. **频繁提交**：本地频繁 commit，PR 中可 squash
3. **及时同步**：定期 rebase main，避免冲突累积
4. **清晰描述**：PR 描述要包含变更原因和测试方法
5. **自我审查**：发起 PR 前先自我 review 一遍

---

*更新于 2025-01-31*
