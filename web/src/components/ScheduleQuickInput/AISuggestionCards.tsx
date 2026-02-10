import { Check, Feather } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

// Parrot emoji for schedule assistant
const PARROT_ICON = "🦜";

// Screen reader announcement for selection changes
const announceSelection = (message: string) => {
  const announcement = document.createElement("div");
  announcement.setAttribute("role", "status");
  announcement.setAttribute("aria-live", "polite");
  announcement.className = "sr-only";
  announcement.textContent = message;
  document.body.appendChild(announcement);
  setTimeout(() => document.body.removeChild(announcement), 1000);
};

/**
 * Parsed suggestion from AI response
 */
export interface ScheduleSuggestion {
  id: string;
  title: string;
  date: string; // "今天", "明天", or formatted date
  startTime: string; // "15:00"
  endTime?: string; // "16:00"
  rawText: string; // Original text for reference
}

/**
 * Parse AI response to extract schedule suggestions
 * Supports formats like:
 * - "建议创建：明天下午3点开会"
 * - "可以帮您创建：今天15:00-16:00的会议"
 */
export function parseSuggestions(
  aiResponse: string,
  todayStr = "今天",
  tomorrowStr = "明天",
  t?: (key: string) => string | unknown,
): ScheduleSuggestion[] {
  if (!aiResponse) return [];

  const suggestions: ScheduleSuggestion[] = [];
  const lines = aiResponse.split("\n").filter((line) => line.trim());

  // Common patterns
  const patterns = [
    // "建议创建：明天下午3点开会"
    /(?:建议创建|可以创建|为您创建|创建)\s*[:：]?\s*(今天|明天)?(?:上午|下午|晚上)?(\d{1,2})(?:点|时)(?:半)?(?:\d{1,2}分?)?(?:到|-)?(?:\d{1,2})?(?:点|时)?(?:半)?的?(.+)/gi,
    // "今天15:00开会" or "明天 3:00 PM 会议"
    /(今天|明天|\d{1,2}月\d{1,2}日)\s*(\d{1,2}:\d{2}|\d{1,2}\s*(?:AM|PM|am|pm)?)\s*(?:到|-)\s*(\d{1,2}:\d{2}|\d{1,2}\s*(?:AM|PM|am|pm)?)?\s*的?(.+)/gi,
  ];

  for (const line of lines) {
    for (const pattern of patterns) {
      pattern.lastIndex = 0; // Reset regex
      const match = pattern.exec(line);
      if (match) {
        const [, datePart, timePart, endTimePart, titlePart] = match;
        const suggestion = parseMatchToSuggestion(datePart, timePart, endTimePart, titlePart, line, todayStr, tomorrowStr, t);
        if (suggestion) {
          suggestions.push(suggestion);
          break; // Use first match only per line
        }
      }
    }
  }

  // Also check for numbered list items like "1. 明天3点开会"
  const listPattern = /^\d+[.、]\s*(.+)/;
  for (const line of lines) {
    const listMatch = line.match(listPattern);
    if (listMatch) {
      const content = listMatch[1];
      // Try to parse the content
      for (const pattern of patterns) {
        pattern.lastIndex = 0;
        const match = pattern.exec(content);
        if (match) {
          const [, datePart, timePart, endTimePart, titlePart] = match;
          const suggestion = parseMatchToSuggestion(datePart, timePart, endTimePart, titlePart, content, todayStr, tomorrowStr, t);
          if (suggestion && !suggestions.find((s) => s.rawText === content)) {
            suggestions.push(suggestion);
            break;
          }
        }
      }
    }
  }

  return suggestions.slice(0, 3); // Max 3 suggestions
}

function parseMatchToSuggestion(
  datePart: string | undefined,
  timePart: string | undefined,
  endTimePart: string | undefined,
  titlePart: string | undefined,
  rawText: string,
  todayStr: string,
  tomorrowStr: string,
  t?: (key: string) => string | unknown,
): ScheduleSuggestion | null {
  // Parse date
  let date = todayStr;
  if (datePart) {
    if (datePart.includes("明天")) date = tomorrowStr;
    else if (datePart.includes("今天")) date = todayStr;
    else date = datePart; // Use as-is for specific dates
  }

  // Parse time
  let startTime = "09:00";
  let endTime: string | undefined;

  if (timePart) {
    startTime = normalizeTime(timePart);
  }

  if (endTimePart) {
    endTime = normalizeTime(endTimePart);
  }

  // Parse title
  const title = titlePart?.trim() || "";
  const defaultTitle = (t?.("schedule.quick-input.default-title") as string) ?? "";

  return {
    id: `${Date.now()}-${Math.random()}`,
    title: title || defaultTitle,
    date,
    startTime,
    endTime,
    rawText,
  };
}

