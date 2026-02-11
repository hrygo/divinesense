# 未暴露功能清单

> **保鲜状态**: ✅ 已验证 (2026-02-10) | **最后检查**: v0.97.0 | **建议**: 功能已验证

> 后端已实现、前端 hooks 已定义，但 UI 中尚未暴露的功能

---

## 概述

DivineSense 部分功能的 API 和前端 hooks 已完整实现，但尚未在用户界面中暴露入口。

---

## 🔴 待实现功能

### 1. 重复检测 (DetectDuplicates)

**实现位置：**
- 后端 API：`DetectDuplicates` RPC
- 前端 Hook：`web/src/hooks/useAIQueries.ts` - `useDetectDuplicates()`
- UI 集成：❌ 无

**功能描述：**
检测与当前 memo 重复或高度相似的内容，帮助用户避免创建重复笔记。

**建议入口：**
- 编辑器工具栏按钮
- 或保存时自动检测弹窗

---

### 2. 合并 Memos (MergeMemos)

**实现位置：**
- 后端 API：`MergeMemos` RPC
- 前端 Hook：`web/src/hooks/useAIQueries.ts` - `useMergeMemos()`
- UI 集成：❌ 无

**功能描述：**
将源 memo 的内容合并到目标 memo，保留目标 memo 的唯一标识符。

**建议入口：**
- Memo 操作菜单「合并到...」
- 或重复检测结果后的「一键合并」

---

## ✅ 已实现功能

以下功能已完整集成，无需额外工作：

| 功能 | UI 入口 | 位置 |
|:-----|:-------|:-----|
| **关联 Memos** | 编辑器「插入」菜单 | `InsertMenu.tsx` → LinkMemoDialog |
| **相关 Memos** | Memo 详情页 | `MemoDetail.tsx` → MemoRelatedList |
| **AI 标签建议** | 编辑器工具栏 | AITagSuggestPopover |
| **语义搜索** | 搜索栏 | 主页 |

---

## 实现优先级

### P0 - 高价值

| 功能 | 复杂度 | 用户价值 |
|:-----|:-------|:---------|
| 重复检测 | 低 | 🔴 避免重复内容 |
| 合并 Memos | 中 | 🔴 整理重复内容 |

---

## API 参考

### DetectDuplicates

```
POST /api/v1/ai/detect-duplicates
```

```json
{
  "title": "可选",
  "content": "笔记内容（必需）",
  "tags": ["标签1", "标签2"],
  "top_k": 5
}
```

### MergeMemos

```
POST /api/v1/ai/merge-memos
```

---

## 更新日志

| 日期 | 更新内容 |
|:-----|:---------|
| 2026-02-01 | 验证功能状态：LinkMemos/RelatedMemos 已暴露 |
| 2025-01-29 | 初始版本 |
