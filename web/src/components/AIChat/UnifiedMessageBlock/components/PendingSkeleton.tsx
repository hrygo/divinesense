/**
 * PendingSkeleton Component
 *
 * Skeleton placeholder for blocks in pending/processing state.
 * Replaces "Processing..." text with a more visual loading indicator.
 *
 * Phase 3: Interaction Feedback Enhancement
 */

import { memo } from "react";
import { cn } from "@/lib/utils";

export interface PendingSkeletonProps {
  /** Optional custom message to display */
  message?: string;
  /** Optional additional CSS classes */
  className?: string;
}

/**
 * PendingSkeleton component
 *
 * Renders a skeleton loading indicator for blocks
 * that are waiting for AI response generation.
 */
export const PendingSkeleton = memo(function PendingSkeleton({ message, className }: PendingSkeletonProps) {
  return (
    <div className={cn("flex items-center gap-3 py-2", className)}>
      {/* Animated spinner */}
      <div className="relative w-5 h-5">
        <div className="absolute inset-0 rounded-full border-2 border-border/30" />
        <div className="absolute inset-0 rounded-full border-2 border-primary border-t-transparent animate-spin" />
      </div>

      {/* Skeleton text lines */}
      <div className="flex-1 space-y-2">
        <div className="h-4 bg-muted/50 rounded animate-pulse w-3/4" />
        <div className="h-4 bg-muted/30 rounded animate-pulse w-1/2" />
      </div>

      {/* Optional message */}
      {message && <span className="text-xs text-muted-foreground">{message}</span>}
    </div>
  );
});