function normalizeTime(timeStr: string): string {
  // Handle "3点" -> "03:00"
  const hourMatch = timeStr.match(/(\d{1,2})\s*(?:点|时)/);
  if (hourMatch) {
    let hour = parseInt(hourMatch[1], 10);
    // Check for 下午/晚上
    if ((timeStr.includes("下午") || timeStr.includes("晚上")) && hour < 12) {
      hour += 12;
    }
    // Check for 上午
    if (timeStr.includes("上午") && hour === 12) {
      hour = 0;
    }
    // Check for "半" (30 minutes)
    const minute = timeStr.includes("半") ? "30" : "00";
    return `${hour.toString().padStart(2, "0")}:${minute}`;
  }

  // Handle "15:00" or "3:00 PM"
  const standardMatch = timeStr.match(/(\d{1,2}):(\d{2})\s*(AM|PM|am|pm)?/);
  if (standardMatch) {
    let hour = parseInt(standardMatch[1], 10);
    const minute = standardMatch[2];
    const meridiem = standardMatch[3]?.toUpperCase();

    if (meridiem === "PM" && hour < 12) hour += 12;
    if (meridiem === "AM" && hour === 12) hour = 0;

    return `${hour.toString().padStart(2, "0")}:${minute}`;
  }

  // Handle "3 PM" or "3pm"
  const simpleMatch = timeStr.match(/(\d{1,2})\s*(AM|PM|am|pm)/);
  if (simpleMatch) {
    let hour = parseInt(simpleMatch[1], 10);
    const meridiem = simpleMatch[2].toUpperCase();

    if (meridiem === "PM" && hour < 12) hour += 12;
    if (meridiem === "AM" && hour === 12) hour = 0;

    return `${hour.toString().padStart(2, "0")}:00`;
  }

  return "09:00"; // Default
}

interface AISuggestionCardsProps {
  /** Parsed suggestions to display */
  suggestions: ScheduleSuggestion[];
  /** Called when user confirms a suggestion */
  onConfirmSuggestion: (suggestion: ScheduleSuggestion) => void;
  /** Optional className */
  className?: string;
}

/**
 * Displays AI-suggested schedules as clickable cards.
 * UX Design:
 * - Desktop: Double-click to create directly
 * - Mobile: Single click to select, click again to confirm
 */
