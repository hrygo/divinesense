# SPEC-007: 前端 AIChatDrawer 组件

**优先级**: P2 (用户体验)
**预计工时**: 8 小时
**依赖**: SPEC-006

## 目标
实现右侧滑出式 AI 聊天面板,支持流式响应展示和引用跳转。

## 实施内容

### 1. 创建 React Hook
**文件路径**: `web/src/hooks/useAIChat.ts`

```typescript
import { useMutation } from "@tanstack/react-query";
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { AIService } from "@/types/proto/api/v1/ai_service_connect";
import { ChatRequest } from "@/types/proto/api/v1/ai_service_pb";

const client = createPromiseClient(
  AIService,
  createConnectTransport({ baseUrl: "/api/v1" })
);

export function useAIChat() {
  const chatMutation = useMutation({
    mutationFn: async ({ message, filter }: { message: string; filter?: string }) => {
      const stream = client.chat({
        message,
        filter,
      });

      const response = {
        async *[Symbol.asyncIterator]() {
          for await (const chunk of stream) {
            yield chunk;
          }
        },
      };

      return response;
    },
  });

  return {
    chat: chatMutation.mutate,
    isLoading: chatMutation.isPending,
    error: chatMutation.error,
  };
}
```

### 2. 主组件实现
**文件路径**: `web/src/components/AI/AIChatDrawer.tsx`

```typescript
import React, { useState, useRef, useEffect } from "react";
import { X, Send, Loader } from "lucide-react";
import { useAIChat } from "@/hooks/useAIChat";
import { ThinkingBubble } from "./ThinkingBubble";
import { Citation } from "./Citation";
import clsx from "clsx";

interface Props {
  open: boolean;
  onClose: () => void;
  filter?: string; // 可选: 过滤器
}

export const AIChatDrawer: React.FC<Props> = ({ open, onClose, filter }) => {
  const [messages, setMessages] = useState<Array<{ role: "user" | "assistant"; content: string }>>([]);
  const [inputValue, setInputValue] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [citations, setCitations] = useState<Citation[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { chat } = useAIChat();

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async () => {
    if (!inputValue.trim() || isLoading) return;

    const userMessage = inputValue;
    setMessages((prev) => [...prev, { role: "user", content: userMessage }]);
    setInputValue("");
    setIsLoading(true);
    setCitations([]);

    try {
      const responseStream = await chat({ message: userMessage, filter });
      let assistantMessage = "";

      for await (const chunk of responseStream) {
        if (chunk.answer) {
          assistantMessage += chunk.answer;
          setMessages((prev) => {
            const newMessages = [...prev];
            const lastMessage = newMessages[newMessages.length - 1];
            if (lastMessage?.role === "assistant") {
              lastMessage.content = assistantMessage;
            } else {
              newMessages.push({ role: "assistant", content: assistantMessage });
            }
            return newMessages;
          });
        }

        if (chunk.citations) {
          setCitations(chunk.citations);
        }

        if (chunk.done) {
          setIsLoading(false);
        }
      }
    } catch (error) {
      console.error("Chat failed:", error);
      setMessages((prev) => [
        ...prev,
        {
          role: "assistant",
          content: "抱歉,我遇到了一些问题。请稍后再试。",
        },
      ]);
      setIsLoading(false);
    }
  };

  return (
    <div
      className={clsx(
        "fixed inset-y-0 right-0 w-[480px] bg-white shadow-2xl transform transition-transform duration-300 ease-in-out",
        open ? "translate-x-0" : "translate-x-full"
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <h2 className="text-lg font-semibold">AI 助手</h2>
        <button
          onClick={onClose}
          className="p-2 hover:bg-gray-100 rounded-full"
        >
          <X className="w-5 h-5" />
        </button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4 h-[calc(100vh-180px)]">
        {messages.map((msg, idx) => (
          <div
            key={idx}
            className={clsx(
              "flex",
              msg.role === "user" ? "justify-end" : "justify-start"
            )}
          >
            <div
              className={clsx(
                "max-w-[80%] rounded-lg px-4 py-2",
                msg.role === "user"
                  ? "bg-blue-500 text-white"
                  : "bg-gray-100 text-gray-900"
              )}
            >
              {msg.content}
            </div>
          </div>
        ))}

        {isLoading && <ThinkingBubble />}

        <div ref={messagesEndRef} />
      </div>

      {/* Citations */}
      {citations.length > 0 && (
        <div className="border-t p-4 bg-gray-50">
          <h3 className="text-sm font-semibold mb-2">引用来源</h3>
          <div className="space-y-2">
            {citations.map((citation, idx) => (
              <Citation key={idx} citation={citation} />
            ))}
          </div>
        </div>
      )}

      {/* Input */}
      <div className="border-t p-4">
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyPress={(e) => e.key === "Enter" && handleSend()}
            placeholder="输入问题..."
            disabled={isLoading}
            className="flex-1 px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={handleSend}
            disabled={isLoading || !inputValue.trim()}
            className="p-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoading ? (
              <Loader className="w-5 h-5 animate-spin" />
            ) : (
              <Send className="w-5 h-5" />
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
```

### 3. ThinkingBubble 组件
**文件路径**: `web/src/components/AI/ThinkingBubble.tsx`

