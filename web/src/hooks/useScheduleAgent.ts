import { useMutation, useQueryClient } from "@tanstack/react-query";
import { aiServiceClient } from "@/connect";
import { AgentType } from "@/types/proto/api/v1/ai_service_pb";

/**
 * Hook to chat with Schedule Agent (non-streaming)
 * Uses AIService.Chat with AGENT_TYPE_SCHEDULE
 */
export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

interface ScheduleAgentChatRequest {
  message: string;
  userTimezone?: string;
  history?: ChatMessage[];
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
      // Build context-aware message
      const parts: string[] = [];

      // Conversation History
      if (request.history && request.history.length > 0) {
        parts.push("[Conversation History]");
        request.history.forEach((msg) => {
          parts.push(`${msg.role === "user" ? "User" : "Assistant"}: ${msg.content}`);
        });
      }

      // Current Message
      parts.push(`User: ${request.message}`);

      const fullMessage = parts.join("\n\n");

      // Use AIService.Chat with SCHEDULE agent type
      let response = "";
      for await (const chunk of aiServiceClient.chat({
        message: fullMessage,
        userTimezone: request.userTimezone || "Asia/Shanghai",
        agentType: AgentType.SCHEDULE,
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
 */
export function parseEvent(eventJSON: string): ParsedEvent | null {
  try {
    const event = JSON.parse(eventJSON);
    return {
      type: event.type,
      data: event.data,
    };
  } catch (e) {
    console.error("Failed to parse event:", eventJSON, e);
    return null;
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
  });

  for await (const chunk of response) {
    // Parse the event from eventType and eventData fields
    if (chunk.eventType) {
      const parsed = parseEvent(chunk.eventData || "{}");
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
