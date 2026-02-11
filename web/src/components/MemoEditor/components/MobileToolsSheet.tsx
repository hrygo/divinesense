import { ChevronRight, Image, Link2, MapPin, Paperclip } from "lucide-react";
import type { FC } from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { Visibility } from "@/types/proto/api/v1/memo_service_pb";
import type { MobileToolsSheetProps } from "../types";

/**
 * MobileToolsSheet - 移动端展开工具面板
 *
 * 设计特点：
 * - Sheet 组件 + 4列工具网格
 * - 高级选项（可见性、关联笔记）
 * - 虚拟键盘适配
 * - 禅意智识风格
 */
export const MobileToolsSheet: FC<MobileToolsSheetProps> = ({
  open,
  onOpenChange,
  onUploadFile,
  onLinkMemo,
  onAddLocation,
  onVisibilityChange,
  keyboardHeight,
}) => {
  const toolItems = [
    {
      icon: Image,
      label: "图片",
      description: "添加图片",
      onClick: () => {
        onUploadFile();
        onOpenChange(false);
      },
    },
    {
      icon: Paperclip,
      label: "文件",
      description: "上传附件",
      onClick: () => {
        onUploadFile();
        onOpenChange(false);
      },
    },
    {
      icon: Link2,
      label: "关联",
      description: "关联笔记",
      onClick: () => {
        onLinkMemo();
        onOpenChange(false);
      },
    },
    {
      icon: MapPin,
      label: "位置",
      description: "添加位置",
      onClick: () => {
        onAddLocation();
        onOpenChange(false);
      },
    },
  ];

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="bottom"
        className={cn(
          "rounded-t-3xl border-t-0",
          "pb-8 pt-6 px-6",
          keyboardHeight && keyboardHeight > 0 ? `pb-[${keyboardHeight + 32}px]` : "",
        )}
        style={{
          paddingBottom: keyboardHeight && keyboardHeight > 0 ? `${keyboardHeight + 32}px` : undefined,
        }}
      >
        <SheetHeader className="mb-6">
          <SheetTitle className="text-center font-medium">工具</SheetTitle>
        </SheetHeader>

        {/* Tool grid */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          {toolItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                key={item.label}
                type="button"
                onClick={item.onClick}
                className={cn(
                  "flex flex-col items-center gap-2",
                  "p-3 rounded-2xl bg-muted/30",
                  "hover:bg-accent/50 hover:scale-105 active:scale-95",
                  "transition-all duration-200",
                )}
              >
                <div className={cn("h-12 w-12 rounded-xl", "bg-primary/10 flex items-center justify-center")}>
                  <Icon className="w-6 h-6 text-primary" />
                </div>
                <span className="text-xs text-foreground/70">{item.label}</span>
              </button>
            );
          })}
        </div>

        {/* Visibility option */}
        <div className="space-y-3">
          <button
            type="button"
            onClick={() => {
              onVisibilityChange(Visibility.PRIVATE);
              onOpenChange(false);
            }}
            className={cn(
              "w-full flex items-center justify-between",
              "p-4 rounded-xl bg-muted/30",
              "hover:bg-accent/50 active:scale-[0.98]",
              "transition-all duration-200",
            )}
          >
            <div className="flex items-center gap-3">
              <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
                <span className="text-lg">👁️</span>
              </div>
              <span className="text-sm font-medium">可见性</span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <span className="text-sm">Private</span>
              <ChevronRight className="w-4 h-4" />
            </div>
          </button>
        </div>
      </SheetContent>
    </Sheet>
  );
};

MobileToolsSheet.displayName = "MobileToolsSheet";
