/**
 * ExploreHeroSection - 探索页首屏
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：呼应 logo 的 gentle breathe 动效
 * - 留白：东方美学的空灵意境
 * - 简洁：只保留核心标题信息
 *
 * 设计规范：
 * - 间距：使用 --spacing-* 变量系统 (xs:4px, sm:8px, md:16px, lg:24px, xl:32px)
 * - 字体：base:16px, sm:14px, lg:18px, xl:20px, 2xl:24px, 3xl:30px
 * - 行高：snug:1.375, normal:1.5, relaxed:1.625
 */

import { Compass } from "lucide-react";
import { memo, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

export interface ExploreHeroSectionProps {
  className?: string;
}

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

/**
 * 探索页首屏组件 - 简洁版
 */
export const ExploreHeroSection = memo(function ExploreHeroSection({ className }: ExploreHeroSectionProps) {
  const { t } = useTranslation();
  const [hasAnimated, setHasAnimated] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);

  // Entry animation
  useEffect(() => {
    if (!hasAnimated) {
      setHasAnimated(true);
    }
  }, [hasAnimated]);

  return (
    <div className={cn("relative pb-8", className)} ref={containerRef}>
      {/* Ambient background - subtle "exploration field" */}
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        {/* Primary ambient glow - 右侧 */}
        <div
          className="absolute -right-32 -top-32 h-96 w-96 rounded-full bg-primary/2 blur-3xl"
          style={{ animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite` }}
        />
        {/* Secondary ambient glow - 左侧 */}
        <div
          className="absolute -left-24 top-32 h-64 w-64 rounded-full bg-accent/2 blur-3xl"
          style={{
            animation: `breathe ${BREATH_DURATION}ms ease-in-out infinite`,
            animationDelay: `${BREATH_DURATION / 2}ms`,
          }}
        />
      </div>

      {/* Main content */}
      <div className="space-y-4">
        {/* Page header - 禅意风格 */}
        <div
          className={cn("transition-all duration-1000 ease-out", hasAnimated ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-4")}
        >
          {/* Date with subtle accent */}
          <div className="mb-4 flex items-center gap-3">
            <div className="h-px flex-1 bg-gradient-to-r from-transparent to-border" />
            <div className="flex items-center gap-2 text-xs font-medium tracking-widest text-muted-foreground">
              <Compass className="h-3.5 w-3.5 text-primary" />
              <span>{t("explore.title")}</span>
            </div>
            <div className="h-px flex-1 bg-gradient-to-l from-transparent to-border" />
          </div>

          {/* Main title - 响应式字体大小 */}
          <h1 className="text-center font-semibold text-2xl sm:text-3xl lg:text-4xl xl:text-5xl text-foreground leading-snug">
            {t("explore.hero.title_prefix")} <span className="text-primary">{t("explore.hero.title_highlight")}</span>
          </h1>

          {/* Subtitle */}
          <p className="mx-auto max-w-[28rem] text-center text-sm text-muted-foreground transition-opacity duration-300">
            {t("explore.hero.subtitle")}
          </p>
        </div>
      </div>
    </div>
  );
});

ExploreHeroSection.displayName = "ExploreHeroSection";

/* ============================================================
   Global animations (syncs with logo)
   ============================================================ */

// Inject animations if not exists
if (typeof document !== "undefined") {
  const styleId = "explore-hero-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      @keyframes breathe {
        0%, 100% { opacity: 0.3; transform: scale(1); }
        50% { opacity: 0.6; transform: scale(1.05); }
      }
    `;
    document.head.appendChild(style);
  }
}
