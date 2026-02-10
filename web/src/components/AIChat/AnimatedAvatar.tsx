import { forwardRef, memo, useState } from "react";
import { cn } from "@/lib/utils";

interface AnimatedAvatarProps {
  src: string;
  alt?: string;
  size?: "sm" | "md" | "lg" | "xl";
  isThinking?: boolean;
  isTyping?: boolean;
  className?: string;
  onClick?: () => void;
}

// 性能优化：将 sizeClasses 提取到组件外部，避免每次渲染重新计算
const sizeClasses: Record<"sm" | "md" | "lg" | "xl", string> = {
  sm: "w-9 h-9",
  md: "w-9 h-9 md:w-10 md:h-10",
  lg: "w-12 h-12 md:w-14 md:h-14",
  xl: "w-16 h-16 md:w-20 md:h-20",
};

const imgSizeClasses: Record<"sm" | "md" | "lg" | "xl", string> = {
  sm: "w-8 h-8",
  md: "w-8 h-8 md:w-9 md:h-9",
  lg: "w-10 h-10 md:w-12 md:h-12",
  xl: "w-14 h-14 md:w-16 md:h-16",
};

/**
 * AnimatedAvatar - 带交互动效的头像组件
 *
 * 性能优化：
 * - 使用 memo 避免不必要的重新渲染
 * - 静态类名提取到组件外部
 * - CSS 动画优先于 JavaScript 动画
 * - 使用 will-change 仅在动画时
 *
 * 交互效果：
 * - Hover: 轻微放大、倾斜、光晕
 * - Thinking: 呼吸动画
 * - Typing: 波浪/脉冲效果
 */
export const AnimatedAvatar = memo(
  forwardRef<HTMLDivElement, AnimatedAvatarProps>(
    ({ src, alt, size = "md", isThinking = false, isTyping = false, className, onClick }, ref) => {
      const [isHovered, setIsHovered] = useState(false);

      // 性能优化：仅在有动画状态时应用 will-change
      const hasAnimation = isThinking || isTyping || isHovered;

      return (
        <div
          ref={ref}
          onClick={onClick}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          className={cn(
            "relative flex items-center justify-center overflow-hidden cursor-pointer rounded-full transition-all duration-300 ease-out",
            // 基础样式
            "shadow-sm",
            // Hover 效果
            isHovered && "scale-110 shadow-sm",
            // 思考状态 - 呼吸动画
            isThinking && "animate-avatar-breathe",
            // 打字状态 - 脉冲效果
            isTyping && "animate-avatar-pulse",
            // 性能优化：仅在动画时启用 GPU 加速
            hasAnimation && "will-change-transform",
            // 不同尺寸
            sizeClasses[size],
            className,
          )}
        >
          {/* 光晕背景层 */}
          <div
            className={cn(
              "absolute inset-0 bg-gradient-to-br from-primary/20 to-primary/5 opacity-0 transition-opacity duration-300",
              (isHovered || isThinking || isTyping) && "opacity-100",
            )}
          />

          {/* 波纹环 - 打字时显示 */}
          {isTyping && (
            <>
              <div className="absolute inset-0 rounded-lg border-2 border-primary/30 animate-avatar-wave" />
              <div className="absolute inset-0 rounded-lg border-2 border-primary/20 animate-avatar-wave delay-100" />
              <div className="absolute inset-0 rounded-lg border-2 border-primary/10 animate-avatar-wave delay-200" />
            </>
          )}

          {/* 头像图片 */}
          <img
            src={src}
            alt={alt}
            className={cn(
              "relative z-10 object-cover rounded-full transition-transform duration-300",
              imgSizeClasses[size],
              // Hover 时的 3D 倾斜效果
              isHovered && "animate-avatar-tilt",
            )}
          />

          {/* 状态指示器 */}
          {(isThinking || isTyping) && (
            <div className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-primary rounded-full border-2 border-background animate-pulse" />
          )}
        </div>
      );
    },
  ),
);

AnimatedAvatar.displayName = "AnimatedAvatar";
