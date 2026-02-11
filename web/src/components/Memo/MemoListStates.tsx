/**
 * MemoListStates - 禅意加载态与空状态
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：与 logo-breathe-gentle 同步的韵律动画
 * - 留白：东方美学的空灵意境
 * - 意识流：思绪浮现的视觉隐喻
 * - 渐进：信息温和呈现，不打断心流
 *
 * ## 设计规范
 * - 间距：--spacing-* 变量系统 (xs:4px, sm:8px, md:16px, lg:24px, xl:32px)
 * - 圆角：--radius-* 变量系统 (sm:6px, md:8px, lg:12px, xl:16px)
 * - 呼吸动画：3000ms 周期，与 logo 同步
 */

import { BookOpen, Feather, Loader2, Sparkles } from "lucide-react";
import { memo, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

// ============================================================================
// 设计常量
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// ============================================================================
// 类型定义
// ============================================================================

export interface LoadingSkeletonProps {
  count?: number;
  className?: string;
}

export interface EmptyStateProps {
  type?: "all" | "filtered" | "search";
  className?: string;
}

export interface EndIndicatorProps {
  className?: string;
}

// ============================================================================
// 分页加载骨架屏 - 「思绪波动」
// ============================================================================

/**
 * ZenSkeletonLine - 禅意骨架线条
 *
 * 模拟思绪波动的微动画，而非机械的 shimmer
 */
const ZenSkeletonLine = memo(function ZenSkeletonLine({ width, delay = 0 }: { width: string; delay?: number }) {
  return (
    <div
      className={cn("h-4 rounded-full bg-gradient-to-r from-transparent via-muted/60 to-transparent", "animate-pulse")}
      style={{
        width,
        animationDelay: `${delay}ms`,
        animationDuration: `${1500 + Math.random() * 500}ms`,
      }}
    />
  );
});

/**
 * MemoCardSkeleton - 笔记卡片骨架
 *
 * 设计要点：
 * - 使用透明渐变而非硬边 shimmer
 * - 每条线条独立脉动，模拟自然呼吸
 * - 保持与 MemoBlockV2 相同的结构
 */
const MemoCardSkeleton = memo(function MemoCardSkeleton({ index = 0 }: { index?: number }) {
  return (
    <div
      className={cn("rounded-xl border border-border/30 bg-background/50 backdrop-blur-sm", "p-4 sm:p-5")}
      style={{
        animation: `subtle-fade-in ${0.6 + index * 0.1}s ease-out`,
      }}
    >
      {/* Header - 头部 */}
      <div className="mb-3 flex items-center justify-between">
        <div className="h-4 w-24 rounded-full bg-muted/40" />
        <div className="flex gap-2">
          <div className="h-4 w-4 rounded-full bg-muted/30" />
          <div className="h-4 w-4 rounded-full bg-muted/30" />
        </div>
      </div>

      {/* Content - 内容线条 */}
      <div className="space-y-3">
        <ZenSkeletonLine width="100%" delay={index * 100} />
        <ZenSkeletonLine width={index % 3 === 0 ? "85%" : "92%"} delay={index * 100 + 100} />
        {index % 2 === 0 && <ZenSkeletonLine width="64%" delay={index * 100 + 200} />}
      </div>
    </div>
  );
});

/**
 * PaginationSkeleton - 分页加载骨架
 *
 * 设计要点：
 * - 只显示 2 个卡片，暗示「正在加载更多」
 * - 使用更轻的视觉重量
 * - 底部有微妙的加载指示器
 */
export const PaginationSkeleton = memo(function PaginationSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("flex flex-col gap-4", className)}>
      {/* 骨架卡片 */}
      <MemoCardSkeleton index={0} />
      <MemoCardSkeleton index={1} />

      {/* 底部加载提示 */}
      <div className="flex items-center justify-center gap-2 py-4 text-muted-foreground/60">
        <Loader2 className="h-4 w-4 animate-spin" style={{ animationDuration: "2s" }} />
        <LoadingText />
      </div>
    </div>
  );
});

/**
 * LoadingSkeleton - 初始加载骨架
 *
 * 用于首次加载时的完整骨架屏
 */
export const LoadingSkeleton = memo(function LoadingSkeleton({ count = 4, className }: LoadingSkeletonProps) {
  return (
    <div className={cn("flex flex-col gap-4", className)}>
      {Array.from({ length: count }).map((_, index) => (
        <MemoCardSkeleton key={index} index={index} />
      ))}
    </div>
  );
});

// ============================================================================
// 空状态 - 「空灵意境」
// ============================================================================

/**
 * BreathingGlow - 呼吸光晕组件
 *
 * 创建与 logo 同步的呼吸效果
 */
const BreathingGlow = memo(function BreathingGlow({ size = "md" }: { size?: "sm" | "md" | "lg" }) {
  const sizeClasses = {
    sm: "h-16 w-16",
    md: "h-24 w-24",
    lg: "h-32 w-32",
  };

  return (
    <div className="relative">
      {/* 外层光晕 */}
      <div
        className={cn("absolute inset-0 rounded-full bg-primary/10 blur-xl", sizeClasses[size])}
        style={{
          animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite`,
        }}
      />
      {/* 中层光晕 */}
      <div
        className={cn(
          "absolute inset-0 rounded-full bg-primary/15 blur-md",
          size === "sm" ? "h-12 w-12" : size === "md" ? "h-20 w-20" : "h-28 w-28",
          "left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2",
        )}
        style={{
          animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite`,
          animationDelay: `${BREATH_DURATION / 3}ms`,
        }}
      />
    </div>
  );
});

