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

- 后端 `ChatRouter` 处理意图分类
- 位置：`plugin/ai/agent/chat_router.go`
- 规则匹配 (0ms) → LLM 降级 (~400ms)
- 路由到：MEMO / SCHEDULE / AMAZING 代理

---

> **文件编辑**：连续 3 次 Edit 失败时，改用 `Read 完整文件 → Write 整体重写`。详见 @.claude/rules/file-editing.md
