import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs, { Dayjs } from "dayjs";
import { CalendarDays, ChevronLeft, ChevronRight, Clock, MapPin } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";

interface ScheduleTimelineProps {
  schedules: Schedule[];
  selectedDate?: string;
  onDateClick?: (date: string) => void;
  onScheduleEdit?: (schedule: Schedule) => void;
  className?: string;
}

// Date Strip Component - Optimized for spacing
interface DateStripProps {
  currentDate: Dayjs;
  selectedDate: string;
  schedules: Schedule[];
  onDateSelect: (date: Dayjs) => void;
  onPrevWeek: () => void;
  onNextWeek: () => void;
}

const DateStrip = ({ currentDate, selectedDate, schedules, onDateSelect, onPrevWeek, onNextWeek }: DateStripProps) => {
  const todayStr = dayjs().format("YYYY-MM-DD");
  const startDate = currentDate.subtract(3, "day");
  const toDayjs = (ts: bigint) => dayjs(timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 })));

  return (
    <div className="flex items-center justify-between px-2 sm:px-4 py-2 w-full bg-background/50 backdrop-blur-sm">
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 shrink-0 rounded-full text-muted-foreground/70 hover:text-foreground"
        onClick={onPrevWeek}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <div className="flex items-center justify-between flex-1 px-2 sm:px-6">
        {Array.from({ length: 7 }, (_, i) => {
          const date = startDate.add(i, "day");
          const dateStr = date.format("YYYY-MM-DD");
          const isToday = dateStr === todayStr;
          const isSelected = dateStr === selectedDate;
          const scheduleCount = schedules.filter((s) => toDayjs(s.startTs).format("YYYY-MM-DD") === dateStr).length;

          return (
            <button
              key={i}
              onClick={() => onDateSelect(date)}
              className={cn(
                "flex flex-col items-center justify-center w-11 h-16 rounded-xl transition-all duration-200 relative group",
                isSelected
                  ? "bg-primary text-primary-foreground shadow-lg shadow-primary/20 scale-105"
                  : isToday
                    ? "bg-primary/10 text-primary font-medium"
                    : "hover:bg-muted/80 text-muted-foreground",
              )}
            >
              <span
                className={cn(
                  "text-[11px] font-medium uppercase tracking-wider mb-0.5",
                  isSelected ? "opacity-90" : "opacity-60 group-hover:opacity-80",
                )}
              >
                {date.format("ddd")}
              </span>
              <span className={cn("text-xl font-semibold leading-none", (isSelected || isToday) && "font-bold")}>{date.format("D")}</span>

              {/* Indicators */}
              <div className="flex gap-0.5 h-1 mt-1.5">
                {scheduleCount > 0 && <span className={cn("w-1 h-1 rounded-full", isSelected ? "bg-white/90" : "bg-foreground/40")} />}
                {scheduleCount > 2 && <span className={cn("w-1 h-1 rounded-full", isSelected ? "bg-white/90" : "bg-foreground/40")} />}
              </div>
            </button>
          );
        })}
      </div>

      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 shrink-0 rounded-full text-muted-foreground/70 hover:text-foreground"
        onClick={onNextWeek}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
};

