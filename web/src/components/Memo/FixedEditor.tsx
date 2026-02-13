/**
 * FixedEditor - 固定底部编辑器 v2.0
 *
 * 设计哲学：「禅意智识 · 意识之镜」
 * - 呼吸感：编辑器如「意识之镜」，随呼吸律动（3000ms 周期）
 * - 留白：充足的空间让思绪流淌
 * - 沉浸：聚焦当下，最小化干扰
 * - 温和：所有交互都有柔和的反馈
 * - 状态感知：视觉反馈随内容状态变化
 *
 * ## 设计规范
 * - 间距：--spacing-* 变量系统 (xs:4px, sm:8px, md:16px, lg:24px, xl:32px)
 * - 圆角：rounded-2xl (16px) - 与 HeroSection 统一
 * - 呼吸动画：3000ms 周期，与 logo-breathe-gentle 同步
 * - 光晕层次：三层渐变（外层扩散 → 中层聚焦 → 内层柔和）
 *
 * ## UX 改进 v2.0
 * - 内容状态指示（空/有内容/聚焦）
 * - 快捷键提示
 * - 发送按钮呼吸动画（有内容时）
 * - 玻璃态背景 + 渐变边框
 */

import { ChevronDown, Feather, ImagePlus, Paperclip, PenLine, Send, Sparkles } from "lucide-react";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import MemoEditor from "@/components/MemoEditor";
import { cn } from "@/lib/utils";

// ============================================================================
// 设计常量
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// 设计令牌 - 与 HeroSection 统一
const DESIGN_TOKENS = {
  borderRadius: "rounded-2xl", // 16px
  glow: {
    layers: 3, // 光晕层数
    baseOpacity: 0.03,
    focusedOpacity: 0.08,
  },
  animation: {
    duration: 300, // 过渡时长
    breath: BREATH_DURATION,
  },
} as const;

// ============================================================================
// 类型定义
// ============================================================================

export interface FixedEditorProps {
  placeholder?: string;
  className?: string;
}

type EditorState = "empty" | "hasContent" | "focused";

// ============================================================================
// 主组件
// ============================================================================

/**
 * FixedEditor - 固定底部编辑器
 *
 * 设计要点：
 * - 粘性定位，始终可见
 * - 三层光晕效果：聚焦时产生「意识场」
 * - 玻璃态背景 + 渐变边框
 * - 状态感知的视觉反馈
 * - 移动端键盘适配
 * - 收起/展开功能（v3.0）
 */
