# AI Core 模块提升重构 - 实施总结

> **完成日期**: 2026-02-03
> **关联 Issue**: [#51](https://github.com/hrygo/divinesense/issues/51)
> **关联分支**: `refactor/51-ai-core-promotion`

---

## 概述

本次重构将 AI 功能从 `plugin/ai/` 提升为一级模块 `ai/`，反映其作为 DivineSense 核心卖点的地位。同时整合了分散在 `server/ai/` 和 `server/retrieval/` 的 AI 基础设施代码。

**核心变化**：
- `plugin/ai/*` → `ai/*`
- `server/ai/*` → `ai/core/embedding/*`
- `server/retrieval/*` → `ai/core/retrieval/*`

---

## 新目录结构

```
ai/                              # 🔴 AI 核心模块（一级模块）
├── agent/                       #   代理系统
│   ├── amazing_parrot.go       #     综合助理（折衷）
│   ├── chat_router.go          #     聊天路由
│   ├── memo_parrot.go          #     笔记助手（灰灰）
│   ├── schedule_parrot_v2.go   #     日程助理（时巧）
│   ├── tools/                  #     代理工具
│   ├── types.go                #     类型定义
│   └── cc_runner.go            #     Claude Code CLI 集成
├── core/                        #   AI 基础设施
│   ├── embedding/              #     嵌入服务（从 server/ai/ 迁移）
│   │   ├── embedder.go
│   │   ├── provider.go
│   │   └── chunker.go
│   ├── retrieval/              #     检索系统（从 server/retrieval/ 迁移）
│   │   ├── adaptive_retrieval.go
│   │   ├── hybrid_search.go
│   │   ├── reranker.go
│   │   └── bm25.go
│   ├── reranker/               #     重排服务
│   └── llm/                    #     LLM 客户端
├── router/                      #   三层意图路由
├── vector/                     #   Embedding 服务
├── memory/                     #   情景记忆
├── session/                    #   对话持久化
├── cache/                      #   LRU 缓存层
├── metrics/                    #   代理性能追踪
├── rag/                        #   RAG 高级功能
├── tags/                       #   标签建议
├── duplicate/                  #   重复检测
├── habit/                      #   习惯学习
├── genui/                      #   生成式 UI
├── graph/                      #   知识图谱
├── prediction/                 #   预测引擎
├── reminder/                   #   提醒系统
├── schedule/                   #   日程 AI
├── aitime/                     #   AI 时间解析
├── timeout/                    #   超时处理
├── review/                     #   审查服务
├── context/                    #   上下文构建
└── config.go                   #   AI 配置

plugin/                         # 其他可选插件（非 AI）
├── cron/                       # 任务调度
├── email/                      # 邮件
├── filter/                     # 过滤器
├── idp/                        # 身份提供商
├── markdown/                   # Markdown 插件
├── ocr/                        # OCR 插件
├── scheduler/                  # 调度器
├── textextract/                # 文本提取
└── webhook/                    # Webhook 插件
```

---

## Import 路径映射

| 旧路径 | 新路径 |
|:-------|:-------|
| `github.com/hrygo/divinesense/plugin/ai` | `github.com/hrygo/divinesense/ai` |
| `github.com/hrygo/divinesense/server/retrieval` | `github.com/hrygo/divinesense/ai/core/retrieval` |
| `github.com/hrygo/divinesense/server/ai` | `github.com/hrygo/divinesense/ai/core/embedding` |

---

## 迁移统计

| 类型 | 数量 |
|:-----|:-----|
| **文件迁移** | 192 个 |
| **Import 更新** | 57 处 |
| **目录删除** | 3 个（plugin/ai, server/ai, server/retrieval） |
| **新增常量** | 4 个（amazing_parrot.go 配置） |

---

## 关键改进

### 1. 代码质量修复

**P2: 流式发送效率**
- `memo_parrot.go`: chunkSize 20 → 80（4x 效率提升）

**P2: 魔法数字提取**
- `amazing_parrot.go`: 提取 4 个常量
  - `concurrentRetrievalTimeout = 45s`
  - `uiPreviewCardLimit = 5`
  - `casualChatShortThreshold = 30`
  - `casualChatModerateThreshold = 100`

**P3: 日志级别修正**
- `chat_router.go`: Debug → Info（规则匹配日志）

### 2. SafeCallback 引入

为非关键事件处理引入安全包装器，防止回调错误影响主流程：

```go
// ai/agent/types.go
func SafeCallback(callback EventCallback) SafeCallbackFunc {
    if callback == nil {
        return nil
    }
    return func(eventType string, eventData interface{}) {
        if err := callback(eventType, eventData); err != nil {
            slog.Default().LogAttrs(context.Background(), slog.LevelWarn,
                "callback failed (non-critical)",
                slog.String("event_type", eventType),
                slog.Any("error", err),
            )
        }
    }
}
```

### 3. Pre-commit Hook 优化

修复开发环境中 `go:embed` 文件检查失败的问题：

```bash
# scripts/pre-commit
# 只在前端文件变更时才检查 dist/ 存在性
FRONTEND_CHANGED=$(git diff --cached --name-only | grep -cE '^(server/router/frontend/|web/)')
if [ "$FRONTEND_CHANGED" -gt 0 ]; then
    # 自动构建前端（如需要）
fi
```

---

## 兼容性

- **向后兼容**: ✅ 仅迁移位置，API 不变
- **数据库迁移**: ✅ 无需变更
- **环境变量**: ✅ 无需变更
- **前端集成**: ✅ 无需变更

---

## 提交记录

| Commit | 描述 |
|:-------|:-----|
| `02d30a4` | refactor(ai): migrate plugin/ai, server/ai, server/retrieval to ai/ module |
| `4e2e1ef` | fix(ai): resolve golangci-lint issues in ai/ module |
| `e9485f8` | refactor(agent): introduce SafeCallback for non-critical event handling |
| `8784d92` | docs(ai): update documentation paths after AI module promotion |
| `7d3cc99` | fix: smart embed check in pre-commit hook |
| `0c997a4` | fix(ai): resolve code review issues from second audit |

---

## 升级指南

### 开发者

如果你在本地代码中引用了旧的 AI 路径，需要更新 import：

```go
// 旧
import "github.com/hrygo/divinesense/plugin/ai"
import "github.com/hrygo/divinesense/server/retrieval"
import "github.com/hrygo/divinesense/server/ai"

// 新
import "github.com/hrygo/divinesense/ai"
import "github.com/hrygo/divinesense/ai/core/retrieval"
import "github.com/hrygo/divinesense/ai/core/embedding"
```

### 部署者

无需变更。环境变量和配置保持不变。

---

## 相关文档

- [调研报告](../research/ai-core-refactor-research.md)
- [架构文档](../dev-guides/ARCHITECTURE.md)
- [路径速查](../dev-guides/PROJECT_PATHS.md)
- [元认知系统](../specs/META_COGNITION.md)

---

**实施者**: 黄飞鸿 + Claude Opus 4.5
**状态**: ✅ 完成
