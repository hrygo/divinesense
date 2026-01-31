# AI-013: Chat API

## 概述

实现 RAG 对话 API，基于用户笔记回答问题。

## 目标

提供上下文感知的 AI 对话能力。

## 交付物

- `server/router/api/v1/ai_service.go` (扩展 Chat 方法)

## 实现规格

### API 实现

```go
func (s *AIService) Chat(req *apiv1.ChatRequest, stream apiv1.AIService_ChatServer) error {
    ctx := stream.Context()

    if !s.IsEnabled() {
        return status.Errorf(codes.Unavailable, "AI features are disabled")
    }
    
    // 1. 获取当前用户
    user, err := getCurrentUser(ctx, s.Store)
    if err != nil {
        return status.Errorf(codes.Unauthenticated, "unauthorized")
    }
    
    // 2. 参数校验
    if req.Message == "" {
        return status.Errorf(codes.InvalidArgument, "message is required")
    }
    
    // 3. 语义检索相关笔记 (Top 5, Score > 0.5)
    queryVector, err := s.EmbeddingService.Embed(ctx, req.Message)
    if err != nil {
        return status.Errorf(codes.Internal, "failed to embed query: %v", err)
    }
    
    results, err := s.Store.SearchMemosByVector(ctx, &store.VectorSearchOptions{
        UserID: user.ID,
        Vector: queryVector,
        Limit:  5,
    })
    if err != nil {
        return status.Errorf(codes.Internal, "failed to search: %v", err)
    }
    
    // 4. 构建上下文 (最大字符数限制: 3000)
    var contextBuilder strings.Builder
    var sources []string
    totalChars := 0
    maxChars := 3000
    
    for i, r := range results {
        if r.Score < 0.5 { continue } // 忽略低相关性
    
        content := r.Memo.Content
        if totalChars + len(content) > maxChars {
            break // 停止添加上下文
        }
    
        contextBuilder.WriteString(fmt.Sprintf("### 笔记 %d\n%s\n\n", i+1, content))
        sources = append(sources, fmt.Sprintf("memos/%s", r.Memo.UID))
        totalChars += len(content)
    }
    
    // 5. 构建 Prompt
    messages := []ai.Message{
        {
            Role: "system",
            Content: `你是一个基于用户个人笔记的AI助手。请根据以下笔记内容回答问题。
如果笔记中没有相关信息，请明确告知用户。
回答时使用中文，保持简洁准确。`,
        },
    }
    
    // 添加历史对话
    for i := 0; i < len(req.History)-1; i += 2 {
        if i+1 < len(req.History) {
            messages = append(messages, ai.Message{Role: "user", Content: req.History[i]})
            messages = append(messages, ai.Message{Role: "assistant", Content: req.History[i+1]})
        }
    }
    
    // 添加当前问题
    userMessage := fmt.Sprintf("## 相关笔记\n%s\n## 用户问题\n%s", contextBuilder.String(), req.Message)
    messages = append(messages, ai.Message{Role: "user", Content: userMessage})
    
    // 6. 流式调用 LLM
    contentChan, errChan := s.LLMService.ChatStream(ctx, messages)
    
    // 先发送来源信息
    if err := stream.Send(&apiv1.ChatResponse{
        Sources: sources,
    }); err != nil {
        return err
    }
    
    // 流式发送内容
    for {
        select {
        case content, ok := <-contentChan:
            if !ok {
                contentChan = nil // 标记为已关闭
                if errChan == nil {
                    return stream.Send(&apiv1.ChatResponse{Done: true})
                }
                continue
            }
            if err := stream.Send(&apiv1.ChatResponse{
                Content: content,
            }); err != nil {
                return err
            }
            
        case err, ok := <-errChan:
            if !ok {
                errChan = nil // 标记为已关闭
                if contentChan == nil {
                    return stream.Send(&apiv1.ChatResponse{Done: true})
                }
                continue
            }
            if err != nil {
                return status.Errorf(codes.Internal, "LLM error: %v", err)
            }
            
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

## 验收标准

### AC-1: 编译通过
- [x] `go build ./server/...` 无错误

### AC-2: 认证检查
- [x] 未登录时返回 Unauthenticated

### AC-3: RAG 流程
- [x] 正确检索相关笔记
- [x] 笔记内容注入到 Prompt

### AC-4: 流式响应
- [x] 首先返回 sources
- [x] 内容分块流式返回
- [x] 最后发送 done=true

### AC-5: 对话历史
- [x] history 参数正确传递
- [x] 多轮对话上下文连贯

### AC-6: 错误处理
- [x] LLM 错误正确传递
- [x] 连接中断时正确处理

## 实现状态

✅ **已完成** - 实现于 [server/router/api/v1/ai_service.go:261-378](../../server/router/api/v1/ai_service.go#L261-L378)

## 测试命令

```bash
# gRPC 流式测试 (需要 grpcurl)
grpcurl -d '{"message": "我最近记录了什么?"}' \
  -H "Authorization: Bearer $TOKEN" \
  localhost:28081 memos.api.v1.AIService/Chat
```

## 依赖

- AI-012 (SemanticSearch API)
- AI-010 (LLM 服务)

## 预估时间

2 小时
