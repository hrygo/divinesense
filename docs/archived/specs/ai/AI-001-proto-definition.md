# AI-001: Proto 定义

## 概述

定义 AI 服务的 gRPC Protocol Buffer 接口。

## 目标

创建 `ai_service.proto` 文件，定义所有 AI 相关的 gRPC 服务和消息类型。

## 交付物

- `proto/api/v1/ai_service.proto`

## 实现规格

### 服务定义

```protobuf
syntax = "proto3";
package memos.api.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";

option go_package = "github.com/usememos/memos/proto/gen/api/v1";

service AIService {
  // 语义搜索
  rpc SemanticSearch(SemanticSearchRequest) returns (SemanticSearchResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/search"
      body: "*"
    };
  }

  // 标签推荐
  rpc SuggestTags(SuggestTagsRequest) returns (SuggestTagsResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/suggest-tags"
      body: "*"
    };
  }

  // AI 对话 (流式)
  rpc Chat(ChatRequest) returns (stream ChatResponse) {
    option (google.api.http) = {
      post: "/api/v1/ai/chat"
      body: "*"
    };
  }

  // 相关笔记推荐
  rpc GetRelatedMemos(GetRelatedMemosRequest) returns (GetRelatedMemosResponse) {
    option (google.api.http) = {
      get: "/api/v1/{name=memos/*}/related"
    };
  }
}
```

### 消息定义

```protobuf
message SemanticSearchRequest {
  string query = 1 [(google.api.field_behavior) = REQUIRED];
  int32 limit = 2;  // 默认 10, 最大 50
}

message SemanticSearchResponse {
  repeated SearchResult results = 1;
}

message SearchResult {
  string name = 1;      // memos/{id}
  string snippet = 2;   // 内容摘要
  float score = 3;      // 相关性分数
}

message SuggestTagsRequest {
  string content = 1 [(google.api.field_behavior) = REQUIRED];
  int32 limit = 2;  // 默认 5
}

message SuggestTagsResponse {
  repeated string tags = 1;
}

message ChatRequest {
  string message = 1 [(google.api.field_behavior) = REQUIRED];
  repeated string history = 2;  // 对话历史
}

message ChatResponse {
  string content = 1;           // 流式内容块
  repeated string sources = 2;  // 引用来源 memos/{id}
  bool done = 3;                // 流结束标记
}

message GetRelatedMemosRequest {
  string name = 1 [(google.api.field_behavior) = REQUIRED];  // memos/{id}
  int32 limit = 2;  // 默认 5
}

message GetRelatedMemosResponse {
  repeated SearchResult memos = 1;
}
```

## 验收标准

### AC-1: 文件创建
- [x] `proto/api/v1/ai_service.proto` 文件存在
- [x] 文件语法正确，无编译错误

### AC-2: 代码生成
- [x] 执行 `cd proto && buf generate` 成功
- [x] `proto/gen/api/v1/ai_service.pb.go` 生成
- [x] `proto/gen/api/v1/ai_service_grpc.pb.go` 生成
- [x] `web/src/types/proto/api/v1/ai_service_pb.ts` 生成


### AC-3: Lint 检查
- [x] 执行 `cd proto && buf lint` 无错误

## 实现状态

✅ **已完成** - 实现于 [proto/api/v1/ai_service.proto](../../proto/api/v1/ai_service.proto)

## 测试命令

```bash
cd proto
buf lint
buf generate
ls -la gen/api/v1/ai_service*.go
ls -la ../web/src/types/proto/api/v1/ai_service_pb.ts
```

## 依赖

无

## 预估时间

2 小时
