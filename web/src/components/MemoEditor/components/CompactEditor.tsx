import { Link2, Paperclip, Send } from "lucide-react";
import { forwardRef, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useEditorContext } from "../state";
import type { CompactEditorProps } from "../types";
import type { EditorRefActions } from "../types/editor";

/**
 * CompactEditor - 移动端紧凑编辑器
 *
 * 设计特点：
 * - 动态高度 (44-120px)
 * - Enter 发送，Shift+Enter 换行
 * - 快捷附件和关联按钮
 * - 虚拟键盘适配
 * - 禅意智识风格
 */
export const CompactEditor = forwardRef<EditorRefActions, CompactEditorProps>(({ placeholder, onSave, onExpand, keyboardHeight }, ref) => {
  const { state, actions, dispatch } = useEditorContext();
  const [inputHeight, setInputHeight] = useState(44);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = (ref as React.RefObject<HTMLTextAreaElement>).current;
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
  }, [state.content, ref]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Enter to send, Shift+Enter for new line
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (state.content.trim()) {
        onSave?.();
      }
    }
  };

  const handleContentChange = (content: string) => {
    dispatch(actions.updateContent(content));
  };

  const handleAttachmentClick = () => {
    // Expand to show attachment options
    onExpand?.();
  };

  const handleRelationClick = () => {
    // Expand to show relation options
    onExpand?.();
  };

  const canSend = state.content.trim().length > 0;

  return (
    <div
      className={cn(
        "relative w-full flex items-end gap-2",
        "p-3 rounded-xl border-2 border-border/60 bg-card/80 backdrop-blur-xl",
        "transition-all duration-300",
      )}
      style={{
        paddingBottom: keyboardHeight && keyboardHeight > 0 ? `${keyboardHeight + 12}px` : undefined,
      }}
    >
      {/* Quick action buttons - attachment and relation */}
      <div className="flex items-center gap-1.5 pb-1 flex-shrink-0">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn("h-9 w-9 rounded-xl", "hover:bg-accent/50 hover:scale-105 active:scale-95", "transition-all duration-200")}
          onClick={handleAttachmentClick}
          aria-label="Add attachment"
        >
          <Paperclip className="w-4 h-4" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn("h-9 w-9 rounded-xl", "hover:bg-accent/50 hover:scale-105 active:scale-95", "transition-all duration-200")}
          onClick={handleRelationClick}
          aria-label="Link memo"
        >
          <Link2 className="w-4 h-4" />
        </Button>
      </div>

      {/* Text input area */}
      <div className="flex-1 min-w-0">
        <Textarea
          ref={ref as React.RefObject<HTMLTextAreaElement>}
          value={state.content}
          onChange={(e) => handleContentChange(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder ?? ""}
          className={cn(
            "min-h-[44px] max-h-[120px] py-2 px-3 resize-none",
            "border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0",
            "text-sm",
            "scrollbar-thin scrollbar-thumb-muted/30",
          )}
          style={{ height: `${inputHeight}px` }}
          rows={1}
        />
      </div>

      {/* Send button */}
      <Button
        type="button"
        size="sm"
        onClick={onSave}
        disabled={!canSend}
        className={cn(
          "h-9 min-w-[36px] rounded-lg p-0 flex-shrink-0",
          "transition-all duration-200",
          canSend ? "bg-primary text-primary-foreground hover:bg-primary/90 shadow-sm" : "bg-transparent text-muted-foreground",
        )}
      >
        <Send className="w-4 h-4" />
      </Button>

      {/* Hint text */}
      {state.content.length === 0 && (
        <div className="absolute -bottom-5 left-0 right-0 text-center">
          <span className="text-[10px] text-muted-foreground/60">Enter 发送 • Shift+Enter 换行</span>
        </div>
      )}
    </div>
  );
});

CompactEditor.displayName = "CompactEditor";
