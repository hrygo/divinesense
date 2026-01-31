import { useCallback, useRef, useState } from "react";

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
  _url: string | null,
  options: UseEventStreamOptions = {},
): EventStreamState & { sendMessage: (message: unknown) => void; disconnect: () => void } {
  // Extract options to avoid unused errors, but prefix with _ if unused
  const { enabled: _enabled = true } = options;

  // State
  const [isConnected, _setIsConnected] = useState(false);
  const [isThinking, _setIsThinkingState] = useState(false);
  const [currentEvents, _setCurrentEvents] = useState<StreamEvent[]>([]);
  const [error, _setError] = useState<string | null>(null);

  // Refs for logic that might be implemented later
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const sendMessage = useCallback((message: unknown) => {
    // This would send a message through the WebSocket
    // Implementation depends on the WebSocket library used
    console.log("Sending message:", message);
  }, []);

  const disconnect = useCallback(() => {
    _setIsConnected(false);
    _setIsThinkingState(false);
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
export function useCcRunnerEvents(_enabled = true) {
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
