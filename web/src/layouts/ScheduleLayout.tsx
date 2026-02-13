import { Search, X } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Outlet } from "react-router-dom";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleSearchBar } from "@/components/AIChat/ScheduleSearchBar";
import NavigationDrawer from "@/components/NavigationDrawer";
import RouteHeaderImage from "@/components/RouteHeaderImage";
import { Button } from "@/components/ui/button";
import { SidebarCollapseButton } from "@/components/ui/SidebarCollapseButton";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import useMediaQuery from "@/hooks/useMediaQuery";
import { useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";

const ScheduleSidebar = () => {
  const { selectedDate, setSelectedDate } = useScheduleContext();

  // Anchor date for schedule fetching - use selected date or today
  const anchorDate = selectedDate ? new Date(selectedDate + "T00:00:00") : new Date();
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const schedules = schedulesData?.schedules || [];

  return (
    <div className="h-full overflow-y-auto pt-4 px-4">
      <ScheduleCalendar schedules={schedules} selectedDate={selectedDate} onDateClick={setSelectedDate} showMobileHint={false} />
    </div>
  );
};

/**
 * ScheduleLayout - Layout for schedule/calendar pages
 *
 * === APPLICATION LAYOUT NOTES ===
 * This is an app-style layout with its own spacing management:
 * - Sidebar width: w-80 (320px)
 * - Sidebar padding: pt-4 px-4 (16px)
 * - Main content: NO top padding - app manages its own spacing
 *
 * @see docs/research/layout-spacing-unification.md
 */
const ScheduleLayout = () => {
  const { t } = useTranslation();
  const lg = useMediaQuery("lg");
  const { setFilteredSchedules, setHasSearchFilter } = useScheduleContext();
  const [showSearch, setShowSearch] = useState(false);

  // Desktop sidebar state - persisted to localStorage
  const [desktopSidebarOpen, setDesktopSidebarOpen] = useState(() => {
    if (typeof window === "undefined") return true;
    try {
      const saved = localStorage.getItem("schedule-sidebar-open");
      return saved !== "false"; // default to true
    } catch {
      return true;
    }
  });

  const toggleDesktopSidebar = useCallback(() => {
    setDesktopSidebarOpen((prev) => {
      const newValue = !prev;
      try {
        localStorage.setItem("schedule-sidebar-open", String(newValue));
      } catch {
        // ignore storage errors
      }
      return newValue;
    });
  }, []);

  // Fetch schedules for search
  const anchorDate = new Date();
  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const allSchedules = schedulesData?.schedules || [];

  return (
    <section className="@container w-full h-screen flex flex-col lg:h-screen overflow-hidden">
      {/* Mobile Header with Search */}
      <div className="lg:hidden flex-none relative flex items-center justify-center px-4 h-14 shrink-0 border-b border-border/50 bg-background">
        {showSearch ? (
          <div className="flex items-center w-full gap-2">
            <ScheduleSearchBar
              schedules={allSchedules}
              onFilteredChange={setFilteredSchedules}
              onHasFilterChange={setHasSearchFilter}
              className="flex-1 min-w-0"
              autoFocus
            />
            <Button
              variant="ghost"
              size="icon"
              onClick={() => {
                setShowSearch(false);
                setHasSearchFilter(false);
              }}
            >
              <X className="w-5 h-5" />
            </Button>
          </div>
        ) : (
          <>
            <div className="absolute left-4 top-0 bottom-0 flex items-center">
              <NavigationDrawer />
            </div>

            <RouteHeaderImage />

            <div className="absolute right-4 top-0 bottom-0 flex items-center">
              <Button variant="ghost" size="icon" onClick={() => setShowSearch(true)}>
                <Search className="w-5 h-5" />
              </Button>
            </div>
          </>
        )}
      </div>

      {/* Desktop Sidebar - Always rendered to maintain layout, hidden via class */}
      {lg && (
        <div
          className={cn(
            "fixed top-0 left-16 shrink-0 h-svh border-r border-border bg-background w-80 overflow-y-auto transition-all duration-300",
            !desktopSidebarOpen && "hidden",
          )}
        >
          <ScheduleSidebar />
        </div>
      )}

      {/* Main Content */}
      <div className={cn("flex-1 min-h-0 overflow-hidden transition-all duration-300", lg && desktopSidebarOpen ? "pl-80" : "")}>
        <Outlet />
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
  );
};

export default ScheduleLayout;
