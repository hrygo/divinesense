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
 * Extract thinking steps from Block event stream
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
      // Parse meta to get round number (if available)
      let round = 0;
      if (event.meta) {
        try {
          const meta = JSON.parse(event.meta);
          round = typeof meta.round === "number" ? meta.round : 0;
        } catch {
          // Invalid JSON, use default round
          round = 0;
        }
      }

      steps.push({
        content: event.content,
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
 * @param eventStream - Array of BlockEvent objects
 * @returns Array of ToolCall objects
 */
export function extractToolCalls(eventStream: BlockEvent[] | undefined): ToolCall[] {
  if (!eventStream || eventStream.length === 0) {
    return [];
  }

  const toolCallsMap = new Map<string, ToolCall>();

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
      // Parse tool_use event
      let meta: Record<string, unknown> | undefined;
      try {
        meta = event.meta ? JSON.parse(event.meta) : undefined;
      } catch {
        meta = undefined;
      }

      const toolCall: ToolCall = {
        name: asString(meta?.name) || event.content || "unknown",
        toolId: asString(meta?.tool_id),
        inputSummary: event.content || asString(meta?.input_summary),
      };

      const toolId = toolCall.toolId || toolCall.name;
      toolCallsMap.set(toolId, toolCall);
    } else if (event.type === "tool_result") {
      // Parse tool_result event
      let meta: Record<string, unknown> | undefined;
      try {
        meta = event.meta ? JSON.parse(event.meta) : undefined;
      } catch {
        meta = undefined;
      }

      const toolId = asString(meta?.tool_id);
      if (toolId && toolCallsMap.has(toolId)) {
        // Update existing tool call with result
        const existing = toolCallsMap.get(toolId)!;
        existing.outputSummary = event.content || asString(meta?.output_summary);
        existing.isError = asBool(meta?.is_error);
        existing.duration = asNumber(meta?.duration);
        existing.exitCode = asNumber(meta?.exitCode);
        existing.filePath = asString(meta?.file_path);
      }
    }
  }

  return Array.from(toolCallsMap.values());
}
