# P2-B004: Generative UI 增强

> **状态**: 🔲 待开发  
> **优先级**: P2 (增强)  
> **投入**: 4 人天  
> **负责团队**: 团队 B  
> **Sprint**: Sprint 4

---

## 1. 目标与背景

### 1.1 核心目标

增强 Generative UI 能力，让 AI 能够根据上下文动态生成丰富的交互组件（日程卡片、确认按钮、时间选择器等），而非纯文本回复。

### 1.2 用户价值

- 从"读文字"到"点击操作"
- 减少输入，提升效率
- 更直观的交互体验

### 1.3 技术价值

- 组件化响应标准
- 前后端解耦
- 可扩展的 UI 类型系统

---

## 2. 依赖关系

### 2.1 前置依赖

- [x] P1-B001: 工具可靠性增强（工具调用稳定）
- [x] P2-B002: 快速创建模式（预览卡片基础）

### 2.2 并行依赖

- P2-B003: 预检 API（可并行）

### 2.3 后续依赖

- P3-B001: 预测性交互（主动推送 UI）
- P3-B002: 提醒系统（提醒卡片）

---

## 3. 功能设计

### 3.1 架构图

```
                    Generative UI 架构
┌────────────────────────────────────────────────────────────┐
│                                                            │
│   Agent 响应                                                │
│       │                                                    │
│       ▼                                                    │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              UI Component Registry                   │ │
│   │                                                      │ │
│   │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │ │
│   │  │ TextMsg │ │Schedule │ │ Confirm │ │ TimePick│   │ │
│   │  │         │ │  Card   │ │  Dialog │ │   er    │   │ │
│   │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │ │
│   │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │ │
│   │  │  Memo   │ │ Options │ │Progress │ │  Error  │   │ │
│   │  │  Card   │ │  List   │ │   Bar   │ │  Alert  │   │ │
│   │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │ │
│   └─────────────────────────────────────────────────────┘ │
│       │                                                    │
│       ▼                                                    │
│   ┌─────────────────────────────────────────────────────┐ │
│   │              Frontend Renderer                       │ │
│   │                                                      │ │
│   │  switch(component.type) {                           │ │
│   │    case "schedule_card": <ScheduleCard />           │ │
│   │    case "confirm_dialog": <ConfirmDialog />         │ │
│   │    case "options_list": <OptionsList />             │ │
│   │    ...                                              │ │
│   │  }                                                  │ │
│   └─────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 3.2 UI 组件类型定义

```go
// plugin/ai/genui/types.go

type ComponentType string

const (
    ComponentText          ComponentType = "text"
    ComponentScheduleCard  ComponentType = "schedule_card"
    ComponentMemoCard      ComponentType = "memo_card"
    ComponentConfirmDialog ComponentType = "confirm_dialog"
    ComponentOptionsList   ComponentType = "options_list"
    ComponentTimePicker    ComponentType = "time_picker"
    ComponentProgressBar   ComponentType = "progress_bar"
    ComponentErrorAlert    ComponentType = "error_alert"
    ComponentSuccessBanner ComponentType = "success_banner"
)

type UIComponent struct {
    Type    ComponentType `json:"type"`
    ID      string        `json:"id"`
    Data    any           `json:"data"`
    Actions []UIAction    `json:"actions,omitempty"`
}

type UIAction struct {
    ID      string `json:"id"`
    Type    string `json:"type"`     // "button", "link", "submit"
    Label   string `json:"label"`
    Style   string `json:"style"`    // "primary", "secondary", "danger"
    Payload any    `json:"payload,omitempty"`
}

// Agent 响应增强
type AgentResponse struct {
    Text       string        `json:"text,omitempty"`        // 纯文本
    Components []UIComponent `json:"components,omitempty"`  // UI 组件
    Streaming  bool          `json:"streaming,omitempty"`   // 是否流式
}
```

### 3.3 日程卡片组件

```go
// plugin/ai/genui/schedule_card.go

