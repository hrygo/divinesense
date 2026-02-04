import { AlertCircle, CheckCircle, Sparkles, Wrench, XCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
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
 * EventBadge - Displays event type with icon and styling
 * EventBadge - 显示事件类型徽章
 */
export function EventBadge({ type, className }: EventBadgeProps) {
  const { t } = useTranslation();

  const eventConfig: Record<CcEventType, { icon: React.ElementType; key: string; color: string }> = {
    thinking: {
      icon: Sparkles,
      key: "ai.events.thinking",
      color: "text-slate-500 dark:text-slate-400",
    },
    tool_use: {
      icon: Wrench,
      key: "ai.events.tool_use",
      color: "text-blue-500 dark:text-blue-400",
    },
    tool_result: {
      icon: CheckCircle,
      key: "ai.events.tool_result",
      color: "text-green-500 dark:text-green-400",
    },
    answer: {
      icon: CheckCircle,
      key: "ai.events.answer",
      color: "text-emerald-600 dark:text-emerald-400",
    },
    error: {
      icon: AlertCircle,
      key: "ai.events.error",
      color: "text-red-500 dark:text-red-400",
    },
  };

  const config = eventConfig[type] || eventConfig.answer;
  const Icon = config.icon;
  const label = t(config.key) as string;

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
      <span>{label}</span>
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
  const { t } = useTranslation();

  const config = isError
    ? { icon: XCircle, key: "ai.events.failed", color: "text-red-500 dark:text-red-400" }
    : { icon: CheckCircle, key: "ai.events.success", color: "text-green-500 dark:text-green-400" };
  const Icon = config.icon;
  const label = t(config.key) as string;

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
      <span>{label}</span>
    </div>
  );
}
