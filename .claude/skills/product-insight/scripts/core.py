#!/usr/bin/env python3
"""
Product Insight - Main Entry Point

产品洞察主入口，支持初始化和运行全量分析。
替代 core.sh

Usage:
    python core.py init    # 初始化洞察状态
    python core.py run     # 运行全量洞察分析
    python core.py status  # 查看洞察状态
"""

import logging
import os
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

# 导入同级模块（简化结构后，所有模块在 scripts/ 目录）
from state import StateManager, InsightState
from scan import CapabilityScanner

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s][%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def check_dependencies() -> list[str]:
    """
    检查依赖

    Returns:
        缺失的依赖列表
    """
    missing = []

    # 检查 gh
    try:
        subprocess.run(
            ["gh", "--version"],
            capture_output=True,
            check=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        missing.append("gh")

    # 检查 git
    try:
        subprocess.run(
            ["git", "--version"],
            capture_output=True,
            check=True,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        missing.append("git")

    return missing


def check_gh_auth() -> bool:
    """
    检查 gh 认证状态

    Returns:
        是否已认证
    """
    try:
        result = subprocess.run(
            ["gh", "auth", "status"],
            capture_output=True,
            check=True,
        )
        return result.returncode == 0
    except subprocess.CalledProcessError:
        return False


def get_target_repo() -> str:
    """获取对标目标仓库（支持 BENCHMARK_TARGET 环境变量）"""
    return os.environ.get("BENCHMARK_TARGET", "openclaw/openclaw")


def get_target_sha(repo: str) -> str:
    """
    获取目标仓库最新 SHA

    Args:
        repo: 仓库路径 (如 owner/repo)

    Returns:
        Git SHA

    Raises:
        subprocess.CalledProcessError: 获取失败
    """
    result = subprocess.run(
        ["gh", "api", f"repos/{repo}/commits"],
        capture_output=True,
        text=True,
        check=True,
        timeout=30,  # 30秒超时
    )
    import json

    data = json.loads(result.stdout)
    sha = data.get("sha", "")
    if not sha:
        raise ValueError(f"无法获取 {repo} SHA")
    return sha


def init_insight(state_mgr: StateManager) -> None:
    """初始化洞察状态"""
    logger.info("初始化产品洞察状态...")

    try:
        state_mgr.init()
        logger.info("初始化成功")
        print()
        print("下一步:")
        print("  python -m product_insight.core run    # 运行全量洞察分析")
        print("  python -m product_insight.core status  # 查看洞察状态")
    except FileExistsError as e:
        logger.error(f"初始化失败: {e}")
        sys.exit(1)


def run_insight(state_mgr: StateManager, scanner: CapabilityScanner) -> None:
    """执行全量洞察分析"""
    logger.info("开始产品洞察分析...")

    # 检查依赖
    missing = check_dependencies()
    if missing:
        logger.error(f"缺少必需依赖: {', '.join(missing)}")
        print("请安装:", file=sys.stderr)
        print("  brew install gh git" if sys.platform != "linux" else "  apt install gh git", file=sys.stderr)
        sys.exit(1)

    # 检查 gh 认证
    if not check_gh_auth():
        logger.error("GitHub CLI 未登录")
        print("请运行: gh auth login", file=sys.stderr)
        sys.exit(1)

    # 获取当前状态
    last_oc_sha = state_mgr.query("openclaw_sha") or ""
    last_ds_sha = state_mgr.query("divinesense_sha") or ""

    # 显示当前状态
    print()
    print(state_mgr.summary())
    print()

    # 获取当前 SHA
    target_repo = get_target_repo()
    try:
        current_oc_sha = get_target_sha(target_repo)
    except (subprocess.CalledProcessError, ValueError) as e:
        logger.error(str(e))
        sys.exit(1)

    current_ds_sha = scanner._get_git_sha()

    logger.info(f"目标仓库: {target_repo}")
    logger.info(f"目标 SHA: {current_oc_sha[:8]}...")
    logger.info(f"DivineSense SHA: {current_ds_sha[:8]}...")

    # 检查是否有变化
    if last_oc_sha == current_oc_sha:
        logger.warning("竞品自上次对标后无变化")

    if last_ds_sha == current_ds_sha:
        logger.info("DivineSense 代码无变化")
    else:
        logger.info("DivineSense 代码有更新，重新扫描能力...")
        print(scanner.summary())

    # 这里是标对的占位符逻辑
    # 实际对标需要通过 product-insight Skill 执行
    logger.info("洞察分析需要通过 product-insight Skill 执行")
    print()
    print("使用 Skill 执行对标:")
    print("  /product-insight")
    print()
    print(f"或自定义目标产品后执行:")
    print(f'  BENCHMARK_TARGET="{target_repo}" /product-insight')
    print()
    print("或手动分析:")
    print(f"  1. 分析 {target_repo} 功能: https://github.com/{target_repo}")
    print("  2. 检查 DivineSense 实现情况")
    print('  3. 生成 Issue: gh issue create --title \'[feat] 功能\' --body \'...\'')
    print()

    # 询问是否更新状态
    # 支持 BENCHMARK_AUTO_CONFIRM 环境变量（非终端环境或 Skill 调用）
    should_update = False
    auto_confirm = os.environ.get("BENCHMARK_AUTO_CONFIRM", "").lower() == "true"

    if auto_confirm:
        should_update = True
    elif sys.stdin.isatty() and sys.stdout.isatty():
        # 终端环境，交互式询问
        try:
            response = input("是否更新洞察状态? (y/N): ").strip().lower()
            should_update = response == "y"
        except (EOFError, KeyboardInterrupt):
            print()
    else:
        # 非终端环境，默认不更新（静默跳过）
        logger.info("非终端环境，跳过状态更新（设置 BENCHMARK_AUTO_CONFIRM=true 自动更新）")

    if should_update:
        timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        new_state = InsightState(
            timestamp=timestamp,
            openclaw_sha=current_oc_sha,
            divinesense_sha=current_ds_sha,
            analyzed_features=[],
            discovered_functions=[],
            created_issues=[],
        )
        state_mgr.append(new_state)
        logger.info("状态已更新")


def show_status(state_mgr: StateManager, scanner: CapabilityScanner) -> None:
    """查看洞察状态"""
    logger.info("查看洞察状态...")
    print()
    print(state_mgr.summary())
    print()
    print("DivineSense 能力:")
    print(scanner.summary())


def print_help() -> None:
    """显示帮助信息"""
    print("""Product Insight - 产品洞察引擎

用法:
  python core.py <command>

命令:
  init         初始化洞察状态（首次运行）
  run          运行全量洞察分析
  status       查看洞察状态和 DivineSense 能力
  help         显示此帮助信息

首次使用:
  1. python core.py init
  2. /product-insight  # 通过 Skill 执行对标
  3. python core.py status

环境变量:
  - BENCHMARK_TARGET  # 自定义目标产品仓库（默认 openclaw/openclaw）
  - BENCHMARK_AUTO_CONFIRM  # 跳过交互确认（默认 false）

依赖:
  - gh (GitHub CLI)
  - git (版本控制)

对标目标:
  - 默认: OpenClaw (https://github.com/openclaw/openclaw)
  - 自定义: BENCHMARK_TARGET="owner/repo" python core.py run""")


def main() -> None:
    """CLI 入口（替代 core.sh 的直接执行）"""
    # 初始化管理器
    state_mgr = StateManager()
    scanner = CapabilityScanner()

    if len(sys.argv) < 2:
        print_help()
        sys.exit(1)

    cmd = sys.argv[1]

    match cmd:
        case "init":
            missing = check_dependencies()
            if missing:
                logger.warning(f"缺少依赖: {', '.join(missing)} (init 命令可跳过)")
            init_insight(state_mgr)

        case "run" | "analyze":
            run_insight(state_mgr, scanner)

        case "status" | "info":
            show_status(state_mgr, scanner)

        case "help" | "--help" | "-h":
            print_help()

        case _:
            print(f"错误: 未知命令 '{cmd}'", file=sys.stderr)
            print()
            print("使用 'help' 查看帮助信息", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
