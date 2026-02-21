export interface StreamingEvent {
  type: string;
  data: string;
  timestamp: number;
}

export const SCHEDULE_PHASES = [
  { key: "understand", labelKey: "schedule.phase.understand", eventTypes: ["plan", "thinking"] },
  { key: "parse", labelKey: "schedule.phase.parse", eventTypes: ["task_start"] },
  { key: "check", labelKey: "schedule.phase.check", eventTypes: ["tool_use"] },
  { key: "create", labelKey: "schedule.phase.create", eventTypes: ["tool_result", "answer"] },
] as const;

export type SchedulePhaseKey = (typeof SCHEDULE_PHASES)[number]["key"];

export function getCurrentPhase(events: StreamingEvent[]): number {
  if (events.length === 0) return 0;

  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];

    if (event.type === "tool_result" || event.type === "answer") {
      if (event.data.includes("schedule") || event.data.includes("日程") || event.data.includes("Schedule")) {
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

export function getStatusTextKey(event: StreamingEvent | null): string {
  if (!event) return "schedule.ai.thinking";

  switch (event.type) {
    case "plan":
    case "thinking":
      return event.data ? "" : "schedule.ai.understanding";
    case "task_start":
      return "schedule.ai.parsing";
    case "tool_use": {
      const toolMatch = event.data.match(/^(\w+)(?::|$)/);
      const toolName = toolMatch ? toolMatch[1] : "";
      switch (toolName) {
        case "schedule_query":
          return "schedule.ai.checking-schedule";
        case "schedule_add":
          return "schedule.ai.creating-schedule";
        case "schedule_update":
          return "schedule.ai.updating-schedule";
        case "find_free_time":
          return "schedule.ai.finding-free-time";
        default:
          return "schedule.ai.using-tool";
      }
    }
    case "tool_result":
      return "schedule.ai.processing-result";
    case "answer":
      return "schedule.ai.generating";
    case "error":
      return "schedule.ai.error";
    default:
      return "";
  }
}
