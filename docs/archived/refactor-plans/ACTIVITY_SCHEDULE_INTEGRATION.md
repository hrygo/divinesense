# 活动日历与日程融合 - 产品设计文档

> **文档版本**: v1.0
> **创建日期**: 2025-01-22
> **产品负责人**: Memos Team
> **设计理念**: 渐进式融合 - 保持热力图简洁，按需展示日程

---

## 📋 目录

1. [产品概述](#产品概述)
2. [核心设计理念](#核心设计理念)
3. [详细设计](#详细设计)
4. [交互设计](#交互设计)
5. [视觉设计规范](#视觉设计规范)
6. [技术实现](#技术实现)
7. [实施路线图](#实施路线图)
8. [成功指标](#成功指标)

---

## 🎯 产品概述

### 背景与问题

**当前状态**：
- **活动日历**（Activity Calendar）：位于左侧边栏，显示笔记活动热力图
- **日程管理**（Schedule）：独立页面 `/schedule`，提供时间线和日历视图

**用户痛点**：
1. ❌ **时间视角割裂**：笔记活动和日程事件在两个地方
2. ❌ **上下文缺失**：查看某天笔记时，看不到那天的日程
3. ❌ **操作冗余**：需要切换页面才能管理日程
4. ❌ **认知负担**：需要在脑海中拼凑完整的时间线

### 产品愿景

**"在统一的时间视图中，看到你的所有活动"**

将笔记活动（Activity）和日程事件（Schedule）有机融合到一个日历界面中，提供：
- 📊 **统一的时间视角**：一张日历看所有
- 🔍 **按需展开详情**：保持简洁，需要时展开
- ⚡ **快速操作入口**：直接从日历管理日程
- 🎨 **优雅的视觉层次**：清晰区分不同类型的信息

---

## 💡 核心设计理念

### 渐进式披露（Progressive Disclosure）

**原则**：信息按需展示，从简到繁

```
Level 1: 默认状态（热力图）
  ├─ 笔记活动强度（绿色渐变）
  └─ 日程提示圆点（蓝色）

Level 2: 悬停状态（快速预览）
  ├─ 日期
  ├─ 笔记数量
  └─ 日程数量

Level 3: 点击展开（详情面板）
  ├─ 日程列表（完整信息）
  ├─ 笔记列表（预览）
  ├─ 快速操作（添加日程/创建笔记）
  └─ 跳转链接（完整管理）
```

### 清晰的视觉层次

**信息优先级**：
1. **笔记活动**（主要功能）：热力图色彩强度
2. **日程提示**（辅助信息）：小巧的圆点标记
3. **详细内容**（按需展开）：侧滑/下拉面板

### 保持原有体验

**不破坏现有功能**：
- ✅ 点击日期仍然筛选笔记（核心功能）
- ✅ 热力图视觉保持不变
- ✅ 响应式布局不受影响

---

## 🎨 详细设计

### 1. 日历单元格设计

#### 1.1 默认状态（Level 1）

**无活动日期**：
```
┌─────────────┐
│             │
│    15       │  ← 日期数字
│             │
└─────────────┘
颜色: text-muted-foreground/30
```

**仅笔记活动**：
```
┌─────────────┐
│    15       │  ← 日期数字（font-semibold）
│   ████      │  ← 笔记热力图（绿色渐变）
│             │
└─────────────┘
背景: 根据活动强度（1-4 级绿色）
```

**有日程的日期**：
```
┌─────────────┐
│    15       │
│   ████      │  ← 笔记热力图
│    ●        │  ← 日程提示圆点（蓝色）
└─────────────┘
圆点: 3px, bg-blue-500, absolute bottom-1
```

**今天日期**：
```
┌─────────────┐
│◎    15       │  ← 双环标记（ring-2 ring-primary/30）
│   ████ ●    │
└─────────────┘
```

**选中日期**：
```
┌─────────────┐
│⦿    15       │  ← 粗环标记（ring-2 ring-primary）
│   ████ ●    │
└─────────────┘
```

#### 1.2 悬停状态（Level 2）

**Tooltip 设计**：
```
┌──────────────────────────┐
│  📅 2025年1月15日 周三    │
│  ──────────────────────  │
│  📝 3 条笔记              │
│  📅 2 个日程              │
│                          │
│  [点击查看详情 →]         │
└──────────────────────────┘

样式：
- 宽度: 200px
- 背景: bg-popover
- 边框: border border-border
- 阴影: shadow-md
- 箭头: 有（指向单元格）
```

**悬停交互**：
```
鼠标悬停 → {
  单元格: scale-110, shadow-md, z-20
  Tooltip: fade-in, delay-150ms
}
```

#### 1.3 点击展开（Level 3）

**详情面板 - 侧滑式（桌面端）**：

```
屏幕布局：
┌────────────┬─────────────────────┐
│            │                     │
│  左侧边栏   │   主内容区           │
│            │   ┌───────────────┐ │
│  ┌──────┐  │   │  详情面板      │ │
│  │日历  │  │   │  (侧滑展开)   │ │
│  │      │  │   │               │ │
│  └──────┘  │   │  日程列表      │ │
│            │   │  笔记列表      │ │
│            │   │  快速操作      │ │
└────────────┴───┴───────────────┘

面板尺寸：
- 宽度: 400px
- 位置: 主内容区左侧
- 动画: slide-in-from-left-4, fade-in
- 背景: bg-background/95 backdrop-blur-md
```

**详情面板内容**：
```
┌────────────────────────────────┐
│ 2025年1月15日 周三         [×] │  ← 头部（可关闭）
│ ───────────────────────────── │
│                                │
│ 📅 日程 (2)                    │
│ ┌──────────────────────────┐  │
│ │ ━━━━━━━━━━━━━━━━━━━━━━ │  │
│ │                            │  │
│ │ 🕐 10:00 - 11:00          │  │
│ │    团队周会                │  │
│ │    📍 会议室 A             │  │
│ │                            │  │
│ │ ━━━━━━━━━━━━━━━━━━━━━━ │  │
│ │                            │  │
│ │ 🕐 14:00 - 15:30          │  │
│ │    产品评审                │  │
│ │    📍 线上会议             │  │
│ │                            │  │
│ └──────────────────────────┘  │
│ [+ 添加日程]                   │
│                                │
│ 📝 笔记 (3)                    │
│ ┌──────────────────────────┐  │
│ │ • 会议纪要 - 关于...      │  │
│ │ • 产品讨论 - 新功能...    │  │
│ │ • 待办事项 - 1. ...      │  │
│ └──────────────────────────┘  │
│ [+ 创建笔记]                   │
│                                │
│ ───────────────────────────── │
│ [查看全部日程 →]  [筛选笔记 →] │
└────────────────────────────────┘
```

**详情面板 - 抽屉式（移动端）**：

```
屏幕布局：
┌──────────────┐
│   顶部栏      │
│ ───────────── │
│              │
│  日历内容     │
│              │
│  ═══════════  ← 半透明遮罩
│ ┌──────────┐ │
│ │ 详情抽屉  │ │ ← 从底部滑出
│ │          │ │
│ │ 日程列表  │ │
│ │ 笔记列表  │ │
│ │          │ │
│ └──────────┘ │
└──────────────┘

抽屉尺寸：
- 高度: 70vh
- 位置: fixed bottom-0
- 动画: slide-in-from-bottom
- 圆角: rounded-t-2xl
```

---

### 2. 组件结构设计

#### 2.1 组件树

```
StatisticsView
└─ MonthCalendar
   ├─ MonthNavigator (已有)
   └─ CalendarGrid
      └─ CalendarCell (扩展)
         ├─ ActivityHeatmap (已有)
         ├─ ScheduleIndicator (新增)
         └─ Tooltip (扩展)
            └─ QuickPreview

DayDetailPanel (新增)
├─ DayDetailHeader
├─ ScheduleList
│  ├─ ScheduleItem
│  └─ AddScheduleButton
├─ MemoList
│  ├─ MemoItem
│  └─ CreateMemoButton
└─ ActionFooter
```

#### 2.2 数据流

```
MonthCalendar
  ├─ useMonthData() (Hook)
  │   ├─ useActivityStats() (已有)
  │   └─ useSchedulesByMonth() (新增)
  │
  ├─ groupSchedulesByDate() (Utils)
  │   └─ { "2025-01-15": [schedule1, schedule2], ... }
  │
  └─ render cells with schedules

DayDetailPanel
  ├─ useDayData(date) (Hook)
  │   ├─ useMemosByDate(date)
  │   └─ useSchedulesByDate(date)
  │
  └─ render lists
```

---

## 🔄 交互设计

### 场景 1: 快速预览（Hover）

```
用户操作流：
1. 鼠标悬停日期单元格
2. 延迟 150ms 显示 Tooltip
3. 显示笔记数量 + 日程数量
4. 鼠标移开 → Tooltip 淡出

技术细节：
- 事件: onMouseEnter, onMouseLeave
- 防抖: 150ms delay
- 动画: fade-in 150ms
```

### 场景 2: 查看详情（Click）

```
桌面端流程：
1. 点击日期单元格
2. 详情面板从左侧滑入（300ms）
3. 加载该日的日程和笔记数据
4. 渲染列表（骨架屏 → 实际内容）
5. 用户可滚动查看
6. 点击 [×] 或面板外区域 → 面板滑出

移动端流程：
1. 点击日期单元格
2. 半透明遮罩淡入
3. 详情抽屉从底部滑入（300ms）
4. 显示内容
5. 下拉抽屉或点击遮罩 → 关闭
```

### 场景 3: 快速添加日程

```
用户操作流：
1. 在详情面板点击 [+ 添加日程]
2. 打开简化对话框（复用 ScheduleInput）
3. 自动填充选中日期
4. 用户输入：
   - 时间：默认 10:00
   - 标题：必填
   - 描述：可选
5. 点击保存 → 日程添加成功
6. 面板数据刷新（乐观更新）
7. 日历单元格圆点更新

对话框设计：
┌──────────────────────────┐
│  添加日程          [×]   │
│  ──────────────────────  │
│  📅 日期: 2025-01-15     │
│  🕐 时间: [10:00]        │
│  📝 标题: ______________ │
│  📄 描述: ______________ │
│             ______________ │
│                          │
│  [取消]        [保存]    │
└──────────────────────────┘
```

### 场景 4: 筛选笔记（保留原有功能）

```
用户操作流：
1. 点击有笔记活动的日期
2. 如果该日没有日程 → 直接筛选笔记（原有行为）
3. 如果该日有日程 → 显示详情面板
4. 在详情面板点击 [筛选笔记] → 跳转到笔记列表（筛选状态）
```

### 场景 5: 跳转到日程管理

```
用户操作流：
1. 在详情面板点击 [查看全部日程 →]
2. 或点击面板中的某个日程项
3. 跳转到 /schedule 页面
4. 自动选中对应日期
5. 滚动到该日程位置

URL 设计：
/schedule?date=2025-01-15&focus=schedule-123
```

---

## 🎨 视觉设计规范

### 颜色系统

#### 笔记活动（Activity）- 绿色系

```css
/* 热力图渐变（基于活动强度 1-4） */
--activity-level-0: transparent;        /* 无活动 */
--activity-level-1: #dcfce7;  /* 浅绿 - 1-2 条笔记 */
--activity-level-2: #86efac;  /* 中绿 - 3-5 条笔记 */
--activity-level-3: #22c55e;  /* 深绿 - 6-10 条笔记 */
--activity-level-4: #15803d;  /* 浓绿 - 10+ 条笔记 */

/* 选中/今天状态 */
--activity-today-ring: ring-2 ring-primary/30 ring-offset-1;
--activity-selected-ring: ring-2 ring-primary ring-offset-1;
```

#### 日程事件（Schedule）- 蓝色系

```css
/* 日程指示器 */
--schedule-indicator: #3b82f6;           /* 蓝色圆点 */
--schedule-indicator-hover: #2563eb;     /* 悬停深蓝 */
--schedule-high-priority: #ef4444;       /* 紧急红色 */

/* 日程卡片 */
--schedule-card-bg: bg-blue-50 dark:bg-blue-900/20;
--schedule-card-border: border-blue-200 dark:border-blue-800;
```

#### 交互反馈

```css
/* 悬停状态 */
--cell-hover-bg: hover:bg-muted/50;
--cell-hover-scale: hover:scale-110;
--cell-hover-shadow: hover:shadow-md;
--cell-hover-z: hover:z-20;

/* Tooltip */
--tooltip-bg: bg-popover;
--tooltip-border: border border-border;
--tooltip-shadow: shadow-md;
--tooltip-text: text-popover-foreground;

/* 详情面板 */
--panel-bg: bg-background/95 backdrop-blur-md;
--panel-border: border border-border/50;
--panel-shadow: shadow-xl;
```

### 字体系统

```css
/* 日期数字 */
.cell-date {
  font-size: 0.875rem;      /* 14px - default */
  font-weight: font-medium;  /* 500 */
}

.cell-date-today {
  font-weight: font-semibold;  /* 600 */
}

.cell-date-selected {
  font-weight: font-bold;     /* 700 */
}

/* Tooltip 文本 */
.tooltip-title {
  font-size: 0.875rem;      /* 14px */
  font-weight: font-semibold;
  color: text-foreground;
}

.tooltip-content {
  font-size: 0.8125rem;     /* 13px */
  color: text-muted-foreground;
}

/* 详情面板 */
.panel-header {
  font-size: 1.125rem;      /* 18px */
  font-weight: font-semibold;
}

.panel-section-title {
  font-size: 0.875rem;      /* 14px */
  font-weight: font-semibold;
  color: text-muted-foreground;
}
```

### 间距系统

```css
/* 单元格 */
.cell-gap: gap-1;           /* 4px - mobile */
.cell-gap-desktop: gap-2;   /* 8px - desktop */

.cell-size: w-9 h-9;        /* 36px - mobile */
.cell-size-desktop: w-10 h-10; /* 40px - desktop */

/* 详情面板 */
.panel-padding: p-4;        /* 16px */
.panel-gap: space-y-4;      /* 16px */

.section-gap: space-y-2;    /* 8px - between items */
```

### 动画规范

```css
/* 过渡时间 */
.transition-fast: 150ms;   /* Tooltip */
.transition-normal: 300ms; /* 面板滑入/滑出 */
.transition-slow: 500ms;   /* 数据加载 */

/* 缓动函数 */
.ease-bounce: cubic-bezier(0.34, 1.56, 0.64, 1);  /* 弹跳 */
.ease-smooth: cubic-bezier(0.4, 0, 0.2, 1);       /* 平滑 */
```

### 响应式断点

```css
/* Mobile */
@media (max-width: 768px) {
  .cell-size: w-9 h-9;
  .cell-gap: gap-1;
  .panel-mode: drawer;
  .panel-width: 100%;
}

/* Desktop */
@media (min-width: 769px) {
  .cell-size: w-10 h-10;
  .cell-gap: gap-2;
  .panel-mode: sidebar;
  .panel-width: 400px;
}
```

---

## 💻 技术实现

### 1. 数据结构设计

#### 1.1 月度数据聚合

```typescript
interface MonthData {
  month: string; // "2025-01"

  // 笔记活动数据（已有）
  activityStats: Record<string, number>; // { "2025-01-15": 3, ... }

  // 日程数据（新增）
  schedulesByDate: Record<string, ScheduleSummary[]>; // { "2025-01-15": [...], ... }

  // 统计信息
  totalMemos: number;
  totalSchedules: number;
  maxMemoCount: number;
}

interface ScheduleSummary {
  uid: string;
  title: string;
  startTs: number; // Unix timestamp
  endTs: number;
  allDay: boolean;
  location?: string;
  status: 'ACTIVE' | 'CANCELLED';
  priority?: 'HIGH' | 'MEDIUM' | 'LOW';
}
```

#### 1.2 日期单元格数据

```typescript
interface CalendarDayCell {
  date: string; // "2025-01-15"
  label: string; // "15"

  // 笔记活动（已有）
  count: number;
  maxCount: number;

  // 日程信息（新增）
  scheduleCount: number;
  schedules: ScheduleSummary[];

  // 状态
  isCurrentMonth: boolean;
  isToday: boolean;
  isSelected: boolean;

  // 交互状态
  isHovered: boolean;
}
```

### 2. Hooks 设计

#### 2.1 useMonthData Hook

```typescript
function useMonthData(month: string): MonthData {
  // 并行获取笔记活动和日程数据
  const { data: activityStats } = useActivityStats(month);
  const { data: schedules } = useSchedulesByMonth(month);

  // 聚合日程到日期
  const schedulesByDate = useMemo(() => {
    if (!schedules) return {};

    return groupSchedulesByDate(schedules, month);
  }, [schedules, month]);

  // 计算统计信息
  const maxMemoCount = useMemo(() => {
    const counts = Object.values(activityStats || {});
    return Math.max(...counts, 1);
  }, [activityStats]);

  return {
    month,
    activityStats: activityStats || {},
    schedulesByDate,
    totalMemos: Object.values(activityStats || {}).reduce((a, b) => a + b, 0),
    totalSchedules: Object.values(schedulesByDate).reduce((a, b) => a + b.length, 0),
    maxMemoCount,
  };
}
```

#### 2.2 useDayDetail Hook

```typescript
function useDayDetail(date: string) {
  // 获取指定日期的笔记和日程
  const { data: memos, isLoading: memosLoading } = useMemosByDate(date);
  const { data: schedules, isLoading: schedulesLoading } = useSchedulesByDate(date);

  return {
    memos: memos || [],
    schedules: schedules || [],
    isLoading: memosLoading || schedulesLoading,
    isEmpty: !memos?.length && !schedules?.length,
  };
}
```

### 3. 组件实现

#### 3.1 扩展 CalendarCell

```typescript
// CalendarCell.tsx

interface CalendarCellProps {
  day: CalendarDayCell;
  maxCount: number;
  tooltipText: string;
  onClick?: (date: string) => void;
  size?: CalendarSize;
  schedules?: ScheduleSummary[]; // 新增
}

export const CalendarCell = memo((props: CalendarCellProps) => {
  const { day, maxCount, tooltipText, onClick, schedules = [], size = "default" } = props;

  const handleClick = () => {
    if (day.count > 0 && onClick) {
      onClick(day.date);
    }
  };

  const scheduleCount = schedules.length;
  const hasSchedule = scheduleCount > 0;

  // 渲染日程指示器圆点
  const ScheduleIndicator = hasSchedule ? (
    <div className="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-blue-500" />
  ) : null;

  // 扩展 Tooltip 内容
  const enhancedTooltipText = hasSchedule
    ? `${tooltipText}\n📅 ${scheduleCount} 个日程`
    : tooltipText;

  // ... 渲染逻辑
});
```

#### 3.2 新建 DayDetailPanel

```typescript
// DayDetailPanel.tsx

interface DayDetailPanelProps {
  date: string;
  isOpen: boolean;
  onClose: () => void;
  onNavigateToSchedule?: (date: string) => void;
}

export const DayDetailPanel: React.FC<DayDetailPanelProps> = ({
  date,
  isOpen,
  onClose,
  onNavigateToSchedule,
}) => {
  const { t } = useTranslate();
  const { memos, schedules, isLoading } = useDayDetail(date);
  const queryClient = useQueryClient();

  // 快速添加日程
  const handleAddSchedule = () => {
    // 打开 ScheduleInput 对话框
    // 自动填充日期
  };

  // 创建笔记
  const handleCreateMemo = () => {
    // 跳转到创建笔记页面
    // 预填充日期标签
  };

  // 跳转到日程管理
  const handleNavigateToSchedule = () => {
    onNavigateToSchedule?.(date);
  };

  return (
    <>
      {/* 遮罩（移动端） */}
      {isOpen && <Overlay onClick={onClose} />}

      {/* 面板 */}
      <Panel isOpen={isOpen} onClose={onClose}>
        <PanelHeader date={date} onClose={onClose} />

        {isLoading ? (
          <SkeletonLoader />
        ) : (
          <>
            {/* 日程列表 */}
            <ScheduleList
              schedules={schedules}
              onAddSchedule={handleAddSchedule}
            />

            {/* 笔记列表 */}
            <MemoList
              memos={memos}
              onCreateMemo={handleCreateMemo}
            />

            {/* 底部操作 */}
            <ActionFooter
              onNavigateToSchedule={handleNavigateToSchedule}
              scheduleCount={schedules.length}
              memoCount={memos.length}
            />
          </>
        )}
      </Panel>
    </>
  );
};
```

### 4. 性能优化

#### 4.1 虚拟滚动

```typescript
// 使用 react-window 处理大量日程

import { FixedSizeList } from 'react-window';

const ScheduleList = ({ schedules }: { schedules: ScheduleSummary[] }) => {
  return (
    <FixedSizeList
      height={400}
      itemCount={schedules.length}
      itemSize={80}
      width="100%"
    >
      {({ index, style }) => (
        <div style={style}>
          <ScheduleItem schedule={schedules[index]} />
        </div>
      )}
    </FixedSizeList>
  );
};
```

#### 4.2 数据缓存

```typescript
// React Query 缓存策略

const useSchedulesByMonth = (month: string) => {
  return useQuery({
    queryKey: ['schedules', 'month', month],
    queryFn: () => fetchSchedulesByMonth(month),
    staleTime: 5 * 60 * 1000, // 5 分钟
    cacheTime: 10 * 60 * 1000, // 10 分钟
  });
};

// 乐观更新

const mutation = useMutation({
  mutationFn: addSchedule,
  onMutate: async (newSchedule) => {
    // 取消正在进行的查询
    await queryClient.cancelQueries(['schedules']);

    // 快照当前值
    const previousSchedules = queryClient.getQueryData(['schedules']);

    // 乐观更新
    queryClient.setQueryData(['schedules'], (old) => [...old, newSchedule]);

    return { previousSchedules };
  },
  onError: (err, newSchedule, context) => {
    // 回滚
    queryClient.setQueryData(['schedules'], context.previousSchedules);
  },
});
```

#### 4.3 懒加载

```typescript
// 详情面板数据懒加载

const DayDetailPanel = ({ date }: { date: string }) => {
  const [shouldLoad, setShouldLoad] = useState(false);

  // 只在面板打开时加载数据
  const { data } = useDayDetail(date, {
    enabled: shouldLoad,
  });

  useEffect(() => {
    if (isOpen) {
      setShouldLoad(true);
    }
  }, [isOpen]);

  // ...
};
```

---

## 🚀 实施路线图

### Phase 1: MVP（最小可行产品）⏱️ 1-2 周

**目标**: 在活动日历上显示日程圆点提示

#### 任务清单

- [ ] **后端 API**
  - [ ] 扩展 `useSchedulesByMonth` Hook
  - [ ] 实现月度日程数据查询
  - [ ] 添加数据聚合逻辑

- [ ] **CalendarCell 扩展**
  - [ ] 添加 `schedules` prop
  - [ ] 实现日程指示器圆点
  - [ ] 扩展 Tooltip 显示日程数量

- [ ] **MonthCalendar 更新**
  - [ ] 集成 `useMonthData` Hook
  - [ ] 传递日程数据到单元格
  - [ ] 更新类型定义

- [ ] **测试**
  - [ ] 单元测试（Hooks, 组件）
  - [ ] 集成测试（数据流）
  - [ ] 手动测试（交互）

#### 验收标准

- ✅ 日历单元格显示日程圆点（蓝色小点）
- ✅ 悬停显示 Tooltip："📝 X 条笔记\n📅 X 个日程"
- ✅ 点击日期仍然筛选笔记（原有功能不受影响）
- ✅ 性能：渲染 30 天日历 < 100ms

---

### Phase 2: 详情面板 ⏱️ 2-3 周

**目标**: 点击日期展开详情面板

#### 任务清单

- [ ] **DayDetailPanel 组件**
  - [ ] 实现面板布局（侧滑/抽屉）
  - [ ] DayDetailHeader（日期 + 关闭按钮）
  - [ ] ScheduleList（日程列表组件）
  - [ ] MemoList（笔记列表组件）
  - [ ] ActionFooter（底部操作栏）

- [ ] **数据 Hooks**
  - [ ] `useDayDetail` Hook
  - [ ] `useMemosByDate` Hook
  - [ ] `useSchedulesByDate` Hook

- [ ] **交互功能**
  - [ ] 点击日期展开面板
  - [ ] 点击遮罩/外部关闭面板
  - [ ] 面板动画（滑入/滑出）

- [ ] **快速操作**
  - [ ] [+ 添加日程] 按钮
  - [ ] 集成 ScheduleInput 对话框
  - [ ] 自动填充选中日期

- [ ] **响应式设计**
  - [ ] 桌面端：侧滑面板（400px）
  - [ ] 移动端：底部抽屉（70vh）

- [ ] **测试**
  - [ ] 单元测试（所有组件）
  - [ ] 集成测试（交互流程）
  - [ ] E2E 测试（关键路径）

#### 验收标准

- ✅ 点击日期 → 详情面板滑入（300ms 动画）
- ✅ 面板显示该日的日程和笔记
- ✅ [+ 添加日程] 打开对话框（日期预填充）
- ✅ 桌面端侧滑 / 移动端抽屉
- ✅ 性能：面板打开 < 300ms，数据加载 < 500ms

---

### Phase 3: 深度集成 ⏱️ 3-4 周

**目标**: 完整的工作流集成

#### 任务清单

- [ ] **高级交互**
  - [ ] 从日历拖拽创建日程
  - [ ] 日程与笔记关联
  - [ ] 批量操作（多选日程）

- [ ] **智能功能**
  - [ ] AI 建议最佳时间
  - [ ] 检测时间冲突
  - [ ] 智能提醒

- [ ] **优化体验**
  - [ ] 骨架屏加载
  - [ ] 乐观更新
  - [ ] 错误处理

- [ ] **扩展功能**
  - [ ] 周视图/月视图切换
  - [ ] 日程颜色标签
  - [ ] 导出日历（.ics 格式）

- [ ] **测试**
  - [ ] 完整的 E2E 测试套件
  - [ ] 性能测试
  - [ ] 可访问性测试（WCAG 2.1 AA）

#### 验收标准

- ✅ 拖拽创建日程流畅
- ✅ 日程与笔记可关联
- ✅ 冲突检测准确
- ✅ 所有交互有加载状态
- ✅ 无障碍访问（键盘导航）

---

## 📊 成功指标

### 用户行为指标

| 指标 | 基线 | 目标 | 测量方法 |
|------|------|------|----------|
| 日历使用率 | - | +50% | 日活用户中打开日历的比例 |
| 日程创建效率 | 基准 | -30% | 从想法到日程创建的时间 |
| 详情面板使用率 | - | >40% | 打开详情面板的用户比例 |
| 笔记-日程关联率 | 0% | >20% | 创建关联的笔记/日程比例 |

### 技术性能指标

| 指标 | 目标 | 测量方法 |
|------|------|----------|
| 日历渲染时间 | <100ms | Performance API |
| 面板打开时间 | <300ms | Performance API |
| 数据加载时间 | <500ms | React Query DevTools |
| 内存占用增长 | <10MB | Chrome DevTools |

### 用户满意度指标

| 指标 | 目标 | 测量方法 |
|------|------|----------|
| NPS 分数 | >50 | 季度用户调研 |
| 功能满意度 | >4.0/5.0 | 应用内反馈 |
| Bug 报告率 | <2% | 用户反馈分析 |

---

## 🎨 设计交付物

### 1. Figma 原型

**需要创建的设计稿**：
- [ ] 活动日历 - 默认状态（所有日期类型）
- [ ] Tooltip - 悬停预览
- [ ] 详情面板 - 桌面端（侧滑）
- [ ] 详情面板 - 移动端（抽屉）
- [ ] 交互流程图（完整用户旅程）
- [ ] 组件规范（尺寸、颜色、间距）

### 2. 交互原型

**使用工具**：
- Figma Prototype（桌面端）
- Figma Mirror（移动端）

**关键交互**：
- 悬停显示 Tooltip
- 点击展开面板
- 快速添加日程
- 筛选笔记

### 3. 设计系统文档

**需要更新的内容**：
- [ ] 颜色系统（添加日程相关颜色）
- [ ] 组件库（CalendarCell, DayDetailPanel）
- [ ] 动画规范（过渡时间、缓动函数）
- [ ] 响应式断点

---

## 📝 附录

### A. 技术术语表

| 术语 | 解释 |
|------|------|
| Activity Calendar | 活动日历，显示笔记活动热力图 |
| Schedule | 日程，时间管理事件 |
| Heatmap | 热力图，用颜色强度表示数据密度 |
| Progressive Disclosure | 渐进式披露，按需展示信息 |
| Skeleton Screen | 骨架屏，加载占位符 |
| Optimistic Update | 乐观更新，假设成功立即更新 UI |

### B. 相关文档

- [ActivityCalendar 组件文档](./ACTIVITY_CALENDAR.md)
- [Schedule 功能设计](./SCHEDULE_FEATURE.md)
- [Memos 产品架构](./PRODUCT_ARCHITECTURE.md)

### C. 变更日志

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2025-01-22 | 初始版本 | Memos Team |

---

## ✅ 下一步行动

1. **评审文档** - 产品、设计、技术团队评审
2. **创建原型** - Figma 设计稿
3. **技术评估** - 确认技术可行性
4. **开始开发** - 从 Phase 1 MVP 开始

---

**文档状态**: ✅ 已完成
**最后更新**: 2025-01-22
**负责人**: Memos Product Team
