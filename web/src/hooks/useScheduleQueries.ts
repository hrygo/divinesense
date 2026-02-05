import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useCallback, useRef, useState } from "react";
import { scheduleServiceClient } from "@/connect";
import type {
  BatchCreateSchedulesRequest,
  BatchParseScheduleRequest,
  CheckConflictRequest,
  ListSchedulesRequest,
  ListSchedulesResponse,
  ParseAndCreateScheduleRequest,
  Schedule,
} from "@/types/proto/api/v1/schedule_service_pb";
import {
  BatchCreateSchedulesRequestSchema,
  BatchParseScheduleRequestSchema,
  CheckConflictRequestSchema,
  CreateScheduleRequestSchema,
  ListSchedulesRequestSchema,
  ParseAndCreateScheduleRequestSchema,
  ScheduleSchema,
} from "@/types/proto/api/v1/schedule_service_pb";

export type { ParsedEvent } from "./useScheduleAgent";
// Re-export ScheduleAgent hooks for convenience
export { scheduleAgentChatStream, useScheduleAgentChat } from "./useScheduleAgent";

/**
 * Streaming event from the Agent
 */
export interface StreamingEvent {
  type: "thinking" | "tool_use" | "tool_result" | "answer" | "error";
  data: string;
  timestamp: number;
}

/**
 * Hook state for streaming chat
 */
export interface StreamingChatState {
  isStreaming: boolean;
  events: StreamingEvent[];
  currentStep: string;
  finalAnswer: string;
  error: string | null;
}

/**
 * Hook to chat with Schedule Agent with streaming feedback
 * Provides real-time events for UI feedback
 */
export function useScheduleAgentStreamingChat() {
  const queryClient = useQueryClient();
  const [state, setState] = useState<StreamingChatState>({
    isStreaming: false,
    events: [],
    currentStep: "",
    finalAnswer: "",
    error: null,
  });
  const abortControllerRef = useRef<AbortController | null>(null);

  const startChat = useCallback(
    async (message: string, userTimezone = "Asia/Shanghai") => {
      // Reset state
      setState({
        isStreaming: true,
        events: [],
        currentStep: "",
        finalAnswer: "",
        error: null,
      });

      // Create abort controller for cancellation
      abortControllerRef.current = new AbortController();

      try {
        // Import dynamically to avoid circular dependencies
        const { scheduleAgentChatStream } = await import("./useScheduleAgent");

        const eventHandler = (event: { type: string; data: string }) => {
          const streamingEvent: StreamingEvent = {
            type: event.type as StreamingEvent["type"],
            data: event.data,
            timestamp: Date.now(),
          };

          setState((prev) => ({
            ...prev,
            events: [...prev.events, streamingEvent],
            currentStep: formatCurrentStep(event.type, event.data),
          }));
        };

        let finalContent = "";
        for await (const chunk of scheduleAgentChatStream(message, userTimezone, eventHandler)) {
          // Check for cancellation
          if (abortControllerRef.current?.signal.aborted) {
            break;
          }

          if (chunk.done) {
            finalContent = chunk.content || finalContent;
          } else if (chunk.content) {
            finalContent = chunk.content;
          }
        }

        // Update final state
        setState((prev) => ({
          ...prev,
          isStreaming: false,
          finalAnswer: finalContent,
        }));

        // Invalidate schedule queries to refresh
        queryClient.invalidateQueries({ queryKey: ["schedules"] });

        return finalContent;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : "Unknown error";
        setState((prev) => ({
          ...prev,
          isStreaming: false,
          error: errorMessage,
        }));
        throw error;
      }
    },
    [queryClient],
  );

  const cancelChat = useCallback(() => {
    abortControllerRef.current?.abort();
    setState((prev) => ({
      ...prev,
      isStreaming: false,
    }));
  }, []);

  const reset = useCallback(() => {
    setState({
      isStreaming: false,
      events: [],
      currentStep: "",
      finalAnswer: "",
      error: null,
    });
  }, []);

  return {
    ...state,
    startChat,
    cancelChat,
    reset,
  };
}

/**
 * Format current step for display
 */
function formatCurrentStep(type: string, data: string): string {
  switch (type) {
    case "thinking":
      return data || "Thinking...";
    case "tool_use": {
      const toolMatch = data.match(/^(\w+)(?::|$)/);
      const toolName = toolMatch ? toolMatch[1] : "tool";
      switch (toolName) {
        case "schedule_query":
          return "Checking schedules...";
        case "schedule_add":
          return "Creating schedule...";
        case "schedule_update":
          return "Updating schedule...";
        case "find_free_time":
          return "Finding free time...";
        default:
          return `Using ${toolName}...`;
      }
    }
    case "tool_result":
      return "Processing result...";
    case "answer":
      return "";
    default:
      return "";
  }
}

