# SPEC-003: AI Provider 初始化与配置

**优先级**: P0 (阻塞)
**预计工时**: 4 小时
**依赖**: SPEC-001

## 目标
实现 AI Provider 模块,封装 LLM 和 Embedding API 调用,包括重试机制和模型探测。

## 实施内容

### 1. 新增模块
**文件路径**: `server/ai/provider.go`

核心功能:
```go
package ai

type Provider struct {
    client *openaiclient.Client
    config *Config
}

type Config struct {
    BaseURL          string
    APIKey          string
    EmbeddingModel  string  // 默认: "BAAI/bge-m3"
    ChatModel       string  // 默认: "gpt-4o-mini"
    MaxRetries      int     // 默认: 3
    Timeout         time.Duration // 默认: 30s
}

// NewProvider 创建并初始化 Provider
func NewProvider(cfg *Config) (*Provider, error)

// Embedding 将文本转换为向量
func (p *Provider) Embedding(ctx context.Context, text string) ([]float32, error)

// Chat 进行对话调用
func (p *Provider) Chat(ctx context.Context, messages []llms.MessageContent) (string, error)

// ChatStream 流式对话
func (p *Provider) ChatStream(ctx context.Context, messages []llms.MessageContent) (<-chan llms.GenerationChunk, error)

// ListModels 探测可用模型列表
func (p *Provider) ListModels(ctx context.Context) ([]string, error)

// Validate 验证配置有效性
func (p *Provider) Validate(ctx context.Context) error
```

### 2. 环境变量配置
**文件路径**: `cmd/memos/main.go`

新增环境变量读取:
```go
type AIConfig struct {
    BaseURL         string
    APIKey         string
    EmbeddingModel string
    ChatModel      string
}

func readAIConfig() *AIConfig {
    return &AIConfig{
        BaseURL:         os.Getenv("MEMOS_AI_BASE_URL"),
        APIKey:         os.Getenv("MEMOS_AI_API_KEY"),
        EmbeddingModel: getEnv("MEMOS_AI_EMBEDDING_MODEL", "BAAI/bge-m3"),
        ChatModel:      getEnv("MEMOS_AI_CHAT_MODEL", "gpt-4o-mini"),
    }
}
```

### 3. 启动时验证
**文件路径**: `cmd/memos/main.go`

在 server 启动前验证 AI 配置:
```go
// 初始化 AI Provider
aiConfig := readAIConfig()
provider, err := ai.NewProvider(&ai.Config{
    BaseURL:         aiConfig.BaseURL,
    APIKey:         aiConfig.APIKey,
    EmbeddingModel: aiConfig.EmbeddingModel,
    ChatModel:      aiConfig.ChatModel,
})
if err != nil {
    log.Fatal("Failed to initialize AI provider:", err)
}

// 验证 API 连通性
if err := provider.Validate(context.Background()); err != nil {
    log.Warn("AI provider validation failed:", err)
    // 非致命错误,允许降级启动
}
```

### 4. 依赖管理
**文件**: `go.mod`

添加依赖:
```go
require (
    github.com/tmc/langchaingo v0.1.9  // 锁定版本
)
```

### 5. 重试机制
**文件**: `server/ai/provider.go`

实现指数退避重试:
```go
func (p *Provider) doWithRetry(ctx context.Context, fn func() error) error {
    var lastErr error
    for i := 0; i < p.config.MaxRetries; i++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
            waitTime := time.Duration(math.Pow(2, float64(i))) * time.Second
            select {
            case <-time.After(waitTime):
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
    return lastErr
}
```

## 验收标准

### AC-1: 代码编译通过
```bash
# 执行
go mod tidy
go build ./cmd/memos

# 预期结果
- 编译成功
- langchaingo 依赖已下载
```

### AC-2: 环境变量配置测试
```bash
# 执行
export MEMOS_AI_BASE_URL="https://api.openai.com/v1"
export MEMOS_AI_API_KEY="sk-test"
export MEMOS_AI_EMBEDDING_MODEL="text-embedding-3-small"

go run ./cmd/memos --mode dev

# 预期结果
- 服务成功启动
- 日志显示 "AI provider initialized with model: text-embedding-3-small"
```

### AC-3: Embedding API 调用测试
**测试文件**: `server/ai/provider_test.go`
```go
func TestEmbedding(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    provider := setupTestProvider(t)
    embedding, err := provider.Embedding(context.Background(), "Hello, world!")

    assert.NoError(t, err)
    assert.Len(t, embedding, 1024) // BAAI/bge-m3 输出 1024 维
}
```

### AC-4: 重试机制测试
```bash
# 执行
# 需要模拟网络故障场景(可使用 httpbin.org/delay/10)

# 预期结果
- 请求失败后自动重试
- 重试间隔符合指数退避(1s, 2s, 4s)
- 最终失败或成功返回明确错误
```

### AC-5: 模型探测测试
```bash
# 执行
curl -X POST http://localhost:8081/api/v1/ai/models/debug \
  -H "Authorization: Bearer <token>"

# 预期结果
- 返回可用模型列表
- 包含 Embedding 模型
- 包含 Chat 模型
```

### AC-6: 内存使用检查
```bash
# 执行
ps aux | grep memos

# 预期结果
- 常驻内存 < 200MB
- 无内存泄漏
```

## 回滚方案
- 移除 `server/ai/provider.go`
- 从 `main.go` 中移除 AI 初始化代码
- 回滚 `go.mod`

## 注意事项
- API Key 必须通过环境变量传递,禁止硬编码
- Langchaingo 版本锁定,避免 Breaking Changes
- 重试机制设置超时,防止无限等待
- 默认模型需兼容 OpenAI 接口规范