# MemoBlock 彩色便签纸 + MemoList 禅意看板 调研报告

> **Issue**: #172 | **调研时间**: 2026-02-13 | **调研工具**: idea-researcher v3.1

---

## 需求概述

| 组件 | 改造目标 | 设计参考 |
|:-----|:---------|:---------|
| **MemoBlock** | 彩色便签纸设计 DNA | Apple Notes |
| **MemoList** | 禅意看板（无泳道） | 响应式双列 |

### 关键决策

| 问题 | 决策 |
|:-----|:-----|
| 颜色分配方式 | 标签映射（`#work` → 蓝色） |
| 物理质感程度 | 中等（彩色背景 + 阴影 + 折角） |
| 参考风格 | Apple Notes |
| 看板泳道 | 不需要（简化版） |
| 布局响应式 | 移动端单列 / PC 端双列 |
| 内容展示 | 200 字摘要（无折叠） |

---

## 技术可行性

### 标签颜色映射

- **现状**: `tags: string[]` 无颜色属性
- **方案**: 前端纯计算，哈希映射 + 规则覆盖
- **优势**: 无需后端改动，即时生效

### 6 色调色板

| 颜色 | 用途 | Light | Dark |
|:-----|:-----|:------|:-----|
| amber | 默认/无标签 | `bg-amber-50` | `dark:bg-amber-950/30` |
| rose | 个人/情感 | `bg-rose-50` | `dark:bg-rose-950/30` |
| sky | 工作/学习 | `bg-sky-50` | `dark:bg-sky-950/30` |
| emerald | 健康/习惯 | `bg-emerald-50` | `dark:bg-emerald-950/30` |
| violet | 创意/想法 | `bg-violet-50` | `dark:bg-violet-950/30` |
| orange | 紧急/待办 | `bg-orange-50` | `dark:bg-orange-950/30` |

### 看板布局

```css
.memo-kanban {
  display: grid;
  gap: 1rem;
  grid-template-columns: 1fr;
}

@media (min-width: 640px) {
  .memo-kanban {
    grid-template-columns: repeat(2, 1fr);
  }
}
```

---

## 实现方案

### 新增文件

| 文件 | 说明 |
|:-----|:-----|
| `web/src/utils/tag-colors.ts` | 便签颜色映射系统 |
| `web/src/utils/text.ts` | 文本截断工具 |
| `web/src/components/Memo/MemoBlockV3.tsx` | 便签组件 |
| `web/src/components/Memo/MemoListV3.tsx` | 看板布局组件 |

### 修改文件

| 文件 | 变更 |
|:-----|:-----|
| `web/src/components/Memo/index.ts` | 导出新组件 |
| `web/src/pages/Home.tsx` | 使用 MemoListV3 |
| `web/src/index.css` | 折角效果 CSS |

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|:-----|:-----|:---------|
| 深色模式对比度 | 中 | `dark:` 变体 + WCAG AA 验证 |
| 多标签颜色冲突 | 低 | 取第一个标签为主色 |
| 200 字截断断句 | 低 | 智能截断（空格/标点） |

---

## 复杂度

| 模块 | 工作量 |
|:-----|:-------|
| 便签颜色系统 | 0.5 人周 |
| MemoBlock 便签化 | 1.5 人周 |
| MemoList 看板化 | 1 人周 |
| 文本截断 | 0.3 人周 |
| **总计** | **~3 人周** |

---

## 参考资料

- Apple Notes 便签设计
- [DivineSense UI 设计系统](../design/ui-design-system.md)
- [MemoBlockV2 设计规范](../design/memo-block-v2-design.md)

---

*本报告由 idea-researcher 自动生成*
