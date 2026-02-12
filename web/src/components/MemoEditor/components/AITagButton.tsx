/**
 * AITagButton - AI 标签提取按钮
 *
 * 设计哲学：「禅意智识」
 * - 智能：一键调用 AI 分析内容
 * - 美感：渐变背景突出 AI 功能
 * - 反馈：Loading 状态清晰可见
 */

import { Loader2, Sparkles } from "lucide-react";
import { type FC, useCallback, useState } from "react";
import { toast } from "react-hot-toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSuggestTags } from "@/hooks/useAIQueries";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

interface AITagButtonProps {
  content: string;
  onInsertTags: (tags: string[]) => void;
  disabled?: boolean;
  compact?: boolean;
}

export const AITagButton: FC<AITagButtonProps> = ({ content, onInsertTags, disabled = false, compact = false }) => {
  const t = useTranslate();
  const [open, setOpen] = useState(false);
  const [suggestedTags, setSuggestedTags] = useState<string[]>([]);
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());
  const { mutate: suggestTags, isPending } = useSuggestTags();

  // Extract existing tags from content
  const getExistingTags = useCallback((text: string): Set<string> => {
    const tagRegex = /#([^\s#]+)/g;
    const tags = new Set<string>();
    let match;
    while ((match = tagRegex.exec(text)) !== null) {
      tags.add(match[1].toLowerCase());
    }
    return tags;
  }, []);

  const handleOpenChange = (isOpen: boolean) => {
    setOpen(isOpen);
    if (isOpen && content.length >= 5) {
      suggestTags(
        { content },
        {
          onSuccess: (tags) => {
            const existingTags = getExistingTags(content);
            const newTags = tags.filter((tag) => !existingTags.has(tag.toLowerCase()));
            setSuggestedTags(newTags);
            setSelectedTags(new Set(newTags));
          },
          onError: (err) => {
            console.error("[AITagButton]", err);
            toast.error(t("editor.ai-suggest-tags-error"));
            setOpen(false);
          },
        },
      );
    }
  };

  const toggleTag = (tag: string) => {
    setSelectedTags((prev) => {
      const next = new Set(prev);
      if (next.has(tag)) {
        next.delete(tag);
      } else {
        next.add(tag);
      }
      return next;
    });
  };

  const handleInsert = () => {
    if (selectedTags.size > 0) {
      onInsertTags(Array.from(selectedTags));
      toast.success(t("editor.ai-suggest-tags-inserted"));
    }
    setOpen(false);
    setSuggestedTags([]);
    setSelectedTags(new Set());
  };

  const isContentTooShort = content.length < 5;

  // Compact mode: icon only button
  if (compact) {
    return (
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={disabled || isContentTooShort}
            className={cn(
              "h-9 w-9 rounded-xl flex items-center justify-center",
              "transition-all duration-200",
              "bg-gradient-to-r from-violet-500/10 to-purple-500/10",
              "border border-violet-500/20",
              "hover:from-violet-500/15 hover:to-purple-500/15",
              "active:scale-95",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
            title={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-tag.button")}
            aria-label={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-tag.button")}
            aria-disabled={disabled || isContentTooShort}
            aria-haspopup="dialog"
            aria-expanded={open}
          >
            {isPending ? <Loader2 className="w-4 h-4 animate-spin text-violet-500" /> : <Sparkles className="w-4 h-4 text-violet-500" />}
          </button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-64 p-4">
          <TagSelectionContent
            isPending={isPending}
            suggestedTags={suggestedTags}
            selectedTags={selectedTags}
            toggleTag={toggleTag}
            handleInsert={handleInsert}
          />
        </PopoverContent>
      </Popover>
    );
  }

  // Full mode: button with label
  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button
          type="button"
          disabled={disabled || isContentTooShort}
          className={cn(
            "h-9 px-3 rounded-xl flex items-center gap-1.5",
            "transition-all duration-200",
            "bg-gradient-to-r from-violet-500/10 to-purple-500/10",
            "border border-violet-500/20",
            "hover:from-violet-500/15 hover:to-purple-500/15",
            "active:scale-95",
            "disabled:opacity-50 disabled:cursor-not-allowed",
          )}
          title={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-tag.tooltip")}
        >
          {isPending ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin text-violet-500" />
              <span className="text-sm font-medium text-violet-500 hidden lg:inline">{t("editor.ai-tag.extracting")}</span>
            </>
          ) : (
            <>
              <Sparkles className="w-4 h-4 text-violet-500" />
              <span className="text-sm font-medium text-violet-500 hidden lg:inline">{t("editor.ai-tag.button")}</span>
            </>
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-4">
        <TagSelectionContent
          isPending={isPending}
          suggestedTags={suggestedTags}
          selectedTags={selectedTags}
          toggleTag={toggleTag}
          handleInsert={handleInsert}
        />
      </PopoverContent>
    </Popover>
  );
};

// ============================================================================
// Tag Selection Content (shared between modes)
// ============================================================================

interface TagSelectionContentProps {
  isPending: boolean;
  suggestedTags: string[];
  selectedTags: Set<string>;
  toggleTag: (tag: string) => void;
  handleInsert: () => void;
}

const TagSelectionContent: FC<TagSelectionContentProps> = ({ isPending, suggestedTags, selectedTags, toggleTag, handleInsert }) => {
  const t = useTranslate();

  if (isPending) {
    return (
      <div className="flex items-center justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (suggestedTags.length === 0) {
    return <p className="text-sm text-muted-foreground py-2">{t("editor.ai-suggest-tags-empty")}</p>;
  }

  return (
    <div className="space-y-3">
      <h4 className="text-sm font-medium">{t("editor.ai-suggest-tags-title")}</h4>
      <div className="flex flex-wrap gap-2">
        {suggestedTags.map((tag) => {
          const isSelected = selectedTags.has(tag);
          return (
            <Badge
              key={tag}
              role="button"
              tabIndex={0}
              variant={isSelected ? "default" : "outline"}
              className={cn("cursor-pointer select-none transition-all", "hover:scale-105 active:scale-95", isSelected && "pr-1")}
              onClick={() => toggleTag(tag)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  toggleTag(tag);
                }
              }}
            >
              #{tag}
              {isSelected && <span className="ml-0.5">✓</span>}
            </Badge>
          );
        })}
      </div>
      <Button size="sm" className="w-full" onClick={handleInsert} disabled={selectedTags.size === 0}>
        {t("editor.ai-suggest-tags-insert")} ({selectedTags.size})
      </Button>
    </div>
  );
};
