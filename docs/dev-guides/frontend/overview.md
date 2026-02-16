# 前端开发指南

> **保鲜状态**: ✅ 已更新 (2026-02-12) | **最后检查**: v0.99.0 (Orchestrator-Workers)

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

> **保鲜状态**: ✅ 已更新 (2026-02-12) | **覆盖范围**: `web/src/layouts/*.tsx` | **最后检查**: v0.99.0

### 布局层级

```
RootLayout (全局导航 + 认证)
    │
    ├── MemoLayout (可折叠侧边栏：MemoExplorer)
    │   └── /memo, /explore, /archived, /u/:username
    │
    ├── GeneralLayout (无侧边栏，全宽内容)
    │   └── /knowledge-graph, /inbox, /attachments, /setting, /memos/:uid, /review, /403, /404
    │
    ├── AIChatLayout (固定侧边栏：AIChatSidebar，多模式主题)
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

### 侧边栏宽度规范（统一标准）

> **更新时间**: 2026-02-12 | **规范版本**: v1.0

**所有 Sidebar 组件必须使用 `w-80` (320px) 作为标准宽度。**

| 组件类型 | 宽度类 | 像素值 | 主内容左边距 |
|:---------|:-------|:-------|:-------------|
| **Desktop Sidebar** | `w-80` | 320px | `pl-80` |
| **Mobile Sheet** | `w-80 max-w-full` | 320px | - |
| **Navigation Drawer** | `w-80 max-w-full` | 320px | - |
| **MemoDetail Sidebar** | `sm:w-80` | 320px | - |

**适用范围**（所有已实现组件）：
- `MemoLayout` Desktop Sidebar (`w-80` + `pl-80`)
- `MemoLayout` Mobile Sheet (`w-80 max-w-full`)
- `AIChatLayout` Desktop Sidebar (`w-80` + `pl-80`)
- `AIChatLayout` Mobile Sheet (`w-80 max-w-full`)
- `ScheduleLayout` Desktop Sidebar (`w-80` + `pl-80`)
- `NavigationDrawer` (`w-80 max-w-full`)
- `MemoDetailSidebarDrawer` (`sm:w-80`)
- `MemoExplorerDrawer` (`w-80 max-w-full`)

**内部间距规范**（与 AIChatSidebar 对齐）：
```tsx
// Sidebar 内容容器
<MemoExplorer className="h-full px-4 pt-4" />
<AIChatSidebar className="h-full" />
  ├── 新建按钮区域: px-4 pt-4 pb-2
  ├── Tabs 区域: px-4 pb-2
  └── 面板内容: overflow-hidden
```

**新建功能布局模板**：

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

      {/* 桌面侧边栏 - 统一宽度 w-80 */}
      {lg && (
        <div className="fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-80 overflow-hidden">
          <FeatureSidebar className="h-full px-4 pt-4" />
        </div>
      )}

      {/* 主内容 - 统一左边距 pl-80 */}
      <div className={cn("flex-1 min-h-0 overflow-x-hidden", lg ? "pl-80" : "")}>
        <Outlet />
      </div>
    </section>
  );
};
```

**废弃宽度选项**（以下宽度不再使用）：
| 类 | 像素 | 状态 |
|:-------|:-----|:-----|
| `w-56` | 224px | ❌ 已废弃 |
| `w-64` | 256px | ❌ 已废弃 |
| `w-72` | 288px | ❌ 已废弃 |

**响应式断点**：
| 断点 | 宽度 | 行为 |
|:-----|:-----|:-----|
| `sm` | 640px | 导航栏出现 |
| `md` | 768px | 侧边栏变为固定 |
| `lg` | 1024px | 完整侧边栏宽度 |

---

## 页面组件

> **保鲜状态**: ✅ 已更新 (2026-02-12) | **覆盖范围**: `web/src/pages/*.tsx` | **最后检查**: v0.99.0

### 可用页面

| 路径 | 组件 | 布局 | 用途 |
|:-----|:-----|:-----|:-----|
| `/` | 重定向到 `/chat` | RootLayout | 默认入口 |
| `/auth/*` | 认证页面组 | RootLayout | 登录/注册/OAuth 回调 |
| `/memo` | `Home.tsx` | MemoLayout | 主时间线 + 笔记编辑器 |
| `/explore` | `Explore.tsx` | MemoLayout | 搜索和探索内容 |
| `/archived` | `Archived.tsx` | MemoLayout | 已归档笔记 |
| `/chat` | `AIChat.tsx` | AIChatLayout | AI 聊天界面（多模式） |
| `/schedule` | `Schedule.tsx` | ScheduleLayout | 日历视图 |
| `/knowledge-graph` | `KnowledgeGraph.tsx` | GeneralLayout | 知识图谱可视化 |
| `/inbox` | `Inboxes.tsx` | GeneralLayout | 收件箱 |
| `/attachments` | `Attachments.tsx` | GeneralLayout | 附件管理 |
| `/review` | `Review.tsx` | GeneralLayout | 每日回顾 |
| `/setting` | `Setting.tsx` | GeneralLayout | 用户设置 |
| `/u/:username` | `UserProfile.tsx` | MemoLayout | 公开用户资料 |
| `/memos/:uid` | `MemoDetail.tsx` | GeneralLayout | 笔记详情页 |
| `/m/:uid` | `MemoDetailRedirect` | GeneralLayout | 笔记详情重定向 |
| `/403` | `PermissionDenied.tsx` | GeneralLayout | 权限拒绝 |
| `/404` | `NotFound.tsx` | GeneralLayout | 404 页面 |

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
    └── zh-Hans.json  # 简体中文
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

