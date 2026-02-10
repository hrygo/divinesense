/**
 * Timeline Types - Generic Timeline Block System
 *
 * A reusable abstraction for timeline-based content blocks,
 * shared between AIChat (UnifiedMessageBlock) and Memo (MemoBlock).
 */

/**
 * Timeline node types for visual distinction
 */
export type TimelineNodeType =
  | "user" // User input
  | "thinking" // AI thinking process
  | "tool" // Tool/function call
  | "answer" // AI response
  | "error" // Error state
  | "edit" // Edit action (Memo)
  | "archive"; // Archive action (Memo)

/**
 * Base theme configuration for timeline blocks
 */
export interface BlockTheme {
  border: string;
  headerBg: string;
  footerBg: string;
  badgeBg: string;
  badgeText: string;
  ringColor: string;
}

/**
 * Generic timeline block props
 */
export interface TimelineBlockProps<T> {
  item: T;
  isExpanded: boolean;
  onToggle: () => void;
  isLatest?: boolean;
  isStreaming?: boolean;
  theme?: BlockTheme;
  className?: string;
}

/**
 * Block action for footer buttons
 */
export interface BlockAction {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  onClick: () => void;
  variant?: "default" | "danger" | "success";
  showOnMobile?: boolean; // Show as icon-only on mobile
  isStreaming?: boolean; // Only show during streaming
}

/**
 * Timeline event for chronological ordering
 */
export interface TimelineEvent {
  id: string;
  type: TimelineNodeType;
  timestamp: number;
  data: unknown;
}
