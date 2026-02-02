# Phase 1 Kickoff - 团队通知

> **日期**: 2026-01-27  
> **状态**: Sprint 0 完成，Phase 1 已解锁  
> **版本**: v1.0

---

## 总体状态

Sprint 0 接口契约已完成，包括：
- 7 个公共服务接口定义
- 7 个 Mock 实现（含测试数据）
- 7 套契约测试（全部通过）
- 3 个数据库迁移文件
- Code Review 修复 8 项问题

**三个团队可并行启动 Phase 1 开发。**

---

## 团队 B (助理+日程)

### 可用服务

| 服务 | 路径 | 用途 |
|:---|:---|:---|
| MemoryService | `plugin/ai/memory/` | 会话记忆 + 用户偏好 |
| RouterService | `plugin/ai/router/` | 意图分类 + 模型选择 |
| TimeService | `plugin/ai/aitime/` | 时间表达解析 |
| CacheService | `plugin/ai/cache/` | 通用缓存 |
| MetricsService | `plugin/ai/metrics/` | 指标记录 |
| SessionService | `plugin/ai/session/` | 会话持久化 |

### 重要接口变更

```go
// SearchEpisodes 新增必填参数 userID，确保多租户隔离
SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]EpisodicMemory, error)
```

### 使用示例

```go
import "github.com/usememos/memos/plugin/ai/memory"

svc := memory.NewMockMemoryService()

// 获取用户偏好
prefs, _ := svc.GetPreferences(ctx, userID)

// 搜索用户的历史记忆（必须指定 userID）
episodes, _ := svc.SearchEpisodes(ctx, userID, "会议", 10)
```

### Phase 1 任务

- P1-B001: 工具可靠性增强 (依赖 P1-A001)
- P1-B002: 错误恢复机制 (无依赖)

---

## 团队 C (笔记增强)

### 可用服务

| 服务 | 路径 | 用途 |
|:---|:---|:---|
| MemoryService | `plugin/ai/memory/` | 用户偏好获取 |
| RouterService | `plugin/ai/router/` | 任务模型选择 |
| VectorService | `plugin/ai/vector/` | 向量检索 + 混合搜索 |
| CacheService | `plugin/ai/cache/` | 结果缓存 |
| MetricsService | `plugin/ai/metrics/` | 指标记录 |

### 重要 Contract 约定

1. **VectorService.SearchSimilar filter 为严格匹配**
   - 缺失 filter key 视为不匹配（多租户安全）
   - 必须显式传入 `user_id` 过滤

2. **VectorResult.Score 范围保证 [0, 1]**
   - 已 clamp 处理，可安全用于 UI 展示

3. **HybridSearch.MatchType 三种状态**
   - `keyword`: 仅关键字命中
   - `vector`: 仅向量相似命中
   - `hybrid`: 两者都命中

### 使用示例

```go
import "github.com/usememos/memos/plugin/ai/vector"

svc := vector.NewMockVectorService()

// 向量搜索（必须指定 user_id filter）
filter := map[string]any{"user_id": int32(1)}
results, _ := svc.SearchSimilar(ctx, queryVector, 10, filter)

// 混合搜索
searchResults, _ := svc.HybridSearch(ctx, "项目进度", 5)
```

### Phase 1 任务

- P1-C001: 搜索结果高亮 ✅ 已完成
- P1-C002: 上下文智能摘录 🔄 进行中

---

## 团队 A (公共服务)

### Phase 1 准备工作

- Mock 实现已可供团队 B/C 并行开发
- 真实实现需遵循 Mock 的 Contract 行为
- 数据库迁移文件已就绪

### 数据库迁移

| 文件 | 内容 |
|:---|:---|
| `V0.53.0__add_episodic_memory.sql` | 情景记忆表 |
| `V0.53.1__add_user_preferences.sql` | 用户偏好表 |
| `V0.53.2__add_conversation_context.sql` | 会话上下文表 |

### Phase 1 任务

- P1-A001: 轻量记忆系统 (无依赖)
- P1-A002: 基础评估指标 (无依赖)

---

## 运行测试

```bash
# 验证所有契约测试
go test ./plugin/ai/memory/... ./plugin/ai/router/... ./plugin/ai/vector/... \
        ./plugin/ai/aitime/... ./plugin/ai/cache/... ./plugin/ai/metrics/... \
        ./plugin/ai/session/...
```

---

## 下一步行动

| 团队 | 行动 | 阻塞状态 |
|:---|:---|:---|
| B | 启动 P1-B001 工具可靠性 | 无阻塞 |
| C | 启动 P1-C001 搜索高亮 | 无阻塞 |
| A | 启动 P1-A001 记忆系统真实实现 | 无阻塞 |

---

## 参考文档

- [Sprint 0 Spec](./sprint-0/S0-interface-contract.md)
- [实施计划 INDEX](./INDEX.md)
- [主路线图](../archived/research_20250202/00-master-roadmap.md)
