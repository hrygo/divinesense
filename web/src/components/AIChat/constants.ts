/**
 * UnifiedMessageBlock Constants
 *
 * Centralized configuration for the UnifiedMessageBlock component
 */

// ============================================================
// Timestamp Calculation Constants
// ============================================================

/** Multiplier for calculating round-based timestamps */
export const ROUND_TIMESTAMP_MULTIPLIER = 1_000_000;

/** Offset in milliseconds between tool calls in the same round */
export const TOOL_CALL_OFFSET_MS = 1000;

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
