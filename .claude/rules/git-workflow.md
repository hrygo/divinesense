# Git 工作流

## 快速流程

```
Issue → 分支 → 开发 → PR → 合并
```

## 关键命令

```bash
# Issue
gh issue create --title "[feat] 描述" --body "详细内容"

# 分支（引用 Issue #123）
git checkout -b feat/123-description

# 定期同步（重要！）
git fetch origin && git rebase origin/main

# 提交（Conventional Commits）
git commit -m "feat(scope): description (Refs #123)"

# PR
gh pr create --title "feat(scope): description" --body "Resolves #123"
```

## Commit 格式

| 类型       | 说明     | 示例                         |
| :--------- | :------- | :--------------------------- |
| `feat`     | 新功能   | `feat(ai): 添加意图路由`     |
| `fix`      | Bug 修复 | `fix(db): 修复竞态条件`      |
| `refactor` | 重构     | `refactor(ui): 提取 hooks`   |
| `docs`     | 文档     | `docs(readme): 更新安装说明` |
| `test`     | 测试     | `test(ai): 添加路由测试`     |
| `chore`    | 杂项     | `chore(deps): 升级依赖`      |

## 分支命名

- `feat/<issue-id>-描述`
- `fix/<issue-id>-描述`
- `refactor/<issue-id>-描述`

## PR 创建规范（重要）

### Issue 链接（必须）

PR 描述**必须**包含 Issue 链接，否则 CI 检查会失败：

```
Resolves #123    # 完成时会自动关闭 Issue
Refs #123        # 仅关联，不自动关闭
```

**正确示例**：
```markdown
## Summary
实现 xxx 功能

Resolves #123

## Changes
...
```

**错误示例**（CI 会失败）：
```markdown
## Summary
实现 xxx 功能

关联 Issue: #123    ❌ 不符合格式
Issue #123          ❌ 不符合格式
(Refs #123)         ❌ 只在 commit 中有效，PR 描述需要单独声明
```

### PR 描述模板

```markdown
## Summary
简要描述变更内容

Resolves #XXX    # 或 Refs #XXX（必须单独一行）

## Changes
- 变更 1
- 变更 2

## Test plan
- [ ] 测试项 1
- [ ] 测试项 2
```

### 创建 PR 前检查清单

- [ ] 分支命名包含 Issue ID（如 `feat/169-xxx`）
- [ ] PR 描述包含 `Resolves #XXX` 或 `Refs #XXX`
- [ ] `make check-all` 通过
- [ ] Commit message 符合 Conventional Commits

## 发布流程

1. 合并 PR → 2. 更新相关文档 → 3. **创建 git tag** → 4. 推送 tag → 5. GitHub Release

### 需要更新的文档

- `CHANGELOG.md` - 版本变更记录
- `CLAUDE.md` - 如有架构变更
- `docs/` - 相关技术文档
- `README.md` - 如有必要

```bash
git tag -a v0.XX.0 -m "Release v0.XX.0"
git push origin v0.XX.0
gh release create v0.XX.0 --notes "Release notes"
```