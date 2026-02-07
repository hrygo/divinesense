# P2-B003: 后端预检 API

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 2 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 4

---

## 1. 目标与背景

### 1.1 核心目标

提供日程创建前的预检 API，在用户确认前验证时间冲突、格式有效性，减少无效创建和后续取消。

### 1.2 用户价值

- 冲突提前告知，减少撤销操作
- 创建成功率提升至 95%+
- 更流畅的创建体验

### 1.3 技术价值

- 前后端职责分离
- 可复用的验证逻辑
- 为批量创建预检铺路

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-A005: 通用缓存层（缓存预检结果）
- [x] P1-A004: 时间解析服务（时间验证）

### 2.2 并行依赖

- P2-B002: 快速创建模式（集成预检）

### 2.3 后续依赖

- P3-B003: 批量日程支持（批量预检）
- P2-C002: 重复检测（相似笔记预检）

---

## 3. 功能设计

### 3.1 架构图

```
                    预检 API 流程
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   前端请求: POST /api/v1/schedule/precheck                 │
│                     │                                      │
│                     ▼                                      │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              PrecheckService                         │ │
│   │                                                      │ │
│   │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │ │
│   │  │ 时间格式验证 │  │ 冲突检测    │  │ 业务规则    │  │ │
│   │  │             │  │             │  │             │  │ │
│   │  │ • 非空     │  │ • 同时段   │  │ • 工作时间 │  │ │
│   │  │ • 非过去   │  │ • 重叠检测 │  │ • 最大时长 │  │ │
│   │  │ • 格式正确 │  │ • 缓冲时间 │  │ • 频率限制 │  │ │
│   │  └─────────────┘  └─────────────┘  └─────────────┘  │ │
│   │         │                │                │          │ │
│   │         └────────────────┼────────────────┘          │ │
│   │                          ▼                           │ │
│   │                  ┌─────────────┐                     │ │
│   │                  │ 汇总结果    │                     │ │
│   │                  └─────────────┘                     │ │
│   └─────────────────────────────────────────────────────┘ │
│                          │                                 │
│                          ▼                                 │
│   ┌─────────────────────────────────────────────────────┐ │
│   │  响应:                                               │ │
│   │  {                                                   │ │
│   │    "valid": true/false,                             │ │
│   │    "errors": [...],                                 │ │
│   │    "warnings": [...],                               │ │
│   │    "suggestions": [...]                             │ │
│   │  }                                                   │ │
│   └─────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 API 定义

```go
// server/router/api/v1/schedule_precheck.go

// POST /api/v1/schedule/precheck
type PrecheckRequest struct {
    Title     string    `json:"title"`
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
    Duration  int       `json:"duration"`  // 分钟
    Location  string    `json:"location,omitempty"`
}

type PrecheckResponse struct {
    Valid       bool                `json:"valid"`
    Errors      []PrecheckError     `json:"errors,omitempty"`
    Warnings    []PrecheckWarning   `json:"warnings,omitempty"`
    Suggestions []PrecheckSuggestion `json:"suggestions,omitempty"`
}

type PrecheckError struct {
    Code    string `json:"code"`     // "TIME_CONFLICT", "INVALID_TIME", etc.
    Message string `json:"message"`
    Field   string `json:"field,omitempty"`
}

type PrecheckWarning struct {
    Code    string `json:"code"`     // "OUTSIDE_WORK_HOURS", "LONG_DURATION", etc.
    Message string `json:"message"`
}

type PrecheckSuggestion struct {
    Type  string `json:"type"`      // "alternative_time"
    Value any    `json:"value"`
}
```

### 3.3 预检服务

```go
// plugin/ai/agent/schedule/precheck_service.go

type PrecheckService struct {
    scheduleStore ScheduleStore
    timeService   TimeService
    cache         CacheService
}

func NewPrecheckService(store ScheduleStore, timeSvc TimeService, cache CacheService) *PrecheckService {
    return &PrecheckService{
        scheduleStore: store,
        timeService:   timeSvc,
        cache:         cache,
    }
}

func (s *PrecheckService) Precheck(ctx context.Context, userID int32, req *PrecheckRequest) *PrecheckResponse {
    response := &PrecheckResponse{Valid: true}
    
    // 1. 时间格式验证
    s.validateTimeFormat(req, response)
    
    // 2. 冲突检测
    s.detectConflicts(ctx, userID, req, response)
    
    // 3. 业务规则验证
    s.validateBusinessRules(req, response)
    
    // 4. 生成建议
    if !response.Valid {
        s.generateSuggestions(ctx, userID, req, response)
    }
    
    return response
}
```

### 3.4 时间格式验证

```go
// plugin/ai/agent/schedule/time_validator.go

