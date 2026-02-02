#!/usr/bin/env python3
"""
Product Insight - Capability Scanning

扫描 DivineSense 项目的能力矩阵。
替代 scan.sh

Usage:
    from scan import CapabilityScanner

    scanner = CapabilityScanner()
    matrix = scanner.generate_matrix()
    print(scanner.summary())
"""

import dataclasses
import json
import logging
import os
import re
import subprocess
import sys
from datetime import datetime, timezone
from functools import lru_cache
from pathlib import Path
from typing import Optional

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s][%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


@dataclasses.dataclass(frozen=True)
class CapabilityMatrix:
    """能力矩阵数据结构"""

    parrots: int
    tools: int
    pages: int
    tables: int
    parrot_names: list[str]
    tool_names: list[str]
    page_names: list[str]
    table_names: list[str]
    divinesense_sha: str
    timestamp: str

    def to_json(self) -> dict:
        """转换为 JSON 可序列化的字典"""
        return dataclasses.asdict(self)


class CapabilityScanner:
    """DivineSense 能力矩阵扫描器"""

    # 路径常量
    AGENT_DIR = "plugin/ai/agent"
    TOOLS_DIR = "plugin/ai/agent/tools"
    PAGES_DIR = "web/src/pages"
    MIGRATION_DIR = "store/migration/postgres"

    def __init__(self, project_root: Optional[Path] = None):
        """
        初始化扫描器

        Args:
            project_root: 项目根目录，自动检测
        """
        self.project_root = self._find_project_root(project_root)
        self._cache: dict[str, Any] = {}

    def _find_project_root(self, project_root: Optional[Path]) -> Path:
        """查找并验证项目根目录"""
        if project_root is None:
            # 从脚本位置向上查找
            # __file__ = scripts/scan.py
            # parent = scripts/
            # 从 scripts/ 向上 4 层: scripts/ → product-insight/ → skills/ → .claude/ → project_root/
            scripts_dir = Path(__file__).parent
            project_root = scripts_dir.parent.parent.parent.parent

        # 验证项目根目录
        if not (project_root / "go.mod").exists():
            raise FileNotFoundError(
                f"错误: 必须在 DivineSense 项目根目录下运行 "
                f"({project_root / 'go.mod'} 不存在)"
            )

        if not (project_root / "plugin" / "ai").exists():
            raise FileNotFoundError(
                f"错误: plugin/ai 目录不存在，请确认在 DivineSense 项目中"
            )

        return project_root.resolve()

    def _get_git_sha(self) -> str:
        """获取当前 Git SHA"""
        try:
            result = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                cwd=self.project_root,
                capture_output=True,
                text=True,
                check=True,
                timeout=10,  # 10秒超时
            )
            return result.stdout.strip()
        except (subprocess.CalledProcessError, FileNotFoundError):
            return "unknown"

    def _scan_files(self, directory: str, pattern: str, exclude: Optional[str] = None) -> list[Path]:
        """
        扫描目录下匹配模式的文件

        Args:
            directory: 相对于项目根的目录路径
            pattern: 文件匹配模式 (如 *.go)
            exclude: 排除模式 (如 *_test.go)

        Returns:
            匹配的文件路径列表
        """
        base_path = self.project_root / directory
        if not base_path.exists():
            return []

        files = list(base_path.glob(pattern))

        if exclude:
            files = [f for f in files if not f.match(exclude)]

        return sorted(files)

    def _extract_name(self, path: Path, patterns: list[str]) -> str:
        """
        从文件路径提取名称

        Args:
            path: 文件路径
            patterns: 要移除的后缀模式列表

        Returns:
            提取的名称
        """
        name = path.name
        for pattern in patterns:
            name = re.sub(pattern, "", name)
        return name

    @property
    def parrots(self) -> int:
        """扫描 AI 代理数量"""
        cache_key = "parrots"
        if cache_key in self._cache:
            return self._cache[cache_key]

        agent_dir = self.project_root / self.AGENT_DIR
        if not agent_dir.exists():
            self._cache[cache_key] = 0
            return 0

        # 扫描标准 Parrot 和 V2 变体
        parrot_files = list(agent_dir.glob("*_parrot.go")) + list(agent_dir.glob("*_parrot_v2.go"))
        count = len(parrot_files)

        self._cache[cache_key] = count
        return count

    @property
    def parrot_names(self) -> list[str]:
        """扫描 AI 代理名称列表"""
        cache_key = "parrot_names"
        if cache_key in self._cache:
            return self._cache[cache_key]

        agent_dir = self.project_root / self.AGENT_DIR
        if not agent_dir.exists():
            self._cache[cache_key] = []
            return []

        # 获取所有 parrot 文件
        parrot_files = list(agent_dir.glob("*_parrot.go")) + list(agent_dir.glob("*_parrot_v2.go"))

        # 提取名称（移除 _parrot.go 或 _parrot_v2.go 后缀）
        names = set()
        for f in parrot_files:
            name = f.name
            name = re.sub(r"_parrot_v2\.go$", "", name)
            name = re.sub(r"_parrot\.go$", "", name)
            names.add(name)

        self._cache[cache_key] = sorted(names)
        return self._cache[cache_key]

    @property
    def tools(self) -> int:
        """扫描工具数量"""
        cache_key = "tools"
        if cache_key in self._cache:
            return self._cache[cache_key]

        # 排除测试文件
        tool_files = self._scan_files(self.TOOLS_DIR, "*.go", "*_test.go")
        self._cache[cache_key] = len(tool_files)
        return len(tool_files)

    @property
    def tool_names(self) -> list[str]:
        """扫描工具名称列表"""
        cache_key = "tool_names"
        if cache_key in self._cache:
            return self._cache[cache_key]

        tool_files = self._scan_files(self.TOOLS_DIR, "*.go", "*_test.go")
        names = sorted(set(f.stem for f in tool_files))
        self._cache[cache_key] = names
        return names

    @property
    def pages(self) -> int:
        """扫描前端页面数量"""
        cache_key = "pages"
        if cache_key in self._cache:
            return self._cache[cache_key]

        page_files = self._scan_files(self.PAGES_DIR, "*.tsx")
        # 过滤掉 index 和子目录入口
        count = len(page_files)
        self._cache[cache_key] = count
        return count

    @property
    def page_names(self) -> list[str]:
        """扫描前端页面名称列表"""
        cache_key = "page_names"
        if cache_key in self._cache:
            return self._cache[cache_key]

        page_files = self._scan_files(self.PAGES_DIR, "*.tsx")
        names = sorted(set(f.stem for f in page_files))
        self._cache[cache_key] = names
        return names

    @property
    def tables(self) -> int:
        """扫描数据库表数量"""
        cache_key = "tables"
        if cache_key in self._cache:
            return self._cache[cache_key]

        table_names = self.table_names
        self._cache[cache_key] = len(table_names)
        return len(table_names)

    @property
    def table_names(self) -> list[str]:
        """扫描数据库表名称列表"""
        cache_key = "table_names"
        if cache_key in self._cache:
            return self._cache[cache_key]

        migration_dir = self.project_root / self.MIGRATION_DIR
        if not migration_dir.exists():
            self._cache[cache_key] = []
            return []

        # 扫描 CREATE TABLE 语句
        # 兼容 IF NOT EXISTS 和普通格式
        pattern = re.compile(
            r"CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?[`'\"]?([a-zA-Z_][a-zA-Z0-9_]*)",
            re.IGNORECASE,
        )

        tables = set()
        for sql_file in migration_dir.rglob("*.sql"):
            try:
                content = sql_file.read_text(encoding="utf-8")
                for match in pattern.finditer(content):
                    table_name = match.group(1)
                    # 过滤掉 SQL 关键字
                    if table_name.upper() not in {
                        "IF",
                        "SELECT",
                        "INSERT",
                        "UPDATE",
                        "DELETE",
                        "CREATE",
                        "ALTER",
                        "DROP",
                    }:
                        tables.add(table_name)
            except Exception:
                pass

        self._cache[cache_key] = sorted(tables)
        return self._cache[cache_key]

    def has_feature(self, pattern: str) -> bool:
        """
        检查是否已实现某功能

        Args:
            pattern: 搜索模式（固定字符串，非正则）

        Returns:
            是否找到匹配

        Raises:
            ValueError: 如果模式包含非法字符
        """
        if not pattern:
            raise ValueError("has_feature 需要提供搜索模式")

        # 验证输入只包含安全字符
        if not re.match(r"^[a-zA-Z0-9_./:-]+$", pattern):
            raise ValueError(f"搜索模式包含非法字符: {pattern}")

        # 搜索关键目录
        search_dirs = [
            self.project_root / self.AGENT_DIR,
            self.project_root / "web/src",
        ]

        pattern_bytes = pattern.encode("utf-8")

        for search_dir in search_dirs:
            if not search_dir.exists():
                continue

            for root, _, files in os.walk(search_dir):
                for filename in files:
                    file_path = Path(root) / filename

                    # 跳过二进制文件和大型文件
                    if file_path.suffix in {".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2"}:
                        continue
                    if file_path.stat().st_size > 1024 * 1024:  # > 1MB
                        continue

                    try:
                        content = file_path.read_bytes()
                        if pattern_bytes in content:
                            return True
                    except Exception:
                        pass

        return False

    def generate_matrix(self) -> dict:
        """
        生成完整能力矩阵 (JSON)

        Returns:
            JSON 格式的能力矩阵
        """
        timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        ds_sha = self._get_git_sha()

        return {
            "parrots": self.parrots,
            "tools": self.tools,
            "pages": self.pages,
            "tables": self.tables,
            "parrot_names": self.parrot_names,
            "tool_names": self.tool_names,
            "page_names": self.page_names,
            "table_names": self.table_names,
            "divinesense_sha": ds_sha,
            "timestamp": timestamp,
        }

    def summary(self) -> str:
        """显示人类可读的能力摘要"""
        lines = [
            "DivineSense 能力矩阵:",
            f"  AI 代理: {self.parrots} 个",
            f"  工具: {self.tools} 个",
            f"  前端页面: {self.pages} 个",
            f"  数据库表: {self.tables} 个",
            "",
            "AI 代理列表:",
        ]
        lines.extend(f"  - {name}" for name in self.parrot_names)
        lines.append("")
        lines.append("工具列表:")
        lines.extend(f"  - {name}" for name in self.tool_names)

        return "\n".join(lines)


