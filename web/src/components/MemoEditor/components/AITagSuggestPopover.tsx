/**
 * AITagSuggestPopover - AI 标签建议弹窗
 *
 * 设计哲学：「禅意智识」
 * - 微妙：触发器不干扰心流
 * - 智识：AI 建议以轻柔方式呈现
 * - 互动：标签选择有清晰的视觉反馈
 *
 * ## 功能保留
 * - AI 根据内容建议标签
 * - 过滤已存在的标签
 * - 多选支持
 * - 一键插入
 */

import { Check, Loader2, Sparkles } from "lucide-react";
import { type FC, useCallback, useState } from "react";
import { toast } from "react-hot-toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useSuggestTags } from "@/hooks/useAIQueries";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

interface AITagSuggestPopoverProps {
  content: string;
  onInsertTags: (tags: string[]) => void;
  disabled?: boolean;
}

/**
 * AI Tag Suggest Popover
 * - Calls AI to suggest tags based on content
 * - Filters out tags already present in content
 * - Allows user to select and insert tags
 */
export const AITagSuggestPopover: FC<AITagSuggestPopoverProps> = ({ content, onInsertTags, disabled }) => {
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
      // Fetch suggestions when opening
      suggestTags(
        { content },
        {
          onSuccess: (tags) => {
            const existingTags = getExistingTags(content);
            // Filter out tags already in content
            const newTags = tags.filter((tag) => !existingTags.has(tag.toLowerCase()));
            setSuggestedTags(newTags);
            // Auto-select all new tags
            setSelectedTags(new Set(newTags));
          },
          onError: (err) => {
            console.error("[AITagSuggest]", err);
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

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 text-muted-foreground hover:text-primary hover:bg-primary/5 transition-colors"
          disabled={disabled || isContentTooShort}
          title={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-suggest-tags")}
        >
          <Sparkles className="h-4 w-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-4">
        <div className="space-y-3">
          <h4 className="text-sm font-medium">{t("editor.ai-suggest-tags-title")}</h4>

          {isPending ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : suggestedTags.length === 0 ? (
            <p className="text-sm text-muted-foreground py-2">{t("editor.ai-suggest-tags-empty")}</p>
          ) : (
            <>
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
                      {isSelected && <Check className="h-3 w-3 ml-0.5" />}
                    </Badge>
                  );
                })}
              </div>

              <Button size="sm" className="w-full" onClick={handleInsert} disabled={selectedTags.size === 0}>
                {t("editor.ai-suggest-tags-insert")} ({selectedTags.size})
              </Button>
            </>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
};
