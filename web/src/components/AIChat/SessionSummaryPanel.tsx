import { AlertCircle, CheckCircle2, Clock, FileCode, Wrench, XCircle, Zap } from "lucide-react";
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
  success: { color: "text-green-500", bg: "bg-green-500/10", icon: CheckCircle2, label: "Success" },
  error: { color: "text-red-500", bg: "bg-red-500/10", icon: XCircle, label: "Error" },
  cancelled: { color: "text-yellow-500", bg: "bg-yellow-500/10", icon: AlertCircle, label: "Cancelled" },
} as const;

type StatusType = keyof typeof STATUS_CONFIG;

/**
 * SessionSummaryPanel - Displays session statistics for Geek/Evolution modes
 *
 * Shows timing breakdown, token usage, and tool call summary
 */
export function SessionSummaryPanel({ summary, className }: SessionSummaryPanelProps) {
  // Format duration in human-readable format
  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  // Format large numbers
  const formatNumber = (num: number) => {
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
  };

  // Calculate total tokens
  const totalTokens = (summary.totalInputTokens || 0) + (summary.totalOutputTokens || 0);

  // Get status configuration
  const statusKey: StatusType =
    summary.status?.toLowerCase() === "error" ? "error" : summary.status?.toLowerCase() === "cancelled" ? "cancelled" : "success";
  const statusCfg = STATUS_CONFIG[statusKey] || STATUS_CONFIG.success;

  // Don't render if no meaningful data
  if (!summary.totalDurationMs && !summary.toolCallCount && !totalTokens) {
    return null;
  }

  return (
    <div className={cn("rounded-lg border border-border/50 bg-background overflow-hidden", "shadow-sm", className)}>
      {/* Header */}
      <div className="px-4 py-2 border-b border-border/50 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FileCode className="w-4 h-4 text-muted-foreground" />
          <span className="font-medium text-sm">Session Summary</span>
        </div>
        <div className={cn("flex items-center gap-1.5 text-xs px-2 py-1 rounded-full", statusCfg.bg, statusCfg.color)}>
          <statusCfg.icon className="w-3.5 h-3.5" />
          {statusCfg.label}
        </div>
      </div>

      {/* Content */}
      <div className="p-4 space-y-4">
        {/* Timing Section */}
        {((summary.totalDurationMs && summary.totalDurationMs > 0) ||
          (summary.toolDurationMs && summary.toolDurationMs > 0) ||
          (summary.generationDurationMs && summary.generationDurationMs > 0)) && (
          <div>
            <div className="text-xs text-muted-foreground mb-2 flex items-center gap-1.5">
              <Clock className="w-3.5 h-3.5" />
              Timing
            </div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              {summary.totalDurationMs && summary.totalDurationMs > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Total</span>
                  <span className="font-mono">{formatDuration(summary.totalDurationMs)}</span>
                </div>
              )}
              {summary.toolDurationMs && summary.toolDurationMs > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Tools</span>
                  <span className="font-mono">{formatDuration(summary.toolDurationMs)}</span>
                </div>
              )}
              {summary.generationDurationMs && summary.generationDurationMs > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Generation</span>
                  <span className="font-mono">{formatDuration(summary.generationDurationMs)}</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Token Section */}
        {totalTokens > 0 && (
          <div>
            <div className="text-xs text-muted-foreground mb-2 flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5" />
              Tokens
            </div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              {summary.totalInputTokens && summary.totalInputTokens > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Input</span>
                  <span className="font-mono">{formatNumber(summary.totalInputTokens)}</span>
                </div>
              )}
              {summary.totalOutputTokens && summary.totalOutputTokens > 0 && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Output</span>
                  <span className="font-mono">{formatNumber(summary.totalOutputTokens)}</span>
                </div>
              )}
              <div className="flex justify-between col-span-2">
                <span className="text-muted-foreground">Total</span>
                <span className="font-mono">{formatNumber(totalTokens)}</span>
              </div>
              {((summary.totalCacheReadTokens && summary.totalCacheReadTokens > 0) ||
                (summary.totalCacheWriteTokens && summary.totalCacheWriteTokens > 0)) && (
                <div className="flex justify-between col-span-2">
                  <span className="text-muted-foreground">Cache</span>
                  <span className="font-mono text-xs">
                    R: {formatNumber(summary.totalCacheReadTokens || 0)} / W: {formatNumber(summary.totalCacheWriteTokens || 0)}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tools Section */}
        {((summary.toolCallCount && summary.toolCallCount > 0) || (summary.toolsUsed && summary.toolsUsed.length > 0)) && (
          <div>
            <div className="text-xs text-muted-foreground mb-2 flex items-center gap-1.5">
              <Wrench className="w-3.5 h-3.5" />
              Tools ({summary.toolCallCount || 0})
            </div>
            {summary.toolsUsed && summary.toolsUsed.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {summary.toolsUsed.map((tool, i) => (
                  <span key={i} className="px-2 py-1 rounded-md bg-muted text-xs font-mono">
                    {tool}
                  </span>
                ))}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground italic">No tools used</div>
            )}
          </div>
        )}

        {/* Files Section (Evolution Mode) */}
        {summary.filesModified && summary.filesModified > 0 && (
          <div>
            <div className="text-xs text-muted-foreground mb-2">Files Modified ({summary.filesModified})</div>
            {summary.filePaths && summary.filePaths.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {summary.filePaths.slice(0, 5).map((path, i) => (
                  <span key={i} className="px-2 py-1 rounded-md bg-muted text-xs truncate max-w-[200px]" title={path}>
                    {path.split("/").pop()}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
        )}
      </div>
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
