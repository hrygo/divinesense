#!/usr/bin/env python3
"""
docs_helper.py 单元测试

运行: pytest docs_helper_test.py -v
或: python -m pytest docs_helper_test.py -v
"""

import json
import pytest
from pathlib import Path
from dataclasses import asdict

# 导入被测试模块
import sys
sys.path.insert(0, str(Path(__file__).parent))
from docs_helper import (
    Reference,
    DocNode,
    find_project_root,
    extract_references,
    classify_document,
    get_next_spec_id,
    CHINESE_REF_KEYWORDS,
    ENGLISH_REF_KEYWORDS,
)


class TestConstants:
    """测试常量定义"""

    def test_chinese_keywords(self):
        """中文引用关键词应包含预期字符"""
        assert "详" in CHINESE_REF_KEYWORDS
        assert "见" in CHINESE_REF_KEYWORDS
        assert "参" in CHINESE_REF_KEYWORDS
        assert "考" in CHINESE_REF_KEYWORDS
        assert "查" in CHINESE_REF_KEYWORDS
        assert "看" in CHINESE_REF_KEYWORDS

    def test_english_keywords(self):
        """英文引用关键词应包含预期单词"""
        assert "see" in ENGLISH_REF_KEYWORDS
        assert "refer" in ENGLISH_REF_KEYWORDS
        assert "to" in ENGLISH_REF_KEYWORDS


class TestProjectRoot:
    """测试项目根目录检测"""

    def test_find_project_root_returns_path(self):
        """应返回 Path 对象"""
        root = find_project_root()
        assert isinstance(root, Path)

    def test_find_project_root_exists(self):
        """返回的路径应该存在"""
        root = find_project_root()
        assert root.exists()

    def test_find_project_root_has_markers(self):
        """项目根应包含至少一个标记文件"""
        root = find_project_root()
        markers = [".git", "go.mod", "CLAUDE.md"]
        assert any((root / m).exists() for m in markers)


class TestReference:
    """测试 Reference 数据类"""

    def test_reference_creation(self):
        """应能创建 Reference 实例"""
        ref = Reference(
            source="test.md",
            target="docs/target.md",
            line=10,
            ref_type="markdown",
            context="some context"
        )
        assert ref.source == "test.md"
        assert ref.target == "docs/target.md"
        assert ref.line == 10
        assert ref.ref_type == "markdown"

    def test_reference_serializable(self):
        """Reference 应可序列化为 dict"""
        ref = Reference(
            source="test.md",
            target="docs/target.md",
            line=10,
            ref_type="markdown",
            context="some context"
        )
        data = asdict(ref)
        assert data["source"] == "test.md"
        assert data["line"] == 10


class TestDocNode:
    """测试 DocNode 数据类"""

    def test_doc_node_defaults(self):
        """默认值应为空列表"""
        node = DocNode(path="test.md")
        assert node.references == []
        assert node.referenced_by == []

    def test_doc_node_with_data(self):
        """应能存储引用数据"""
        node = DocNode(path="test.md")
        node.references.append({"target": "other.md", "type": "markdown"})
        node.referenced_by.append("source.md")
        assert len(node.references) == 1
        assert len(node.referenced_by) == 1


class TestExtractReferences:
    """测试引用提取"""

    def setup_method(self):
        """创建测试文件"""
        self.test_dir = Path(__file__).parent / "test_data"
        self.test_dir.mkdir(exist_ok=True)

    def teardown_method(self):
        """清理测试文件"""
        import shutil
        test_dir = Path(__file__).parent / "test_data"
        if test_dir.exists():
            shutil.rmtree(test_dir)

    def test_extract_markdown_reference(self):
        """应提取 Markdown 链接引用"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("[链接文字](docs/target.md)")
        refs = extract_references(test_file)
        assert len(refs) == 1
        assert refs[0].target == "docs/target.md"
        assert refs[0].ref_type == "markdown"

    def test_extract_at_syntax(self):
        """应提取 @ 语法引用"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("参见 @docs/architecture.md")
        refs = extract_references(test_file)
        assert len(refs) == 1
        assert refs[0].target == "docs/architecture.md"
        assert refs[0].ref_type == "at_syntax"

    def test_extract_chinese_plain_reference(self):
        """应提取中文平铺引用"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("详见 docs/guide.md")
        refs = extract_references(test_file)
        assert len(refs) == 1
        assert refs[0].target == "docs/guide.md"
        assert refs[0].ref_type == "plain"

    def test_extract_english_plain_reference(self):
        """应提取英文平铺引用"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("see docs/api.md")
        refs = extract_references(test_file)
        assert len(refs) == 1
        assert refs[0].target == "docs/api.md"
        assert refs[0].ref_type == "plain"

    def test_extract_multiple_references(self):
        """应提取多个引用"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("""
