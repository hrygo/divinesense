# 已归档规格文档 (Archived Specs)

> **归档时间**: 2026-01-23 → 2026-01-31
> **状态**: 历史参考，不用于当前开发

---

## 📁 目录结构

```
archived/specs/
├── ai/                          # AI 后端规格 (AI-001 ~ AI-016)
│   ├── AI-001-proto-definition.md
│   ├── AI-002-profile-config.md
│   ├── AI-003-db-migration.md
│   ├── AI-004-memo-embedding-model.md
│   ├── AI-005-driver-interface.md
│   ├── AI-006-postgres-vector-search.md
│   ├── AI-007-ai-plugin-config.md
│   ├── AI-008-embedding-service.md
│   ├── AI-009-reranker-service.md
│   ├── AI-010-llm-service.md
│   ├── AI-011-embedding-runner.md
│   ├── AI-012-semantic-search-api.md
│   ├── AI-013-chat-api.md
│   ├── AI-014-suggest-tags-api.md
│   ├── AI-015-related-memos-api.md
│   └── AI-016-frontend-hooks.md
├── frontend/                    # 前端规格
│   └── FE-001-layout-architecture.md
├── general/                     # 通用规格
│   ├── SPEC-001-INFRA-BASE-PARROT.md
│   ├── SPEC-001-docker-compose-postgres.md
│   ├── SPEC-002-AGENT-MEMO-CREATIVE.md
│   ├── SPEC-002-pgvector-extension.md
│   ├── SPEC-003-AGENT-AMAZING.md
│   ├── SPEC-003-ai-provider-init.md
│   ├── SPEC-004-FRONTEND-UI-UX.md
│   ├── SPEC-004-memo-embedding-pipeline.md
│   ├── SPEC-005-vector-search-rerank.md
│   ├── SPEC-006-chat-api-streaming.md
│   ├── SPEC-007-frontend-ai-chat-drawer.md
│   ├── SPEC-008-chat-ui-optimization.md
│   ├── product-iteration-plan.md
│   ├── schedule-ai-native-refactor.md
│   ├── schedule-management-ux-optimization.md
│   └── sync-user-role-and-mode-refactor-20260121.md
├── INDEX.md                     # 原始索引
└── README.md                    # 本文件
```

---

## 📊 归档原因

这些规格文档已实现完成，归档用于：
1. **历史参考** - 了解设计决策和实现思路
2. **迁移参考** - 如需重新实现类似功能
3. **审计追踪** - 记录产品演进历程

---

## 🔍 当前规格文档

活跃开发的规格文档已移至 [`../../specs/`](../../specs/)：

- **Phase 1-3 Specs** - Sprint 实施规格
- **Evolution Mode** - 进化模式规格
- **INDEX.md** - 当前规格索引

---

> **维护**: 无需更新，仅作历史参考
