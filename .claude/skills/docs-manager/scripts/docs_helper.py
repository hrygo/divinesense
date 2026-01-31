#!/usr/bin/env python3
"""
DivineSense 文档管理辅助脚本 v2.0

为 docs-manager skill 提供核心功能实现：
1. Spec ID 自动生成
2. 智能分类规则
3. 重复内容检测 (优化版)
4. 链接有效性检查
5. 引用图构建
6. JSON 输出模式 (AI 友好)

使用方式：
  python docs_helper.py check        # 检查文档完整性
  python docs_helper.py refs        # 构建引用图
  python docs_helper.py refs --json # JSON 输出
  python docs_helper.py next-spec   # 生成下一个 Spec ID
  python docs_helper.py duplicates  # 检测重复内容
"""

import argparse
import json
import logging
import os
import re
import sys
from dataclasses import dataclass, asdict
from pathlib import Path
from typing import Dict, List, Tuple, Optional
from collections import defaultdict
import difflib

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(levelname)s: %(message)s'
)
logger = logging.getLogger(__name__)


def find_project_root() -> Path:
    """通过 .git 目录或 go.mod 定位项目根目录"""
    # 从当前脚本开始向上查找
    current = Path(__file__).resolve().parent
    max_iterations = 10  # 防止无限循环

    for _ in range(max_iterations):
        # 检查当前目录是否是项目根（有 .git 或 go.mod）
        if (current / ".git").exists():
            return current
        if (current / "go.mod").exists():
            return current
        if (current / "CLAUDE.md").exists():  # DivineSense 特有文件
            return current

        # 向上一层
        current = current.parent

    # 降级方案：使用固定深度
    # .claude/skills/docs-manager/docs_helper.py → 项目根 = 4 层向上
    return Path(__file__).parent.parent.parent.parent


PROJECT_ROOT = find_project_root()
DOCS_DIR = PROJECT_ROOT / "docs"

logger.info(f"项目根目录: {PROJECT_ROOT}")
logger.info(f"文档目录: {DOCS_DIR}")


@dataclass
class Reference:
    """文档引用"""
    source: str      # 引用源文件
    target: str      # 被引用文件
    line: int        # 行号
    ref_type: str    # 引用类型
    context: str     # 上下文


@dataclass
class DocNode:
    """文档节点"""
    path: str
    references: List[dict] = None
    referenced_by: List[str] = None

    def __post_init__(self):
        if self.references is None:
            self.references = []
        if self.referenced_by is None:
            self.referenced_by = []


def glob_docs(pattern: str = "**/*.md") -> List[Path]:
    """扫描文档文件，排除 node_modules"""
    docs = []
    for doc in DOCS_DIR.rglob(pattern):
        # 排除不需要的目录
        if "node_modules" in str(doc):
            continue
        docs.append(doc)
    return docs


def extract_references(file_path: Path) -> List[Reference]:
    """从文件中提取所有引用"""
    references = []

    try:
        content = file_path.read_text(encoding="utf-8")
    except PermissionError:
        logger.warning(f"无权限读取: {file_path}")
        return references
    except UnicodeDecodeError:
        logger.warning(f"编码错误: {file_path}")
        return references
    except Exception as e:
        logger.error(f"读取失败 {file_path}: {e}")
        return references

    lines = content.split("\n")

    # 改进的引用正则模式
    patterns = [
        # Markdown 链接
        (r"\[([^\]]+)\]\((docs/[^)]+\.md)\)", "markdown"),
        (r"\[([^\]]+)\]\(\.\./(docs/[^)]+\.md)\)", "markdown"),
        # @ 语法
        (r"@docs/[\w/-]+\.md", "at_syntax"),
        # 绝对 URL
        (r"https://github\.com/[^/]+/[^/]+/docs/[\w/-]+\.md", "absolute_url"),
        # 代码注释 - 改进的正则
        (r"(?:[\u8be6\u89c1\u53c2\u8003]+|see|refer to)\s+[`'\"()]?docs/[\w/-]+\.md", "plain"),
    ]

    for line_no, line in enumerate(lines, 1):
        for pattern, ref_type in patterns:
            try:
                for match in re.finditer(pattern, line, re.IGNORECASE):
                    target = match.group(0)

                    # 清理目标路径
                    if ref_type == "at_syntax":
                        target = target.replace("@", "")
                    elif ref_type == "plain":
                        # 提取 docs/xxx.md 部分
                        doc_match = re.search(r"docs/[\w/-]+\.md", target)
                        if doc_match:
                            target = doc_match.group(0)
                        else:
                            continue
                    elif ref_type == "absolute_url":
                        target = "/docs/" + target.split("/docs/")[-1]

                    references.append(Reference(
                        source=str(file_path.relative_to(PROJECT_ROOT)),
                        target=target,
                        line=line_no,
                        ref_type=ref_type,
                        context=line.strip()[:80]
                    ))
            except Exception as e:
                logger.debug(f"正则匹配失败: {e}")

    return references


