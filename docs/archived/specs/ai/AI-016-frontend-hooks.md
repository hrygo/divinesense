# AI-016: 前端 AI Hooks

## 概述

实现前端 React Query Hooks，封装 AI 服务调用。

## 目标

为前端组件提供类型安全的 AI API 封装。

## 交付物

- `web/src/hooks/useAIQueries.ts` (新增)

## 实现规格

```typescript
import { useQuery, useMutation } from "@tanstack/react-query";
import { aiServiceClient } from "@/grpcweb";

// 语义搜索
export function useSemanticSearch(query: string, enabled = true) {
  return useQuery({
    queryKey: ["ai", "search", query],
    queryFn: () => aiServiceClient.semanticSearch({ query, limit: 10 }),
    enabled: enabled && query.length > 2,
    staleTime: 60 * 1000,
  });
}

// 标签推荐
export function useSuggestTags() {
  return useMutation({
    mutationFn: (content: string) =>
      aiServiceClient.suggestTags({ content, limit: 5 }),
  });
}

// 相关笔记
export function useRelatedMemos(memoName: string, enabled = true) {
  return useQuery({
    queryKey: ["ai", "related", memoName],
    queryFn: () => aiServiceClient.getRelatedMemos({ name: memoName, limit: 5 }),
    enabled: enabled && !!memoName,
    staleTime: 5 * 60 * 1000,
  });
}

// AI 对话 (需特殊处理流式响应)
export function useChat() {
  // 流式实现，需要使用 grpc-web 流式 API
}
```

## 验收标准

### AC-1: 文件创建
- [x] `web/src/hooks/useAIQueries.ts` 文件存在

### AC-2: 类型正确
- [x] TypeScript 编译无错误
- [x] 返回类型与 Proto 定义一致

### AC-3: Hook 功能
- [x] `useSemanticSearch` 正常工作
- [x] `useSuggestTags` 正常工作
- [x] `useRelatedMemos` 正常工作

### AC-4: 缓存策略
- [x] 搜索结果适当缓存 (1分钟)
- [x] 相关笔记长时间缓存 (5分钟)

## 实现状态

✅ **已完成** - 实现于 [web/src/hooks/useAIQueries.ts](../../web/src/hooks/useAIQueries.ts)

**实现细节**:
- `useChat` 使用 Connect RPC 异步迭代器实现流式接收
- 支持 `onContent`, `onSources`, `onDone`, `onError` 回调
- 返回完整内容 + 来源列表

**使用示例**:
```typescript
const { stream } = useChat();

await stream(
  { message: "我最近记录了什么?" },
  {
    onContent: (chunk) => console.log("收到:", chunk),
    onSources: (sources) => console.log("来源:", sources),
    onDone: () => console.log("完成"),
    onError: (err) => console.error("错误:", err),
  }
);
```

## 测试命令

```bash
cd web && npm run build
```

## 依赖

- AI-012, AI-013, AI-014, AI-015 (后端 API)

## 预估时间

2 小时
