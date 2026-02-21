import { useMutation, useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import { AgentType } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Hook to chat with Schedule Agent (non-streaming)
 * Uses AIService.Chat with AGENT_TYPE_SCHEDULE
 * Note: History is now built by backend using conversationId (context-engineering.md Phase 1)
 */

interface ScheduleAgentChatRequest {
  message: string;
  userTimezone?: string;
  conversationId?: number; // Backend will build history from this ID
}

/**
 * Parsed event from the agent stream
 */
export interface ParsedEvent {
  type: string;
  data: string;
}

/**
 * Hook to chat with Schedule Agent (non-streaming)
 * Delegates to AIService.Chat with agentType=SCHEDULE
 */
export function useScheduleAgentChat() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: ScheduleAgentChatRequest) => {
      // Backend-driven context: use conversationId to build history
      // No longer use frontend-provided history (context-engineering.md Phase 1)

      // Use AIService.Chat with SCHEDULE agent type
      let response = "";
      for await (const chunk of aiServiceClient.chat({
        message: request.message,
        userTimezone: request.userTimezone || "Asia/Shanghai",
        agentType: AgentType.SCHEDULE,
        conversationId: request.conversationId,
      })) {
        if (chunk.content) {
          response += chunk.content;
        }
      }
      return { response };
    },
    onSuccess: () => {
      // Invalidate schedule lists to refetch
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    },
  });
}

/**
 * Parse an event JSON string into a ParsedEvent
 * Handles both JSON format and plain text events from backend
 */
export function parseEvent(eventData: string, eventType?: string): ParsedEvent | null {
  if (!eventData) {
    return null;
  }

  // Try to parse as JSON first
  try {
    const event = JSON.parse(eventData);
    return {
      type: event.type || eventType || "unknown",
      data: event.data || eventData,
    };
  } catch {
    // If JSON parsing fails, treat the entire string as data
    // This handles plain text events like "thinking" and "answer" from backend
    return {
      type: eventType || "unknown",
      data: eventData,
    };
  }
}

/**
 * Hook to chat with Schedule Agent (streaming)
 * Uses AIService.Chat with AGENT_TYPE_SCHEDULE
 * Returns an async generator that yields stream events
 */
export async function* scheduleAgentChatStream(
  message: string,
  userTimezone = "Asia/Shanghai",
  onEvent?: (event: { type: string; data: string }) => void,
): AsyncGenerator<{ type: string; data: string; content?: string; done?: boolean }, void> {
  const response = aiServiceClient.chat({
    message,
    userTimezone,
    agentType: AgentType.SCHEDULE,
    isTempConversation: true,
  });

  for await (const chunk of response) {
    // Parse the event from eventType and eventData fields
    if (chunk.eventType) {
      const parsed = parseEvent(chunk.eventData || "", chunk.eventType);
      if (parsed) {
        onEvent?.(parsed);
        yield parsed;
      }
    }

    // Yield the raw chunk for compatibility
    yield {
      type: chunk.eventType || "data",
      data: chunk.eventData || "",
      content: chunk.content,
      done: chunk.done,
    };
  }
}
