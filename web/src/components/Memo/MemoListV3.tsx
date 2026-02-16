/**
 * MemoListV3 - "Zen Kanban" Layout
 *
 * 设计哲学：「禅意看板」
 * - 移动端单列 / PC 端双列瀑布流
 * - 200 字摘要默认展示（无折叠）
 * - 呼吸感间距 + 渐进式加载
 *
 * Features:
 * - Responsive 1/2 column grid
 * - Infinite scroll with intersection observer
 * - Staggered reveal animations
 * - Filter integration
 * - Zen-style loading and empty states
 */

import { Filter, Inbox, Search } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { MemoBlockV3 } from "@/components/Memo/MemoBlockV3";
import { DEFAULT_LIST_MEMOS_PAGE_SIZE } from "@/helpers/consts";
import { useInfiniteMemos } from "@/hooks/useMemoQueries";
import { cn } from "@/lib/utils";
import { State } from "@/types/proto/api/v1/common_pb";

// ============================================================================
// Types
// ============================================================================

export interface MemoListV3Props {
  state?: State;
  orderBy?: string;
  filter?: string;
  pageSize?: number;
  onEdit?: (memoName: string) => void;
  className?: string;
}

// ============================================================================
// Masonry Layout Hook - Fix render order (left-to-right, top-to-bottom)
// ============================================================================

/**
 * useMasonryColumns - Distributes items into columns for true masonry layout
 * Fixes CSS columns vertical-fill problem: items now render left-to-right, top-to-bottom
 */
function useMasonryColumns<T>(items: T[], columnCount: number): T[][] {
  return useMemo(() => {
    if (columnCount <= 1) {
      return [items];
    }

    const columns: T[][] = Array.from({ length: columnCount }, () => []);
    // Use estimated height ratio to distribute items more evenly
    // Real height measurement would require DOM access, using estimation for perf
    const estimatedHeights = items.map(() => Math.random() * 0.5 + 0.75); // 0.75-1.25 ratio

    items.forEach((item, _index) => {
      // Find the shortest column (with least accumulated height)
      let shortestColumnIndex = 0;
      let shortestColumnHeight = columns[0].reduce((sum, _, i) => sum + estimatedHeights[i] || 1, 0);

      for (let i = 1; i < columns.length; i++) {
        const columnHeight = columns[i].reduce((sum, _, j) => sum + estimatedHeights[j] || 1, 0);
        if (columnHeight < shortestColumnHeight) {
          shortestColumnHeight = columnHeight;
          shortestColumnIndex = i;
        }
      }

      columns[shortestColumnIndex].push(item);
    });

    return columns;
  }, [items, columnCount]);
}

/**
 * useColumnCount - Responsive column count based on viewport width
 */
