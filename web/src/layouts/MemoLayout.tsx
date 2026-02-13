import { MenuIcon } from "lucide-react";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { matchPath, Outlet, useLocation } from "react-router-dom";
import { MemoExplorer, type MemoExplorerContext } from "@/components/MemoExplorer";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import { Button } from "@/components/ui/button";
import { SidebarCollapseButton } from "@/components/ui/SidebarCollapseButton";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { userServiceClient } from "@/connect";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useFilteredMemoStats } from "@/hooks/useFilteredMemoStats";
import useMediaQuery from "@/hooks/useMediaQuery";

import { cn } from "@/lib/utils";
import { Routes } from "@/router";

// Context for sidebar toggle state - allows child components to trigger sidebar toggle
// Note: immersiveMode is now derived from !desktopSidebarOpen for simplicity
interface MemoLayoutContextValue {
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  /** @deprecated Use !sidebarOpen instead */
  immersiveMode: boolean;
  /** @deprecated Use toggleSidebar instead */
  toggleImmersiveMode: (enabled: boolean) => void;
}

const MemoLayoutContext = createContext<MemoLayoutContextValue | undefined>(undefined);

export const useMemoLayout = () => {
  const context = useContext(MemoLayoutContext);
  if (!context) {
    throw new Error("useMemoLayout must be used within MemoLayout");
  }
  return context;
};

