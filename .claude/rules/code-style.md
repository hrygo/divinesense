# 代码风格规范

## Go
- 文件：`snake_case.go`
- 日志：`log/slog` 结构化
- 错误：始终检查并处理

## React/TypeScript
- 组件：`PascalCase.tsx`
- Hooks：`use` 前缀
- 样式：Tailwind CSS
- i18n：使用 `t("key")`

## Tailwind CSS 4
> **切勿使用 `max-w-sm/md/lg/xl`** —— 解析为 ~16px

| 错误 | 正确 |
|:-----|:-----|
| `max-w-md` | `max-w-[28rem]` (448px) |
| `max-w-lg` | `max-w-[32rem]` (512px) |

## AI 路由
- 三层路由：规则 → 历史 → LLM
- 位置：`ai/agents/chat_router.go`