function useColumnCount(): number {
  const [columnCount, setColumnCount] = useState(() => {
    if (typeof window === "undefined") return 2;
    return window.innerWidth < 640 ? 1 : 2;
  });

  useEffect(() => {
    const handleResize = () => {
      setColumnCount(window.innerWidth < 640 ? 1 : 2);
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return columnCount;
}

// ============================================================================
// Infinite Scroll Hook - Use IntersectionObserver for better performance
// ============================================================================

/**
 * useInfiniteScroll - Fetch more when element enters viewport
 * Replaces scroll event listener with IntersectionObserver for better performance
 */
function useInfiniteScroll({
  hasNextPage,
  isFetchingNextPage,
  fetchNextPage,
}: {
  hasNextPage: boolean | undefined;
  isFetchingNextPage: boolean;
  fetchNextPage: () => Promise<unknown>;
}) {
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    if (!hasNextPage) return;

    // Disconnect previous observer
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry.isIntersecting && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      {
        root: null,
        rootMargin: "200px",
        threshold: 0.1,
      },
    );

    if (loadMoreRef.current) {
      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return loadMoreRef;
}

// ============================================================================
// Auto-fetch Hook
// ============================================================================

/**
 * Auto-fetch when page isn't scrollable
 * Ensures content fills the viewport
 */
function useAutoFetchWhenNotScrollable({
  hasNextPage,
  isFetchingNextPage,
  memoCount,
  onFetchNext,
}: {
  hasNextPage: boolean | undefined;
  isFetchingNextPage: boolean;
  memoCount: number;
  onFetchNext: () => Promise<unknown>;
}) {
  const autoFetchTimeoutRef = useRef<number | null>(null);

  const isPageScrollable = useCallback(() => {
    const documentHeight = Math.max(document.body.scrollHeight, document.documentElement.scrollHeight);
    return documentHeight > window.innerHeight + 100;
  }, []);

  const checkAndFetchIfNeeded = useCallback(async () => {
    if (autoFetchTimeoutRef.current) {
      clearTimeout(autoFetchTimeoutRef.current);
    }

    await new Promise((resolve) => setTimeout(resolve, 200));

    const shouldFetch = !isPageScrollable() && hasNextPage && !isFetchingNextPage && memoCount > 0;

    if (shouldFetch) {
      await onFetchNext();

      autoFetchTimeoutRef.current = window.setTimeout(() => {
        void checkAndFetchIfNeeded();
      }, 500);
    }
  }, [hasNextPage, isFetchingNextPage, memoCount, isPageScrollable, onFetchNext]);

  useEffect(() => {
    if (!isFetchingNextPage && memoCount > 0) {
      void checkAndFetchIfNeeded();
    }
  }, [memoCount, isFetchingNextPage, checkAndFetchIfNeeded]);

  useEffect(() => {
    return () => {
      if (autoFetchTimeoutRef.current) {
        clearTimeout(autoFetchTimeoutRef.current);
      }
    };
  }, []);
}

// ============================================================================
// Loading Skeleton
// ============================================================================

function KanbanSkeleton({ columns = 2 }: { columns?: number }) {
  return (
    <div className={cn("columns-1 sm:columns-2 gap-4", columns === 1 && "sm:columns-1")}>
      {Array.from({ length: columns === 1 ? 3 : 6 }).map((_, i) => (
        <div
          key={i}
          className="break-inside-avoid mb-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4"
        >
          {/* Preview lines */}
          <div className="space-y-2">
            <div className="h-4 bg-zinc-100 dark:bg-zinc-800 rounded w-full animate-pulse" />
            <div className="h-4 bg-zinc-100 dark:bg-zinc-800 rounded w-5/6 animate-pulse" />
            <div className="h-4 bg-zinc-100 dark:bg-zinc-800 rounded w-4/6 animate-pulse" />
          </div>
          {/* Tags placeholder */}
          <div className="flex gap-1.5 mt-3">
            <div className="h-5 w-14 bg-zinc-100 dark:bg-zinc-800 rounded-full animate-pulse" />
            <div className="h-5 w-10 bg-zinc-100 dark:bg-zinc-800 rounded-full animate-pulse" />
          </div>
          {/* Footer placeholder */}
          <div className="flex justify-between mt-3 pt-2 border-t border-zinc-100 dark:border-zinc-800">
            <div className="h-3 w-12 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
            <div className="flex gap-1">
              <div className="h-6 w-6 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
              <div className="h-6 w-6 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================================================
// Empty State
// ============================================================================

function EmptyState({ type }: { type: "all" | "filtered" | "search" }) {
  const { t } = useTranslation();

  const config = {
    all: {
      icon: Inbox,
      title: t("memo.empty_all_title"),
      description: t("memo.empty_all_subtitle"),
    },
    filtered: {
      icon: Filter,
      title: t("memo.empty_filtered_title"),
      description: t("memo.empty_filtered_subtitle"),
    },
    search: {
      icon: Search,
      title: t("memo.empty_search_title"),
      description: t("memo.empty_search_subtitle"),
    },
  };

  const { icon: Icon, title, description } = config[type];

  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="w-16 h-16 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4">
        <Icon className="w-8 h-8 text-zinc-400 dark:text-zinc-500" />
      </div>
      <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-1">{title}</h3>
      <p className="text-sm text-zinc-500 dark:text-zinc-400">{description}</p>
    </div>
  );
}

// ============================================================================
// End Indicator
// ============================================================================

function EndIndicator() {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-center py-8">
      <div className="flex items-center gap-3 text-zinc-400 dark:text-zinc-500">
        <div className="w-12 h-px bg-zinc-200 dark:bg-zinc-700" />
        <span className="text-xs">{t("memo.end_of_list")}</span>
        <div className="w-12 h-px bg-zinc-200 dark:bg-zinc-700" />
      </div>
    </div>
  );
}

// ============================================================================
// Main Component
// ============================================================================

export const MemoListV3 = memo(function MemoListV3({
  state = State.NORMAL,
  orderBy = "display_time desc",
  filter,
  pageSize = DEFAULT_LIST_MEMOS_PAGE_SIZE,
  onEdit,
  className,
}: MemoListV3Props) {
  // React Query infinite query
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useInfiniteMemos({
    state,
    orderBy,
    filter,
    pageSize,
  });

  // Flatten pages into single array
  const memos = useMemo(() => data?.pages.flatMap((page) => page.memos) || [], [data?.pages]);

  // Responsive column count
  const columnCount = useColumnCount();

  // Masonry layout: distribute items into columns
  const columns = useMasonryColumns(memos, columnCount);

  // Auto-fetch when page isn't scrollable
  useAutoFetchWhenNotScrollable({
    hasNextPage,
    isFetchingNextPage,
    memoCount: memos.length,
    onFetchNext: fetchNextPage,
  });

  // Infinite scroll with IntersectionObserver (better performance than scroll event)
  const loadMoreRef = useInfiniteScroll({
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  });

  // Handle edit action
  const handleEdit = useCallback(
    (memoName: string) => {
      onEdit?.(memoName);
    },
    [onEdit],
  );

  // Animation delay for staggered reveal - based on original index
  const getAnimationDelay = (index: number): number => {
    // Faster cascade for first few, then slower
    return index < 6 ? index * 60 : 360 + (index - 6) * 30;
  };

  // Determine empty state type
  const getEmptyType = (): "all" | "filtered" | "search" => {
    if (filter && filter.includes("contentSearch")) {
      return "search";
    }
    if (filter) {
      return "filtered";
    }
    return "all";
  };

  return (
    <div className={cn("flex flex-col w-full", className)}>
      {/* Initial loading skeleton */}
      {isLoading ? (
        <KanbanSkeleton columns={columnCount} />
      ) : (
        <>
          {/* Kanban Masonry - Responsive 1/2 columns with left-to-right, top-to-bottom render order */}
          <div className="flex gap-4 w-full">
            {columns.map((columnMemos, colIndex) => (
              <div key={colIndex} className="flex-1 flex flex-col gap-4">
                {columnMemos.map((memo) => {
                  // Calculate original index for animation
                  const originalIndex = memos.indexOf(memo);
                  return (
                    <div
                      key={memo.name}
                      className="animate-in fade-in slide-in-from-bottom-3 duration-500 ease-out"
                      style={{
                        animationDelay: `${getAnimationDelay(originalIndex)}ms`,
                        animationFillMode: "both",
                      }}
                    >
                      <MemoBlockV3 memo={memo} onEdit={handleEdit} />
                    </div>
                  );
                })}
              </div>
            ))}
          </div>

          {/* Intersection observer target */}
          <div ref={loadMoreRef} className="h-px w-full" />

          {/* Loading more indicator */}
          {isFetchingNextPage && (
            <div className="py-4">
              <KanbanSkeleton columns={2} />
            </div>
          )}

          {/* Empty state */}
          {!isFetchingNextPage && memos.length === 0 && <EmptyState type={getEmptyType()} />}

          {/* End of list indicator */}
          {!isFetchingNextPage && !hasNextPage && memos.length > 0 && <EndIndicator />}
        </>
      )}
    </div>
  );
});

MemoListV3.displayName = "MemoListV3";
