/**
 * MemoTimelineNode - Timeline Node for Memo List
 *
 * Inspired by Chat's TimelineNode design, adapted for Memo context:
 * - Normal state: amber/gold (matches MEMO_BLOCK_THEME)
 * - Pinned state: rose/red (highlighted)
 * - Archived state: slate/gray (dimmed)
 * - Error state: red (alert)
 */

import { Archive, Pin } from "lucide-react";
import React from "react";
import { cn } from "@/lib/utils";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

/**
 * Memo node types for timeline visualization
 */
export type MemoNodeType = "normal" | "pinned" | "archived" | "error";

/**
 * Get node type from memo state
 */
export function getMemoNodeType(memo: Memo): MemoNodeType {
  if (memo.state === 2) return "archived"; // ARCHIVED
  if (memo.pinned) return "pinned";
  return "normal";
}

/**
 * Node color configuration
 */
const NODE_COLORS = {
  normal: {
    core: "bg-amber-500 dark:bg-amber-400",
    glow: "bg-amber-500/20 dark:bg-amber-400/20",
    ring: "ring-amber-500/30 dark:ring-amber-400/30",
  },
  pinned: {
    core: "bg-rose-500 dark:bg-rose-400",
    glow: "bg-rose-500/20 dark:bg-rose-400/20",
    ring: "ring-rose-500/30 dark:ring-rose-400/30",
  },
  archived: {
    core: "bg-slate-400 dark:bg-slate-500",
    glow: "bg-slate-400/20 dark:bg-slate-500/20",
    ring: "ring-slate-400/30 dark:ring-slate-500/30",
  },
  error: {
    core: "bg-red-500 dark:bg-red-400",
    glow: "bg-red-500/20 dark:bg-red-400/20",
    ring: "ring-red-500/30 dark:ring-red-400/30",
  },
} as const;

/**
 * Timeline node size variants
 */
type NodeSize = "sm" | "md" | "lg";

const NODE_SIZES = {
  sm: { outer: "w-3 h-3", inner: "w-1.5 h-1.5" },
  md: { outer: "w-4 h-4", inner: "w-2 h-2" },
  lg: { outer: "w-5 h-5", inner: "w-2.5 h-2.5" },
} as const;

export interface MemoTimelineNodeProps {
  memo: Memo;
  size?: NodeSize;
  isLatest?: boolean;
  className?: string;
}

/**
 * MemoTimelineNode Component
 *
 * Visual timeline indicator for memo items
 */
export const MemoTimelineNode = React.memo(function MemoTimelineNode({
  memo,
  size = "md",
  isLatest = false,
  className,
}: MemoTimelineNodeProps) {
  const nodeType = getMemoNodeType(memo);
  const colors = NODE_COLORS[nodeType];
  const sizes = NODE_SIZES[size];

  // Special icon for pinned/archived states
  const renderStateIcon = () => {
    if (nodeType === "pinned") {
      return <Pin className="w-2 h-2 text-white absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2" />;
    }
    if (nodeType === "archived") {
      return <Archive className="w-2 h-2 text-white absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2" />;
    }
    return null;
  };

  return (
    <div className={cn("flex items-center justify-center", className)}>
      {/* Outer glow ring for latest item */}
      {isLatest && <div className={cn("absolute rounded-full animate-pulse-slow", sizes.outer, colors.glow)} />}

      {/* Core node */}
      <div
        className={cn(
          "rounded-full relative z-10 shadow-sm",
          sizes.outer,
          colors.core,
          isLatest && colors.ring.replace("bg-", "ring-2 ring-"),
        )}
      >
        {/* Inner solid circle */}
        <div
          className={cn(
            "rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white/30 dark:bg-black/20",
            sizes.inner,
          )}
        />

        {/* State icon overlay */}
        {renderStateIcon()}
      </div>
    </div>
  );
});

MemoTimelineNode.displayName = "MemoTimelineNode";
