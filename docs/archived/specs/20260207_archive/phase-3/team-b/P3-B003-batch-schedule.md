# P3-B003: 批量日程支持

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 6 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 6

---

## 1. 目标与背景

### 1.1 核心目标

支持批量创建重复日程（每周例会、每日站会等），一句话创建系列日程。

### 1.2 用户价值

- "每周一开例会" 一句话搞定
- 减少重复操作 90%
- 更高效的日程管理

---

## 2. 依赖关系

- [x] P1-A004: 时间解析服务
- [x] P2-B002: 快速创建模式

---

## 3. 功能设计

### 3.1 重复规则

```
┌────────────────────────────────────────────────────────────┐
│                    重复规则类型                             │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  每日: "每天早上9点站会"                                    │
│  ├─ 工作日: "每个工作日"                                   │
│  └─ 每天: "每天"                                          │
│                                                            │
│  每周: "每周一下午2点例会"                                  │
│  ├─ 单日: "每周一"                                        │
│  └─ 多日: "每周一三五"                                     │
│                                                            │
│  每月: "每月1号汇报"                                       │
│  ├─ 固定日期: "每月15号"                                  │
│  └─ 相对日期: "每月最后一个周五"                          │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心实现

```go
// plugin/ai/agent/schedule/batch_create.go

type RecurrenceRule struct {
    Type      string   `json:"type"`       // daily, weekly, monthly
    Interval  int      `json:"interval"`   // 间隔（每2周）
    DaysOfWeek []int   `json:"days_of_week"` // 0=周日, 1=周一...
    DayOfMonth int     `json:"day_of_month"`
    EndDate   *time.Time `json:"end_date"`
    Count     int      `json:"count"`      // 重复次数
}

type BatchCreateRequest struct {
    Title      string          `json:"title"`
    StartTime  time.Time       `json:"start_time"`
    Duration   int             `json:"duration"`
    Recurrence *RecurrenceRule `json:"recurrence"`
}

func (h *BatchHandler) Parse(input string) (*BatchCreateRequest, error) {
    // 识别重复模式
    // "每周一下午2点例会" → weekly, [1], 14:00, "例会"
    
    patterns := map[string]*RecurrenceRule{
        `每天`:     {Type: "daily", Interval: 1},
        `每个工作日`: {Type: "daily", Interval: 1, DaysOfWeek: []int{1,2,3,4,5}},
        `每周[一二三四五六日]`: nil, // 动态解析
        `每月\d+[号日]`: nil,
    }
    
    // ... 解析逻辑
    return req, nil
}

func (h *BatchHandler) Generate(req *BatchCreateRequest) ([]*Schedule, error) {
    var schedules []*Schedule
    current := req.StartTime
    
    for i := 0; i < req.Recurrence.Count || req.Recurrence.EndDate != nil; i++ {
        if req.Recurrence.EndDate != nil && current.After(*req.Recurrence.EndDate) {
            break
        }
        
        schedules = append(schedules, &Schedule{
            Title:     req.Title,
            StartTime: current,
            EndTime:   current.Add(time.Duration(req.Duration) * time.Minute),
        })
        
        current = h.nextOccurrence(current, req.Recurrence)
        
        if len(schedules) >= 52 { // 最多一年
            break
        }
    }
    
    return schedules, nil
}
```

### 3.3 前端预览

```tsx
// web/src/components/schedule/BatchPreview.tsx

export function BatchPreview({ schedules, onConfirm, onCancel }: Props) {
  return (
    <div className="rounded-lg border p-4">
      <h3 className="font-medium">将创建 {schedules.length} 个日程</h3>
      
      <div className="mt-2 max-h-[300px] overflow-y-auto">
        {schedules.slice(0, 10).map((s, i) => (
          <div key={i} className="flex justify-between py-1 text-sm">
            <span>{s.title}</span>
            <span className="text-gray-500">
              {format(s.startTime, 'MM/dd EEE HH:mm')}
            </span>
          </div>
        ))}
        {schedules.length > 10 && (
          <p className="text-sm text-gray-400">... 还有 {schedules.length - 10} 个</p>
        )}
      </div>
      
      <div className="mt-4 flex gap-2">
        <Button onClick={onConfirm}>确认创建</Button>
        <Button variant="outline" onClick={onCancel}>取消</Button>
      </div>
    </div>
  );
}
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1-2 | 重复规则解析 |
| 3-4 | 批量生成逻辑 |
| 5 | 前端预览组件 |
| 6 | 测试与边界处理 |

---

## 5. 验收标准

- [ ] "每周一下午2点例会" 正确解析
- [ ] 批量预览显示正确
- [ ] 支持最多 52 周（一年）

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
