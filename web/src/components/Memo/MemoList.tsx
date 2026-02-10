/**
 * MemoList - Single-column Timeline List Container
 *
 * Replaces PagedMemoList + MasonryView with a simpler
 * single-column timeline layout.
 *
 * Features:
 * - Infinite scroll (reuses existing logic)
 * - Filter integration
 * - Loading states
 * - Empty states
 */

import { memo, useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import Empty from "@/components/Empty";
import { MemoBlock } from "@/components/Memo";
import { MemoTimelineNode } from "@/components/Memo/MemoTimelineNode";
import MemoFilters from "@/components/MemoFilters";
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
  onDelete?: (memo: Memo) => void;
  onArchive?: (name: string, archived: boolean) => void;
  onPin?: (name: string, pinned: boolean) => void;
  onCopy?: (content: string) => void;
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
  onDelete,
  onArchive,
  onPin,
  onCopy,
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
  const memos = useMemo(() => data?.pages.flatMap((page) => page.memos) || [], [data]);

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

  // Handle memo actions
  const handleEdit = useCallback(
    (memo: Memo) => {
      onEdit?.(memo);
    },
    [onEdit],
  );

  const handleDelete = useCallback(
    (memo: Memo) => {
      onDelete?.(memo);
    },
    [onDelete],
  );

  const handleArchive = useCallback(
    (name: string, archived: boolean) => {
      onArchive?.(name, archived);
    },
    [onArchive],
  );

  const handlePin = useCallback(
    (name: string, pinned: boolean) => {
      onPin?.(name, pinned);
    },
    [onPin],
  );

  const handleCopy = useCallback(
    (content: string) => {
      onCopy?.(content);
    },
    [onCopy],
  );

  return (
    <div className={cn("flex flex-col w-full", className)}>
      {/* Show skeleton loader during initial load */}
      {isLoading ? (
        <Skeleton showCreator={showCreator} count={4} />
      ) : (
        <>
          {/* Filter Bar */}
          <div className="mb-4">
            <MemoFilters />
          </div>

          {/* Memo List - Single Column Timeline */}
          <div className="flex flex-col gap-4 max-w-4xl mx-auto w-full">
            {memos.map((memo, index) => (
              <div key={memo.name} className="flex items-start gap-2">
                {/* Timeline Node - Inspired by Chat's design */}
                <div className="flex flex-col items-center pt-2 shrink-0">
                  <MemoTimelineNode memo={memo} isLatest={index === 0} size="sm" />
                  {/* Timeline connector line */}
                  {index < memos.length - 1 && <div className="w-px h-6 bg-border/40 mt-0.5" />}
                </div>

                {/* Memo Block */}
                <div className="flex-1 min-w-0">
                  <MemoBlock
                    memo={memo}
                    isLatest={index === 0}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                    onArchive={handleArchive}
                    onPin={handlePin}
                    onCopy={handleCopy}
                  />
                </div>
              </div>
            ))}

            {/* Loading indicator for pagination */}
            {isFetchingNextPage && <Skeleton showCreator={showCreator} count={2} />}

            {/* Empty state */}
            {!isFetchingNextPage && memos.length === 0 && (
              <div className="w-full mt-12 mb-8 flex flex-col justify-center items-center">
                <Empty />
                <p className="mt-2 text-muted-foreground">{t("message.no-data")}</p>
              </div>
            )}

            {/* End of list indicator */}
            {!isFetchingNextPage && !hasNextPage && memos.length > 0 && (
              <div className="w-full text-center py-8 text-muted-foreground text-sm">
                {t("memo.end_of_list") || "You've reached the end"}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
});

MemoList.displayName = "MemoList";
