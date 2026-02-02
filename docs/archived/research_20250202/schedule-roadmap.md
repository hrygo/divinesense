# 日程管理 Agent 升级路线图

> **产品定位**: 私人日程助理 | 私有化部署 | 单人专属 | 非SaaS

**文档导航**: [主路线图](./00-master-roadmap.md) | [调研报告](./schedule-research.md)

---

## 0. 设计原则

| 原则 | 说明 | 实践 |
|------|------|------|
| **静默优先** | 多路选择时默认走不打扰用户的路径 | 冲突自动调整，仅在响应中说明 |
| **职责单一** | 前端仅展示，规则由后端实现 | 预检API返回结果，前端直接渲染 |
| **资源友好** | 私有化部署，资源受限 | 减少LLM调用，本地规则优先 |
| **减法主义** | 能不加的功能就不加 | 聚焦核心场景，拒绝过度设计 |

---

## 1. 路线图概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           升级路线图 (3个阶段)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1: 稳定基础                 Phase 2: 效率提升                         │
│  ════════════════                 ════════════════                          │
│  ┌─────────────────┐              ┌─────────────────┐                       │
│  │ 1.1 时间解析加固 │              │ 2.1 快速创建模式 │                       │
│  │ 1.2 规则分类扩展 │              │ 2.2 后端预检API  │                       │
│  │ 1.3 错误恢复机制 │              │ 2.3 智能缓存层   │                       │
│  └─────────────────┘              └─────────────────┘                       │
│        ↓ (基础稳固)                      ↓ (体验流畅)                        │
│                                                                             │
│                          Phase 3: 能力扩展                                  │
│                          ════════════════                                   │
│                          ┌─────────────────┐                                │
│                          │ 3.1 批量日程支持 │                                │
│                          │ 3.2 会话持久化   │                                │
│                          │ 3.3 提醒系统集成 │                                │
│                          └─────────────────┘                                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: 稳定基础

**目标**: 消除现有痛点，确保核心功能可靠

### 1.1 时间解析加固

**问题**: LLM 可能生成错误格式的时间，导致工具调用失败

**技术方案**:
```go
// plugin/ai/agent/tools/time_normalizer.go

// TimeNormalizer 时间格式标准化器
type TimeNormalizer struct {
    timezone *time.Location
    now      func() time.Time  // 可注入，便于测试
}

// Normalize 将各种时间表达标准化为 RFC3339
// 支持: "明天3点", "下午三点", "2026-1-28", "15:00" 等
func (n *TimeNormalizer) Normalize(input string) (time.Time, error) {
    // 1. 尝试标准格式解析
    if t, err := time.Parse(time.RFC3339, input); err == nil {
        return t, nil
    }
    
    // 2. 常见格式兜底
    patterns := []string{
        "2006-01-02T15:04:05",
        "2006-01-02 15:04",
        "2006-1-2 15:04",
        "15:04",
    }
    for _, p := range patterns {
        if t, err := time.ParseInLocation(p, input, n.timezone); err == nil {
            return n.adjustDate(t), nil
        }
    }
    
    // 3. 自然语言解析 (本地规则，不调用LLM)
    return n.parseNaturalLanguage(input)
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 2-3 天 |
| 预期收益 | 工具调用成功率从 ~85% 提升至 ~98% |
| 风险 | 低 - 纯后端改动，不影响现有逻辑 |

---

### 1.2 规则分类器扩展

**问题**: LLM 意图分类需要 ~400ms，且消耗 API 配额

**技术方案**:
```go
// plugin/ai/agent/intent_classifier.go

