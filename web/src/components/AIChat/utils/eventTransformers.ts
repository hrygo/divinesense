/**
 * Event Transformation Utilities
 *
 * Helper functions to transform BlockEvent[] into UI-friendly formats.
 * Used by ChatMessages component to extract thinking steps and tool calls
 * from the Block event stream.
 */

import type { BlockEvent } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Thinking step extracted from event stream
 */
export interface ThinkingStep {
  content: string;
  timestamp: number;
  round: number; // 第几轮思考（0-based）
}

/**
 * Tool call extracted from event stream
 */
export interface ToolCall {
  name: string;
  toolId?: string;
  inputSummary?: string;
  outputSummary?: string;
  filePath?: string;
  duration?: number;
  exitCode?: number;
  isError?: boolean;
}

/**
 * Normalize timestamp from bigint or number to number
 *
 * @param timestamp - bigint or number timestamp
 * @returns number timestamp in milliseconds
 */
export function normalizeTimestamp(timestamp: bigint | number | undefined): number {
  if (timestamp === undefined || timestamp === null) {
    return Date.now();
  }
  return typeof timestamp === "bigint" ? Number(timestamp) : timestamp;
}

/**
 * Parse event content that may be in JSON format.
 * JSON format: {"data": "...", "meta": {...}}
 *
 * @param content - Raw event content string
 * @returns Parsed content and optional meta object
 */
function parseEventContent(content: string): { data: string; meta?: Record<string, unknown> } {
  const trimmed = content.trim();

  // Try to parse as JSON if it starts with '{'
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed && typeof parsed === "object") {
        return {
          data: typeof parsed.data === "string" ? parsed.data : "",
          meta: parsed.meta as Record<string, unknown> | undefined,
        };
      }
    } catch {
      // Not valid JSON, return as-is
    }
  }

  // Plain text content
  return { data: trimmed };
}

/**
 * Extract thinking steps from Block event stream
 *
 * Handles two content formats:
 * 1. Plain text: "thinking content..."
 * 2. JSON format (from orchestrator): {"data": "...", "meta": {...}}
 *
 * @param eventStream - Array of BlockEvent objects
 * @returns Array of ThinkingStep objects
 */
export function extractThinkingSteps(eventStream: BlockEvent[] | undefined): ThinkingStep[] {
  if (!eventStream || eventStream.length === 0) {
    return [];
  }

  const steps: ThinkingStep[] = [];

  for (const event of eventStream) {
    if (event.type === "thinking" && event.content) {
      // Parse content (handle JSON format)
      const { data: content, meta } = parseEventContent(event.content);

      // Skip empty content
      if (!content) {
        continue;
      }

      // Parse round from meta (prefer event-level meta, fallback to content meta)
      let round = 0;
      const metaToParse = event.meta || meta;
      if (metaToParse) {
        try {
          const parsed = typeof metaToParse === "string" ? JSON.parse(metaToParse) : metaToParse;
          round = typeof parsed.round === "number" ? parsed.round : 0;
        } catch {
          // Invalid JSON, use default round
          round = 0;
        }
      }

      steps.push({
        content,
        timestamp: normalizeTimestamp(event.timestamp),
        round,
      });
    }
  }

  return steps;
}

/**
 * Extract tool calls from Block event stream
 *
 * Parses tool_use and tool_result events to build a complete picture
 * of tool invocations and their results.
 *
 * Uses occurrence-based deduplication: if the same tool (by name) appears
 * multiple times without a unique tool_id, only the last occurrence is kept.
 *
 * @param eventStream - Array of BlockEvent objects
 * @returns Array of ToolCall objects
 */
