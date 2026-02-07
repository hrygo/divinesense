# P3-B002: 主动提醒系统

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 4 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 5

---

## 1. 目标与背景

### 1.1 核心目标

实现智能提醒系统，支持日程提醒、待办提醒，基于用户习惯智能调整提醒时间。

### 1.2 用户价值

- 不遗漏重要事项
- 智能提醒时机
- 多渠道通知

---

## 2. 依赖关系

- [x] P1-A004: 时间解析服务
- [x] P2-B001: 用户习惯学习

---

## 3. 功能设计

### 3.1 提醒类型

```
┌────────────────────────────────────────────────────────────┐
│                    提醒类型与触发                           │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  类型 1: 日程提醒                                          │
│  ├─ 提前 15/30/60 分钟                                    │
│  └─ 基于用户习惯调整                                       │
│                                                            │
│  类型 2: 待办提醒                                          │
│  ├─ 截止日期提醒                                           │
│  └─ 周期性复查                                             │
│                                                            │
│  类型 3: 智能提醒                                          │
│  ├─ 天气变化（户外活动）                                   │
│  └─ 交通状况（需要出行）                                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心实现

```go
// plugin/ai/reminder/service.go

type ReminderService struct {
    scheduleStore ScheduleStore
    habitService  *HabitAnalyzer
    notifier      Notifier
}

type Reminder struct {
    ID         string    `json:"id"`
    Type       string    `json:"type"`       // schedule, todo, smart
    TargetID   string    `json:"target_id"`
    TriggerAt  time.Time `json:"trigger_at"`
    Message    string    `json:"message"`
    Channels   []string  `json:"channels"`   // app, email, webhook
}

func (s *ReminderService) CreateForSchedule(ctx context.Context, schedule *Schedule, userID int32) (*Reminder, error) {
    // 获取用户习惯的提前量
    prefs, _ := s.habitService.GetUserPreferences(ctx, userID)
    leadMinutes := 15  // 默认
    if prefs != nil && prefs.ReminderLeadMin > 0 {
        leadMinutes = prefs.ReminderLeadMin
    }
    
    triggerAt := schedule.StartTime.Add(-time.Duration(leadMinutes) * time.Minute)
    
    return &Reminder{
        ID:        generateID(),
        Type:      "schedule",
        TargetID:  schedule.ID,
        TriggerAt: triggerAt,
        Message:   fmt.Sprintf("您有一个日程「%s」将在 %d 分钟后开始", schedule.Title, leadMinutes),
        Channels:  []string{"app"},
    }, nil
}

// 后台任务：检查并发送提醒
func (s *ReminderService) ProcessDueReminders(ctx context.Context) error {
    reminders, _ := s.store.GetDueReminders(ctx, time.Now())
    
    for _, r := range reminders {
        for _, channel := range r.Channels {
            s.notifier.Send(ctx, channel, r.Message)
        }
        s.store.MarkSent(ctx, r.ID)
    }
    
    return nil
}
```

### 3.3 通知渠道

```go
// plugin/ai/reminder/notifier.go

type Notifier interface {
    Send(ctx context.Context, channel string, message string) error
}

type MultiChannelNotifier struct {
    appPush   AppPushNotifier
    email     EmailNotifier
    webhook   WebhookNotifier
}

func (n *MultiChannelNotifier) Send(ctx context.Context, channel string, message string) error {
    switch channel {
    case "app":
        return n.appPush.Send(ctx, message)
    case "email":
        return n.email.Send(ctx, message)
    case "webhook":
        return n.webhook.Send(ctx, message)
    default:
        return fmt.Errorf("unknown channel: %s", channel)
    }
}
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1 | Reminder 数据模型与存储 |
| 2 | 创建逻辑（日程/待办） |
| 3 | 后台处理任务 |
| 4 | 通知渠道与测试 |

---

## 5. 验收标准

- [ ] 日程创建后自动生成提醒
- [ ] 提醒准时触发（误差 < 1分钟）
- [ ] 支持 App 内通知

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
