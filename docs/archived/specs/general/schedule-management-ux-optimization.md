# 日程管理 UX 优化方案

## 核心目标
1. **减少确认**：约定 > 配置 > 询问，直接创建日程
2. **AI Native 体验**：日历/时间轴已满意，重构其他交互组件

---

## 优化项清单

### P0 - 自动冲突解决
**问题**：冲突时需手动选择替代时间（ConflictResolution.tsx 第69-90行）

**方案**：
- 利用后端 `auto_resolved` 字段（已实现评分算法）
- 前端直接采纳最佳时间，自动创建
- 显示 Toast + 3秒 Undo 按钮

**修改文件**：
- `web/src/components/ScheduleAI/ConflictResolution.tsx` - 增加自动解决逻辑
- `web/src/components/ScheduleQuickInput/ScheduleQuickInput.tsx` - 集成 Undo Toast

**交互变更**：
```
Before: 冲突 → 选时间槽 → 点"重新安排" (3步)
After:  冲突 → 自动调整 + Toast "已调整到15:30" [撤销] (0步)
```

---

### P0 - 单次点击创建
**问题**：建议卡片需点击"确认"按钮（ScheduleSuggestionCard.tsx 第51-63行）

**方案**：
- 移除"确认/取消"按钮
- 整个卡片可点击，点击即创建
- 添加 hover 效果 + 点击动画（卡片收缩 + checkmark）

**修改文件**：
- `web/src/components/ScheduleAI/ScheduleSuggestionCard.tsx` - 重构为单击交互

**交互变更**：
```
Before: 看卡片 → 点"确认" (2步)
After:  点卡片 → 完成 (1步)
```

---

### P1 - 工具状态简化
**问题**：StreamingFeedback 显示过多工具细节

**方案**：
- 默认只显示最新 1 条状态（当前 3 条）
- 工具名称简化为单行文案："正在安排..."
- 可选"查看详情"展开

**修改文件**：
- `web/src/components/ScheduleAI/StreamingFeedback.tsx` - 精简显示逻辑

---

### P1 - 快速编辑浮窗
**问题**：编辑日程需打开完整对话框

**方案**：
- 日程卡片右键/长按 → 浮动输入框
- 支持快捷指令："延后30分钟"、"改到明天"
- AI 解析后直接更新

**新增文件**：
- `web/src/components/ScheduleQuickInput/QuickEditPopover.tsx`

**修改文件**：
- `web/src/components/AIChat/ScheduleTimeline.tsx` - 添加编辑触发器

---

### P2 - 语义搜索
**问题**：搜索栏仅支持文本过滤

**方案**：
- 支持自然语言："下周的会议"、"本月重要日程"
- 复用后端 QueryRouter 时间解析能力
- 显示解析后的时间范围 Tag

**修改文件**：
- `web/src/components/AIChat/ScheduleSearchBar.tsx` - 增加语义搜索
- `web/src/hooks/useScheduleQueries.ts` - 新增搜索 Hook

---

## 技术实现要点

### 自动冲突解决核心逻辑
```typescript
// ConflictResolution.tsx
useEffect(() => {
  if (data.auto_resolved && autoResolveEnabled) {
    // 直接创建，跳过手动选择
    onAction("reschedule", data.auto_resolved);
    showUndoToast(`已调整到 ${data.auto_resolved.label}`, () => {
      // Undo: 删除并重新打开冲突面板
    });
  }
}, [data.auto_resolved]);
```

### 单击卡片创建
```typescript
// ScheduleSuggestionCard.tsx
<div 
  onClick={handleConfirm}
  className="cursor-pointer hover:bg-primary/15 transition-colors"
>
  {/* 移除按钮，整个卡片可点击 */}
</div>
```

### 配置逃生门
```typescript
// 用户可关闭自动解决
const AUTO_RESOLVE = localStorage.getItem('schedule.autoResolve') !== 'false';
```

---

## 文件清单

| 优先级 | 操作 | 文件 |
|--------|------|------|
| P0 | 修改 | `web/src/components/ScheduleAI/ConflictResolution.tsx` |
| P0 | 修改 | `web/src/components/ScheduleAI/ScheduleSuggestionCard.tsx` |
| P0 | 修改 | `web/src/components/ScheduleQuickInput/ScheduleQuickInput.tsx` |
| P1 | 修改 | `web/src/components/ScheduleAI/StreamingFeedback.tsx` |
| P1 | 新增 | `web/src/components/ScheduleQuickInput/QuickEditPopover.tsx` |
| P1 | 修改 | `web/src/components/AIChat/ScheduleTimeline.tsx` |
| P2 | 修改 | `web/src/components/AIChat/ScheduleSearchBar.tsx` |
| P2 | 修改 | `web/src/hooks/useScheduleQueries.ts` |

---

## 验证方案

1. **自动冲突解决**：
   - 创建与现有日程冲突的新日程
   - 验证自动调整到最佳时间
   - 验证 Undo 功能正常

2. **单击创建**：
   - AI 生成建议卡片
   - 单击卡片验证直接创建
   - 验证无需二次确认

3. **整体流程**：
   - `make start` 启动服务
   - 在日程页面测试完整创建流程
   - 验证操作步骤从 3-4 步减少到 1 步

---

## 预期效果
- 操作步骤：从 3-4 步 → 1 步
- 冲突解决：自动完成率 > 80%
- 用户感知：像聊天一样自然创建日程