```typescript
import React, { useEffect, useState } from "react";
import { Loader } from "lucide-react";

const steps = [
  { text: "理解问题...", delay: 0 },
  { text: "检索知识库...", delay: 800 },
  { text: "重排结果...", delay: 1600 },
  { text: "生成答案...", delay: 2400 },
];

export const ThinkingBubble: React.FC = () => {
  const [currentStep, setCurrentStep] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setCurrentStep((prev) => (prev + 1) % steps.length);
    }, 800);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex items-center gap-2 text-gray-500">
      <Loader className="w-4 h-4 animate-spin" />
      <span className="text-sm">{steps[currentStep].text}</span>
    </div>
  );
};
```

### 4. Citation 组件
**文件路径**: `web/src/components/AI/Citation.tsx`

```typescript
import React from "react";
import { ExternalLink } from "lucide-react";

interface Props {
  citation: {
    memoId: number;
    content: string;
    score: number;
  };
}

export const Citation: React.FC<Props> = ({ citation }) => {
  return (
    <a
      href={`/memo/${citation.memoId}`}
      target="_blank"
      rel="noopener noreferrer"
      className="block p-3 bg-white rounded-lg border hover:shadow-md transition-shadow"
    >
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm text-gray-700 line-clamp-2">
          {citation.content}
        </p>
        <ExternalLink className="w-4 h-4 text-gray-400 flex-shrink-0 mt-1" />
      </div>
      <div className="mt-2 flex items-center gap-2 text-xs text-gray-500">
        <span>相关度: {Math.round(citation.score * 100)}%</span>
        <span>•</span>
        <span>#{citation.memoId}</span>
      </div>
    </a>
  );
};
```

### 5. 集成到主布局
**文件路径**: `web/src/pages/Home.tsx`

```typescript
import { useState } from "react";
import { MessageSquare } from "lucide-react";
import { AIChatDrawer } from "@/components/AI/AIChatDrawer";

export const Home: React.FC = () => {
  const [aiChatOpen, setAiChatOpen] = useState(false);

  return (
    <div>
      {/* AI Chat Toggle Button */}
      <button
        onClick={() => setAiChatOpen(true)}
        className="fixed bottom-6 right-6 p-4 bg-blue-500 text-white rounded-full shadow-lg hover:bg-blue-600"
      >
        <MessageSquare className="w-6 h-6" />
      </button>

      {/* AI Chat Drawer */}
      <AIChatDrawer open={aiChatOpen} onClose={() => setAiChatOpen(false)} />
    </div>
  );
};
```

## 验收标准

### AC-1: TypeScript 编译通过
```bash
# 执行
cd web && pnpm lint

# 预期结果
- 无类型错误
- 无 lint 错误
```

### AC-2: 组件渲染测试
```bash
# 启动开发服务器
cd web && pnpm dev

# 访问 http://localhost:5173

# 预期结果
- 右下角显示蓝色圆形按钮 (MessageSquare 图标)
- 点击按钮后,右侧滑出聊天面板
- 面板宽度 480px
- 背景有半透明遮罩 (可选)
```

### AC-3: 流式响应测试
```
# 测试步骤
1. 在输入框输入 "如何使用 Go 语言?"
2. 点击发送按钮
3. 观察输出

# 预期结果
- 用户消息立即显示在右侧
- 显示 ThinkingBubble,步骤依次切换
- 1-2 秒后开始显示 AI 回答
- 回答逐步追加 (打字机效果)
- 最后引用来源显示在底部
- 发送按钮禁用状态正确切换
```

### AC-4: 引用跳转测试
```
# 测试步骤
1. 发送问题,等待回答完成
2. 点击引用卡片中的某个引用
3. 观察行为

# 预期结果
- 在新标签页打开对应 Memo 详情页
- URL 格式: /memo/123
- 引用卡片 hover 有阴影效果
```

### AC-5: 错误处理测试
```
# 场景 1: 空输入
1. 输入框为空时,发送按钮应禁用
2. 按 Enter 键无响应

# 场景 2: 网络错误
1. 模拟网络断开 (DevTools -> Offline)
2. 发送消息
3. 预期: 显示错误消息 "抱歉,我遇到了一些问题..."
4. 输入框恢复可用状态

# 场景 3: API 超时
1. 设置超时为 1s
2. 发送问题
3. 预期: 显示错误提示
```

### AC-6: 响应式测试
```bash
# 测试不同屏幕尺寸

# 桌面 (1920x1080)
# 预期: 面板宽度 480px,正常显示

# 平板 (768x1024)
# 预期: 面板宽度 100%,正常显示

# 手机 (375x667)
# 预期: 面板全屏,关闭按钮位于左上角
```

### AC-7: 性能测试
```
# 使用 React DevTools Profiler
1. 发送 10 条消息
2. 记录渲染时间

# 预期结果
- 每次渲染 < 16ms (60fps)
- 无内存泄漏
- 长消息 (1000+ 字符) 不卡顿
```

### AC-8: 可访问性测试
```bash
# 使用键盘导航
1. Tab 键聚焦到输入框
2. 输入消息
3. 按 Enter 发送
4. Escape 关闭面板

# 预期结果
- 所有交互可通过键盘完成
- 焦点指示器清晰可见
- ARIA 标签正确
```

## 回滚方案
- 从 `Home.tsx` 中移除 `AIChatDrawer` 组件
- 禁用 AI Chat 按钮

## 注意事项
- 流式响应需正确处理 chunk 拼接,避免重复字符
- 长消息需限制高度,支持滚动
- Citation 内容需截断,避免过长
- 移动端需优化布局 (全屏模式)
- 考虑添加 "清空对话" 功能
- 考虑添加 "复制回答" 功能