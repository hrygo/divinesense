import { Calendar, Clock, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

interface StreamingEvent {
  type: string;
  data: string;
  timestamp: number;
}

interface StreamingFeedbackProps {
  events: StreamingEvent[];
  isStreaming: boolean;
  className?: string;
}

/**
 * Simplified StreamingFeedback component for schedule creation
 * Shows real-time AI thinking and tool use feedback
 */
export function StreamingFeedback({ events, isStreaming, className }: StreamingFeedbackProps) {
  const t = useTranslate();

  if (!isStreaming || events.length === 0) {
    return null;
  }

  // Get the most recent thinking event
  const lastThinking = events.filter((e) => e.type === "thinking").at(-1);

  // Get the most recent tool_use event
  const lastToolUse = events.filter((e) => e.type === "tool_use").at(-1);

  const formatToolName = (toolName: string): string => {
    const usingToolText = (t("schedule.ai.using-tool") as string) || "Using tool";
    switch (toolName) {
      case "schedule_add":
        return (t("schedule.ai.creating-schedule") as string) || "Creating schedule...";
      case "schedule_query":
        return (t("schedule.ai.checking-schedule") as string) || "Checking schedules...";
      case "schedule_update":
        return (t("schedule.ai.updating-schedule") as string) || "Updating schedule...";
      case "find_free_time":
        return (t("schedule.ai.finding-free-time") as string) || "Finding free time...";
      default:
        return `${usingToolText}...`;
    }
  };

  const thinkingText = (t("schedule.ai.thinking") as string) || "Thinking...";
  const processingText = (t("schedule.ai.processing") as string) || "Processing...";

  return (
    <div className={cn("flex items-center gap-3 px-4 py-3 bg-muted/50 rounded-lg border border-border/50", className)}>
      <Loader2 className="h-5 w-5 animate-spin text-primary" />
      <div className="flex-1">
        {lastToolUse ? (
          <div className="flex items-center gap-2 text-sm">
            <Calendar className="h-4 w-4 text-primary" />
            <span className="text-muted-foreground">{formatToolName(lastToolUse.data.split(":")[0] || lastToolUse.data)}</span>
          </div>
        ) : lastThinking ? (
          <div className="flex items-center gap-2 text-sm">
            <Clock className="h-4 w-4 text-primary" />
            <span className="text-muted-foreground">{lastThinking.data || thinkingText}</span>
          </div>
        ) : (
          <span className="text-sm text-muted-foreground">{processingText}</span>
        )}
      </div>
    </div>
  );
}
