import { useMutation, useQueryClient } from "@tanstack/react-query";
import { scheduleAgentServiceClient } from "@/connect";

/**
 * Hook to chat with Schedule Agent (non-streaming)
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

      const response = await scheduleAgentServiceClient.chat({
        message: fullMessage,
        userTimezone: request.userTimezone || "Asia/Shanghai",
      });
      return response;
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
 * Returns an async generator that yields stream events
 */
export async function* scheduleAgentChatStream(
  message: string,
  userTimezone = "Asia/Shanghai",
  onEvent?: (event: { type: string; data: string }) => void,
): AsyncGenerator<{ type: string; data: string; content?: string; done?: boolean }, void> {
  const response = await scheduleAgentServiceClient.chatStream({
    message,
    userTimezone,
  });

  for await (const chunk of response) {
    // Parse the event JSON
    if (chunk.event) {
      const parsed = parseEvent(chunk.event);
      if (parsed) {
        onEvent?.(parsed);
        yield parsed;
      }
    }

    // Yield the raw chunk for compatibility
    yield {
      type: chunk.event ? "raw" : "data",
      data: chunk.event || "",
      content: chunk.content,
      done: chunk.done,
    };
  }
}
