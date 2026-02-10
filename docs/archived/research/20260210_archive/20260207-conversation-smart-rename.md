# 会话智能重命名 - 调研报告

> 调研时间: 2026-02-07 | 版本: v1.0

## 概述

实现会话智能重命名功能，当首轮对话结束后自动生成有意义的会话标题，支持手动触发智能重命名，用户可覆盖 AI 生成的标题。

## 需求来源

- 当前会话标题为 "New Chat" 或简单截取用户首条消息前 20 字符
- 无法反映对话主题，不利于会话列表查找

## 竞品分析

| 竞品          | 实现方式                       |
| :------------ | :----------------------------- |
| **ChatGPT**   | 首次响应后自动生成，可手动编辑 |
| **Claude**    | 自动生成但较简略，可手动编辑   |
| **LM Studio** | 提供 "Chat AI Naming" 配置项   |

### 最佳实践

1. **生成时机**: 首次 AI 响应完成后（有足够上下文）
2. **内容来源**: 用户首条消息 + AI 首次响应摘要
3. **语言检测**: 跟随用户输入语言
4. **Fallback**: LLM 失败时保留简单截取

## 技术可行性

| 维度         | 评级     | 说明                                      |
| :----------- | :------- | :---------------------------------------- |
| **后端支持** | ✅ 高     | `UpdateAIConversation` API 已存在         |
| **数据模型** | ✅ 小改动 | 新增 `title_source` 字段                  |
| **LLM 能力** | ✅ 已就绪 | 复用 `LLMIntentClassifier` 模型配置       |
| **前端基础** | ✅ 已有   | `generateSemanticTitle()` 可作为 fallback |

## 现有代码分析

### 前端现有实现

```typescript
// web/src/contexts/AIChatContext.tsx
function generateSemanticTitle(message: string): string | null {
  const cleaned = message.replace(/[#@\n\r]/g, " ").replace(/\s+/g, " ").trim();
  if (cleaned.length === 0) return null;
  if (cleaned.length <= 20) return cleaned;
  return cleaned.slice(0, 20) + "...";
}
```

当前仅截取首条消息前 20 字符，需升级为 LLM 智能生成。

### 意图识别模型配置

```go
// ai/agent/llm_intent_classifier.go
type LLMIntentConfig struct {
    APIKey  string
    BaseURL string // https://api.siliconflow.cn/v1
    Model   string // Qwen/Qwen2.5-7B-Instruct
}
```

标题生成将复用相同的模型配置，保持一致性和成本控制。

## 技术方案

### 数据库迁移

```sql
ALTER TABLE ai_conversation ADD COLUMN title_source VARCHAR(20) DEFAULT 'default';
-- 可选值: 'default' | 'auto' | 'user'
```

### TitleGenerator 服务

```go
type TitleGenerator struct {
    client *openai.Client  // 复用 LLMIntentClassifier 相同的 client
    model  string          // Qwen/Qwen2.5-7B-Instruct
    store  TitleStore
}

func (g *TitleGenerator) Generate(ctx context.Context, userMessage, assistantResponse string) (string, error)
```

### Prompt 设计

```
你是对话标题生成器。根据首轮对话生成简短标题。

规则:
1. 长度 5-15 字符
2. 使用与用户输入相同的语言
3. 提取核心话题
4. 不使用引号或特殊符号
5. 直接输出标题，不要解释
```

### 触发逻辑

1. **自动触发**: 首轮 AI 响应完成后，异步生成标题
2. **手动触发**: 用户点击「智能重命名」按钮
3. **条件检查**: 仅当 `title_source = 'default'` 时自动生成

### API 设计

```protobuf
rpc GenerateConversationTitle(GenerateTitleRequest) returns (GenerateTitleResponse) {
    option (google.api.http) = {
        post: "/api/v1/ai/conversations/{id}/generate-title"
    };
}
```

## 文件变更清单

| 文件                                              | 操作 | 说明                    |
| :------------------------------------------------ | :--- | :---------------------- |
| `store/migration/postgres/migrate/20260207*`      | 新建 | 迁移脚本                |
| `store/ai_conversation.go`                        | 修改 | 新增 TitleSource 字段   |
| `store/db/postgres/ai_conversation.go`            | 修改 | 读写 TitleSource        |
| `store/db/sqlite/ai_conversation.go`              | 修改 | 读写 TitleSource        |
| `proto/api/v1/ai_service.proto`                   | 修改 | 新增 RPC + title_source |
| `server/router/api/v1/ai/title_generator.go`      | 新建 | 标题生成服务            |
| `server/router/api/v1/ai_service_conversation.go` | 修改 | API Handler             |
| `server/router/api/v1/ai/handler.go`              | 修改 | 首轮完成触发            |
| `web/src/contexts/AIChatContext.tsx`              | 修改 | 新增 regenerateTitle    |
| `web/src/components/AIChat/ConversationItem.tsx`  | 修改 | 编辑按钮                |
| `web/src/components/AIChat/TitleEditDialog.tsx`   | 新建 | 编辑弹窗                |

## 复杂度评估

| 维度       | 评估                     |
| :--------- | :----------------------- |
| **工作量** | ~1 人周                  |
| **风险**   | 低（非核心路径，可降级） |
| **依赖项** | 无前置 Issue             |

## Token 成本

| 场景     | 输入 Tokens | 输出 Tokens | 成本                 |
| :------- | :---------- | :---------- | :------------------- |
| 每次生成 | ~80         | ~20         | ¥0.0002 (Qwen2.5-7B) |

## 风险与缓解

| 风险         | 影响 | 措施                       |
| :----------- | :--- | :------------------------- |
| LLM 响应慢   | 中   | 异步生成 + 前端 loading 态 |
| LLM 生成失败 | 低   | Fallback 到现有截取逻辑    |
| 标题内容不当 | 低   | 长度限制 + 特殊字符过滤    |

---

*Co-Authored-By: Claude <noreply@anthropic.com>*
