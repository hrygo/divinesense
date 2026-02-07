# P3-A001: 本地模型集成

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 5 人天  
> **负责团队**: 团队 A  
> **Sprint**: Sprint 5

---

## 1. 目标与背景

### 1.1 核心目标

集成本地 LLM（Ollama/llama.cpp），支持离线运行和隐私敏感场景，降低 API 成本 80%+。

### 1.2 用户价值

- 完全离线使用
- 数据不出本地
- API 成本归零（本地推理）

### 1.3 技术价值

- 降低云端依赖
- 支持私有部署
- 为模型路由（P3-A002）奠定基础

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A003: LLM 路由优化（路由基础）

### 2.2 后续依赖

- P3-A002: 模型路由器（本地/云端切换）

---

## 3. 功能设计

### 3.1 架构图

```
┌────────────────────────────────────────────────────────────┐
│                    本地模型集成架构                          │
├────────────────────────────────────────────────────────────┤
│                                                            │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              LocalLLMProvider                        │ │
│   │                                                      │ │
│   │  支持后端:                                           │ │
│   │  ├─ Ollama (推荐，易部署)                           │ │
│   │  ├─ llama.cpp (轻量级)                              │ │
│   │  └─ vLLM (高性能)                                   │ │
│   └─────────────────────────────────────────────────────┘ │
│                            │                               │
│                            ▼                               │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              推荐模型                                │ │
│   │                                                      │ │
│   │  • Qwen2.5-7B-Instruct (中文最优)                   │ │
│   │  • Llama-3.2-3B (轻量)                              │ │
│   │  • Mistral-7B (通用)                                │ │
│   └─────────────────────────────────────────────────────┘ │
│                                                            │
│   硬件要求: 8GB+ RAM (7B模型) | 16GB+ (13B模型)           │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口

```go
// plugin/ai/llm/local_provider.go

type LocalLLMProvider interface {
    LLMProvider
    
    // 检查本地模型是否可用
    IsAvailable(ctx context.Context) bool
    
    // 获取已安装模型列表
    ListModels(ctx context.Context) ([]LocalModel, error)
    
    // 拉取模型
    PullModel(ctx context.Context, modelName string) error
}

type LocalModel struct {
    Name      string `json:"name"`
    Size      int64  `json:"size"`
    Quantization string `json:"quantization"`  // q4_0, q8_0, f16
}
```

### 3.3 Ollama 集成

```go
// plugin/ai/llm/ollama.go

type OllamaProvider struct {
    baseURL string
    timeout time.Duration
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
    if baseURL == "" {
        baseURL = "http://localhost:11434"
    }
    return &OllamaProvider{
        baseURL: baseURL,
        timeout: 60 * time.Second,
    }
}

func (p *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    payload := map[string]any{
        "model":  req.Model,
        "prompt": req.Prompt,
        "stream": false,
        "options": map[string]any{
            "temperature": req.Temperature,
            "num_predict": req.MaxTokens,
        },
    }
    
    resp, err := p.post(ctx, "/api/generate", payload)
    if err != nil {
        return nil, err
    }
    
    return &CompletionResponse{
        Content: resp["response"].(string),
        Model:   req.Model,
        Usage: TokenUsage{
            PromptTokens:     resp["prompt_eval_count"].(int),
            CompletionTokens: resp["eval_count"].(int),
        },
    }, nil
}

func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
    resp, err := http.Get(p.baseURL + "/api/tags")
    return err == nil && resp.StatusCode == 200
}
```

### 3.4 配置

```yaml
# configs/ai.yaml
local_llm:
  enabled: true
  provider: "ollama"  # ollama, llamacpp, vllm
  
  ollama:
    base_url: "http://localhost:11434"
    default_model: "qwen2.5:7b"
    timeout: 60s
    
  models:
    chat: "qwen2.5:7b"
    embedding: "nomic-embed-text"
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1-2 | Ollama Provider 实现 |
| 3 | 模型管理（列表、拉取） |
| 4 | Embedding 支持 |
| 5 | 测试与文档 |

---

## 5. 验收标准

- [ ] Ollama 可用时自动检测
- [ ] 本地推理延迟 < 5s（7B模型）
- [ ] 支持 Qwen2.5-7B 中文对话

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
