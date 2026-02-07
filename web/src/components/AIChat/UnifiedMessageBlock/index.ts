/**
 * UnifiedMessageBlock Module
 *
 * Clean architecture implementation of the AI chat message block.
 *
 * Export Structure:
 * - components: React components
 * - hooks: Custom React hooks
 * - utils: Utility functions
 * - types: TypeScript types
 * - constants: Configuration constants
 */

export type {
  BlockFooterProps,
  BlockFooterTheme,
  BlockHeaderProps,
  BlockHeaderTheme,
} from "./components";
// Components
export {
  BlockFooter,
  BlockHeader,
  InlineToolCall,
  PendingSkeleton,
  StreamingProgressBar,
  TimelineNode,
  ToolCallCard,
  useVirtualBlockList,
  VirtualBlockList,
} from "./components";
// Constants
export {
  BREAKPOINTS,
  COLLAPSE_PREVIEW_MAX_CHARS,
  COLLAPSE_PREVIEW_MAX_LINES,
  KEYBOARD_SHORTCUTS,
  NODE_COLORS,
  RESPONSIVE_CONFIG,
  STREAMING_PROGRESS_ANIMATION_MS,
  STREAMING_PROGRESS_HEIGHT,
  TIMELINE_NODE_CONFIG,
  VIRTUAL_RENDER_STAY_TIME,
  VIRTUAL_SCROLL_OVERSCAN,
} from "./constants";
// Hooks
export {
  useBlockCollapse,
  useKeyboardNav,
  useStreamingProgress,
} from "./hooks";

// Types
export type {
  InlineToolCallProps,
  PendingSkeletonProps,
  StreamingProgressBarProps,
  TimelineNodeProps,
  TimelineNodeType,
  ToolCallCardProps,
  UseBlockCollapseOptions,
  UseBlockCollapseReturn,
  UseKeyboardNavOptions,
  UseKeyboardNavReturn,
  UseStreamingProgressOptions,
  UseStreamingProgressReturn,
} from "./types";
// Utils
export {
  buildCopyContent,
  extractToolName,
  extractUserInitial,
  formatRelativeTime,
  generatePreviewText,
  getVisualWidth,
  truncateByVisualWidth,
} from "./utils";
