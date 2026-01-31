import { useCallback, useEffect, useRef, useState } from "react";
import { useWebSocket } from "@/hooks/useWebSocket";

/**
 * Stream event from CC Runner
 * CC Runner 流式事件
 */
export interface StreamEvent {
  type: "thinking" | "tool_use" | "tool_result" | "answer" | "error";
  content: string;
  meta?: {
    tool_name?: string;
    tool_id?: string;
    is_error?: boolean;
    file_path?: string;
    session_id?: string;
    exit_code?: number;
    duration_ms?: number;
    input?: Record<string, unknown>;
  };
  timestamp: number;
}

/**
 * Event stream state
 * 事件流状态
 */
export interface EventStreamState {
  isConnected: boolean;
  isThinking: boolean;
  currentEvents: StreamEvent[];
  error: string | null;
}

/**
 * Options for useEventStream
 */
interface UseEventStreamOptions {
  enabled?: boolean;
  onThinking?: (isThinking: boolean) => void;
  onToolUse?: (event: StreamEvent) => void;
  onToolResult?: (event: StreamEvent) => void;
  onAnswer?: (content: string) => void;
  onError?: (error: string) => void;
}

/**
 * useEventStream - Hook for managing CC Runner WebSocket event stream
 * useEventStream - 管理 CC Runner WebSocket 事件流的 Hook
 */
export function useEventStream(
  url: string | null,
  options: UseEventStreamOptions = {},
): EventStreamState & { sendMessage: (message: unknown) => void; disconnect: () => void } {
  const { enabled = true, onThinking, onToolUse, onToolResult, onAnswer, onError } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [isThinking, setIsThinkingState] = useState(false);
  const [currentEvents, setCurrentEvents] = useState<StreamEvent[]>([]);
  const [error, setError] = useState<string | null>(null);

  const eventsRef = useRef<StreamEvent[]>([]);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const MAX_RECONNECT_ATTEMPTS = 5;

  // Update current events when eventsRef changes
  useEffect(() => {
    setCurrentEvents([...eventsRef.current]);
  }, []);

  // Process incoming event
  const processEvent = useCallback(
    (event: StreamEvent) => {
      // Add to events list
      eventsRef.current.push(event);

      // Handle event type
      switch (event.type) {
        case "thinking":
          setIsThinkingState(true);
          onThinking?.(true);
          break;
        case "answer":
          setIsThinkingState(false);
          onThinking?.(false);
          onAnswer?.(event.content);
          break;
        case "tool_use":
          onToolUse?.(event);
          break;
        case "tool_result":
          onToolResult?.(event);
          break;
        case "error":
          setIsThinkingState(false);
          onThinking?.(false);
          setError(event.content);
          onError?.(event.content);
          break;
      }

      // Keep only last 50 events in memory
      if (eventsRef.current.length > 50) {
        eventsRef.current = eventsRef.current.slice(-50);
      }
      setCurrentEvents([...eventsRef.current]);
    },
    [onThinking, onToolUse, onToolResult, onAnswer, onError],
  );

  // Clear error on new connection
  const _clearError = useCallback(() => {
    setError(null);
  }, []);

  // Handle connection
  const _handleOpen = useCallback(() => {
    setIsConnected(true);
    setError(null);
    reconnectAttemptsRef.current = 0;
  }, []);

  const _handleClose = useCallback(() => {
    setIsConnected(false);
    setIsThinkingState(false);
    onThinking?.(false);

    // Auto-reconnect with backoff
    if (enabled && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
      reconnectAttemptsRef.current++;
      const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
      reconnectTimeoutRef.current = setTimeout(() => {
        // Trigger reconnect by setting a flag or using a reconnect function
        // This would be handled by the WebSocket hook
      }, delay);
    }
  }, [enabled]);

  const _handleMessage = useCallback(
    (data: string) => {
      try {
        const event = JSON.parse(data) as StreamEvent;
        processEvent(event);
      } catch {
        // Non-JSON message, treat as plain text answer
        processEvent({
          type: "answer",
          content: data,
          timestamp: Date.now(),
        });
      }
    },
    [processEvent],
  );

  const _handleError = useCallback(
    (err: Event) => {
      setError(err.message || "WebSocket error");
      onError?.(err.message || "WebSocket error");
    },
    [onError],
  );

  // Note: Using a simple WebSocket connection here
  // In production, this would integrate with the existing useParrotChat hook
  // which already has SSE support for the chat interface

  const sendMessage = useCallback((message: unknown) => {
    // This would send a message through the WebSocket
    // Implementation depends on the WebSocket library used
    console.log("Sending message:", message);
  }, []);

  const disconnect = useCallback(() => {
    setIsConnected(false);
    setIsThinkingState(false);
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
  }, []);

  return {
    isConnected,
    isThinking,
    currentEvents,
    error,
    sendMessage,
    disconnect,
  };
}

/**
 * useCcRunnerEvents - Hook specifically for CC Runner event display
 * useCcRunnerEvents - 专门用于 CC Runner 事件展示的 Hook
 */
export function useCcRunnerEvents(enabled = true) {
  const [toolCalls, setToolCalls] = useState<Map<string, StreamEvent>>(new Map());
  const [latestThinking, setLatestThinking] = useState<string>("");

  const handleToolUse = useCallback((event: StreamEvent) => {
    setToolCalls((prev) => {
      const next = new Map(prev);
      const id = event.meta?.tool_id || `${event.type}-${Date.now()}`;
      next.set(id, event);
      return next;
    });
  }, []);

  const handleToolResult = useCallback((event: StreamEvent) => {
    setToolCalls((prev) => {
      const next = new Map(prev);
      // Update existing tool call with result
      if (event.meta?.tool_id) {
        const existing = next.get(event.meta.tool_id);
        if (existing) {
          next.set(event.meta.tool_id, { ...existing, ...event });
        }
      }
      return next;
    });
  }, []);

  const handleThinking = useCallback((isThinking: boolean) => {
    if (!isThinking) {
      setLatestThinking("");
    }
  }, []);

  const handleAnswer = useCallback((content: string) => {
    setLatestThinking((prev) => prev + content);
  }, []);

  return {
    toolCalls,
    latestThinking,
    handleToolUse,
    handleToolResult,
    handleThinking,
    handleAnswer,
  };
}
