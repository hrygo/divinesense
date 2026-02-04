import { MonitorIcon } from "lucide-react";
import { useTranslate } from "@/utils/i18n";

interface DesktopOnlyBannerProps {
  /** Feature-specific message key (e.g., "member-desktop-only") */
  messageKey?: string;
  /** Custom description text to override default */
  description?: string;
  /** Custom title to override default */
  title?: string;
}

/**
 * Mobile banner indicating that a feature is desktop-only.
 * Used for admin/settings features that don't have mobile-optimized interfaces.
 */
const DesktopOnlyBanner = ({ messageKey, description, title }: DesktopOnlyBannerProps) => {
  const t = useTranslate();

  return (
    <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
      <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
        <MonitorIcon className="w-8 h-8 text-muted-foreground" />
      </div>
      <h3 className="text-lg font-semibold mb-2">{title || t("setting.mobile.desktop-required-title")}</h3>
      <p className="text-sm text-muted-foreground max-w-[28rem]">
        {/* biome-ignore lint/suspicious/noExplicitAny: i18n dynamic key access */}
        {messageKey ? (t as any)(`setting.mobile.${messageKey}`) : description || t("setting.mobile.desktop-required-description")}
      </p>
    </div>
  );
};

export default DesktopOnlyBanner;
