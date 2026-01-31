import { AlertCircle, CheckCircle, Sparkles, Wrench, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

/**
 * Event types for CC Runner async events
 * CC Runner 异步事件类型
 */
export type CcEventType = "thinking" | "tool_use" | "tool_result" | "answer" | "error";

interface EventBadgeProps {
  type: CcEventType;
  className?: string;
}

/**
 * Event display configuration for each event type
 * 事件显示配置
 */
const EVENT_CONFIG: Record<CcEventType, { icon: React.ElementType; label: string; color: string }> = {
  thinking: {
    icon: Sparkles,
    label: "Thinking",
    color: "text-slate-500 dark:text-slate-400",
  },
  tool_use: {
    icon: Wrench,
    label: "Tool",
    color: "text-blue-500 dark:text-blue-400",
  },
  tool_result: {
    icon: CheckCircle,
    label: "Result",
    color: "text-green-500 dark:text-green-400",
  },
  answer: {
    icon: CheckCircle,
    label: "Answer",
    color: "text-emerald-600 dark:text-emerald-400",
  },
  error: {
    icon: AlertCircle,
    label: "Error",
    color: "text-red-500 dark:text-red-400",
  },
};

/**
 * EventBadge - Displays event type with icon and styling
 * EventBadge - 显示事件类型徽章
 */
export function EventBadge({ type, className }: EventBadgeProps) {
  const config = EVENT_CONFIG[type] || EVENT_CONFIG.answer;
  const Icon = config.icon;

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium",
        "bg-slate-100 dark:bg-slate-800",
        config.color,
        "border border-slate-200 dark:border-slate-700",
        className,
      )}
    >
      <Icon className="w-3.5 h-3.5" />
      <span>{config.label}</span>
    </div>
  );
}

/**
 * ToolResultBadge - Special badge for tool_result with success/error states
 * ToolResultBadge - 工具结果专用徽章（支持成功/失败状态）
 */
interface ToolResultBadgeProps {
  isError?: boolean;
  className?: string;
}

export function ToolResultBadge({ isError = false, className }: ToolResultBadgeProps) {
  const config = isError
    ? { icon: XCircle, label: "Failed", color: "text-red-500 dark:text-red-400" }
    : { icon: CheckCircle, label: "Success", color: "text-green-500 dark:text-green-400" };
  const Icon = config.icon;

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium",
        "bg-slate-100 dark:bg-slate-800",
        config.color,
        "border border-slate-200 dark:border-slate-700",
        className,
      )}
    >
      <Icon className="w-3.5 h-3.5" />
      <span>{config.label}</span>
    </div>
  );
}
