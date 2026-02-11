/**
 * MemoList - Modern Grid Layout with MemoBlockV2
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：笔记如思绪般浮现
 * - 留白：充足的间距让内容呼吸
 * - 流动：渐进式展示，不打断心流
 *
 * Features:
 * - Single column layout (mobile + desktop)
 * - MemoBlockV2 with Fluid Card design
 * - Infinite scroll with intersection observer
 * - Filter integration
 * - Zen-style loading and empty states
 * - Staggered reveal animations
 */

import { memo, useCallback, useEffect, useMemo, useRef } from "react";
import { MemoBlockV2 } from "@/components/Memo/MemoBlockV2";
import { DEFAULT_LIST_MEMOS_PAGE_SIZE } from "@/helpers/consts";
import { useInfiniteMemos } from "@/hooks/useMemoQueries";
import { cn } from "@/lib/utils";
import { State } from "@/types/proto/api/v1/common_pb";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";
import { EmptyState, EndIndicator, LoadingSkeleton, PaginationSkeleton } from "./MemoListStates";

export interface MemoListProps {
  state?: State;
  orderBy?: string;
  filter?: string;
  pageSize?: number;
  onEdit?: (memo: Memo) => void;
  showCreator?: boolean;
  className?: string;
}

/**
 * Auto-fetch hook for non-scrollable pages
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

/**
 * MemoList Component
 */
export const MemoList = memo(function MemoList({
  state = State.NORMAL,
  orderBy = "display_time desc",
  filter,
  pageSize = DEFAULT_LIST_MEMOS_PAGE_SIZE,
  onEdit,
  className,
}: MemoListProps) {
  // const { t } = useTranslation();
  // 保留 t 以备将来国际化使用
  // eslint-disable-next-line @typescript-eslint/no-unused-vars

  // Use React Query's infinite query for pagination
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useInfiniteMemos({
    state,
    orderBy,
    filter,
    pageSize,
  });

  // Flatten pages into a single array of memos
  const memos = useMemo(() => data?.pages.flatMap((page) => page.memos) || [], [data?.pages]);

  // Auto-fetch hook: fetches more content when page isn't scrollable
  useAutoFetchWhenNotScrollable({
    hasNextPage,
    isFetchingNextPage,
    memoCount: memos.length,
    onFetchNext: fetchNextPage,
  });

  // Infinite scroll: fetch more when user scrolls near bottom
  useEffect(() => {
    if (!hasNextPage) return;

    const handleScroll = () => {
      const nearBottom = window.innerHeight + window.scrollY >= document.body.offsetHeight - 300;
      if (nearBottom && !isFetchingNextPage) {
        fetchNextPage();
      }
    };

    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  // Handle memo actions - MemoBlock 内部处理大部分操作，只有 Edit 需要外部处理
  const handleEdit = useCallback(
    (memo: Memo) => {
      onEdit?.(memo);
    },
    [onEdit],
  );

  // Animation delay for staggered reveal - 思绪浮现效果
  const getAnimationDelay = (index: number): number => {
    // 前几个快速浮现，后面的更慢，模拟思绪涌现
    return index < 5 ? index * 40 : 200 + (index - 5) * 20;
  };

  // 判断空状态类型
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
      {/* Initial loading - 初始加载骨架屏 */}
      {isLoading ? (
        <LoadingSkeleton count={4} />
      ) : (
        <>
          {/* Memo Grid - Single column layout */}
          <div className="flex flex-col gap-4 w-full">
            {memos.map((memo, index) => (
              <div
                key={memo.name}
                className="animate-in fade-in slide-in-from-bottom-4 duration-500 ease-out"
                style={{
                  animationDelay: `${getAnimationDelay(index)}ms`,
                  animationFillMode: "both",
                }}
              >
                <MemoBlockV2 memo={memo} onEdit={handleEdit} />
              </div>
            ))}
          </div>

          {/* Pagination skeleton - 分页加载骨架屏 */}
          {isFetchingNextPage && <PaginationSkeleton />}

          {/* Empty state - 空状态 */}
          {!isFetchingNextPage && memos.length === 0 && <EmptyState type={getEmptyType()} />}

          {/* End of list indicator - 列表结束指示器 */}
          {!isFetchingNextPage && !hasNextPage && memos.length > 0 && <EndIndicator />}
        </>
      )}
    </div>
  );
});

MemoList.displayName = "MemoList";
