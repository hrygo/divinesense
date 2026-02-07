/**
 * TimelineNode Component
 *
 * Unified timeline node for AI chat timeline.
 * Ensures consistent sizing and styling across all node types.
 *
 * Phase 1: Visual Hierarchy Optimization
 */

import { memo } from "react";
import { cn } from "@/lib/utils";
import { NODE_COLORS, TIMELINE_NODE_CONFIG } from "../constants";
import type { TimelineNodeProps, TimelineNodeType } from "../types";

/** Icon components for each node type */
const NODE_ICONS: Record<TimelineNodeType, React.ReactNode> = {
  user: (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
      />
    </svg>
  ),
  thinking: (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  ),
  tool: (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  ),
  answer: (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
      />
    </svg>
  ),
  error: (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  ),
};

/**
 * TimelineNode component
 *
 * Renders a unified circular node for the AI chat timeline.
 * All nodes use the same base size (w-6 h-6) with only colors varying by type.
 */
export const TimelineNode = memo(function TimelineNode({ type, icon, className, onClick }: TimelineNodeProps) {
  const baseClasses = cn(
    // Unified size and border
    TIMELINE_NODE_CONFIG.size,
    TIMELINE_NODE_CONFIG.border,
    TIMELINE_NODE_CONFIG.radius,
    // Type-specific colors
    NODE_COLORS[type],
    // Flex center for icon
    "flex items-center justify-center shrink-0",
    // Optional click cursor
    onClick && "cursor-pointer hover:scale-110 transition-transform",
    className,
  );

  const content = icon || NODE_ICONS[type];

  return (
    <div className={baseClasses} onClick={onClick} aria-label={`${type} node`}>
      {content}
    </div>
  );
});