export function AISuggestionCards({ suggestions, onConfirmSuggestion, className }: AISuggestionCardsProps) {
  const t = useTranslate();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [lastClickTime, setLastClickTime] = useState(0);

  if (suggestions.length === 0) {
    return null;
  }

  const handleCardClick = (suggestion: ScheduleSuggestion) => {
    const now = Date.now();
    const timeSinceLastClick = now - lastClickTime;

    // Desktop: Double-click detection (within 300ms)
    const isDoubleClick = timeSinceLastClick < 300 && selectedId === suggestion.id;

    // Mobile: Toggle selection
    if (isDoubleClick) {
      // Double-click on desktop: create directly
      onConfirmSuggestion(suggestion);
      setSelectedId(null);
      announceSelection(t("schedule.quick-input.schedule-created") as string);
    } else if (selectedId === suggestion.id) {
      // Second click on same card (mobile): confirm
      onConfirmSuggestion(suggestion);
      setSelectedId(null);
      announceSelection(t("schedule.quick-input.schedule-created") as string);
    } else {
      // First click: select the card
      setSelectedId(suggestion.id);
      const defaultTitle = t("schedule.quick-input.default-title") as string;
      announceSelection(
        (t("schedule.quick-input.selected-label") as string)
          .replace("{title}", suggestion.title || defaultTitle)
          .replace("{date}", suggestion.date)
          .replace("{time}", suggestion.startTime),
      );
    }

    setLastClickTime(now);
  };

  const handleKeyDown = (e: React.KeyboardEvent, suggestion: ScheduleSuggestion) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      handleCardClick(suggestion);
    } else if (e.key === "Escape" && selectedId) {
      e.preventDefault();
      setSelectedId(null);
      announceSelection(t("schedule.quick-input.selection-canceled") as string);
    }
  };

  return (
    <div className={cn("w-full", className)}>
      {/* Header with parrot */}
      <div className="flex items-center gap-2 px-1 mb-2">
        <span className="text-base" role="img" aria-label="金刚鹦鹉">
          {PARROT_ICON}
        </span>
        <span className="text-xs text-muted-foreground">{t("schedule.quick-input.ai-suggestions") as string}</span>
      </div>

      {/* Desktop: horizontal row | Mobile: vertical stack */}
      <div
        className="flex flex-col sm:flex-row gap-2 sm:gap-3"
        role="listbox"
        aria-label={t("schedule.quick-input.ai-suggestions") as string}
      >
        {suggestions.map((suggestion) => {
          const isSelected = selectedId === suggestion.id;
          const defaultTitle = t("schedule.quick-input.default-title") as string;
          const ariaLabel = (t("schedule.quick-input.create-schedule-label") as string)
            .replace("{title}", suggestion.title || defaultTitle)
            .replace("{date}", suggestion.date)
            .replace("{time}", suggestion.startTime)
            .replace("{endTime}", suggestion.endTime ? ` - ${suggestion.endTime}` : "");

          return (
            <button
              key={suggestion.id}
              type="button"
              role="option"
              aria-selected={isSelected}
              aria-label={ariaLabel}
              tabIndex={0}
              onClick={() => handleCardClick(suggestion)}
              onKeyDown={(e) => handleKeyDown(e, suggestion)}
              className={cn(
                "group flex-1 min-w-0 text-left relative",
                "p-3 rounded-lg border-2 transition-all duration-200",
                "active:scale-[0.98]",
                // Focus visible styles for keyboard navigation
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2",
                // Normal state
                !isSelected &&
                  "border-amber-500/20 bg-amber-50/50 dark:bg-amber-950/20 hover:border-amber-500/40 hover:bg-amber-50/80 dark:hover:bg-amber-950/30",
                // Selected state - improved contrast with amber theme for parrot
                isSelected && "border-amber-600 bg-amber-100 dark:bg-amber-900/50 ring-2 ring-amber-500/40 shadow-sm",
              )}
            >
              {/* Selected indicator */}
              {isSelected && (
                <div className="absolute top-2 right-2 w-5 h-5 rounded-full bg-amber-500 flex items-center justify-center animate-in zoom-in-50">
                  <Check className="w-3 h-3 text-white" />
                </div>
              )}

              {/* Parrot icon badge */}
              <div className="flex items-center gap-2 mb-2">
                <div className="w-6 h-6 rounded-full bg-gradient-to-br from-amber-400 to-orange-500 flex items-center justify-center text-xs shadow-sm">
                  {PARROT_ICON}
                </div>
                <span className="text-[10px] text-amber-600 dark:text-amber-400 font-medium">金刚</span>
              </div>

              {/* Title */}
              <div
                className={cn(
                  "font-medium text-sm truncate mb-2 pr-6",
                  !isSelected && "group-hover:text-amber-700 dark:group-hover:text-amber-300",
                )}
              >
                {suggestion.title || (t("schedule.quick-input.default-title") as string)}
              </div>

              {/* Time */}
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Feather className="w-3 h-3 flex-shrink-0" />
                <span>{suggestion.date}</span>
                <span className="font-medium text-foreground">
                  {suggestion.startTime}
                  {suggestion.endTime && ` - ${suggestion.endTime}`}
                </span>
              </div>

              {/* Hint text */}
              <div className="mt-2 text-[10px]">
                {isSelected ? (
                  <span className="text-amber-700 dark:text-amber-300 font-semibold">
                    {t("schedule.quick-input.click-again-create") as string}
                  </span>
                ) : (
                  <span className="text-amber-600/80 hidden sm:inline">{t("schedule.quick-input.double-click-create") as string}</span>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
