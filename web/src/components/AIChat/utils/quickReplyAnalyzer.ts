/**
 * Quick Reply Analyzer
 *
 * Analyzes AI responses and tool calls to generate context-aware quick reply suggestions.
 * Used by QuickReplies component to provide smart follow-up actions.
 *
 * Response Types:
 * - ScheduleCreated: User created a schedule successfully
 * - MemoFound: Search returned memo results
 * - ScheduleQuery: User queried their schedule
 * - FreeTimeFound: User found available time slots
 * - Error: Something went wrong
 * - Generic: Default fallback
 */

import type { ToolCall } from "./eventTransformers";

/**
 * Quick reply action that user can take
 */
export interface QuickReplyAction {
  /** Unique identifier for this action */
  id: string;
  /** Display text (i18n key or plain text) */
  label: string;
  /** Icon name from lucide-react */
  icon: string;
  /** Action type determines behavior */
  type: "fill_input" | "navigate" | "trigger_tool" | "copy";
  /** Action payload - depends on type */
  payload: string;
  /** Optional hint text */
  hint?: string;
}

/**
 * Analysis result containing suggested quick replies
 */
export interface QuickReplyAnalysis {
  /** Detected response type */
  responseType: "ScheduleCreated" | "MemoFound" | "ScheduleQuery" | "FreeTimeFound" | "Error" | "Generic";
  /** Suggested actions (max 4) */
  actions: QuickReplyAction[];
  /** Confidence score (0-1) */
  confidence: number;
}

/**
 * Tool result data structure
 */
// interface ToolResult {
//   name: string;
//   outputSummary?: string;
//   isError?: boolean;
//   duration?: number;
// }

/**
 * Analyzes tool calls and response content to determine response type
 */
export function analyzeQuickReplies(toolCalls: ToolCall[], responseContent: string, hasError: boolean): QuickReplyAnalysis {
  // Error state takes priority
  if (hasError) {
    return {
      responseType: "Error",
      actions: getErrorActions(),
      confidence: 0.9,
    };
  }

  // Get all tool names for easier matching
  const toolNames = toolCalls.map((t) => (typeof t === "string" ? t : t.name));

  // Analyze based on tool calls and response content
  const analysis = detectResponseType(toolNames, responseContent, toolCalls);

  return analysis;
}

/**
 * Detects response type based on tool calls and content
 */
function detectResponseType(toolNames: string[], content: string, toolCalls: ToolCall[]): QuickReplyAnalysis {
  const lowerContent = content.toLowerCase();

  // Check for schedule_add success (Chinese and English patterns)
  const hasScheduleAdd = toolNames.some((n) => n.toLowerCase().includes("schedule_add"));
  const scheduleCreatedPatterns = ["已创建", "created", "successfully created", "✓", "✅", "安排好了", "scheduled"];
  const isScheduleCreated = hasScheduleAdd && scheduleCreatedPatterns.some((p) => lowerContent.includes(p.toLowerCase()));

  if (isScheduleCreated) {
    return {
      responseType: "ScheduleCreated",
      actions: getScheduleCreatedActions(content),
      confidence: 0.85,
    };
  }

  // Check for schedule_query results
  const hasScheduleQuery = toolNames.some((n) => n.toLowerCase().includes("schedule_query"));
  const scheduleQueryPatterns = ["found", "schedule", "日程", "找到", "no schedules"];
  const isScheduleQuery = hasScheduleQuery && scheduleQueryPatterns.some((p) => lowerContent.includes(p.toLowerCase()));

  if (isScheduleQuery) {
    return {
      responseType: "ScheduleQuery",
      actions: getScheduleQueryActions(content),
      confidence: 0.8,
    };
  }

  // Check for find_free_time results
  const hasFindFreeTime = toolNames.some((n) => n.toLowerCase().includes("find_free_time"));
  const freeTimePatterns = ["available", "free slot", "free time", "空闲", "available time"];
  const isFreeTimeFound = hasFindFreeTime && freeTimePatterns.some((p) => lowerContent.includes(p.toLowerCase()));

  if (isFreeTimeFound) {
    return {
      responseType: "FreeTimeFound",
      actions: getFreeTimeFoundActions(toolCalls),
      confidence: 0.85,
    };
  }

  // Check for memo_search results
  const hasMemoSearch = toolNames.some((n) => n.toLowerCase().includes("memo_search"));
  const memoFoundPatterns = ["found", "memo", "note", "笔记", "找到", "related", "search result"];
  const isMemoFound = hasMemoSearch && memoFoundPatterns.some((p) => lowerContent.includes(p.toLowerCase()));

  if (isMemoFound) {
    return {
      responseType: "MemoFound",
      actions: getMemoFoundActions(content),
      confidence: 0.75,
    };
  }

  // Default generic responses
  return {
    responseType: "Generic",
    actions: getGenericActions(),
    confidence: 0.5,
  };
}

/**
 * Actions for ScheduleCreated response
 */
function getScheduleCreatedActions(_content: string): QuickReplyAction[] {
  const actions: QuickReplyAction[] = [
    {
      id: "view_schedule",
      label: "ai.quick_replies.view_schedule",
      icon: "Calendar",
      type: "navigate",
      payload: "/schedule",
      hint: "View on calendar",
    },
    {
      id: "add_another",
      label: "ai.quick_replies.add_another",
      icon: "Plus",
      type: "fill_input",
      payload: "help me schedule",
      hint: "Create another event",
    },
    {
      id: "query_today",
      label: "ai.quick_replies.query_today",
      icon: "Clock",
      type: "fill_input",
      payload: "what do I have planned for today",
      hint: "Check today's schedule",
    },
  ];

  return actions;
}

