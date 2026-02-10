/**
 * ServiceUnavailable - 显示服务不可用的友好提示
 *
 * 当后端服务停止或网络连接失败时显示
 */

import { RefreshCw, WifiOff } from "lucide-react";
import { useCallback } from "react";
import { useInstance } from "@/contexts/InstanceContext";
import { useTranslate } from "@/utils/i18n";
import { Button } from "./ui/button";

export interface ServiceUnavailableProps {
  /** 重试回调 */
  onRetry?: () => void;
  /** 自定义消息 */
  message?: string;
  /** 是否显示详情 */
  showDetails?: boolean;
  /** 是否使用全屏模式（用于独立页面） */
  fullscreen?: boolean;
}

export function ServiceUnavailable({ onRetry, message, showDetails = false, fullscreen = true }: ServiceUnavailableProps) {
  const { initialize } = useInstance();
  const t = useTranslate();

  const handleRetry = useCallback(() => {
    if (onRetry) {
      onRetry();
    } else {
      initialize();
    }
  }, [onRetry, initialize]);

  // 全屏模式：用于独立页面
  if (fullscreen) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-background p-4">
        <div className="max-w-[28rem] w-full p-6 space-y-4">
          {/* Icon */}
          <div className="flex justify-center">
            <div className="p-4 rounded-full bg-muted">
              <WifiOff className="w-12 h-12 text-muted-foreground" />
            </div>
          </div>

          {/* Title */}
          <div className="text-center">
            <h1 className="text-xl font-bold text-foreground">{t("error.service_unavailable.title")}</h1>
            <p className="text-sm text-muted-foreground mt-2">{message || t("error.service_unavailable.message")}</p>
          </div>

          {/* Details */}
          {showDetails && (
            <div className="bg-muted p-3 rounded-md text-sm text-muted-foreground">
              <p className="font-medium mb-1">{t("error.service_unavailable.reasons_title")}</p>
              <ul className="list-disc list-inside space-y-1 text-xs">
                <li>{t("error.service_unavailable.reason_backend")}</li>
                <li>{t("error.service_unavailable.reason_network")}</li>
                <li>{t("error.service_unavailable.reason_temporary")}</li>
              </ul>
            </div>
          )}

          {/* Retry button */}
          <Button onClick={handleRetry} className="w-full gap-2">
            <RefreshCw className="w-4 h-4" />
            {t("error.service_unavailable.retry")}
          </Button>
        </div>
      </div>
    );
  }

  // 内联模式：用于嵌入到其他页面中（如 SignUp/SignIn）
  return (
    <div className="w-full p-4 space-y-4">
      {/* Icon */}
      <div className="flex justify-center">
        <div className="p-4 rounded-full bg-muted">
          <WifiOff className="w-12 h-12 text-muted-foreground" />
        </div>
      </div>

      {/* Title */}
      <div className="text-center">
        <h1 className="text-xl font-bold text-foreground">{t("error.service_unavailable.title")}</h1>
        <p className="text-sm text-muted-foreground mt-2">{message || t("error.service_unavailable.message")}</p>
      </div>

      {/* Details */}
      {showDetails && (
        <div className="bg-muted p-3 rounded-md text-sm text-muted-foreground">
          <p className="font-medium mb-1">{t("error.service_unavailable.reasons_title")}</p>
          <ul className="list-disc list-inside space-y-1 text-xs">
            <li>{t("error.service_unavailable.reason_backend")}</li>
            <li>{t("error.service_unavailable.reason_network")}</li>
            <li>{t("error.service_unavailable.reason_temporary")}</li>
          </ul>
        </div>
      )}

      {/* Retry button */}
      <Button onClick={handleRetry} className="w-full gap-2">
        <RefreshCw className="w-4 h-4" />
        {t("error.service_unavailable.retry")}
      </Button>
    </div>
  );
}
