/**
 * Text Utilities - Sticky Note Preview
 *
 * 文本处理工具：用于便签摘要展示
 *
 * Features:
 * - Markdown 语法清理
 * - 智能截断（按词/句）
 * - 长度限制
 */

// ============================================================================
// Types
// ============================================================================

export interface TruncateOptions {
  maxLength?: number;
  preserveLinks?: boolean;
  ellipsis?: string;
}

// ============================================================================
// Constants
// ============================================================================

/** Maximum input length for stripMarkdown to prevent ReDoS (50KB) */
const MAX_MARKDOWN_INPUT_LENGTH = 50000;

// ============================================================================
// Markdown Cleaning
// ============================================================================

/**
 * Remove Markdown syntax from text for plain preview
 *
 * Note: For very long content (>10KB), consider using a dedicated markdown parser
 * for better performance. Current implementation uses regex which may have
 * backtracking issues on extremely long inputs.
 *
 * @param text - Raw text (may contain Markdown)
 * @param options - Processing options
 * @returns Clean text without Markdown syntax
 */
export function stripMarkdown(text: string, options?: { preserveLinks?: boolean }): string {
  // Prevent ReDoS by limiting input length
  let result = text.length > MAX_MARKDOWN_INPUT_LENGTH ? text.slice(0, MAX_MARKDOWN_INPUT_LENGTH) : text;

  // Remove headers (# ## ### etc.)
  result = result.replace(/^#{1,6}\s+/gm, "");

  // Remove bold/italic (**bold** *italic* ___underline___)
  result = result.replace(/\*\*\*(.+?)\*\*\*/g, "$1");
  result = result.replace(/\*\*(.+?)\*\*/g, "$1");
  result = result.replace(/\*(.+?)\*/g, "$1");
  result = result.replace(/__(.+?)__/g, "$1");
  result = result.replace(/_(.+?)_/g, "$1");

  // Remove strikethrough (~~text~~)
  result = result.replace(/~~(.+?)~~/g, "$1");

  // Remove inline code (`code`)
  result = result.replace(/`([^`]+)`/g, "$1");

  // Remove code blocks (```code```)
  result = result.replace(/```[\s\S]*?```/g, (match) => {
    // Keep first line of code block
    const lines = match.replace(/```\w*\n?/g, "").split("\n");
    return lines.length > 0 ? `[code: ${lines[0].slice(0, 30)}...]` : "";
  });

  // Handle links
  if (options?.preserveLinks) {
    // Convert [text](url) to text (url)
    result = result.replace(/\[(.+?)\]\((.+?)\)/g, "$1 ($2)");
  } else {
    // Remove links, keep text
    result = result.replace(/\[(.+?)\]\((.+?)\)/g, "$1");
  }

  // Remove images ![alt](url)
  result = result.replace(/!\[.*?\]\(.+?\)/g, "[image]");

  // Remove blockquotes (> quote)
  result = result.replace(/^>\s+/gm, "");

  // Remove horizontal rules (--- *** ___)
  result = result.replace(/^[-*_]{3,}\s*$/gm, "");

  // Remove list markers (- * + 1.)
  result = result.replace(/^[\s]*[-*+]\s+/gm, "");
  result = result.replace(/^[\s]*\d+\.\s+/gm, "");

  // Remove checkbox markers ([ ] [x])
  result = result.replace(/\[[ x]\]\s*/gi, "");

  // Clean up extra whitespace
  result = result.replace(/\n{3,}/g, "\n\n");
  result = result.trim();

  return result;
}

// ============================================================================
// Smart Truncation
// ============================================================================

/**
 * Find a good break point for truncation
 */
function findBreakPoint(text: string, maxLength: number): number {
  // Look for sentence end within range
  const sentenceEnders = ["。", "！", "？", ".", "!", "?"];
  const searchStart = Math.max(0, maxLength - 50);
  const searchEnd = Math.min(text.length, maxLength + 20);

  for (let i = searchEnd; i >= searchStart; i--) {
    if (sentenceEnders.includes(text[i])) {
      return i + 1;
    }
  }

  // Look for word boundary (space, comma, etc.)
  for (let i = searchEnd; i >= searchStart; i--) {
    if (/[\s,，、;；]/.test(text[i])) {
      return i;
    }
  }

  // No good break point found, hard truncate
  return maxLength;
}

/**
 * Truncate text to specified length with smart breaking
 */
export function truncateText(text: string, maxLength: number = 200, ellipsis: string = "..."): string {
  if (text.length <= maxLength) {
    return text;
  }

  const breakPoint = findBreakPoint(text, maxLength);
  return text.slice(0, breakPoint).trim() + ellipsis;
}

// ============================================================================
// Main API
// ============================================================================

/**
 * Generate preview text for sticky note
 *
 * @param content - Raw memo content (may contain Markdown)
 * @param options - Truncation options
 * @returns Clean preview text ready for display
 */
export function generatePreview(content: string, options: TruncateOptions = {}): string {
  const { maxLength = 200, preserveLinks = false, ellipsis = "..." } = options;

  // Strip Markdown syntax
  const cleanText = stripMarkdown(content, { preserveLinks });

  // Truncate to max length
  return truncateText(cleanText, maxLength, ellipsis);
}

/**
 * Count visible characters (excluding Markdown syntax)
 */
export function countVisibleChars(text: string): number {
  return stripMarkdown(text).length;
}

/**
 * Check if content is long enough to need truncation
 */
export function needsTruncation(content: string, maxLength: number = 200): boolean {
  return countVisibleChars(content) > maxLength;
}

/**
 * Extract first line as title/preview
 */
export function extractTitle(content: string, maxLength: number = 50): string {
  const firstLine = content.split("\n")[0];
  const cleanTitle = stripMarkdown(firstLine);

  if (cleanTitle.length <= maxLength) {
    return cleanTitle;
  }

  return truncateText(cleanTitle, maxLength, "...");
}
