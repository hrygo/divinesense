/**
 * TimelineNode - Unified Timeline Node Component
 *
 * Consistent visual styling for timeline nodes across
 * AIChat (UnifiedMessageBlock) and Memo (MemoBlock).
 *
 * Size: w-6 h-6 (24px)
 * Border: border-2
 * Shape: rounded-full
 */

import { Loader2 } from "lucide-react";
import { memo } from "react";
import { cn } from "@/lib/utils";
import type { TimelineNodeType } from "./types";

export interface TimelineNodeProps {
  type: TimelineNodeType;
  isLoading?: boolean;
  className?: string;
}

const NODE_CONFIG = {
  size: "w-6 h-6",
  border: "border-2",
  radius: "rounded-full",
  ring: "ring-4 ring-background",
} as const;

const NODE_COLORS: Record<TimelineNodeType, string> = {
  user: "bg-blue-500 border-blue-600 dark:bg-blue-400 dark:border-blue-500",
  thinking: "bg-blue-100 border-blue-500 dark:bg-blue-900/40 dark:border-blue-400",
  tool: "bg-purple-100 border-purple-500 dark:bg-purple-900/40 dark:border-purple-400",
  answer: "bg-amber-100 border-amber-500 dark:bg-amber-900/40 dark:border-amber-400",
  error: "bg-red-100 border-red-500 dark:bg-red-900/40 dark:border-red-400",
  edit: "bg-green-100 border-green-500 dark:bg-green-900/40 dark:border-green-400",
  archive: "bg-zinc-100 border-zinc-500 dark:bg-zinc-800 dark:border-zinc-400",
};

const ICONS: Partial<Record<TimelineNodeType, React.ComponentType<{ className?: string }>>> = {
  user: undefined, // Uses user initial instead
  thinking: Loader2,
  tool: undefined, // Uses tool icon from props
  answer: undefined, // Uses plain node
  error: undefined, // Uses plain node
  edit: undefined,
  archive: undefined,
};

export const TimelineNode = memo(function TimelineNode({ type, isLoading = false, className }: TimelineNodeProps) {
  const IconComponent = ICONS[type];

  return (
    <div
      className={cn(
        NODE_CONFIG.size,
        NODE_CONFIG.border,
        NODE_CONFIG.radius,
        NODE_CONFIG.ring,
        "flex items-center justify-center shrink-0 z-10 transition-colors",
        NODE_COLORS[type],
        isLoading && "animate-pulse",
        className,
      )}
    >
      {IconComponent && <IconComponent className="w-3.5 h-3.5" />}
    </div>
  );
});

TimelineNode.displayName = "TimelineNode";

// Export config for use in other components
export { NODE_CONFIG, NODE_COLORS };
