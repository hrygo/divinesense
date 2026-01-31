# FE-001: Frontend Layout Architecture

## 概述

定义 Memos 前端的布局架构规范，确保新增功能时布局一致性和可维护性。

## Layout Hierarchy

```
RootLayout (global Nav + auth)
    │
    ├── MainLayout (collapsible sidebar: MemoExplorer)
    │   └── /, /explore, /archived, /u/:username
    │
    ├── AIChatLayout (fixed sidebar: AIChatSidebar)
    │   └── /chat
    │
    └── ScheduleLayout (fixed sidebar: ScheduleCalendar)
        └── /schedule
```

## Feature Layout Template

新增功能页面时，如需专用侧边栏，遵循以下模板：

```tsx
import { Outlet } from "react-router-dom";
import NavigationDrawer from "@/components/NavigationDrawer";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

const FeatureLayout = () => {
  const lg = useMediaQuery("lg");

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header */}
      <div className="lg:hidden flex-none flex items-center gap-2 px-4 py-3 border-b border-border/50 bg-background">
        <NavigationDrawer />
        {/* Mobile-specific controls */}
      </div>

      {/* Desktop Sidebar */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-{WIDTH} overflow-{auto|hidden}">
          <FeatureSidebar />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-{WIDTH}" : "")}>
        <Outlet />
      </div>
    </section>
  );
};
```

## Sidebar Width Options

| Class   | Pixels | Use Case                          |
|---------|--------|-----------------------------------|
| `w-56`  | 224px  | Collapsible sidebar (MainLayout md) |
| `w-64`  | 256px  | Standard sidebar                  |
| `w-72`  | 288px  | Default feature sidebar (AIChat)   |
| `w-80`  | 320px  | Wide sidebar (Schedule)           |

## Responsive Breakpoints

| Breakpoint | Width | Behavior                      |
|------------|-------|-------------------------------|
| `sm`       | 640px | Nav bar appears               |
| `md`       | 768px | Sidebar becomes fixed         |
| `lg`       | 1024px| Full sidebar width            |

## Layout Selection Guide

| Use Case                    | Layout Type      | Sidebar Type             |
|-----------------------------|------------------|--------------------------|
| Content-heavy pages         | `MainLayout`     | Collapsible `MemoExplorer` |
| Feature-specific (Chat, Schedule) | Dedicated Layout | Fixed feature sidebar |
| Full-screen modals          | No layout        | None                     |
| Simple pages                | `MainLayout`     | Reuse existing           |

## Adding a New Feature Layout

### 步骤

1. **创建 Layout 文件**: `web/src/layouts/FeatureLayout.tsx`
2. **创建 Sidebar 组件** (如需要): `web/src/components/FeatureSidebar.tsx`
3. **添加路由**: 在 `web/src/router/index.tsx` 中添加

### 路由配置示例

```tsx
{
  path: "/feature",
  element: <FeatureLayout />,
  children: [
    { index: true, element: <FeaturePage /> }
  ]
}
```

## 关键 CSS Classes

| Class                  | 用途                           |
|------------------------|--------------------------------|
| `h-svh`                | 全屏高度 (包含滚动条)           |
| `@container`           | 容器查询支持                   |
| `fixed top-0 left-16`  | 固定定位侧边栏 (Nav宽度 64px)   |
| `flex-1 min-h-0`       | 内容区域填充剩余空间            |
| `overflow-hidden`      | 防止外层滚动                   |
| `overflow-x-hidden`    | 仅允许内容区垂直滚动            |

## Context Providers

Layout 层可包裹 Context Provider：

```tsx
const FeatureLayout = () => {
  return (
    <FeatureProvider>
      <FeatureLayoutContent />
    </FeatureProvider>
  );
};
```

示例：
- `AIChatLayout` → `AIChatProvider`
- `ScheduleLayout` → `ScheduleContext` (全局，无需 provider)

## 状态

✅ 已完成
