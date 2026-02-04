import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import { ReferencedMemo } from "@/types/aichat";

interface MiniMemoCardProps {
  memo: ReferencedMemo;
  rank?: number;
  showRank?: boolean;
  className?: string;
}

export function MiniMemoCard({ memo, rank, showRank = true, className }: MiniMemoCardProps) {
  const { t } = useTranslation();
  const scorePercentage = Math.round(memo.score * 100);
  const scoreColor = getScoreColor(memo.score);

  return (
    <Link
      to={`/memos/${memo.uid}`}
      className={cn("block p-2.5 rounded-lg border hover:bg-muted transition-colors group", "bg-card border-border", className)}
    >
      <div className="flex items-start gap-2">
        {showRank && rank !== undefined && (
          <div
            className={cn(
              "flex-shrink-0 w-5 h-5 rounded flex items-center justify-center text-xs font-medium",
              rank <= 3 ? "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300" : "bg-muted text-muted-foreground",
            )}
          >
            {rank}
          </div>
        )}

        <div className="flex-1 min-w-0">
          <p className="text-xs text-muted-foreground line-clamp-3 leading-relaxed">{memo.content}</p>
          <div className="flex items-center justify-between mt-1.5">
            <span className="text-xs text-muted-foreground">{memo.timestamp ? formatTime(memo.timestamp, t) : ""}</span>
            <span className={cn("text-xs px-1.5 py-0.5 rounded", scoreColor)}>{scorePercentage}%</span>
          </div>
        </div>
      </div>
    </Link>
  );
}

function getScoreColor(score: number): string {
  if (score >= 0.9) {
    return "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300";
  } else if (score >= 0.7) {
    return "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300";
  } else if (score >= 0.5) {
    return "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300";
  } else {
    return "bg-muted text-muted-foreground";
  }
}

function formatTime(timestamp: number, t: (key: string, params?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return t("ai.mini_memo_card.just_now");
  if (diffMins < 60) return t("ai.mini_memo_card.minutes_ago", { count: diffMins });
  if (diffHours < 24) return t("ai.mini_memo_card.hours_ago", { count: diffHours });
  if (diffDays < 7) return t("ai.mini_memo_card.days_ago", { count: diffDays });

  return date.toLocaleDateString();
}
