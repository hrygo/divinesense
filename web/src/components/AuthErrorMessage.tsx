import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface AuthErrorMessageProps {
  message: string;
  className?: string;
}

/** 认证错误提示组件 - 提供友好的错误信息展示 */
export function AuthErrorMessage({ message, className }: AuthErrorMessageProps) {
  return (
    <div
      className={cn(
        "mb-4 flex items-start gap-3 p-4 text-sm text-destructive bg-destructive/5 border border-destructive/20 rounded-lg animate-in fade-in slide-in-from-top-2",
        className,
      )}
    >
      <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
      <div className="flex-1">
        <p className="font-medium">{message}</p>
      </div>
    </div>
  );
}
