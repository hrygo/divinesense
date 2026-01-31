# AI-007: AI 插件配置

## 概述

创建 AI 插件的配置解析模块，从 Profile 读取并验证配置。

## 目标

提供类型安全的 AI 配置访问。

## 交付物

- `plugin/ai/config.go` (新增)

## 实现规格

### 配置结构

```go
package ai

import (
    "errors"
    
    "github.com/usememos/memos/internal/profile"
)

// Config AI 配置
type Config struct {
    Enabled bool
    
    Embedding EmbeddingConfig
    Reranker  RerankerConfig
    LLM       LLMConfig
}

// EmbeddingConfig 向量化配置
type EmbeddingConfig struct {
    Provider   string  // siliconflow, openai, ollama
    Model      string  // BAAI/bge-m3
    Dimensions int     // 1024
    APIKey     string
    BaseURL    string
}

// RerankerConfig 重排序配置
type RerankerConfig struct {
    Enabled  bool
    Provider string  // siliconflow, cohere
    Model    string  // BAAI/bge-reranker-v2-m3
    APIKey   string
    BaseURL  string
}

// LLMConfig 大模型配置
type LLMConfig struct {
    Provider    string  // deepseek, openai, ollama
    Model       string  // deepseek-chat
    APIKey      string
    BaseURL     string
    MaxTokens   int     // 默认 2048
    Temperature float32 // 默认 0.7
}

// NewConfigFromProfile 从 Profile 创建配置
func NewConfigFromProfile(p *profile.Profile) *Config {
    cfg := &Config{
        Enabled: p.AIEnabled,
    }
    
    if !cfg.Enabled {
        return cfg
    }
    
    // Embedding 配置
    cfg.Embedding = EmbeddingConfig{
        Provider:   p.AIEmbeddingProvider,
        Model:      p.AIEmbeddingModel,
        Dimensions: 1024,
    }
    
    switch p.AIEmbeddingProvider {
    case "siliconflow":
        cfg.Embedding.APIKey = p.AISiliconFlowAPIKey
        cfg.Embedding.BaseURL = p.AISiliconFlowBaseURL
    case "openai":
        cfg.Embedding.APIKey = p.AIOpenAIAPIKey
        cfg.Embedding.BaseURL = "https://api.openai.com/v1"
    case "ollama":
        cfg.Embedding.BaseURL = p.AIOllamaBaseURL
    }
    
    // Reranker 配置
    cfg.Reranker = RerankerConfig{
        Enabled:  p.AISiliconFlowAPIKey != "",
        Provider: "siliconflow",
        Model:    p.AIRerankModel,
        APIKey:   p.AISiliconFlowAPIKey,
        BaseURL:  p.AISiliconFlowBaseURL,
    }
    
    // LLM 配置
    cfg.LLM = LLMConfig{
        Provider:    p.AILLMProvider,
        Model:       p.AILLMModel,
        MaxTokens:   2048,
        Temperature: 0.7,
    }
    
    switch p.AILLMProvider {
    case "deepseek":
        cfg.LLM.APIKey = p.AIDeepSeekAPIKey
        cfg.LLM.BaseURL = p.AIDeepSeekBaseURL
    case "openai":
        cfg.LLM.APIKey = p.AIOpenAIAPIKey
        cfg.LLM.BaseURL = "https://api.openai.com/v1"
    case "ollama":
        cfg.LLM.BaseURL = p.AIOllamaBaseURL
    }
    
    return cfg
}

// Validate 验证配置
func (c *Config) Validate() error {
    if !c.Enabled {
        return nil
    }
    
    if c.Embedding.Provider == "" {
        return errors.New("embedding provider is required")
    }
    
    if c.Embedding.Provider != "ollama" && c.Embedding.APIKey == "" {
        return errors.New("embedding API key is required")
    }
    
    if c.LLM.Provider == "" {
        return errors.New("LLM provider is required")
    }
    
    if c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
        return errors.New("LLM API key is required")
    }
    
    return nil
}
```

## 验收标准

### AC-1: 文件创建
- [x] `plugin/ai/config.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./plugin/ai/...` 无错误

### AC-3: 配置解析正确
- [x] SiliconFlow 配置正确解析
- [x] DeepSeek 配置正确解析
- [x] OpenAI 配置正确解析
- [x] Ollama 配置正确解析

### AC-4: 验证逻辑正确
- [x] `Enabled=false` 时跳过验证
- [x] 缺少 API Key 时返回错误 (Ollama 除外)

## 实现状态

✅ **已完成** - 实现于 [plugin/ai/config.go](../../plugin/ai/config.go)

## 测试命令

```bash
go build ./plugin/ai/...
go test ./plugin/ai/... -v
```

## 依赖

- AI-002 (Profile 配置)

## 预估时间

1 小时
