# AI-014: SuggestTags API

## 概述

实现标签推荐 API，基于内容自动推荐合适的标签。

## 目标

减少手动标签整理工作。

## 交付物

- `server/router/api/v1/ai_service.go` (扩展 SuggestTags 方法)

## 实现规格

### API 实现

```go
func (s *AIService) SuggestTags(ctx context.Context, req *apiv1.SuggestTagsRequest) (*apiv1.SuggestTagsResponse, error) {
    // 1. 获取当前用户
    user, err := getCurrentUser(ctx, s.Store)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
    }
    
    // 2. 参数校验
    if req.Content == "" {
        return nil, status.Errorf(codes.InvalidArgument, "content is required")
    }
    
    limit := int(req.Limit)
    if limit <= 0 {
        limit = 5
    }
    if limit > 10 {
        limit = 10
    }
    
    // 3. 获取用户已有标签（作为参考）
    existingTags, err := s.getExistingTags(ctx, user.ID)
    if err != nil {
        // 非关键错误，继续
        existingTags = []string{}
    }
    
    // 4. 构建 Prompt
    prompt := fmt.Sprintf(`请为以下内容推荐 %d 个合适的标签。

## 内容
%s

## 已有标签（参考）
%s

## 要求
1. 每个标签不超过10个字符
2. 标签要准确反映内容主题
3. 优先使用已有标签列表中的标签
4. 只返回标签列表，每行一个，不要其他内容
`, limit, req.Content, strings.Join(existingTags, ", "))
    
    messages := []ai.Message{
        {Role: "user", Content: prompt},
    }
    
    // 5. 调用 LLM
    response, err := s.LLMService.Chat(ctx, messages)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to generate tags: %v", err)
    }
    
    // 6. 解析结果
    tags := parseTags(response, limit)
    
    return &apiv1.SuggestTagsResponse{Tags: tags}, nil
}

func (s *AIService) getExistingTags(ctx context.Context, userID int32) ([]string, error) {
    // 查询用户所有 Memo 中的标签
    // 实现略
    return nil, nil
}

func parseTags(response string, limit int) []string {
    lines := strings.Split(response, "\n")
    var tags []string
    
    for _, line := range lines {
        tag := strings.TrimSpace(line)
        tag = strings.TrimPrefix(tag, "-")
        tag = strings.TrimPrefix(tag, "#")
        tag = strings.TrimSpace(tag)
        
        if tag != "" && len(tag) <= 20 {
            tags = append(tags, tag)
            if len(tags) >= limit {
                break
            }
        }
    }
    
    return tags
}
```

## 验收标准

### AC-1: 编译通过
- [x] `go build ./server/...` 无错误

### AC-2: 认证检查
- [x] 未登录时返回 Unauthenticated

### AC-3: 参数校验
- [x] 空 content 返回 InvalidArgument
- [x] limit 限制在 1-10

### AC-4: 标签推荐
- [x] 返回相关标签
- [x] 标签数量符合 limit

### AC-5: 标签格式
- [x] 标签不包含 # 前缀
- [x] 标签长度合理

## 实现状态

✅ **已完成** - 实现于 [server/router/api/v1/ai_service.go:150-259](../../server/router/api/v1/ai_service.go#L150-L259)

## 测试命令

```bash
curl -X POST http://localhost:8081/api/v1/ai/suggest-tags \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "今天学习了 Go 语言的并发编程", "limit": 5}'
```

## 依赖

- AI-010 (LLM 服务)

## 预估时间

1 小时
