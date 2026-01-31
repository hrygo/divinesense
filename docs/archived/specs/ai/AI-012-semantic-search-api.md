# AI-012: SemanticSearch API

## 概述

实现语义搜索 gRPC API，组合向量检索和重排序。

## 目标

提供基于语义的笔记搜索能力。

## 交付物

- `server/router/api/v1/ai_service.go` (新增)

## 实现规格

### API 实现

```go
package apiv1

import (
    "context"
    
    "github.com/usememos/memos/plugin/ai"
    apiv1 "github.com/usememos/memos/proto/gen/api/v1"
    "github.com/usememos/memos/store"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type AIService struct {
    apiv1.UnimplementedAIServiceServer
    
    Store            *store.Store
    EmbeddingService ai.EmbeddingService
    RerankerService  ai.RerankerService
    LLMService       ai.LLMService
}

func (s *AIService) IsEnabled() bool {
    return s.EmbeddingService != nil
}

func (s *AIService) SemanticSearch(ctx context.Context, req *apiv1.SemanticSearchRequest) (*apiv1.SemanticSearchResponse, error) {
    if !s.IsEnabled() {
        return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
    }

    // 1. 获取当前用户
    user, err := getCurrentUser(ctx, s.Store)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
    }
    
    // 2. 参数校验
    if req.Query == "" {
        return nil, status.Errorf(codes.InvalidArgument, "query is required")
    }
    
    limit := int(req.Limit)
    if limit <= 0 {
        limit = 10
    }
    if limit > 50 {
        limit = 50
    }
    
    // 3. 向量化查询
    queryVector, err := s.EmbeddingService.Embed(ctx, req.Query)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to embed query: %v", err)
    }
    
    // 4. 向量检索 (Top 10, 优化 for 2C2G)
    results, err := s.Store.SearchMemosByVector(ctx, &store.VectorSearchOptions{
        UserID: user.ID,
        Vector: queryVector,
        Limit:  10,  // 减少初始召回量
    })
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
    }
    
    // 过滤低相关性结果 (Threshold: 0.5)
    var filteredResults []*store.MemoWithScore
    for _, r := range results {
        if r.Score >= 0.5 {
            filteredResults = append(filteredResults, r)
        }
    }
    results = filteredResults
    
    if len(results) == 0 {
        return &apiv1.SemanticSearchResponse{Results: []*apiv1.SearchResult{}}, nil
    }
    
    // 5. 重排序 (可选)
    if s.RerankerService.IsEnabled() && len(results) > limit {
        documents := make([]string, len(results))
        for i, r := range results {
            documents[i] = r.Memo.Content
        }
        
        rerankResults, err := s.RerankerService.Rerank(ctx, req.Query, documents, limit)
        if err == nil {
            // 按重排序结果重新排列
            reordered := make([]*store.MemoWithScore, len(rerankResults))
            for i, rr := range rerankResults {
                reordered[i] = results[rr.Index]
                reordered[i].Score = rr.Score
            }
            results = reordered
        }
    }
    
    // 6. 截取结果
    if len(results) > limit {
        results = results[:limit]
    }
    
    // 7. 构建响应
    response := &apiv1.SemanticSearchResponse{
        Results: make([]*apiv1.SearchResult, len(results)),
    }
    
    for i, r := range results {
        snippet := r.Memo.Content
        if len(snippet) > 200 {
            snippet = snippet[:200] + "..."
        }
        
        response.Results[i] = &apiv1.SearchResult{
            Name:    fmt.Sprintf("memos/%s", r.Memo.UID),
            Snippet: snippet,
            Score:   r.Score,
        }
    }
    
    return response, nil
}
```

### 权限配置

在 `acl_config.go` 中确保 AI 接口需要认证。

## 验收标准

### AC-1: 文件创建
- [x] `server/router/api/v1/ai_service.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./server/...` 无错误

### AC-3: 认证检查
- [x] 未登录时返回 Unauthenticated

### AC-4: 参数校验
- [x] 空 query 返回 InvalidArgument
- [x] limit 自动限制在 1-50

### AC-5: 搜索结果
- [x] 返回语义相关的 Memo
- [x] 只返回当前用户的 Memo
- [x] Score 在 0-1 范围

### AC-6: Rerank 效果
- [x] 启用 Rerank 时结果更精准
- [x] Rerank 失败时降级到原始排序

## 实现状态

✅ **已完成** - 实现于 [server/router/api/v1/ai_service.go:51-148](../../server/router/api/v1/ai_service.go#L51-L148)

## 测试命令

```bash
# API 测试
curl -X POST http://localhost:8081/api/v1/ai/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "测试搜索", "limit": 10}'
```

## 依赖

- AI-001 (Proto 定义)
- AI-006 (PostgreSQL 向量搜索)
- AI-008 (Embedding 服务)
- AI-009 (Reranker 服务)

## 预估时间

2 小时
