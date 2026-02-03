/**
 * PWA Install Hook
 *
 * Manages PWA installation prompt for different platforms:
 * - Chrome/Edge (Android/Desktop): Uses beforeinstallprompt event
 * - Safari (iOS): Shows manual install guide
 */

import { useCallback, useEffect, useState } from "react";

// TypeScript types for PWA install prompt event
interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

interface PWAInstallState {
  isInstallable: boolean;
  isIOS: boolean;
  isDesktop: boolean;
  isInstalled: boolean;
  deferredPrompt: BeforeInstallPromptEvent | null;
  showPrompt: boolean;
}

const STORAGE_KEY = "pwa-install-prompt-dismissed";
const DISMISSAL_DURATION = 7 * 24 * 60 * 60 * 1000; // 7 days

/**
 * Check if the browser is running on iOS Safari
 */
const checkIsIOS = (): boolean => {
  const ua = window.navigator.userAgent;
  return /iPad|iPhone|iPod/.test(ua) && !(window as { MSStream?: unknown }).MSStream;
};

/**
 * Check if the browser is running on desktop (not mobile)
 */
const checkIsDesktop = (): boolean => {
  const ua = window.navigator.userAgent;
  // Check for common desktop indicators
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua);
  const isTablet = /Tablet|iPad/i.test(ua);
  return !isMobile && !isTablet;
};

/**
 * Check if app is already installed (running in standalone mode)
 *
 * iOS Safari < 15.4 doesn't support display-mode media query,
 * so we rely on navigator.standalone as the primary check for iOS.
 */
const checkIsInstalled = (): boolean => {
  // For iOS Safari (most reliable, works on all versions)
  const isStandalone = (window.navigator as { standalone?: boolean }).standalone === true;

  // For Chrome/Android/Desktop (supports display-mode since Chrome 47)
  const isDisplayModeStandalone = window.matchMedia("(display-mode: standalone)").matches;

  // Additional check: if running as PWA, window.history won't have a referrer
  const hasMinimalHistory = window.history.length <= 1;

  // Fallback for older browsers: check if we're in a window that looks like a PWA
  const looksLikePWA = !window.matchMedia("(display-mode: browser)").matches;

  return isStandalone || isDisplayModeStandalone || (hasMinimalHistory && looksLikePWA);
};

/**
 * Check if the install prompt was recently dismissed
 * Uses try-catch for private browsing mode compatibility
 */
const wasRecentlyDismissed = (): boolean => {
  try {
    const dismissed = localStorage.getItem(STORAGE_KEY);
    if (!dismissed) return false;
    const dismissedTime = parseInt(dismissed, 10);
    return Date.now() - dismissedTime < DISMISSAL_DURATION;
  } catch {
    // localStorage unavailable (private mode, cookies blocked)
    return false;
  }
};

export const usePWAInstall = () => {
  const [state, setState] = useState<PWAInstallState>({
    isInstallable: false,
    isIOS: false,
    isDesktop: false,
    isInstalled: false,
    deferredPrompt: null,
    showPrompt: false,
  });

  // Memoize expensive checks
  const isIOS = checkIsIOS();
  const isDesktop = checkIsDesktop();
  const isInstalled = checkIsInstalled();
  const recentlyDismissed = wasRecentlyDismissed();

  // Hide install prompt
  const hideInstallPrompt = useCallback(() => {
    try {
      localStorage.setItem(STORAGE_KEY, Date.now().toString());
    } catch {
      // Silently fail if localStorage is unavailable
    }
    setState((prev) => ({ ...prev, showPrompt: false }));
  }, []);

  // Prompt installation (for Chrome/Android/Desktop)
  const promptInstall = useCallback(async () => {
    const { deferredPrompt } = state;
    if (!deferredPrompt) return false;

    deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;

    setState((prev) => ({
      ...prev,
      deferredPrompt: null,
      isInstallable: false,
      showPrompt: false,
    }));

    return outcome === "accepted";
  }, [state.deferredPrompt]);

  useEffect(() => {
    // Determine if we should show the prompt
    const shouldShowPrompt = !isInstalled && !recentlyDismissed;

    setState((prev) => ({
      ...prev,
      isInstalled,
      isIOS,
      isDesktop,
      showPrompt: shouldShowPrompt && (isIOS || isDesktop),
    }));

    // Skip event listeners if already installed
    if (isInstalled) return;

    // Handle beforeinstallprompt event (Chrome/Edge/Firefox on Android/Desktop)
    const handleBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setState((prev) => ({
        ...prev,
        isInstallable: true,
        deferredPrompt: e as BeforeInstallPromptEvent,
        showPrompt: !recentlyDismissed,
      }));
    };

    // Handle appinstalled event
    const handleAppInstalled = () => {
      setState({
        isInstallable: false,
        isIOS: false,
        isDesktop: false,
        isInstalled: true,
        deferredPrompt: null,
        showPrompt: false,
      });
    };

    window.addEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
    window.addEventListener("appinstalled", handleAppInstalled);

    return () => {
      window.removeEventListener("beforeinstallprompt", handleBeforeInstallPrompt);
      window.removeEventListener("appinstalled", handleAppInstalled);
    };
  }, [isInstalled, isIOS, isDesktop, recentlyDismissed]);

  return {
    ...state,
    promptInstall,
    hideInstallPrompt,
  };
};

export type { PWAInstallState };
