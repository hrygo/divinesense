import { Check, Copy, Terminal } from "lucide-react";
import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface TerminalOutputProps {
  output: string;
  className?: string;
  language?: string;
  command?: string;
  exitCode?: number;
}

/**
 * TerminalOutput - Displays command output in a terminal-style container
 * TerminalOutput - 以终端风格展示命令输出
 */
export const TerminalOutput = memo(function TerminalOutput({
  output,
  className,
  language = "bash",
  command,
  exitCode,
}: TerminalOutputProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(output);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        "relative group rounded-lg overflow-hidden",
        "bg-slate-950 dark:bg-slate-900",
        "border border-slate-800 dark:border-slate-700",
        className,
      )}
    >
      {/* Terminal Header */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-slate-900 dark:bg-black border-b border-slate-800 dark:border-slate-700">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-slate-400" />
          <span className="text-xs text-slate-400 font-mono">{language}</span>
          {command && <span className="text-xs text-slate-500 font-mono truncate max-w-[200px]">{command}</span>}
        </div>
        <button
          onClick={handleCopy}
          className={cn(
            "flex items-center gap-1 px-2 py-0.5 rounded text-xs",
            "text-slate-400 hover:text-slate-200",
            "hover:bg-slate-800 transition-colors",
          )}
          type="button"
          aria-label={t("common.copy")}
        >
          {copied ? <Check className="w-3.5 h-3.5 text-green-500" /> : <Copy className="w-3.5 h-3.5" />}
          <span className="sr-only">{copied ? t("common.copied") : t("common.copy")}</span>
          {copied ? t("common.copied") : t("common.copy")}
        </button>
      </div>

      {/* Terminal Content */}
      <pre className="p-3 text-xs text-slate-300 font-mono overflow-x-auto whitespace-pre-wrap break-all">{output}</pre>

      {/* Exit Code Indicator */}
      {exitCode !== undefined && (
        <div
          className={cn(
            "px-3 py-1 text-xs font-mono border-t border-slate-800 dark:border-slate-700",
            exitCode === 0 ? "text-green-500" : "text-red-500",
          )}
        >
          {t("ai.events.exit_code")} {exitCode}
        </div>
      )}
    </div>
  );
});

/**
 * CompactTerminal - Smaller terminal variant for inline display
 * CompactTerminal - 紧凑型终端组件，用于内联展示
 */
interface CompactTerminalProps {
  output: string;
  maxLines?: number;
  className?: string;
}

export function CompactTerminal({ output, maxLines = 3, className }: CompactTerminalProps) {
  const lines = output.split("\n");
  const displayLines = maxLines > 0 ? lines.slice(0, maxLines) : lines;
  const isTruncated = lines.length > maxLines && maxLines > 0;

  return (
    <div className={cn("rounded-md bg-slate-950 dark:bg-slate-900 border border-slate-800 dark:border-slate-700 p-2", className)}>
      <code className="text-xs text-slate-300 font-mono whitespace-pre-wrap break-all">
        {displayLines.join("\n")}
        {isTruncated && <span className="text-slate-500">...</span>}
      </code>
    </div>
  );
}
