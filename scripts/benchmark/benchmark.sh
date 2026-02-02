#!/usr/bin/env bash
# Competitive Benchmark - Main Script
#
# 竞品对标主入口，支持初始化和运行全量分析
#
# Usage:
#   ./scripts/benchmark/benchmark.sh init    # 初始化对标状态
#   ./scripts/benchmark/benchmark.sh run     # 运行全量对标分析
#   ./scripts/benchmark/benchmark.sh status  # 查看对标状态

set -euo pipefail

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# 项目根目录
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# 验证项目根目录
validate_project_root() {
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]] || [[ ! -d "$PROJECT_ROOT/plugin/ai" ]]; then
        echo "错误: 必须在 DivineSense 项目根目录下运行" >&2
        exit 1
    fi
}

validate_project_root

# 切换到项目根目录
cd "$PROJECT_ROOT" || { echo "错误: 无法切换到项目目录" >&2; exit 1; }

# 日志函数
log_info() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][INFO] $*"; }
log_error() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][ERROR] $*"; }
log_warn() { echo "[$(date -u +"%Y-%m-%d %H:%M:%S")][WARN] $*"; }
log_step() { echo "▸ $*"; }

# 检查依赖
check_dependencies() {
    local missing=()

    for cmd in jq gh; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "缺少必需依赖: ${missing[*]}"
        echo "请安装:" >&2
        echo "  brew install jq gh" >&2
        echo "  或" >&2
        echo "  apt install jq gh" >&2
        exit 1
    fi
}

# 检查 gh 认证状态
check_gh_auth() {
    if ! gh auth status >/dev/null 2>&1; then
        log_error "GitHub CLI 未登录"
        echo "请运行: gh auth login" >&2
        exit 1
    fi
}

# 初始化对标状态
init_benchmark() {
    log_info "初始化竞品对标状态..."

    # 调用 state.sh init
    if "$SCRIPT_DIR/state.sh" init; then
        log_info "初始化成功"
        echo ""
        echo "下一步:"
        echo "  ./scripts/benchmark/benchmark.sh run    # 运行全量对标分析"
        echo "  ./scripts/benchmark/benchmark.sh status  # 查看对标状态"
    else
        log_error "初始化失败"
        exit 1
    fi
}

# 获取 OpenClaw 最新 SHA
get_openclaw_sha() {
    local sha
    sha=$(gh api repos/openclaw/openclaw/commits 2>/dev/null | jq -r '.sha // empty')
    if [[ -z "$sha" ]]; then
        log_error "无法获取 OpenClaw SHA"
        return 1
    fi
    echo "$sha"
}

# 扫描 DivineSense 能力
scan_divinesense_capabilities() {
    log_step "扫描 DivineSense 能力矩阵..."
    "$SCRIPT_DIR/scan.sh" summary
}

# 获取对标目标仓库（目前硬编码为 OpenClaw）
get_target_repo() {
    echo "openclaw/openclaw"
}

# 执行全量对标分析
run_benchmark() {
    log_info "开始竞品对标分析..."

    # 检查依赖和认证
    check_dependencies
    check_gh_auth

    # 获取当前状态
    local last_oc_sha last_ds_sha
    if last_oc_sha=$("$SCRIPT_DIR/state.sh" query openclaw_sha 2>/dev/null); then
        : # 有值
    else
        last_oc_sha=""
    fi

    if last_ds_sha=$("$SCRIPT_DIR/state.sh" query divinesense_sha 2>/dev/null); then
        : # 有值
    else
        last_ds_sha=""
    fi

    # 显示当前状态
    echo ""
    "$SCRIPT_DIR/state.sh" summary
    echo ""

    # 获取当前 SHA
    local current_oc_sha current_ds_sha
    current_oc_sha=$(get_openclaw_sha)
    current_ds_sha=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

    log_info "OpenClaw SHA: ${current_oc_sha:0:8}..."
    log_info "DivineSense SHA: ${current_ds_sha:0:8}..."

    # 检查是否有变化
    if [[ "$last_oc_sha" == "$current_oc_sha" ]]; then
        log_warn "OpenClaw 自上次对标后无变化"
    fi

    if [[ "$last_ds_sha" == "$current_ds_sha" ]]; then
        log_info "DivineSense 代码无变化"
    else
        log_info "DivineSense 代码有更新，重新扫描能力..."
        scan_divinesense_capabilities
    fi

    # 这里是标对的占位符逻辑
    # 实际对标需要通过 competitive-benchmark Skill 执行
    log_info "对标分析需要通过 competitive-benchmark Skill 执行"
    echo ""
    echo "使用 Skill 执行对标:"
    echo "  /competitive-benchmark"
    echo ""
    echo "或手动分析:"
    echo "  1. 对比 OpenClaw 功能: https://github.com/openclaw/openclaw"
    echo "  2. 检查 DivineSense 实现情况"
    echo "  3. 生成 Issue: gh issue create --title '[feat] 功能' --body '...'"

    # 询问是否更新状态
    echo ""
    read -p "是否更新对标状态? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        "$SCRIPT_DIR/state.sh" append "$timestamp" "$current_oc_sha" "$current_ds_sha" '[]' '[]'
        log_info "状态已更新"
    fi
}

# 查看对标状态
show_status() {
    log_info "查看对标状态..."
    echo ""
    "$SCRIPT_DIR/state.sh" summary
    echo ""

    # 显示 DivineSense 能力
    echo "DivineSense 能力:"
    "$SCRIPT_DIR/scan.sh" summary
}

# 主命令路由
case "${1:-help}" in
    init)
        check_dependencies
        init_benchmark
        ;;
    run|analyze)
        run_benchmark
        ;;
    status|info)
        show_status
        ;;
    help|--help|-h)
        cat <<'EOF'
Competitive Benchmark - 竞品对标工具

用法:
  ./scripts/benchmark/benchmark.sh <command>

命令:
  init         初始化对标状态（首次运行）
  run          运行全量对标分析
  status       查看对标状态和 DivineSense 能力
  help         显示此帮助信息

首次使用:
  1. ./scripts/benchmark/benchmark.sh init
  2. /competitive-benchmark  # 通过 Skill 执行对标
  3. ./scripts/benchmark/benchmark.sh status

依赖:
  - jq (JSON 处理)
  - gh (GitHub CLI)

对标目标:
  - OpenClaw (https://github.com/openclaw/openclaw)
EOF
        ;;
    *)
        echo "错误: 未知命令 '$1'" >&2
        echo "" >&2
        echo "使用 'help' 查看帮助信息" >&2
        exit 1
        ;;
esac
