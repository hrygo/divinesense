import { ChevronUp, Send, Sparkles } from "lucide-react";
import { KeyboardEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

interface QuickInputProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  onExpand?: () => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  showExpandButton?: boolean;
}

/**
 * QuickInput component for fast memo capture
 *
 * Design Principles:
 * - 圆角：统一 rounded-xl / rounded-2xl
 * - 按钮高度：h-11 (44px) 触摸友好
 * - 发送按钮：带微交互动画和状态指示
 * - 展开按钮：使用 Sparkles 图标（AI 感知）
 *
 * Minimal UI with expand button for standard mode
 * Reuses ChatInput's virtual keyboard adaptation logic
 */
export function QuickInput({
  value,
  onChange,
  onSend,
  onExpand,
  disabled = false,
  placeholder,
  className,
  showExpandButton = true,
}: QuickInputProps) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const lastHeightRef = useRef(0);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);

  // Detect macOS for correct shortcut display
  const sendShortcut = useMemo(() => {
    const isMac = typeof window !== "undefined" && /Mac|iPod|iPhone|iPad/.test(window.navigator.platform);
    return isMac ? "⌘+Enter" : "Ctrl+Enter";
  }, []);

  // Handle mobile keyboard visibility with debouncing (reused from ChatInput)
  useEffect(() => {
    if (typeof window === "undefined" || !window.visualViewport) return;

    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let lastHeight = 0;

    const handleResize = () => {
      const viewport = window.visualViewport;
      if (!viewport) return;

      const currentHeight = viewport.height;
      if (Math.abs(currentHeight - lastHeight) < 10) {
        return;
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

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter") {
        if (e.ctrlKey || e.metaKey) {
          // Insert newline at cursor position
          e.preventDefault();
          const target = e.target as HTMLTextAreaElement;
          const start = target.selectionStart;
          const end = target.selectionEnd;
          const val = target.value;
          const newValue = val.slice(0, start) + "\n" + val.slice(end);
          target.value = newValue;
          target.selectionStart = target.selectionEnd = start + 1;
          target.dispatchEvent(new Event("input", { bubbles: true }));
        } else {
          // Send memo
          e.preventDefault();
          onSend();
        }
      }
    },
    [onSend],
  );

  const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
    const target = e.target as HTMLTextAreaElement;

    if (rafIdRef.current) {
      cancelAnimationFrame(rafIdRef.current);
    }

    rafIdRef.current = requestAnimationFrame(() => {
      if (!target || !textareaRef.current) return;

      const currentScrollHeight = target.scrollHeight;
      const maxHeight = 120;
      const newHeight = Math.min(currentScrollHeight, maxHeight);

      if (newHeight !== lastHeightRef.current) {
        lastHeightRef.current = newHeight;
        target.style.height = `${newHeight}px`;
      }

      rafIdRef.current = null;
    });
  }, []);

  // Reset height when value changes externally
  useEffect(() => {
    if (textareaRef.current && !value) {
      textareaRef.current.style.height = "auto";
    }
  }, [value]);

  const placeholderText = placeholder || t("editor.quick-placeholder");
  const hasContent = value.trim().length > 0;
  const showAIHint = value.trim().length > 20; // 显示 AI 提示的阈值

  return (
    <div
      className={cn("shrink-0 p-3 md:p-4 border-t border-border/50 bg-background", className)}
      style={{
        paddingBottom: keyboardHeight > 0 ? `${keyboardHeight + 16}px` : "max(16px, env(safe-area-inset-bottom))",
      }}
    >
      <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto">
        {/* Shortcut hint */}
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-muted-foreground/80">
            <kbd className="px-1.5 py-0.5 bg-muted/70 rounded-md text-[10px] font-mono">Enter</kbd> {t("editor.send")}
            <span className="mx-1 text-muted-foreground/50">·</span>
            <kbd className="px-1.5 py-0.5 bg-muted/70 rounded-md text-[10px] font-mono">{sendShortcut}</kbd> {t("editor.newline")}
            {showAIHint && (
              <>
                <span className="mx-1 text-muted-foreground/50">·</span>
                <span className="flex items-center gap-1 text-purple-500/70">
                  <Sparkles className="w-3 h-3" />
                  AI ready
                </span>
              </>
            )}
          </span>
          {showExpandButton && onExpand && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onExpand}
              className="h-7 px-2.5 text-xs text-muted-foreground hover:text-foreground hover:bg-accent/50 rounded-lg transition-all hover:scale-105 active:scale-95"
              aria-label={t("editor.expand-to-standard")}
            >
              <Sparkles className="w-3 h-3 mr-1" />
              {t("editor.more-tools")}
              <ChevronUp className="w-3 h-3 ml-1" />
            </Button>
          )}
        </div>

        {/* Input Box */}
        <div
          className={cn(
            "flex items-end gap-2 md:gap-3 p-3 rounded-2xl border shadow-sm",
            "bg-muted/30 hover:bg-muted/50 transition-colors",
            "focus-within:ring-2 focus-within:ring-primary/20 focus-within:border-primary/50 focus-within:bg-muted/50",
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
              "flex-1 min-h-[44px] max-h-[120px] bg-transparent border-0 outline-none resize-none",
              "text-sm leading-relaxed transition-colors",
              "text-foreground placeholder:text-muted-foreground/70",
            )}
            rows={1}
          />
          <Button
            size="icon"
            className={cn(
              "shrink-0 h-11 min-w-[44px] rounded-xl transition-all duration-200",
              "hover:scale-105 active:scale-95",
              "relative overflow-hidden",
              hasContent
                ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm hover:shadow"
                : "bg-muted text-muted-foreground cursor-not-allowed",
              "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100",
            )}
            onClick={onSend}
            disabled={disabled || !hasContent}
            aria-label={t("editor.send")}
          >
            <Send className="w-5 h-5" />
          </Button>
        </div>
      </div>
    </div>
  );
}
