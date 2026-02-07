# 鹦鹉系统集成指南 (AIChat.tsx)

> **状态**: 前端组件已完成，待集成到 AIChat.tsx
> **更新时间**: 2025-01-22

---

## 已完成的前端组件

### 1. 类型定义 ✅
**文件**: `web/src/types/parrot.ts`

- `ParrotAgentType` 枚举
- `ParrotAgent` 接口
- `MemoQueryResultData`、`ScheduleQueryResultData` 类型
- `ParrotChatCallbacks`、`ParrotChatParams` 接口
- 辅助函数（`getAvailableParrots`、`getParrotAgent`）

### 2. Hooks ✅
**文件**: `web/src/hooks/useParrotChat.ts`

- `useParrotChat()` Hook
- `streamChat()` 函数支持鹦鹉代理
- 事件处理逻辑（`handleParrotEvent`）
- React Query 集成

**文件**: `web/src/hooks/useAIQueries.ts` (已扩展)

- 添加 `agentType` 参数支持
- 添加鹦鹉特定回调
- 事件处理集成

### 3. UI 组件 ✅

#### ParrotSelector (`web/src/components/AIChat/ParrotSelector.tsx`)
- @ 符号触发鹦鹉选择器
- 显示 4 只鹦鹉列表
- 键盘导航（↑↓ Enter Esc）
- 响应式设计

#### ParrotQuickActions (`web/src/components/AIChat/ParrotQuickActions.tsx`)
- 快捷操作卡片
- 点击切换鹦鹉
- 视觉反馈（选中状态、颜色主题）

#### ParrotStatus (`web/src/components/AIChat/ParrotStatus.tsx`)
- 显示当前鹦鹉状态
- `ParrotStatus` - 完整状态显示
- `ParrotStatusCompact` - 紧凑状态显示
- `ParrotThinkingIndicator` - 思考指示器

#### MemoQueryResult (`web/src/components/AIChat/MemoQueryResult.tsx`)
- 显示笔记查询结果
- 按相关度排序
- 点击跳转到笔记
- 相关度分数显示

---

## AIChat.tsx 集成步骤

### 步骤 1: 添加导入（已完成）✅

在 `AIChat.tsx` 顶部添加以下导入：

```typescript
// 鹦鹉组件
import { ParrotSelector } from "@/components/AIChat/ParrotSelector";
import { ParrotQuickActions } from "@/components/AIChat/ParrotQuickActions";
import { ParrotStatus, ParrotStatusCompact } from "@/components/AIChat/ParrotStatus";
import { MemoQueryResult } from "@/components/AIChat/MemoQueryResult";

// 鹦鹉类型和 Hook
import { ParrotAgent, ParrotAgentType, getAvailableParrots } from "@/types/parrot";
import type { MemoQueryResultData } from "@/types/parrot";
```

### 步骤 2: 添加状态变量（已完成）✅

在 `AIChat` 组件中添加以下状态：

```typescript
// Parrot-related state (Milestone 1)
const [currentParrot, setCurrentParrot] = useState<ParrotAgent | null>(null);
const [showParrotSelector, setShowParrotSelector] = useState(false);
const [parrotSelectorPosition, setParrotSelectorPosition] = useState<{ x: number; y: number } | null>(null);
const [isParrotThinking, setIsParrotThinking] = useState(false);
const [memoQueryResults, setMemoQueryResults] = useState<MemoQueryResultData[]>([]);
const textareaRef = useRef<HTMLTextAreaElement>(null);
```

### 步骤 3: 添加事件处理函数

在 `AIChat` 组件中添加以下函数：

