/**
 * StreamingProgressBar Component
 *
 * Progress bar shown at the bottom of blocks during streaming.
 * Provides visual feedback that content is being generated.
 *
 * Phase 3: Interaction Feedback Enhancement
 */

import { memo } from "react";
import { cn } from "@/lib/utils";

export interface StreamingProgressBarProps {
  /** Progress percentage (0-100) */
  progress: number;
  /** Whether the stream is active */
  isActive: boolean;
  /** Optional additional CSS classes */
  className?: string;
}

/**
 * StreamingProgressBar component
 *
 * Renders a thin progress bar at the bottom of a block
 * during active streaming response generation.
 */
export const StreamingProgressBar = memo(function StreamingProgressBar({ progress, isActive, className }: StreamingProgressBarProps) {
  if (!isActive) return null;

  return (
    <div
      className={cn("absolute bottom-0 left-0 right-0 h-1 bg-primary/20 overflow-hidden", className)}
      role="progressbar"
      aria-valuenow={progress}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label="AI response streaming progress"
    >
      <div
        className="h-full bg-primary transition-all duration-300 ease-out"
        style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
      />
    </div>
  );
});
