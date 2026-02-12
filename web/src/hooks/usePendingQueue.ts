import { useEffect } from "react";
import { useAIChat } from "@/contexts/AIChatContext";
import { Block, isActiveStatus } from "@/types/block";

/**
 * Hook to manage pending queue flushing
 *
 * Watches the blocks list and the pending queue.
 * When the pending queue has items and there are no active blocks (streaming/pending),
 * it automatically flushes the queue by combining messages and triggering onSend.
 */
export function usePendingQueue(blocks: Block[], onSend: (message: string) => void) {
  const { state, clearPendingQueue } = useAIChat();
  const { pendingQueue } = state;

  useEffect(() => {
    // If queue is empty, nothing to do
    if (pendingQueue.messages.length === 0) return;

    // Check if there are any active blocks
    // We check the latest block usually, but safest to check if any block is active
    // However, usually only the last one is active.
    const hasActiveBlock = blocks.some((b) => isActiveStatus(b.status));

    if (!hasActiveBlock) {
      // No active blocks, safe to flush queue

      // 1. Combine messages with newlines
      const combinedContent = pendingQueue.messages.map((m) => m.content).join("\n");

      // 2. Clear queue immediately to prevent double submission
      clearPendingQueue();

      // 3. Trigger send
      // Use setImmediate/setTimeout to break the render cycle just in case
      setTimeout(() => {
        onSend(combinedContent);
      }, 0);
    }
  }, [blocks, pendingQueue.messages, clearPendingQueue, onSend]);
}
