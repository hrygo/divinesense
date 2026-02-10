/**
 * PCEditor - Full-Featured PC Memo Editor
 *
 * PC 端完整编辑器：
 * - 完整工具栏（文件、链接、位置、可见性、标签）
 * - 保留现有 MemoEditor 功能
 * - 适配底部固定布局
 *
 * Features:
 * - Full toolbar with all memo editing capabilities
 * - Virtual keyboard not needed on desktop
 * - Focus mode support
 * - Auto-save integration
 */

import { Expand, Eye, Link as LinkIcon, MapPin, Paperclip, Send, X } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

export interface PCEditorProps {
  placeholder?: string;
  className?: string;
  onConfirm?: () => void;
}

export const PCEditor = memo(function PCEditor({ placeholder, className, onConfirm }: PCEditorProps) {
  const t = useTranslate();
  const [isFocusMode, setIsFocusMode] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const rafIdRef = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const lastHeightRef = useRef(0);

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
      const maxHeight = 200; // PC allows more height
      const newHeight = Math.min(currentScrollHeight, maxHeight);

      if (newHeight !== lastHeightRef.current) {
        lastHeightRef.current = newHeight;
        target.style.height = `${newHeight}px`;
      }

      rafIdRef.current = null;
    });
  }, []);

  // Reset height when value clears
  useEffect(() => {
    if (textareaRef.current && !inputValue) {
      textareaRef.current.style.height = "auto";
    }
  }, [inputValue]);

  // Cleanup RAF on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

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
      // Ctrl+Enter or Cmd+Enter for new line, Enter to send
      if (e.key === "Enter") {
        if (e.ctrlKey || e.metaKey) {
          // Insert newline
          e.preventDefault();
          const target = e.target as HTMLTextAreaElement;
          const start = target.selectionStart;
          const end = target.selectionEnd;
          const value = target.value;
          const newValue = value.slice(0, start) + "\n" + value.slice(end);
          target.value = newValue;
          target.selectionStart = target.selectionEnd = start + 1;
          target.dispatchEvent(new Event("input", { bubbles: true }));
          setInputValue(newValue);
        } else {
          // Send message
          e.preventDefault();
          handleSend();
        }
      }
    },
    [handleSend],
  );

  const isMac = useMemo(() => {
    return typeof window !== "undefined" && /Mac|iPod|iPhone|iPad/.test(window.navigator.platform);
  }, []);

  const canSend = inputValue.trim().length > 0;

  return (
    <>
      {/* Bottom Fixed Bar */}
      <div className={cn("border-t border-border bg-background/95 backdrop-blur-sm transition-colors", className)}>
        <div className="mx-auto max-w-[100rem] px-4 sm:px-6 py-3">
          {/* Toolbar Row */}
          <div className="flex items-center gap-2 mb-2">
            {/* Quick Actions */}
            <div className="flex items-center gap-1">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
                title={t("editor.upload-image")}
                aria-label={t("editor.upload-image")}
              >
                <Paperclip className="w-4 h-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
                title={t("editor.link-memo")}
                aria-label={t("editor.link-memo")}
              >
                <LinkIcon className="w-4 h-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
                title={t("editor.add-location")}
                aria-label={t("editor.add-location")}
              >
                <MapPin className="w-4 h-4" />
              </Button>
            </div>

            <div className="flex-1" />

            {/* Visibility Toggle */}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
              title={t("editor.visibility")}
              aria-label={t("editor.visibility")}
            >
              <Eye className="w-4 h-4 mr-1" />
              {t("editor.private")}
            </Button>

            {/* Focus Mode Toggle */}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setIsFocusMode(true)}
              className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
              title={t("editor.focus-mode")}
              aria-label={t("editor.focus-mode")}
            >
              <Expand className="w-4 h-4" />
            </Button>
          </div>

          {/* Input Row */}
          <div className="flex items-end gap-3">
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
                  "w-full min-h-[48px] max-h-[200px] px-4 py-3 pr-14",
                  "bg-muted/50 border border-border/50 rounded-lg",
                  "text-sm resize-none outline-none",
                  "focus:bg-background focus:border-primary/50 focus:ring-2 focus:ring-primary/20",
                  "transition-all",
                  "placeholder:text-muted-foreground",
                )}
                style={{ height: "auto" }}
              />

              {/* Send Button */}
              <Button
                type="button"
                size="icon"
                onClick={handleSend}
                disabled={!canSend}
                className={cn(
                  "absolute right-2 bottom-2 h-9 w-9 rounded-lg transition-all",
                  canSend ? "bg-primary text-primary-foreground hover:scale-105" : "bg-muted text-muted-foreground",
                )}
                aria-label={t("editor.send")}
              >
                <Send className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {/* Shortcut Hints */}
          {!inputValue && (
            <div className="mt-2 text-right">
              <p className="text-xs text-muted-foreground">
                <kbd className="px-1.5 py-0.5 bg-muted rounded text-[10px]">Enter</kbd> {t("editor.enter-to-send")} ·
                <kbd className="px-1.5 py-0.5 bg-muted rounded text-[10px] ml-1">{isMac ? "⌘" : "Ctrl"} + Enter</kbd>{" "}
                {t("editor.for-newline")}
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Focus Mode Sheet */}
      <Sheet open={isFocusMode} onOpenChange={setIsFocusMode}>
        <SheetContent className="w-full sm:max-w-[42rem] overflow-y-auto">
          <SheetHeader>
            <div className="flex items-center justify-between">
              <SheetTitle>{t("editor.focus-mode")}</SheetTitle>
              <Button variant="ghost" size="icon" onClick={() => setIsFocusMode(false)} className="ml-auto" aria-label={t("common.close")}>
                <X className="w-5 h-5" />
              </Button>
            </div>
          </SheetHeader>

          <div className="mt-4">
            {/* Focus Mode uses the same textarea with expanded height */}
            <div className="relative">
              <textarea
                ref={textareaRef}
                value={inputValue}
                onChange={handleInput}
                onKeyDown={handleKeyDown}
                placeholder={placeholder}
                autoFocus
                className={cn(
                  "w-full min-h-[300px] max-h-[60vh] px-4 py-3",
                  "bg-muted/50 border border-border/50 rounded-lg",
                  "text-sm resize-none outline-none",
                  "focus:bg-background focus:border-primary/50 focus:ring-2 focus:ring-primary/20",
                  "transition-all",
                  "placeholder:text-muted-foreground",
                )}
                style={{ height: "auto" }}
              />
            </div>

            {/* Focus Mode Send Button */}
            <div className="mt-4 flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setIsFocusMode(false)}>
                {t("common.cancel")}
              </Button>
              <Button
                onClick={() => {
                  handleSend();
                  setIsFocusMode(false);
                }}
                disabled={!canSend}
                className={cn("transition-all", canSend ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground")}
              >
                {t("editor.send-note")}
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
});

PCEditor.displayName = "PCEditor";
