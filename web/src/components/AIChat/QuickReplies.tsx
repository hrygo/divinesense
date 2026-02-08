/**
 * QuickReplies Component
 *
 * Context-aware quick reply suggestions based on AI response analysis.
 * Mobile-friendly design with horizontal scroll on small screens.
 *
 * ## Features
 * - Auto-generated suggestions based on response type
 * - Mobile-optimized with horizontal scroll
 * - Click to fill input or navigate
 * - Smooth animations and hover effects
 *
 * ## Integration
 * Used in UnifiedMessageBlock after assistant response is complete.
 */

import {
  AlertTriangle,
  Calendar,
  CalendarDays,
  CalendarPlus,
  Clock,
  ExternalLink,
  FileText,
  type LucideIcon,
  Plus,
  RefreshCw,
  Search,
} from "lucide-react";
import { memo, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import type { QuickReplyAction, QuickReplyAnalysis } from "./utils/quickReplyAnalyzer";
import { hasQuickReplies } from "./utils/quickReplyAnalyzer";

// Icon mapping from string to component
const ICON_MAP: Record<string, LucideIcon> = {
  Calendar,
  CalendarDays,
  CalendarPlus,
  Clock,
  ExternalLink,
  FileText,
  Plus,
  RefreshCw,
  Search,
  AlertTriangle,
};

/**
 * Props for QuickReplies component
 */
export interface QuickRepliesProps {
  /** Analysis result from quickReplyAnalyzer */
  analysis: QuickReplyAnalysis;
  /** Callback when user clicks a quick reply */
  onAction: (action: QuickReplyAction) => void;
  /** Optional custom className */
  className?: string;
}

/**
 * QuickReplyButton - Individual quick reply button
 */
interface QuickReplyButtonProps {
  action: QuickReplyAction;
  onClick: (action: QuickReplyAction) => void;
  index: number;
}

const QuickReplyButton = memo(function QuickReplyButton({ action, onClick, index }: QuickReplyButtonProps) {
  const { t } = useTranslation();
  const IconComponent = ICON_MAP[action.icon] || Search;

  const handleClick = useCallback(() => {
    onClick(action);
  }, [action, onClick]);

  // Get display label (try i18n first, fallback to plain text)
  const label = action.label.startsWith("ai.") ? t(action.label) : action.label;

  // Get display hint (try i18n first, fallback to plain text or label)
  const hint = action.hint?.startsWith("ai.") ? t(action.hint) : action.hint || label;

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "group flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium",
        "transition-all duration-200",
        "bg-white dark:bg-zinc-800/50",
        "border border-zinc-200 dark:border-zinc-700",
        "hover:border-blue-300 dark:hover:border-blue-600",
        "hover:bg-blue-50 dark:hover:bg-blue-900/20",
        "hover:shadow-sm",
        "active:scale-95",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
      title={hint}
      style={{
        animationDelay: `${index * 50}ms`,
      }}
    >
      <IconComponent className="w-4 h-4 text-zinc-500 dark:text-zinc-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" />
      <span className="text-zinc-700 dark:text-zinc-300 group-hover:text-blue-700 dark:group-hover:text-blue-300 transition-colors">
        {label}
      </span>
    </button>
  );
});

/**
 * QuickReplies - Container for quick reply suggestions
 */
export const QuickReplies = memo(function QuickReplies({ analysis, onAction, className }: QuickRepliesProps) {
  const navigate = useNavigate();

  // Handle quick reply action
  const handleAction = useCallback(
    (action: QuickReplyAction) => {
      switch (action.type) {
        case "navigate":
          // Navigate to route
          navigate(action.payload);
          break;
        case "fill_input":
          // Fill input with payload (triggered via callback)
          onAction(action);
          break;
        case "trigger_tool":
          // Trigger a specific tool
          onAction(action);
          break;
        case "copy":
          // Copy to clipboard
          navigator.clipboard.writeText(action.payload);
          onAction(action);
          break;
      }
    },
    [navigate, onAction],
  );

  // Don't render if no valid actions
  if (!hasQuickReplies(analysis)) {
    return null;
  }

  // Limit to 4 actions max
  const actions = useMemo(() => analysis.actions.slice(0, 4), [analysis.actions]);

  return (
    <div className={cn("quick-replies", className)}>
      {/* Mobile: Horizontal scroll, Desktop: Wrapped */}
      <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-hide sm:overflow-visible sm:flex-wrap animate-in fade-in slide-in-from-bottom-2 duration-300">
        {actions.map((action, index) => (
          <QuickReplyButton key={action.id} action={action} onClick={handleAction} index={index} />
        ))}
      </div>
    </div>
  );
});

export default QuickReplies;
