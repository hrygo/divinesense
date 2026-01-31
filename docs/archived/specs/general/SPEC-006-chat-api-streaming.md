# SPEC-006: Chat API 与流式响应

**优先级**: P1 (核心功能)
**预计工时**: 8 小时
**依赖**: SPEC-003, SPEC-005

## 目标
实现支持 SSE (Server-Sent Events) 的 Chat API,提供流式 AI 对话能力。

## 实施内容

### 1. Proto 定义
**文件路径**: `proto/api/v1/ai_service.proto`

```protobuf
syntax = "proto3";

package memos.api.v1;

import "google/api/annotations.proto";

service AIService {
  // Chat 流式对话
  rpc Chat(ChatRequest) returns (stream ChatResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/chat"
      body: "*"
    };
  }

  // Summarize 摘要
  rpc Summarize(SummarizeRequest) returns (SummarizeResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/summarize/{id}"
      body: "*"
    };
  }
}

message ChatRequest {
  string message = 1;
  string filter = 2; // 可选: 过滤器 (如 "tag:important")
}

message ChatResponse {
  string answer = 1; // 流式文本片段
  repeated Citation citations = 2; // 引用来源
  bool done = 3; // 是否结束
}

message Citation {
  int32 memo_id = 1;
  string content = 2;
  float32 score = 3; // 相似度分数
}

message SummarizeRequest {
  int32 id = 1; // Memo ID
}

message SummarizeResponse {
  string summary = 1;
  repeated string tags = 2; // 建议标签
}
```

### 2. 重新生成代码
```bash
cd proto && buf generate
```

### 3. Chat 服务实现
**文件路径**: `server/router/api/v1/ai_service.go`

```go
package v1

import (
    "context"
    "io"
    "github.com/usememos/memos/server/ai"
)

// Chat 实现流式对话
func (s *APIV1Service) Chat(ctx context.Context, req *connect.Request[v1.ChatRequest], stream *connect.ServerStream[v1.ChatResponse]) error {
    userID := ctx.Value(userIDKey).(int32)

    // 1. 构建 RAG Pipeline
    rag := s.ragPipeline

    // 2. 检索相关文档
    memos, err := rag.Retrieve(ctx, req.Msg.Message, &ai.RetrievalConfig{
        TopK:       20,
        RerankTopK: 5,
        MinScore:   0.5,
    })
    if err != nil {
        return err
    }

    // 3. 构建 Prompt
    prompt := buildChatPrompt(req.Msg.Message, memos)

    // 4. 调用 LLM (流式)
    messages := []llms.MessageContent{
        llms.TextParts(llms.RoleSystem, prompt),
        llms.TextParts(llms.RoleUser, req.Msg.Message),
    }

    chunkChan, err := rag.provider.ChatStream(ctx, messages)
    if err != nil {
        return err
    }

    // 5. 流式发送响应
    var fullAnswer strings.Builder
    for chunk := range chunkChan {
        fullAnswer.Write(chunk.Text)

        // 首次发送时包含 Citations
        if len(citations) == 0 {
            citations = buildCitations(memos)
        }

        err := stream.Send(&connect.Response[v1.ChatResponse]{
            Msg: &v1.ChatResponse{
                Answer:    chunk.Text,
                Citations: citations,
                Done:      false,
            },
        })
        if err != nil {
            return err
        }
    }

    // 6. 发送结束标记
    return stream.Send(&connect.Response[v1.ChatResponse]{
        Msg: &v1.ChatResponse{
            Answer:    "",
            Citations: citations,
            Done:      true,
        },
    })
}

// buildChatPrompt 构建 RAG Prompt
func buildChatPrompt(query string, memos []*store.Memo) string {
    context := "以下是相关的知识库内容:\n\n"
    for i, memo := range memos {
        context += fmt.Sprintf("[%d] %s\n", i+1, memo.Content)
    }

    return context + fmt.Sprintf(`
你是一个智能助手,基于上述知识库内容回答用户问题。

用户问题: %s

回答要求:
1. 仅基于知识库内容回答,不要编造
2. 引用来源时使用 [1], [2] 格式
3. 如果知识库内容不足以回答,明确告知
`, query)
}

// buildCitations 构建引用来源
func buildCitations(memos []*store.Memo) []*v1.Citation {
    citations := make([]*v1.Citation, len(memos))
    for i, memo := range memos {
        citations[i] = &v1.Citation{
            MemoId:  memo.ID,
            Content: truncate(memo.Content, 200),
            Score:   0.8, // 从 Rerank 结果获取
        }
    }
    return citations
}
```

### 4. Summarize 服务实现
**文件路径**: `server/router/api/v1/ai_service.go`

