/**
 * ProgressIndicator Component
 *
 * Displays progressive progress feedback for AI chat operations.
 * Shows the current phase of processing and estimated remaining time.
 *
 * Phases:
 * - analyzing: Understanding the user request
 * - planning: Determining the approach
 * - retrieving: Searching for relevant data
 * - synthesizing: Generating the response
 *
 * Issue #97: Progressive Progress Feedback
 */

import { Brain, FileText, Loader2, Search } from "lucide-react";
import { memo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

/**
 * Processing phases in order
 */
export type ProcessingPhase = "analyzing" | "planning" | "retrieving" | "synthesizing";

/**
 * Phase configuration with icon and translation key
 */
const PHASE_CONFIG: Record<ProcessingPhase, { icon: typeof Loader2; i18nKey: string; color: string }> = {
  analyzing: {
    icon: Brain,
    i18nKey: "ai.states.analyzing",
    color: "text-blue-500",
  },
  planning: {
    icon: Loader2,
    i18nKey: "ai.states.planning",
    color: "text-purple-500",
  },
  retrieving: {
    icon: Search,
    i18nKey: "ai.states.retrieving",
    color: "text-orange-500",
  },
  synthesizing: {
    icon: FileText,
    i18nKey: "ai.states.synthesizing",
    color: "text-green-500",
  },
};

/**
 * Order of phases for progress bar
 */
const PHASE_ORDER: ProcessingPhase[] = ["analyzing", "planning", "retrieving", "synthesizing"];

export interface ProgressIndicatorProps {
  /** Current processing phase */
  phase?: ProcessingPhase;
  /** Progress percentage (0-100) */
  progress?: number;
  /** Estimated remaining time in seconds */
  estimatedTimeRemaining?: number;
  /** Optional additional CSS classes */
  className?: string;
}

/**
 * Format time remaining in human-readable format
 */
function formatTimeRemaining(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
}

/**
 * ProgressIndicator component
 *
 * Displays:
 * - Current phase with animated icon
 * - Progress bar with phase markers
 * - Estimated time remaining
 */
export const ProgressIndicator = memo(function ProgressIndicator({
  phase,
  progress = 0,
  estimatedTimeRemaining,
  className,
}: ProgressIndicatorProps) {
  const { t } = useTranslation();

  // If no phase specified, don't render
  if (!phase) {
    return null;
  }

  const config = PHASE_CONFIG[phase];
  const Icon = config.icon;
  const phaseIndex = PHASE_ORDER.indexOf(phase);
  const phaseProgress = ((phaseIndex + progress / 100) / PHASE_ORDER.length) * 100;

  return (
    <div className={cn("flex flex-col gap-3 py-2", className)}>
      {/* Phase indicator with icon and text */}
      <div className="flex items-center gap-2 text-sm">
        <Icon className={cn("w-4 h-4", config.color, phase === "planning" && "animate-spin")} />
        <span className={cn(config.color, "font-medium")}>{t(config.i18nKey)}</span>
        {estimatedTimeRemaining !== undefined && (
          <span className="text-xs text-muted-foreground ml-auto">
            {t("ai.progress.timeRemaining", { time: formatTimeRemaining(estimatedTimeRemaining) })}
          </span>
        )}
      </div>

      {/* Progress bar with phase markers */}
      <div className="relative h-1.5 bg-muted rounded-full overflow-hidden">
        {/* Background track with phase markers */}
        <div className="absolute inset-0 flex">
          {PHASE_ORDER.map((p, index) => (
            <div
              key={p}
              className="flex-1 border-r border-background/20 last:border-r-0"
              style={{
                backgroundColor: index < phaseIndex || (index === phaseIndex && progress > 0) ? "currentColor" : undefined,
              }}
            />
          ))}
        </div>

        {/* Animated progress overlay */}
        <div
          className={cn("absolute top-0 left-0 h-full transition-all duration-300 ease-out", config.color.replace("text-", "bg-"))}
          style={{ width: `${phaseProgress}%` }}
        />

        {/* Shimmer effect */}
        <div
          className={cn(
            "absolute top-0 left-0 h-full w-1/3",
            "bg-gradient-to-r from-transparent via-white/20 to-transparent",
            "animate-[shimmer_1.5s_infinite]",
          )}
          style={{
            animationDelay: `${phaseIndex * 0.1}s`,
          }}
        />
      </div>

      {/* Phase labels */}
      <div className="flex justify-between text-xs text-muted-foreground px-1">
        {PHASE_ORDER.map((p) => (
          <span key={p} className={cn("transition-colors", PHASE_ORDER.indexOf(p) <= phaseIndex && config.color)}>
            {t(`ai.progress.phases.${p}`)}
          </span>
        ))}
      </div>
    </div>
  );
});

ProgressIndicator.displayName = "ProgressIndicator";

export default ProgressIndicator;