def build_reference_graph() -> Dict[str, DocNode]:
    """构建文档引用图"""
    graph = {}
    docs = glob_docs()

    logger.info(f"扫描 {len(docs)} 个文档...")

    for doc_file in docs:
        try:
            rel_path = str(doc_file.relative_to(DOCS_DIR))
            node = DocNode(path=rel_path)

            refs = extract_references(doc_file)
            for ref in refs:
                node.references.append({
                    "target": ref.target,
                    "type": ref.ref_type,
                    "line": ref.line,
                    "context": ref.context
                })

            graph[rel_path] = node
        except Exception as e:
            logger.warning(f"处理文档失败 {doc_file}: {e}")

    # 构建反向引用
    for path, node in graph.items():
        for ref in node.references:
            target = ref["target"]
            # 标准化路径
            if target.startswith("docs/"):
                target = target[5:]
            elif target.startswith("../docs/"):
                target = target[8:]
            elif target.startswith("/docs/"):
                target = target[6:]

            if target in graph:
                if path not in graph[target].referenced_by:
                    graph[target].referenced_by.append(path)

    return graph


def check_links() -> Dict[str, List[str]]:
    """检查链接有效性"""
    issues = defaultdict(list)
    graph = build_reference_graph()
    existing_docs = set(graph.keys())

    for path, node in graph.items():
        for ref in node.references:
            target = ref["target"]
            # 标准化
            if target.startswith("docs/"):
                target = target[5:]
            elif target.startswith("../docs/"):
                target = target[8:]
            elif target.startswith("/docs/"):
                target = target[6:]

            if target not in existing_docs:
                issues["broken_links"].append({
                    "source": path,
                    "line": ref["line"],
                    "target": target,
                    "type": ref["type"]
                })

    return dict(issues)


def get_next_spec_id(phase: int, team: str) -> str:
    """生成下一个 Spec ID"""
    pattern = f"P{phase}-{team}*.md"
    team_dir = DOCS_DIR / "specs" / f"phase-{phase}" / f"team-{team}"

    if not team_dir.exists():
        return f"P{phase}-{team}001"

    existing = list(team_dir.glob(pattern))

    if not existing:
        return f"P{phase}-{team}001"

    max_id = 0
    for f in existing:
        match = re.search(rf"P{phase}-{team}(\d+)", f.stem)
        if match:
            max_id = max(max_id, int(match.group(1)))

    return f"P{phase}-{team}{max_id + 1:03d}"


def detect_duplicates_fast(threshold: float = 0.7) -> List[Tuple[str, str, float]]:
    """快速检测重复内容 - 仅检查前 1000 个字符"""
    duplicates = []
    docs = glob_docs()
    contents = {}

    # 过滤和预处理
    for doc in docs:
        if "archived" in str(doc) or "node_modules" in str(doc):
            continue
        try:
            content = doc.read_text(encoding="utf-8", errors="ignore")
            # 只取前 1000 字符快速检测
            preview = content[:1000]
            lines = [l.strip() for l in preview.split("\n") if l.strip()]
            contents[doc] = " ".join(lines)
        except Exception:
            pass

    # 两两比较 (仅预览)
    doc_list = list(contents.items())
    for i in range(len(doc_list)):
        for j in range(i + 1, len(doc_list)):
            doc1, content1 = doc_list[i]
            doc2, content2 = doc_list[j]

            similarity = difflib.SequenceMatcher(None, content1, content2).ratio()

            if similarity >= threshold:
                duplicates.append((
                    str(doc1.relative_to(DOCS_DIR)),
                    str(doc2.relative_to(DOCS_DIR)),
                    similarity
                ))

    return sorted(duplicates, key=lambda x: -x[2])