func (s *PrecheckService) validateTimeFormat(req *PrecheckRequest, resp *PrecheckResponse) {
    now := time.Now()
    
    // 检查开始时间非空
    if req.StartTime.IsZero() {
        resp.Valid = false
        resp.Errors = append(resp.Errors, PrecheckError{
            Code:    "MISSING_START_TIME",
            Message: "请选择开始时间",
            Field:   "start_time",
        })
        return
    }
    
    // 检查时间不是过去
    if req.StartTime.Before(now) {
        resp.Valid = false
        resp.Errors = append(resp.Errors, PrecheckError{
            Code:    "PAST_TIME",
            Message: "开始时间不能是过去",
            Field:   "start_time",
        })
    }
    
    // 检查时间在合理范围内（1年内）
    maxDate := now.AddDate(1, 0, 0)
    if req.StartTime.After(maxDate) {
        resp.Valid = false
        resp.Errors = append(resp.Errors, PrecheckError{
            Code:    "TIME_TOO_FAR",
            Message: "开始时间不能超过一年",
            Field:   "start_time",
        })
    }
    
    // 检查结束时间在开始时间之后
    if !req.EndTime.IsZero() && req.EndTime.Before(req.StartTime) {
        resp.Valid = false
        resp.Errors = append(resp.Errors, PrecheckError{
            Code:    "END_BEFORE_START",
            Message: "结束时间不能早于开始时间",
            Field:   "end_time",
        })
    }
}
```

### 3.5 冲突检测

```go
// plugin/ai/agent/schedule/conflict_detector.go

const (
    BufferMinutes = 15  // 日程间缓冲时间
)

func (s *PrecheckService) detectConflicts(ctx context.Context, userID int32, req *PrecheckRequest, resp *PrecheckResponse) {
    // 计算检查时间范围
    checkStart := req.StartTime.Add(-time.Duration(BufferMinutes) * time.Minute)
    checkEnd := req.EndTime.Add(time.Duration(BufferMinutes) * time.Minute)
    
    // 查询该时间段的已有日程
    existingSchedules, err := s.scheduleStore.GetSchedulesInRange(ctx, userID, checkStart, checkEnd)
    if err != nil {
        // 查询失败，添加警告但不阻止
        resp.Warnings = append(resp.Warnings, PrecheckWarning{
            Code:    "CONFLICT_CHECK_FAILED",
            Message: "无法检查时间冲突，请自行确认",
        })
        return
    }
    
    for _, existing := range existingSchedules {
        // 检查时间重叠
        if s.hasOverlap(req.StartTime, req.EndTime, existing.StartTime, existing.EndTime) {
            resp.Valid = false
            resp.Errors = append(resp.Errors, PrecheckError{
                Code:    "TIME_CONFLICT",
                Message: fmt.Sprintf("与已有日程「%s」冲突", existing.Title),
                Field:   "start_time",
            })
        } else if s.hasBufferConflict(req.StartTime, req.EndTime, existing.StartTime, existing.EndTime) {
            // 缓冲时间冲突（警告）
            resp.Warnings = append(resp.Warnings, PrecheckWarning{
                Code:    "BUFFER_CONFLICT",
                Message: fmt.Sprintf("与「%s」间隔较短（少于%d分钟）", existing.Title, BufferMinutes),
            })
        }
    }
}

func (s *PrecheckService) hasOverlap(start1, end1, start2, end2 time.Time) bool {
    return start1.Before(end2) && end1.After(start2)
}

func (s *PrecheckService) hasBufferConflict(start1, end1, start2, end2 time.Time) bool {
    buffer := time.Duration(BufferMinutes) * time.Minute
    return start1.Before(end2.Add(buffer)) && end1.Add(buffer).After(start2)
}
```

### 3.6 业务规则验证

```go
// plugin/ai/agent/schedule/business_rules.go

const (
    MaxDurationMinutes = 480  // 8小时
    WorkStartHour     = 8
    WorkEndHour       = 22
)

func (s *PrecheckService) validateBusinessRules(req *PrecheckRequest, resp *PrecheckResponse) {
    // 检查时长
    if req.Duration > MaxDurationMinutes {
        resp.Warnings = append(resp.Warnings, PrecheckWarning{
            Code:    "LONG_DURATION",
            Message: fmt.Sprintf("日程时长超过 %d 小时，请确认", MaxDurationMinutes/60),
        })
    }
    
    // 检查是否在工作时间外
    hour := req.StartTime.Hour()
    if hour < WorkStartHour || hour >= WorkEndHour {
        resp.Warnings = append(resp.Warnings, PrecheckWarning{
            Code:    "OUTSIDE_WORK_HOURS",
            Message: "该时间在常规工作时间外",
        })
    }
    
    // 检查标题长度
    if len(req.Title) > 100 {
        resp.Warnings = append(resp.Warnings, PrecheckWarning{
            Code:    "LONG_TITLE",
            Message: "标题较长，建议精简",
        })
    }
    
    // 检查是否是周末
    weekday := req.StartTime.Weekday()
    if weekday == time.Saturday || weekday == time.Sunday {
        resp.Warnings = append(resp.Warnings, PrecheckWarning{
            Code:    "WEEKEND_SCHEDULE",
            Message: "该日程安排在周末",
        })
    }
}
```

### 3.7 智能建议

```go
// plugin/ai/agent/schedule/suggestions.go

