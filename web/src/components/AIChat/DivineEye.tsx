import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

/**
 * 神识之眼 - 模式感知的动态 Logo
 *
 * 根据不同模式显示不同的动效：
 * - 普通模式：柔和的呼吸效果
 * - 极客模式：数字脉冲 + 扫描线效果
 * - 进化模式：有机脉动 + DNA 双螺旋效果
 *
 * 性能优化：使用 CSS class 替代内联 filter，避免 CPU 渲染
 */
interface DivineEyeProps {
  mode: AIMode;
  className?: string;
  isActive?: boolean; // 是否正在活动（打字/思考中）
}

export function DivineEye({ mode, className, isActive = false }: DivineEyeProps) {
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
    <svg
      viewBox="0 0 200 200"
      className={cn(
        "h-8 w-8 transition-all duration-300",
        mode === "geek" && "divine-eye-geek-glow",
        mode === "evolution" && "divine-eye-evolution-glow",
        isActive && "divine-eye-active",
        className,
      )}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label={isActive ? `神识之眼 - ${mode}模式 - 活跃状态` : `神识之眼 - ${mode}模式`}
    >
      <title>
        {mode === "geek" ? "极客模式" : mode === "evolution" ? "进化模式" : "普通模式"} - {isActive ? "活跃" : "静态"}
      </title>
      <defs>
        {/* 普通模式渐变 */}
        <linearGradient id="eye_gradient_normal" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#06b6d4" />
          <stop offset="50%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#eab308" />
        </linearGradient>

        {/* 极客模式渐变 - 绿色终端风格 */}
        <linearGradient id="eye_gradient_geek" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#16a34a" />
          <stop offset="100%" stopColor="#4ade80" />
        </linearGradient>

        {/* 进化模式渐变 - 紫蓝风格 */}
        <linearGradient id="eye_gradient_evolution" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#a855f7" />
          <stop offset="50%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#4f46e5" />
        </linearGradient>

        {/* 发光滤镜 - 仅在静态时使用，活动时用 CSS 替代 */}
        <filter id="eye_glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur in="SourceGraphic" stdDeviation="2" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>

        {/* 极客模式扫描线渐变 */}
        <linearGradient id="geek_scan_gradient" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="transparent" />
          <stop offset="50%" stopColor="#22c55e" stopOpacity="0.4" />
          <stop offset="100%" stopColor="transparent" />
        </linearGradient>
      </defs>

      {/* 背景发光 - 活动状态增强（使用 CSS 替代内联 filter） */}
      {isActive && (
        <circle
          cx="100"
          cy="100"
          r="75"
          className={cn(
            "transition-all duration-300",
            mode === "geek" && "animate-[ping_2s_ease-in-out_infinite] divine-eye-glow-bg",
            mode === "evolution" && "animate-[pulse_2s_ease-in-out_infinite] divine-eye-glow-bg",
          )}
        />
      )}

      {/* 外圈 - 根据模式有不同动效 */}
      <ellipse
        cx="100"
        cy="100"
        rx="70"
        ry="70"
        className={cn(
          "transition-all duration-500",
          isActive && "scale-105",
          modeClasses.eye,
          mode === "geek"
            ? "stroke-[url(#eye_gradient_geek)]"
            : mode === "evolution"
              ? "stroke-[url(#eye_gradient_evolution)]"
              : "stroke-[url(#eye_gradient_normal)]",
          mode === "evolution" && "animate-[organicFlow_8s_ease-in-out_infinite]",
        )}
        strokeWidth="3"
        fill="none"
        filter="url(#eye_glow)"
        opacity="0.6"
      />

      {/* 极客模式 - 数字雨射线 + 扫描线 */}
      {mode === "geek" && (
        <g className={modeClasses.rays}>
          {/* 扫描线效果 */}
          {isActive && (
            <rect
              x="30"
              y="30"
              width="140"
              height="2"
              fill="url(#geek_scan_gradient)"
              className="animate-[scanlineMove_3s_linear_infinite]"
              opacity="0.6"
            />
          )}
          {/* 射线 - 优化：减少动画复杂度 */}
          {[...Array(8)].map((_, i) => (
            <line
              key={i}
              x1="100"
              y1="30"
              x2="100"
              y2="50"
              stroke="url(#eye_gradient_geek)"
              strokeWidth="2"
              opacity={isActive ? 0.7 : 0.4}
              transform={`rotate(${i * 45} 100 100)`}
              className={cn("origin-center", isActive && "animate-[pulse_1.5s_ease-in-out_infinite]")}
              style={isActive ? { animationDelay: `${i * 0.1}s` } : undefined}
            />
          ))}
        </g>
      )}

      {/* 进化模式 - DNA 双螺旋效果 */}
      {mode === "evolution" && (
        <g className={modeClasses.rays}>
          {/* DNA 双螺旋 - 优化：减少粒子数量 */}
          {[...Array(8)].map((_, i) => (
            <circle
              key={i}
              cx={100 + Math.cos((i * 45 * Math.PI) / 180) * 50}
              cy={100 + Math.sin((i * 45 * Math.PI) / 180) * 50}
              r="3"
              fill={i % 2 === 0 ? "url(#eye_gradient_evolution)" : "url(#eye_gradient_evolution)"}
              opacity={isActive ? 0.8 : 0.5}
              className={cn("origin-center", isActive && "animate-[dnaWave_3s_ease-in-out_infinite]")}
              style={isActive ? { animationDelay: `${i * 0.2}s` } : undefined}
            />
          ))}
        </g>
      )}

      {/* 瞳孔 - 根据模式有不同动效 */}
      <circle
        cx="100"
        cy="100"
        r={isActive ? 28 : 25}
        className={cn(
          "transition-all duration-300",
          modeClasses.pupil,
          mode === "geek" && isActive && "animate-[terminalCursor_1s_step-end_infinite]",
          mode === "geek"
            ? "fill-[url(#eye_gradient_geek)]"
            : mode === "evolution"
              ? "fill-[url(#eye_gradient_evolution)]"
              : "fill-[url(#eye_gradient_normal)]",
        )}
        filter="url(#eye_glow)"
      />

      {/* 中心亮点 */}
      <circle cx="100" cy="100" r={isActive ? 10 : 8} fill="white" opacity={isActive ? 1 : 0.8} className="transition-all duration-300" />

      {/* 进化模式额外光晕 - 优化：减少到 1 个 */}
      {mode === "evolution" && isActive && (
        <circle
          cx="100"
          cy="100"
          r="40"
          fill="none"
          stroke="url(#eye_gradient_evolution)"
          strokeWidth="1"
          opacity="0.3"
          className="animate-[ping_2s_ease-in-out_infinite]"
        />
      )}
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
  isActive?: boolean;
}

export function DivineEyeCompact({ mode, className, isActive = false }: DivineEyeCompactProps) {
  return (
    <div className={cn("relative h-6 w-6", className)}>
      {/* 外圈 */}
      <div
        className={cn(
          "absolute inset-0 rounded-full border-2 transition-all duration-500",
          mode === "geek" && "border-green-500",
          mode === "evolution" && "border-purple-500",
          mode === "normal" && "border-cyan-500",
          isActive && mode === "geek" && "animate-[ping_1.5s_ease-in-out_infinite]",
          isActive && mode === "evolution" && "animate-[pulse_2s_ease-in-out_infinite]",
        )}
      />

      {/* 瞳孔 */}
      <div
        className={cn(
          "absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full transition-all duration-300",
          mode === "geek" && "bg-green-500",
          mode === "evolution" && "bg-purple-500",
          mode === "normal" && "bg-cyan-500",
          isActive && "scale-125",
        )}
      />

      {/* 中心亮点 */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-1 rounded-full bg-white/80" />
    </div>
  );
}
