# AI-015: GetRelatedMemos API

## 概述

实现相关笔记推荐 API，基于向量相似度推荐。

## 目标

在 Memo 详情页展示相关内容。

## 交付物

- `server/router/api/v1/ai_service.go` (扩展 GetRelatedMemos 方法)

## 验收标准

### AC-1: 编译通过
- [x] `go build ./server/...` 无错误

### AC-2: 认证检查
- [x] 未登录时返回 Unauthenticated

### AC-3: 权限检查
- [x] 私有 Memo 只能所有者访问

### AC-4: 相关推荐
- [x] 返回语义相似的 Memo
- [x] 不包含自己
- [x] 结果按相似度排序

### AC-5: 向量缺失处理
- [x] Memo 没有向量时实时生成

## 实现状态

✅ **已完成** - 实现于 [server/router/api/v1/ai_service.go:381-475](../../server/router/api/v1/ai_service.go#L381-L475)

**优化**: 向量缺失时自动调用 EmbeddingService 实时生成并保存到数据库。

## 测试命令

```bash
curl http://localhost:8081/api/v1/memos/abc123/related?limit=5 \
  -H "Authorization: Bearer $TOKEN"
```

## 依赖

- AI-006 (PostgreSQL 向量搜索)
- AI-008 (Embedding 服务)

## 预估时间

1.5 小时
