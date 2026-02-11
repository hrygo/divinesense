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

| 错误       | 正确                    |
| :--------- | :---------------------- |
| `max-w-md` | `max-w-[28rem]` (448px) |
| `max-w-lg` | `max-w-[32rem]` (512px) |

---

## AI Provider 决策
- **推荐配置**：SiliconFlow + 智谱 Z.AI GLM
- **理由**：
  - SiliconFlow 提供向量 Embedding 和重排服务（国内稳定）
  - Z.AI GLM 提供 Claude 兼容对话服务（国内访问稳定）
  - 成本优化：Embedding 和 Rerank 使用同一家供应商减少开销
- **默认**：`.env.example` 和代码默认值均使用此配置

> **文件编辑**：连续 3 次 Edit 失败时，改用 `Read 完整文件 → Write 整体重写`。详见 @.claude/rules/file-editing.md