/**
 * Hook to fetch schedules for a specific date range with buffer days
 * Returns schedules for [targetDate-1, targetDate, targetDate+1] to cover cross-day schedules
 * Ensures fresh data from backend for accurate conflict detection
 */
export function useSchedulesForDate(date: Date | undefined) {
  return useQuery({
    queryKey: scheduleKeys.list({ _date: date?.toISOString() }),
    queryFn: async () => {
      if (!date) {
        return { schedules: [] };
      }

      // Get a larger range: [day-1 00:00, day+1 23:59:59] to cover cross-day schedules
      // This ensures we don't miss schedules that start on previous day and end on target day,
      // or start on target day and end on next day.
      const dayStart = dayjs(date).startOf("day").subtract(1, "day");
      const dayEnd = dayjs(date).endOf("day").add(1, "day");

      const request = create(ListSchedulesRequestSchema, {
        startTs: BigInt(Math.floor(dayStart.unix())),
        endTs: BigInt(Math.floor(dayEnd.unix())),
      } as Record<string, unknown>);

      const response = await scheduleServiceClient.listSchedules(request);
      return response;
    },
    enabled: !!date,
    staleTime: 0, // Always fetch fresh data for conflict detection
    gcTime: 1000 * 10, // Keep in cache for 10 seconds
  });
}

// Type for query parameters with string timestamps (for React Query cache keys)
// This avoids BigInt serialization issues in JSON.stringify()
export type ListSchedulesRequestWithStringTs = Omit<ListSchedulesRequest, "startTs" | "endTs"> & {
  startTs?: string;
  endTs?: string;
  month?: string; // For month-based grouping queries
  _date?: string; // Internal cache key for date-based queries
};

// Query keys factory for consistent cache management
export const scheduleKeys = {
  all: ["schedules"] as const,
  lists: () => [...scheduleKeys.all, "list"] as const,
  list: (filters: Partial<ListSchedulesRequestWithStringTs>) => [...scheduleKeys.lists(), filters] as const,
  details: () => [...scheduleKeys.all, "detail"] as const,
  detail: (name: string) => [...scheduleKeys.details(), name] as const,
  conflicts: () => [...scheduleKeys.all, "conflicts"] as const,
};

/**
 * Hook to fetch schedules with optional filters
 */
export function useSchedules(request: Partial<ListSchedulesRequestWithStringTs> = {}) {
  return useQuery({
    queryKey: scheduleKeys.list(request),
    queryFn: async () => {
      try {
        // Convert string timestamps to bigint for Protobuf serialization
        const requestWithBigint = {
          ...request,
          startTs: request.startTs ? BigInt(request.startTs) : undefined,
          endTs: request.endTs ? BigInt(request.endTs) : undefined,
        };
        const response = await scheduleServiceClient.listSchedules(
          create(ListSchedulesRequestSchema, requestWithBigint as Record<string, unknown>),
        );
        return response;
      } catch (error) {
        if (import.meta.env.DEV) {
          console.error("[useSchedules] API Error:", error);
        }
        throw error;
      }
    },
    staleTime: 1000 * 30, // 30 seconds (optimized for multi-user sync)
  });
}

/**
 * Hook to fetch schedules with time range optimization (today ± 15 days)
 */
export function useSchedulesOptimized(anchorDate?: Date) {
  // Calculate time range: anchorDate ± 15 days
  const now = anchorDate || new Date();
  const startOfRange = new Date(now);
  startOfRange.setDate(now.getDate() - 15);
  startOfRange.setHours(0, 0, 0, 0);

  const endOfRange = new Date(now);
  endOfRange.setDate(now.getDate() + 15);
  endOfRange.setHours(23, 59, 59, 999);

  // Convert to Unix timestamps (seconds) as STRING to avoid BigInt serialization issues
  // Will be converted to BigInt in useSchedules queryFn
  const startTs = Math.floor(startOfRange.getTime() / 1000).toString();
  const endTs = Math.floor(endOfRange.getTime() / 1000).toString();

  return useSchedules({
    startTs,
    endTs,
  });
}

/**
 * Hook to fetch a single schedule by name
 */
export function useSchedule(name: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: scheduleKeys.detail(name),
    queryFn: async () => {
      const schedule = await scheduleServiceClient.getSchedule({ name });
      return schedule;
    },
    enabled: options?.enabled ?? true,
    staleTime: 1000 * 30, // 30 seconds (should be >= list cache time for consistency)
  });
}

/**
 * Hook to create a new schedule
 */
