/**
 * AIFormatButton - AI 格式化按钮
 *
 * 设计哲学：「禅意智识」
 * - 智能：一键调用 AI 格式化内容
 * - 美感：渐变背景突出 AI 功能
 * - 反馈：Loading 状态清晰可见
 */

import { Loader2, Wand2 } from "lucide-react";
import { type FC } from "react";
import { toast } from "react-hot-toast";
import { useFormatContent } from "@/hooks/useAIQueries";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";

interface AIFormatButtonProps {
  content: string;
  onFormat: (formattedContent: string) => void;
  disabled?: boolean;
  compact?: boolean;
}

export const AIFormatButton: FC<AIFormatButtonProps> = ({ content, onFormat, disabled = false, compact = false }) => {
  const t = useTranslate();
  const { mutate: formatContent, isPending } = useFormatContent();

  const isContentTooShort = content.trim().length < 10;

  const handleClick = () => {
    if (isContentTooShort || isPending) return;

    formatContent(
      { content },
      {
        onSuccess: (formatted) => {
          if (formatted) {
            onFormat(formatted);
            toast.success(t("editor.ai-format.success"));
          } else {
            toast.error(t("editor.ai-format.empty-result"));
          }
        },
        onError: (err) => {
          console.error("[AIFormatButton]", err);
          toast.error(t("editor.ai-format.error"));
        },
      },
    );
  };

  // Compact mode: icon only button
  if (compact) {
    return (
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled || isContentTooShort || isPending}
        className={cn(
          "h-9 w-9 rounded-xl flex items-center justify-center",
          "transition-all duration-200",
          "bg-gradient-to-r from-violet-500/10 to-purple-500/10",
          "border border-violet-500/20",
          "hover:from-violet-500/15 hover:to-purple-500/15",
          "active:scale-95",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
        title={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-format.tooltip")}
        aria-label={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-format.tooltip")}
        aria-disabled={disabled || isContentTooShort || isPending}
      >
        {isPending ? <Loader2 className="w-4 h-4 animate-spin text-violet-500" /> : <Wand2 className="w-4 h-4 text-violet-500" />}
      </button>
    );
  }

  // Full mode: button with label
  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={disabled || isContentTooShort || isPending}
      className={cn(
        "h-9 px-3 rounded-xl flex items-center gap-1.5",
        "transition-all duration-200",
        "bg-gradient-to-r from-violet-500/10 to-purple-500/10",
        "border border-violet-500/20",
        "hover:from-violet-500/15 hover:to-purple-500/15",
        "active:scale-95",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
      title={isContentTooShort ? t("editor.content-too-short") : t("editor.ai-format.tooltip")}
    >
      {isPending ? (
        <>
          <Loader2 className="w-4 h-4 animate-spin text-violet-500" />
          <span className="text-sm font-medium text-violet-500 hidden lg:inline">{t("editor.ai-format.formatting")}</span>
        </>
      ) : (
        <>
          <Wand2 className="w-4 h-4 text-violet-500" />
          <span className="text-sm font-medium text-violet-500 hidden lg:inline">{t("editor.ai-format.button")}</span>
        </>
      )}
    </button>
  );
};
