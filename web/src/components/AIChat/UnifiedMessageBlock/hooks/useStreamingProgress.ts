/**
 * useStreamingProgress Hook
 *
 * Manages streaming progress state for blocks.
 * Calculates progress based on content length.
 *
 * Phase 3: Interaction Feedback Enhancement
 */

import { useEffect, useState } from "react";

export interface UseStreamingProgressOptions {
  /** Whether streaming is active */
  isStreaming: boolean;
  /** Current content being streamed */
  content: string;
  /** Expected max content length for progress calculation (optional) */
  expectedMaxLength?: number;
}

export interface UseStreamingProgressReturn {
  /** Current progress percentage (0-100) */
  progress: number;
  /** Whether progress is being shown */
  isShowingProgress: boolean;
}

/**
 * Hook for managing streaming progress
 *
 * Calculates progress based on:
 * - Content length vs expected max length (if provided)
 * - Animation state during active streaming
 */
export function useStreamingProgress({
  isStreaming,
  content,
  expectedMaxLength = 500, // Default expected length
}: UseStreamingProgressOptions): UseStreamingProgressReturn {
  const [progress, setProgress] = useState(0);

  // Calculate progress based on content length
  useEffect(() => {
    if (!isStreaming) {
      setProgress(0);
      return;
    }

    const contentLength = content.length;
    const calculatedProgress = Math.min(
      95, // Cap at 95% until complete
      Math.round((contentLength / expectedMaxLength) * 100),
    );

    setProgress(calculatedProgress);
  }, [isStreaming, content, expectedMaxLength]);

  // Complete progress when streaming ends
  useEffect(() => {
    if (!isStreaming && progress > 0) {
      // Briefly show 100% then hide
      const timer = setTimeout(() => setProgress(0), 300);
      return () => clearTimeout(timer);
    }
  }, [isStreaming, progress]);

  const isShowingProgress = isStreaming || progress > 0;

  return {
    progress,
    isShowingProgress,
  };
}
