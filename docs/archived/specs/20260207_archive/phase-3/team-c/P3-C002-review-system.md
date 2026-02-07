# P3-C002: 智能回顾系统

> **状态**: 🔲 待开发  
> **优先级**: P3 (可选)  
> **投入**: 8 人天  
> **负责团队**: 团队 C  
> **Sprint**: Sprint 6

---

## 1. 目标与背景

### 1.1 核心目标

实现智能笔记回顾系统，基于遗忘曲线和重要性主动推送复习内容，帮助用户巩固知识。

### 1.2 用户价值

- 不遗忘重要笔记
- 周期性知识巩固
- 更好的学习效果

---

## 2. 依赖关系

- [x] P1-A001: 轻量记忆系统
- [x] P2-C001: 智能标签建议

---

## 3. 功能设计

### 3.1 回顾策略

```
┌────────────────────────────────────────────────────────────┐
│                    智能回顾策略                             │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  策略 1: 遗忘曲线                                          │
│  ├─ Day 1: 首次复习                                       │
│  ├─ Day 3: 第二次                                         │
│  ├─ Day 7: 第三次                                         │
│  ├─ Day 14: 第四次                                        │
│  └─ Day 30: 第五次                                        │
│                                                            │
│  策略 2: 重要性排序                                        │
│  ├─ 高重要性: 优先推送                                    │
│  ├─ 标签匹配: 与当前工作相关                              │
│  └─ 访问频率: 常看但久未看                                │
│                                                            │
│  策略 3: 时间窗口                                          │
│  ├─ 每日回顾: 早上 9:00                                   │
│  ├─ 周回顾: 周日晚上                                      │
│  └─ 月回顾: 月末                                          │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 核心实现

```go
// plugin/ai/review/service.go

type ReviewService struct {
    memoStore    MemoStore
    memoryStore  MemoryStore
    scheduler    ReviewScheduler
}

type ReviewItem struct {
    MemoID      string    `json:"memo_id"`
    Title       string    `json:"title"`
    Snippet     string    `json:"snippet"`
    Tags        []string  `json:"tags"`
    LastReview  time.Time `json:"last_review"`
    ReviewCount int       `json:"review_count"`
    NextReview  time.Time `json:"next_review"`
    Priority    float64   `json:"priority"`
}

// 遗忘曲线间隔（天）
var reviewIntervals = []int{1, 3, 7, 14, 30, 60, 120}

func (s *ReviewService) GetDueReviews(ctx context.Context, userID int32, limit int) ([]ReviewItem, error) {
    // 获取待复习的笔记
    candidates, _ := s.memoStore.GetMemosNeedReview(ctx, userID, time.Now())
    
    // 计算优先级
    for i := range candidates {
        candidates[i].Priority = s.calculatePriority(candidates[i])
    }
    
    // 排序
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Priority > candidates[j].Priority
    })
    
    if len(candidates) > limit {
        candidates = candidates[:limit]
    }
    
    return candidates, nil
}

func (s *ReviewService) calculatePriority(item ReviewItem) float64 {
    priority := 0.0
    
    // 1. 逾期天数（越久越优先）
    overdueDays := time.Since(item.NextReview).Hours() / 24
    priority += min(overdueDays * 0.1, 1.0)
    
    // 2. 重要性标签
    for _, tag := range item.Tags {
        if tag == "重要" || tag == "核心" {
            priority += 0.5
        }
    }
    
    // 3. 复习次数（新笔记优先）
    if item.ReviewCount < 3 {
        priority += 0.3
    }
    
    return priority
}

func (s *ReviewService) RecordReview(ctx context.Context, userID int32, memoID string, quality int) error {
    // quality: 1=困难, 2=一般, 3=容易
    
    reviewState, _ := s.memoryStore.GetReviewState(ctx, memoID)
    reviewState.ReviewCount++
    reviewState.LastReview = time.Now()
    
    // 根据质量调整下次复习时间
    interval := reviewIntervals[min(reviewState.ReviewCount, len(reviewIntervals)-1)]
    if quality == 1 {
        interval = interval / 2  // 困难：缩短间隔
    } else if quality == 3 {
        interval = interval * 2  // 容易：延长间隔
    }
    
    reviewState.NextReview = time.Now().AddDate(0, 0, interval)
    
    return s.memoryStore.UpdateReviewState(ctx, reviewState)
}
```

### 3.3 前端回顾界面

```tsx
// web/src/components/review/DailyReview.tsx

export function DailyReview() {
  const [items, setItems] = useState<ReviewItem[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);

  const currentItem = items[currentIndex];

  const handleReview = async (quality: number) => {
    await api.post(`/review/${currentItem.memoId}`, { quality });
    
    if (currentIndex < items.length - 1) {
      setCurrentIndex(currentIndex + 1);
    } else {
      // 完成回顾
      toast.success('今日回顾完成！');
    }
  };

  return (
    <div className="mx-auto max-w-lg p-4">
      <h2 className="text-xl font-bold">每日回顾</h2>
      <p className="text-sm text-gray-500">
        {currentIndex + 1} / {items.length}
      </p>

      <div className="mt-4 rounded-lg border p-4">
        <h3 className="font-medium">{currentItem?.title}</h3>
        <p className="mt-2 text-gray-600">{currentItem?.snippet}</p>
        <div className="mt-2 flex gap-1">
          {currentItem?.tags.map((tag) => (
            <span key={tag} className="rounded bg-gray-100 px-2 py-1 text-xs">
              #{tag}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-4 flex justify-center gap-4">
        <Button variant="outline" onClick={() => handleReview(1)}>
          困难
        </Button>
        <Button variant="outline" onClick={() => handleReview(2)}>
          一般
        </Button>
        <Button onClick={() => handleReview(3)}>
          容易
        </Button>
      </div>
    </div>
  );
}
```

---

## 4. 实现路径

| Day | 任务 |
|-----|------|
| 1-2 | 回顾数据模型与存储 |
| 3-4 | 遗忘曲线算法 |
| 5 | 优先级计算 |
| 6 | API 与推送 |
| 7-8 | 前端回顾界面 |

---

## 5. 验收标准

- [ ] 新笔记自动加入回顾队列
- [ ] 遗忘曲线正确计算下次复习时间
- [ ] 回顾质量影响后续间隔
- [ ] 每日推送待回顾列表

---

> **版本**: v1.0 | **更新时间**: 2026-01-27
