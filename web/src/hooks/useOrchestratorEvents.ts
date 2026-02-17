/**
 * useOrchestratorEvents - Hook for handling Orchestrator-Workers events
 *
 * This hook processes events from the Orchestrator and provides
 * structured data for UI rendering.
 *
 * @see docs/research/orchestrator-workers-research.md
 * @see Issue #169
 */
import { useCallback, useState } from "react";
import {
  type OrchestratorPlanEvent,
  type OrchestratorTask,
  type OrchestratorTaskEndEvent,
  type OrchestratorTaskStartEvent,
  ParrotEventType,
  type TaskStatus,
} from "@/types/parrot";

/**
 * Orchestrator state for UI rendering
 */
export interface OrchestratorState {
  // Current plan analysis
  analysis: string | null;

  // Tasks with their current status
  tasks: OrchestratorTask[];

  // Whether tasks are executed in parallel
  isParallel: boolean;

  // Current phase of orchestration
  phase: "idle" | "planning" | "executing" | "completed" | "error";

  // Error message if any
  error: string | null;
}

const initialState: OrchestratorState = {
  analysis: null,
  tasks: [],
  isParallel: false,
  phase: "idle",
  error: null,
};

/**
 * Helper to handle parsing errors with UI feedback
 */
function handleParseError(context: string, error: unknown): Partial<OrchestratorState> {
  const errorMsg = error instanceof Error ? error.message : String(error);
  console.error(`${context}:`, error);
  return {
    phase: "error",
    error: `${context}: ${errorMsg}`,
  };
}

/**
 * Hook for handling Orchestrator events
 *
 * @example
 * ```tsx
 * const { state, handleEvent, reset } = useOrchestratorEvents();
 *
 * // In your event handler
 * handleEvent(eventType, eventData);
 *
 * // Render based on state
 * if (state.phase === "planning") {
 *   return <PlanDisplay analysis={state.analysis} tasks={state.tasks} />;
 * }
 * ```
 */
export function useOrchestratorEvents() {
  const [state, setState] = useState<OrchestratorState>(initialState);

  /**
   * Handle an orchestrator event
   */
  const handleEvent = useCallback((eventType: string, eventData: string) => {
    switch (eventType) {
      case ParrotEventType.PLAN: {
        try {
          const plan = JSON.parse(eventData) as OrchestratorPlanEvent;
          setState((prev) => ({
            ...prev,
            analysis: plan.analysis,
            tasks: plan.tasks.map((t) => ({ ...t, status: "pending" as TaskStatus })),
            isParallel: plan.parallel,
            phase: "executing",
            error: null,
          }));
        } catch (e) {
          setState((prev) => ({ ...prev, ...handleParseError("Failed to parse plan event", e) }));
        }
        break;
      }

      case ParrotEventType.TASK_START: {
        try {
          const taskStart = JSON.parse(eventData) as OrchestratorTaskStartEvent;
          setState((prev) => {
            const newTasks = [...prev.tasks];
            if (newTasks[taskStart.index]) {
              newTasks[taskStart.index] = {
                ...newTasks[taskStart.index],
                status: "running",
              };
            }
            return { ...prev, tasks: newTasks };
          });
        } catch (e) {
          setState((prev) => ({
            ...prev,
            ...handleParseError("Failed to parse task_start event", e),
          }));
        }
        break;
      }

      case ParrotEventType.TASK_END: {
        try {
          const taskEnd = JSON.parse(eventData) as OrchestratorTaskEndEvent;
          setState((prev) => {
            const newTasks = [...prev.tasks];
            if (newTasks[taskEnd.index]) {
              newTasks[taskEnd.index] = {
                ...newTasks[taskEnd.index],
                status: taskEnd.status,
                error: taskEnd.error,
              };
            }

            // Check if all tasks are completed
            const allCompleted = newTasks.every((t) => t.status === "completed" || t.status === "failed");
            const hasError = newTasks.some((t) => t.status === "failed");

            return {
              ...prev,
              tasks: newTasks,
              phase: allCompleted ? (hasError ? "error" : "completed") : prev.phase,
            };
          });
        } catch (e) {
          setState((prev) => ({
            ...prev,
            ...handleParseError("Failed to parse task_end event", e),
          }));
        }
        break;
      }

      case ParrotEventType.ERROR: {
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: eventData,
        }));
        break;
      }

      case ParrotEventType.DECOMPOSE_START: {
        // Show analyzing state during task decomposition
        setState((prev) => ({
          ...prev,
          phase: "planning",
          error: null,
        }));
        break;
      }

      case ParrotEventType.DECOMPOSE_END: {
        // Decomposition complete, show task preview if available
        try {
          const data = JSON.parse(eventData);
          if (data.task_count) {
            // Pre-allocate tasks array for visual feedback
            setState((prev) => ({
              ...prev,
              tasks: Array(data.task_count)
                .fill(null)
                .map((_, i) => ({
                  id: `pending-${i}`,
                  agent: "",
                  input: "",
                  purpose: "",
                  status: "pending" as TaskStatus,
                })),
            }));
          }
        } catch {
          // Ignore parse errors for decompose_end
        }
        break;
      }

      default:
        // Ignore other event types
        break;
    }
  }, []);

  /**
   * Reset the orchestrator state
   */
  const reset = useCallback(() => {
    setState(initialState);
  }, []);

  /**
   * Start a new orchestration
   */
  const startPlanning = useCallback(() => {
    setState({
      ...initialState,
      phase: "planning",
    });
  }, []);

  return {
    state,
    handleEvent,
    reset,
    startPlanning,
  };
}

/**
 * Helper function to get task progress percentage
 */
export function getTaskProgress(tasks: OrchestratorTask[]): number {
  if (tasks.length === 0) return 0;
  const completed = tasks.filter((t) => t.status === "completed" || t.status === "failed").length;
  return Math.round((completed / tasks.length) * 100);
}

/**
 * Helper function to get task status summary
 */
export function getTaskSummary(tasks: OrchestratorTask[]): {
  total: number;
  pending: number;
  running: number;
  completed: number;
  failed: number;
} {
  return {
    total: tasks.length,
    pending: tasks.filter((t) => t.status === "pending").length,
    running: tasks.filter((t) => t.status === "running").length,
    completed: tasks.filter((t) => t.status === "completed").length,
    failed: tasks.filter((t) => t.status === "failed").length,
  };
}
