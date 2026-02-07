# 文档归档中心

本目录用于存放 DivineSense 项目中已完成或过时的历史文档，按语义进行二级分类以方便检索。

## 目录结构

### 📂 [specs/](./specs/)
归档的实施规格说明书。
- [20260207_archive/](./specs/20260207_archive/): 包含 Sprint 0, Phase 2 和 Phase 3 的所有规格。
- [phase-1-completed/](./specs/phase-1-completed/): Phase 1 的历史规格。

### 📂 [research/](./research/)
技术研究报告、调研规划及路线图。
- [20260207_archive/](./research/20260207_archive/): Agent 技术报告、性能优化调研、会话模型研究等。
- [20260131_archive/](./research/20260131_archive/): 历史方法论、稳定性报告。

### 📂 [projects/](./projects/)
针对特定项目或功能专题的完整文档集。
- [parrot/](./projects/parrot/): Parrot 智能代理集成全套文档（计划、指南、进度报告）。

### 📂 [reviews/](./reviews/)
代码评审报告、安全审计及架构评审记录。
- 包含 RAG 评审、UI/UX 审计、代码修复计划等。

### 📂 [refactor-plans/](./refactor-plans/)
核心架构的重构提案和子系统集成计划。
- 包含 CC Runner 异步升级、统一搜索方案、Memos 重构计划等。

### 📂 [misc/](./misc/)
启动计划、ROI 分析、会议纪要等杂项文档。

---

## 归档原则
1. **分类明确**: 避免在根目录下直接放置文件。
2. **保留历史**: 归档文件应通过 `git mv` 移动以保留版本历史。
3. **索引同步**: 移动文件后需及时更新 `docs/README.md` 及相关索引文档。
