import dayjs from "dayjs";
import { SearchIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParseScheduleQuery } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleSearchBarProps {
  schedules: Schedule[];
  onFilteredChange?: (filtered: Schedule[]) => void;
  onHasFilterChange?: (hasFilter: boolean) => void;
  className?: string;
  autoFocus?: boolean;
}

/** Search and filter schedules by title, location, and description */
export const ScheduleSearchBar = ({ schedules, onFilteredChange, onHasFilterChange, className, autoFocus }: ScheduleSearchBarProps) => {
  const t = useTranslate();
  const [searchQuery, setSearchQuery] = useState("");
  const [semanticFilter, setSemanticFilter] = useState<{ startTs: bigint; endTs: bigint; label: string } | null>(null);
  const { mutateAsync: parseQuery, isPending: isParsing } = useParseScheduleQuery();

  // Debounce semantic parsing
  useEffect(() => {
    const timer = setTimeout(async () => {
      const query = searchQuery.trim();
      if (!query || query.length < 4) {
        // Only parse if enough context
        setSemanticFilter(null);
        return;
      }

      // If simple text match works well, maybe don't force AI?
      // But user wants "Next week".
      try {
        const parsed = await parseQuery(query);
        if (parsed) {
          setSemanticFilter({
            startTs: parsed.startTs,
            endTs: parsed.endTs,
            label: dayjs.unix(Number(parsed.startTs)).format("MM-DD HH:mm"),
          });
        } else {
          setSemanticFilter(null);
        }
      } catch (_e) {
        // Ignore parse errors, treat as pure text
        setSemanticFilter(null);
      }
    }, 800); // 800ms debounce

    return () => clearTimeout(timer);
  }, [searchQuery, parseQuery]);

  // Filter schedules based on search query AND/OR semantic time
  const filteredSchedules = useMemo(() => {
    let result = schedules;

    // 1. Semantic Time Filter (Overlap)
    if (semanticFilter) {
      result = result.filter((s) => {
        // Check overlap
        // Semantic window usually is the specific time detected.
        // If user says "Next week", AI might give specific slot.
        // We'll trust AI's specific slot for now or just use it as a "focus"
        // Logic: Schedule STARTs within the semantic window OR overlaps?
        // Let's use loose intersection
        const sStart = Number(s.startTs);
        const sEnd = Number(s.endTs);
        const fStart = Number(semanticFilter.startTs);
        const fEnd = Number(semanticFilter.endTs);

        // Overlap logic: StartA < EndB && StartB < EndA
        // But "Meeting next week" might return a 1 hour slot. filtering ONLY that 1 hour is too restrictive if the user meant "sometime".
        // BUT if the backend returns a 1 hour slot, we can't guess "Next Week".
        // So for now, we only use it if it seems to be useful.

        // Actually, let's treating semantic filter as "boost" or strictly matches?
        // Spec says: "Display parsed time range Tag".

        // Let's being conservative: If semantic filter is active, we PRIORITIZE it but maybe showing it as filter is better.
        return sStart < fEnd && sEnd > fStart;
      });
      // If result is empty after semantic, maybe fallback to text?
      if (result.length === 0) result = schedules;
    }

    // 2. Text Filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase().trim();
      result = result.filter((schedule) => {
        const title = schedule.title.toLowerCase();
        const location = (schedule.location || "").toLowerCase();
        const description = (schedule.description || "").toLowerCase();
        return title.includes(query) || location.includes(query) || description.includes(query);
      });
    }

    return result;
  }, [schedules, searchQuery, semanticFilter]);

  // Use refs to track previous values and avoid unnecessary callbacks
  const prevFilteredLengthRef = useRef(0);
  const prevFilteredNamesRef = useRef<string>("");

  // Notify parent of filtered results (only when content actually changes)
  useEffect(() => {
    const currentLength = filteredSchedules.length;
    const currentNames = filteredSchedules
      .map((s) => s.name)
      .sort()
      .join(",");

    // Only notify if the actual filtered content changed
    if (currentLength !== prevFilteredLengthRef.current || currentNames !== prevFilteredNamesRef.current) {
      prevFilteredLengthRef.current = currentLength;
      prevFilteredNamesRef.current = currentNames;
      onFilteredChange?.(filteredSchedules);
    }
  }, [filteredSchedules, onFilteredChange]);

  // Notify parent of filter state
  const hasFilter = searchQuery.trim().length > 0;
  const prevHasFilterRef = useRef(false);

  useEffect(() => {
    if (hasFilter !== prevHasFilterRef.current) {
      prevHasFilterRef.current = hasFilter;
      onHasFilterChange?.(hasFilter);
    }
  }, [hasFilter, onHasFilterChange]);

  const handleClear = useCallback(() => {
    setSearchQuery("");
    setSemanticFilter(null);
  }, []);

  const resultCount = filteredSchedules.length;
  const totalCount = schedules.length;

  return (
    <div className={cn("relative w-full", className)} role="search">
      <div className="relative flex items-center gap-2">
        <div className="relative flex-1">
          <SearchIcon
            className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none"
            aria-hidden="true"
          />
          <input
            type="text"
            id="schedule-search-input"
            role="searchbox"
            value={searchQuery}
            autoFocus={autoFocus}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("schedule.search-placeholder") || "Search schedules..."}
            aria-label={t("schedule.search-schedule") as string}
            aria-describedby={hasFilter ? "search-result-count" : undefined}
            className={cn(
              "h-9 w-full pl-9 pr-20 rounded-lg border border-border bg-background text-sm",
              "focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary",
              "focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2",
              "transition-colors",
            )}
          />
          {isParsing && (
            <div className="absolute right-2 top-1/2 -translate-y-1/2">
              <span className="w-4 h-4 block rounded-full border-2 border-primary/30 border-t-primary animate-spin" />
            </div>
          )}
          {hasFilter && !isParsing && (
            <div
              className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1"
              id="search-result-count"
              role="status"
              aria-live="polite"
            >
              <span className="text-xs text-muted-foreground" aria-label={`找到 ${resultCount} 个结果，共 ${totalCount} 个日程`}>
                {resultCount}/{totalCount}
              </span>
              <button
                type="button"
                onClick={handleClear}
                className="p-1.5 rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors min-h-[36px] min-w-[36px] focus-visible:ring-2 focus-visible:ring-border focus-visible:ring-offset-2 flex items-center justify-center"
                aria-label={t("schedule.clear-search") as string}
              >
                <XIcon className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Semantic Tag Display */}
      {semanticFilter && (
        <div className="mt-2 flex items-center gap-2 animate-in fade-in slide-in-from-top-1">
          <span className="text-xs font-medium text-muted-foreground">{t("schedule.time_filter")}:</span>
          <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-primary/10 text-primary border border-primary/20">
            {semanticFilter.label}
          </span>
        </div>
      )}
    </div>
  );
};

export default ScheduleSearchBar;