# 文档标题
参见 @docs/architecture.md

详见 docs/guide.md

更多信息见 [API 文档](docs/api.md)
""")
        refs = extract_references(test_file)
        assert len(refs) == 3

    def test_extract_no_references(self):
        """无引用时应返回空列表"""
        test_file = self.test_dir / "test.md"
        test_file.write_text("# 标题\n\n无引用的普通文本")
        refs = extract_references(test_file)
        assert len(refs) == 0

    def test_extract_from_nonexistent_file(self):
        """文件不存在时应返回空列表"""
        test_file = self.test_dir / "nonexistent.md"
        refs = extract_references(test_file)
        assert len(refs) == 0


class TestClassifyDocument:
    """测试文档分类"""

    def setup_method(self):
        """设置测试环境"""
        from docs_helper import DOCS_DIR
        self.test_dir = DOCS_DIR

    def test_classify_core_document(self):
        """00- 开头的文件应分类为核心路线图"""
        file = self.test_dir / "00-master-roadmap.md"
        category, description = classify_document(file)
        assert category == "core"
        assert description == "核心路线图"

    def test_classify_research_report(self):
        """*-research.md 结尾应分类为研究报告"""
        file = self.test_dir / "assistant-research.md"
        category, description = classify_document(file)
        assert category == "reports"
        assert description == "研究报告"

    def test_classify_roadmap(self):
        """*-roadmap.md 结尾应分类为路线图"""
        file = self.test_dir / "feature-roadmap.md"
        category, description = classify_document(file)
        assert category == "roadmaps"
        assert description == "路线图"

    def test_classify_practice(self):
        """PRACTICE 开头应分类为最佳实践"""
        file = self.test_dir / "PRACTICE_GUIDE.md"
        category, description = classify_document(file)
        assert category == "practices"
        assert description == "最佳实践"

    def test_classify_other(self):
        """其他文件应分类为其他"""
        file = self.test_dir / "random-file.md"
        category, description = classify_document(file)
        assert category == "other"
        assert description == "其他"


class TestGetNextSpecId:
    """测试 Spec ID 生成"""

    def test_spec_id_format(self):
        """生成的 Spec ID 应符合 P{X}-{Y}{ZZZ} 格式"""
        # 这是一个简化的测试，实际测试需要 mock 文件系统
        spec_id = get_next_spec_id(2, "a")
        # team 字母是小写
        assert spec_id.startswith("P2-a")
        assert len(spec_id) >= 6


class TestIntegration:
    """集成测试"""

    def test_full_reference_extraction_workflow(self):
        """测试完整的引用提取工作流"""
        # 创建测试文件
        test_dir = Path(__file__).parent / "test_data"
        test_dir.mkdir(exist_ok=True)

        test_file = test_dir / "complex.md"
        content = """# 复杂文档

## 引用测试

- Markdown 链接: [架构](docs/dev-guides/ARCHITECTURE.md)
- @ 语法: @docs/research/00-master-roadmap.md
- 中文平铺: 详见 docs/dev-guides/FRONTEND.md
- 英文平铺: see docs/dev-guides/BACKEND_DB.md

代码示例:
```
// 详见 docs/api/users.md
const api = getAPI();
```
"""
        test_file.write_text(content)

        refs = extract_references(test_file)

        # 应提取至少 4 个引用
        assert len(refs) >= 4

        # 验证引用类型分布
        ref_types = {r.ref_type for r in refs}
        assert "markdown" in ref_types
        assert "at_syntax" in ref_types or "plain" in ref_types

        # 清理
        import shutil
        shutil.rmtree(test_dir)


if __name__ == "__main__":
    # 支持直接运行
    pytest.main([__file__, "-v"])