```typescript
// Handle @ symbol to trigger parrot selector
const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
  const value = e.target.value;
  setInput(value);

  // Check if user typed @ symbol
  if (value.endsWith("@") && textareaRef.current) {
    const rect = textareaRef.current.getBoundingClientRect();
    const x = rect.left;
    const y = rect.bottom + window.scrollY;
    setParrotSelectorPosition({ x, y });
    setShowParrotSelector(true);
  }
};

// Handle parrot selection
const handleParrotSelect = (parrot: ParrotAgent) => {
  setCurrentParrot(parrot);
  // Remove @ symbol from input
  setInput(input.slice(0, -1));
  setShowParrotSelector(false);
};

// Handle parrot chat with callbacks
const handleParrotChat = async (message: string, history: string[]) => {
  if (!currentParrot) {
    // Use default chat flow
    return handleSend(message);
  }

  setIsParrotThinking(true);
  setMemoQueryResults([]);

  try {
    await chatHook.stream(
      {
        message,
        history,
        agentType: currentParrot.id,
        userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      },
      {
        onThinking: (msg) => {
          console.log("[Parrot Thinking]", msg);
        },
        onToolUse: (toolName) => {
          console.log("[Parrot Tool Use]", toolName);
        },
        onToolResult: (result) => {
          console.log("[Parrot Tool Result]", result);
        },
        onMemoQueryResult: (result) => {
          setMemoQueryResults(prev => [...prev, result]);
        },
        onContent: (content) => {
          // Update message content
          setItems(prev => {
            const newItems = [...prev];
            const lastItem = newItems[newItems.length - 1];
            if (lastItem && 'role' in lastItem && lastItem.role === "assistant") {
              lastItem.content += content;
            }
            return newItems;
          });
        },
        onDone: () => {
          setIsParrotThinking(false);
          setIsTyping(false);
        },
        onError: (error) => {
          setIsParrotThinking(false);
          setIsTyping(false);
          console.error("[Parrot Error]", error);
        },
      }
    );
  } catch (error) {
    setIsParrotThinking(false);
    setIsTyping(false);
    console.error("[Parrot Chat Error]", error);
  }
};
```

### 步骤 4: 修改 Textarea 组件

找到 `Textarea` 组件，添加 `ref` 和 `onChange` 处理：

```typescript
<Textarea
  ref={textareaRef}
  value={input}
  onChange={handleInputChange}
  placeholder={currentParrot
    ? `与 ${currentParrot.displayName} 对话...`
    : "输入消息，输入 @ 选择鹦鹉助手..."
  }
  onKeyDown={(e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }}
  rows={4}
  className="resize-none"
/>
```

### 步骤 5: 添加 ParrotQuickActions 组件

在聊天输入框上方添加鹦鹉快捷操作：

```typescript
{/* Parrot Quick Actions */}
<div className="mb-4">
  <ParrotQuickActions
    currentParrot={currentParrot}
    onParrotChange={setCurrentParrot}
    disabled={isTyping}
  />
</div>
```

### 步骤 6: 添加 ParrotSelector 组件

在组件的返回 JSX 中添加选择器：

```typescript
{showParrotSelector && parrotSelectorPosition && (
  <ParrotSelector
    onSelect={handleParrotSelect}
    onClose={() => setShowParrotSelector(false)}
    position={parrotSelectorPosition}
  />
)}
```

### 步骤 7: 显示当前鹦鹉状态

在聊天消息区域上方显示鹦鹉状态：

```typescript
{/* Current Parrot Status */}
{currentParrot && (
  <div className="mb-4">
    <ParrotStatus
      parrot={currentParrot}
      thinking={isParrotThinking}
    />
  </div>
)}
```

### 步骤 8: 显示笔记查询结果

在消息列表中显示笔记查询结果：

```typescript
{/* Memo Query Results */}
{memoQueryResults.map((result, index) => (
  <div key={index} className="mb-4">
    <MemoQueryResult result={result} />
  </div>
))}
```

### 步骤 9: 修改 handleSend 函数

修改 `handleSend` 函数以支持鹦鹉：

