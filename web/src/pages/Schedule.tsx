import { useQueryClient } from "@tanstack/react-query";
import { Calendar, LayoutList } from "lucide-react";
import { useMemo, useState } from "react";
import { ScheduleCalendar } from "@/components/AIChat/ScheduleCalendar";
import { ScheduleInput } from "@/components/AIChat/ScheduleInput";
import { ScheduleSearchBar } from "@/components/AIChat/ScheduleSearchBar";
import { ScheduleTimeline } from "@/components/AIChat/ScheduleTimeline";
import type { UIToolEvent } from "@/components/ScheduleAI/types";
import { ScheduleQuickInput } from "@/components/ScheduleQuickInput/ScheduleQuickInput";
import { Button } from "@/components/ui/button";
import { useScheduleContext } from "@/contexts/ScheduleContext";
import { useSchedulesOptimized } from "@/hooks/useScheduleQueries";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

type ViewTab = "calendar" | "timeline";

const Schedule = () => {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const { selectedDate, setSelectedDate, filteredSchedules, hasSearchFilter, setFilteredSchedules, setHasSearchFilter } =
    useScheduleContext();

  const [viewTab, setViewTab] = useState<ViewTab>("timeline");
  const [scheduleInputOpen, setScheduleInputOpen] = useState(false);
  const [editSchedule, setEditSchedule] = useState<Schedule | null>(null);
  const [uiTools, setUITools] = useState<UIToolEvent[]>([]);

  const anchorDate = useMemo(() => {
    return selectedDate ? new Date(selectedDate + "T00:00:00") : new Date();
  }, [selectedDate]);

  const { data: schedulesData } = useSchedulesOptimized(anchorDate);
  const allSchedules = schedulesData?.schedules || [];
  const displaySchedules = hasSearchFilter ? filteredSchedules : allSchedules;
  const effectiveViewTab = hasSearchFilter ? "timeline" : viewTab;

  const handleEditSchedule = (schedule: Schedule) => {
    setEditSchedule(schedule);
    setScheduleInputOpen(true);
  };

  const handleCloseInput = () => {
    setScheduleInputOpen(false);
    setEditSchedule(null);
  };

  const handleDateClick = (date: string) => {
    setSelectedDate(date);
    setViewTab("timeline");
  };

  const handleScheduleCreated = () => {
    queryClient.invalidateQueries({ queryKey: ["schedules"] });
  };

  const handleUIAction = (action: { type: string; toolId: string; data?: unknown }) => {
    setUITools((prev) => prev.filter((t) => t.id !== action.toolId));
    queryClient.invalidateQueries({ queryKey: ["schedules"] });
  };

  const handleUIDismiss = (toolId: string) => {
    setUITools((prev) => prev.filter((t) => t.id !== toolId));
  };

  return (
    <div className="w-full max-w-[100rem] mx-auto h-full flex overflow-hidden">
      {/* Main Content Area */}
      <div className="flex-1 flex flex-col overflow-hidden w-full">
        {/* Header with View Tabs and Search (desktop) */}
        <div className="hidden lg:flex flex-none px-4 py-3 border-b border-border/50 overflow-hidden">
          <div className="flex items-center justify-between gap-4 w-full">
            {!hasSearchFilter ? (
              <div className="flex items-center gap-1.5 p-1 bg-muted/50 rounded-lg" role="tablist">
                <Button
                  role="tab"
                  aria-selected={viewTab === "timeline"}
                  variant={viewTab === "timeline" ? "default" : "ghost"}
                  size="sm"
                  className="h-9 px-3 text-sm font-medium rounded-md"
                  onClick={() => setViewTab("timeline")}
                >
                  <LayoutList className="w-4 h-4 mr-1.5" />
                  {t("schedule.timeline") || "Timeline"}
                </Button>
                <Button
                  role="tab"
                  aria-selected={viewTab === "calendar"}
                  variant={viewTab === "calendar" ? "default" : "ghost"}
                  size="sm"
                  className="h-9 px-3 text-sm font-medium rounded-md"
                  onClick={() => setViewTab("calendar")}
                >
                  <Calendar className="w-4 h-4 mr-1.5" />
                  {t("schedule.month-view") || "Month"}
                </Button>
              </div>
            ) : (
              <span className="text-sm text-muted-foreground">
                {filteredSchedules.length} {t("schedule.search-results") || "results"}
              </span>
            )}

            <div className="flex items-center gap-2 justify-end">
              <ScheduleSearchBar
                schedules={allSchedules}
                onFilteredChange={setFilteredSchedules}
                onHasFilterChange={setHasSearchFilter}
                className="w-80"
              />
            </div>
          </div>
        </div>

        {/* Mobile: View Tabs */}
        <div className="lg:hidden flex-none px-3 py-2 flex items-center justify-between gap-2 border-b border-border/50">
          {!hasSearchFilter ? (
            <div className="flex items-center gap-1 p-1 bg-muted/50 rounded-lg">
              <Button
                aria-selected={viewTab === "timeline"}
                variant={viewTab === "timeline" ? "default" : "ghost"}
                size="sm"
                className="h-10 w-10 p-0 rounded-md min-h-[44px] min-w-[44px]"
                onClick={() => setViewTab("timeline")}
              >
                <LayoutList className="w-4 h-4" />
              </Button>
              <Button
                aria-selected={viewTab === "calendar"}
                variant={viewTab === "calendar" ? "default" : "ghost"}
                size="sm"
                className="h-10 w-10 p-0 rounded-md min-h-[44px] min-w-[44px]"
                onClick={() => setViewTab("calendar")}
              >
                <Calendar className="w-4 h-4" />
              </Button>
            </div>
          ) : (
            <span className="text-xs text-muted-foreground">
              {filteredSchedules.length} {t("schedule.search-results") || "results"}
            </span>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 pb-4 overflow-x-hidden">
          {effectiveViewTab === "calendar" ? (
            <ScheduleCalendar
              schedules={displaySchedules}
              selectedDate={selectedDate}
              onDateClick={handleDateClick}
              showMobileHint={false}
            />
          ) : (
            <ScheduleTimeline
              schedules={displaySchedules}
              selectedDate={selectedDate}
              onDateClick={handleDateClick}
              onScheduleEdit={handleEditSchedule}
            />
          )}
        </div>

        {/* Quick Input with Templates */}
        <div className="flex-none p-4 bg-background/95 backdrop-blur-sm border-t border-border/50">
          <ScheduleQuickInput
            initialDate={selectedDate}
            onScheduleCreated={handleScheduleCreated}
            // editingSchedule prop removed to decouple from edit mode (Unified Dialog handles edits now)
            uiTools={uiTools}
            onUIAction={handleUIAction}
            onUIDismiss={handleUIDismiss}
          />
        </div>
      </div>

      {/* Schedule Input Dialog */}
      <ScheduleInput
        open={scheduleInputOpen}
        onOpenChange={handleCloseInput}
        editSchedule={editSchedule}
        contextDate={selectedDate}
        onSuccess={() => {
          handleCloseInput();
          queryClient.invalidateQueries({ queryKey: ["schedules"] });
        }}
      />
    </div>
  );
};

export default Schedule;
