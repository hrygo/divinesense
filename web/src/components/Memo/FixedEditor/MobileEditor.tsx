/**
 * MobileEditor - Progressive Mobile Memo Editor
 *
 * 移动端渐进式编辑器：
 * - 默认态：单行输入框 + 展开按钮 + 发送按钮
 * - 展开态：Bottom Sheet 面板（iOS Share Sheet 风格）
 *
 * Features:
 * - Virtual keyboard adaptation
 * - Progressive disclosure for tools
 * - Quick input for fast memo capture
 */

import { Image, Link as LinkIcon, MapPin, Paperclip, Send, X } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

export interface MobileEditorProps {
  placeholder?: string;
  className?: string;
  onConfirm?: () => void;
}

export const MobileEditor = memo(function MobileEditor({ placeholder, className, onConfirm }: MobileEditorProps) {
  const t = useTranslate();
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const [isExpanded, setIsExpanded] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const lastHeightRef = useRef(0);

  // Handle mobile keyboard visibility with debouncing
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

  // Auto-resize textarea with RAF optimization
  const handleInput = useCallback((e: React.FormEvent<HTMLTextAreaElement>) => {
    const target = e.target as HTMLTextAreaElement;
    setInputValue(target.value);

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

  // Cleanup RAF on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  // Reset height when value clears
  useEffect(() => {
    if (textareaRef.current && !inputValue) {
      textareaRef.current.style.height = "auto";
    }
  }, [inputValue]);

  const handleSend = useCallback(() => {
    if (inputValue.trim()) {
      onConfirm?.();
      setInputValue("");
      if (textareaRef.current) {
        textareaRef.current.style.height = "auto";
      }
    }
  }, [inputValue, onConfirm]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter") {
        if (e.ctrlKey || e.metaKey) {
          // Insert newline at cursor position
          e.preventDefault();
          const target = e.target as HTMLTextAreaElement;
          const start = target.selectionStart;
          const end = target.selectionEnd;
          const value = target.value;
          const newValue = value.slice(0, start) + "\n" + value.slice(end);

          // Update React state first (React will handle DOM)
          setInputValue(newValue);

          // Use queueMicrotask to update cursor after React renders
          // Use ref to ensure we get the latest DOM element after React re-renders
          queueMicrotask(() => {
            const currentTextarea = textareaRef.current;
            if (currentTextarea) {
              currentTextarea.selectionStart = currentTextarea.selectionEnd = start + 1;
            }
          });
        } else {
          // Send message
          e.preventDefault();
          handleSend();
        }
      }
    },
    [handleSend],
  );

  const handleExpand = useCallback(() => {
    setIsExpanded(true);
    // Focus textarea when sheet opens
    setTimeout(() => textareaRef.current?.focus(), 100);
  }, []);

  // Platform-specific keyboard shortcut
  const shortcutKey = useMemo(() => {
    if (typeof window === "undefined") return "Ctrl";
    return /Mac|iPod|iPhone|iPad/.test(window.navigator.platform) ? "⌘" : "Ctrl";
  }, []);

  const canSend = inputValue.trim().length > 0;

  return (
    <>
      {/* Bottom Fixed Bar */}
      <div
        className={cn("fixed bottom-0 left-0 right-0 z-50 bg-background border-t border-border transition-all duration-300", className)}
        style={{
          paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : "max(1rem, env(safe-area-inset-bottom))",
        }}
      >
        <div className="mx-auto max-w-[100rem] px-4 py-3">
          <div className="flex items-end gap-2">
            {/* Main Input */}
            <div className="flex-1 relative">
              <textarea
                ref={textareaRef}
                value={inputValue}
                onChange={handleInput}
                onKeyDown={handleKeyDown}
                placeholder={placeholder}
                rows={1}
                className={cn(
                  "w-full min-h-[44px] max-h-[120px] px-4 py-3 pr-24",
                  "bg-muted/50 border border-border/50 rounded-xl",
                  "text-sm resize-none outline-none",
                  "focus:bg-background focus:border-primary/50",
                  "transition-colors",
                  "placeholder:text-muted-foreground",
                )}
                style={{ height: "auto" }}
              />

              {/* Right Side Actions */}
              <div className="absolute right-2 bottom-2 flex items-center gap-1">
                {/* Quick Expand Button */}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={handleExpand}
                  className="h-9 w-9 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted"
                  aria-label={t("editor.expand-toolbar")}
                >
                  <Paperclip className="w-5 h-5" />
                </Button>

                {/* Send Button */}
                <Button
                  type="button"
                  size="icon"
                  onClick={handleSend}
                  disabled={!canSend}
                  className={cn(
                    "h-9 w-9 rounded-md transition-all",
                    canSend ? "bg-primary text-primary-foreground hover:scale-105" : "bg-muted text-muted-foreground",
                  )}
                  aria-label={t("editor.send")}
                >
                  <Send className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>

          {/* Hint Text */}
          {!inputValue && (
            <div className="mt-2 text-center">
              <p className="text-xs text-muted-foreground">
                <kbd className="px-1.5 py-0.5 bg-muted rounded">Enter</kbd> {t("editor.enter-to-send")} ·
                <kbd className="px-1.5 py-0.5 bg-muted rounded ml-1">{shortcutKey} + Enter</kbd> {t("editor.for-newline")}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Expanded Bottom Sheet */}
      <Sheet open={isExpanded} onOpenChange={setIsExpanded}>
        <SheetContent
          side="bottom"
          className="rounded-t-3xl border-border"
          style={{
            paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : "max(1rem, env(safe-area-inset-bottom))",
          }}
        >
          <SheetHeader className="pb-4">
            <div className="flex items-center justify-between">
              <SheetTitle className="text-center flex-1">{t("editor.add-attachment")}</SheetTitle>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsExpanded(false)}
                className="absolute right-4 top-4"
                aria-label={t("common.close")}
              >
                <X className="w-5 h-5" />
              </Button>
            </div>
          </SheetHeader>

          {/* Tool Grid */}
          <div className="grid grid-cols-4 gap-4 py-4">
            {[
              { icon: Image, label: t("editor.upload-image"), color: "text-blue-500", bgColor: "bg-blue-50 dark:bg-blue-900/20" },
              { icon: Paperclip, label: t("editor.upload-file"), color: "text-amber-500", bgColor: "bg-amber-50 dark:bg-amber-900/20" },
              { icon: LinkIcon, label: t("editor.link-memo"), color: "text-purple-500", bgColor: "bg-purple-50 dark:bg-purple-900/20" },
              { icon: MapPin, label: t("editor.add-location"), color: "text-green-500", bgColor: "bg-green-50 dark:bg-green-900/20" },
            ].map((tool) => {
              const Icon = tool.icon;
              return (
                <button
                  key={tool.label}
                  type="button"
                  className={cn(
                    "flex flex-col items-center justify-center gap-2 p-4 rounded-xl",
                    "hover:bg-accent transition-colors",
                    "active:scale-95 transition-transform",
                  )}
                >
                  <div className={cn("w-12 h-12 rounded-full flex items-center justify-center", tool.bgColor)}>
                    <Icon className={cn("w-6 h-6", tool.color)} />
                  </div>
                  <span className="text-xs text-muted-foreground">{tool.label}</span>
                </button>
              );
            })}
          </div>

          {/* Advanced Options */}
          <div className="mt-4 space-y-2">
            <button
              type="button"
              className="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
            >
              <span className="text-sm">{t("editor.visibility")}</span>
              <span className="text-sm text-muted-foreground">{t("editor.private")}</span>
            </button>
            <button
              type="button"
              className="w-full flex items-center justify-between px-4 py-3 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
            >
              <span className="text-sm">{t("editor.related-memo")}</span>
              <span className="text-xs text-muted-foreground">...</span>
            </button>
          </div>

          {/* Confirm Button */}
          <div className="mt-6">
            <Button
              onClick={() => {
                handleSend();
                setIsExpanded(false);
              }}
              disabled={!canSend}
              className={cn(
                "w-full h-12 rounded-lg text-base font-medium transition-all",
                canSend ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground",
              )}
            >
              {canSend ? t("editor.send-note") : t("editor.input-content-after-send")}
            </Button>
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
});

MobileEditor.displayName = "MobileEditor";
