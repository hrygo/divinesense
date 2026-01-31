# 代码风格规范

## Go

- 文件命名：`snake_case.go`
- 日志：使用 `log/slog` 结构化日志
- 遵循标准 Go 项目布局
- 错误处理：始终检查并处理错误
- 注释：导出函数必须有文档注释

## React/TypeScript

- 组件：PascalCase 命名 (`UserProfile.tsx`)
- Hooks：`use` 前缀 (`useUserData()`)
- 样式：使用 Tailwind CSS 类名
- 状态管理：优先使用 React Context
- 类型：避免 `any`，使用具体类型

## Tailwind CSS 4 — 关键陷阱

> **切勿使用语义化 `max-w-sm/md/lg/xl`** —— 在 Tailwind v4 中它们解析为约 16px。
>
> **请使用显式值**：`max-w-[24rem]`, `max-w-[28rem]` 等

详见 `docs/dev-guides/FRONTEND.md` 中的完整说明。

## AI 路由

- 后端 `ChatRouter` 处理意图分类
- 位置：`plugin/ai/agent/chat_router.go`
- 规则匹配 (0ms) → LLM 降级 (~400ms)
- 路由到：MEMO / SCHEDULE / AMAZING 代理
