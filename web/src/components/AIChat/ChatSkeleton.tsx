import { cn } from "@/lib/utils";

interface ChatSkeletonProps {
  className?: string;
}

/**
 * Chat message skeleton for AI chat loading states.
 * Provides visual feedback during message loading and streaming.
 */
export function ChatSkeleton({ className }: ChatSkeletonProps) {
  return (
    <div className={cn("flex gap-3 md:gap-4", className)}>
      {/* Avatar skeleton */}
      <div className="w-9 h-9 md:w-10 md:h-10 rounded-full bg-muted shrink-0 animate-pulse" />

      {/* Message content skeleton */}
      <div className="flex-1 min-w-[120px] max-w-[85%] md:max-w-[80%]">
        <div className="px-4 py-3 rounded-2xl bg-muted/30 border border-border">
          {/* Text lines */}
          <div className="space-y-2">
            <div className="h-4 bg-muted/60 rounded w-3/4 animate-pulse" />
            <div className="h-4 bg-muted/60 rounded w-full animate-pulse" />
            <div className="h-4 bg-muted/60 rounded w-5/6 animate-pulse" />
          </div>
        </div>
      </div>
    </div>
  );
}

interface ChatSkeletonListProps {
  count?: number;
  className?: string;
}

/**
 * Multiple chat skeleton items for initial loading states.
 */
export function ChatSkeletonList({ count = 3, className }: ChatSkeletonListProps) {
  return (
    <div className={cn("max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto space-y-4", className)}>
      {Array.from({ length: count }).map((_, index) => (
        <ChatSkeleton key={index} className={index > 0 ? "opacity-50" : ""} />
      ))}
    </div>
  );
}

export default ChatSkeleton;
