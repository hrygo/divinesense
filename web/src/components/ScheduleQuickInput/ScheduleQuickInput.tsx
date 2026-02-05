import dayjs from "dayjs";
import { Check, ChevronRight, Loader2, Send, Sparkles, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useScheduleAgentStreamingChat } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { type Translations, useTranslate } from "@/utils/i18n";
import { AISuggestionCards, parseSuggestions, type ScheduleSuggestion } from "./AISuggestionCards";
import { QuickTemplateDropdown } from "./QuickTemplates";
import { StreamingFeedback } from "./StreamingFeedback";
import type { ScheduleTemplate } from "./types";

interface ScheduleQuickInputProps {
  initialDate?: string;
  onScheduleCreated?: () => void;
  editingSchedule?: Schedule | null;
  onClearEditing?: () => void;
  className?: string;
}

const MAX_INPUT_HEIGHT = 120;
const LINE_HEIGHT = 24;

export function ScheduleQuickInput({
  initialDate,
  onScheduleCreated,
  editingSchedule,
  onClearEditing,
  className,
}: ScheduleQuickInputProps) {
  const streamingChat = useScheduleAgentStreamingChat();
  const t = useTranslate();

  // State
  const [input, setInput] = useState("");
  const [inputHeight, setInputHeight] = useState(LINE_HEIGHT);
  const [lastInput, setLastInput] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const [showTemplates, setShowTemplates] = useState(false);
  const [aiMessage, setAiMessage] = useState("");
  const [aiSuggestions, setAiSuggestions] = useState<ScheduleSuggestion[]>([]);
  const [showSuccess, setShowSuccess] = useState(false);

  // Use streaming state for processing indicator
  const isProcessing = streamingChat.isStreaming;

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const resize = () => {
      if (!textareaRef.current) return;
      textareaRef.current.style.height = "auto";
      const newHeight = Math.min(Math.max(textareaRef.current.scrollHeight, LINE_HEIGHT), MAX_INPUT_HEIGHT);
      textareaRef.current.style.height = `${newHeight}px`;
      setInputHeight(newHeight);
    };

    resize();
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, [input]);

  // Reset success after delay
  useEffect(() => {
    if (showSuccess) {
      const timer = setTimeout(() => setShowSuccess(false), 2000);
      return () => clearTimeout(timer);
    }
  }, [showSuccess]);

  const handleScheduleCreated = () => {
    setShowSuccess(true);
    toast.success(t("schedule.schedule-created") || (t("schedule.quick.input.schedule-created-fallback") as string));
    setInput("");
    setLastInput("");
    setAiMessage("");
    setAiSuggestions([]);
    if (textareaRef.current) {
      textareaRef.current.style.height = `${LINE_HEIGHT}px`;
      setInputHeight(LINE_HEIGHT);
    }
    onScheduleCreated?.();
  };

  const handleSend = async () => {
    const trimmedInput = input.trim();
    if (!trimmedInput || isProcessing) return;

    // Save the input and clear immediately for better UX
    setLastInput(trimmedInput);
    setInput(""); // Clear input immediately
    setAiMessage("");

    // Build message with editing context if available
    let messageToSend = trimmedInput;
    if (editingSchedule) {
      const editTime = `${dayjs.unix(Number(editingSchedule.startTs)).format("HH:mm")}-${dayjs.unix(Number(editingSchedule.endTs)).format("HH:mm")}`;
      messageToSend = `把「${editingSchedule.title}」（${editTime}）${trimmedInput}`;
    }

    // Add date context to message for backend
    if (initialDate) {
      messageToSend = `[日期: ${initialDate}] ${messageToSend}`;
    }

    try {
      const response = await streamingChat.startChat(messageToSend, Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai");

      const aiResponse = response || "";
      const createdSchedule =
        aiResponse.includes("已成功创建") ||
        aiResponse.includes("成功创建日程") ||
        aiResponse.includes("successfully created") ||
        aiResponse.includes("已安排") ||
        aiResponse.includes("已为您创建") ||
        aiResponse.includes("已更新") ||
        aiResponse.includes("updated successfully") ||
        aiResponse.includes("修改成功");

      if (createdSchedule) {
        handleScheduleCreated();
        // Clear editing state after successful update
        if (editingSchedule) {
          onClearEditing?.();
        }
      } else {
        setAiMessage(aiResponse);
        const todayStr = t("schedule.quick-input.today") as string;
        const tomorrowStr = t("schedule.quick-input.tomorrow") as string;
        const suggestions = parseSuggestions(aiResponse, todayStr, tomorrowStr);
        setAiSuggestions(suggestions);
      }
    } catch (error) {
      console.error("[ScheduleQuickInput] AI error:", error);
      toast.error(t("schedule.parse-error") || (t("schedule.quick.input.parse-error-fallback") as string));
      // Restore input on error so user can retry
      setInput(trimmedInput);
    }
  };

  const handleTemplateSelect = (template: ScheduleTemplate) => {
    // Get natural language prompt for input display
    const promptText = template.promptI18nKey
      ? (t(template.promptI18nKey as Translations) as string) || template.prompt || template.title
      : template.prompt || template.title;

    // Only fill the input, let user edit before sending
    setInput(promptText);
    // Focus the textarea for editing
    textareaRef.current?.focus();
  };

  const handleAISuggestionSelect = async (suggestion: ScheduleSuggestion) => {
    const message = t("schedule.quick.input.suggestion-message", {
      date: suggestion.date,
      title: suggestion.title,
      startTime: suggestion.startTime,
      endTime: suggestion.endTime || "",
    }) as string;

    try {
      const response = await streamingChat.startChat(message, Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai");

      const aiResponse = response || "";
      if (aiResponse.includes("已成功创建") || aiResponse.includes("已安排") || aiResponse.includes("已为您创建")) {
        handleScheduleCreated();
      } else {
        setAiMessage(aiResponse);
      }
    } catch (error) {
      console.error("[ScheduleQuickInput] Suggestion error:", error);
      toast.error(t("schedule.quick.input.create-failed") as string);
    }
  };

  const handleClear = () => {
    setInput("");
    setLastInput("");
    setAiMessage("");
    setAiSuggestions([]);
    setShowSuccess(false);
    if (textareaRef.current) {
      textareaRef.current.style.height = `${LINE_HEIGHT}px`;
      setInputHeight(LINE_HEIGHT);
    }
  };

  const getPlaceholder = () => {
    // Always use placeholder without date for cleaner UI
    // Date context is sent to backend separately
    return t("schedule.quick.input.placeholder") as string;
  };

  // Display text: show current input (already cleared on send)
  // Don't show lastInput during processing - user wants input cleared
  const displayText = input;

  return (
    <div className={cn("w-full flex flex-col gap-2", className)}>
      {/* Priority 2: Success Message (only when not processing) */}
      {showSuccess && !isProcessing && (
        <div className="flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-green-500/10 to-green-500/5 rounded-xl border border-green-500/20 animate-in slide-in-from-top-2">
          <div className="flex-shrink-0 h-8 w-8 rounded-full bg-green-500 flex items-center justify-center">
            <Check className="h-5 w-5 text-white" />
          </div>
          <div>
            <p className="text-sm font-medium text-green-600 dark:text-green-400">{t("schedule.quick.input.created-success") as string}</p>
          </div>
        </div>
      )}

      {/* Priority 3: Streaming Feedback (only when processing) */}
      {isProcessing && (
        <>
          {/* Processing Status Bar - shown when streaming has no events yet */}
          {streamingChat.events.length === 0 && (
            <div className="flex items-center gap-3 px-4 py-3 bg-gradient-to-r from-primary/10 to-primary/5 rounded-xl border border-primary/20 animate-pulse">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              <div className="flex-1">
                <p className="text-sm font-medium text-foreground">{t("schedule.quick.input.creating") as string}</p>
                <p className="text-xs text-muted-foreground mt-0.5">"{lastInput}"</p>
              </div>
              <div className="flex gap-1">
                <span className="flex h-2 w-2 rounded-full bg-primary/40 animate-bounce [animation-delay:-0.3s]" />
                <span className="flex h-2 w-2 rounded-full bg-primary/40 animate-bounce [animation-delay:-0.15s]" />
                <span className="flex h-2 w-2 rounded-full bg-primary/40 animate-bounce" />
              </div>
            </div>
          )}

          {/* Streaming Feedback - Real-time AI thinking process (only when there are events) */}
          {streamingChat.events.length > 0 && (
            <StreamingFeedback events={streamingChat.events} isStreaming={isProcessing} className="mb-2" />
          )}
        </>
      )}

      {/* Priority 4: AI Response Message (only when not processing) */}
      {aiMessage && !isProcessing && (
        <div className="flex items-start gap-3 px-4 py-3 bg-muted/50 rounded-xl border border-border/50">
          <Sparkles className="h-5 w-5 text-primary flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <p className="text-sm text-foreground/90">{aiMessage}</p>
            {aiSuggestions.length > 0 && (
              <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                <ChevronRight className="h-3 w-3" />
                <span>{t("schedule.quick.input.select-suggestion") as string}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* AI Suggestions Cards */}
      {aiSuggestions.length > 0 && <AISuggestionCards suggestions={aiSuggestions} onConfirmSuggestion={handleAISuggestionSelect} />}

      {/* Input Bar */}
      <div
        className={cn(
          "flex items-center gap-2 p-2.5 rounded-xl border-2 transition-all duration-300",
          isProcessing && "border-primary/40 bg-primary/5",
          showSuccess && "border-green-500/40 bg-green-500/5",
          !isProcessing && !showSuccess && "border-border bg-background",
        )}
      >
        {/* Templates */}
        <div className="flex-shrink-0">
          <QuickTemplateDropdown
            open={showTemplates}
            onToggle={() => setShowTemplates(!showTemplates)}
            onSelect={handleTemplateSelect}
            disabled={isProcessing}
          />
        </div>

        {/* Text Input */}
        <div className="flex-1 min-w-0">
          <Textarea
            ref={textareaRef}
            value={displayText}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
              if (e.key === "Escape" && input) {
                e.preventDefault();
                handleClear();
              }
            }}
            placeholder={getPlaceholder()}
            className={cn(
              "min-h-[24px] max-h-[120px] py-2 px-3 resize-none",
              "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
              "text-sm",
              isProcessing && "opacity-70",
            )}
            style={{ height: `${inputHeight}px` }}
            rows={1}
            disabled={isProcessing}
          />
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-1.5 flex-shrink-0">
          {isProcessing ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin text-primary" />
            </div>
          ) : (
            <>
              {input.length > 0 && (
                <Button size="sm" variant="ghost" onClick={handleClear} className="h-8 w-8 rounded-full p-0">
                  <X className="h-4 w-4" />
                </Button>
              )}
              <Button
                size="sm"
                onClick={handleSend}
                disabled={!input.trim()}
                className={cn(
                  // Golden ratio: width ≈ height * 1.618, using 36x58 to match template button
                  "h-9 w-[52px] rounded-lg p-0 transition-all duration-200",
                  input.trim()
                    ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm"
                    : "bg-transparent text-muted-foreground hover:bg-muted/50",
                )}
              >
                <Send className="h-4 w-4" />
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
