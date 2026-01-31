# AI-009: Reranker 服务

## 概述

实现重排序服务，调用 SiliconFlow bge-reranker-v2-m3 模型。

## 目标

提升语义搜索结果的精准度。

## 交付物

- `plugin/ai/reranker.go` (新增)

## 实现规格

### 接口定义

```go
package ai

import "context"

// RerankResult 重排序结果
type RerankResult struct {
    Index int     // 原始索引
    Score float32 // 相关性分数
}

// RerankerService 重排序服务
type RerankerService interface {
    // Rerank 对文档进行重排序
    Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
    
    // IsEnabled 是否启用
    IsEnabled() bool
}
```

### 实现

```go
package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "sort"
)

type rerankerService struct {
    enabled bool
    apiKey  string
    baseURL string
    model   string
    client  *http.Client
}

// NewRerankerService 创建 Reranker 服务
func NewRerankerService(cfg *RerankerConfig) RerankerService {
    return &rerankerService{
        enabled: cfg.Enabled,
        apiKey:  cfg.APIKey,
        baseURL: cfg.BaseURL,
        model:   cfg.Model,
        client:  &http.Client{},
    }
}

func (s *rerankerService) IsEnabled() bool {
    return s.enabled
}

func (s *rerankerService) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
    if !s.enabled {
        // 未启用时返回原始顺序
        results := make([]RerankResult, len(documents))
        for i := range documents {
            results[i] = RerankResult{Index: i, Score: 1.0 - float32(i)*0.01}
        }
        return results[:min(topN, len(results))], nil
    }
    
    // 调用 SiliconFlow Rerank API
    reqBody := map[string]interface{}{
        "model":     s.model,
        "query":     query,
        "documents": documents,
        "top_n":     topN,
    }
    
    body, _ := json.Marshal(reqBody)
    // SiliconFlow API endpoint revision: /v1/rerank
    req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/rerank", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := s.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("rerank API error: %s", string(body))
    }
    
    var result struct {
        Results []struct {
            Index int     `json:"index"`
            Score float32 `json:"relevance_score"`
        } `json:"results"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    results := make([]RerankResult, len(result.Results))
    for i, r := range result.Results {
        results[i] = RerankResult{Index: r.Index, Score: r.Score}
    }
    
    // 按分数降序排序
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    return results, nil
}
```

## 验收标准

### AC-1: 文件创建
- [x] `plugin/ai/reranker.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./plugin/ai/...` 无错误

### AC-3: 降级逻辑
- [x] `IsEnabled()=false` 时返回原始顺序
- [x] 不调用外部 API

### AC-4: SiliconFlow 集成测试
- [x] 能成功调用 SiliconFlow Rerank API
- [x] 返回按相关性排序的结果
- [x] topN 参数生效

### AC-5: 错误处理
- [x] API 错误时返回明确错误信息
- [x] 网络超时时正确处理

## 实现状态

✅ **已完成** - 实现于 [plugin/ai/reranker.go](../../plugin/ai/reranker.go)

## 测试命令

```bash
# 单元测试
go test ./plugin/ai/... -run TestReranker -v

# 集成测试
MEMOS_AI_SILICONFLOW_API_KEY=xxx go test ./plugin/ai/... -run TestRerankerIntegration -v
```

## 依赖

- AI-007 (AI 插件配置)

## 预估时间

1.5 小时
