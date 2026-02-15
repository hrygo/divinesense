import { create } from "@bufbuild/protobuf";
import { useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import {
  DangerBlockEvent,
  type EventMetadata,
  MemoQueryResultData,
  ParrotAgentType,
  ParrotChatCallbacks,
  ParrotChatParams,
  ParrotEventType,
  parrotToProtoAgentType,
  ScheduleQueryResultData,
} from "@/types/parrot";
import type { EventMetadata as ProtoEventMetadata } from "@/types/proto/api/v1/ai_service_pb";
import { ChatRequestSchema } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Convert protobuf EventMetadata to local EventMetadata for UI callbacks.
 * Handles bigint to number conversion and optional field mapping.
 */
function protoToEventMetadata(proto: ProtoEventMetadata): EventMetadata | undefined {
  if (!proto?.toolName) return undefined;
  return {
    durationMs: proto.durationMs ? Number(proto.durationMs) : undefined,
    totalDurationMs: proto.totalDurationMs ? Number(proto.totalDurationMs) : undefined,
    toolName: proto.toolName,
    toolId: proto.toolId || undefined,
    status: proto.status,
    errorMsg: proto.errorMsg || undefined,
    inputTokens: proto.inputTokens,
    outputTokens: proto.outputTokens,
    cacheWriteTokens: proto.cacheWriteTokens,
    cacheReadTokens: proto.cacheReadTokens,
    inputSummary: proto.inputSummary,
    outputSummary: proto.outputSummary,
    filePath: proto.filePath,
    lineCount: proto.lineCount,
  };
}

/**
 * useParrotChat provides a hook for chatting with parrot agents.
 *
 * @example
 * ```tsx
 * const parrotChat = useParrotChat();
 *
 * const handleChat = async () => {
 *   await parrotChat.streamChat(
 *     {
 *       agentType: ParrotAgentType.MEMO,
 *       message: "查询 Python 笔记"
 *     },
 *     {
 *       onContent: (content) => console.log(content),
 *       onMemoQueryResult: (result) => console.log(result.memos),
 *       onDone: () => console.log("Done")
 *     }
 *   );
 * };
 * ```
 */
export function useParrotChat() {
  const queryClient = useQueryClient();

  return {
    /**
     * Stream chat with a parrot agent.
     *
     * @param params - Chat parameters including agent type and message
     * @param callbacks - Optional callbacks for streaming events
     * @returns A promise that resolves when streaming completes
     */
    streamChat: async (params: ParrotChatParams, callbacks?: ParrotChatCallbacks) => {
      const request = create(ChatRequestSchema, {
        message: params.message,
        // history field removed - backend-driven context construction (context-engineering.md Phase 1)
        agentType: parrotToProtoAgentType(params.agentType),
        userTimezone: params.userTimezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone,
        conversationId: params.conversationId ?? 0,
      });

      try {
        // Use the streaming method from Connect RPC client
        const stream = aiServiceClient.chat(request);

        let fullContent = "";
        let doneCalled = false;

        for await (const response of stream) {
          // Handle parrot-specific events (eventType and eventData)
          if (response.eventType && response.eventData) {
            handleParrotEvent(response.eventType, response.eventData, response.eventMeta, callbacks);
          }

          // Handle content chunks (for backward compatibility)
          if (response.content) {
            fullContent += response.content;
            callbacks?.onContent?.(response.content);
          }

          // Handle completion
          if (response.done === true) {
            doneCalled = true;
            callbacks?.onDone?.();
            break;
          }
        }

        // Fallback: if stream ended without done signal, call onDone
        if (!doneCalled) {
          callbacks?.onDone?.();
        }

        return { content: fullContent };
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        callbacks?.onError?.(err);
        throw err;
      }
    },

    /**
     * Invalidate parrot-related queries after chat
     */
    invalidate: async () => {
      await queryClient.invalidateQueries({ queryKey: ["parrot"] });
    },
  };
}

/**
 * Handle parrot-specific events from the server.
 *
 * @param eventType - The type of event
 * @param eventData - The event data (JSON string or plain text)
 * @param eventMeta - The event metadata (structured data from EventWithMeta)
 * @param callbacks - Optional callbacks to handle events
 */
function handleParrotEvent(eventType: string, eventData: string, eventMeta?: ProtoEventMetadata, callbacks?: ParrotChatCallbacks) {
  try {
    switch (eventType) {
      case ParrotEventType.THINKING:
        callbacks?.onThinking?.(eventData);
        break;

      case ParrotEventType.TOOL_USE:
        // Use eventMeta if available (Normal mode with EventWithMeta)
        // For backward compatibility, fall back to eventData if meta is missing
        if (eventMeta?.toolName) {
          const meta = protoToEventMetadata(eventMeta);
          callbacks?.onToolUse?.(eventMeta.toolName, meta);
        } else {
          // Backward compatibility: old format without meta
          callbacks?.onToolUse?.(eventData);
        }
        break;

      case ParrotEventType.TOOL_RESULT:
        // Use eventMeta if available (Normal mode with EventWithMeta)
        if (eventMeta?.toolName) {
          const meta = protoToEventMetadata(eventMeta);
          callbacks?.onToolResult?.(eventData, meta);
        } else {
          // Backward compatibility: old format without meta
          callbacks?.onToolResult?.(eventData);
        }
        break;

      case ParrotEventType.DANGER_BLOCK:
        try {
          const result = JSON.parse(eventData) as DangerBlockEvent;
          callbacks?.onDangerBlock?.(result);
        } catch (parseError) {
          const err = parseError instanceof Error ? parseError : new Error(String(parseError));
          console.error("Failed to parse danger block event:", err);
          console.error("Event data:", eventData);
          // Include truncated raw data in error message for debugging
          const rawDataPreview = eventData.length > 100 ? eventData.substring(0, 100) + "..." : eventData;
          callbacks?.onError?.(new Error(`Failed to parse danger block event: ${err.message}. Raw data: ${rawDataPreview}`));
        }
        break;

      case ParrotEventType.MEMO_QUERY_RESULT:
        try {
          const result = JSON.parse(eventData) as MemoQueryResultData;
          callbacks?.onMemoQueryResult?.(result);
        } catch (parseError) {
          const err = parseError instanceof Error ? parseError : new Error(String(parseError));
          console.error("Failed to parse memo query result:", err);
          console.error("Event data:", eventData);
          const rawDataPreview = eventData.length > 100 ? eventData.substring(0, 100) + "..." : eventData;
          callbacks?.onError?.(new Error(`Failed to parse memo query result: ${err.message}. Raw data: ${rawDataPreview}`));
        }
        break;

      case ParrotEventType.SCHEDULE_QUERY_RESULT:
        try {
          const result = JSON.parse(eventData) as ScheduleQueryResultData;
          callbacks?.onScheduleQueryResult?.(result);
        } catch (parseError) {
          const err = parseError instanceof Error ? parseError : new Error(String(parseError));
          console.error("Failed to parse schedule query result:", err);
          console.error("Event data:", eventData);
          const rawDataPreview = eventData.length > 100 ? eventData.substring(0, 100) + "..." : eventData;
          callbacks?.onError?.(new Error(`Failed to parse schedule query result: ${err.message}. Raw data: ${rawDataPreview}`));
        }
        break;

      case ParrotEventType.SCHEDULE_UPDATED:
        break;

      case ParrotEventType.ERROR: {
        const error = new Error(eventData);
        callbacks?.onError?.(error);
        break;
      }

      case ParrotEventType.ANSWER:
        // Final answer (already handled by content chunks)
        break;

      default:
        console.warn("Unknown parrot event type:", eventType, eventData);
    }
  } catch (error) {
    console.error("Error handling parrot event:", error);
    // Propagate unexpected errors to the callback
    const err = error instanceof Error ? error : new Error(String(error));
    callbacks?.onError?.(err);
  }
}

/**
 * Query keys factory for parrot-related queries
 * Note: history key removed - history now built by backend (context-engineering.md Phase 1)
 */
export const parrotKeys = {
  all: ["parrot"] as const,
  chat: (agentType: ParrotAgentType) => [...parrotKeys.all, "chat", agentType] as const,
};
