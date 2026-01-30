import { Pin, PinOff, X } from "lucide-react";
import { memo, useCallback, useEffect, useState } from "react";
import type { GenerativeUIContainerProps } from "@/components/ScheduleAI/types";
import { cn } from "@/lib/utils";

/**
 * Phase 3: 生成式 UI 持久化容器
 *
 * 功能:
 * - 用户可固定重要工具卡片
 * - 固定的工具不会自动消失
 * - 工具优先级排序（固定的在顶部）
 * - 本地存储持久化固定状态
 */

const PINNED_TOOLS_KEY = "divinesense.pinned_tools";

interface PersistentToolContainerProps {
  tools: GenerativeUIContainerProps["tools"];
  onAction: GenerativeUIContainerProps["onAction"];
  onDismiss: GenerativeUIContainerProps["onDismiss"];
  children: React.ReactNode;
  className?: string;
}

interface PinnedTool {
  id: string;
  timestamp: number;
}

function loadPinnedTools(): Set<string> {
  if (typeof window === "undefined") return new Set();
  try {
    const saved = localStorage.getItem(PINNED_TOOLS_KEY);
    if (saved) {
      const data = JSON.parse(saved) as PinnedTool[];
      // 过滤超过 24 小时的固定
      const dayAgo = Date.now() - 24 * 60 * 60 * 1000;
      return new Set(data.filter((t) => t.timestamp > dayAgo).map((t) => t.id));
    }
  } catch (e) {
    console.warn("Failed to load pinned tools:", e);
  }
  return new Set();
}

function savePinnedTools(pinned: Set<string>) {
  if (typeof window === "undefined") return;
  try {
    const data: PinnedTool[] = Array.from(pinned).map((id) => ({
      id,
      timestamp: Date.now(),
    }));
    localStorage.setItem(PINNED_TOOLS_KEY, JSON.stringify(data));
  } catch (e) {
    console.warn("Failed to save pinned tools:", e);
  }
}

export const PersistentToolContainer = memo(function PersistentToolContainer({
  tools,
  onAction: _onAction,
  onDismiss,
  children,
  className,
}: PersistentToolContainerProps) {
  const [pinnedToolIds, setPinnedToolIds] = useState<Set<string>>(() => loadPinnedTools());

  // 持久化固定状态
  useEffect(() => {
    savePinnedTools(pinnedToolIds);
  }, [pinnedToolIds]);

  // 切换固定状态
  const togglePin = useCallback((toolId: string) => {
    setPinnedToolIds((prev) => {
      const next = new Set(prev);
      if (next.has(toolId)) {
        next.delete(toolId);
      } else {
        next.add(toolId);
      }
      return next;
    });
  }, []);

  // 排序后的工具列表：固定的在顶部
  const sortedTools = [...tools].sort((a, b) => {
    const aPinned = pinnedToolIds.has(a.id);
    const bPinned = pinnedToolIds.has(b.id);
    if (aPinned && !bPinned) return -1;
    if (!aPinned && bPinned) return 1;
    return b.timestamp - a.timestamp; // 新的在前
  });

  // 处理工具关闭
  const handleDismiss = useCallback(
    (toolId: string) => {
      // 固定的工具不能自动关闭，只能手动解除固定
      if (pinnedToolIds.has(toolId)) {
        return;
      }
      onDismiss?.(toolId);
    },
    [onDismiss, pinnedToolIds],
  );

  // 手动关闭固定工具
  const handleManualDismiss = useCallback(
    (toolId: string) => {
      togglePin(toolId);
      onDismiss?.(toolId);
    },
    [togglePin, onDismiss],
  );

  return (
    <div className={cn("space-y-3", className)}>
      {/* 固定工具区域 */}
      {sortedTools
        .filter((tool) => pinnedToolIds.has(tool.id))
        .map((tool) => (
          <PinnedToolCard key={tool.id} tool={tool} onDismiss={() => handleManualDismiss(tool.id)} onUnpin={() => togglePin(tool.id)} />
        ))}

      {/* 普通工具区域 */}
      {sortedTools
        .filter((tool) => !pinnedToolIds.has(tool.id))
        .map((tool) => (
          <ToolCardWithPin key={tool.id} tool={tool} onDismiss={() => handleDismiss(tool.id)} onPin={() => togglePin(tool.id)} />
        ))}

      {children}
    </div>
  );
});

// 普通工具卡片（带固定按钮）
interface ToolCardProps {
  tool: GenerativeUIContainerProps["tools"][number];
  onDismiss: () => void;
  onPin: () => void;
}

const ToolCardWithPin = memo(function ToolCardWithPin({ tool, onDismiss: _onDismiss, onPin }: ToolCardProps) {
  return (
    <div className="relative group">
      {/* 固定按钮 - hover 时显示 */}
      <button
        onClick={onPin}
        className={cn(
          "absolute -top-2 -right-2 z-10",
          "p-1.5 rounded-full bg-primary text-primary-foreground shadow-lg",
          "opacity-0 group-hover:opacity-100 transition-opacity",
          "hover:scale-110 active:scale-95",
        )}
        title="固定此卡片"
      >
        <Pin className="w-3 h-3" />
      </button>

      {/* 工具内容渲染由父组件的 children 处理 */}
      <div className="p-3 rounded-xl border bg-card shadow-sm animate-in fade-in slide-in-from-top-2 duration-300">
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1">
            {/* 这里渲染具体的工具组件 */}
            <div className="text-sm text-muted-foreground">[{tool.type}] 工具内容</div>
          </div>
        </div>
      </div>
    </div>
  );
});

// 固定的工具卡片（带解除固定按钮）
interface PinnedToolCardProps {
  tool: GenerativeUIContainerProps["tools"][number];
  onDismiss: () => void;
  onUnpin: () => void;
}

const PinnedToolCard = memo(function PinnedToolCard({ tool, onDismiss, onUnpin }: PinnedToolCardProps) {
  return (
    <div className="relative">
      <div className="absolute -top-1 -right-1 flex gap-1">
        {/* 解除固定按钮 */}
        <button
          onClick={onUnpin}
          className={cn("p-1 rounded-full bg-green-500 text-white shadow-md", "hover:scale-110 active:scale-95 transition-transform")}
          title="解除固定"
        >
          <PinOff className="w-3 h-3" />
        </button>
        {/* 关闭按钮 */}
        <button
          onClick={onDismiss}
          className={cn(
            "p-1 rounded-full bg-muted text-muted-foreground shadow-md",
            "hover:bg-destructive hover:text-destructive-foreground",
            "hover:scale-110 active:scale-95 transition-transform",
          )}
          title="关闭"
        >
          <X className="w-3 h-3" />
        </button>
      </div>

      {/* 固定工具内容 */}
      <div className="p-3 rounded-xl border-2 border-primary/50 bg-primary/5 shadow-md">
        <div className="flex items-center gap-1.5 mb-2">
          <Pin className="w-3 h-3 text-primary" />
          <span className="text-xs font-medium text-primary">已固定</span>
        </div>
        <div className="text-sm text-muted-foreground">[{tool.type}] 工具内容</div>
      </div>
    </div>
  );
});

export default PersistentToolContainer;
