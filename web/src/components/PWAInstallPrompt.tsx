/**
 * PWA Install Prompt Component
 *
 * Displays installation prompt for PWA:
 * - Android/Chrome: Native install banner
 * - iOS Safari: Manual install guide with steps
 * - Desktop: Manual install guide for Chrome/Edge
 */

import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { usePWAInstall } from "@/hooks/usePWAInstall";
import { cn } from "@/lib/utils";

const IOS_STEP_CYCLE_DURATION = 3000; // ms
const DESKTOP_STEP_CYCLE_DURATION = 4000; // ms (slower for desktop)

interface PWAInstallPromptProps {
  className?: string;
}

export const PWAInstallPrompt = ({ className }: PWAInstallPromptProps) => {
  const { t } = useTranslation();
  const { isInstallable, isIOS, isDesktop, showPrompt, promptInstall, hideInstallPrompt } = usePWAInstall();
  const [iosGuideStep, setIosGuideStep] = useState(1);
  const [desktopGuideStep, setDesktopGuideStep] = useState(1);
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);

  // Check for reduced motion preference
  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-reduced-motion: reduce)");
    setPrefersReducedMotion(mediaQuery.matches);

    const handleChange = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches);
    mediaQuery.addEventListener("change", handleChange);

    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  // Auto-cycle iOS guide steps (respects reduced motion preference)
  useEffect(() => {
    if (prefersReducedMotion || !showPrompt || !isIOS) return;
    const interval = setInterval(() => {
      setIosGuideStep((prev) => (prev % 3) + 1);
    }, IOS_STEP_CYCLE_DURATION);
    return () => clearInterval(interval);
  }, [showPrompt, isIOS, prefersReducedMotion]);

  // Auto-cycle desktop guide steps (respects reduced motion preference)
  useEffect(() => {
    if (prefersReducedMotion || !showPrompt || !isDesktop) return;
    const interval = setInterval(() => {
      setDesktopGuideStep((prev) => (prev % 2) + 1);
    }, DESKTOP_STEP_CYCLE_DURATION);
    return () => clearInterval(interval);
  }, [showPrompt, isDesktop, prefersReducedMotion]);

  if (!showPrompt) return null;

  const handleInstall = async () => {
    if (isInstallable) {
      await promptInstall();
    } else {
      // For iOS and desktop, the guide is shown inline
      hideInstallPrompt();
    }
  };

  const getActiveStepClass = (currentStep: number, activeStep: number) =>
    cn(
      "flex items-start gap-2 transition-colors duration-300",
      currentStep === activeStep && !prefersReducedMotion && "text-foreground font-medium",
    );

  return (
    <div className={cn("fixed bottom-4 left-4 right-4 sm:left-auto sm:right-4 sm:w-80 z-50 animate-in slide-in-from-bottom-4", className)}>
      <div className="bg-background border border-border rounded-lg shadow-lg p-4">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-blue-500 flex items-center justify-center shrink-0">
              <svg className="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 2L2 7l10 5 10-5-10-5z" />
                <path d="M2 17l10 5 10-5" />
                <path d="M2 12l10 5 10-5" />
              </svg>
            </div>
            <div>
              <h3 className="font-medium text-sm">{t("pwa.install.title")}</h3>
              <p className="text-xs text-muted-foreground mt-0.5">{t("pwa.install.description")}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={hideInstallPrompt}
            className="shrink-0 p-1 hover:bg-muted rounded transition-colors"
            aria-label={t("common.close")}
          >
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        {/* iOS Guide */}
        {isIOS && (
          <div className="mt-3 p-3 bg-muted/50 rounded-md">
            <p className="text-xs font-medium mb-2">{t("pwa.install.iosSteps.title")}</p>
            <ol className="text-xs text-muted-foreground space-y-1.5">
              <li className={getActiveStepClass(1, iosGuideStep)}>
                <span className="shrink-0 w-4 h-4 rounded-full bg-primary/10 text-primary flex items-center justify-center text-[10px]">
                  1
                </span>
                {t("pwa.install.iosSteps.share")}
              </li>
              <li className={getActiveStepClass(2, iosGuideStep)}>
                <span className="shrink-0 w-4 h-4 rounded-full bg-primary/10 text-primary flex items-center justify-center text-[10px]">
                  2
                </span>
                {t("pwa.install.iosSteps.addToHome")}
              </li>
              <li className={getActiveStepClass(3, iosGuideStep)}>
                <span className="shrink-0 w-4 h-4 rounded-full bg-primary/10 text-primary flex items-center justify-center text-[10px]">
                  3
                </span>
                {t("pwa.install.iosSteps.confirm")}
              </li>
            </ol>
          </div>
        )}

        {/* Desktop Guide */}
        {isDesktop && !isInstallable && (
          <div className="mt-3 p-3 bg-muted/50 rounded-md">
            <p className="text-xs font-medium mb-2">{t("pwa.install.desktopSteps.title")}</p>
            <ol className="text-xs text-muted-foreground space-y-1.5">
              <li className={getActiveStepClass(1, desktopGuideStep)}>
                <span className="shrink-0 w-4 h-4 rounded-full bg-primary/10 text-primary flex items-center justify-center text-[10px]">
                  1
                </span>
                {t("pwa.install.desktopSteps.installMenu")}
              </li>
              <li className={getActiveStepClass(2, desktopGuideStep)}>
                <span className="shrink-0 w-4 h-4 rounded-full bg-primary/10 text-primary flex items-center justify-center text-[10px]">
                  2
                </span>
                {t("pwa.install.desktopSteps.confirm")}
              </li>
            </ol>
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-2 mt-3">
          {isInstallable ? (
            <button
              type="button"
              onClick={handleInstall}
              className="flex-1 px-3 py-2 bg-primary text-primary-foreground text-sm font-medium rounded-md hover:bg-primary/90 transition-colors"
            >
              {t("pwa.install.install")}
            </button>
          ) : isIOS || isDesktop ? (
            <button
              type="button"
              onClick={hideInstallPrompt}
              className="flex-1 px-3 py-2 bg-primary text-primary-foreground text-sm font-medium rounded-md hover:bg-primary/90 transition-colors"
            >
              {t("common.got-it")}
            </button>
          ) : null}
          <button
            type="button"
            onClick={hideInstallPrompt}
            className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            {t("common.not-now")}
          </button>
        </div>
      </div>
    </div>
  );
};
