import { CheckCircle, Clock, Loader2, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { SCHEDULE_PHASES, type StreamingEvent } from "./phaseConfig";

interface PhaseProgressProps {
  currentPhase: number;
  isComplete?: boolean;
  hasError?: boolean;
  className?: string;
}

export function PhaseProgress({ currentPhase, isComplete, hasError, className }: PhaseProgressProps) {
  const t = useTranslate();
  const phases = SCHEDULE_PHASES;

  return (
    <div className={cn("flex items-start justify-between w-full gap-1", className)}>
      {phases.map((phase, index) => {
        const isCompleted = index < currentPhase || isComplete;
        const isCurrent = index === currentPhase && !isComplete && !hasError;
        const isPending = index > currentPhase && !isComplete;

        return (
          <div key={phase.key} className="flex flex-col items-center flex-1">
            <div
              className={cn(
                "w-10 h-10 rounded-full flex items-center justify-center transition-all duration-500 shadow-sm",
                isCompleted && "bg-green-500 text-white ring-2 ring-green-500/30",
                isCurrent && !hasError && "bg-primary text-primary-foreground ring-2 ring-primary/30 animate-pulse",
                isPending && "bg-muted text-muted-foreground ring-2 ring-muted",
                hasError && index <= currentPhase && "bg-destructive text-destructive-foreground ring-2 ring-destructive/30",
              )}
            >
              {isCompleted ? (
                <CheckCircle className="w-5 h-5" />
              ) : isCurrent ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : hasError && index <= currentPhase ? (
                <XCircle className="w-5 h-5" />
              ) : (
                <div className={cn("w-2.5 h-2.5 rounded-full bg-current", isCurrent && "animate-ping")} />
              )}
            </div>
            <span
              className={cn(
                "text-xs mt-1.5 font-medium text-center",
                isCompleted && "text-green-600 dark:text-green-400",
                isCurrent && !hasError && "text-primary",
                isPending && "text-muted-foreground/60",
                hasError && index <= currentPhase && "text-destructive",
              )}
            >
              {t(phase.labelKey as "schedule.phase.understand")}
            </span>
          </div>
        );
      })}
    </div>
  );
}

interface StreamingFeedbackProps {
  events: StreamingEvent[];
  isStreaming: boolean;
  className?: string;
}

export function StreamingFeedback({ events, isStreaming, className }: StreamingFeedbackProps) {
  const t = useTranslate();

  if (!isStreaming && events.length === 0) {
    return null;
  }

  const hasError = events.some((e) => e.type === "error");
  const isComplete = !isStreaming && events.length > 0 && !hasError;
  const currentPhase = Math.min(getCurrentPhase(events), 3);

  const lastEvent = events.at(-1) || null;
  const statusText = getEventDescription(lastEvent, t, isStreaming);

  return (
    <div
      className={cn(
        "flex flex-col gap-3 px-4 py-4 bg-gradient-to-r from-muted/30 to-muted/60 rounded-xl border border-border/60 shadow-sm",
        className,
      )}
    >
      <PhaseProgress currentPhase={currentPhase} isComplete={isComplete} hasError={hasError} />

      <div className="flex items-center gap-3 pt-1 border-t border-border/30">
        <div
          className={cn(
            "flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center",
            isStreaming && "bg-primary/10",
            isComplete && !hasError && "bg-green-500/10",
            hasError && "bg-destructive/10",
          )}
        >
          {isStreaming ? (
            <Loader2 className="w-5 h-5 animate-spin text-primary" />
          ) : hasError ? (
            <XCircle className="w-5 h-5 text-destructive" />
          ) : isComplete ? (
            <CheckCircle className="w-5 h-5 text-green-500" />
          ) : (
            <Clock className="w-5 h-5 text-muted-foreground" />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <p className={cn("text-sm font-medium truncate", hasError ? "text-destructive" : "text-foreground")}>{statusText}</p>
          {lastEvent?.data && lastEvent.type !== "thinking" && (
            <p className="text-xs text-muted-foreground truncate mt-0.5">
              {lastEvent.data.substring(0, 60)}
              {lastEvent.data.length > 60 ? "..." : ""}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

function getEventDescription(event: StreamingEvent | null, t: ReturnType<typeof useTranslate>, isStreaming: boolean): string {
  if (!event) return t("schedule.ai.thinking");

  switch (event.type) {
    case "thinking":
    case "plan":
      return event.data || t("schedule.ai.thinking");
    case "task_start":
      return t("schedule.ai.parsing");
    case "tool_use": {
      const toolMatch = event.data.match(/^(\w+)(?::|$)/);
      const toolName = toolMatch ? toolMatch[1] : "";
      switch (toolName) {
        case "schedule_query":
          return t("schedule.ai.checking-schedule");
        case "schedule_add":
          return t("schedule.ai.creating-schedule");
        case "schedule_update":
          return t("schedule.ai.updating-schedule");
        case "find_free_time":
          return t("schedule.ai.finding-free-time");
        default:
          return t("schedule.ai.using-tool");
      }
    }
    case "tool_result":
      return t("schedule.ai.processing-result");
    case "answer":
      return isStreaming ? t("schedule.ai.generating") : t("schedule.ai.completed");
    case "error":
      return event.data || t("schedule.ai.error");
    default:
      return "";
  }
}

function getCurrentPhase(events: StreamingEvent[]): number {
  if (events.length === 0) return 0;

  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];

    if (event.type === "tool_use") {
      const toolMatch = event.data.match(/^(\w+)(?::|$)/);
      const toolName = toolMatch ? toolMatch[1] : "";
      if (toolName === "schedule_add") {
        return 3;
      }
    }

    if (event.type === "tool_use") {
      const toolMatch = event.data.match(/^(\w+)(?::|$)/);
      const toolName = toolMatch ? toolMatch[1] : "";
      if (toolName === "schedule_query" || toolName === "find_free_time") {
        return 2;
      }
    }

    if (event.type === "task_start") {
      return 1;
    }

    if (event.type === "plan" || event.type === "thinking") {
      return 0;
    }
  }

  return 0;
}
