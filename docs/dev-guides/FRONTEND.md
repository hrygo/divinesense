# 前端开发指南

> **保鲜状态**: ✅ 已验证 (2026-02-03) | **最后检查**: v6.1 (Chat Apps 集成)

## 技术栈
- **框架**：React 18 + Vite 7
- **语言**：TypeScript
- **样式**：Tailwind CSS 4, Radix UI 组件库
- **状态管理**：TanStack Query (React Query)
- **国际化**：`web/src/locales/` (i18next)
- **Markdown**：React Markdown + KaTeX + Mermaid + GFM
- **日历**：FullCalendar 日历可视化

---

## 工作流程

### 命令（在 `web/` 目录下运行）

```bash
pnpm dev            # 启动开发服务器（端口 25173）
pnpm build          # 生产环境构建
pnpm lint           # 运行 TypeScript 和 Biome 检查
pnpm lint:fix       # 自动修复 lint 问题
```

### 从项目根目录

```bash
make web            # 启动前端开发服务器
make build-web      # 构建前端（生产环境）
make check-i18n     # 验证 i18n key 完整性
make ci-frontend    # 前端 CI 检查（lint + build）
```

---

## 🔒 Git Hooks

DivineSense 使用 **pre-commit + pre-push** hooks 确保代码质量。

> **详细规范**：参见 [Git 工作流](../../.claude/rules/git-workflow.md)

---

## Tailwind CSS 4 陷阱

### 关键：切勿使用语义化 `max-w-sm/md/lg/xl`

**根本原因**：Tailwind CSS 4 重新定义了这些类，使用 `--spacing-*` 变量（约 16px）替代传统的容器宽度（384-512px）。这会导致 Dialog、Sheet 和模态框坍缩成无法使用的「细条」。

| 语义化类 | Tailwind 3 | Tailwind 4 |
|:---------|:-----------|:-----------|
| `max-w-sm` | 384px | ~16px（损坏） |
| `max-w-md` | 448px | ~16px（损坏） |
| `max-w-lg` | 512px | ~16px（损坏） |

**错误**（会坍缩至约 16px）：
```tsx
<DialogContent className="max-w-md">
<SheetContent className="sm:max-w-sm">
```

**正确**（使用显式 rem 值）：
```tsx
<DialogContent className="max-w-[28rem]">  {/* 448px */}
<SheetContent className="sm:max-w-[24rem]"> {/* 384px */}
```

**参考表**：
| 宽度 | rem 值 | 用途 |
|:-----|:-------|:-----|
| 384px | `max-w-[24rem]` | 小对话框、侧边栏 |
| 448px | `max-w-[28rem]` | 标准对话框 |
| 512px | `max-w-[32rem]` | 大对话框、表单 |
| 672px | `max-w-[42rem]` | 宽内容 |

### 避免在 Grid 容器上使用 `max-w-*`

**错误**（导致重叠/挤压）：
```tsx
<div className="grid grid-cols-2 gap-3 w-full max-w-xs">
  {/* 320px / 2 = 每列 160px - 内容被挤压 */}
</div>
```

**正确**：
```tsx
<div className="grid grid-cols-2 gap-3 w-full">
  {/* 让 gap 和父级 padding 控制宽度 */}
</div>
```

| 适用 `max-w-*` | 不适用 `max-w-*` |
|:---------------|:----------------|
| Dialog/Modal/Popover | Grid 容器 |
| Tooltip/Alert 文本 | 需要填充的 Flex 项目 |
| Sidebar/Drawer | 响应式布局中的卡片 |

**规则**：Grid 使用 `gap` 而非 `max-w-*`。如果 `max-width / column_count < 200px`，不要使用 `max-w-*`。

### Go embed 兼容性

**关键**：Go 的 `//go:embed` 会忽略以下划线 `_` 开头的文件。

对于单二进制部署，前端构建产物必须避免生成以下划线开头的文件名。

**问题示例**：
```
lodash-es 内部模块被拆分为：
- _baseFlatten-xxx.js  ❌ 被 Go embed 忽略
- _baseMap-xxx.js       ❌ 被 Go embed 忽略
```