export const FixedEditor = memo(function FixedEditor({ placeholder, className }: FixedEditorProps) {
  const { t } = useTranslation();
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const [isFocused, setIsFocused] = useState(false);
  const [hasContent, setHasContent] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  // Collapsed state - persisted to localStorage
  const [isCollapsed, setIsCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    try {
      return localStorage.getItem("fixed-editor-collapsed") === "true";
    } catch {
      return false;
    }
  });

  // 计算编辑器状态
  const editorState: EditorState = isFocused ? "focused" : hasContent ? "hasContent" : "empty";

  // Toggle collapse
  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => {
      const newValue = !prev;
      try {
        localStorage.setItem("fixed-editor-collapsed", String(newValue));
      } catch {
        // ignore storage errors
      }
      return newValue;
    });
  }, []);

  // 移动端键盘高度处理
  useEffect(() => {
    if (typeof window === "undefined") return;

    const handleResize = () => {
      const currentHeight = window.visualViewport?.height ?? window.innerHeight;
      const windowHeight = window.innerHeight;
      const diff = windowHeight - currentHeight;

      if (diff > 50) {
        setKeyboardHeight(diff);
      } else if (diff < 10) {
        setKeyboardHeight(0);
      }
    };

    window.visualViewport?.addEventListener("resize", handleResize);
    return () => window.visualViewport?.removeEventListener("resize", handleResize);
  }, []);

  // 聚焦状态追踪
  const handleFocus = useCallback(() => {
    setIsFocused(true);
  }, []);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
  }, []);

  // 监听内容变化（通过 MutationObserver 或事件）
  useEffect(() => {
    const contentEl = contentRef.current;
    if (!contentEl) return;

    // 检测内容变化的函数
    const checkContent = () => {
      const textarea = contentEl.querySelector("textarea");
      if (textarea) {
        setHasContent(textarea.value.trim().length > 0);
      }
    };

    // 初始检查
    checkContent();

    // 监听 input 事件
    const handleInput = () => checkContent();
    contentEl.addEventListener("input", handleInput, true);

    return () => {
      contentEl.removeEventListener("input", handleInput, true);
    };
  }, []);

  // ═══════════════════════════════════════════════════════════════
  // Collapsed State - Mini Trigger
  // ═══════════════════════════════════════════════════════════════
  if (isCollapsed) {
    return (
      <div className={cn("sticky bottom-0 left-0 right-0 z-50 flex justify-center", className)}>
        {/* Mini Trigger Button */}
        <button
          onClick={toggleCollapse}
          className={cn(
            "flex items-center justify-center",
            "w-12 h-12 rounded-full",
            "bg-background/80 backdrop-blur-xl",
            "border-2 border-border/40",
            "shadow-lg shadow-primary/5",
            "text-muted-foreground hover:text-primary",
            "transition-all duration-300",
            "hover:border-primary/30 hover:shadow-primary/10",
            "active:scale-95",
          )}
          aria-label={t("editor.expand")}
          title={t("editor.expand")}
          style={{
            animationName: "breath-trigger",
            animationDuration: `${BREATH_DURATION}ms`,
            animationTimingFunction: "ease-in-out",
            animationIterationCount: "infinite",
          }}
        >
          <Sparkles className="w-5 h-5" />
        </button>
      </div>
    );
  }

  // ═══════════════════════════════════════════════════════════════
  // Expanded State - Full Editor
  // ═══════════════════════════════════════════════════════════════
  return (
    <div
      ref={containerRef}
      className={cn(
        "sticky bottom-0 left-0 right-0 z-50",
        // 顶部渐变边框 - 更精致的处理
        "before:absolute before:inset-x-0 before:top-0 before:h-px",
        "before:bg-gradient-to-r before:from-transparent before:via-border/60 before:to-transparent",
        // 背景 - 玻璃态
        "bg-background/80 backdrop-blur-xl",
        // 移动端键盘适配
        keyboardHeight > 0 && "pb-safe",
        className,
      )}
      style={{ paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : undefined }}
    >
      {/* ═══════════════════════════════════════════════════════════
          光晕系统 - 三层渐变
          - 外层：扩散光晕（大范围、低透明度）
          - 中层：聚焦光晕（中等范围、中透明度）
          - 内层：柔和光晕（小范围、高透明度）
          ═══════════════════════════════════════════════════════════ */}
      <div className="pointer-events-none absolute inset-0 overflow-hidden">
        {/* 外层扩散光晕 */}
        <div
          className={cn(
            "absolute -left-1/4 -right-1/4 -top-20 h-40",
            "bg-gradient-to-b from-primary/5 via-primary/3 to-transparent",
            "blur-2xl transition-opacity duration-500",
            editorState === "focused" ? "opacity-100" : "opacity-0",
          )}
          style={
            editorState === "focused"
              ? {
                  animationName: "breath-glow",
                  animationDuration: `${BREATH_DURATION}ms`,
                  animationTimingFunction: "ease-in-out",
                  animationIterationCount: "infinite",
                }
              : undefined
          }
        />

        {/* 中层聚焦光晕 - 居中 */}
        <div
          className={cn(
            "absolute left-1/2 -translate-x-1/2 -top-8",
            "w-3/4 h-24 rounded-full",
            "bg-primary/10 blur-xl",
            "transition-all duration-500",
            editorState === "focused" ? "opacity-100 scale-100" : "opacity-0 scale-95",
          )}
          style={
            editorState === "focused"
              ? {
                  animationName: "breath-glow",
                  animationDuration: `${BREATH_DURATION}ms`,
                  animationTimingFunction: "ease-in-out",
                  animationIterationCount: "infinite",
                  animationDelay: "500ms",
                }
              : undefined
          }
        />
      </div>

      {/* ═══════════════════════════════════════════════════════════
          内容容器 - 响应式宽度
          ═══════════════════════════════════════════════════════════ */}
      <div className="relative mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl px-4 sm:px-6 py-3 sm:py-4">
        {/* Collapse Button - Top Center */}
        <button
          onClick={toggleCollapse}
          className={cn(
            "absolute -top-3 left-1/2 -translate-x-1/2 z-10",
            "flex items-center justify-center",
            "w-8 h-6 rounded-full",
            "bg-background/80 backdrop-blur-sm border border-border/50",
            "text-muted-foreground hover:text-foreground",
            "transition-all duration-300",
            "hover:bg-muted/50 hover:border-border",
            "active:scale-95",
          )}
          aria-label={t("editor.collapse")}
          title={t("editor.collapse")}
        >
          <ChevronDown className="w-4 h-4" />
        </button>

        {/* 编辑器卡片 */}
        <div
          ref={contentRef}
          onFocus={handleFocus}
          onBlur={handleBlur}
          className={cn(
            "relative group",
            DESIGN_TOKENS.borderRadius,
            // 玻璃态背景
            "bg-background/60 backdrop-blur-md",
            // 边框 - 状态感知
            "border-2 transition-all duration-300",
            editorState === "focused"
              ? "border-primary/30 shadow-lg shadow-primary/5"
              : editorState === "hasContent"
                ? "border-primary/15 shadow-md"
                : "border-border/40 hover:border-border/60",
            // 阴影层次
            "shadow-sm",
          )}
        >
          {/* 顶部装饰线 - 状态指示 */}
          <div
            className={cn(
              "absolute top-0 left-1/2 -translate-x-1/2 -translate-y-px",
              "h-0.5 w-16 rounded-full transition-all duration-500",
              editorState === "focused"
                ? "bg-primary/50 w-24"
                : editorState === "hasContent"
                  ? "bg-primary/30 w-20"
                  : "bg-transparent w-12",
            )}
          />

          {/* 左侧图标 - 状态指示器 (仅 PC 端显示) */}
          <div
            className={cn(
              "hidden sm:flex",
              "absolute left-4 top-4",
              "items-center justify-center",
              "w-8 h-8 rounded-xl transition-all duration-300",
              editorState === "focused"
                ? "bg-primary/10 text-primary"
                : editorState === "hasContent"
                  ? "bg-primary/5 text-primary/70"
                  : "bg-muted/50 text-muted-foreground",
            )}
          >
            {editorState === "focused" ? (
              <PenLine className="w-4 h-4" />
            ) : editorState === "hasContent" ? (
              <Feather className="w-4 h-4" />
            ) : (
              <Sparkles className="w-4 h-4" />
            )}
          </div>

          {/* 编辑器主体 */}
          <div className="pl-3 sm:pl-14 pr-3 sm:pr-4 py-3 sm:py-4">
            <MemoEditor
              placeholder={placeholder || t("editor.any-thoughts")}
              onConfirm={() => {
                window.dispatchEvent(new Event("memo-created"));
                setHasContent(false);
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
});

FixedEditor.displayName = "FixedEditor";

// ============================================================================
// 全局动画定义
// ============================================================================

/**
 * 注入禅意动画关键帧
 */
if (typeof document !== "undefined") {
  const styleId = "fixed-editor-animations-v2";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      /* 呼吸光晕 - 多层同步 */
      @keyframes breath-glow {
        0%, 100% {
          opacity: 0.6;
          transform: scale(1);
        }
        50% {
          opacity: 1;
          transform: scale(1.02);
        }
      }

      /* 呼吸脉冲 - 指示器 */
      @keyframes breath-pulse {
        0%, 100% {
          opacity: 0.5;
          transform: scale(1);
        }
        50% {
          opacity: 1;
          transform: scale(1.2);
        }
      }

      /* 意识场光晕 - 保留兼容 */
      @keyframes consciousness-field {
        0%, 100% {
          opacity: 0.3;
          transform: translateX(-50%) scale(1);
        }
        50% {
          opacity: 0.6;
          transform: translateX(-50%) scale(1.05);
        }
      }

      /* 迷你触发器呼吸动画 - v3.0 */
      @keyframes breath-trigger {
        0%, 100% {
          box-shadow: 0 4px 12px rgba(var(--primary), 0.1);
        }
        50% {
          box-shadow: 0 4px 20px rgba(var(--primary), 0.2);
        }
      }
    `;
    document.head.appendChild(style);
  }
}

// ============================================================================
// 简化版编辑器组件 (可选，用于快速输入场景)
// ============================================================================

export interface QuickEditorProps {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  showAttachment?: boolean;
  showImage?: boolean;
}

/**
 * QuickEditor - 快速编辑器
 *
 * 用于需要更紧凑编辑器的场景
 * 保持禅意设计的同时减少视觉重量
 */
export const QuickEditor = memo(function QuickEditor({
  value,
  onChange,
  onSend,
  placeholder,
  disabled = false,
  showAttachment = true,
  showImage = true,
  className,
}: QuickEditorProps) {
  const { t } = useTranslation();
  const [isFocused, setIsFocused] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 自动调整高度
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = "auto";
    textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
  }, [value]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        onSend();
      }
    },
    [onSend],
  );

  const canSend = value.trim().length > 0 && !disabled;

  return (
    <div className={cn("relative", className)}>
      {/* 聚焦光晕 */}
      {isFocused && (
        <div
          className="absolute -inset-1 rounded-2xl bg-primary/10 blur-xl transition-opacity duration-300"
          style={{
            animationName: "quick-editor-glow",
            animationDuration: `${BREATH_DURATION}ms`,
            animationTimingFunction: "ease-in-out",
            animationIterationCount: "infinite",
          }}
        />
      )}

      {/* 主容器 */}
      <div
        className={cn(
          "relative flex items-end gap-3",
          "rounded-2xl border-2 bg-background/80 backdrop-blur-md",
          "transition-all duration-300",
          isFocused ? "border-primary/30 shadow-lg shadow-primary/5" : "border-border/50 shadow-sm",
        )}
      >
        {/* 左侧工具栏 */}
        {(showAttachment || showImage) && (
          <div className="flex gap-2 p-3 pb-3">
            {showAttachment && (
              <button
                type="button"
                className={cn(
                  "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl",
                  "text-muted-foreground transition-all duration-200",
                  "hover:bg-muted hover:text-foreground",
                  "active:scale-95",
                )}
                aria-label="添加附件"
              >
                <Paperclip className="h-4 w-4" />
              </button>
            )}
            {showImage && (
              <button
                type="button"
                className={cn(
                  "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl",
                  "text-muted-foreground transition-all duration-200",
                  "hover:bg-muted hover:text-foreground",
                  "active:scale-95",
                )}
                aria-label="添加图片"
              >
                <ImagePlus className="h-4 w-4" />
              </button>
            )}
          </div>
        )}

        {/* 输入区域 */}
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          placeholder={placeholder || t("editor.placeholder")}
          disabled={disabled}
          className={cn(
            "flex-1 min-h-[44px] max-h-[200px] py-3 resize-none",
            "bg-transparent text-base outline-none",
            "placeholder:text-muted-foreground/50",
            "transition-colors duration-200",
          )}
          rows={1}
        />

        {/* 发送按钮 */}
        <button
          type="button"
          onClick={onSend}
          disabled={!canSend}
          className={cn(
            "m-2 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
            "transition-all duration-200",
            "active:scale-95",
            canSend
              ? "bg-primary text-primary-foreground shadow-md shadow-primary/20 hover:bg-primary/90"
              : "bg-muted text-muted-foreground cursor-not-allowed opacity-50",
          )}
          aria-label="发送"
        >
          {canSend ? <Send className="h-4 w-4" /> : <Sparkles className="h-4 w-4 opacity-50" />}
        </button>
      </div>
    </div>
  );
});

QuickEditor.displayName = "QuickEditor";

// ============================================================================
// QuickEditor 动画补充
// ============================================================================

if (typeof document !== "undefined") {
  const styleId = "quick-editor-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      @keyframes quick-editor-glow {
        0%, 100% {
          opacity: 0.5;
        }
        50% {
          opacity: 0.8;
        }
      }
    `;
    document.head.appendChild(style);
  }
}