### UnifiedMessageBlock (Warp Block 风格)

UnifiedMessageBlock 用于将用户输入 + AI 回复封装为一个统一的可折叠 Block：

```tsx
import { UnifiedMessageBlock } from "@/components/AIChat/UnifiedMessageBlock";

<UnifiedMessageBlock
  userMessage={userMsg}
  assistantMessage={assistantMsg}
  sessionSummary={summary}
  parrotId="GEEK"
  isLatest={true}
  isStreaming={false}
  onCopy={() => navigator.clipboard.writeText(content)}
  onRegenerate={() => regenerate()}
  onDelete={() => deleteMessage()}
/>
```

**功能**：
- Block Header: 用户消息预览 + 时间戳 + 状态徽章
- Block Body: 可折叠内容（思考/工具/结果/回答/会话统计）
- Block Footer: 操作栏（复制/重新生成/删除）
- 支持 5 种 Parrot 主题适配（MEMO/SCHEDULE/AMAZING/GEEK/EVOLUTION）+ AUTO 路由标记
- 自动折叠策略：新/最新 Block 展开，历史 Block 折叠

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

**详见**：[Chat Apps 用户指南](../user-guides/CHAT_APPS.md)

---

## 核心 Hooks

### AI 相关 Hooks（49KB）

| Hook | 大小 | 描述 |
|:-----|:-----|:-----|
| `useAIQueries` | 41KB | AI 查询管理（流式聊天） |
| `useBlockQueries` | 21KB | Block 模型支持（Unified Block Model） |
| `useParrotChat` | 8KB | 鹦鹉聊天 Hook |
| `useScheduleQueries` | 20KB | 日程查询 |
| `useBranchTree` | 5KB | 分支树管理（支持 Block 分支） |
| `useIntentPrediction` | - | 意图预测 |

### 其他核心 Hooks

| Hook | 描述 |
|:-----|:-----|
| `useUserQueries` | 用户查询（8KB） |
| `useMemoQueries` | 笔记查询（5KB） |
| `useAttachmentQueries` | 附件查询 |
| `useScheduleAgent` | 日程代理 |
| `useInstanceQueries` | 实例查询 |
| `useParrots` | 鹦鹉配置 |

---

## AI 聊天组件架构

### 组件结构（49+ 组件）

**核心组件**：
- `ChatMessages` (21KB) - 消息列表渲染
- `ChatInput` (12KB) - 输入框（支持快捷指令）
- `AIChatSidebar` - 会话侧边栏
- `UnifiedMessageBlock` (49KB) - 统一消息块
- `StreamingMarkdown` - 流式 Markdown 渲染

**Block 相关**：
- `BlockHeader` - Block 头部（状态/时间戳）
- `BlockBody` - Block 内容（可折叠）
- `BlockFooter` - Block 操作栏
- `BlockCostBadge` - 成本徽章
- `BlockEditDialog` - Block 编辑对话框
- `BlockStatusBadge` - 状态徽章

**Session 相关**：
- `SessionBar` - 会话栏
- `SessionSummaryPanel` - 会话摘要面板
- `SessionSwitcher` - 会话切换器

**工具展示**：
- `CompactToolCall` - 轻量级工具调用卡片
- `ToolCallsSection` - 工具调用区域
- `ThinkingSection` - 思考过程展示
- `EventBadge` - 事件类型徽章

**其他**：
- `QuickReplies` - 快捷回复
- `ModeSwitcher` - 模式切换（NORMAL/GEEK/EVOLUTION）
- `BranchIndicator` - 分支指示器
- `RegenerateButton` - 重新生成按钮

### 多模式主题支持

| 模式 | 主题色 | 用途 |
|:-----|:------|:-----|
| `NORMAL` | 默认蓝 | 普通模式（三层路由） |
| `GEEK` | 极客紫 | Geek Mode（Claude Code CLI） |
| `EVOLUTION` | 进化橙 | Evolution Mode（系统自我进化） |

主题通过 `PARROT_THEMES` 配置，支持动态切换。

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

### Block Hooks (Unified Block Model)

> **实现状态**: ✅ 完成 (Issue #71) | **文件**: `web/src/hooks/useBlockQueries.ts`

Block hooks 提供 AI 聊天对话持久化的 React Query 集成：

```tsx
import { useBlocks, useCreateBlock, useUpdateBlock } from "@/hooks/useBlockQueries";

// 获取会话的所有 Blocks
const { data: blocks, isLoading } = useBlocks(conversationId, { isActive: true });

// 创建新 Block
const createBlock = useCreateBlock();
createBlock.mutate({
  conversationId: 123,
  blockType: BlockType.MESSAGE,
  mode: BlockMode.NORMAL,
  userInputs: [{ content: "Hello", timestamp: Date.now() }],
});

// 更新 Block 状态
const updateBlock = useUpdateBlock();
updateBlock.mutate({
  id: BigInt(blockId),
  status: BlockStatus.COMPLETED,
  assistantContent: "Response here",
});
```

**可用 Hooks**：

| Hook | 描述 |
|:-----|:-----|
| `useBlocks(conversationId, filters, options)` | 获取会话 Blocks 列表 |
| `useBlock(id, options)` | 获取单个 Block 详情 |
| `useCreateBlock()` | 创建新 Block（乐观更新） |
| `useUpdateBlock()` | 更新 Block（支持流式状态） |
| `useDeleteBlock()` | 删除 Block |
| `useAppendUserInput()` | 追加用户输入 |
| `useAppendEvent()` | 追加流式事件 |
| `useStreamingBlock(blockId)` | 流式 Block 状态管理 |
| `usePrefetchBlock()` | 预加载 Block 数据 |