export function useCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (scheduleToCreate: Partial<Schedule>) => {
      // Create the Schedule message first using create()
      const scheduleMessage = create(ScheduleSchema, scheduleToCreate as Record<string, unknown>);

      // Then create the request with the properly constructed schedule message
      const request = create(CreateScheduleRequestSchema, {
        schedule: scheduleMessage,
      } as Record<string, unknown>);

      const response = await scheduleServiceClient.createSchedule(request);
      return response;
    },
    onSuccess: (newSchedule) => {
      // Invalidate schedule lists to refetch
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
      // Add new schedule to cache
      queryClient.setQueryData(scheduleKeys.detail(newSchedule.name), newSchedule);
    },
  });
}

/**
 * Hook to update a schedule
 */
export function useUpdateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ schedule, updateMask }: { schedule: Partial<Schedule>; updateMask: string[] }) => {
      const updated = await scheduleServiceClient.updateSchedule({
        schedule: schedule as Schedule,
        updateMask: { paths: updateMask },
      });
      return updated;
    },
    onMutate: async ({ schedule }) => {
      if (!schedule.name) {
        return { previousSchedule: undefined };
      }

      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: scheduleKeys.detail(schedule.name) });

      // Snapshot previous value
      const previousSchedule = queryClient.getQueryData<Schedule>(scheduleKeys.detail(schedule.name));

      // Optimistically update
      if (previousSchedule) {
        // We can't easily merge partial schedule locally without logic
        // For now just setting it directly assuming it's substantial, or better just invalidate on success
        // queryClient.setQueryData(scheduleKeys.detail(schedule.name), schedule as Schedule);
      }

      return { previousSchedule };
    },
    onError: (_err, { schedule }, context) => {
      // Rollback on error
      if (context?.previousSchedule && schedule.name) {
        queryClient.setQueryData(scheduleKeys.detail(schedule.name), context.previousSchedule);
      }
    },
    onSuccess: (updatedSchedule) => {
      // Update cache with server response
      queryClient.setQueryData(scheduleKeys.detail(updatedSchedule.name), updatedSchedule);
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
    },
  });
}

/**
 * Hook to delete a schedule
 */
export function useDeleteSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) => {
      await scheduleServiceClient.deleteSchedule({ name });
      return name;
    },
    onSuccess: (name) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: scheduleKeys.detail(name) });
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
    },
  });
}

/**
 * Hook to check for schedule conflicts
 */
export function useCheckConflict() {
  return useMutation({
    mutationFn: async (request: Partial<CheckConflictRequest>) => {
      const response = await scheduleServiceClient.checkConflict(create(CheckConflictRequestSchema, request as Record<string, unknown>));
      return response;
    },
  });
}

/**
 * Hook to parse natural language and optionally create a schedule
 */
export function useParseAndCreateSchedule() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: Partial<ParseAndCreateScheduleRequest>) => {
      const response = await scheduleServiceClient.parseAndCreateSchedule(
        create(ParseAndCreateScheduleRequestSchema, request as Record<string, unknown>),
      );
      return response;
    },
    onSuccess: (response) => {
      // If a schedule was created, invalidate cache
      if (response.createdSchedule) {
        queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
        queryClient.setQueryData(scheduleKeys.detail(response.createdSchedule.name), response.createdSchedule);
      }
    },
  });
}

/**
 * Hook to parse natural language into a schedule without creating it
 * Useful for semantic search to understand time ranges
 */
export function useParseScheduleQuery() {
  return useMutation({
    mutationFn: async (text: string) => {
      const response = await scheduleServiceClient.parseAndCreateSchedule(
        create(ParseAndCreateScheduleRequestSchema, {
          text,
          autoConfirm: false,
        }),
      );
      return response.parsedSchedule;
    },
  });
}

/**
 * Hook to fetch schedules by month with buffer days for cross-month schedules
 * Query range: month start - 7 days to month end + 7 days
 * Handles pagination to ensure all schedules are fetched.
 */
export function useSchedulesByMonthGrouped(month: string) {
  return useQuery({
    queryKey: scheduleKeys.list({ month }),
    queryFn: async () => {
      // Calculate month range with buffer for cross-month schedules
      const monthStart = dayjs(month).startOf("month");
      const monthEnd = dayjs(month).endOf("month");
      const startTs = monthStart.subtract(7, "day").unix().toString();
      const endTs = monthEnd.add(7, "day").unix().toString();

      // Set a large page size to minimize pagination
      // A month can have at most 31 days, plus 14 days buffer = 45 days
      // Even with multiple schedules per day, 1000 should be more than enough
      const pageSize = 1000;

      const allSchedules: Schedule[] = [];
      let pageToken: string | undefined = undefined;
      let lastResponse: ListSchedulesResponse | undefined = undefined;

      do {
        const response = await scheduleServiceClient.listSchedules(
          create(ListSchedulesRequestSchema, {
            startTs,
            endTs,
            pageSize,
            pageToken,
          } as Record<string, unknown>),
        );

        lastResponse = response;
        allSchedules.push(...(response.schedules || []));
        pageToken = response.nextPageToken;
      } while (pageToken);

      // Return combined response with all schedules
      return {
        ...lastResponse,
        schedules: allSchedules,
      } as ListSchedulesResponse;
    },
    enabled: !!month,
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 10, // 10 minutes
  });
}

