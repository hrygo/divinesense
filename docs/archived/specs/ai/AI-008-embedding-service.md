# AI-008: Embedding 服务

## 概述

实现向量化服务，支持多供应商切换 (SiliconFlow, OpenAI, Ollama)。

## 目标

提供文本向量化能力，使用 langchaingo 统一接口。

## 交付物

- `plugin/ai/embedding.go` (新增)

## 实现规格

### 接口定义

```go
package ai

import "context"

// EmbeddingService 向量化服务
type EmbeddingService interface {
    // Embed 生成单个文本的向量
    Embed(ctx context.Context, text string) ([]float32, error)
    
    // EmbedBatch 批量生成向量
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    
    // Dimensions 返回向量维度
    Dimensions() int
}
```

### 实现

```go
package ai

import (
    "context"
    
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/ollama"
    "github.com/tmc/langchaingo/llms/openai"
)

type embeddingService struct {
    embedder   embeddings.Embedder
    dimensions int
}

// NewEmbeddingService 创建 Embedding 服务
func NewEmbeddingService(cfg *EmbeddingConfig) (EmbeddingService, error) {
    var embedder embeddings.Embedder
    
    switch cfg.Provider {
    case "siliconflow":
        // SiliconFlow 兼容 OpenAI API
        llm, createErr := openai.New(
            openai.WithToken(cfg.APIKey),
            openai.WithBaseURL(cfg.BaseURL),
            openai.WithEmbeddingModel(cfg.Model),
        )
        if createErr != nil {
            return nil, createErr
        }
        var embedErr error
        embedder, embedErr = embeddings.NewEmbedder(llm)
        if embedErr != nil {
            return nil, embedErr
        }
        
    case "openai":
        llm, createErr := openai.New(
            openai.WithToken(cfg.APIKey),
            openai.WithEmbeddingModel(cfg.Model),
        )
        if createErr != nil {
            return nil, createErr
        }
        var embedErr error
        embedder, embedErr = embeddings.NewEmbedder(llm)
        if embedErr != nil {
            return nil, embedErr
        }
        
    case "ollama":
        // 使用 langchaingo ollama 支持
        llm, err := ollama.New(
            ollama.WithModel(cfg.Model),
            ollama.WithServerURL(cfg.BaseURL),
        )
        if err != nil {
            return nil, err
        }
        embedder, err = embeddings.NewEmbedder(llm)
        if err != nil {
            return nil, err
        }
        
    default:
        return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
    }
    
    return &embeddingService{
        embedder:   embedder,
        dimensions: cfg.Dimensions,
    }, nil
}

func (s *embeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
    vectors, err := s.embedder.EmbedDocuments(ctx, []string{text})
    if err != nil {
        return nil, err
    }
    if len(vectors) == 0 {
        return nil, errors.New("empty embedding result")
    }
    return toFloat32(vectors[0]), nil
}

func (s *embeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    vectors, err := s.embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        return nil, err
    }
    result := make([][]float32, len(vectors))
    for i, v := range vectors {
        result[i] = toFloat32(v)
    }
    return result, nil
}

func (s *embeddingService) Dimensions() int {
    return s.dimensions
}

func toFloat32(v []float64) []float32 {
    result := make([]float32, len(v))
    for i, f := range v {
        result[i] = float32(f)
    }
    return result
}
```

## 验收标准

### AC-1: 文件创建
- [x] `plugin/ai/embedding.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./plugin/ai/...` 无错误

### AC-3: SiliconFlow 集成测试
- [x] 能成功调用 SiliconFlow bge-m3 API
- [x] 返回 1024 维向量
- [x] 批量处理正常工作

### AC-4: 接口一致性
- [x] 切换到 OpenAI 后功能正常
- [x] 切换到 Ollama 后功能正常 (需本地运行)

## 实现状态

✅ **已完成** - 实现于 [plugin/ai/embedding.go](../../plugin/ai/embedding.go)

## 测试命令

```bash
# 单元测试 (Mock)
go test ./plugin/ai/... -run TestEmbedding -v

# 集成测试 (需要 API Key)
MEMOS_AI_SILICONFLOW_API_KEY=xxx go test ./plugin/ai/... -run TestEmbeddingIntegration -v
```

## 依赖

- AI-007 (AI 插件配置)
- `go get github.com/tmc/langchaingo`

## 预估时间

2 小时