**解决方案**：在 `vite.config.mts` 中配置 `manualChunks` 将 lodash-es 打包为单个 chunk：

```typescript
manualChunks(id) {
  if (id.includes("lodash-es") || id.includes("/_base")) {
    return "lodash-vendor";  // 生成 lodash-vendor-xxx.js
  }
  // ...
}
```

**构建验证**：
```bash
ls web/dist/assets/ | grep "^_"  # 应该为空
```

详见：@docs/research/DEBUG_LESSONS.md → "Go embed 忽略以下划线开头的文件"

---

## 布局架构

> **保鲜状态**: ✅ 已验证 (2025-02-02) | **覆盖范围**: `web/src/layouts/*.tsx` | **最后检查**: v6.0

### 布局层级

```
RootLayout (全局导航 + 认证)
    │
    ├── MemoLayout (可折叠侧边栏：MemoExplorer)
    │   └── /, /explore, /archived, /u/:username
    │
    ├── GeneralLayout (无侧边栏，全宽内容)
    │   └── /knowledge-graph, /inboxes, /attachments
    │
    ├── AIChatLayout (固定侧边栏：AIChatSidebar)
    │   └── /chat
    │
    └── ScheduleLayout (固定侧边栏：ScheduleCalendar)
        └── /schedule
```

### 布局文件

| 文件 | 用途 | 侧边栏类型 | 响应式 |
|:-----|:-----|:-----------|:-------|
| `RootLayout.tsx` | 全局导航和认证 | 无 | N/A |
| `MemoLayout.tsx` | 内容密集页面 | 可折叠 `MemoExplorer` | md: 固定 |
| `GeneralLayout.tsx` | 全宽功能页面 | 无 | sm: 导航栏 |
| `AIChatLayout.tsx` | AI 聊天界面 | 固定 `AIChatSidebar` | 始终固定 |
| `ScheduleLayout.tsx` | 日程/日历 | 固定 `ScheduleCalendar` | 始终固定 |

### 功能布局模板

对于需要专用侧边栏的新功能页面：

```tsx
import { Outlet } from "react-router-dom";
import NavigationDrawer from "@/components/NavigationDrawer";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";

const FeatureLayout = () => {
  const lg = useMediaQuery("lg");

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* 移动端头部 */}
      <div className="lg:hidden flex-none flex items-center gap-2 px-4 py-3 border-b border-border/50 bg-background">
        <NavigationDrawer />
      </div>

      {/* 桌面侧边栏 */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-72 overflow-auto">
          <FeatureSidebar />
        </div>
      )}

      {/* 主内容 */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-72" : "")}>
        <Outlet />
      </div>
    </section>
  );
};
```

**侧边栏宽度选项**：
| 类 | 像素 | 用途 |
|:-------|:-----|:-----|
| `w-56` | 224px | 可折叠侧边栏（MemoLayout md） |
| `w-64` | 256px | 标准侧边栏 |
| `w-72` | 288px | 默认功能侧边栏（AIChat 等） |
| `w-80` | 320px | 宽侧边栏（Schedule） |

**响应式断点**：
| 断点 | 宽度 | 行为 |
|:-----|:-----|:-----|
| `sm` | 640px | 导航栏出现 |
| `md` | 768px | 侧边栏变为固定 |
| `lg` | 1024px | 完整侧边栏宽度 |

---

## 页面组件

> **保鲜状态**: ✅ 已验证 (2025-02-02) | **覆盖范围**: `web/src/pages/*.tsx` | **最后检查**: v6.0

### 可用页面

