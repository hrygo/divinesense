import { AlertCircle, CheckCircle2, Clock, FileEdit, Wrench, XCircle, Zap } from "lucide-react";
import { cn } from "@/lib/utils";

interface SessionSummaryData {
  sessionId?: string;
  totalDurationMs?: number;
  thinkingDurationMs?: number;
  toolDurationMs?: number;
  generationDurationMs?: number;
  totalInputTokens?: number;
  totalOutputTokens?: number;
  totalCacheWriteTokens?: number;
  totalCacheReadTokens?: number;
  toolCallCount?: number;
  toolsUsed?: string[];
  filesModified?: number;
  filePaths?: string[];
  status?: string;
}

interface SessionSummaryPanelProps {
  summary: SessionSummaryData;
  className?: string;
}

// Status configuration
const STATUS_CONFIG = {
  success: { color: "text-emerald-600 dark:text-emerald-400", icon: CheckCircle2 },
  error: { color: "text-red-600 dark:text-red-400", icon: XCircle },
  cancelled: { color: "text-amber-600 dark:text-amber-400", icon: AlertCircle },
} as const;

type StatusType = keyof typeof STATUS_CONFIG;

/**
 * SessionSummaryPanel - Compact single-line session summary for Geek/Evolution modes
 *
 * Shows: status | duration | tokens | tools | files
 */
export function SessionSummaryPanel({ summary, className }: SessionSummaryPanelProps) {
  // Format duration in human-readable format
  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  // Format large numbers with unit suffix
  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  // Get status configuration
  const statusKey: StatusType =
    summary.status?.toLowerCase() === "error" ? "error" : summary.status?.toLowerCase() === "cancelled" ? "cancelled" : "success";
  const statusCfg = STATUS_CONFIG[statusKey] || STATUS_CONFIG.success;

  // Calculate meaningful metrics - only show values that are meaningful
  const totalTokens = (summary.totalInputTokens || 0) + (summary.totalOutputTokens || 0);
  const hasTiming = summary.totalDurationMs && summary.totalDurationMs > 0;
  const hasTokens = totalTokens > 0;
  const hasTools = summary.toolCallCount && summary.toolCallCount > 0;
  const hasFiles = summary.filesModified && summary.filesModified > 0;

  // Filter out zero-value duration breakdown parts
  const hasThinkingBreakdown = summary.thinkingDurationMs && summary.thinkingDurationMs > 0;
  const hasToolBreakdown = summary.toolDurationMs && summary.toolDurationMs > 0;
  const hasGenerationBreakdown = summary.generationDurationMs && summary.generationDurationMs > 0;

  // Don't render if no meaningful data
  if (!hasTiming && !hasTokens && !hasTools && !hasFiles) {
    return null;
  }

  const StatusIcon = statusCfg.icon;

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground px-3 py-2",
        "bg-muted/30 rounded-lg border border-border/50",
        className,
      )}
    >
      {/* Status */}
      <div className={cn("flex items-center gap-1", statusCfg.color)}>
        <StatusIcon className="w-3.5 h-3.5" />
        <span className="font-medium capitalize">{statusKey}</span>
      </div>

      {/* Separator */}
      <span className="text-border/50">|</span>

      {/* Duration with breakdown */}
      {hasTiming && (
        <div className="flex items-center gap-1.5">
          <Clock className="w-3.5 h-3.5 text-blue-500" />
          <span className="font-mono font-medium">{formatDuration(summary.totalDurationMs!)}</span>
          {/* Breakdown with space separators - only show non-zero values */}
          {(hasThinkingBreakdown || hasToolBreakdown || hasGenerationBreakdown) && (
            <span className="text-[10px] text-muted-foreground/70 hidden sm:inline">
              (
              {[
                (summary.thinkingDurationMs || 0) > 0 && `💭${formatDuration(summary.thinkingDurationMs!)}`,
                (summary.toolDurationMs || 0) > 0 && `🔧${formatDuration(summary.toolDurationMs!)}`,
                (summary.generationDurationMs || 0) > 0 && `✍${formatDuration(summary.generationDurationMs!)}`,
              ]
                .filter(Boolean)
                .join(" + ")}
              )
            </span>
          )}
        </div>
      )}

      {/* Tokens with breakdown */}
      {hasTokens && (
        <>
          {hasTiming && <span className="text-border/50">•</span>}
          <div className="flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5 text-amber-500" />
            <span className="font-mono font-medium">{formatNumber(totalTokens)}</span>
            <span className="text-[10px]">tokens</span>
            {/* Input/Output breakdown */}
            {(summary.totalInputTokens || summary.totalOutputTokens) && (
              <span className="text-[10px] text-muted-foreground/70 hidden sm:inline">
                (in:{formatNumber(summary.totalInputTokens || 0)}/out:{formatNumber(summary.totalOutputTokens || 0)})
              </span>
            )}
          </div>
        </>
      )}

      {/* Tools */}
      {hasTools && (
        <>
          {(hasTiming || hasTokens) && <span className="text-border/50">•</span>}
          <div className="flex items-center gap-1.5">
            <Wrench className="w-3.5 h-3.5 text-purple-500" />
            <span className="font-mono font-medium">{summary.toolCallCount}</span>
            <span className="text-[10px]">calls</span>
          </div>
        </>
      )}

      {/* Files (Evolution Mode) */}
      {hasFiles && (
        <>
          {(hasTiming || hasTokens || hasTools) && <span className="text-border/50">•</span>}
          <div className="flex items-center gap-1.5">
            <FileEdit className="w-3.5 h-3.5 text-green-500" />
            <span className="font-mono font-medium">{summary.filesModified}</span>
            <span className="text-[10px]">files</span>
          </div>
        </>
      )}
    </div>
  );
}

/**
 * CompactSessionSummary - Minimal inline summary for chat footer
 */
interface CompactSessionSummaryProps {
  summary: SessionSummaryData;
  className?: string;
}

export function CompactSessionSummary({ summary, className }: CompactSessionSummaryProps) {
  const totalTokens = (summary.totalInputTokens || 0) + (summary.totalOutputTokens || 0);

  return (
    <div className={cn("flex items-center gap-3 text-xs text-muted-foreground", className)}>
      {summary.totalDurationMs && summary.totalDurationMs > 0 && <span>⏱ {summary.totalDurationMs}ms</span>}
      {totalTokens > 0 && <span>⚡ {totalTokens} tokens</span>}
      {summary.toolCallCount && summary.toolCallCount > 0 && <span>🔧 {summary.toolCallCount} calls</span>}
    </div>
  );
}
