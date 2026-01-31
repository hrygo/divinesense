# AI-010: LLM 服务

## 概述

实现大语言模型服务，支持 DeepSeek, OpenAI, Ollama 切换。

## 目标

提供统一的 LLM 调用接口，支持同步和流式响应。

## 交付物

- `plugin/ai/llm.go` (新增)

## 实现规格

### 接口定义

```go
package ai

import "context"

// Message 对话消息
type Message struct {
    Role    string // system, user, assistant
    Content string
}

// LLMService 大语言模型服务
type LLMService interface {
    // Chat 同步对话
    Chat(ctx context.Context, messages []Message) (string, error)
    
    // ChatStream 流式对话
    ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error)
}
```

### 实现

```go
package ai

import (
    "context"
    
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

type llmService struct {
    model       llms.Model
    maxTokens   int
    temperature float32
}

// NewLLMService 创建 LLM 服务
func NewLLMService(cfg *LLMConfig) (LLMService, error) {
    var model llms.Model
    var err error
    
    switch cfg.Provider {
    case "deepseek":
        // DeepSeek 兼容 OpenAI API
        model, err = openai.New(
            openai.WithToken(cfg.APIKey),
            openai.WithBaseURL(cfg.BaseURL),
            openai.WithModel(cfg.Model),
        )
        
    case "openai":
        model, err = openai.New(
            openai.WithToken(cfg.APIKey),
            openai.WithModel(cfg.Model),
        )
        
    case "ollama":
        // 使用 langchaingo ollama 支持
        // 实现略
        
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
    }
    
    if err != nil {
        return nil, err
    }
    
    return &llmService{
        model:       model,
        maxTokens:   cfg.MaxTokens,
        temperature: cfg.Temperature,
    }, nil
}

func (s *llmService) Chat(ctx context.Context, messages []Message) (string, error) {
    llmMessages := make([]llms.MessageContent, len(messages))
    for i, m := range messages {
        role := llms.ChatMessageTypeHuman
        switch m.Role {
        case "system":
            role = llms.ChatMessageTypeSystem
        case "assistant":
            role = llms.ChatMessageTypeAI
        }
        llmMessages[i] = llms.MessageContent{
            Role:  role,
            Parts: []llms.ContentPart{llms.TextPart(m.Content)},
        }
    }
    
    resp, err := s.model.GenerateContent(ctx, llmMessages,
        llms.WithMaxTokens(s.maxTokens),
        llms.WithTemperature(float64(s.temperature)),
    )
    if err != nil {
        return "", err
    }
    
    if len(resp.Choices) == 0 {
        return "", errors.New("empty response")
    }
    
    return resp.Choices[0].Content, nil
}

func (s *llmService) ChatStream(ctx context.Context, messages []Message) (<-chan string, <-chan error) {
    contentChan := make(chan string)
    errChan := make(chan error, 1)
    
    go func() {
        defer close(contentChan)
        defer close(errChan)
        
        llmMessages := make([]llms.MessageContent, len(messages))
        for i, m := range messages {
            role := llms.ChatMessageTypeHuman
            switch m.Role {
            case "system":
                role = llms.ChatMessageTypeSystem
            case "assistant":
                role = llms.ChatMessageTypeAI
            }
            llmMessages[i] = llms.MessageContent{
                Role:  role,
                Parts: []llms.ContentPart{llms.TextPart(m.Content)},
            }
        }
        
        _, err := s.model.GenerateContent(ctx, llmMessages,
            llms.WithMaxTokens(s.maxTokens),
            llms.WithTemperature(float64(s.temperature)),
            llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
                select {
                case contentChan <- string(chunk):
                case <-ctx.Done():
                    return ctx.Err()
                }
                return nil
            }),
        )
        
        if err != nil {
            errChan <- err
        }
    }()
    
    return contentChan, errChan
}
```

## 验收标准

### AC-1: 文件创建
- [x] `plugin/ai/llm.go` 文件存在

### AC-2: 编译通过
- [x] `go build ./plugin/ai/...` 无错误

### AC-3: 同步对话
- [x] DeepSeek Chat 返回完整响应
- [x] 消息角色正确传递

### AC-4: 流式对话
- [x] 流式返回内容块
- [x] 完成后 channel 正确关闭
- [x] 错误正确传递

### AC-5: 多供应商支持
- [x] OpenAI 模式正常工作
- [x] Ollama 模式正常工作 (本地)

## 实现状态

✅ **已完成** - 实现于 [plugin/ai/llm.go](../../plugin/ai/llm.go)

## 测试命令

```bash
# 单元测试
go test ./plugin/ai/... -run TestLLM -v

# 集成测试
MEMOS_AI_DEEPSEEK_API_KEY=xxx go test ./plugin/ai/... -run TestLLMIntegration -v
```

## 依赖

- AI-007 (AI 插件配置)
- `go get github.com/tmc/langchaingo`

## 预估时间

2 小时
