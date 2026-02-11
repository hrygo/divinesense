/**
 * VisibilitySelector - 禅意可见性选择器
 *
 * 设计哲学：「禅意智识」
 * - 微妙：触发器轻如鸿毛
 * - 清晰：每个选项都有明确图标和状态
 * - 和谐：与整体设计语言保持一致
 *
 * ## 功能保留
 * - 私有 (PRIVATE)
 * - 工作区 (PROTECTED)
 * - 公开 (PUBLIC)
 */

import { Check, ChevronDown } from "lucide-react";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import VisibilityIcon from "@/components/VisibilityIcon";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import { useTranslate } from "@/utils/i18n";
import type { VisibilitySelectorProps } from "../types";

const VisibilitySelector = (props: VisibilitySelectorProps) => {
  const { value, onChange } = props;
  const t = useTranslate();

  const visibilityOptions = [
    { value: Visibility.PRIVATE, label: t("memo.visibility.private") },
    { value: Visibility.PROTECTED, label: t("memo.visibility.protected") },
    { value: Visibility.PUBLIC, label: t("memo.visibility.public") },
  ] as const;

  const currentLabel = visibilityOptions.find((option) => option.value === value)?.label || "";

  return (
    <DropdownMenu onOpenChange={props.onOpenChange}>
      <DropdownMenuTrigger asChild>
        <button className="inline-flex h-9 items-center gap-2 px-3 text-sm text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-xl transition-all cursor-pointer">
          <VisibilityIcon visibility={value} className="h-4 w-4 opacity-60" />
          <span className="whitespace-nowrap">{currentLabel}</span>
          <ChevronDown className="h-4 w-4 opacity-40 transition-transform ui-open:rotate-180" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {visibilityOptions.map((option) => {
          const isSelected = value === option.value;
          return (
            <DropdownMenuItem key={option.value} className="cursor-pointer gap-3" onClick={() => onChange(option.value)}>
              <VisibilityIcon visibility={option.value} className="h-4 w-4 text-muted-foreground" />
              <span className="flex-1">{option.label}</span>
              {isSelected && <Check className="h-4 w-4 text-primary" />}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};

export default VisibilitySelector;