```go
// Summarize 生成摘要和建议标签
func (s *APIV1Service) Summarize(ctx context.Context, req *connect.Request[v1.SummarizeRequest]) (*connect.Response[v1.SummarizeResponse], error) {
    // 1. 获取 Memo
    memo, err := s.store.GetMemo(ctx, &store.FindMemo{
        ID: &req.Msg.Id,
    })
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "memo not found")
    }

    // 2. 构建 Prompt
    prompt := fmt.Sprintf(`
请为以下笔记生成摘要和建议标签:

笔记内容:
%s

要求:
1. 摘要: 1-2 句话概括核心内容
2. 标签: 3-5 个相关标签 (用逗号分隔)

输出格式 (JSON):
{
  "summary": "...",
  "tags": ["tag1", "tag2", "tag3"]
}
`, memo.Content)

    // 3. 调用 LLM
    messages := []llms.MessageContent{
        llms.TextParts(llms.RoleUser, prompt),
    }

    response, err := s.provider.Chat(ctx, messages)
    if err != nil {
        return nil, err
    }

    // 4. 解析 JSON
    result := struct {
        Summary string   `json:"summary"`
        Tags    []string `json:"tags"`
    }{}

    if err := json.Unmarshal([]byte(response), &result); err != nil {
        return nil, err
    }

    return connect.NewResponse(&v1.SummarizeResponse{
        Summary: result.Summary,
        Tags:    result.Tags,
    }), nil
}
```

### 5. Connect RPC 注册
**文件路径**: `server/router/api/v1/connect_services.go`

```go
func RegisterAIServiceServices(mux *http.ServeMux, server *APIV1Service) {
    path, handler := v1connect.NewAIServiceHandler(server)
    mux.Handle(path, handler)
}
```

## 验收标准

### AC-1: Proto 编译成功
```bash
# 执行
cd proto && buf generate

# 预期结果
- Go 代码生成: `proto/gen/api/v1/ai_service.pb.go`
- TypeScript 代码生成: `web/src/types/proto/api/v1/ai_service_pb.ts`
- 无编译错误
```

### AC-2: Chat API 手动测试
```bash
# 执行 SSE 请求
curl -N http://localhost:8081/memos.api.v1.AIService/Chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message": "如何使用 Go 语言?"}' \
  --no-buffer -v

# 预期结果
- HTTP 200
- Content-Type: text/event-stream
- 收到多个 chunk (逐步流式输出)
- 最后一个 chunk done=true
- citations 字段包含引用的 Memo ID
```

### AC-3: Summarize API 测试
```bash
# 执行
curl -X POST http://localhost:8081/memos.api.v1.AIService/Summarize \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"id": 123}'

# 预期结果
- 返回 JSON 格式
- summary 字段为非空字符串
- tags 数组包含 3-5 个标签
- HTTP 200
```

### AC-4: 集成测试
**测试文件**: `server/router/api/v1/ai_service_test.go`

```go
func TestChatService(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    server := setupTestServer(t)

    // 模拟流式请求
    stream := &mockServerStream{}
    req := connect.NewRequest(&v1.ChatRequest{
        Message: "测试查询",
    })

    err := server.Chat(ctx, req, stream)

    assert.NoError(t, err)
    assert.Greater(t, len(stream.sent), 1) // 至少发送 2 次 (开始 + 结束)
    assert.True(t, stream.sent[len(stream.sent)-1].Msg.Done)
}
```

### AC-5: 性能测试
```bash
# 测试首字节延迟 (Time to First Token)
curl -w "@curl-format.txt" -N http://localhost:8081/memos.api.v1.AIService/Chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"message": "测试查询"}'

# curl-format.txt 内容:
#      time_starttransfer: %{time_starttransfer}\n

# 预期结果
- 首字节延迟 < 2s
- 总响应时间 < 10s
```

### AC-6: 错误处理测试
```bash
# 场景 1: 无效请求
curl -X POST http://localhost:8081/memos.api.v1.AIService/Chat \
  -H "Content-Type: application/json" \
  -d '{"message": ""}'

# 预期: HTTP 400 InvalidArgument

# 场景 2: 未授权
curl -X POST http://localhost:8081/memos.api.v1.AIService/Chat \
  -H "Content-Type: application/json" \
  -d '{"message": "测试"}'

# 预期: HTTP 401 Unauthenticated
```

### AC-7: TypeScript 类型检查
```bash
# 执行
cd web && pnpm lint

# 预期结果
- TypeScript 编译通过
- 类型定义正确
```

## 回滚方案
- 禁用 Chat API 路由
- 保留 Proto 定义,不影响现有 API

## 注意事项
- SSE 需正确处理连接断开 (context cancel)
- 流式响应需设置合理的超时时间 (30s)
- Citations 仅在首次响应时发送,避免重复
- Summarize 需处理 JSON 解析失败场景
- 内存使用需监控,避免长连接泄漏