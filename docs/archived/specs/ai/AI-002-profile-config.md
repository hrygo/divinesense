# AI-002: Profile 配置扩展

## 概述

扩展 Profile 结构体，添加 AI 相关配置字段，支持通过环境变量配置。

## 目标

让 AI 功能可通过环境变量启用和配置，支持多供应商切换。

## 交付物

- `internal/profile/profile.go` (修改)

## 实现规格

### 新增字段

```go
type Profile struct {
    // ... 现有字段 ...

    // AI 配置
    AIEnabled            bool   // MEMOS_AI_ENABLED
    AIEmbeddingProvider  string // MEMOS_AI_EMBEDDING_PROVIDER (默认: siliconflow)
    AILLMProvider        string // MEMOS_AI_LLM_PROVIDER (默认: deepseek)
    AISiliconFlowAPIKey  string // MEMOS_AI_SILICONFLOW_API_KEY
    AISiliconFlowBaseURL string // MEMOS_AI_SILICONFLOW_BASE_URL (默认: https://api.siliconflow.cn/v1)
    AIDeepSeekAPIKey     string // MEMOS_AI_DEEPSEEK_API_KEY
    AIDeepSeekBaseURL    string // MEMOS_AI_DEEPSEEK_BASE_URL (默认: https://api.deepseek.com)
    AIOpenAIAPIKey       string // MEMOS_AI_OPENAI_API_KEY
    AIOllamaBaseURL      string // MEMOS_AI_OLLAMA_BASE_URL (默认: http://localhost:11434)
    AIEmbeddingModel     string // MEMOS_AI_EMBEDDING_MODEL (默认: BAAI/bge-m3)
    AIRerankModel        string // MEMOS_AI_RERANK_MODEL (默认: BAAI/bge-reranker-v2-m3)
    AILLMModel           string // MEMOS_AI_LLM_MODEL (默认: deepseek-chat)
}
```

### 辅助方法

```go
func (p *Profile) IsAIEnabled() bool {
    return p.AIEnabled && (p.AISiliconFlowAPIKey != "" || p.AIOpenAIAPIKey != "" || p.AIOllamaBaseURL != "")
}
```

### 环境变量读取

在 `cmd/memos/main.go` 或配置加载处添加：

```go
AIEnabled:            os.Getenv("MEMOS_AI_ENABLED") == "true",
AIEmbeddingProvider:  getEnvOrDefault("MEMOS_AI_EMBEDDING_PROVIDER", "siliconflow"),
AILLMProvider:        getEnvOrDefault("MEMOS_AI_LLM_PROVIDER", "deepseek"),
AISiliconFlowAPIKey:  os.Getenv("MEMOS_AI_SILICONFLOW_API_KEY"),
AISiliconFlowBaseURL: getEnvOrDefault("MEMOS_AI_SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1"),
AIDeepSeekAPIKey:     os.Getenv("MEMOS_AI_DEEPSEEK_API_KEY"),
AIDeepSeekBaseURL:    getEnvOrDefault("MEMOS_AI_DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
AIOpenAIAPIKey:       os.Getenv("MEMOS_AI_OPENAI_API_KEY"),
AIOllamaBaseURL:      getEnvOrDefault("MEMOS_AI_OLLAMA_BASE_URL", "http://localhost:11434"),
AIEmbeddingModel:     getEnvOrDefault("MEMOS_AI_EMBEDDING_MODEL", "BAAI/bge-m3"),
AIRerankModel:        getEnvOrDefault("MEMOS_AI_RERANK_MODEL", "BAAI/bge-reranker-v2-m3"),
AILLMModel:           getEnvOrDefault("MEMOS_AI_LLM_MODEL", "deepseek-chat"),
```

## 验收标准

### AC-1: 编译通过
- [x] `go build ./...` 无错误

### AC-2: 默认值正确
- [x] 未设置任何 AI 环境变量时，`AIEnabled = false`
- [x] `AIEmbeddingProvider` 默认值为 `siliconflow`
- [x] `AILLMProvider` 默认值为 `deepseek`

### AC-3: 环境变量生效
- [x] 设置 `MEMOS_AI_ENABLED=true` 后，`profile.AIEnabled == true`
- [x] 设置 `MEMOS_AI_SILICONFLOW_API_KEY=xxx` 后，能正确读取

### AC-4: IsAIEnabled 逻辑
- [x] `AIEnabled=true` 且有 API Key 时，`IsAIEnabled()` 返回 true
- [x] `AIEnabled=false` 时，`IsAIEnabled()` 返回 false
- [x] `AIEnabled=true` 但无任何 API Key 时，`IsAIEnabled()` 返回 false

## 实现状态

✅ **已完成** - 实现于 [internal/profile/profile.go](../../internal/profile/profile.go)

## 测试命令

```bash
go build ./...
MEMOS_AI_ENABLED=true MEMOS_AI_SILICONFLOW_API_KEY=test go test ./internal/profile/... -v
```

## 依赖

无

## 预估时间

1 小时