/**
 * EmptyState - 空状态组件
 *
 * 设计要点：
 * - 避免空洞的「暂无数据」
 * - 提供禅意的视觉体验
 * - 鼓励用户开始记录
 * - 不同场景有不同文案
 */
export const EmptyState = memo(function EmptyState({ type = "all", className }: EmptyStateProps) {
  const { t } = useTranslation();
  const [hasAnimated, setHasAnimated] = useState(false);

  useEffect(() => {
    setHasAnimated(true);
  }, []);

  // 根据类型显示不同内容
  const content = useMemo(() => {
    switch (type) {
      case "search":
        return {
          icon: <Sparkles className="h-8 w-8 sm:h-10 sm:w-10" />,
          title: t("search.empty_title"),
          subtitle: t("search.empty_subtitle"),
        };
      case "filtered":
        return {
          icon: <Feather className="h-8 w-8 sm:h-10 sm:w-10" />,
          title: t("filter.empty_title"),
          subtitle: t("filter.empty_subtitle"),
        };
      default:
        return {
          icon: <BookOpen className="h-8 w-8 sm:h-10 sm:w-10" />,
          title: t("memo.empty_title"),
          subtitle: t("memo.empty_subtitle"),
        };
    }
  }, [type, t]);

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-16 sm:py-24",
        hasAnimated && "animate-in fade-in slide-in-from-bottom-4 duration-700",
        className,
      )}
    >
      {/* 呼吸光晕背景 */}
      <div className="relative mb-6">
        <BreathingGlow size="lg" />
        {/* 图标 */}
        <div className="relative z-10 flex h-32 w-32 items-center justify-center">
          <div
            className={cn("text-primary/70 transition-transform duration-1000", hasAnimated && "scale-100")}
            style={{
              animation: `gentle-float ${BREATH_DURATION}ms ease-in-out infinite`,
            }}
          >
            {content.icon}
          </div>
        </div>
      </div>

      {/* 文字内容 */}
      <div className="text-center space-y-2">
        <h3 className="text-base font-medium text-foreground/80 sm:text-lg">{content.title}</h3>
        <p className="text-sm text-muted-foreground/70 sm:text-base">{content.subtitle}</p>
      </div>
    </div>
  );
});

// ============================================================================
// 列表结束指示器 - 「止」的意境
// ============================================================================

/**
 * EndIndicator - 列表结束指示器
 *
 * 设计要点：
 * - 东方美学的「止」— 不是结束，而是停顿
 * - 使用圆点或短横线，而非长文字
 * - 微妙的呼吸动画
 * - 与整体设计融为一体
 */
export const EndIndicator = memo(function EndIndicator({ className }: EndIndicatorProps) {
  return (
    <div className={cn("flex items-center justify-center gap-3 py-8 sm:py-10", className)}>
      {/* 左侧装饰线 */}
      <div className="h-px w-8 bg-gradient-to-r from-transparent to-border/50" />

      {/* 中心元素 */}
      <div className="flex items-center gap-2 text-muted-foreground/50">
        {/* 三个小圆点，依次呼吸 */}
        <span
          className="h-1 w-1 rounded-full bg-current"
          style={{
            animation: `dot-breathe ${BREATH_DURATION}ms ease-in-out infinite`,
          }}
        />
        <span
          className="h-1 w-1 rounded-full bg-current"
          style={{
            animation: `dot-breathe ${BREATH_DURATION}ms ease-in-out infinite`,
            animationDelay: `${BREATH_DURATION / 3}ms`,
          }}
        />
        <span
          className="h-1 w-1 rounded-full bg-current"
          style={{
            animation: `dot-breathe ${BREATH_DURATION}ms ease-in-out infinite`,
            animationDelay: `${(BREATH_DURATION / 3) * 2}ms`,
          }}
        />
      </div>

      {/* 右侧装饰线 */}
      <div className="h-px w-8 bg-gradient-to-l from-transparent to-border/50" />
    </div>
  );
});

// ============================================================================
// 加载文字动画
// ============================================================================

/**
 * LoadingText - 循环文字动画
 *
 * 显示："正在加载" → "正在加载." → "正在加载.." → "正在加载..."
 */
const LoadingText = memo(function LoadingText() {
  const [dots, setDots] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setDots((prev) => (prev + 1) % 4);
    }, 500);
    return () => clearInterval(interval);
  }, []);

  return <span className="text-xs">正在加载{".".repeat(dots)}</span>;
});

// ============================================================================
// 全局动画定义
// ============================================================================

/**
 * 注入禅意动画关键帧
 */
if (typeof document !== "undefined") {
  const styleId = "memo-list-states-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      /* 呼吸动画 - 与 logo 同步 */
      @keyframes breathe {
        0%, 100% {
          opacity: 0.4;
          transform: scale(1);
        }
        50% {
          opacity: 0.8;
          transform: scale(1.08);
        }
      }

      /* 圆点呼吸 - 更轻柔 */
      @keyframes dot-breathe {
        0%, 100% {
          opacity: 0.3;
          transform: scale(0.8);
        }
        50% {
          opacity: 0.7;
          transform: scale(1.2);
        }
      }

      /* 轻柔浮动 */
      @keyframes gentle-float {
        0%, 100% {
          transform: translateY(0);
        }
        50% {
          transform: translateY(-6px);
        }
      }

      /* 微妙淡入 */
      @keyframes subtle-fade-in {
        from {
          opacity: 0;
          transform: translateY(8px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }
    `;
    document.head.appendChild(style);
  }
}
