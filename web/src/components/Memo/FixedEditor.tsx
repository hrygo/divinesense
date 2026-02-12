/**
 * FixedEditor - 固定底部编辑器
 *
 * 设计哲学：「禅意智识」
 * - 呼吸感：编辑器如「意识之镜」，随呼吸律动
 * - 留白：充足的空间让思绪流淌
 * - 沉浸：聚焦当下，最小化干扰
 * - 温和：所有交互都有柔和的反馈
 *
 * ## 设计规范
 * - 间距：--spacing-* 变量系统
 * - 圆角：--radius-* 变量系统
 * - 呼吸动画：3000ms 周期
 * - 边框光晕：聚焦时产生「意识场」效果
 */

import { ImagePlus, Paperclip, Send, Sparkles } from "lucide-react";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import MemoEditor from "@/components/MemoEditor";
import { cn } from "@/lib/utils";

// ============================================================================
// 设计常量
// ============================================================================

const BREATH_DURATION = 3000; // 与 logo-breathe-gentle 同步

// ============================================================================
// 类型定义
// ============================================================================

export interface FixedEditorProps {
  placeholder?: string;
  className?: string;
}

// ============================================================================
// 主组件
// ============================================================================

/**
 * FixedEditor - 固定底部编辑器
 *
 * 设计要点：
 * - 粘性定位，始终可见
 * - 聚焦时产生柔和的「意识场」光晕
 * - 工具栏渐进式展开
 * - 移动端键盘适配
 */
export const FixedEditor = memo(function FixedEditor({ placeholder, className }: FixedEditorProps) {
  const { t } = useTranslation();
  const [keyboardHeight, setKeyboardHeight] = useState(0);
  const [isFocused, setIsFocused] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const glowRef = useRef<HTMLDivElement>(null);

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

  return (
    <div
      ref={containerRef}
      className={cn(
        "sticky bottom-0 left-0 right-0 z-50",
        // 顶部边框渐变
        "border-t border-border/50",
        // 背景渐变
        "bg-gradient-to-b from-background/95 to-background",
        // 移动端键盘适配
        keyboardHeight > 0 && "pb-safe",
        className,
      )}
      style={{ paddingBottom: keyboardHeight > 0 ? `${keyboardHeight}px` : undefined }}
    >
      {/* 意识场光晕 - 聚焦时显示 */}
      <div
        ref={glowRef}
        className={cn(
          "pointer-events-none absolute bottom-0 left-0 right-0",
          "h-24 mx-auto rounded-t-full",
          "bg-primary/3 blur-3xl",
          "transition-opacity duration-500",
          isFocused ? "opacity-100" : "opacity-0",
        )}
        style={{
          animation: isFocused ? `consciousness-field ${BREATH_DURATION}ms ease-in-out infinite` : undefined,
        }}
      />

      {/* 内容容器 - 响应式宽度 */}
      <div className={cn("mx-auto max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl", "px-4 sm:px-6", "py-3 sm:py-4")}>
        {/* 编辑器包装 */}
        <div className="relative" onFocus={handleFocus} onBlur={handleBlur}>
          {/* 内边框 - 聚焦时出现 */}
          <div
            className={cn(
              "absolute inset-0 rounded-2xl",
              "border-2 border-transparent transition-colors duration-300",
              isFocused && "border-primary/20",
            )}
          />

          {/* MemoEditor */}
          <MemoEditor
            placeholder={placeholder || t("editor.any-thoughts")}
            onConfirm={() => {
              window.dispatchEvent(new Event("memo-created"));
            }}
          />
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
  const styleId = "fixed-editor-animations";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = `
      /* 意识场光晕 - 与呼吸同步 */
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
            animation: `quick-editor-glow ${BREATH_DURATION}ms ease-in-out infinite`,
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
