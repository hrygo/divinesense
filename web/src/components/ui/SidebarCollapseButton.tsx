/**
 * SidebarCollapseButton - 侧边栏收起按钮
 *
 * 设计哲学：「禅意智识 · 意识之镜」
 * - 简约：仅图标，无多余装饰
 * - 温和：柔和的过渡动画
 * - 状态感知：视觉反馈随状态变化
 *
 * 位置：侧边栏右边缘，垂直居中
 */

import { ChevronLeft } from "lucide-react";
import { memo } from "react";
import { cn } from "@/lib/utils";

// ============================================================================
// 类型定义
// ============================================================================

export interface SidebarCollapseButtonProps {
  /** 侧边栏是否展开 */
  isExpanded: boolean;
  /** 切换侧边栏状态 */
  onToggle: () => void;
  /** 额外的 className */
  className?: string;
  /** aria-label for accessibility */
  expandLabel?: string;
  collapseLabel?: string;
}

// ============================================================================
// 主组件
// ============================================================================

export const SidebarCollapseButton = memo(function SidebarCollapseButton({
  isExpanded,
  onToggle,
  className,
  expandLabel = "Expand sidebar",
  collapseLabel = "Collapse sidebar",
}: SidebarCollapseButtonProps) {
  return (
    <button
      onClick={onToggle}
      className={cn(
        // Positioning - fixed at sidebar right edge, vertically centered
        // left-16 (64px) is the nav bar width
        // w-80 (320px) is the sidebar width
        // When expanded: left-[calc(4rem+320px-1px)] to sit at right edge
        // When collapsed: left-[calc(4rem-1px)] to sit at nav bar right edge
        "fixed top-1/2 -translate-y-1/2 z-40",
        "transition-all duration-300 ease-out",
        isExpanded ? "left-[calc(4rem+320px-1px)]" : "left-[calc(4rem-1px)]",
        // Size and shape
        "w-6 h-12 flex items-center justify-center",
        // Background - subtle glass effect
        "bg-background/80 backdrop-blur-sm",
        // Border - rounded right side only
        "border border-l-0 border-border/50 rounded-r-lg",
        // Shadow
        "shadow-sm hover:shadow-md",
        // Text color
        "text-muted-foreground hover:text-foreground",
        // Hover and active states
        "hover:bg-muted/50 active:scale-95",
        // Focus visible for accessibility
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50",
        className,
      )}
      aria-label={isExpanded ? collapseLabel : expandLabel}
      title={isExpanded ? collapseLabel : expandLabel}
    >
      {/* Icon with rotation animation */}
      <div className={cn("transition-transform duration-300 ease-out", isExpanded ? "rotate-0" : "rotate-180")}>
        <ChevronLeft className="w-4 h-4" />
      </div>
    </button>
  );
});

SidebarCollapseButton.displayName = "SidebarCollapseButton";
