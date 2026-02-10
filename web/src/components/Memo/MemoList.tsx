/**
 * MemoList - Modern Grid Layout with MemoBlockV2
 *
 * Features:
 * - Responsive 2-column grid (desktop) / 1-column (mobile)
 * - MemoBlockV2 with Fluid Card design
 * - Infinite scroll with intersection observer
 * - Filter integration
 * - Loading and empty states
 * - Staggered reveal animations
 */

import { memo, useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import Empty from "@/components/Empty";
import { MemoBlockV2 } from "@/components/Memo/MemoBlockV2";
import Skeleton from "@/components/Skeleton";
import { DEFAULT_LIST_MEMOS_PAGE_SIZE } from "@/helpers/consts";
import { useInfiniteMemos } from "@/hooks/useMemoQueries";
import { cn } from "@/lib/utils";
import { State } from "@/types/proto/api/v1/common_pb";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

export interface MemoListProps {
  state?: State;
  orderBy?: string;
  filter?: string;
  pageSize?: number;
  showCreator?: boolean;
  onEdit?: (memo: Memo) => void;
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
  showCreator,
  onEdit,
  className,
}: MemoListProps) {
  const { t } = useTranslation();

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

  // Animation delay for staggered reveal
  const getAnimationDelay = (index: number): number => {
    return index < 5 ? index * 50 : 50 + (index - 5) * 30;
  };

  return (
    <div className={cn("flex flex-col w-full", className)}>
      {/* Show skeleton loader during initial load */}
      {isLoading ? (
        <div className="w-full">
          <Skeleton showCreator={showCreator} count={4} />
        </div>
      ) : (
        <>
          {/* Memo Grid - Single column layout */}
          <div className="flex flex-col gap-4 w-full">
            {memos.map((memo, index) => (
              <div
                key={memo.name}
                className="animate-in fade-in slide-in-from-bottom-4 duration-300"
                style={{
                  animationDelay: `${getAnimationDelay(index)}ms`,
                  animationFillMode: "both",
                }}
              >
                <MemoBlockV2 memo={memo} onEdit={handleEdit} />
              </div>
            ))}
          </div>

          {/* Loading indicator for pagination */}
          {isFetchingNextPage && (
            <div className="flex flex-col gap-4 mt-4">
              {[1, 2, 3, 4].map((i) => (
                <div key={`skeleton-${i}`} className="h-40 bg-zinc-100 dark:bg-zinc-800 rounded-xl animate-pulse" />
              ))}
            </div>
          )}

          {/* Empty state */}
          {!isFetchingNextPage && memos.length === 0 && (
            <div className="w-full mt-12 mb-8 flex flex-col justify-center items-center">
              <Empty />
              <p className="mt-2 text-muted-foreground">{t("message.no-data")}</p>
            </div>
          )}

          {/* End of list indicator */}
          {!isFetchingNextPage && !hasNextPage && memos.length > 0 && (
            <div className="w-full text-center py-8 text-muted-foreground text-sm">{t("memo.end_of_list") || "You've reached the end"}</div>
          )}
        </>
      )}
    </div>
  );
});

MemoList.displayName = "MemoList";