// 扩展规则分类器，覆盖 90%+ 常见场景
func (ic *IntentClassifier) Classify(input string) TaskIntent {
    lowerInput := strings.ToLower(input)
    
    // 高置信度模式 (直接返回，不走LLM)
    patterns := []struct {
        regex  *regexp.Regexp
        intent TaskIntent
    }{
        // 创建: 时间 + 动作/事件
        {regexp.MustCompile(`(明天|后天|下周|今天).*(点|时).*(开会|会议|面试|约|见)`), IntentSimpleCreate},
        {regexp.MustCompile(`(上午|下午|晚上|早上).*(安排|约|预约)`), IntentSimpleCreate},
        
        // 查询: 疑问词 + 时间
        {regexp.MustCompile(`(今天|明天|这周|下周).*(有什么|什么安排|忙吗|有空)`), IntentSimpleQuery},
        {regexp.MustCompile(`(查|看|显示).*(日程|安排|计划)`), IntentSimpleQuery},
        
        // 修改: 修改动词 + 目标
        {regexp.MustCompile(`(改|换|调|推迟|提前|取消|删除).*(会议|日程|安排)`), IntentSimpleUpdate},
        
        // 批量: 重复关键词
        {regexp.MustCompile(`每(天|周|月|年)|工作日|周一到周五`), IntentBatchCreate},
    }
    
    for _, p := range patterns {
        if p.regex.MatchString(lowerInput) {
            return p.intent
        }
    }
    
    // 兜底: 有时间词+动作词 → 创建
    if ic.hasTimeAndAction(input) {
        return IntentSimpleCreate
    }
    
    // 真正不确定时才走 LLM
    return IntentUnknown  // 触发 LLM 分类
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 1-2 天 |
| 预期收益 | LLM 调用减少 70%+，响应延迟降低 300ms+ |
| 风险 | 低 - 规则不匹配时仍走 LLM 兜底 |

---

### 1.3 错误恢复机制

**问题**: 工具调用失败时，用户需要重新输入

**技术方案**:
```go
// plugin/ai/agent/scheduler_v2.go

// ExecuteWithRetry 带自动恢复的执行
func (a *SchedulerAgentV2) ExecuteWithRetry(ctx context.Context, input string, ...) (string, error) {
    maxRetries := 2
    
    for attempt := 0; attempt <= maxRetries; attempt++ {
        result, err := a.execute(ctx, input)
        
        if err == nil {
            return result, nil
        }
        
        // 可恢复错误: 自动修正后重试
        if recoverable, fixedInput := a.tryRecover(err, input); recoverable {
            input = fixedInput
            continue
        }
        
        // 不可恢复: 返回友好提示
        return a.formatUserFriendlyError(err), nil
    }
    
    return "抱歉，处理遇到问题，请稍后重试", nil
}

// tryRecover 尝试自动恢复
func (a *SchedulerAgentV2) tryRecover(err error, input string) (bool, string) {
    switch {
    case errors.Is(err, ErrInvalidTimeFormat):
        // 时间格式错误 → 重新解析
        return true, a.normalizeTimeInInput(input)
    case errors.Is(err, ErrToolNotFound):
        // 工具不存在 → 重新路由
        return true, input
    default:
        return false, ""
    }
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 1 天 |
| 预期收益 | 用户重试率降低 50%+，体验提升 |
| 风险 | 低 |

---

## Phase 2: 效率提升

**目标**: 减少不必要的 LLM 调用，提升响应速度

### 2.1 快速创建模式 (Fast Path)

**问题**: 简单创建需要 2 轮工具调用 (query → add)

**技术方案**:
```go
// server/service/schedule/fast_create.go

// FastCreate 快速创建模式
// 条件: 时间明确 + 后端自动检测冲突 + 自动调整
func (s *service) FastCreate(ctx context.Context, userID int32, req *FastCreateRequest) (*FastCreateResult, error) {
    // 1. 解析时间 (后端统一处理)
    startTime, err := s.timeNormalizer.Normalize(req.TimeExpression)
    if err != nil {
        return nil, err
    }
    
    // 2. 检测冲突
    conflicts, _ := s.CheckConflicts(ctx, userID, startTime.Unix(), nil, nil)
    
    // 3. 自动调整 (静默优先)
    actualStart := startTime
    adjusted := false
    if len(conflicts) > 0 {
        slot, err := s.conflictResolver.FindBestSlot(ctx, userID, startTime, req.Duration)
        if err != nil {
            return &FastCreateResult{
                Success: false,
                Message: "该时间段已满，无法自动调整",
                Conflicts: conflicts,
            }, nil
        }
        actualStart = slot.Start
        adjusted = true
    }
    
    // 4. 创建日程
    schedule, err := s.CreateSchedule(ctx, userID, &CreateScheduleRequest{
        Title:   req.Title,
        StartTs: actualStart.Unix(),
        EndTs:   toPtr(actualStart.Add(req.Duration).Unix()),
    })
    
    return &FastCreateResult{
        Success:  true,
        Schedule: schedule,
        Adjusted: adjusted,
        OriginalTime: startTime,
        Message: s.formatSuccessMessage(schedule, adjusted, startTime),
    }, nil
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 3-4 天 |
| 预期收益 | 简单创建延迟从 ~2s 降至 ~200ms (无需LLM) |
| 风险 | 中 - 需要前端配合调用新接口 |

---

### 2.2 后端预检 API

**问题**: 用户选择时间后才知道有冲突，体验差

**技术方案**:
```go
// server/router/api/v1/schedule_service.go

// CheckAvailability 检查时间段可用性
// GET /api/v1/schedules/check?start=&end=&date=
func (s *ScheduleService) CheckAvailability(ctx context.Context, req *CheckAvailabilityRequest) (*CheckAvailabilityResponse, error) {
    userID := auth.GetUserID(ctx)
    
    // 支持两种查询模式
    var conflicts []*store.Schedule
    var freeSlots []TimeSlot
    
    if req.Date != "" {
        // 模式1: 查询某天的空闲时段
        date, _ := time.Parse("2006-01-02", req.Date)
        freeSlots, _ = s.scheduleSvc.FindFreeSlots(ctx, userID, date, time.Hour)
    } else {
        // 模式2: 检查指定时段是否可用
        conflicts, _ = s.scheduleSvc.CheckConflicts(ctx, userID, req.StartTs, req.EndTs, nil)
    }
    
    return &CheckAvailabilityResponse{
        Available:    len(conflicts) == 0,
        Conflicts:    conflicts,
        FreeSlots:    freeSlots,
        Suggestion:   s.getBestSuggestion(conflicts, freeSlots),
    }, nil
}
```

**前端调用** (职责单一，仅展示):
```typescript
// 前端仅调用 + 展示，不做规则判断
const { data } = useQuery({
  queryKey: ['availability', selectedDate],
  queryFn: () => scheduleService.checkAvailability({ date: selectedDate }),
});

// 渲染后端返回的结果
return (
  <TimeGrid 
    freeSlots={data.freeSlots}      // 绿色显示
    conflicts={data.conflicts}       // 红色显示
    suggestion={data.suggestion}     // 推荐高亮
  />
);
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 2 天 |
| 预期收益 | 用户可即时看到冲突，减少无效提交 |
| 风险 | 低 - 新增API，不影响现有流程 |

---

### 2.3 智能缓存层

**问题**: 重复查询同一时间段的日程

**技术方案**:
```go
// server/service/schedule/cache.go

// ScheduleCache 日程缓存
type ScheduleCache struct {
    cache    *lru.Cache  // 使用 LRU 策略
    ttl      time.Duration
    maxItems int
}

func NewScheduleCache() *ScheduleCache {
    return &ScheduleCache{
        cache:    lru.New(100),  // 私有化部署，100条足够
        ttl:      5 * time.Minute,
        maxItems: 100,
    }
}

// 缓存键: user_id:date
func (c *ScheduleCache) Get(userID int32, date string) ([]*ScheduleInstance, bool) {
    key := fmt.Sprintf("%d:%s", userID, date)
    if item, ok := c.cache.Get(key); ok {
        entry := item.(*cacheEntry)
        if time.Since(entry.createdAt) < c.ttl {
            return entry.schedules, true
        }
        c.cache.Remove(key)
    }
    return nil, false
}

// 写操作触发失效
func (c *ScheduleCache) InvalidateUser(userID int32) {
    // 简单策略: 清除该用户所有缓存
    // 私有化部署场景，用户数=1，直接清空即可
    c.cache.Purge()
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 1 天 |
| 预期收益 | 重复查询响应 <10ms，数据库压力降低 |
| 风险 | 低 - 私有化部署，缓存一致性问题可控 |

---

## Phase 3: 能力扩展

**目标**: 扩展核心能力，支持更多场景

### 3.1 批量日程支持 (Plan-Execute 模式)

**问题**: "每周一到周五早上9点站会" 无法处理

**技术方案**:
```go
// plugin/ai/agent/plan_executor.go

// PlanExecutor 计划执行器
// 用于批量创建等复杂任务
type PlanExecutor struct {
    planner  *SchedulePlanner
    executor *ScheduleExecutor
}

// Execute 执行批量创建计划
func (pe *PlanExecutor) Execute(ctx context.Context, input string) (*BatchResult, error) {
    // 1. 规划阶段: 解析重复规则
    plan, err := pe.planner.Plan(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // 2. 预览生成的实例 (可选，用于确认)
    instances := plan.GeneratePreview(10)  // 预览前10个
    
    // 3. 执行阶段: 创建日程模板 + 重复规则
    schedule, err := pe.executor.CreateRecurring(ctx, plan)
    if err != nil {
        return nil, err
    }
    
    return &BatchResult{
        Schedule:  schedule,
        Instances: instances,
        RRule:     plan.RecurrenceRule,
    }, nil
}

// SchedulePlanner 日程规划器
type SchedulePlanner struct{}

// Plan 解析自然语言为重复规则
func (p *SchedulePlanner) Plan(ctx context.Context, input string) (*SchedulePlan, error) {
    // 规则匹配 (不调用LLM)
    patterns := map[*regexp.Regexp]string{
        regexp.MustCompile(`每天`):          "FREQ=DAILY",
        regexp.MustCompile(`每周`):          "FREQ=WEEKLY",
        regexp.MustCompile(`工作日|周一到周五`): "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
        regexp.MustCompile(`每月(\d+)号`):   "FREQ=MONTHLY;BYMONTHDAY=$1",
    }
    
    for pattern, rrule := range patterns {
        if pattern.MatchString(input) {
            return &SchedulePlan{
                Title:          p.extractTitle(input),
                RecurrenceRule: rrule,
                StartTime:      p.extractStartTime(input),
            }, nil
        }
    }
    
    return nil, fmt.Errorf("无法解析重复规则")
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 5-7 天 |
| 预期收益 | 支持重复日程，覆盖工作场景 |
| 风险 | 中 - 需要处理复杂的时间计算逻辑 |

---

### 3.2 会话持久化

**问题**: 服务重启后会话丢失

**技术方案**:
```sql
-- store/db/postgres/migration/xxx_add_conversation_context.sql

CREATE TABLE conversation_context (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL REFERENCES "user"(id),
    context_data JSONB NOT NULL DEFAULT '{}',
    created_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    updated_ts BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
    
    INDEX idx_conversation_user (user_id),
    INDEX idx_conversation_updated (updated_ts)
);

-- 自动清理30天前的会话
CREATE OR REPLACE FUNCTION cleanup_old_conversations() RETURNS void AS $$
BEGIN
    DELETE FROM conversation_context 
    WHERE updated_ts < EXTRACT(EPOCH FROM NOW()) - 30 * 24 * 3600;
END;
$$ LANGUAGE plpgsql;
```

```go
// store/conversation_context.go

type ConversationContextStore struct {
    db *sql.DB
}

func (s *ConversationContextStore) Save(ctx context.Context, sessionID string, context *agent.ConversationContext) error {
    data, _ := json.Marshal(context)
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO conversation_context (session_id, user_id, context_data, updated_ts)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (session_id) DO UPDATE SET 
            context_data = $3, 
            updated_ts = $4
    `, sessionID, context.UserID, data, time.Now().Unix())
    return err
}

func (s *ConversationContextStore) Load(ctx context.Context, sessionID string) (*agent.ConversationContext, error) {
    var data []byte
    err := s.db.QueryRowContext(ctx, 
        `SELECT context_data FROM conversation_context WHERE session_id = $1`, 
        sessionID,
    ).Scan(&data)
    if err != nil {
        return nil, err
    }
    
    var context agent.ConversationContext
    json.Unmarshal(data, &context)
    return &context, nil
}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 2-3 天 |
| 预期收益 | 服务重启后会话恢复，支持多设备同步 |
| 风险 | 低 - 标准 CRUD 操作 |

---

### 3.3 提醒系统集成

**问题**: 创建日程后无法提醒

**技术方案** (私有化部署适配):
```go
// plugin/scheduler/reminder.go

// ReminderScheduler 提醒调度器
// 私有化部署: 使用本地定时任务，不依赖外部服务
type ReminderScheduler struct {
    store    *store.Store
    notifier Notifier
    ticker   *time.Ticker
}

// Start 启动提醒检查循环
func (r *ReminderScheduler) Start(ctx context.Context) {
    r.ticker = time.NewTicker(1 * time.Minute)
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-r.ticker.C:
            r.checkAndNotify(ctx)
        }
    }
}

// checkAndNotify 检查并发送提醒
func (r *ReminderScheduler) checkAndNotify(ctx context.Context) {
    now := time.Now()
    upcoming := now.Add(15 * time.Minute)  // 检查未来15分钟
    
    schedules, _ := r.store.ListSchedules(ctx, &store.FindSchedule{
        StartTs: toPtr(now.Unix()),
        EndTs:   toPtr(upcoming.Unix()),
    })
    
    for _, sched := range schedules {
        if r.shouldNotify(sched, now) {
            r.notifier.Notify(ctx, sched)
        }
    }
}

// Notifier 通知接口 (支持多种方式)
type Notifier interface {
    Notify(ctx context.Context, schedule *store.Schedule) error
}

// WebPushNotifier 浏览器推送 (PWA)
type WebPushNotifier struct{}

// DesktopNotifier 桌面通知 (Electron/Tauri)
type DesktopNotifier struct{}
```

**ROI 分析**:
| 指标 | 值 |
|------|-----|
| 开发投入 | 3-5 天 |
| 预期收益 | 日程提醒功能，提升实用性 |
| 风险 | 中 - 需要处理浏览器兼容性、权限等 |

---

## 4. 总览与优先级

### 4.1 投入产出对比

| 阶段 | 特性 | 投入 | 收益 | 优先级 |
|------|------|------|------|--------|
| P1 | 时间解析加固 | 2-3天 | 成功率 +13% | ⭐⭐⭐⭐⭐ |
| P1 | 规则分类扩展 | 1-2天 | 延迟 -300ms | ⭐⭐⭐⭐⭐ |
| P1 | 错误恢复机制 | 1天 | 体验提升 | ⭐⭐⭐⭐ |
| P2 | 快速创建模式 | 3-4天 | 延迟 -1.8s | ⭐⭐⭐⭐ |
| P2 | 后端预检API | 2天 | 体验提升 | ⭐⭐⭐ |
| P2 | 智能缓存层 | 1天 | 性能提升 | ⭐⭐⭐ |
| P3 | 批量日程支持 | 5-7天 | 功能扩展 | ⭐⭐⭐ |
| P3 | 会话持久化 | 2-3天 | 可靠性 | ⭐⭐ |
| P3 | 提醒系统集成 | 3-5天 | 功能扩展 | ⭐⭐ |

### 4.2 里程碑计划

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            里程碑计划                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  M1: 稳定版 (Phase 1)                                                   │
│  ─────────────────────                                                  │
│  - 时间解析成功率 > 98%                                                  │
│  - LLM 调用减少 70%                                                      │
│  - 错误自动恢复                                                          │
│                                                                         │
│  M2: 高效版 (Phase 2)                                                   │
│  ─────────────────────                                                  │
│  - 简单创建 < 500ms                                                      │
│  - 预检 API 上线                                                         │
│  - 缓存命中率 > 60%                                                      │
│                                                                         │
│  M3: 完整版 (Phase 3)                                                   │
│  ─────────────────────                                                  │
│  - 支持重复日程                                                          │
│  - 会话持久化                                                            │
│  - 基础提醒功能                                                          │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 规则分类器误判 | 用户意图被错误路由 | LLM 兜底 + 置信度阈值 |
| 时间解析歧义 | "下午" 解析为不同时间 | 响应中明确显示实际时间 |
| 缓存一致性 | 看到过期数据 | 写操作触发失效 + 短TTL |
| 提醒推送兼容性 | 部分浏览器不支持 | 优雅降级 + 多通道 |

---

## 6. 总结

本路线图遵循 **稳定 → 高效 → 扩展** 的渐进式策略：

1. **Phase 1** 解决现有痛点，确保基础可靠
2. **Phase 2** 优化关键路径，提升响应速度
3. **Phase 3** 扩展核心能力，覆盖更多场景

关键设计决策：
- **静默优先**: 冲突自动调整，不打扰用户
- **后端为主**: 复杂规则由后端实现，前端仅展示
- **资源友好**: 减少 LLM 调用，适合私有化部署
