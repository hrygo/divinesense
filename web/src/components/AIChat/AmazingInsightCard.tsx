import dayjs from "dayjs";
import { Calendar, FileText, Lightbulb, TrendingUp } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import type { MemoSummary, ScheduleSummary } from "@/types/parrot";

interface AmazingInsightCardProps {
  memos: MemoSummary[];
  schedules: ScheduleSummary[];
  insight?: string;
  onMemoClick?: (uid: string) => void;
  onScheduleClick?: (schedule: ScheduleSummary) => void;
  className?: string;
}

/**
 * AmazingInsightCard - A generative UI card for Amazing Agent
 *
 * Displays a two-column layout with:
 * - Left: Key Memos (recent/relevant notes)
 * - Right: Upcoming Events (schedules)
 * - Bottom: Purple gradient insight bar with AI-generated insight
 *
 * 生成式 UI 卡片组件，用于 Amazing Agent 的综合洞察展示
 */
export function AmazingInsightCard({ memos, schedules, insight, onMemoClick, onScheduleClick, className }: AmazingInsightCardProps) {
  const { t } = useTranslation();

  // Limit display to 3 items each
  const displayMemos = memos.slice(0, 3);
  const displaySchedules = schedules.slice(0, 3);

  // Don't render if both are empty
  if (displayMemos.length === 0 && displaySchedules.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "rounded-2xl border border-purple-100 dark:border-purple-800/30",
        "bg-white/80 dark:bg-zinc-800/50 backdrop-blur-sm",
        "shadow-[0_4px_20px_-4px_rgba(139,92,246,0.1)] overflow-hidden",
        className,
      )}
    >
      {/* Header - Minimalist */}
      <div className="px-4 py-2.5 flex items-center justify-between border-b border-purple-50 dark:border-purple-800/20">
        <h3 className="font-semibold text-sm text-purple-900 dark:text-purple-100 flex items-center gap-2">
          <TrendingUp className="w-4 h-4 text-purple-600 dark:text-purple-400" />
          {t("ai.aichat.amazing-insight.title")}
        </h3>
        <span className="text-[11px] uppercase tracking-wider text-purple-400 font-medium">
          {t("ai.aichat.amazing-insight.auto_generated")}
        </span>
      </div>

      {/* Content Grid */}
      <div className="grid grid-cols-2 divide-x divide-purple-50 dark:divide-purple-800/20">
        {/* Left Column: Key Memos */}
        <div className="p-3">
          <div className="flex items-center gap-1.5 mb-2 opacity-80">
            <FileText className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
            <h4 className="font-medium text-[11px] uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
              {t("ai.aichat.amazing-insight.key-memos")}
            </h4>
          </div>

          {displayMemos.length === 0 ? (
            <p className="text-xs text-zinc-400 dark:text-zinc-500 py-2 italic">{t("ai.aichat.memo-query.no-results")}</p>
          ) : (
            <ul className="space-y-1.5">
              {displayMemos.map((memo) => (
                <li key={memo.uid}>
                  <Link
                    to={`/memo/${memo.uid}`}
                    onClick={(e) => {
                      if (onMemoClick) {
                        e.preventDefault();
                        onMemoClick(memo.uid);
                      }
                    }}
                    className="group block p-1.5 -m-1.5 rounded-lg hover:bg-purple-50/50 dark:hover:bg-purple-900/10 transition-colors"
                  >
                    <div className="flex items-start gap-2">
                      <span className="text-purple-300 dark:text-purple-600 mt-1 font-bold text-[10px]">#</span>
                      <span className="flex-1 text-xs text-zinc-600 dark:text-zinc-300 leading-relaxed group-hover:text-purple-700 dark:group-hover:text-purple-300 transition-colors break-words">
                        {memo.content}
                      </span>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Right Column: Upcoming Events */}
        <div className="p-3">
          <div className="flex items-center gap-1.5 mb-2 opacity-80">
            <Calendar className="w-3.5 h-3.5 text-purple-600 dark:text-purple-400" />
            <h4 className="font-medium text-[11px] uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
              {t("ai.aichat.amazing-insight.upcoming-events")}
            </h4>
          </div>

          {displaySchedules.length === 0 ? (
            <p className="text-xs text-zinc-400 dark:text-zinc-500 py-2 italic border-zinc-100 dark:border-zinc-800">
              {t("schedule.no-schedules-this-period")}
            </p>
          ) : (
            <ul className="space-y-1.5">
              {displaySchedules.map((schedule) => (
                <li key={schedule.uid}>
                  <button
                    onClick={() => onScheduleClick?.(schedule)}
                    className="group w-full block p-1.5 -m-1.5 rounded-lg text-left hover:bg-purple-50/50 dark:hover:bg-purple-900/10 transition-colors cursor-pointer"
                  >
                    <div className="flex items-start gap-2 text-wrap">
                      <span className="w-1.5 h-1.5 rounded-full bg-purple-400 dark:bg-purple-500 shrink-0 shadow-[0_0_8px_rgba(167,139,250,0.5)] mt-1.5" />
                      <div className="flex-1 min-w-0">
                        <div className="text-[11px] font-bold text-purple-600 dark:text-purple-400 uppercase tracking-tighter">
                          {formatScheduleTime(schedule, t)}
                        </div>
                        <div className="text-xs text-zinc-700 dark:text-zinc-300 group-hover:text-purple-700 dark:group-hover:text-purple-300 transition-colors break-words">
                          {schedule.title}
                        </div>
                      </div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* Bottom Insight Bar */}
      {insight && (
        <div className="px-4 py-3 bg-gradient-to-r from-purple-600 to-indigo-600 dark:from-purple-700 dark:to-indigo-700">
          <div className="flex items-start gap-2">
            <Lightbulb className="w-4 h-4 text-white/90 mt-0.5 shrink-0" />
            <p className="text-sm text-white/95 leading-relaxed">
              <span className="font-medium">{t("ai.aichat.amazing-insight.insight-prefix")}:</span> {insight}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Format schedule time for display
 */
function formatScheduleTime(schedule: ScheduleSummary, t: (key: string) => string): string {
  const start = dayjs.unix(schedule.startTimestamp);
  const now = dayjs();

  // Determine date prefix
  let datePrefix = "";
  if (start.isSame(now, "day")) {
    datePrefix = "";
  } else if (start.isSame(now.add(1, "day"), "day")) {
    datePrefix = "Tomorrow, ";
  } else if (start.diff(now, "day") < 7) {
    datePrefix = `${start.format("ddd")}, `;
  } else {
    datePrefix = `${start.format("MM/DD")}, `;
  }

  // Format time
  const timeStr = schedule.allDay ? t("ai.amazing_insight.all_day") : start.format("h:mm A");

  return `${datePrefix}${timeStr}`;
}
