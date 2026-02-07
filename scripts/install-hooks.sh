#!/bin/bash
# install-hooks.sh - Install smart git hooks from scripts directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOKS_DIR="$(git rev-parse --git-common-dir)/hooks"

echo "📦 Installing smart git hooks..."
echo ""

# Copy pre-commit hook (smart checks based on file types)
if [ -f "$SCRIPT_DIR/pre-commit" ]; then
    cp "$SCRIPT_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
    chmod +x "$HOOKS_DIR/pre-commit"
    echo "  ✓ pre-commit  → 智能快速检查 (~2-10s)"
    echo "                  • 仅后端: fmt + vet"
    echo "                  • 仅前端: lint:fix"
    echo "                  • 仅文档: 跳过检查"
    echo "                  • 混合:   按需检查"
else
    echo "  ✗ pre-commit hook not found in $SCRIPT_DIR"
    exit 1
fi

# Copy pre-push hook (smart CI checks based on file types)
if [ -f "$SCRIPT_DIR/pre-push" ]; then
    cp "$SCRIPT_DIR/pre-push" "$HOOKS_DIR/pre-push"
    chmod +x "$HOOKS_DIR/pre-push"
    echo "  ✓ pre-push   → 智能完整 CI (~10-60s)"
    echo "                  • 仅后端: tidy + golangci-lint + test"
    echo "                  • 仅前端: lint + build"
    echo "                  • 仅文档: 跳过检查"
    echo "                  • 混合:   按需检查"
else
    echo "  ✗ pre-push hook not found in $SCRIPT_DIR"
    exit 1
fi

echo ""
echo "✅ Smart git hooks installed!"
echo ""
echo "检查策略:"
echo "  类型        pre-commit        pre-push"
echo "  ──────────  ──────────────    ─────────────"
echo "  仅后端     fmt + vet         tidy + lint + test"
echo "  仅前端     lint:fix           lint + build"
echo "  仅文档     跳过               跳过"
echo "  混合       按需检查           按需检查"
echo ""
echo "跳过检查:"
echo "  • commit:  git commit --no-verify -m 'msg'"
echo "  • push:   git push --no-verify"
echo ""
