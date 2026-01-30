import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

/**
 * 神识之眼 - 模式感知的动态 Logo
 *
 * 根据不同模式显示不同的动效：
 * - 普通模式：柔和的呼吸效果
 * - 极客模式：数字脉冲效果
 * - 进化模式：有机脉动效果
 */
interface DivineEyeProps {
  mode: AIMode;
  className?: string;
}

export function DivineEye({ mode, className }: DivineEyeProps) {
  const getModeClasses = () => {
    switch (mode) {
      case "geek":
        return {
          eye: "divine-eye-geek",
          pupil: "divine-pupil-geek",
          rays: "divine-rays-geek",
        };
      case "evolution":
        return {
          eye: "divine-eye-evolution",
          pupil: "divine-pupil-evolution",
          rays: "divine-rays-evolution",
        };
      default:
        return {
          eye: "divine-eye-normal",
          pupil: "divine-pupil-normal",
          rays: "divine-rays-normal",
        };
    }
  };

  const modeClasses = getModeClasses();

  return (
    <svg viewBox="0 0 200 200" className={cn("h-8 w-8", className)} fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        {/* 普通模式渐变 */}
        <linearGradient id="eye_gradient_normal" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#06b6d4" />
          <stop offset="50%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#eab308" />
        </linearGradient>

        {/* 极客模式渐变 - 绿色终端风格 */}
        <linearGradient id="eye_gradient_geek" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#22c55e" />
          <stop offset="100%" stopColor="#4ade80" />
        </linearGradient>

        {/* 进化模式渐变 - 紫蓝风格 */}
        <linearGradient id="eye_gradient_evolution" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#bc13fe" />
          <stop offset="50%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#4d4dff" />
        </linearGradient>

        {/* 发光滤镜 */}
        <filter id="eye_glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      {/* 外圈 - 根据模式有不同动效 */}
      <ellipse
        cx="100"
        cy="100"
        rx="70"
        ry="70"
        className={cn(
          "transition-all duration-500",
          modeClasses.eye,
          mode === "geek"
            ? "stroke-[url(#eye_gradient_geek)]"
            : mode === "evolution"
              ? "stroke-[url(#eye_gradient_evolution)]"
              : "stroke-[url(#eye_gradient_normal)]",
        )}
        strokeWidth="3"
        fill="none"
        filter="url(#eye_glow)"
        opacity="0.6"
      />

      {/* 极客模式 - 数字雨射线 */}
      {mode === "geek" && (
        <g className={modeClasses.rays}>
          {[...Array(8)].map((_, i) => (
            <line
              key={i}
              x1="100"
              y1="30"
              x2="100"
              y2="50"
              stroke="url(#eye_gradient_geek)"
              strokeWidth="2"
              opacity="0.4"
              transform={`rotate(${i * 45} 100 100)`}
              className="origin-center"
            />
          ))}
        </g>
      )}

      {/* 进化模式 - 有机触须 */}
      {mode === "evolution" && (
        <g className={modeClasses.rays}>
          {[...Array(6)].map((_, i) => (
            <path
              key={i}
              d={`M100,30 Q${100 + Math.sin(i * 60) * 20},50 ${100 + Math.cos(i * 60) * 30},70`}
              stroke="url(#eye_gradient_evolution)"
              strokeWidth="2"
              fill="none"
              opacity="0.5"
              transform={`rotate(${i * 60} 100 100)`}
              className="origin-center"
            />
          ))}
        </g>
      )}

      {/* 瞳孔 - 根据模式有不同动效 */}
      <circle
        cx="100"
        cy="100"
        r="25"
        className={cn(
          "transition-all duration-300",
          modeClasses.pupil,
          mode === "geek"
            ? "fill-[url(#eye_gradient_geek)]"
            : mode === "evolution"
              ? "fill-[url(#eye_gradient_evolution)]"
              : "fill-[url(#eye_gradient_normal)]",
        )}
        filter="url(#eye_glow)"
      />

      {/* 中心亮点 */}
      <circle cx="100" cy="100" r="8" fill="white" opacity="0.8" />

      {/* 四角小圆点 */}
      <circle cx="30" cy="100" r="5" className={modeClasses.pupil} opacity="0.5" />
      <circle cx="170" cy="100" r="5" className={modeClasses.pupil} opacity="0.5" />
      <circle cx="100" cy="30" r="5" className={modeClasses.pupil} opacity="0.5" />
      <circle cx="100" cy="170" r="5" className={modeClasses.pupil} opacity="0.5" />
    </svg>
  );
}

/**
 * 神识之眼 - 紧凑版（用于移动端头部）
 * 替代原来的呼吸灯小点
 */
interface DivineEyeCompactProps {
  mode: AIMode;
  className?: string;
}

export function DivineEyeCompact({ mode, className }: DivineEyeCompactProps) {
  return (
    <div className={cn("relative h-6 w-6", className)}>
      {/* 外圈 */}
      <div
        className={cn(
          "absolute inset-0 rounded-full border-2 transition-all duration-500",
          mode === "geek" && "border-green-500 divine-eye-compact-geek",
          mode === "evolution" && "border-purple-500 divine-eye-compact-evolution",
          mode === "normal" && "border-cyan-500 divine-eye-compact-normal",
        )}
      />

      {/* 瞳孔 */}
      <div
        className={cn(
          "absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full transition-all duration-300",
          mode === "geek" && "bg-green-500 divine-pupil-compact-geek",
          mode === "evolution" && "bg-purple-500 divine-pupil-compact-evolution",
          mode === "normal" && "bg-cyan-500 divine-pupil-compact-normal",
        )}
      />

      {/* 中心亮点 */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-1 rounded-full bg-white/80" />
    </div>
  );
}