type ScheduleCardData struct {
    ID          string    `json:"id,omitempty"`
    Title       string    `json:"title"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Duration    int       `json:"duration"`
    Location    string    `json:"location,omitempty"`
    Description string    `json:"description,omitempty"`
    Status      string    `json:"status"`  // "preview", "confirmed", "conflict"
}

func NewScheduleCard(schedule *ScheduleRequest, status string) *UIComponent {
    cardData := &ScheduleCardData{
        Title:     schedule.Title,
        StartTime: schedule.StartTime,
        EndTime:   schedule.EndTime,
        Duration:  schedule.Duration,
        Location:  schedule.Location,
        Status:    status,
    }
    
    actions := []UIAction{}
    
    if status == "preview" {
        actions = append(actions,
            UIAction{
                ID:    "confirm",
                Type:  "button",
                Label: "确认创建",
                Style: "primary",
                Payload: schedule,
            },
            UIAction{
                ID:    "edit",
                Type:  "button",
                Label: "修改",
                Style: "secondary",
            },
            UIAction{
                ID:    "cancel",
                Type:  "button",
                Label: "取消",
                Style: "ghost",
            },
        )
    }
    
    return &UIComponent{
        Type:    ComponentScheduleCard,
        ID:      generateID(),
        Data:    cardData,
        Actions: actions,
    }
}
```

### 3.4 确认对话框组件

```go
// plugin/ai/genui/confirm_dialog.go

type ConfirmDialogData struct {
    Title       string `json:"title"`
    Message     string `json:"message"`
    ConfirmText string `json:"confirm_text"`
    CancelText  string `json:"cancel_text"`
    Danger      bool   `json:"danger"`  // 危险操作（红色按钮）
}

func NewConfirmDialog(title, message string, payload any, danger bool) *UIComponent {
    return &UIComponent{
        Type: ComponentConfirmDialog,
        ID:   generateID(),
        Data: &ConfirmDialogData{
            Title:       title,
            Message:     message,
            ConfirmText: "确认",
            CancelText:  "取消",
            Danger:      danger,
        },
        Actions: []UIAction{
            {
                ID:      "confirm",
                Type:    "button",
                Label:   "确认",
                Style:   ternary(danger, "danger", "primary"),
                Payload: payload,
            },
            {
                ID:    "cancel",
                Type:  "button",
                Label: "取消",
                Style: "secondary",
            },
        },
    }
}
```

### 3.5 选项列表组件

```go
// plugin/ai/genui/options_list.go

type OptionsListData struct {
    Title       string       `json:"title"`
    Description string       `json:"description,omitempty"`
    Options     []OptionItem `json:"options"`
    MultiSelect bool         `json:"multi_select"`
}

type OptionItem struct {
    ID          string `json:"id"`
    Label       string `json:"label"`
    Description string `json:"description,omitempty"`
    Icon        string `json:"icon,omitempty"`
    Selected    bool   `json:"selected"`
}

func NewOptionsList(title string, options []OptionItem, multiSelect bool) *UIComponent {
    return &UIComponent{
        Type: ComponentOptionsList,
        ID:   generateID(),
        Data: &OptionsListData{
            Title:       title,
            Options:     options,
            MultiSelect: multiSelect,
        },
        Actions: []UIAction{
            {
                ID:    "submit",
                Type:  "submit",
                Label: "确定",
                Style: "primary",
            },
        },
    }
}

// 使用示例：时间段选择
func NewTimeSlotPicker(slots []time.Time) *UIComponent {
    options := make([]OptionItem, len(slots))
    for i, slot := range slots {
        options[i] = OptionItem{
            ID:    fmt.Sprintf("slot_%d", i),
            Label: slot.Format("15:04"),
            Description: slot.Format("01月02日"),
        }
    }
    
    return NewOptionsList("请选择时间", options, false)
}
```

### 3.6 时间选择器组件

```go
// plugin/ai/genui/time_picker.go

type TimePickerData struct {
    Label       string    `json:"label"`
    DefaultDate time.Time `json:"default_date,omitempty"`
    MinDate     time.Time `json:"min_date,omitempty"`
    MaxDate     time.Time `json:"max_date,omitempty"`
    ShowTime    bool      `json:"show_time"`
}

func NewTimePicker(label string, defaultDate time.Time) *UIComponent {
    return &UIComponent{
        Type: ComponentTimePicker,
        ID:   generateID(),
        Data: &TimePickerData{
            Label:       label,
            DefaultDate: defaultDate,
            MinDate:     time.Now(),
            MaxDate:     time.Now().AddDate(1, 0, 0),
            ShowTime:    true,
        },
        Actions: []UIAction{
            {
                ID:    "submit",
                Type:  "submit",
                Label: "确定",
                Style: "primary",
            },
        },
    }
}
```

### 3.7 UI 组件生成器

```go
// plugin/ai/genui/generator.go

type UIGenerator struct {
    registry map[string]ComponentBuilder
}

type ComponentBuilder func(data any) *UIComponent

func NewUIGenerator() *UIGenerator {
    return &UIGenerator{
        registry: make(map[string]ComponentBuilder),
    }
}

func (g *UIGenerator) Register(name string, builder ComponentBuilder) {
    g.registry[name] = builder
}

// 根据 Agent 输出决定生成什么 UI
func (g *UIGenerator) GenerateFromAgentOutput(output *AgentOutput) *AgentResponse {
    response := &AgentResponse{}
    
    switch output.Type {
    case OutputTypeSchedulePreview:
        // 日程预览 → 日程卡片
        card := NewScheduleCard(output.Schedule, "preview")
        response.Components = append(response.Components, *card)
        response.Text = "已为您解析日程，请确认："
        
    case OutputTypeConfirmation:
        // 需要确认 → 确认对话框
        dialog := NewConfirmDialog(
            output.Title,
            output.Message,
            output.Payload,
            output.Danger,
        )
        response.Components = append(response.Components, *dialog)
        
    case OutputTypeTimeAmbiguous:
        // 时间不明确 → 时间选择器
        picker := NewTimePicker("请选择具体时间", output.SuggestedTime)
        response.Components = append(response.Components, *picker)
        response.Text = "请选择具体时间："
        
    case OutputTypeMultipleOptions:
        // 多选项 → 选项列表
        list := NewOptionsList(output.Title, output.Options, false)
        response.Components = append(response.Components, *list)
        
    case OutputTypeSuccess:
        // 成功 → 成功横幅
        banner := &UIComponent{
            Type: ComponentSuccessBanner,
            Data: map[string]string{
                "message": output.Message,
            },
        }
        response.Components = append(response.Components, *banner)
        
    case OutputTypeError:
        // 错误 → 错误提示
        alert := &UIComponent{
            Type: ComponentErrorAlert,
            Data: map[string]string{
                "message": output.Message,
            },
        }
        response.Components = append(response.Components, *alert)
        
    default:
        // 默认纯文本
        response.Text = output.Text
    }
    
    return response
}
```

### 3.8 前端组件渲染器

```tsx
// web/src/components/ai/UIComponentRenderer.tsx

import { ScheduleCard } from './ScheduleCard';
import { ConfirmDialog } from './ConfirmDialog';
import { OptionsList } from './OptionsList';
import { TimePicker } from './TimePicker';
import { SuccessBanner } from './SuccessBanner';
import { ErrorAlert } from './ErrorAlert';

interface UIComponentRendererProps {
  component: UIComponent;
  onAction: (actionId: string, payload?: any) => void;
}

export function UIComponentRenderer({ component, onAction }: UIComponentRendererProps) {
  const handleAction = (actionId: string) => {
    const action = component.actions?.find(a => a.id === actionId);
    onAction(actionId, action?.payload);
  };

  switch (component.type) {
    case 'schedule_card':
      return (
        <ScheduleCard 
          data={component.data as ScheduleCardData}
          actions={component.actions}
          onAction={handleAction}
        />
      );
      
    case 'confirm_dialog':
      return (
        <ConfirmDialog
          data={component.data as ConfirmDialogData}
          onConfirm={() => handleAction('confirm')}
          onCancel={() => handleAction('cancel')}
        />
      );
      
    case 'options_list':
      return (
        <OptionsList
          data={component.data as OptionsListData}
          onSelect={(selected) => onAction('submit', { selected })}
        />
      );
      
    case 'time_picker':
      return (
        <TimePicker
          data={component.data as TimePickerData}
          onSelect={(time) => onAction('submit', { time })}
        />
      );
      
    case 'success_banner':
      return <SuccessBanner message={component.data.message} />;
      
    case 'error_alert':
      return <ErrorAlert message={component.data.message} />;
      
    default:
      return null;
  }
}
```

### 3.9 日程卡片前端组件

```tsx
// web/src/components/ai/ScheduleCard.tsx

interface ScheduleCardProps {
  data: ScheduleCardData;
  actions?: UIAction[];
  onAction: (actionId: string) => void;
}

export function ScheduleCard({ data, actions, onAction }: ScheduleCardProps) {
  const statusColors = {
    preview: 'border-blue-200 bg-blue-50',
    confirmed: 'border-green-200 bg-green-50',
    conflict: 'border-red-200 bg-red-50',
  };

  return (
    <div className={`rounded-lg border p-4 ${statusColors[data.status]}`}>
      <div className="flex items-start gap-3">
        <CalendarIcon className="h-5 w-5 text-blue-600" />
        <div className="flex-1">
          <h3 className="font-medium text-gray-900">{data.title}</h3>
          <p className="text-sm text-gray-600">
            {format(new Date(data.startTime), 'MM月dd日 HH:mm')} - 
            {format(new Date(data.endTime), 'HH:mm')}
          </p>
          <p className="text-xs text-gray-500">
            时长: {data.duration} 分钟
            {data.location && ` • ${data.location}`}
          </p>
        </div>
        
        {data.status === 'conflict' && (
          <span className="rounded-full bg-red-100 px-2 py-1 text-xs text-red-600">
            冲突
          </span>
        )}
      </div>
      
      {actions && actions.length > 0 && (
        <div className="mt-3 flex gap-2">
          {actions.map((action) => (
            <Button
              key={action.id}
              size="sm"
              variant={action.style === 'primary' ? 'default' : action.style}
              onClick={() => onAction(action.id)}
            >
              {action.label}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}
```

---

## 4. 实现路径

### Day 1: 核心类型定义

- [ ] 定义 `UIComponent` 类型系统
- [ ] 实现基础组件（Text, ScheduleCard）
- [ ] 实现 `UIGenerator`

### Day 2: 更多组件

- [ ] 实现 ConfirmDialog
- [ ] 实现 OptionsList
- [ ] 实现 TimePicker

### Day 3: 前端渲染器

- [ ] 实现 `UIComponentRenderer`
- [ ] 实现各组件前端代码
- [ ] 样式完善

### Day 4: 集成与测试

- [ ] 与 Agent 集成
- [ ] Action 处理流程
- [ ] 端到端测试

---

## 5. 交付物

### 5.1 代码产出

| 文件 | 说明 |
|:---|:---|
| `plugin/ai/genui/types.go` | 类型定义 |
| `plugin/ai/genui/schedule_card.go` | 日程卡片 |
| `plugin/ai/genui/confirm_dialog.go` | 确认对话框 |
| `plugin/ai/genui/options_list.go` | 选项列表 |
| `plugin/ai/genui/time_picker.go` | 时间选择器 |
| `plugin/ai/genui/generator.go` | UI 生成器 |
| `web/src/components/ai/UIComponentRenderer.tsx` | 前端渲染器 |
| `web/src/components/ai/*.tsx` | 各组件实现 |

### 5.2 支持的组件类型

| 类型 | 用途 | 示例场景 |
|:---|:---|:---|
| `schedule_card` | 日程预览/确认 | 快速创建日程 |
| `memo_card` | 笔记预览 | 搜索结果展示 |
| `confirm_dialog` | 确认操作 | 删除确认 |
| `options_list` | 多选项 | 时间段选择 |
| `time_picker` | 时间选择 | 明确时间 |
| `progress_bar` | 进度展示 | 批量操作 |
| `error_alert` | 错误提示 | 操作失败 |
| `success_banner` | 成功提示 | 操作成功 |

---

## 6. 验收标准

### 6.1 功能验收

- [ ] 日程预览卡片正确渲染
- [ ] 确认对话框交互正常
- [ ] 选项列表可选择提交
- [ ] Action 回调正确触发

### 6.2 UI 验收

- [ ] 组件样式一致
- [ ] 响应式布局
- [ ] 深色模式支持

### 6.3 测试用例

```tsx
describe('UIComponentRenderer', () => {
  it('renders schedule card correctly', () => {
    const component = {
      type: 'schedule_card',
      data: {
        title: '会议',
        startTime: '2026-01-28T15:00:00',
        endTime: '2026-01-28T16:00:00',
        duration: 60,
        status: 'preview',
      },
      actions: [
        { id: 'confirm', label: '确认', style: 'primary' },
      ],
    };
    
    render(<UIComponentRenderer component={component} onAction={jest.fn()} />);
    
    expect(screen.getByText('会议')).toBeInTheDocument();
    expect(screen.getByText('确认')).toBeInTheDocument();
  });
});
```

---

## 7. ROI 分析

| 投入 | 产出 |
|:---|:---|
| 开发: 4 人天 | 交互效率提升 50%+ |
| 存储: 0 | 用户体验显著提升 |
| 维护: 组件化易扩展 | 为后续功能铺路 |

---

## 8. 风险与缓解

| 风险 | 概率 | 影响 | 缓解措施 |
|:---|:---:|:---:|:---|
| 组件过于复杂 | 中 | 中 | MVP 先实现核心组件 |
| 前后端不一致 | 中 | 中 | 定义明确的类型契约 |
| 样式不统一 | 低 | 低 | 使用设计系统组件 |

---

## 9. 排期

| 日期 | 任务 | 负责人 |
|:---|:---|:---|
| Sprint 4 Day 1 | 核心类型定义 | TBD |
| Sprint 4 Day 2 | 更多组件 | TBD |
| Sprint 4 Day 3 | 前端渲染器 | TBD |
| Sprint 4 Day 4 | 集成与测试 | TBD |

---

> **纲领来源**: [00-master-roadmap.md](../../../archived/research_20250202/00-master-roadmap.md)  
> **研究文档**: [assistant-roadmap.md](../../../research/assistant-roadmap.md)  
> **版本**: v1.0  
> **更新时间**: 2026-01-27
