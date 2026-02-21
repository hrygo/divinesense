import { Calendar, Clock, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { type Translations, useTranslate } from "@/utils/i18n";
import { PhaseProgress } from "./PhaseProgress";
import { getCurrentPhase, getStatusTextKey } from "./phaseConfig";

export interface StreamingEvent {
  type: string;
  data: string;
  timestamp: number;
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
  const currentPhase = getCurrentPhase(events);

  const lastEvent = events.at(-1);
  const statusTextKey = getStatusTextKey(lastEvent || null);
  const statusText = statusTextKey ? (t(statusTextKey as Translations) as string) || lastEvent?.data || "" : lastEvent?.data || "";

  return (
    <div className={cn("flex flex-col gap-3 px-4 py-3 bg-muted/50 rounded-lg border border-border/50", className)}>
      <PhaseProgress currentPhase={currentPhase} isComplete={isComplete} hasError={hasError} />

      <div className="flex items-center gap-3">
        {isStreaming ? (
          <Loader2 className="h-5 w-5 animate-spin text-primary flex-shrink-0" />
        ) : hasError ? (
          <Clock className="h-5 w-5 text-destructive flex-shrink-0" />
        ) : (
          <Calendar className="h-5 w-5 text-green-500 flex-shrink-0" />
        )}
        <span className={cn("text-sm truncate", hasError ? "text-destructive" : "text-muted-foreground")}>{statusText}</span>
      </div>
    </div>
  );
}