/**
 * Hook to check availability and find free time slots
 * Returns available time slots for a given date and duration
 */
export function useCheckAvailability() {
  const { data: schedulesData } = useSchedulesOptimized();
  const allSchedules = schedulesData?.schedules || [];

  /**
   * Find available time slots on a specific day
   */
  const findAvailableSlots = (
    date: Date,
    durationMinutes: number,
    options?: {
      startHour?: number;
      endHour?: number;
      excludeNames?: string[];
    },
  ) => {
    const { startHour = 8, endHour = 22, excludeNames = [] } = options || {};
    const slots: Array<{
      startTs: bigint;
      endTs: bigint;
      label: string;
    }> = [];

    const dayStart = dayjs(date).hour(startHour).minute(0).second(0);
    const dayEnd = dayjs(date).hour(endHour).minute(0).second(0);

    // Get schedules for the day, excluding specified names
    const daySchedules = allSchedules
      .filter((s) => {
        if (excludeNames.includes(s.name)) return false;
        const sStart = dayjs(Number(s.startTs) * 1000);
        return sStart.isSame(dayStart, "day");
      })
      .sort((a, b) => Number(a.startTs) - Number(b.startTs));

    // Find gaps between schedules
    let currentStart = dayStart;

    for (const schedule of daySchedules) {
      const sStart = dayjs(Number(schedule.startTs) * 1000);
      const sEnd = dayjs(Number(schedule.endTs) * 1000);

      // Skip schedules that end before our current start
      if (sEnd.isBefore(currentStart)) continue;

      // Skip schedules that start after our search end
      if (sStart.isAfter(dayEnd)) continue;

      // Check if there's a gap before this schedule
      if (sStart.isAfter(currentStart)) {
        const gapMinutes = sStart.diff(currentStart, "minute");
        if (gapMinutes >= durationMinutes) {
          const slotEnd = currentStart.add(durationMinutes, "minute");
          slots.push({
            startTs: BigInt(currentStart.unix()),
            endTs: BigInt(slotEnd.unix()),
            label: `${currentStart.format("HH:mm")} - ${slotEnd.format("HH:mm")}`,
          });
        }
      }

      // Move current start to after this schedule
      currentStart = sEnd;
    }

    // Check if there's a gap after the last schedule
    if (dayEnd.diff(currentStart, "minute") >= durationMinutes) {
      const slotEnd = currentStart.add(durationMinutes, "minute");
      slots.push({
        startTs: BigInt(currentStart.unix()),
        endTs: BigInt(slotEnd.unix()),
        label: `${currentStart.format("HH:mm")} - ${slotEnd.format("HH:mm")}`,
      });
    }

    return slots;
  };

  /**
   * Check if a specific time slot is available
   */
  const isSlotAvailable = (startTs: bigint, endTs: bigint, excludeNames?: string[]): boolean => {
    const start = Number(startTs);
    const end = Number(endTs);

    return !allSchedules.some((s) => {
      if (excludeNames?.includes(s.name)) return false;
      const sStart = Number(s.startTs);
      const sEnd = Number(s.endTs);
      return start < sEnd && end > sStart;
    });
  };

  return {
    findAvailableSlots,
    isSlotAvailable,
    schedules: allSchedules,
  };
}

/**
 * Hook to parse natural language for batch schedule creation
 * Returns preview of schedules to be created
 */
export function useBatchParseSchedule() {
  return useMutation({
    mutationFn: async (request: Partial<BatchParseScheduleRequest>) => {
      const response = await scheduleServiceClient.batchParseSchedule(
        create(BatchParseScheduleRequestSchema, request as Record<string, unknown>),
      );
      return response;
    },
  });
}

/**
 * Hook to create multiple schedules from a batch request
 */
export function useBatchCreateSchedules() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: Partial<BatchCreateSchedulesRequest>) => {
      const response = await scheduleServiceClient.batchCreateSchedules(
        create(BatchCreateSchedulesRequestSchema, request as Record<string, unknown>),
      );
      return response;
    },
    onSuccess: (response) => {
      // Invalidate cache after batch creation
      if (response.schedules && response.schedules.length > 0) {
        queryClient.invalidateQueries({ queryKey: scheduleKeys.lists() });
        // Add each created schedule to cache
        for (const schedule of response.schedules) {
          queryClient.setQueryData(scheduleKeys.detail(schedule.name), schedule);
        }
      }
    },
  });
}
