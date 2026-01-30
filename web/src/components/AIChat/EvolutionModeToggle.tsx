import { Dna, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

interface EvolutionModeToggleProps {
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
  disabled?: boolean;
  variant?: "header" | "toolbar" | "mobile";
}

/**
 * Evolution Mode Toggle Component
 *
 * Clear state indication:
 * - Not enabled: Shows "进化" + DNA icon (click to enter evolution mode)
 * - Enabled: Shows "退出进化" + X icon (click to exit evolution mode)
 *
 * Design:
 * - Purple/Blue gradient theme to differentiate from Geek Mode
 * - Admin-only access (disabled for non-admin users)
 * - Consistent across all variants
 */
export function EvolutionModeToggle({ enabled, onToggle, disabled = false, variant = "header" }: EvolutionModeToggleProps) {
  const { t } = useTranslation();

  const isToolbar = variant === "toolbar";
  const isHeader = variant === "header";

  // Label based on state
  const label = enabled ? t("ai.evolution_mode.disable_label") : t("ai.evolution_mode.label");

  return (
    <button
      onClick={() => onToggle(!enabled)}
      disabled={disabled}
      className={cn(
        "relative flex items-center justify-center transition-all",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        // Toolbar variant - compact, always shows label
        isToolbar && [
          "h-7 px-2 text-xs",
          !enabled
            ? "text-muted-foreground hover:text-foreground hover:bg-muted"
            : "text-purple-600 dark:text-purple-400 hover:text-purple-700 dark:hover:text-purple-300 hover:bg-purple-500/10",
        ],
        // Header variant - shows label on larger screens
        isHeader && [
          "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm font-medium",
          "hover:bg-muted",
          !enabled ? "text-muted-foreground" : "text-purple-600 dark:text-purple-400",
        ],
      )}
      title={enabled ? t("ai.evolution_mode.enabled") : t("ai.evolution_mode.tooltip")}
      aria-pressed={enabled}
    >
      {/* Icon - DNA for entering, X for exiting */}
      <span className="shrink-0">
        {enabled ? (
          <X className={cn("w-4 h-4", isToolbar && "w-3.5 h-3.5")} />
        ) : (
          <Dna className={cn("w-4 h-4", isToolbar && "w-3.5 h-3.5")} />
        )}
      </span>

      {/* Label */}
      {isToolbar || isHeader ? (
        <span className={cn(isHeader && "hidden sm:inline", isToolbar && "ml-1 whitespace-nowrap")}>{label}</span>
      ) : null}
    </button>
  );
}