const MemoLayout = () => {
  const { t } = useTranslation();
  const lg = useMediaQuery("lg");
  const location = useLocation();
  const currentUser = useCurrentUser();
  const [profileUserName, setProfileUserName] = useState<string | undefined>();
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);

  // Desktop sidebar state - persisted to localStorage
  // immersiveMode is now derived as !desktopSidebarOpen
  const [desktopSidebarOpen, setDesktopSidebarOpen] = useState(() => {
    if (typeof window === "undefined") return true;
    try {
      const saved = localStorage.getItem("memo-sidebar-open");
      return saved !== "false"; // default to true
    } catch {
      return true;
    }
  });

  // Derived state for backward compatibility
  const immersiveMode = !desktopSidebarOpen;

  const toggleDesktopSidebar = useCallback(() => {
    setDesktopSidebarOpen((prev) => {
      const newValue = !prev;
      try {
        localStorage.setItem("memo-sidebar-open", String(newValue));
      } catch {
        // ignore storage errors
      }
      return newValue;
    });
  }, []);

  // Context value for child components
  const layoutContextValue = useMemo(
    () => ({
      sidebarOpen: desktopSidebarOpen,
      toggleSidebar: toggleDesktopSidebar,
      setSidebarOpen: (open: boolean) => {
        setDesktopSidebarOpen(open);
        try {
          localStorage.setItem("memo-sidebar-open", String(open));
        } catch {
          // ignore storage errors
        }
      },
      immersiveMode,
      toggleImmersiveMode: (_enabled: boolean) => {
        // For backward compatibility, toggle sidebar
        toggleDesktopSidebar();
      },
    }),
    [desktopSidebarOpen, immersiveMode, toggleDesktopSidebar],
  );

  // Determine context based on current route
  const context: MemoExplorerContext = useMemo(() => {
    if (location.pathname === Routes.HOME) return "home";
    if (location.pathname === Routes.EXPLORE) return "explore";
    if (matchPath("/archived", location.pathname)) return "archived";
    if (matchPath("/u/:username", location.pathname)) return "profile";
    return "home"; // fallback
  }, [location.pathname]);

  // Extract username from URL for profile context
  useEffect(() => {
    const match = matchPath("/u/:username", location.pathname);
    if (match && context === "profile") {
      const username = match.params.username;
      if (username) {
        // Fetch or get user to obtain user name (e.g., "users/123")
        // Note: User stats will be fetched by useFilteredMemoStats
        userServiceClient
          .getUser({ name: `users/${username}` })
          .then((user) => {
            setProfileUserName(user.name);
          })
          .catch((error) => {
            console.error("Failed to fetch profile user:", error);
            setProfileUserName(undefined);
          });
      }
    } else {
      setProfileUserName(undefined);
    }
  }, [location.pathname, context]);

  // Determine which user name to use for stats
  // - home: current user (uses backend user stats for normal memos)
  // - profile: viewed user (uses backend user stats for normal memos)
  // - archived: undefined (compute from cached archived memos, since user stats only includes normal memos)
  // - explore: undefined (compute from cached memos)
  const statsUserName = useMemo(() => {
    if (context === "home") {
      return currentUser?.name;
    } else if (context === "profile") {
      return profileUserName;
    }
    return undefined; // archived and explore contexts compute from cache
  }, [context, currentUser, profileUserName]);

  // Fetch stats from memo store cache (populated by PagedMemoList)
  // For user-scoped contexts, use backend user stats for tags (unaffected by filters)
  const { statistics, tags } = useFilteredMemoStats({ userName: statsUserName });

  return (
    <MemoLayoutContext.Provider value={layoutContextValue}>
      <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden bg-muted/50 dark:bg-muted/10">
        {/* Mobile Header */}
        <div
          className={cn(
            "lg:hidden flex-none relative flex items-center justify-center px-4 h-14 shrink-0 border-b border-border bg-background/95 backdrop-blur-sm",
            location.pathname !== Routes.HOME && "bg-background",
          )}
        >
          {/* Left - Navigation Drawer */}
          <div className="absolute left-4 top-0 bottom-0 flex items-center">
            <NavigationDrawer />
          </div>

          {/* Center - Title */}
          <div className="flex items-center">
            <RouteHeaderImage />
          </div>

          {/* Right - Sidebar Toggle */}
          <div className="absolute right-0 top-0 bottom-0 px-3 flex items-center">
            <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
              <SheetContent
                side="right"
                className="w-80 max-w-full [&_.absolute.top-4.right-4]:hidden border-l border-border bg-background gap-0"
              >
                <SheetHeader className="p-0">
                  <SheetTitle className="sr-only">Memo Explorer</SheetTitle>
                </SheetHeader>
                <MemoExplorer className="h-full" context={context} statisticsData={statistics} tagCount={tags} />
              </SheetContent>
            </Sheet>
            <Button variant="ghost" size="icon" onClick={() => setMobileSidebarOpen(true)} aria-label="Open sidebar" className="h-11 w-11">
              <MenuIcon className="w-5 h-5" />
            </Button>
          </div>
        </div>

        {/* Desktop Sidebar - Always rendered to maintain layout, hidden via class */}
        <div
          className={cn(
            // Fixed positioning
            "fixed top-0 left-16 shrink-0 h-svh border-r border-border w-80 overflow-y-auto overflow-x-hidden transition-all duration-300 z-30",
            // Visibility: hide on mobile or when collapsed
            !lg || !desktopSidebarOpen ? "hidden" : "",
            // Background and blur - only when visible
            lg && desktopSidebarOpen && "bg-background backdrop-blur-sm",
          )}
        >
          <MemoExplorer className="px-4 pt-4 pb-4" context={context} statisticsData={statistics} tagCount={tags} />
        </div>

        {/* Main Content */}
        <div
          className={cn(
            "flex-1 min-h-0 overflow-y-auto flex flex-col transition-all duration-300 bg-muted/50 dark:bg-muted/10 relative",
            lg && desktopSidebarOpen ? "pl-80" : "",
          )}
        >
          {/* Unified spacing container */}
          <div className="w-full min-h-full pt-4 sm:pt-6">
            {/* Page Content */}
            <Outlet />
          </div>
        </div>

        {/* Sidebar Collapse Button - Desktop only */}
        {lg && (
          <SidebarCollapseButton
            isExpanded={desktopSidebarOpen}
            onToggle={toggleDesktopSidebar}
            expandLabel={t("sidebar.expand")}
            collapseLabel={t("sidebar.collapse")}
          />
        )}
      </section>
    </MemoLayoutContext.Provider>
  );
};

export default MemoLayout;
