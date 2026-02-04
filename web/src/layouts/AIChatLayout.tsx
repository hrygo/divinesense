import { MenuIcon } from "lucide-react";
import { useState } from "react";
import { Outlet } from "react-router-dom";
import { AIChatSidebar } from "@/components/AIChat/AIChatSidebar";
import { ModeCycleButton } from "@/components/AIChat/ModeCycleButton";
import { ModeThemeProvider } from "@/components/AIChat/ModeThemeProvider";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { AIChatProvider, useAIChat } from "@/contexts/AIChatContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { AIMode } from "@/types/aichat";

/**
 * AI Chat Layout - 优化的聊天布局
 *
 * UX/UI 改进：
 * - 优化移动端和桌面端的布局切换
 * - 统一间距和边框样式
 * - 改进侧边栏和主内容的视觉层次
 * - 移动端 Header 支持三种模式视觉反馈（普通/极客/进化）
 *
 * === SPACING DEVIATION NOTES ===
 * This layout intentionally deviates from the standard spacing specification:
 * - NO top padding (pt-6) on main content - full-screen chat experience
 * - Sidebar padding: pt-2 (8px) instead of py-6 - more compact
 * - This is by design for immersive chat interface
 *
 * @see docs/research/layout-spacing-unification.md
 */

// Helper function to get mode-specific styles
function getModeStyles(mode: AIMode) {
  switch (mode) {
    case "geek":
      return {
        headerBorder: "border-green-500/30",
        headerBg: "bg-green-950/20 dark:bg-green-950/40",
        headerAccent: "after:bg-gradient-to-r after:from-transparent via-green-500 to-transparent",
        sidebarBorder: "border-green-500/30",
        sidebarBg: "bg-green-950/10 dark:bg-green-950/20",
        contentBg: "bg-green-50/30 dark:bg-green-950/10",
        iconColor: "text-green-600 dark:text-green-400",
        dotColor: "bg-green-500",
        monoFont: true,
        modeLabel: "GEEK",
      };
    case "evolution":
      return {
        headerBorder: "border-purple-500/30",
        headerBg: "bg-purple-950/20 dark:bg-purple-950/40",
        headerAccent: "after:bg-gradient-to-r after:from-transparent via-purple-500 to-transparent",
        sidebarBorder: "border-purple-500/30",
        sidebarBg: "bg-purple-950/10 dark:bg-purple-950/20",
        contentBg: "bg-purple-50/30 dark:bg-purple-950/10",
        iconColor: "text-purple-600 dark:text-purple-400",
        dotColor: "bg-purple-500",
        monoFont: true,
        modeLabel: "EVOLUTION",
      };
    default:
      return {
        headerBorder: "border-zinc-200 dark:border-zinc-800",
        headerBg: "bg-white dark:bg-zinc-900",
        headerAccent: "",
        sidebarBorder: "border-zinc-200/80 dark:border-zinc-800/80",
        sidebarBg: "bg-zinc-50/95 dark:bg-zinc-900/95",
        contentBg: "bg-white dark:bg-zinc-900",
        iconColor: "",
        dotColor: "",
        monoFont: false,
        modeLabel: "",
      };
  }
}

const AIChatLayoutContent = () => {
  const lg = useMediaQuery("lg");
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const { state, setMode } = useAIChat();
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

        {/* Right - Mode Toggle + Sidebar Toggle */}
        <div className="absolute right-0 top-0 bottom-0 px-3 flex items-center gap-1">
          {/* Mode Cycle Button - Mobile only */}
          <div className="lg:hidden">
            <ModeCycleButton currentMode={currentMode} onModeChange={setMode} variant="mobile" isAdmin={true} />
          </div>
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
          <Button variant="ghost" size="icon" onClick={() => setMobileSidebarOpen(true)} aria-label="Open sidebar" className="h-9 w-9">
            <MenuIcon className={cn("w-5 h-5", isSpecialMode && modeStyles.iconColor)} />
          </Button>
        </div>
      </div>

      {/* Desktop Sidebar - Always rendered to maintain layout, hidden via class */}
      <div
        className={cn(
          // Fixed positioning
          "fixed top-0 left-16 shrink-0 h-svh border-r backdrop-blur-sm w-72 overflow-hidden pt-2 transition-colors",
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
          lg && !immersiveMode ? "pl-72" : "",
        )}
      >
        <Outlet />
      </div>
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
