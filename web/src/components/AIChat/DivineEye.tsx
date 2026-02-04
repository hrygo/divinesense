import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

/**
 * 神识之眼 - 模式感知的动态 Logo
 *
 * 根据不同模式显示不同的动效：
 * - 普通模式：柔和的呼吸效果
 * - 极客模式：数字脉冲 + 扫描线效果
 * - 进化模式：有机脉动 + DNA 双螺旋效果
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
      className={cn("h-8 w-8", className)}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      style={{
        filter:
          mode === "geek"
            ? "drop-shadow(0 0 8px rgba(34, 197, 94, 0.5))"
            : mode === "evolution"
              ? "drop-shadow(0 0 8px rgba(168, 85, 247, 0.5))"
              : "drop-shadow(0 0 4px rgba(139, 92, 246, 0.3))",
      }}
    >
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

        {/* 极客模式扫描线渐变 */}
        <linearGradient id="geek_scan_gradient" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="transparent" />
          <stop offset="50%" stopColor="#22c55e" stopOpacity="0.6" />
          <stop offset="100%" stopColor="transparent" />
        </linearGradient>
      </defs>

      {/* 背景发光 - 活动状态增强 */}
      {isActive && (
        <circle
          cx="100"
          cy="100"
          r="75"
          className={cn(
            "transition-all duration-300",
            mode === "geek" && "animate-[ping_2s_ease-in-out_infinite]",
            mode === "evolution" && "animate-[pulse_2s_ease-in-out_infinite]",
          )}
          fill={mode === "geek" ? "rgba(34, 197, 94, 0.1)" : mode === "evolution" ? "rgba(168, 85, 247, 0.1)" : "rgba(139, 92, 246, 0.05)"}
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
          {/* 射线 */}
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
              style={{ animationDelay: `${i * 0.1}s` }}
            />
          ))}
        </g>
      )}

      {/* 进化模式 - DNA 双螺旋效果 */}
      {mode === "evolution" && (
        <g className={modeClasses.rays}>
          {/* DNA 双螺旋 */}
          {[...Array(12)].map((_, i) => (
            <circle
              key={i}
              cx={100 + Math.cos((i * 30 * Math.PI) / 180) * 50}
              cy={100 + Math.sin((i * 30 * Math.PI) / 180) * 50}
              r="3"
              fill={i % 2 === 0 ? "url(#eye_gradient_evolution)" : "url(#eye_gradient_evolution)"}
              opacity={isActive ? 0.8 : 0.5}
              className={cn("origin-center", isActive && "animate-[dnaWave_3s_ease-in-out_infinite]")}
              style={{ animationDelay: `${i * 0.15}s` }}
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

      {/* 进化模式额外光晕 */}
      {mode === "evolution" && isActive && (
        <>
          <circle
            cx="100"
            cy="100"
            r="35"
            fill="none"
            stroke="url(#eye_gradient_evolution)"
            strokeWidth="1"
            opacity="0.3"
            className="animate-[ping_2s_ease-in-out_infinite]"
          />
          <circle
            cx="100"
            cy="100"
            r="45"
            fill="none"
            stroke="url(#eye_gradient_evolution)"
            strokeWidth="1"
            opacity="0.2"
            className="animate-[ping_2s_ease-in-out_infinite]"
            style={{ animationDelay: "0.5s" }}
          />
        </>
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
        style={{
          boxShadow: isActive
            ? mode === "geek"
              ? "0 0 8px rgba(34, 197, 94, 0.5)"
              : mode === "evolution"
                ? "0 0 8px rgba(168, 85, 247, 0.5)"
                : "0 0 4px rgba(6, 182, 212, 0.3)"
            : "none",
        }}
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
