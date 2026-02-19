/**
 * AgentBadge - 已选专家标签组件
 *
 * 显示用户通过 @ 选择的专家，可点击移除
 *
 * @see Issue #266
 */
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { ParrotInfoFromAPI } from "@/hooks/useParrotsList";
import { cn } from "@/lib/utils";

// Agent 主题色映射
const AGENT_STYLES: Record<string, { bg: string; text: string; border: string }> = {
  memo: {
    bg: "bg-slate-100 dark:bg-slate-800",
    text: "text-slate-700 dark:text-slate-200",
    border: "border-slate-300 dark:border-slate-600",
  },
  schedule: {
    bg: "bg-cyan-100 dark:bg-cyan-900/50",
    text: "text-cyan-700 dark:text-cyan-200",
    border: "border-cyan-300 dark:border-cyan-700",
  },
  general: {
    bg: "bg-amber-100 dark:bg-amber-900/50",
    text: "text-amber-700 dark:text-amber-200",
    border: "border-amber-300 dark:border-amber-700",
  },
  ideation: {
    bg: "bg-violet-100 dark:bg-violet-900/50",
    text: "text-violet-700 dark:text-violet-200",
    border: "border-violet-300 dark:border-violet-700",
  },
};

const DEFAULT_STYLE = {
  bg: "bg-zinc-100 dark:bg-zinc-800",
  text: "text-zinc-700 dark:text-zinc-200",
  border: "border-zinc-300 dark:border-zinc-600",
};

// Agent emoji 映射
const AGENT_EMOJI: Record<string, string> = {
  memo: "🪶",
  schedule: "⏰",
  general: "🤖",
  ideation: "💡",
};

interface AgentBadgeProps {
  agent: ParrotInfoFromAPI;
  onRemove: () => void;
  className?: string;
}

export function AgentBadge({ agent, onRemove, className }: AgentBadgeProps) {
  const { t } = useTranslation();
  const style = AGENT_STYLES[agent.name] || DEFAULT_STYLE;
  const emoji = AGENT_EMOJI[agent.name] || "🦜";
  const displayName = agent.displayName || agent.name;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2 py-1 rounded-full text-xs font-medium border",
        "transition-all duration-200",
        style.bg,
        style.text,
        style.border,
        className,
      )}
    >
      <span className="text-sm">{emoji}</span>
      <span>{displayName}</span>
      <button
        type="button"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onRemove();
        }}
        className={cn("ml-0.5 p-0.5 rounded-full hover:bg-black/10 dark:hover:bg-white/10", "transition-colors duration-150")}
        aria-label={t("ai.mention.remove_agent")}
      >
        <X className="w-3 h-3" />
      </button>
    </span>
  );
}

export default AgentBadge;
