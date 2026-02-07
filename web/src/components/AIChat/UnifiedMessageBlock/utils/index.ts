/**
 * UnifiedMessageBlock Utility Functions
 */

import { COLLAPSE_PREVIEW_MAX_CHARS } from "../constants";

/**
 * Calculate visual width of a string for truncation
 * - ASCII characters = 1
 * - CJK characters = 2
 * - Emoji = 2
 */
export function getVisualWidth(str: string): number {
  let width = 0;
  const chars = [...str];

  for (const char of chars) {
    const code = char.codePointAt(0) || 0;

    // Skip zero-width characters
    if (code === 0x200b || code === 0xfe0e || code === 0xfe0f || (code >= 0x1f3fb && code <= 0x1f3ff)) {
      continue;
    }

    // ASCII: 0-127
    if (code < 128) {
      width += 1;
    } else if (
      code >= 0x1100 &&
      (code <= 0x11ff || (code >= 0x2e80 && code <= 0x9fff) || (code >= 0x3400 && code <= 0x4dbf) || (code >= 0x20000 && code <= 0x2ebef))
    ) {
      // CJK characters = 2
      width += 2;
    } else {
      // Other characters (including emoji) = 2
      width += 2;
    }
  }

  return width;
}

/**
 * Truncate string by visual width
 */
export function truncateByVisualWidth(str: string, maxVisualWidth: number): string {
  let currentWidth = 0;
  let result = "";

  for (const char of str) {
    const code = char.codePointAt(0) || 0;
    const charWidth = code < 128 ? 1 : 2;

    if (currentWidth + charWidth > maxVisualWidth) {
      return result + "...";
    }

    result += char;
    currentWidth += charWidth;
  }

  return result;
}

/**
 * Generate preview text from content for collapsed state
 * Strips markdown and HTML, limits to max characters
 */
export function generatePreviewText(content: string, maxChars: number = COLLAPSE_PREVIEW_MAX_CHARS): string {
  if (!content) return "";

  // Strip markdown and HTML for preview
  const plainText = content
    .replace(/```[\s\S]*?```/g, "[Code]")
    .replace(/\[.*?\]\(.*?\)/g, "[Link]")
    .replace(/[#*_`~[\]]/g, "")
    .replace(/\s+/g, " ")
    .trim();

  if (plainText.length <= maxChars) {
    return plainText;
  }

  return plainText.slice(0, maxChars) + "...";
}

/**
 * Extract user initial from content
 */
export function extractUserInitial(content: string): string {
  const trimmed = content.trim();
  if (trimmed.length === 0) return "U";

  const match = trimmed.match(/[a-zA-Z\u4e00-\u9fa5]/);
  return match ? match[0].toUpperCase() : "U";
}

/**
 * Format timestamp to relative time string
 */
export function formatRelativeTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return t("ai.aichat.sidebar.time-just-now");
  if (diffMins < 60) return t("ai.aichat.sidebar.time-minutes-ago", { count: diffMins });
  if (diffMins < 1440) return t("ai.aichat.sidebar.time-hours-ago", { count: Math.floor(diffMins / 60) });

  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

/**
 * Extract pure tool name from function call string
 * Handles: "search_files(query=\"xxx\")" -> { displayName: "search_files", fullCall: "search_files(query=\"xxx\")" }
 */
export function extractToolName(callName: string): { displayName: string; fullCall: string } {
  const match = callName.match(/^([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
  if (match) {
    return { displayName: match[1], fullCall: callName };
  }
  return { displayName: callName, fullCall: callName };
}

/**
 * Build content for copying (user + assistant messages)
 */
export function buildCopyContent(
  userMessage: { content: string },
  additionalUserInputs?: Array<{ content: string }>,
  assistantMessage?: { content?: string; metadata?: { toolResults?: Array<{ name: string; duration?: number }> } },
): string {
  const userContents = [userMessage.content, ...(additionalUserInputs?.map((m) => m.content) || [])];

  return [
    `User: ${userContents.join("\n> ")}`,
    assistantMessage?.content ? `Assistant: ${assistantMessage.content}` : "",
    assistantMessage?.metadata?.toolResults
      ? `\n\nTools:\n${assistantMessage.metadata.toolResults.map((r) => `- ${r.name}: ${r.duration || 0}ms`).join("\n")}`
      : "",
  ]
    .filter(Boolean)
    .join("\n\n");
}
