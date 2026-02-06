import { AlertCircle, CheckCircle2, Clock, DollarSign, FileEdit, Wrench, XCircle, Zap } from "lucide-react";
import { cn } from "@/lib/utils";
import { type Block } from "@/types/block";

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
  totalCostUSD?: number;
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
  const hasCost = summary.totalCostUSD && summary.totalCostUSD > 0;
  const hasTools = summary.toolCallCount && summary.toolCallCount > 0;
  const hasFiles = summary.filesModified && summary.filesModified > 0;

  // Filter out zero-value duration breakdown parts
  const hasThinkingBreakdown = summary.thinkingDurationMs && summary.thinkingDurationMs > 0;
  const hasToolBreakdown = summary.toolDurationMs && summary.toolDurationMs > 0;
  const hasGenerationBreakdown = summary.generationDurationMs && summary.generationDurationMs > 0;

  // Don't render if no meaningful data
  if (!hasTiming && !hasTokens && !hasCost && !hasTools && !hasFiles) {
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
            <span className="text-[11px] text-muted-foreground/70 hidden sm:inline">
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
            <span className="text-[11px]">token</span>
            {/* Input/Output breakdown */}
            {(summary.totalInputTokens || summary.totalOutputTokens) && (
              <span className="text-[11px] text-muted-foreground/70 hidden sm:inline">
                (in:{formatNumber(summary.totalInputTokens || 0)}/out:{formatNumber(summary.totalOutputTokens || 0)})
              </span>
            )}
          </div>
        </>
      )}

      {/* Cost */}
      {hasCost && (
        <>
          {(hasTiming || hasTokens) && <span className="text-border/50">•</span>}
          <div className="flex items-center gap-1.5">
            <DollarSign className="w-3.5 h-3.5 text-green-500" />
            <span className="font-mono font-medium">${summary.totalCostUSD!.toFixed(4)}</span>
          </div>
        </>
      )}

      {/* Tools */}
      {hasTools && (
        <>
          {(hasTiming || hasTokens || hasCost) && <span className="text-border/50">•</span>}
          <div className="flex items-center gap-1.5">
            <Wrench className="w-3.5 h-3.5 text-purple-500" />
            <span className="font-mono font-medium">{summary.toolCallCount}</span>
            <span className="text-[11px]">calls</span>
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
            <span className="text-[11px]">files</span>
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
      {totalTokens > 0 && <span>⚡ {totalTokens} token</span>}
      {summary.totalCostUSD && summary.totalCostUSD > 0 && <span>💰 ${summary.totalCostUSD.toFixed(4)}</span>}
      {summary.toolCallCount && summary.toolCallCount > 0 && <span>🔧 {summary.toolCallCount} calls</span>}
    </div>
  );
}

/**
 * Convert Block to SessionSummaryData for normal mode display
 *
 * Extracts token usage and timing data from a Block for display in SessionSummaryPanel
 */
export function blockToSessionSummary(block: Block): SessionSummaryData {
  const tokenUsage = block.tokenUsage;
  const sessionStats = block.sessionStats;

  // Use tokenUsage from Block (normal mode) or fall back to sessionStats
  const inputTokens = tokenUsage?.promptTokens || sessionStats?.inputTokens || 0;
  const outputTokens = tokenUsage?.completionTokens || sessionStats?.outputTokens || 0;
  const cacheRead = tokenUsage?.cacheReadTokens || sessionStats?.cacheReadTokens || 0;
  const cacheWrite = tokenUsage?.cacheWriteTokens || sessionStats?.cacheWriteTokens || 0;

  // Use sessionStats timing for Geek/Evolution, or Block metadata for normal mode
  // Normal mode blocks may have timing in metadata JSON
  let thinkingMs: number | undefined;
  let generationMs: number | undefined;
  let totalMs: number | undefined;
  let toolCount: number | undefined;

  // Convert bigint to number if available
  if (sessionStats?.thinkingDurationMs) {
    thinkingMs = Number(sessionStats.thinkingDurationMs);
  }
  if (sessionStats?.generationDurationMs) {
    generationMs = Number(sessionStats.generationDurationMs);
  }
  if (sessionStats?.totalDurationMs) {
    totalMs = Number(sessionStats.totalDurationMs);
  }
  if (sessionStats?.toolCallCount !== undefined) {
    toolCount = sessionStats.toolCallCount;
  }

  // For normal mode, try to extract from metadata if sessionStats is not available
  if (!totalMs && block.metadata) {
    try {
      const meta = JSON.parse(block.metadata);
      thinkingMs = meta.thinking_duration_ms;
      generationMs = meta.generation_duration_ms;
      totalMs = meta.total_duration_ms;
      toolCount = meta.tool_call_count;
    } catch {
      // Ignore parse errors
    }
  }

  // Calculate cost from cost_estimate (in milli-cents) or sessionStats
  let costUSD: number | undefined;
  if (block.costEstimate && block.costEstimate > 0n) {
    // Convert milli-cents to USD: divide by 100000
    costUSD = Number(block.costEstimate) / 100000;
  } else if (sessionStats?.totalCostUsd) {
    costUSD = sessionStats.totalCostUsd;
  }

  return {
    sessionId: block.ccSessionId || undefined,
    totalDurationMs: totalMs,
    thinkingDurationMs: thinkingMs,
    generationDurationMs: generationMs,
    totalInputTokens: inputTokens,
    totalOutputTokens: outputTokens,
    totalCacheWriteTokens: cacheWrite,
    totalCacheReadTokens: cacheRead,
    toolCallCount: toolCount,
    toolsUsed: sessionStats?.toolsUsed,
    filesModified: sessionStats?.filesModified,
    filePaths: sessionStats?.filePaths,
    totalCostUSD: costUSD,
    status: block.status === 4 ? "error" : block.status === 3 ? "success" : undefined,
  };
}

/**
 * NormalModeBlockSummary - Session summary for normal mode blocks
 *
 * Wraps SessionSummaryPanel with data extracted from a Block
 */
interface NormalModeBlockSummaryProps {
  block: Block;
  className?: string;
}

export function NormalModeBlockSummary({ block, className }: NormalModeBlockSummaryProps) {
  const summary = blockToSessionSummary(block);
  return <SessionSummaryPanel summary={summary} className={className} />;
}
