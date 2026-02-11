# Git 工作流

> **详细版本**: 本文档精简版，完整内容见 GitHub

---

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

| 类型 | 说明 | 示例 |
|:-----|:-----|:-----|
| `feat` | 新功能 | `feat(ai): 添加意图路由` |
| `fix` | Bug 修复 | `fix(db): 修复竞态条件` |
| `refactor` | 重构 | `refactor(ui): 提取 hooks` |
| `docs` | 文档 | `docs(readme): 更新安装说明` |
| `test` | 测试 | `test(ai): 添加路由测试` |
| `chore` | 杂项 | `chore(deps): 升级依赖` |

## 分支命名

- `feat/<issue-id>-描述`
- `fix/<issue-id>-描述`
- `refactor/<issue-id>-描述`

## 发布流程

1. 合并 PR → 2. 更新 CHANGELOG.md → 3. **创建 git tag** → 4. 推送 tag → 5. GitHub Release

```bash
git tag -a v0.XX.0 -m "Release v0.XX.0"
git push origin v0.XX.0
gh release create v0.XX.0 --notes "Release notes"
```

**完整文档**: <https://github.com/hrygo/divinesense/blob/main/.claude/rules/git-workflow.md>
