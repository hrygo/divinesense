/**
 * UnifiedMessageBlock Constants
 *
 * Centralized configuration for the UnifiedMessageBlock component
 */

// ============================================================
// Timestamp Calculation Constants
// ============================================================

/** Multiplier for calculating round-based timestamps (converts rounds to microseconds) */
export const ROUND_TIMESTAMP_MULTIPLIER = 1_000_000;

/** Offset in microseconds between tool calls in the same round */
export const TOOL_CALL_OFFSET_US = 1000;

// ============================================================
// UI Threshold Constants
// ============================================================

/** User inputs expand threshold (character count) */
export const USER_INPUTS_EXPAND_THRESHOLD = 300;

/** Header visual width for truncation (~12 Chinese or ~24 English characters) */
export const HEADER_VISUAL_WIDTH = 24;

/** Width reduction for multi-input badge display */
export const BADGE_WIDTH_OFFSET = 4;

// ============================================================
// Animation Duration Constants
// ============================================================

/** Breathing animation duration in milliseconds */
export const BREATHE_ANIMATION_DURATION_MS = 2000;

/** Pulse animation duration in milliseconds */
export const PULSE_ANIMATION_DURATION_MS = 2000;

/** Copy feedback duration in milliseconds */
export const COPY_FEEDBACK_DURATION_MS = 2000;

// ============================================================
// Cache Limits
// ============================================================

/** Maximum number of messages to cache per conversation (FIFO) */
export const MESSAGE_CACHE_LIMIT = 100;

// ============================================================
// Type Guards
// ============================================================

/** Check if a value is a tool call object (not a string) */
export function isToolCallObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

// ============================================================
// Timeline Node Configuration (Phase 1: Visual Hierarchy)
// ============================================================

/**
 * Unified timeline node styling configuration
 * Ensures all timeline nodes use consistent sizing and styling
 */
export const TIMELINE_NODE_CONFIG = {
  /** Base size classes for all timeline nodes */
  size: "w-6 h-6",
  /** Border width for all nodes */
  border: "border-2",
  /** Border radius for circular nodes */
  radius: "rounded-full",
  /** Icon size within nodes */
  iconSize: "w-3.5 h-3.5",
} as const;

/**
 * Color schemes for different timeline node types
 */
export const NODE_COLORS = {
  /** User input node - blue theme */
  user: "bg-blue-100 dark:bg-blue-900/40 border-blue-500 text-blue-600 dark:text-blue-400",
  /** Thinking/processing node - purple theme */
  thinking: "bg-purple-100 dark:bg-purple-900/40 border-purple-500 text-purple-600 dark:text-purple-400",
  /** Tool call node - neutral with hover effect */
  tool: "bg-card border-border group-hover:border-purple-400/50 transition-colors",
  /** AI answer node - zinc theme (neutral gray, distinct from other modes) */
  answer: "bg-zinc-50 dark:bg-zinc-800/40 border-zinc-500 text-zinc-600 dark:text-zinc-400",
  /** Error node - red theme */
  error: "bg-red-100 dark:bg-red-900/30 border-red-500 text-red-600 dark:text-red-400",
  /** Edit node - green theme (for Memo timeline) */
  edit: "bg-green-100 border-green-500 dark:bg-green-900/40 dark:border-green-400 text-green-600 dark:text-green-400",
  /** Archive node - zinc/gray theme (for Memo timeline) */
  archive: "bg-zinc-100 border-zinc-500 dark:bg-zinc-800 dark:border-zinc-400 text-zinc-600 dark:text-zinc-400",
} as const;

// ============================================================
// Responsive Configuration (Phase 2: Responsive Experience)
// ============================================================

/**
 * Breakpoint values matching Tailwind CSS defaults
 */
export const BREAKPOINTS = {
  sm: 640, // Mobile landscape
  md: 768, // Tablet
  lg: 1024, // Desktop
  xl: 1280, // Large desktop
} as const;

/**
 * Responsive behavior configuration
 * Defines what UI elements show/hide at different breakpoints
 */
export const RESPONSIVE_CONFIG = {
  /** Mobile breakpoint (< 768px) */
  mobile: {
    /** Hide statistics in header */
    hideStats: true,
    /** Hide status badges in header */
    hideBadge: true,
    /** Show only icon for footer buttons */
    iconOnly: true,
    /** Show single key stat (cost or latency) */
    singleStat: true,
  },
  /** Desktop breakpoint (≥ 768px) */
  desktop: {
    /** Show all statistics in header */
    showStats: true,
    /** Show status badges in header */
    showBadge: true,
    /** Show icon + label for footer buttons */
    showLabels: true,
    /** Show all available stats */
    allStats: true,
  },
} as const;

// ============================================================
// Collapse Preview Configuration (Phase 1: Visual Hierarchy)
// ============================================================

/** Maximum characters to show in collapsed block preview */
export const COLLAPSE_PREVIEW_MAX_CHARS = 100;

/** Maximum lines to show in collapsed block preview */
export const COLLAPSE_PREVIEW_MAX_LINES = 2;

// ============================================================
// Streaming Progress Configuration (Phase 3: Interaction Feedback)
// ============================================================

/** Height of streaming progress bar in pixels */
export const STREAMING_PROGRESS_HEIGHT = 4; // 1 border unit = 4px

/** Streaming progress animation duration in milliseconds */
export const STREAMING_PROGRESS_ANIMATION_MS = 300;

// ============================================================
// Keyboard Navigation Configuration (Phase 4: Accessibility)
// ============================================================

/** Keyboard shortcuts for UnifiedMessageBlock */
export const KEYBOARD_SHORTCUTS = {
  /** Move focus to next block */
  NEXT_BLOCK: "Tab",
  /** Move focus to previous block */
  PREV_BLOCK: "Shift+Tab",
  /** Copy current block content */
  COPY: "Ctrl+C",
  /** Edit current block */
  EDIT: "Ctrl+E",
  /** Send message */
  SEND: "Ctrl+Enter",
  /** Cancel/close dialog */
  CANCEL: "Escape",
} as const;

// ============================================================
// Virtual Scrolling Configuration (Phase 5: Performance)
// ============================================================

/** Root margin for IntersectionObserver (pixels before viewport) */
export const VIRTUAL_SCROLL_OVERSCAN = 200;

/** Minimum time a block should stay rendered after leaving viewport (ms) */
export const VIRTUAL_RENDER_STAY_TIME = 1000;