func (s *PrecheckService) generateSuggestions(ctx context.Context, userID int32, req *PrecheckRequest, resp *PrecheckResponse) {
    // 如果有时间冲突，推荐可用时段
    for _, err := range resp.Errors {
        if err.Code == "TIME_CONFLICT" {
            alternatives := s.findAlternativeSlots(ctx, userID, req)
            for _, alt := range alternatives {
                resp.Suggestions = append(resp.Suggestions, PrecheckSuggestion{
                    Type:  "alternative_time",
                    Value: alt,
                })
            }
            break
        }
    }
}

type AlternativeSlot struct {
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
    Label     string    `json:"label"`  // "同日稍后", "明天同一时间"
}

func (s *PrecheckService) findAlternativeSlots(ctx context.Context, userID int32, req *PrecheckRequest) []AlternativeSlot {
    var alternatives []AlternativeSlot
    duration := req.EndTime.Sub(req.StartTime)
    
    // 策略 1: 同日后续时段
    sameDay := req.StartTime.Add(2 * time.Hour)
    if s.isSlotAvailable(ctx, userID, sameDay, sameDay.Add(duration)) {
        alternatives = append(alternatives, AlternativeSlot{
            StartTime: sameDay,
            EndTime:   sameDay.Add(duration),
            Label:     "同日稍后",
        })
    }
    
    // 策略 2: 明天同一时间
    nextDay := req.StartTime.AddDate(0, 0, 1)
    if s.isSlotAvailable(ctx, userID, nextDay, nextDay.Add(duration)) {
        alternatives = append(alternatives, AlternativeSlot{
            StartTime: nextDay,
            EndTime:   nextDay.Add(duration),
            Label:     "明天同一时间",
        })
    }
    
    return alternatives
}

func (s *PrecheckService) isSlotAvailable(ctx context.Context, userID int32, start, end time.Time) bool {
    schedules, _ := s.scheduleStore.GetSchedulesInRange(ctx, userID, start, end)
    return len(schedules) == 0
}
```

### 3.8 API Handler

```go
// server/router/api/v1/schedule_precheck_handler.go

func (h *ScheduleHandler) HandlePrecheck(c *gin.Context) {
    var req PrecheckRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    userID := getUserID(c)
    
    // 调用预检服务
    response := h.precheckService.Precheck(c.Request.Context(), userID, &req)
    
    c.JSON(http.StatusOK, response)
}
```

---

## 4. 实现路径

### Day 1: 核心预检逻辑

- [ ] 实现 `PrecheckService`
- [ ] 时间格式验证
- [ ] 冲突检测

### Day 2: 业务规则与集成

- [ ] 业务规则验证
- [ ] 智能建议生成
- [ ] API Handler
- [ ] 单元测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/agent/schedule/precheck_service.go` | 预检服务 |
| `plugin/ai/agent/schedule/time_validator.go` | 时间验证 |
| `plugin/ai/agent/schedule/conflict_detector.go` | 冲突检测 |
| `plugin/ai/agent/schedule/business_rules.go` | 业务规则 |
| `plugin/ai/agent/schedule/suggestions.go` | 智能建议 |
| `server/router/api/v1/schedule_precheck_handler.go` | API Handler |
| `*_test.go` | 单元测试 |

### 5.2 API 文档

```yaml
# openapi.yaml
/api/v1/schedule/precheck:
  post:
    summary: 日程预检
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/PrecheckRequest'
    responses:
      200:
        description: 预检结果
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/PrecheckResponse'
```

---

## 6. 验收标准

### 6.1 功能验收

| 场景 | 期望结果 |
|:---|:---|
| 时间冲突 | valid=false, error=TIME_CONFLICT |
| 过去时间 | valid=false, error=PAST_TIME |
| 周末日程 | valid=true, warning=WEEKEND_SCHEDULE |
| 正常日程 | valid=true, no errors |

### 6.2 性能验收

- [ ] 预检延迟 < 100ms
- [ ] 支持缓存（相同请求 5 分钟内）

### 6.3 测试用例

```go
func TestPrecheckConflict(t *testing.T) {
    // 准备已有日程
    existing := &Schedule{
        StartTime: time.Now().Add(time.Hour),
        EndTime:   time.Now().Add(2 * time.Hour),
        Title:     "已有会议",
    }
    store.Create(context.Background(), 1, existing)
    
    // 测试冲突检测
    req := &PrecheckRequest{
        StartTime: time.Now().Add(90 * time.Minute),
        EndTime:   time.Now().Add(150 * time.Minute),
        Title:     "新会议",
    }
    
    resp := service.Precheck(context.Background(), 1, req)
    
    assert.False(t, resp.Valid)
    assert.Equal(t, "TIME_CONFLICT", resp.Errors[0].Code)
}
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 2 人天 | 创建成功率提升至 95%+ |
| 存储: 0 | 减少撤销/修改操作 |
| 维护: 规则可配置 | 更流畅的用户体验 |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 预检延迟 | 低 | 中 | 缓存 + 异步预检 |
| 规则过严 | 中 | 低 | 警告而非阻止 |
| 并发冲突 | 低 | 低 | 乐观锁 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 4 Day 1 | 核心预检逻辑 | TBD |
| Sprint 4 Day 2 | 业务规则与集成 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [schedule-roadmap.md](../../../research/schedule-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
