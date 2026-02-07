/**
 * Optimized ToolCallCard Component
 *
 * Enhanced version with:
 * - Phase 3: Hover border highlight effect
 * - Phase 5: React.memo optimization
 *
 * This is a drop-in replacement for the original ToolCallCard.
 */

import { ChevronDown, ChevronRight, FileIcon, Play, Terminal as TerminalIcon } from "lucide-react";
import { memo, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ToolResultBadge } from "../../EventBadge";
import { CompactTerminal, TerminalOutput } from "../../TerminalOutput";

/**
 * Tool call metadata from StreamEvent
 */
export interface ToolCallData {
  toolName: string;
  toolId?: string;
  input?: Record<string, unknown>;
  output?: string;
  exitCode?: number;
  isError?: boolean;
  duration?: number; // in milliseconds
}

interface ToolCallCardProps {
  data: ToolCallData;
  className?: string;
}

/**
 * Get icon for tool type
 */
const getToolIcon = (toolName: string) => {
  const name = toolName.toLowerCase();
  if (name.includes("write") || name.includes("edit") || name.includes("file")) {
    return FileIcon;
  }
  if (name.includes("run") || name.includes("exec") || name.includes("bash")) {
    return TerminalIcon;
  }
  if (name.includes("read")) {
    return FileIcon;
  }
  return Play;
};

/**
 * Format input for display
 */
const formatInput = (input?: Record<string, unknown>): string | undefined => {
  if (!input) return undefined;

  // Handle command input
  if (input.command && typeof input.command === "string") {
    return `$ ${input.command}`;
  }

  // Handle file path input
  if (input.file_path && typeof input.file_path === "string") {
    return input.file_path;
  }

  // Handle text input
  if (input.text && typeof input.text === "string") {
    return input.text;
  }

  return JSON.stringify(input, null, 2);
};

/**
 * Custom comparison for ToolCallCard memo
 * Only re-render if data actually changes
 */
const areToolCallPropsEqual = (prev: ToolCallCardProps, next: ToolCallCardProps): boolean => {
  // Fast path: same reference
  if (prev.data === next.data) return true;

  // Compare key properties
  return (
    prev.data.toolName === next.data.toolName &&
    prev.data.toolId === next.data.toolId &&
    prev.data.isError === next.data.isError &&
    prev.data.duration === next.data.duration &&
    prev.data.output === next.data.output &&
    JSON.stringify(prev.data.input) === JSON.stringify(next.data.input)
  );
};

/**
 * ToolCallCard - Displays a tool call with its input, output, and status
 *
 * Phase 3 Enhancement: Hover border highlight effect (purple-500)
 * Phase 5 Optimization: React.memo with custom comparison
 */
