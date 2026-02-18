/**
 * MemoSkeleton - 便签色骨架屏
 *
 * 设计哲学：与 MemoBlockV3 便签风格一致
 * - 使用便签对应颜色的背景
 * - shimmer 动画效果
 * - 响应式双列布局
 */

import { memo } from "react";
import { cn } from "@/lib/utils";
import { getAvailableColors, getMemoColorClasses, StickyColorKey } from "@/utils/tag-colors";

// ============================================================================
// Types
// ============================================================================

export interface MemoSkeletonProps {
  colorKey?: StickyColorKey;
  className?: string;
}

export interface MemoSkeletonGridProps {
  count?: number;
  columns?: 1 | 2;
  className?: string;
}

// ============================================================================
// Single Skeleton Component
// ============================================================================

/**
 * MemoSkeleton - 单个便签骨架屏
 *
 * 使用便签颜色系统的背景色，保持视觉一致性
 */
export const MemoSkeleton = memo(function MemoSkeleton({ colorKey = "amber", className }: MemoSkeletonProps) {
  const colorClasses = getMemoColorClasses([colorKey]);

  return (
    <div className={cn("rounded-lg overflow-hidden", colorClasses.bg, colorClasses.border, "border", "animate-pulse", className)}>
      <div className="p-4 space-y-3">
        {/* Header: Avatar + Title */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-full bg-current/10 shrink-0" />
          <div className="flex-1 space-y-2">
            <div className="h-3 bg-current/10 rounded w-2/3" />
            <div className="h-2 bg-current/10 rounded w-1/3" />
          </div>
        </div>

        {/* Content lines */}
        <div className="space-y-2">
          <div className="h-2 bg-current/10 rounded w-full" />
          <div className="h-2 bg-current/10 rounded w-5/6" />
          <div className="h-2 bg-current/10 rounded w-4/6" />
        </div>

        {/* Tags */}
        <div className="flex gap-1.5">
          <div className="h-5 w-14 bg-current/10 rounded-full" />
          <div className="h-5 w-10 bg-current/10 rounded-full" />
        </div>

        {/* Footer */}
        <div className="flex justify-between pt-2 border-t border-current/10">
          <div className="h-3 w-12 bg-current/10 rounded" />
          <div className="flex gap-1">
            <div className="h-6 w-6 bg-current/10 rounded" />
            <div className="h-6 w-6 bg-current/10 rounded" />
          </div>
        </div>
      </div>
    </div>
  );
});

MemoSkeleton.displayName = "MemoSkeleton";

// ============================================================================
// Grid Skeleton Component
// ============================================================================

/**
 * MemoSkeletonGrid - 骨架屏网格
 *
 * 响应式布局，移动端单列，PC端双列
 * 使用随机便签颜色增加视觉丰富度
 */
export const MemoSkeletonGrid = memo(function MemoSkeletonGrid({ count = 6, columns, className }: MemoSkeletonGridProps) {
  const colors = getAvailableColors();

  // Generate random colors for each skeleton
  const skeletonColors = Array.from({ length: count }, (_, i) => colors[i % colors.length]);

  // Responsive columns
  const columnCount = columns ?? (typeof window !== "undefined" && window.innerWidth < 640 ? 1 : 2);

  if (columnCount === 1) {
    return (
      <div className={cn("flex flex-col gap-4", className)}>
        {skeletonColors.slice(0, Math.ceil(count / 2)).map((color, i) => (
          <MemoSkeleton key={i} colorKey={color} />
        ))}
      </div>
    );
  }

  // Distribute items into 2 columns
  const leftColumn = skeletonColors.filter((_, i) => i % 2 === 0);
  const rightColumn = skeletonColors.filter((_, i) => i % 2 === 1);

  return (
    <div className={cn("flex gap-4 w-full", className)}>
      <div className="flex-1 flex flex-col gap-4">
        {leftColumn.map((color, i) => (
          <MemoSkeleton key={i} colorKey={color} />
        ))}
      </div>
      <div className="flex-1 flex flex-col gap-4">
        {rightColumn.map((color, i) => (
          <MemoSkeleton key={i} colorKey={color} />
        ))}
      </div>
    </div>
  );
});

MemoSkeletonGrid.displayName = "MemoSkeletonGrid";

// ============================================================================
// Shimmer Effect CSS (inject once)
// ============================================================================

if (typeof document !== "undefined") {
  const styleId = "memo-skeleton-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      /* Shimmer effect for skeleton loading */
      @keyframes shimmer {
        0% {
          background-position: -200% 0;
        }
        100% {
          background-position: 200% 0;
        }
      }

      .skeleton-shimmer {
        background: linear-gradient(
          90deg,
          transparent 0%,
          rgba(255, 255, 255, 0.1) 50%,
          transparent 100%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
      }
    `;
    document.head.appendChild(style);
  }
}