/**
 * Actions for ScheduleQuery response
 */
function getScheduleQueryActions(content: string): QuickReplyAction[] {
  const lowerContent = content.toLowerCase();

  // If no schedules found, suggest creating one
  if (lowerContent.includes("no schedules") || lowerContent.includes("未找到") || lowerContent.includes("没有日程")) {
    return [
      {
        id: "create_schedule",
        label: "ai.quick_replies.create_schedule",
        icon: "Plus",
        type: "fill_input",
        payload: "schedule a ",
        hint: "Create a new schedule",
      },
      {
        id: "check_tomorrow",
        label: "ai.quick_replies.check_tomorrow",
        icon: "CalendarDays",
        type: "fill_input",
        payload: "what's on my schedule tomorrow",
        hint: "Check tomorrow",
      },
    ];
  }

  // Schedules found - offer adjustment options
  return [
    {
      id: "adjust_time",
      label: "ai.quick_replies.adjust_time",
      icon: "Clock",
      type: "fill_input",
      payload: "move my ",
      hint: "Adjust schedule time",
    },
    {
      id: "add_related",
      label: "ai.quick_replies.add_related",
      icon: "Plus",
      type: "fill_input",
      payload: "also schedule ",
      hint: "Add related event",
    },
    {
      id: "view_calendar",
      label: "ai.quick_replies.view_calendar",
      icon: "Calendar",
      type: "navigate",
      payload: "/schedule",
      hint: "Open calendar view",
    },
  ];
}

/**
 * Actions for FreeTimeFound response
 */
function getFreeTimeFoundActions(toolCalls: ToolCall[]): QuickReplyAction[] {
  // Try to extract the time slot from tool output
  let timeSlot = "";
  for (const tool of toolCalls) {
    if (typeof tool === "object" && tool.outputSummary) {
      const match = tool.outputSummary.match(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
      if (match) {
        timeSlot = match[0];
        break;
      }
    }
  }

  return [
    {
      id: "create_at_slot",
      label: "ai.quick_replies.create_at_slot",
      icon: "CalendarPlus",
      type: "fill_input",
      payload: timeSlot ? `schedule a meeting at ${timeSlot}` : "schedule a meeting",
      hint: "Schedule at this time",
    },
    {
      id: "find_more_slots",
      label: "ai.quick_replies.find_more_slots",
      icon: "Search",
      type: "fill_input",
      payload: "find more free time",
      hint: "Find other slots",
    },
    {
      id: "check_week",
      label: "ai.quick_replies.check_week",
      icon: "CalendarDays",
      type: "fill_input",
      payload: "what does my week look like",
      hint: "View weekly schedule",
    },
  ];
}

/**
 * Actions for MemoFound response
 */
function getMemoFoundActions(content: string): QuickReplyAction[] {
  const lowerContent = content.toLowerCase();

  // If no memos found, suggest creating one
  if (lowerContent.includes("no memos") || lowerContent.includes("no related") || lowerContent.includes("未找到")) {
    return [
      {
        id: "create_memo",
        label: "ai.quick_replies.create_memo",
        icon: "Plus",
        type: "navigate",
        payload: "/",
        hint: "Create a new note",
      },
      {
        id: "try_different",
        label: "ai.quick_replies.try_different",
        icon: "Search",
        type: "fill_input",
        payload: "search for ",
        hint: "Try different keywords",
      },
    ];
  }

  // Memos found
  return [
    {
      id: "search_related",
      label: "ai.quick_replies.search_related",
      icon: "Search",
      type: "fill_input",
      payload: "find more about ",
      hint: "Search related content",
    },
    {
      id: "summarize",
      label: "ai.quick_replies.summarize",
      icon: "FileText",
      type: "fill_input",
      payload: "summarize these notes",
      hint: "Get a summary",
    },
    {
      id: "open_memo",
      label: "ai.quick_replies.open_memo",
      icon: "ExternalLink",
      type: "navigate",
      payload: "/explore",
      hint: "View in explore",
    },
  ];
}

/**
 * Actions for Error response
 */
function getErrorActions(): QuickReplyAction[] {
  return [
    {
      id: "retry",
      label: "ai.quick_replies.retry",
      icon: "RefreshCw",
      type: "fill_input",
      payload: "",
      hint: "Try again",
    },
    {
      id: "report_issue",
      label: "ai.quick_replies.report_issue",
      icon: "AlertTriangle",
      type: "navigate",
      payload: "https://github.com/hrygo/divinesense/issues",
      hint: "Report on GitHub",
    },
  ];
}

/**
 * Generic fallback actions
 */
function getGenericActions(): QuickReplyAction[] {
  return [
    {
      id: "ask_schedule",
      label: "ai.quick_replies.ask_schedule",
      icon: "Calendar",
      type: "fill_input",
      payload: "what's on my schedule today",
      hint: "Check schedule",
    },
    {
      id: "ask_memo",
      label: "ai.quick_replies.ask_memo",
      icon: "Search",
      type: "fill_input",
      payload: "search my notes for ",
      hint: "Search notes",
    },
    {
      id: "create_note",
      label: "ai.quick_replies.create_note",
      icon: "Plus",
      type: "navigate",
      payload: "/",
      hint: "Create a note",
    },
  ];
}

/**
 * Extract ISO8601 timestamp from text
 */
export function extractTimestamp(text: string): string | null {
  const isoPattern = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/;
  const match = text.match(isoPattern);
  return match ? match[0] : null;
}

/**
 * Check if analysis suggests any actions
 */
export function hasQuickReplies(analysis: QuickReplyAnalysis): boolean {
  return analysis.actions.length > 0 && analysis.confidence > 0.6;
}