def _can_use_emoji() -> bool:
    """检查终端是否支持 emoji"""
    return (
        sys.stdout.isatty()
        and os.environ.get("TERM", "") != "dumb"
        and os.environ.get("TERM", "") != ""
    )


def main() -> None:
    """CLI 入口（替代 scan.sh 的直接执行）"""
    if len(sys.argv) < 2:
        # 默认输出 JSON 矩阵
        scanner = CapabilityScanner()
        print(json.dumps(scanner.generate_matrix(), ensure_ascii=False))
        return

    cmd = sys.argv[1]
    scanner = CapabilityScanner()

    match cmd:
        case "parrots":
            print(scanner.parrots)

        case "parrot-names":
            print("\n".join(scanner.parrot_names))

        case "tools":
            print(scanner.tools)

        case "tool-names":
            print("\n".join(scanner.tool_names))

        case "pages":
            print(scanner.pages)

        case "page-names":
            print("\n".join(scanner.page_names))

        case "tables":
            print(scanner.tables)

        case "table-names":
            print("\n".join(scanner.table_names))

        case "has" | "has-feature":
            if len(sys.argv) < 3:
                print("Usage: has <pattern>", file=sys.stderr)
                sys.exit(1)

            pattern = sys.argv[2]
            try:
                found = scanner.has_feature(pattern)
            except ValueError as e:
                print(f"错误: {e}", file=sys.stderr)
                sys.exit(1)

            if _can_use_emoji():
                status = f"✅ 已实现: {pattern}" if found else f"❌ 未实现: {pattern}"
            else:
                status = f"[PASS] 已实现: {pattern}" if found else f"[FAIL] 未实现: {pattern}"

            print(status)
            sys.exit(0 if found else 1)

        case "summary":
            print(scanner.summary())

        case "matrix" | "json" | "":
            print(json.dumps(scanner.generate_matrix(), ensure_ascii=False))

        case _:
            print(f"Unknown command: {cmd}", file=sys.stderr)
            print(
                "Usage: python -m product_insight.scan "
                "{parrots|tools|pages|tables|has|summary|matrix}",
                file=sys.stderr,
            )
            sys.exit(1)


if __name__ == "__main__":
    main()
