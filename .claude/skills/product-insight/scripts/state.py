#!/usr/bin/env python3
"""
Product Insight - State Management

管理洞察状态文件的读写操作。
支持 JSONL 格式、文件锁、并发安全。

替代 state.sh

Usage:
    from state import StateManager, InsightState

    state_mgr = StateManager()
    state_mgr.append(InsightState(...))
    latest = state_mgr.get_latest()
"""

import dataclasses
import fcntl
import json
import logging
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional, Any

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s][%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


@dataclasses.dataclass(frozen=True)
class InsightState:
    """洞察状态数据结构"""

    timestamp: str
    openclaw_sha: str
    divinesense_sha: str
    analyzed_features: list[str]
    discovered_functions: list[str]
    created_issues: list[int]

    def to_json(self) -> dict[str, Any]:
        """转换为 JSON 可序列化的字典"""
        return dataclasses.asdict(self)

    @classmethod
    def from_json(cls, data: dict[str, Any]) -> "InsightState":
        """从 JSON 字典创建实例"""
        return cls(
            timestamp=data.get("timestamp", ""),
            openclaw_sha=data.get("openclaw_sha", ""),
            divinesense_sha=data.get("divinesense_sha", ""),
            analyzed_features=data.get("analyzed_features", []),
            discovered_functions=data.get("discovered_functions", []),
            created_issues=data.get("created_issues", []),
        )

    def is_empty(self) -> bool:
        """检查是否为空状态"""
        return (
            not self.timestamp
            and not self.openclaw_sha
            and not self.divinesense_sha
            and not self.analyzed_features
            and not self.discovered_functions
            and not self.created_issues
        )


class ValidationError(Exception):
    """输入验证错误"""

    pass


class StateManager:
    """状态文件管理器"""

    # 空状态结构
    EMPTY_STATE = InsightState(
        timestamp="",
        openclaw_sha="",
        divinesense_sha="",
        analyzed_features=[],
        discovered_functions=[],
        created_issues=[],
    )

    def __init__(self, state_file: Optional[Path] = None, project_root: Optional[Path] = None):
        """
        初始化状态管理器

        Args:
            state_file: 状态文件路径，默认为 docs/research/benchmark/state.jsonl
            project_root: 项目根目录，自动检测
        """
        self.project_root = self._find_project_root(project_root)
        self.state_file = self._resolve_state_path(state_file)

    def _find_project_root(self, project_root: Optional[Path]) -> Path:
        """查找并验证项目根目录"""
        if project_root is None:
            # 从脚本位置向上查找
            # __file__ = scripts/state.py
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

    def _resolve_state_path(self, state_file: Optional[Path]) -> Path:
        """解析并验证状态文件路径"""
        if state_file is None:
            state_file = self.project_root / "docs" / "research" / "benchmark" / "state.jsonl"

        # 支持相对路径
        if not state_file.is_absolute():
            state_file = self.project_root / state_file

        state_file = state_file.resolve()

        # 验证路径在项目目录内
        try:
            state_file.relative_to(self.project_root)
        except ValueError:
            raise ValueError("错误: 状态文件必须在项目目录内")

        return state_file

    def validate_timestamp(self, timestamp: str) -> bool:
        """验证 ISO 8601 时间戳格式"""
        pattern = r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$"
        if not re.match(pattern, timestamp):
            raise ValidationError(
                f"无效的时间戳格式: {timestamp} (期望 ISO 8601, 如 2026-02-02T10:00:00Z)"
            )
        return True

    def validate_sha(self, sha: str, name: str = "SHA") -> bool:
        """验证 Git SHA 格式"""
        if not sha:
            raise ValidationError(f"{name} 不能为空")

        if not re.match(r"^[a-fA-F0-9]{7,64}$", sha):
            raise ValidationError(f"无效的 SHA 格式: {sha}")

        return True

    def validate_json_array(self, value: Any, name: str = "数组") -> bool:
        """验证 JSON 数组格式"""
        if value is None:
            raise ValidationError(f"{name} 不能为 None")

        if not isinstance(value, list):
            raise ValidationError(f"{name} 必须是数组类型，收到 {type(value).__name__}")

        return True

    def _acquire_lock(self, lock_file: Path) -> Any:
        """
        获取文件锁（跨平台）

        Unix: 使用 fcntl
        Windows: 使用 msvcrt（简化：macOS/Linux 场景为主）
        """
        lock_file.parent.mkdir(parents=True, exist_ok=True)

        # 尝试使用 fcntl（Unix）
        if hasattr(fcntl, "flock"):
            fd = os.open(lock_file, os.O_CREAT | os.O_WRONLY)
            try:
                fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
                return fd
            except BlockingIOError:
                os.close(fd)
                raise TimeoutError("无法获取文件锁（超时 10s）")
        else:
            # Windows 或无 flock 支持：创建锁文件作为标记
            if lock_file.exists():
                # 检查锁文件年龄
                age = datetime.now(timezone.utc).timestamp() - lock_file.stat().st_mtime
                if age < 10:  # 10秒内的锁认为有效
                    raise TimeoutError("无法获取文件锁（锁文件存在且未过期）")
                # 超时清理旧锁
                lock_file.unlink(missing_ok=True)

            lock_file.parent.mkdir(parents=True, exist_ok=True)
            lock_file.touch()
            return lock_file

    def _release_lock(self, lock: Any) -> None:
        """释放文件锁"""
        if isinstance(lock, int):
            # fcntl fd
            fcntl.flock(lock, fcntl.LOCK_UN)
            os.close(lock)
        elif isinstance(lock, Path):
            # 锁文件
            lock.unlink(missing_ok=True)

    def append(self, state: InsightState) -> None:
        """
        追加新状态记录（使用文件锁保证并发安全）

        Args:
            state: 要追加的状态
        """
        # 验证输入
        self.validate_timestamp(state.timestamp)
        self.validate_sha(state.openclaw_sha, "openclaw_sha")
        self.validate_sha(state.divinesense_sha, "divinesense_sha")
        self.validate_json_array(state.analyzed_features, "analyzed_features")
        self.validate_json_array(state.created_issues, "created_issues")

        # 确保目录存在
        self.state_file.parent.mkdir(parents=True, exist_ok=True)

        # 使用文件锁
        lock_file = self.state_file.with_suffix(".lock")

        try:
            lock = self._acquire_lock(lock_file)

            # 追加 JSON 行
            with open(self.state_file, "a", encoding="utf-8") as f:
                json.dump(state.to_json(), f, ensure_ascii=False)
                f.write("\n")

            logger.info(
                f"状态已追加: oc_sha={state.openclaw_sha[:7]}, "
                f"features={len(state.analyzed_features)} 个, "
                f"issues={len(state.created_issues)} 个"
            )

        except TimeoutError as e:
            logger.error(str(e))
            raise
        finally:
            try:
                self._release_lock(lock)
            except Exception:
                pass

    def get_latest(self) -> InsightState:
        """获取最新状态记录"""
        if not self.state_file.exists() or self.state_file.stat().st_size == 0:
            return self.EMPTY_STATE

        try:
            with open(self.state_file, "r", encoding="utf-8") as f:
                lines = f.readlines()

            if not lines:
                return self.EMPTY_STATE

            # 获取最后一行
            last_line = lines[-1].strip()
            data = json.loads(last_line)

            return InsightState.from_json(data)

        except (json.JSONDecodeError, KeyError) as e:
            logger.error(f"状态文件最后一条记录格式无效: {e}")
            return self.EMPTY_STATE

    def query(self, field: str) -> Any:
        """
        查询状态中的特定字段

        Args:
            field: 字段名 (如 openclaw_sha, timestamp)

        Returns:
            字段值
        """
        state = self.get_latest()
        return getattr(state, field, None)

    def exists(self) -> bool:
        """检查状态文件是否存在且有内容"""
        return self.state_file.exists() and self.state_file.stat().st_size > 0

    def count(self) -> int:
        """获取状态记录数量"""
        if not self.exists():
            return 0

        with open(self.state_file, "r", encoding="utf-8") as f:
            return sum(1 for _ in f)

    def summary(self) -> str:
        """显示状态摘要"""
        if not self.exists():
            return "状态文件不存在，首次运行\n建议运行: python -m product_insight.core init"

        state = self.get_latest()

        lines = [
            "洞察状态摘要:",
            f"  上次对标: {state.timestamp or '无'}",
            f"  OpenClaw SHA: {state.openclaw_sha or '无'}",
            f"  DivineSense SHA: {state.divinesense_sha or '无'}",
            f"  已分析功能: {len(state.analyzed_features)} 个",
            f"  已创建 Issue: {len(state.created_issues)} 个",
        ]
        return "\n".join(lines)

    def init(self) -> None:
        """初始化状态文件（首次运行）"""
        if self.exists():
            print(f"状态文件已存在: {self.state_file}", file=sys.stderr)
            print("如需重新初始化，请先删除现有文件", file=sys.stderr)
            raise FileExistsError("状态文件已存在")

        self.state_file.parent.mkdir(parents=True, exist_ok=True)

        # 获取当前 SHA
        import subprocess

        try:
            result = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                cwd=self.project_root,
                capture_output=True,
                text=True,
                check=True,
            )
            current_sha = result.stdout.strip()
        except (subprocess.CalledProcessError, FileNotFoundError):
            current_sha = "unknown"

        # 写入初始状态
        timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        initial_state = InsightState(
            timestamp=timestamp,
            openclaw_sha="",
            divinesense_sha=current_sha,
            analyzed_features=[],
            discovered_functions=[],
            created_issues=[],
        )

        with open(self.state_file, "w", encoding="utf-8") as f:
            json.dump(initial_state.to_json(), f, ensure_ascii=False)
            f.write("\n")

        print(f"状态文件已初始化: {self.state_file}")
        print(f"DivineSense SHA: {current_sha}")
        print("建议运行: python core.py run")


