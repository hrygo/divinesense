import { Calendar, Clock, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

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
  if (!isStreaming || events.length === 0) {
    return null;
  }

  // Get the most recent thinking event
  const lastThinking = events.filter((e) => e.type === "thinking").at(-1);

  // Get the most recent tool_use event
  const lastToolUse = events.filter((e) => e.type === "tool_use").at(-1);

  const formatToolName = (toolName: string): string => {
    switch (toolName) {
      case "schedule_add":
        return "Creating schedule...";
      case "schedule_query":
        return "Checking schedules...";
      case "schedule_update":
        return "Updating schedule...";
      case "find_free_time":
        return "Finding free time...";
      default:
        return `Using ${toolName}...`;
    }
  };

  return (
    <div className={cn("flex items-center gap-3 px-4 py-3 bg-muted/50 rounded-xl border border-border/50", className)}>
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
            <span className="text-muted-foreground">{lastThinking.data || "Thinking..."}</span>
          </div>
        ) : (
          <span className="text-sm text-muted-foreground">Processing...</span>
        )}
      </div>
    </div>
  );
}