def classify_document(file_path: Path) -> Tuple[str, str]:
    """智能分类文档"""
    name = file_path.name
    rel_path = str(file_path.relative_to(DOCS_DIR))

    if name.startswith("00-"):
        return "core", "核心路线图"
    if name.endswith("-research.md"):
        return "reports", "研究报告"
    if name.endswith("-roadmap.md"):
        return "roadmaps", "路线图"
    if "PRACTICE" in name or name == "DEBUG_LESSONS.md":
        return "practices", "最佳实践"

    spec_match = re.match(r"phase-(\d)/team-([abc])/", rel_path)
    if spec_match:
        phase, team = spec_match.groups()
        return f"Phase {phase}", f"Team {team.upper()}"

    return "other", "其他"


def print_refs_table(graph: Dict[str, DocNode]):
    """打印引用关系表"""
    # 按被引用次数排序
    hot_docs = sorted(
        [(path, len(node.referenced_by)) for path, node in graph.items()],
        key=lambda x: -x[1]
    )[:10]

    print("\n🔥 热门文档 (被引用最多):")
    for path, count in hot_docs:
        if count > 0:
            print(f"  {count:2d} ← {path}")


def main():
    """主入口"""
    parser = argparse.ArgumentParser(description="DivineSense 文档管理工具")
    parser.add_argument("command", nargs="?", default="check",
                       choices=["check", "refs", "next-spec", "duplicates", "tree"],
                       help="命令")
    parser.add_argument("--json", action="store_true", help="JSON 输出")
    parser.add_argument("--phase", type=int, help="Phase (next-spec)")
    parser.add_argument("--team", type=str, help="Team (next-spec)")

    args = parser.parse_args()
    command = args.command

    if command == "check":
        print("📋 文档检查报告")
        print("=" * 50)

        issues = check_links()
        broken = issues.get("broken_links", [])

        if args.json:
            print(json.dumps(broken, indent=2, ensure_ascii=False))
        elif broken:
            print(f"\n✗ 断链 ({len(broken)}):")
            for issue in broken:
                src = issue['source']
                line = issue['line']
                tgt = issue['target']
                print(f"  🔗 {src}:{line} → {tgt}")
        else:
            print("\n✓ 无断链")

    elif command == "refs":
        print("🔗 引用关系图")
        print("=" * 50)

        graph = build_reference_graph()

        if args.json:
            # 转换为 JSON 可序列化格式
            output = {}
            for path, node in graph.items():
                output[path] = {
                    "references": node.references,
                    "referenced_by": node.referenced_by
                }
            print(json.dumps(output, indent=2, ensure_ascii=False))
        else:
            for path, node in sorted(graph.items()):
                if node.references or node.referenced_by:
                    print(f"\n{path}:")
                    print(f"  引用: {len(node.references)} 个")
                    print(f"  被引用: {len(node.referenced_by)} 次")

            print_refs_table(graph)

    elif command == "next-spec":
        phase = args.phase or 2
        team = args.team or "a"
        spec_id = get_next_spec_id(phase, team)

        if args.json:
            print(json.dumps({"spec_id": spec_id, "phase": phase, "team": team}))
        else:
            print(f"📄 下一个 Spec ID: {spec_id}")

    elif command == "duplicates":
        print("🔍 重复内容检测")
        print("=" * 50)

        dupes = detect_duplicates_fast()

        if args.json:
            print(json.dumps(dupes, indent=2))
        elif dupes:
            for doc1, doc2, sim in dupes[:10]:
                print(f"\n{sim:.1%} 相似度:")
                print(f"  1. {doc1}")
                print(f"  2. {doc2}")
        else:
            print("\n✓ 无重复内容")

    elif command == "tree":
        print("📂 docs/ 结构")
        print("=" * 50)

        for item in sorted(DOCS_DIR.iterdir()):
            if item.is_dir() and not item.name.startswith("."):
                count = len(list(item.rglob("*.md")))
                print(f"📁 {item.name}/ ({count} 个 md 文件)")
            elif item.suffix == ".md":
                print(f"📄 {item.name}")


if __name__ == "__main__":
    main()