export const ToolCallCard = memo(function ToolCallCard({ data, className }: ToolCallCardProps) {
  const { t } = useTranslation();
  const [isExpanded, setIsExpanded] = useState(false);
  const Icon = getToolIcon(data.toolName);

  const hasOutput = data.output !== undefined && data.output !== "";
  const hasInput = data.input !== undefined;
  const durationText = data.duration ? `${data.duration}ms` : undefined;

  // Memoize toggle handler
  const handleToggle = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  // Memoize button click handler
  const handleButtonClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    setIsExpanded((prev) => !prev);
  }, []);

  return (
    <div
      className={cn(
        // Base styles
        "rounded-lg border overflow-hidden",
        "bg-slate-50 dark:bg-slate-900/50",
        "border-slate-200 dark:border-slate-700",
        // Phase 3: Hover border highlight effect
        "group",
        "hover:border-purple-400/50 dark:hover:border-purple-500/50",
        "hover:shadow-[0_0_0_1px_rgba(168,85,247,0.1)]",
        "transition-all duration-200",
        className,
      )}
    >
      {/* Header */}
      <div
        className={cn(
          "flex items-center justify-between px-3 py-2",
          "border-b border-slate-200 dark:border-slate-700",
          "cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800/50",
          "transition-colors",
        )}
        onClick={handleToggle}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div
            className={cn(
              "p-1.5 rounded transition-colors",
              "bg-blue-100 dark:bg-blue-900/30",
              "text-blue-600 dark:text-blue-400",
              // Phase 3: Icon background also highlights on group hover
              "group-hover:bg-purple-100 dark:group-hover:bg-purple-900/30",
              "group-hover:text-purple-600 dark:group-hover:text-purple-400",
            )}
          >
            <Icon className="w-3.5 h-3.5" />
          </div>
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300 truncate">{data.toolName}</span>
          {data.toolId && <span className="text-xs text-slate-400 truncate">#{data.toolId.slice(0, 8)}</span>}
        </div>
        <div className="flex items-center gap-2">
          {durationText && <span className="text-xs text-slate-400">{durationText}</span>}
          {hasOutput && data.isError === false && <ToolResultBadge isError={false} />}
          {hasOutput && data.isError === true && <ToolResultBadge isError={true} />}
          {hasOutput && (
            <button
              type="button"
              className="p-1 hover:bg-slate-200 dark:hover:bg-slate-700 rounded transition-colors"
              onClick={handleButtonClick}
              aria-label={isExpanded ? "Collapse" : "Expand"}
            >
              {isExpanded ? <ChevronDown className="w-4 h-4 text-slate-500" /> : <ChevronRight className="w-4 h-4 text-slate-500" />}
            </button>
          )}
        </div>
      </div>

      {/* Collapsible Content */}
      {isExpanded && (
        <div className="p-3 space-y-3">
          {/* Input */}
          {hasInput && (
            <div>
              <div className="text-xs text-slate-500 mb-1.5">{t("ai.events.input") || "Input"}</div>
              <CompactTerminal output={formatInput(data.input) || ""} maxLines={0} />
            </div>
          )}

          {/* Output */}
          {hasOutput && (
            <div>
              <div className="text-xs text-slate-500 mb-1.5">{t("ai.events.output") || "Output"}</div>
              {typeof data.output === "string" && data.output.length > 500 ? (
                <TerminalOutput output={data.output} command={formatInput(data.input)} exitCode={data.exitCode} />
              ) : (
                <CompactTerminal output={data.output || ""} maxLines={0} />
              )}
            </div>
          )}

          {/* No output */}
          {!hasOutput && <div className="text-sm text-slate-400 italic">{t("ai.events.no_output") || "No output yet..."}</div>}
        </div>
      )}
    </div>
  );
}, areToolCallPropsEqual);

/**
 * InlineToolCall - Compact inline tool call indicator
 */
interface InlineToolCallProps {
  toolName: string;
  isError?: boolean;
  className?: string;
  inputSummary?: string;
  filePath?: string;
}

export const InlineToolCall = memo(function InlineToolCall({ toolName, isError, className, inputSummary, filePath }: InlineToolCallProps) {
  const Icon = getToolIcon(toolName);

  // Construct display text: "ToolName" or "ToolName: Summary"
  let displayText = toolName;
  const detail = inputSummary || filePath;
  if (detail) {
    displayText = `${toolName} ${detail}`;
  }

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs max-w-[240px] md:max-w-[360px]",
        "bg-slate-100 dark:bg-slate-800",
        "border border-slate-200 dark:border-slate-700",
        "transition-all duration-200",
        // Phase 3: Hover effect
        "hover:border-purple-400/50 dark:hover:border-purple-500/50",
        isError ? "border-red-300 dark:border-red-700" : "border-blue-200 dark:border-blue-700",
        className,
      )}
      title={detail ? `${toolName}: ${detail}` : toolName}
    >
      <Icon className={cn("w-3.5 h-3.5 shrink-0", isError ? "text-red-500" : "text-blue-500")} />
      <span className={cn("font-medium truncate", isError ? "text-red-600 dark:text-red-400" : "text-slate-700 dark:text-slate-300")}>
        {displayText}
      </span>
    </div>
  );
});
