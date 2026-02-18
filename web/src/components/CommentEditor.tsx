/**
 * CommentEditor - 极简评论编辑器
 *
 * 设计特点：
 * - 极简风格：只有文本框 + 发送按钮 + Emoji
 * - 动态高度 (44-120px)
 * - Ctrl/Cmd+Enter 发送
 * - 禅意智识风格
 */

import { Smile } from "lucide-react";
import { forwardRef, useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

// Common emojis for quick selection
const EMOJI_LIST = ["👍", "👎", "😄", "🎉", "❤️", "🔥", "💡", "🤔", "👀", "✅", "⭐", "🙏", "💪", "🚀", "👏", "😊"];

export interface CommentEditorProps {
  className?: string;
  placeholder?: string;
  autoFocus?: boolean;
  onSend: (content: string) => void | Promise<void>;
  onCancel?: () => void;
}

export const CommentEditor = forwardRef<HTMLTextAreaElement, CommentEditorProps>(
  ({ className, placeholder = "写下你的评论...", autoFocus = false, onSend, onCancel }, ref) => {
    const [content, setContent] = useState("");
    const [isSending, setIsSending] = useState(false);
    const [inputHeight, setInputHeight] = useState(44);
    const internalRef = useRef<HTMLTextAreaElement>(null);
    const textareaRef = (ref as React.RefObject<HTMLTextAreaElement>) || internalRef;

    // Auto-resize textarea
    useEffect(() => {
      const textarea = textareaRef.current;
      if (!textarea) return;

      const resize = () => {
        if (!textarea) return;
        textarea.style.height = "auto";
        const newHeight = Math.min(Math.max(textarea.scrollHeight, 44), 120);
        textarea.style.height = `${newHeight}px`;
        setInputHeight(newHeight);
      };

      resize();
      window.addEventListener("resize", resize);
      return () => window.removeEventListener("resize", resize);
    }, [content, textareaRef]);

    // Auto focus
    useEffect(() => {
      if (autoFocus && textareaRef.current) {
        textareaRef.current.focus();
      }
    }, [autoFocus, textareaRef]);

    const handleKeyDown = (e: React.KeyboardEvent) => {
      // Ctrl/Cmd + Enter to send
      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        if (content.trim() && !isSending) {
          handleSend();
        }
      }
      // Escape to cancel
      if (e.key === "Escape") {
        onCancel?.();
      }
    };

    const handleSend = useCallback(async () => {
      if (!content.trim() || isSending) return;

      setIsSending(true);
      try {
        await onSend(content.trim());
        setContent("");
        // Reset height
        if (textareaRef.current) {
          textareaRef.current.style.height = "44px";
          setInputHeight(44);
        }
      } finally {
        setIsSending(false);
      }
    }, [content, isSending, onSend, textareaRef]);

    const handleEmojiSelect = (emoji: string) => {
      const textarea = textareaRef.current;
      if (textarea) {
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const newContent = content.slice(0, start) + emoji + content.slice(end);
        setContent(newContent);
        // Move cursor after emoji
        setTimeout(() => {
          textarea.selectionStart = textarea.selectionEnd = start + emoji.length;
          textarea.focus();
        }, 0);
      } else {
        setContent(content + emoji);
      }
    };

    const canSend = content.trim().length > 0 && !isSending;

    return (
      <div
        className={cn(
          "relative w-full flex items-end gap-2",
          "p-3 rounded-xl border border-border bg-background",
          "transition-all duration-200",
          "focus-within:border-primary/30 focus-within:ring-1 focus-within:ring-primary/10",
          className,
        )}
      >
        {/* Emoji picker */}
        <Popover>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className={cn(
                "h-9 w-9 rounded-xl p-0 flex-shrink-0",
                "hover:bg-accent/50 hover:scale-105 active:scale-95",
                "transition-all duration-200",
              )}
              aria-label="Add emoji"
            >
              <Smile className="w-4 h-4 text-muted-foreground" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-64 p-2" align="start">
            <div className="grid grid-cols-8 gap-1">
              {EMOJI_LIST.map((emoji) => (
                <button
                  key={emoji}
                  type="button"
                  onClick={() => handleEmojiSelect(emoji)}
                  className="h-8 w-8 flex items-center justify-center rounded-lg hover:bg-accent text-lg transition-colors"
                >
                  {emoji}
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>

        {/* Text input area */}
        <div className="flex-1 min-w-0">
          <Textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            className={cn(
              "min-h-[44px] max-h-[120px] py-2.5 px-3 resize-none",
              "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
              "text-sm placeholder:text-muted-foreground/60",
              "scrollbar-thin scrollbar-thumb-muted/30",
            )}
            style={{ height: `${inputHeight}px` }}
            rows={1}
            disabled={isSending}
          />
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-1.5 pb-0.5 flex-shrink-0">
          {onCancel && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onCancel}
              disabled={isSending}
              className="h-9 px-3 rounded-lg text-muted-foreground hover:text-foreground"
            >
              取消
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            onClick={handleSend}
            disabled={!canSend}
            className={cn(
              "h-9 px-4 rounded-lg",
              "transition-all duration-200",
              canSend ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm" : "bg-transparent text-muted-foreground",
            )}
          >
            {isSending ? "发送中..." : "发送"}
          </Button>
        </div>

        {/* Hint text */}
        {content.length === 0 && (
          <div className="absolute -bottom-5 left-0 right-0 text-center pointer-events-none">
            <span className="text-[10px] text-muted-foreground/50">Ctrl+Enter 发送 • Esc 取消</span>
          </div>
        )}
      </div>
    );
  },
);

CommentEditor.displayName = "CommentEditor";