def main() -> None:
    """CLI 入口（替代 state.sh 的直接执行）"""
    if len(sys.argv) < 2:
        print("Usage: python state.py {append|get|query|summary|count|init}", file=sys.stderr)
        sys.exit(1)

    cmd = sys.argv[1]
    mgr = StateManager()

    match cmd:
        case "append":
            if len(sys.argv) < 6:
                print("Usage: append <timestamp> <oc_sha> <ds_sha> <features_json> <issues_json>", file=sys.stderr)
                sys.exit(1)

            state = InsightState(
                timestamp=sys.argv[2],
                openclaw_sha=sys.argv[3],
                divinesense_sha=sys.argv[4],
                analyzed_features=json.loads(sys.argv[5]),
                discovered_functions=[],
                created_issues=json.loads(sys.argv[6]) if len(sys.argv) > 6 else [],
            )
            mgr.append(state)

        case "get" | "latest":
            state = mgr.get_latest()
            print(json.dumps(state.to_json(), ensure_ascii=False))

        case "query":
            if len(sys.argv) < 3:
                print("Usage: query <field>", file=sys.stderr)
                sys.exit(1)
            value = mgr.query(sys.argv[2])
            print(value if value is not None else "")

        case "summary":
            print(mgr.summary())

        case "count":
            print(mgr.count())

        case "init":
            mgr.init()

        case _:
            print(f"Unknown command: {cmd}", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()
