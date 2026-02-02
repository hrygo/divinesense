import { cn } from "@/lib/utils";

interface AuthSkeletonProps {
  className?: string;
}

/** 骨架屏 - 用于登录/注册页面加载状态，避免视觉闪动 */
export function AuthSkeleton({ className }: AuthSkeletonProps) {
  return (
    <div className={cn("w-full flex flex-col gap-4", className)}>
      {/* Logo/title skeleton */}
      <div className="w-full flex justify-center items-center gap-2 mb-6">
        <div className="h-14 w-14 rounded-full bg-muted/30 animate-pulse" />
        <div className="h-10 w-32 bg-muted/30 rounded animate-pulse" />
      </div>

      {/* Input skeleton */}
      <div className="h-10 w-full bg-muted/30 rounded animate-pulse" />
      <div className="h-10 w-full bg-muted/30 rounded animate-pulse" />

      {/* Button skeleton */}
      <div className="h-10 w-full bg-muted/30 rounded animate-pulse mt-6" />

      {/* OAuth separator skeleton */}
      <div className="relative my-4 w-full">
        <div className="h-px bg-muted/30" />
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="h-4 w-8 bg-muted/30 rounded animate-pulse" />
        </div>
      </div>

      {/* OAuth button skeleton */}
      <div className="h-10 w-full bg-muted/30 rounded animate-pulse" />
    </div>
  );
}

interface AuthFormSkeletonProps {
  className?: string;
}

/** 表单骨架屏 - 用于注册等表单页面 */
export function AuthFormSkeleton({ className }: AuthFormSkeletonProps) {
  return (
    <div className={cn("w-full flex flex-col gap-4", className)}>
      {/* Title skeleton */}
      <div className="h-8 w-48 bg-muted/30 rounded animate-pulse" />

      {/* Username skeleton */}
      <div>
        <div className="h-5 w-20 bg-muted/30 rounded mb-2" />
        <div className="h-10 w-full bg-muted/30 rounded animate-pulse" />
      </div>

      {/* Password skeleton */}
      <div>
        <div className="h-5 w-20 bg-muted/30 rounded mb-2" />
        <div className="h-10 w-full bg-muted/30 rounded animate-pulse" />
      </div>

      {/* Button skeleton */}
      <div className="h-10 w-full bg-muted/30 rounded animate-pulse mt-6" />
    </div>
  );
}
