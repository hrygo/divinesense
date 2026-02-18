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

import { Archive, Filter, Lightbulb, Search, X } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { MemoBlockV3 } from "@/components/Memo/MemoBlockV3";
import { MemoSkeletonGrid } from "@/components/Memo/MemoSkeleton";
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
  onClearSearch?: () => void;
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
// Empty State
// ============================================================================

interface EmptyStateProps {
  type: "all" | "filtered" | "search" | "archived";
  searchKeyword?: string;
  onClearSearch?: () => void;
}

/**
 * EmptyState - 空状态组件
 *
 * 设计要点：
 * - 首次使用：显示欢迎引导和示例
 * - 搜索无结果：显示关键词、建议和清除按钮
 * - 筛选无结果：显示筛选建议
 * - 归档为空：显示归档操作提示
 */
function EmptyState({ type, searchKeyword, onClearSearch }: EmptyStateProps) {
  const { t } = useTranslation();

  // 首次使用引导 - 显示示例
  if (type === "all") {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        {/* 主图标 */}
        <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center mb-4">
          <Lightbulb className="w-8 h-8 text-amber-500 dark:text-amber-400" />
        </div>

        {/* 标题和描述 */}
        <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-1">{t("memo.empty_all_title")}</h3>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 mb-6">{t("memo.empty_all_subtitle")}</p>

        {/* 示例引导 */}
        <div className="bg-zinc-50 dark:bg-zinc-800/50 rounded-xl p-4 max-w-[20rem] text-left">
          <p className="text-xs text-zinc-400 dark:text-zinc-500 mb-3">{t("memo.empty_examples_title")}</p>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
              <span className="w-1.5 h-1.5 rounded-full bg-amber-400" />
              {t("memo.empty_example_1")}
            </div>
            <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
              <span className="w-1.5 h-1.5 rounded-full bg-violet-400" />
              {t("memo.empty_example_2")}
            </div>
            <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
              <span className="w-1.5 h-1.5 rounded-full bg-orange-400" />
              {t("memo.empty_example_3")}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // 搜索无结果 - 显示关键词、建议和清除按钮
  if (type === "search") {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-sky-100 dark:bg-sky-900/30 flex items-center justify-center mb-4">
          <Search className="w-8 h-8 text-sky-500 dark:text-sky-400" />
        </div>
        <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-1">{t("memo.empty_search_title")}</h3>

        {/* 显示搜索关键词 */}
        {searchKeyword && (
          <p className="text-sm text-zinc-600 dark:text-zinc-400 mb-2">{t("memo.empty_search_keyword", { keyword: searchKeyword })}</p>
        )}

        {/* 建议 */}
        <div className="text-sm text-zinc-500 dark:text-zinc-400 mb-4">
          <p>{t("memo.empty_search_suggestions")}</p>
        </div>

        {/* 清除搜索按钮 */}
        {onClearSearch && (
          <button
            onClick={onClearSearch}
            className={cn(
              "flex items-center gap-2 px-4 py-2 rounded-lg",
              "bg-zinc-100 dark:bg-zinc-800 hover:bg-zinc-200 dark:hover:bg-zinc-700",
              "text-sm text-zinc-600 dark:text-zinc-400",
              "transition-colors duration-200",
            )}
          >
            <X className="w-4 h-4" />
            {t("memo.empty_clear_search")}
          </button>
        )}
      </div>
    );
  }

  // 归档为空
  if (type === "archived") {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4">
          <Archive className="w-8 h-8 text-zinc-400 dark:text-zinc-500" />
        </div>
        <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-1">{t("memo.empty_archived_title")}</h3>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{t("memo.empty_archived_subtitle")}</p>
      </div>
    );
  }

  // 筛选无结果
  const config = {
    filtered: {
      icon: Filter,
      title: t("memo.empty_filtered_title"),
      description: t("memo.empty_filtered_subtitle"),
    },
  };

  const { icon: Icon, title, description } = config[type as keyof typeof config] || config.filtered;

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
  onClearSearch,
  className,
}: MemoListV3Props) {
  const { t } = useTranslation();

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

  // Keyboard navigation state (PC only)
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);

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

  // Keyboard navigation handler (PC only)
  useEffect(() => {
    // Only enable on desktop (sm breakpoint and above)
    if (typeof window === "undefined" || window.innerWidth < 640) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't handle if user is typing in an input
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, memos.length - 1));
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, -1));
          break;
        case "Enter":
          if (selectedIndex >= 0 && memos[selectedIndex]) {
            onEdit?.(memos[selectedIndex].name);
          }
          break;
        case "Escape":
          setSelectedIndex(-1);
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [memos, selectedIndex, onEdit]);

  // Scroll selected item into view when selectedIndex changes
  useEffect(() => {
    if (selectedIndex < 0 || !containerRef.current) return;

    const selectedElement = containerRef.current.querySelector(`[data-memo-index="${selectedIndex}"]`);
    if (selectedElement) {
      selectedElement.scrollIntoView({ behavior: "smooth", block: "nearest" });
    }
  }, [selectedIndex]);

  // Animation delay for staggered reveal - based on original index
  const getAnimationDelay = (index: number): number => {
    // Faster cascade for first few, then slower
    return index < 6 ? index * 60 : 360 + (index - 6) * 30;
  };

  // Extract search keyword from filter
  const searchKeyword = useMemo(() => {
    if (filter && filter.includes("contentSearch")) {
      const match = filter.match(/contentSearch=="([^"]+)"/);
      return match ? match[1] : undefined;
    }
    return undefined;
  }, [filter]);

  // Determine empty state type
  const getEmptyType = (): "all" | "filtered" | "search" | "archived" => {
    if (state === State.ARCHIVED) {
      return "archived";
    }
    if (filter && filter.includes("contentSearch")) {
      return "search";
    }
    if (filter) {
      return "filtered";
    }
    return "all";
  };

  return (
    <div ref={containerRef} className={cn("flex flex-col w-full", className)}>
      {/* Initial loading skeleton */}
      {isLoading ? (
        <MemoSkeletonGrid count={columnCount === 1 ? 3 : 6} />
      ) : (
        <>
          {/* Kanban Masonry - Responsive 1/2 columns with left-to-right, top-to-bottom render order */}
          <div className="flex gap-4 w-full">
            {columns.map((columnMemos, colIndex) => (
              <div key={colIndex} className="flex-1 flex flex-col gap-4">
                {columnMemos.map((memo) => {
                  // Calculate original index for animation
                  const originalIndex = memos.indexOf(memo);
                  const isSelected = originalIndex === selectedIndex;
                  return (
                    <div
                      key={memo.name}
                      data-memo-index={originalIndex}
                      className={cn(
                        "animate-in fade-in slide-in-from-bottom-3 duration-500 ease-out relative",
                        // Keyboard navigation: selected state indicator (left border)
                        isSelected && "ring-2 ring-primary/30 rounded-lg",
                      )}
                      style={{
                        animationDelay: `${getAnimationDelay(originalIndex)}ms`,
                        animationFillMode: "both",
                      }}
                      onClick={() => setSelectedIndex(originalIndex)}
                    >
                      {/* Selected indicator - left border */}
                      {isSelected && <div className="absolute left-0 top-0 bottom-0 w-1 bg-primary rounded-l-lg z-10" />}
                      <MemoBlockV3 memo={memo} onEdit={handleEdit} />
                    </div>
                  );
                })}
              </div>
            ))}
          </div>

          {/* Keyboard navigation hint (PC only, when items exist) */}
          {memos.length > 0 && selectedIndex === -1 && (
            <div className="hidden sm:flex items-center justify-center py-4 text-xs text-zinc-400 dark:text-zinc-500 gap-4">
              <span>{t("memo.keyboard_hint")}</span>
            </div>
          )}

          {/* Intersection observer target */}
          <div ref={loadMoreRef} className="h-px w-full" />

          {/* Loading more indicator */}
          {isFetchingNextPage && (
            <div className="py-4">
              <MemoSkeletonGrid count={4} />
            </div>
          )}

          {/* Empty state */}
          {!isFetchingNextPage && memos.length === 0 && (
            <EmptyState type={getEmptyType()} searchKeyword={searchKeyword} onClearSearch={onClearSearch} />
          )}

          {/* End of list indicator */}
          {!isFetchingNextPage && !hasNextPage && memos.length > 0 && <EndIndicator />}
        </>
      )}
    </div>
  );
});

MemoListV3.displayName = "MemoListV3";
