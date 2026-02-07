/**
 * VirtualBlockList Component
 *
 * Virtual scrolling implementation for long conversations.
 * Uses @tanstack/react-virtual for efficient rendering.
 *
 * Phase 5: Performance Optimization
 */

import type { VirtualItem } from "@tanstack/react-virtual";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useRef } from "react";
import { cn } from "@/lib/utils";
import { VIRTUAL_SCROLL_OVERSCAN } from "../constants";

export interface VirtualBlockListOptions {
  /** Total number of blocks */
  count: number;
  /** Estimated height of each block in pixels */
  estimateSize?: number;
  /** Container element ref */
  parentRef: React.RefObject<HTMLElement>;
  /** Overscan for virtual rendering */
  overscan?: number;
}

export interface VirtualBlockListReturn {
  /** Virtual items to render */
  virtualItems: VirtualItem[];
  /** Total size of the virtualized list */
  totalSize: number;
  /** Index of the first virtual item */
  startIndex: number;
}

/**
 * Hook for virtual scrolling block list
 *
 * Only renders blocks visible in viewport plus overscan buffer.
 * Significantly improves performance for conversations with 50+ blocks.
 */
export function useVirtualBlockList({
  count,
  estimateSize = 200, // Default estimated block height
  parentRef,
  overscan = VIRTUAL_SCROLL_OVERSCAN / 50, // Convert pixels to items (approx)
}: VirtualBlockListOptions): VirtualBlockListReturn {
  const virtualizer = useVirtualizer({
    count,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimateSize,
    overscan,
  });

  const virtualItems = virtualizer.getVirtualItems();
  const totalSize = virtualizer.getTotalSize();
  const startIndex = virtualItems.length > 0 ? virtualItems[0].index : 0;

  return {
    virtualItems,
    totalSize,
    startIndex,
  };
}

/**
 * Props for VirtualBlockList render component
 */
export interface VirtualBlockListProps<T> {
  /** Items to render */
  items: T[];
  /** Unique key selector for items */
  getKey: (item: T, index: number) => string;
  /** Render function for each item */
  renderItem: (item: T, index: number, virtualItem: VirtualItem) => React.ReactNode;
  /** Estimated height of each item */
  estimateSize?: number;
  /** Container className */
  className?: string;
  /** Container style */
  style?: React.CSSProperties;
}

/**
 * VirtualBlockList Component
 *
 * Renders a virtualized list of blocks.
 * Only visible items are rendered, improving performance for long lists.
 */
export function VirtualBlockList<T>({ items, getKey, renderItem, estimateSize = 200, className, style }: VirtualBlockListProps<T>) {
  const parentRef = useRef<HTMLDivElement>(null);

  const { virtualItems, totalSize } = useVirtualBlockList({
    count: items.length,
    estimateSize,
    parentRef,
  });

  return (
    <div
      ref={parentRef}
      className={cn("overflow-auto", className)}
      style={{
        height: "100%",
        ...style,
      }}
    >
      <div
        style={{
          height: `${totalSize}px`,
          width: "100%",
          position: "relative",
        }}
      >
        {virtualItems.map((virtualItem: VirtualItem) => {
          const item = items[virtualItem.index];
          const key = getKey(item, virtualItem.index);

          return (
            <div
              key={key}
              data-index={virtualItem.index}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                transform: `translateY(${virtualItem.start}px)`,
              }}
            >
              {renderItem(item, virtualItem.index, virtualItem)}
            </div>
          );
        })}
      </div>
    </div>
  );
}
