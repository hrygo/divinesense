import { MenuIcon } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Outlet } from "react-router-dom";
import { AIChatSidebar } from "@/components/AIChat/AIChatSidebar";
import { ModeThemeProvider } from "@/components/AIChat/ModeThemeProvider";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import { Button } from "@/components/ui/button";
import { SidebarCollapseButton } from "@/components/ui/SidebarCollapseButton";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AIChatProvider, useAIChat } from "@/contexts/AIChatContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";
import { PARROT_THEMES } from "@/types/parrot";

/**
 * AI Chat Layout - 优化的聊天布局
 *
 * UX/UI 改进：
 * - 优化移动端和桌面端的布局切换
 * - 统一间距和边框样式
 * - 改进侧边栏和主内容的视觉层次
 * - 移动端 Header 支持三种模式视觉反馈（普通/极客/进化）
 *
 * === APPLICATION LAYOUT NOTES ===
 * This is an app-style layout with its own spacing management:
 * - Sidebar width: w-80 (320px)
 * - Sidebar padding: managed by AIChatSidebar internally (pt-4 for new button)
 * - Main content: NO top padding - full-screen app experience
 *
 * @see docs/research/layout-spacing-unification.md
 */

// Helper function to get mode-specific styles using PARROT_THEMES
function getModeStyles(mode: AIMode) {
  switch (mode) {
    case "geek": {
      const theme = PARROT_THEMES.GEEK;
      return {
        headerBorder: "border-sky-200 dark:border-slate-700",
        headerBg: theme.headerBg,
        headerAccent: "after:bg-gradient-to-r after:from-transparent via-sky-500 to-transparent",
        sidebarBorder: theme.cardBorder,
        sidebarBg: theme.inputBg,
        contentBg: theme.bubbleBg,
        iconColor: theme.text,
        dotColor: "bg-sky-500",
        monoFont: true,
        modeLabel: "GEEK",
      };
    }
    case "evolution": {
      const theme = PARROT_THEMES.EVOLUTION;
      return {
        headerBorder: "border-emerald-200 dark:border-emerald-700",
        headerBg: theme.headerBg,
        headerAccent: "after:bg-gradient-to-r after:from-transparent via-emerald-500 to-transparent",
        sidebarBorder: theme.cardBorder,
        sidebarBg: theme.inputBg,
        contentBg: theme.bubbleBg,
        iconColor: theme.text,
        dotColor: "bg-emerald-500",
        monoFont: true,
        modeLabel: "EVOLUTION",
      };
    }
    default: {
      const theme = PARROT_THEMES.NORMAL;
      return {
        headerBorder: theme.inputBorder,
        headerBg: theme.headerBg,
        headerAccent: "",
        sidebarBorder: theme.cardBorder,
        sidebarBg: theme.inputBg,
        contentBg: theme.bubbleBg,
        iconColor: theme.text,
        dotColor: "bg-amber-500",
        monoFont: false,
        modeLabel: "",
      };
    }
  }
}

const AIChatLayoutContent = () => {
  const { t } = useTranslation();
  const lg = useMediaQuery("lg");
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const { state, toggleImmersiveMode } = useAIChat();
  const currentMode = state.currentMode || "normal";
  const immersiveMode = state.immersiveMode || false;

  const modeStyles = getModeStyles(currentMode);
  const isSpecialMode = currentMode !== "normal";

  // Get mode-specific background effect class
  const getBgEffectClass = (mode: AIMode) => {
    switch (mode) {
      case "geek":
        return "geek-matrix-bg";
      case "evolution":
        return "evo-bio-bg";
      default:
        return "";
    }
  };

  return (
    <section
      className={cn(
        "@container w-full h-screen flex flex-col lg:h-screen overflow-hidden bg-zinc-50 dark:bg-zinc-950",
        getBgEffectClass(currentMode),
      )}
    >
      {/* Mobile Header */}
      <div
        className={cn(
          "lg:hidden flex-none relative flex items-center justify-center px-4 h-14 shrink-0 border-b transition-colors",
          // Normal mode styles (default)
          !isSpecialMode && modeStyles.headerBorder,
          !isSpecialMode && modeStyles.headerBg,
          // Special mode styles (geek/evolution)
          isSpecialMode && [
            modeStyles.headerBorder,
            modeStyles.headerBg,
            // Bottom accent line for special modes
            modeStyles.headerAccent && "after:content-[''] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-[2px]",
            modeStyles.headerAccent,
          ],
        )}
      >
        {/* Left - Navigation Drawer */}
        <div className="absolute left-4 top-0 bottom-0 flex items-center">
          <NavigationDrawer />
        </div>

        {/* Center - Title with mode-specific styling */}
        <div className={cn("flex items-center gap-2", modeStyles.monoFont && "font-mono")}>
          {isSpecialMode && (
            <span className={cn("flex items-center gap-1 text-xs", modeStyles.iconColor)}>
              <span className={cn("w-2 h-2 rounded-full animate-pulse", modeStyles.dotColor)} />
              <span className="hidden sm:inline">{modeStyles.modeLabel}</span>
            </span>
          )}
          <RouteHeaderImage mode={currentMode} />
        </div>

        {/* Right - Sidebar Toggle */}
        <div className="absolute right-0 top-0 bottom-0 px-3 flex items-center">
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent
              side="right"
              className={cn(
                "w-80 max-w-full [&_.absolute.top-4.right-4]:hidden border-l",
                !isSpecialMode && "bg-zinc-50 dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800",
                isSpecialMode && [modeStyles.sidebarBg, modeStyles.sidebarBorder],
                "gap-0",
              )}
            >
              <SheetHeader className="p-0">
                <SheetTitle className="sr-only">AI Assistant</SheetTitle>
              </SheetHeader>
              <AIChatSidebar className="h-full" onClose={() => setMobileSidebarOpen(false)} />
            </SheetContent>
          </Sheet>
          <Button variant="ghost" size="icon" onClick={() => setMobileSidebarOpen(true)} aria-label="Open sidebar" className="h-11 w-11">
            <MenuIcon className={cn("w-5 h-5", isSpecialMode && modeStyles.iconColor)} />
          </Button>
        </div>
      </div>

      {/* Desktop Sidebar - Always rendered to maintain layout, hidden via class */}
      <div
        className={cn(
          // Fixed positioning
          "fixed top-0 left-16 shrink-0 h-svh border-r backdrop-blur-sm w-80 overflow-hidden transition-colors",
          // Visibility: hide on mobile or in immersive mode, always keep DOM for layout stability
          !lg || immersiveMode ? "hidden" : "",
          // Mode-specific styles (applied based on current mode)
          modeStyles.sidebarBorder,
          modeStyles.sidebarBg,
        )}
      >
        <AIChatSidebar className="h-full" />
      </div>

      {/* Main Content */}
      <div
        className={cn(
          "flex-1 min-h-0 overflow-hidden transition-all duration-300",
          modeStyles.contentBg,
          lg && !immersiveMode ? "pl-80" : "",
        )}
      >
        <Outlet />
      </div>

      {/* Sidebar Collapse Button - Desktop only */}
      {lg && (
        <SidebarCollapseButton
          isExpanded={!immersiveMode}
          onToggle={() => toggleImmersiveMode(!immersiveMode)}
          expandLabel={t("sidebar.expand")}
          collapseLabel={t("sidebar.collapse")}
        />
      )}
    </section>
  );
};

const AIChatLayout = () => {
  return (
    <AIChatProvider>
      <ModeThemeProvider />
      <AIChatLayoutContent />
    </AIChatProvider>
  );
};

export default AIChatLayout;