| 路径 | 组件 | 布局 | 用途 |
|:-----|:-----|:-----|:-----|
| `/` | `Home.tsx` | MemoLayout | 主时间线 + 笔记编辑器 |
| `/explore` | `Explore.tsx` | MemoLayout | 搜索和探索内容 |
| `/archived` | `Archived.tsx` | MemoLayout | 已归档笔记 |
| `/m/:id` | `MemoDetail.tsx` | None | 笔记详情页 |
| `/chat` | `AIChat.tsx` | AIChatLayout | AI 聊天界面 |
| `/schedule` | `Schedule.tsx` | ScheduleLayout | 日历视图 |
| `/inboxes` | `Inboxes.tsx` | GeneralLayout | 收件箱 |
| `/attachments` | `Attachments.tsx` | GeneralLayout | 附件管理 |
| `/knowledge-graph` | `KnowledgeGraph.tsx` | GeneralLayout | 知识图谱 |
| `/review` | `Review.tsx` | MemoLayout | 每日回顾 |
| `/setting` | `Setting.tsx` | MemoLayout | 用户设置 |
| `/u/:username` | `UserProfile.tsx` | MemoLayout | 公开用户资料 |
| `/auth/callback` | `AuthCallback.tsx` | None | OAuth 回调处理 |
| `/auth/signin` | `SignIn.tsx` | None | 登录页 |
| `/auth/signup` | `SignUp.tsx` | None | 注册页 |
| `/auth/admin` | `AdminSignIn.tsx` | None | 管理员登录 |
| `/404` | `NotFound.tsx` | None | 404 页面 |
| `/403` | `PermissionDenied.tsx` | None | 权限拒绝 |

### 添加新页面

1. 在 `web/src/pages/YourPage.tsx` 创建组件
2. 向 `web/src/locales/en.json` 和 `zh-Hans.json` 添加 i18n key
3. 在 `web/src/router/index.tsx` 添加路由：
   ```tsx
   {
     path: "/your-page",
     element: <YourPage />,
   }
   ```
4. 运行 `make check-i18n` 验证翻译

---

## 国际化 (i18n)

### 文件结构

```
web/src/locales/
    ├── en.json       # 英文翻译
    ├── zh-Hans.json  # 简体中文
    └── zh-Hant.json  # 繁体中文
```

### 添加新翻译

1. 向 `en.json` 添加 key：
   ```json
   {
     "your": {
       "key": "Your text"
     }
   }
   ```

2. 向 `zh-Hans.json` 添加 key：
   ```json
   {
     "your": {
       "key": "您的文本"
     }
   }
   ```

3. 在组件中使用：
   ```tsx
   import { t } from "i18next";

   const text = t("your.key");
   ```

4. 验证：`make check-i18n`

**关键**：切勿在组件中硬编码文本。始终使用 `t("key")`。

---

## 组件模式

### MemoCard

MemoCard 用于在整个应用中显示笔记内容：

```tsx
import MemoCard from "@/components/MemoCard";

<MemoCard
  memo={memo}
  onView={() => navigate(`/m/${memo.id}`)}
  onEdit={() => openEditDialog(memo)}
/>
```

### Dialog/Modal 模式

始终使用显式 rem 值作为宽度：

```tsx
import {
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

<DialogContent className="max-w-[28rem]">
  <DialogHeader>
    <DialogTitle>{t("title")}</DialogTitle>
  </DialogHeader>
  {/* 内容 */}
</DialogContent>
```

### ChatAppsSection

ChatAppsSection 用于设置页面管理聊天应用接入（Telegram、钉钉、WhatsApp）：

```tsx
import ChatAppsSection from "@/components/Settings/ChatAppsSection";

<ChatAppsSection />
```

**功能**：
- 注册/列出/删除聊天平台凭证
- 支持 Telegram Bot、钉钉群机器人
- Webhook URL 自动生成
- Token 加密存储（AES-256-GCM）

**相关 API**：
- `api.chatApp.listCredentials()` - 列出已注册凭证
- `api.chatApp.registerCredential()` - 注册新凭证
- `api.chatApp.deleteCredential()` - 删除凭证

**详见**：[Chat Apps 用户指南](../guides/CHAT_APPS.md)

---

## 状态管理

### 数据获取（TanStack Query）

```tsx
import { useQuery } from "@tanstack/react-query";

const { data, isLoading, error } = useQuery({
  queryKey: ["memos"],
  queryFn: () => api.memo.list(),
});
```

### 变更操作

```tsx
import { useMutation } from "@tanstack/react-query";

const mutation = useMutation({
  mutationFn: (memo: MemoCreate) => api.memo.create(memo),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ["memos"] });
  },
});
```
