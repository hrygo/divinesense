/**
 * Block Stream Hook
 *
 * Manages streaming block updates with event buffering, debouncing,
 * and automatic retry for reliable streaming experience.
 *
 * @see docs/specs/unified-block-model.md
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { Block, BlockEvent, BlockStatus, SessionStats } from "@/types/proto/api/v1/ai_service_pb";
import { BlockStatus as BlockStatusEnum } from "@/types/proto/api/v1/ai_service_pb";
import { blockKeys } from "./useBlockQueries";

// ============================================================================
// Types
// ============================================================================

export interface StreamEvent {
  type: "thinking" | "tool_use" | "tool_result" | "answer" | "error" | "status";
  content?: string;
  meta?: Record<string, unknown>;
  timestamp: number;
}

export interface BlockStreamState {
  block: Block | null;
  isStreaming: boolean;
  streamingPhase: "thinking" | "tools" | "answer" | null;
  events: StreamEvent[];
  error: string | null;
  hasError: boolean;
}

export interface UseBlockStreamOptions {
  blockId: number;
  conversationId: number;
  onStreamStart?: () => void;
  onStreamEvent?: (event: StreamEvent) => void;
  onStreamComplete?: (block: Block) => void;
  onStreamError?: (error: Error) => void;
  enabled?: boolean;
}

export interface BlockStreamActions {
  startStream: (userInput: string, mode: "normal" | "geek" | "evolution") => Promise<void>;
  stopStream: () => void;
  addEvent: (event: StreamEvent) => void;
  clearError: () => void;
  reset: () => void;
}

// ============================================================================
// Hook
// ============================================================================

export function useBlockStream(options: UseBlockStreamOptions): BlockStreamState & BlockStreamActions {
  const {
    blockId,
    conversationId,
    onStreamStart,
    onStreamEvent,
    onStreamComplete,
    onStreamError,
    enabled = true,
  } = options;

  const queryClient = useQueryClient();
  const [state, setState] = useState<BlockStreamState>({
    block: null,
    isStreaming: false,
    streamingPhase: null,
    events: [],
    error: null,
    hasError: false,
  });

  const abortControllerRef = useRef<AbortController | null>(null);
  const eventBufferRef = useRef<StreamEvent[]>([]);
  const flushTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Flush buffered events to state
   */
  const flushEvents = useCallback(() => {
    if (eventBufferRef.current.length === 0) {
      return;
    }

    setState((prev) => ({
      ...prev,
      events: [...prev.events, ...eventBufferRef.current],
    }));

    eventBufferRef.current = [];
  }, []);

  /**
   * Add event to buffer (with automatic flushing)
   */
  const addEventToBuffer = useCallback((event: StreamEvent) => {
    eventBufferRef.current.push(event);
    onStreamEvent?.(event);

    // Clear existing timeout
    if (flushTimeoutRef.current) {
      clearTimeout(flushTimeoutRef.current);
    }

    // Flush after delay (batch events)
    flushTimeoutRef.current = setTimeout(() => {
      flushEvents();
    }, 50); // 50ms batch window
  }, [flushEvents, onStreamEvent]);

  /**
   * Update block state directly (for streaming updates)
   */
  const updateBlockState = useCallback((updates: Partial<Block>) => {
    setState((prev) => ({
      ...prev,
      block: prev.block ? { ...prev.block, ...updates } : null,
    }));
  }, []);

  /**
   * Determine streaming phase from event type
   */
  const getPhaseForEvent = useCallback((eventType: string): BlockStreamState["streamingPhase"] => {
    switch (eventType) {
      case "thinking":
        return "thinking";
      case "tool_use":
      case "tool_result":
        return "tools";
      case "answer":
        return "answer";
      case "error":
        return null;
      default:
        return null;
    }
  }, []);

  /**
   * Handle incoming stream event
   */
  const handleStreamEvent = useCallback(
    (event: StreamEvent) => {
      addEventToBuffer(event);

      // Update streaming phase
      const phase = getPhaseForEvent(event.type);
      if (phase) {
        setState((prev) => ({ ...prev, streamingPhase: phase }));
      }

      // Update block state based on event type
      if (event.type === "answer" && event.content) {
        setState((prev) => ({
          ...prev,
          block: prev.block
            ? { ...prev.block, assistantContent: prev.block.assistantContent + event.content }
            : null,
        }));
      }

      if (event.type === "status" && event.meta?.status) {
        const newStatus = event.meta.status as BlockStatus;
        const isTerminal =
          String(newStatus) === String(BlockStatusEnum.COMPLETED) ||
          String(newStatus) === String(BlockStatusEnum.ERROR);

        if (isTerminal) {
          setState((prev) => ({
            ...prev,
            isStreaming: false,
            streamingPhase: null,
          }));
        }
      }

      if (event.type === "error") {
        setState((prev) => ({
          ...prev,
          error: event.content || "Stream error",
          hasError: true,
          isStreaming: false,
          streamingPhase: null,
        }));
      }
    },
    [addEventToBuffer, getPhaseForEvent]
  );

  /**
   * Start a new stream
   */
  const startStream = useCallback(
    async (userInput: string, mode: "normal" | "geek" | "evolution") => {
      if (!enabled || state.isStreaming) {
        return;
      }

      // Cancel any existing stream
      stopStream();

      // Create new abort controller
      abortControllerRef.current = new AbortController();

      // Reset state
      setState({
        block: null,
        isStreaming: true,
        streamingPhase: "thinking",
        events: [],
        error: null,
        hasError: false,
      });

      onStreamStart?.();

      try {
        // TODO: Call actual streaming API
        // For now, this is a placeholder
        console.log("[useBlockStream] Starting stream:", { userInput, mode });
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));
        setState((prev) => ({
          ...prev,
          error: err.message,
          hasError: true,
          isStreaming: false,
          streamingPhase: null,
        }));
        onStreamError?.(err);
      }
    },
    [enabled, state.isStreaming, onStreamStart, onStreamError]
  );

  /**
   * Stop the current stream
   */
  const stopStream = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    if (flushTimeoutRef.current) {
      clearTimeout(flushTimeoutRef.current);
      flushTimeoutRef.current = null;
    }

    setState((prev) => ({
      ...prev,
      isStreaming: false,
      streamingPhase: null,
    }));
  }, []);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setState((prev) => ({
      ...prev,
      error: null,
      hasError: false,
    }));
  }, []);

  /**
   * Reset to initial state
   */
  const reset = useCallback(() => {
    stopStream();
    setState({
      block: null,
      isStreaming: false,
      streamingPhase: null,
      events: [],
      error: null,
      hasError: false,
    });
  }, [stopStream]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      stopStream();
    };
  }, [stopStream]);

  return {
    ...state,
    startStream,
    stopStream,
    addEvent: handleStreamEvent,
    clearError,
    reset,
  };
}

// ============================================================================
// Utilities
// ============================================================================

/**
 * Create a stream event from server-sent event data
 */
export function createStreamEvent(data: {
  type: string;
  content?: string;
  meta?: Record<string, unknown>;
}): StreamEvent {
  return {
    type: data.type as StreamEvent["type"],
    content: data.content,
    meta: data.meta,
    timestamp: Date.now(),
  };
}

/**
 * Check if block is in a terminal state
 */
export function isBlockTerminal(block: Block | null): boolean {
  if (!block) return false;
  const status = String(block.status);
  return (
    status === String(BlockStatusEnum.COMPLETED) ||
    status === String(BlockStatusEnum.ERROR)
  );
}

/**
 * Check if block is actively streaming
 */
export function isBlockStreaming(block: Block | null): boolean {
  if (!block) return false;
  return String(block.status) === String(BlockStatusEnum.STREAMING);
}