export function extractToolCalls(eventStream: BlockEvent[] | undefined): ToolCall[] {
  if (!eventStream || eventStream.length === 0) {
    return [];
  }

  // Map to store tool calls, using a deduplication key
  // Priority: tool_id > name > occurrence-index
  const toolCallsMap = new Map<string, { toolCall: ToolCall; occurrence: number }>();

  // Helper to safely extract string from unknown
  const asString = (val: unknown): string | undefined => {
    if (typeof val === "string") return val;
    return undefined;
  };

  // Helper to safely extract boolean from unknown
  const asBool = (val: unknown): boolean | undefined => {
    if (typeof val === "boolean") return val;
    return undefined;
  };

  // Helper to safely extract number from unknown
  const asNumber = (val: unknown): number | undefined => {
    if (typeof val === "number") return val;
    return undefined;
  };

  for (const event of eventStream) {
    if (event.type === "tool_use") {
      // Parse tool_use event - handle JSON format in content
      // JSON format: {"data": "...", "meta": {...}}
      let meta: Record<string, unknown> | undefined;
      let contentData = event.content;

      // First try to parse content as JSON to extract meta
      if (event.content.trim().startsWith("{")) {
        try {
          const parsed = JSON.parse(event.content);
          if (parsed && typeof parsed === "object") {
            contentData = typeof parsed.data === "string" ? parsed.data : event.content;
            meta = parsed.meta as Record<string, unknown> | undefined;
          }
        } catch {
          // Not valid JSON, fall through to regular meta parsing
        }
      }

      // If meta not from JSON content, try event.meta
      if (!meta && event.meta) {
        try {
          meta = JSON.parse(event.meta);
        } catch {
          meta = undefined;
        }
      }

      const toolName = asString(meta?.tool_name) || asString(meta?.name) || contentData || "unknown";
      const toolId = asString(meta?.tool_id);
      const occurrence = asNumber(meta?.occurrence) ?? 0;

      // Build deduplication key:
      // - If tool_id exists: use it as unique key
      // - Otherwise: use name + occurrence to distinguish multiple calls to same tool
      const dedupeKey = toolId ? `id:${toolId}` : `name:${toolName}:occ:${occurrence}`;

      const toolCall: ToolCall = {
        name: toolName,
        toolId,
        // Prefer meta.input_summary over contentData
        inputSummary: asString(meta?.input_summary) || contentData,
      };

      // Store tool call with its occurrence
      toolCallsMap.set(dedupeKey, { toolCall, occurrence });
    } else if (event.type === "tool_result") {
      // Parse tool_result event - handle JSON format in content
      let meta: Record<string, unknown> | undefined;
      let contentData = event.content;

      // First try to parse content as JSON to extract meta
      if (event.content.trim().startsWith("{")) {
        try {
          const parsed = JSON.parse(event.content);
          if (parsed && typeof parsed === "object") {
            contentData = typeof parsed.data === "string" ? parsed.data : event.content;
            meta = parsed.meta as Record<string, unknown> | undefined;
          }
        } catch {
          // Not valid JSON, fall through to regular meta parsing
        }
      }

      // If meta not from JSON content, try event.meta
      if (!meta && event.meta) {
        try {
          meta = JSON.parse(event.meta);
        } catch {
          meta = undefined;
        }
      }

      const toolId = asString(meta?.tool_id);
      const occurrence = asNumber(meta?.occurrence) ?? 0;

      // Find matching tool_use event by same dedupe key
      // Try both tool_name (preferred) and name (legacy compatibility)
      const toolNameForKey = asString(meta?.tool_name) || asString(meta?.name);
      const resultKey = toolId ? `id:${toolId}` : `name:${toolNameForKey}:occ:${occurrence}`;

      if (toolCallsMap.has(resultKey)) {
        // Update existing tool call with result
        const existing = toolCallsMap.get(resultKey)!.toolCall;
        existing.outputSummary = contentData || asString(meta?.output_summary);
        existing.isError = asBool(meta?.is_error);
        existing.duration = asNumber(meta?.duration);
        existing.exitCode = asNumber(meta?.exit_code);
        existing.filePath = asString(meta?.file_path);
      }
    }
  }

  // Return tool calls sorted by occurrence (if available) or insertion order
  const result = Array.from(toolCallsMap.values())
    .sort((a, b) => a.occurrence - b.occurrence)
    .map((item) => item.toolCall);

  return result;
}
