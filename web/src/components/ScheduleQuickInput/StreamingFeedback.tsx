import { CheckCircle, Clock, Loader2, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { PhaseProgress } from "./PhaseProgress";
import { SCHEDULE_PHASES, type StreamingEvent } from "./phaseConfig";

// Constants
const MAX_PHASE = SCHEDULE_PHASES.length - 1;
const PREVIEW_MAX_LENGTH = 60;

// Pre-compiled regex patterns
const TOOL_NAME_PLAIN_REGEX = /^(\w+)(?::|$)/;

function extractToolName(data: string): string {
  // Try JSON format first
  if (data.includes('"tool_name"') || data.includes('"name"')) {
    try {
      const parsed = JSON.parse(data);
      return parsed.tool_name || parsed.name || "";
    } catch {
      // Fall through to plain text parsing
    }
  }
  // Try plain text format
  const match = data.match(TOOL_NAME_PLAIN_REGEX);
  return match ? match[1] : "";
}

function getCurrentPhase(events: StreamingEvent[]): number {
  if (events.length === 0) return 0;

  let currentPhase = 0;

  for (const event of events) {
    if (event.type === "tool_use") {
      const toolName = extractToolName(event.data);
      if (toolName === "schedule_add") {
        currentPhase = 3;
      } else if (toolName === "schedule_query" || toolName === "find_free_time") {
        currentPhase = 2;
      } else {
        currentPhase = 2;
      }
    } else if (event.type === "task_start") {
      currentPhase = 1;
    } else if (event.type === "tool_result") {
      if (event.data.includes("Created:") || event.data.includes("已创建")) {
        currentPhase = 3;
      } else if (currentPhase < 2) {
        currentPhase = 2;
      }
    } else if (event.type === "answer") {
      if (currentPhase < 2) currentPhase = 2;
    }
  }

  return currentPhase;
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
      const toolName = extractToolName(event.data);
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
  const currentPhase = Math.min(getCurrentPhase(events), MAX_PHASE);

  const lastEvent = events.at(-1) || null;
  const statusText = getEventDescription(lastEvent, t, isStreaming);
  const previewText = lastEvent?.data && lastEvent.type !== "thinking" ? lastEvent.data.slice(0, PREVIEW_MAX_LENGTH) : null;

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
          {previewText && (
            <p className="text-xs text-muted-foreground truncate mt-0.5">
              {previewText}
              {lastEvent!.data.length > PREVIEW_MAX_LENGTH && "..."}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

export type { StreamingEvent };