```typescript
const handleSend = async () => {
  if (!input.trim() || isTyping) return;

  const userMessage = input.trim();
  setInput("");
  setIsTyping(true);

  // Add user message
  setItems(prev => [...prev, { role: "user", content: userMessage }]);

  // Add placeholder for assistant response
  setItems(prev => [...prev, { role: "assistant", content: "" }]);

  // Check if parrot is selected
  if (currentParrot) {
    await handleParrotChat(userMessage, history);
  } else {
    // Use default chat flow
    await chatHook.stream(
      { message: userMessage, history },
      {
        onContent: (content) => {
          setItems(prev => {
            const newItems = [...prev];
            const lastItem = newItems[newItems.length - 1];
            if (lastItem && 'role' in lastItem && lastItem.role === "assistant") {
              lastItem.content += content;
            }
            return newItems;
          });
        },
        onDone: () => {
          setIsTyping(false);
        },
        onError: (error) => {
          setIsTyping(false);
          setErrorMessage(error.message);
        },
        // ... other callbacks
      }
    );
  }

  // Update history
  setHistory(prev => [...prev, userMessage]);
};
```

---

## 使用示例

### 场景 1: 使用默认助手

1. 直接输入消息
2. 点击发送
3. 使用现有的 RAG 系统

### 场景 2: 使用笔记助手（灰灰）

1. 在输入框中输入 `@`
2. 选择 "🦜 灰灰"
3. 输入查询："查询 Python 相关的笔记"
4. 灰灰将检索笔记并返回结果

### 场景 3: 使用日程助手（金刚）

1. 点击快捷操作卡片中的 "🦜 金刚"
2. 输入："明天下午3点开会"
3. 金刚将创建日程

### 场景 4: 切换鹦鹉

1. 点击快捷操作卡片中的其他鹦鹉
2. 当前鹦鹉状态更新
3. 继续对话

---

## 样式和主题

### 鹦鹉颜色主题

- **蓝色** (gray): 🦜 灰灰 - 笔记助手
- **紫色** (purple): 🦜 金刚 - 日程助手
- **橙色** (orange): 🦜 惊奇 - 综合助手（Milestone 2）
- **粉色** (pink): 🦜 灵灵 - 创意助手（Milestone 4）

### 响应式设计

- 移动端：卡片堆叠，横向滚动
- 平板：2 列网格
- 桌面：4 列网格

---

## 性能优化

### 1. 缓存策略
- 笔记查询结果缓存（5 分钟）
- 鹦鹉选择器状态缓存

### 2. 懒加载
- 鹦鹉组件按需加载
- 查询结果虚拟滚动

### 3. 防抖
- 输入框 @ 符号检测防抖（300ms）
- 鹦鹉选择器显示防抖

---

## 测试清单

### 功能测试
- [ ] @ 符号触发鹦鹉选择器
- [ ] 键盘导航（↑↓ Enter Esc）
- [ ] 鹦鹉选择和切换
- [ ] 笔记助手检索笔记
- [ ] 日程助手管理日程
- [ ] 笔记查询结果显示
- [ ] 思考指示器显示
- [ ] 错误处理

### UI 测试
- [ ] 响应式布局
- [ ] 主题切换（亮色/暗色）
- [ ] 动画效果
- [ ] 加载状态

### 性能测试
- [ ] 首屏加载 < 1s
- [ ] 交互响应 < 100ms
- [ ] 笔记检索 < 2s
- [ ] 日程响应 < 3s

---

## 故障排查

### 问题 1: @ 符号不触发选择器
**解决方案**:
- 检查 `textareaRef` 是否正确绑定
- 检查 `handleInputChange` 是否正确调用

### 问题 2: 鹦鹉选择器位置错误
**解决方案**:
- 检查 `parrotSelectorPosition` 计算逻辑
- 检查 CSS `position: fixed` 样式

### 问题 3: 笔记查询结果不显示
**解决方案**:
- 检查 `onMemoQueryResult` 回调是否正确
- 检查 `memoQueryResults` 状态更新
- 检查 `MemoQueryResult` 组件渲染

---

## 下一步

### Milestone 2 (未来)
- 🦜 惊奇 - 综合助手
- 多鹦鹉协作
- 鹦鹉记忆系统

### Milestone 4 (未来)
- 🦜 灵灵 - 创意助手
- 创意写作工具
- 头脑风暴功能

---

**文档版本**: v1.0
**最后更新**: 2025-01-22
**状态**: 待集成到 AIChat.tsx
