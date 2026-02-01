import { ChevronDown, ChevronRight, FileIcon, Play, Terminal as TerminalIcon } from "lucide-react";
import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ToolResultBadge } from "./EventBadge";
import { CompactTerminal, TerminalOutput } from "./TerminalOutput";

/**
 * Tool call metadata from StreamEvent
 * StreamEvent 中的工具调用元数据
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
function getToolIcon(toolName: string) {
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
}

/**
 * Format input for display
 */
function formatInput(input?: Record<string, unknown>): string | undefined {
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
}

/**
 * ToolCallCard - Displays a tool call with its input, output, and status
 * ToolCallCard - 展示工具调用及其输入、输出和状态
 */
export const ToolCallCard = memo(function ToolCallCard({ data, className }: ToolCallCardProps) {
  const { t } = useTranslation();
  const [isExpanded, setIsExpanded] = useState(false);
  const Icon = getToolIcon(data.toolName);

  const hasOutput = data.output !== undefined && data.output !== "";
  const hasInput = data.input !== undefined;
  const durationText = data.duration ? `${data.duration}ms` : undefined;

  return (
    <div
      className={cn(
        "rounded-lg border overflow-hidden",
        "bg-slate-50 dark:bg-slate-900/50",
        "border-slate-200 dark:border-slate-700",
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
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div className={cn("p-1.5 rounded", "bg-blue-100 dark:bg-blue-900/30", "text-blue-600 dark:text-blue-400")}>
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
              onClick={(e) => {
                e.stopPropagation();
                setIsExpanded(!isExpanded);
              }}
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
});

/**
 * InlineToolCall - Compact inline tool call indicator
 * InlineToolCall - 紧凑型内联工具调用指示器
 */
interface InlineToolCallProps {
  toolName: string;
  isError?: boolean;
  className?: string;
  inputSummary?: string;
  filePath?: string;
}

export function InlineToolCall({ toolName, isError, className, inputSummary, filePath }: InlineToolCallProps) {
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
}
