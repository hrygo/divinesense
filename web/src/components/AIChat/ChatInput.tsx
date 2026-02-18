import { MessageSquarePlus, Scissors, SendIcon, Square, Terminal, Trash2 } from "lucide-react";
import { KeyboardEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { ParrotInfoFromAPI } from "@/hooks/useParrotsList";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import { PARROT_THEMES } from "@/types/parrot";
import { canInsertMention, insertAgentMention, shouldTriggerMentionPopover } from "@/utils/agentMention";
import { AgentMentionPopover } from "./AgentMentionPopover";
import { ModeCycleButton } from "./ModeCycleButton";

interface ChatInputProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  onStop?: () => void;
  onNewChat?: () => void;
  onClearContext?: () => void;
  onClearChat?: () => void;
  onModeChange?: (mode: AIMode) => void;
  disabled?: boolean;
  isTyping?: boolean;
  placeholder?: string;
  className?: string;
  showQuickActions?: boolean;
  quickActions?: React.ReactNode;
  currentMode?: AIMode;
}

export function ChatInput({
  value,
  onChange,
  onSend,
  onStop,
  onNewChat,
  onClearContext,
  onClearChat,
  onModeChange,
  disabled = false,
  isTyping = false,
  placeholder,
  className,
  showQuickActions = false,
  quickActions,
  currentMode = "normal",
}: ChatInputProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const inputContainerRef = useRef<HTMLDivElement>(null);
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  // Track height to avoid unnecessary auto-reset causing jitter
  const lastHeightRef = useRef(0);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);

  // Agent Mention Popover state
  const [mentionPopoverOpen, setMentionPopoverOpen] = useState(false);
  const [mentionFilter, setMentionFilter] = useState("");

  // Detect macOS for correct shortcut display - memoized
  const sendShortcut = useMemo(() => {
    const isMac = typeof window !== "undefined" && /Mac|iPod|iPhone|iPad/.test(window.navigator.platform);
    return isMac ? "⌘+Enter" : "Ctrl+Enter";
  }, []);

  // Handle mobile keyboard visibility with debouncing
  useEffect(() => {
    if (typeof window === "undefined" || !window.visualViewport) return;

    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let lastHeight = 0;

    const handleResize = () => {
      const viewport = window.visualViewport;
      if (!viewport) return;

      // Debounce: only update if height significantly changed
      const currentHeight = viewport.height;
      if (Math.abs(currentHeight - lastHeight) < 10) {
        return; // Skip small changes
      }
      lastHeight = currentHeight;

      if (timeoutId) clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        const windowHeight = window.innerHeight;
        const keyboardVisible = currentHeight < windowHeight * 0.85;
        const newKeyboardHeight = keyboardVisible ? windowHeight - currentHeight : 0;

        setKeyboardHeight(newKeyboardHeight);
      }, 100);
    };

    window.visualViewport.addEventListener("resize", handleResize);
    return () => {
      if (timeoutId) clearTimeout(timeoutId);
      window.visualViewport?.removeEventListener("resize", handleResize);
    };
  }, []);

  // Detect @ input and trigger popover
  const checkForMentionTrigger = useCallback((text: string, cursorPosition: number) => {
    const { shouldTrigger, filter } = shouldTriggerMentionPopover(text, cursorPosition);

    if (shouldTrigger && canInsertMention(text, cursorPosition - 1)) {
      setMentionFilter(filter);
      setMentionPopoverOpen(true);
    } else {
      setMentionPopoverOpen(false);
    }
  }, []);

  // Handle agent selection from popover
  const handleAgentSelect = useCallback(
    (agent: ParrotInfoFromAPI) => {
      if (!textareaRef.current) return;

      const textarea = textareaRef.current;
      const cursorPosition = textarea.selectionStart;

      // Find the @ position
      let atPosition = cursorPosition - 1;
      while (atPosition >= 0 && textarea.value[atPosition] !== "@") {
        atPosition--;
      }

      if (atPosition < 0) return;

      // Remove the @ and any filter text after it
      const beforeAt = textarea.value.slice(0, atPosition);
      const afterCursor = textarea.value.slice(cursorPosition);

      // Insert the agent mention
      const displayName = agent.displayName || agent.name;
      const { newText, newCursorPos } = insertAgentMention(beforeAt + afterCursor, beforeAt.length, displayName);

      onChange(newText);

      // Set cursor position after the mention
      setTimeout(() => {
        textarea.focus();
        textarea.setSelectionRange(newCursorPos, newCursorPos);
      }, 0);

      setMentionPopoverOpen(false);
    },
    [onChange],
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      // If popover is open, let it handle navigation keys
      if (mentionPopoverOpen && (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || e.key === "Escape")) {
        // These will be handled by the popover
        if (e.key === "Escape") {
          e.preventDefault();
          setMentionPopoverOpen(false);
        }
        return;
      }

      // Enter to send, Ctrl+Enter or Cmd+Enter for new line
      if (e.key === "Enter") {
        if (e.ctrlKey || e.metaKey) {
          // Insert newline at cursor position
          e.preventDefault();
          const target = e.target as HTMLTextAreaElement;
          const start = target.selectionStart;
          const end = target.selectionEnd;
          const value = target.value;
          const newValue = value.slice(0, start) + "\n" + value.slice(end);
          target.value = newValue;
          target.selectionStart = target.selectionEnd = start + 1;
          // Trigger input event to update height
          target.dispatchEvent(new Event("input", { bubbles: true }));
          onChange(newValue);
        } else {
          // Send message
          e.preventDefault();
          onSend();
        }
      }
    },
    [onSend, onChange, mentionPopoverOpen],
  );

  const handleInput = useCallback(
    (e: React.FormEvent<HTMLTextAreaElement>) => {
      const target = e.target as HTMLTextAreaElement;

      // Check for @ mention trigger
      checkForMentionTrigger(target.value, target.selectionStart);

      // Cancel pending RAF if any
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }

      // Schedule height update in next animation frame
      rafIdRef.current = requestAnimationFrame(() => {
        // Ensure component is still mounted and target is valid
        if (!target || !textareaRef.current) return;

        const currentScrollHeight = target.scrollHeight;
        const maxHeight = 120;
        const newHeight = Math.min(currentScrollHeight, maxHeight);

        // Only update if height actually changed (avoid jitter)
        if (newHeight !== lastHeightRef.current) {
          lastHeightRef.current = newHeight;
          target.style.height = `${newHeight}px`;
        }

        rafIdRef.current = null;
      });
    },
    [checkForMentionTrigger],
  );

  // Reset height when value changes externally
  useEffect(() => {
    if (textareaRef.current && !value) {
      textareaRef.current.style.height = "auto";
    }
  }, [value]);

  // Use mode-specific placeholder - memoized
  const placeholderText = useMemo(() => {
    if (currentMode === "geek") {
      return t("ai.parrot.geek-chat-placeholder");
    }
    if (currentMode === "evolution") {
      return t("ai.parrot.evolution-chat-placeholder");
    }
    return placeholder || t("ai.parrot.chat-default-placeholder");
  }, [currentMode, placeholder, t]);

  // Get theme based on current mode - memoized
  const currentTheme = useMemo(() => {
    switch (currentMode) {
      case "geek":
        return PARROT_THEMES.GEEK;
      case "evolution":
        return PARROT_THEMES.EVOLUTION;
      default:
        return PARROT_THEMES.NORMAL;
    }
  }, [currentMode]);

  // Mode-specific styles - memoized to avoid recreating on every render
  const modeStyles = useMemo(() => {
    return {
      footerBg: currentTheme.footerBg,
      inputBg: currentTheme.inputBg,
      inputBorder: currentTheme.inputBorder,
      inputFocus: currentTheme.inputFocus,
    };
  }, [currentTheme]);

  return (
    <div
      className={cn("shrink-0 p-3 md:p-4 border-t border-border transition-colors", modeStyles.footerBg, className)}
      style={{
        paddingBottom: keyboardHeight > 0 ? `${keyboardHeight + 16}px` : "max(16px, env(safe-area-inset-bottom))",
      }}
    >
      <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto">
        {/* Quick Actions */}
        {showQuickActions && quickActions}

        {/* Toolbar - 工具栏 */}
        {(onNewChat || onClearContext || onClearChat || onModeChange) && (
          <div className="flex items-center gap-2 mb-2">
            {/* Left side buttons */}
            {onNewChat && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onNewChat && onNewChat()}
                aria-label={t("ai.aichat.sidebar.new-chat")}
                className="group/btn h-9 w-9 md:h-8 md:w-8 hover:w-auto px-0 hover:px-2 text-xs text-primary hover:text-primary hover:bg-accent transition-all overflow-hidden"
                title="⌘N"
              >
                <MessageSquarePlus className="w-4 h-4 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.aichat.sidebar.new-chat")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-accent rounded">⌘N</kbd>
                </span>
              </Button>
            )}
            {onClearContext && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onClearContext && onClearContext()}
                aria-label={t("ai.clear-context")}
                className="group/btn h-9 w-9 md:h-8 md:w-8 hover:w-auto px-0 hover:px-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-all overflow-hidden"
                title="⌘K"
              >
                <Scissors className="w-4 h-4 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.clear-context")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-muted rounded">⌘K</kbd>
                </span>
              </Button>
            )}
            {onClearChat && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onClearChat && onClearChat()}
                aria-label={t("ai.clear-chat")}
                className="group/btn h-9 w-9 md:h-8 md:w-8 hover:w-auto px-0 hover:px-2 text-xs text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-all overflow-hidden"
                title="⌘L"
              >
                <Trash2 className="w-4 h-4 shrink-0" />
                <span className="hidden group-hover/btn:inline ml-1 whitespace-nowrap">
                  {t("ai.clear-chat")}
                  <kbd className="ml-1.5 px-1 py-0.5 text-[10px] bg-destructive/20 rounded">⌘L</kbd>
                </span>
              </Button>
            )}
            {/* Spacer to push mode cycle button to the right */}
            <div className="flex-1" />
            {/* Shortcut hint - desktop only */}
            <span className="hidden sm:inline text-xs text-muted-foreground">
              <kbd className="px-1 py-0.5 bg-muted rounded">Enter</kbd> {t("ai.input-hint-send", { key: "Enter" })} ·
              <kbd className="px-1 py-0.5 bg-muted rounded ml-1">{sendShortcut}</kbd> {t("ai.input-hint-newline", { key: sendShortcut })}
            </span>
            {/* Mode Cycle Button - show on all devices */}
            {onModeChange && <ModeCycleButton currentMode={currentMode} onModeChange={onModeChange} variant="toolbar" isAdmin={true} />}
          </div>
        )}
        {/* Shortcut hint for when no toolbar buttons - always visible on sm+ screens */}
        {!onNewChat && !onClearContext && !onClearChat && !onModeChange && (
          <div className="flex items-center justify-end mb-2">
            <span className="hidden sm:inline text-xs text-muted-foreground">
              <kbd className="px-1 py-0.5 bg-muted rounded">Enter</kbd> {t("ai.input-hint-send", { key: "Enter" })} ·
              <kbd className="px-1 py-0.5 bg-muted rounded ml-1">{sendShortcut}</kbd> {t("ai.input-hint-newline", { key: sendShortcut })}
            </span>
          </div>
        )}

        {/* Input Box */}
        <div
          ref={inputContainerRef}
          className={cn(
            "flex items-end gap-2 md:gap-3 p-2.5 md:p-3 rounded-lg border shadow-sm transition-colors",
            modeStyles.inputBg,
            modeStyles.inputBorder,
            modeStyles.inputFocus,
            "focus-within:ring-2 focus-within:ring-offset-2",
          )}
          style={{ contain: "layout" }}
        >
          <Textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => {
              onChange(e.target.value);
              handleInput(e);
            }}
            onKeyDown={handleKeyDown}
            placeholder={placeholderText}
            disabled={disabled}
            className={cn(
              "flex-1 min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none text-sm leading-relaxed transition-colors",
              "text-foreground placeholder:text-muted-foreground",
            )}
            rows={1}
          />
          <Button
            size="icon"
            className={cn(
              "shrink-0 h-11 min-w-[44px] rounded-lg transition-all",
              "hover:scale-105 active:scale-95",
              isTyping
                ? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
                : value.trim()
                  ? `${currentTheme.accent} ${currentTheme.accentText}`
                  : "bg-muted text-muted-foreground",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100",
            )}
            onClick={() => {
              if (isTyping && onStop) {
                onStop();
              } else {
                onSend();
              }
            }}
            // Stop button should always be clickable; Send button requires input
            disabled={disabled || (!isTyping && !value.trim())}
            aria-label={isTyping ? "Stop generating" : `${sendShortcut} Send`}
          >
            {isTyping ? (
              <Square className="w-5 h-5 fill-current" />
            ) : currentMode === "geek" && value.trim() ? (
              <Terminal className="w-5 h-5" />
            ) : currentMode === "evolution" && value.trim() ? (
              <Terminal className="w-5 h-5 text-purple-500" />
            ) : (
              <SendIcon className="w-5 h-5" />
            )}
          </Button>
        </div>

        {/* Agent Mention Popover */}
        <AgentMentionPopover
          open={mentionPopoverOpen}
          onOpenChange={setMentionPopoverOpen}
          onSelect={handleAgentSelect}
          anchorElement={inputContainerRef.current}
          filter={mentionFilter}
        />
      </div>
    </div>
  );
}