export const ScheduleTimeline = ({ schedules, selectedDate, onDateClick, onScheduleEdit, className = "" }: ScheduleTimelineProps) => {
  const t = useTranslate();
  const [currentDate, setCurrentDate] = useState(selectedDate ? dayjs(selectedDate) : dayjs());
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (selectedDate) {
      setCurrentDate(dayjs(selectedDate));
    }
  }, [selectedDate]);

  const selectedDateStr = currentDate.format("YYYY-MM-DD");
  const isToday = selectedDateStr === dayjs().format("YYYY-MM-DD");

  const toDayjs = (ts: bigint) => dayjs(timestampDate(create(TimestampSchema, { seconds: ts, nanos: 0 })));

  const daySchedules = useMemo(() => {
    return schedules
      .filter((s) => toDayjs(s.startTs).format("YYYY-MM-DD") === selectedDateStr)
      .sort((a, b) => Number(a.startTs) - Number(b.startTs));
  }, [schedules, selectedDateStr]);

  const handleDateSelect = (date: Dayjs) => {
    setCurrentDate(date);
    onDateClick?.(date.format("YYYY-MM-DD"));
  };

  return (
    <div className={cn("flex flex-col h-full bg-background/50", className)}>
      {/* Header */}
      <div className="flex-none border-b bg-background/95 backdrop-blur z-20 sticky top-0 shadow-sm">
        <DateStrip
          currentDate={currentDate}
          selectedDate={selectedDateStr}
          schedules={schedules}
          onDateSelect={handleDateSelect}
          onPrevWeek={() => handleDateSelect(currentDate.subtract(1, "week"))}
          onNextWeek={() => handleDateSelect(currentDate.add(1, "week"))}
        />
      </div>

      {/* Agenda List Area */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 scroll-smooth">
        <div className="max-w-3xl lg:max-w-4xl xl:max-w-5xl 2xl:max-w-6xl mx-auto min-h-full flex flex-col">
          {/* Day Header */}
          <div className="mb-6 mt-2 flex items-baseline justify-between px-2">
            <div className="flex items-baseline gap-3">
              <h2 className="text-2xl font-bold tracking-tight text-foreground">
                {isToday ? t("common.today") : currentDate.format("dddd")}
              </h2>
              <span className="text-muted-foreground font-medium">{currentDate.format("MMMM D")}</span>
            </div>
            <span className="text-xs font-medium px-2.5 py-1 rounded-full bg-muted text-muted-foreground">
              {t("schedule.schedule-count", { count: daySchedules.length })}
            </span>
          </div>

          {daySchedules.length === 0 ? (
            <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground/40 pb-20 mt-10">
              <CalendarDays className="w-16 h-16 mb-4 opacity-20" />
              <p className="text-lg font-medium">{t("schedule.no-schedules")}</p>
            </div>
          ) : (
            <div className="relative flex flex-col gap-4 pb-20">
              {/* Vertical Connecting Line */}
              <div className="absolute left-[5rem] top-4 bottom-10 w-px bg-border/60" />

              {daySchedules.map((schedule) => {
                const startDate = toDayjs(schedule.startTs);
                const endDate = toDayjs(schedule.endTs);
                const isPast = endDate.isBefore(dayjs());
                const isCurrent = startDate.isBefore(dayjs()) && endDate.isAfter(dayjs());

                return (
                  <div key={schedule.name} className={cn("flex gap-6 relative group transition-opacity", isPast && "opacity-70")}>
                    {/* Time Column */}
                    <div className="w-[4rem] flex-none text-right pt-2.5 flex flex-col items-end">
                      <span className={cn("text-sm font-bold tabular-nums", isCurrent ? "text-primary" : "text-foreground/90")}>
                        {startDate.format("HH:mm")}
                      </span>
                    </div>

                    {/* Dot Indicator */}
                    <div className="absolute left-[5rem] -translate-x-1/2 pt-3.5 z-10 bg-background py-1">
                      <div
                        className={cn(
                          "w-2.5 h-2.5 rounded-full border-2 transition-all",
                          isCurrent ? "bg-primary border-primary ring-4 ring-primary/10" : "bg-background border-muted-foreground",
                          isPast && "bg-muted border-muted-foreground/30",
                        )}
                      />
                    </div>

                    {/* Content Card */}
                    <div className="flex-1 min-w-0">
                      <div
                        className={cn(
                          "relative w-full rounded-lg border transition-all duration-200 overflow-hidden",
                          "hover:shadow-sm hover:border-primary/20",
                          isCurrent ? "bg-primary/5 border-primary/20" : "bg-card border-border",
                        )}
                      >
                        {/* Main Click Area (View/Edit) */}
                        <div role="button" onClick={() => onScheduleEdit?.(schedule)} className="p-3 sm:p-4 cursor-pointer">
                          <div className="flex justify-between items-start gap-4">
                            <div className="space-y-1.5 min-w-0">
                              <h3 className={cn("font-semibold text-base truncate leading-none", isCurrent && "text-primary")}>
                                {schedule.title || t("schedule.untitled")}
                              </h3>

                              {/* Meta Info Row */}
                              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground/80">
                                {/* Duration */}
                                <div className="flex items-center gap-1.5">
                                  <Clock className="w-3 h-3 shrink-0 opacity-70" />
                                  <span>
                                    {Math.max(15, endDate.diff(startDate, "minute"))} {t("schedule.quick-input.minutes-abbr")}
                                  </span>
                                </div>

                                {/* Location */}
                                {schedule.location && (
                                  <div className="flex items-center gap-1.5 min-w-0 max-w-[150px]">
                                    <MapPin className="w-3 h-3 shrink-0 opacity-70" />
                                    <span className="truncate">{schedule.location}</span>
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>

                          {schedule.description && (
                            <div className="mt-2.5 pt-2.5 border-t border-border/40 text-xs text-muted-foreground/80 line-clamp-2 leading-relaxed">
                              {schedule.description}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
