# 页面间距统一规范调研报告

> **调研日期**: 2026-02-04
> **Issue**: [#74](https://github.com/hrygo/divinesense/issues/74)
> **版本**: v1.0

---

## 一、调研背景

DivineSense 各主模块页面的 Header 与主体内容间距不统一，导致视觉体验不一致。需要调研 PC 端与移动端的布局现状，并制定统一的 UI 规范。

---

## 二、当前现状分析

### 2.1 Layout 组件对比

| 组件 | 文件 | 侧边栏宽度 | 侧边栏内边距 | 顶部间距 | 最大宽度 | 响应式断点 |
|:-----|:-----|:----------|:------------|:---------|:---------|:-----------|
| **MemoLayout** | `MemoLayout.tsx` | `md:w-56`, `lg:w-72` | `px-3 py-6` | `md:pt-6` | `max-w-[100rem]` | `md`, `lg` |
| **AIChatLayout** | `AIChatLayout.tsx` | `w-72` | `pt-2` | 无 | 无 | `lg` |
| **ScheduleLayout** | `ScheduleLayout.tsx` | `w-80` | `py-4 px-3` | 无 | 无 | `lg` |
| **GeneralLayout** | `GeneralLayout.tsx` | 无 | 无 | 无 | 无 | `sm` |

### 2.2 页面组件内联间距

| 页面 | 文件 | 内联间距设置 |
|:-----|:-----|:------------|
| **Inboxes** | `Inboxes.tsx` | `sm:pt-3 md:pt-6 pb-8 max-w-[100rem]` |
| **Attachments** | `Attachments.tsx` | `sm:pt-3 md:pt-6 pb-8 max-w-[100rem]` |
| **Setting** | `Setting.tsx` | `sm:pt-3 md:pt-6 pb-8 max-w-[100rem]` |
| **Review** | `Review.tsx` | `px-4 py-6 max-w-[100rem]` |
| **Home** | `Home.tsx` | 无（依赖 MemoLayout） |
| **Explore** | `Explore.tsx` | 无（依赖 MemoLayout） |

### 2.3 移动端 Header 对比

| Layout | 高度 | padding | 边框 |
|:-------|:-----|:-------|:-----|
| MemoLayout | 未指定 | - | - |
| AIChatLayout | `h-14` | `px-4` | `border-b` |
| ScheduleLayout | `h-14` | `px-4` | `border-b border-border/50` |
| GeneralLayout | `h-14` | `px-4 py-3` | `border-b` |

---

## 三、问题汇总

### 3.1 顶部间距不统一

- MemoLayout: `md:pt-6` ✅
- AIChatLayout: 无 pt 间距 ❌
- ScheduleLayout: 无 pt 间距 ❌
- GeneralLayout: 无 pt 间距，各页面自行管理 ❌

### 3.2 侧边栏宽度不统一

- MemoLayout: `md:w-56`, `lg:w-72`
- AIChatLayout: `w-72` (288px)
- ScheduleLayout: `w-80` (320px) ❌

### 3.3 侧边栏内边距不统一

- MemoLayout: `px-3 py-6`
- AIChatLayout: `pt-2` ❌
- ScheduleLayout: `py-4 px-3`

### 3.4 响应式断点不统一

- MemoLayout: `md`, `lg`
- AIChatLayout: `lg`
- ScheduleLayout: `lg`
- GeneralLayout: `sm` ❌

### 3.5 最大宽度设置分散

- MemoLayout: `max-w-[100rem]` ✅
- GeneralLayout: 无
- 各页面自行设置（Inboxes/Attachments/Setting/Review）

---

## 四、统一规范方案

### 4.1 Layout 层统一结构

```tsx
<section className="@container w-full h-screen/h-screen overflow-hidden">
  {/* Mobile Header */}
  <div className="lg:hidden flex-none relative ... px-4 h-14 shrink-0 border-b">
    {/* Header content */}
  </div>

  {/* Desktop Sidebar */}
  <div className="fixed top-0 left-16 shrink-0 h-svh border-r lg:block w-72">
    {/* Sidebar content with px-3 py-6 */}
  </div>

  {/* Main Content */}
  <div className="flex-1 min-h-0 overflow-y-auto w-full lg:pl-72">
    <div className="w-full mx-auto px-4 sm:px-6 pt-4 sm:pt-6 pb-8 max-w-[100rem]">
      <Outlet />
    </div>
  </div>
</section>
```

### 4.2 Token 定义表

| 用途 | 移动端 (<768px) | PC 端 (≥768px) |
|:-----|:---------------|:---------------|
| 侧边栏宽度 | N/A | `w-72` (288px) |
| 侧边栏内边距 | N/A | `px-3 py-6` |
| 顶部间距 | `pt-4` (16px) | `pt-6` (24px) |
| 底部间距 | `pb-8` (32px) | `pb-8` (32px) |
| 水平内边距 | `px-4` (16px) | `sm:px-6` (24px) |
| 最大内容宽度 | 100% | `max-w-[100rem]` (1600px) |
| Mobile Header | `h-14` (56px) | 隐藏 |
| 响应式断点 | `lg` (1024px) | `lg` (1024px) |

### 4.3 特殊场景说明

1. **AIChatLayout**: 无 pt-6 间距是有意设计（全屏聊天体验），保持现状但添加注释
2. **ScheduleLayout**: 侧边栏 w-80 (320px) 需要更宽空间显示日历，保持现状但添加注释
3. **全屏页面**: 如 AIChat 沉浸模式，可使用 `h-screen` 而非 `h-full`

---

## 五、实施方案

### 5.1 创建间距规范常量

**文件**: `web/src/styles/layout.ts`

```typescript
export const LAYOUT_SPACING = {
  sidebarWidth: "w-72",        // 288px
  sidebarPadding: "px-3 py-6",
  mobileHeaderHeight: "h-14",  // 56px
  paddingTopMobile: "pt-4",    // 16px
  paddingTopDesktop: "pt-6",   // 24px
  paddingBottom: "pb-8",        // 32px
  paddingXMobile: "px-4",      // 16px
  paddingXDesktop: "sm:px-6",  // 24px
  maxWidth: "max-w-[100rem]",  // 1600px
} as const;
```

### 5.2 修改文件清单

| 文件 | 修改类型 | 说明 |
|:-----|:---------|:-----|
| `web/src/styles/layout.ts` | 新增 | 创建间距规范常量 |
| `web/src/layouts/GeneralLayout.tsx` | 重构 | 按规范重构，添加统一间距 |
| `web/src/layouts/AIChatLayout.tsx` | 文档 | 添加注释说明无 pt-6 设计 |
| `web/src/layouts/ScheduleLayout.tsx` | 文档 | 添加注释说明 w-80 特殊用途 |
| `web/src/pages/Inboxes.tsx` | 简化 | 移除页面内冗余间距 |
| `web/src/pages/Attachments.tsx` | 简化 | 移除页面内冗余间距 |
| `web/src/pages/Setting.tsx` | 简化 | 移除页面内冗余间距 |
| `web/src/pages/Review.tsx` | 简化 | 移除页面内冗余间距 |
| `docs/dev-guides/FRONTEND.md` | 更新 | 添加间距规范说明 |

### 5.3 风险与缓解

| 风险 | 影响 | 措施 |
|:-----|:-----|:-----|
| AIChat 沉浸模式破坏 | 中 | 排除在外，添加注释说明 |
| Schedule 日历空间不足 | 低 | 保持 w-80，添加注释说明 |
| 现有用户视觉习惯变化 | 低 | 变化细微，主要是统一 |

---

## 六、验收标准

- [ ] `make check-all` 通过
- [ ] 所有 Layout 组件遵循统一间距规范
- [ ] PC 端顶部间距统一为 `pt-6`（除特殊场景）
- [ ] 移动端顶部间距统一为 `pt-4`
- [ ] 侧边栏宽度统一为 `w-72`（Schedule 例外）
- [ ] 更新 `docs/dev-guides/FRONTEND.md` 文档

---

## 七、参考资源

- [Frontend 开发指南](../dev-guides/FRONTEND.md)
- [架构文档](../dev-guides/ARCHITECTURE.md)
- [Issue #74](https://github.com/hrygo/divinesense/issues/74)
