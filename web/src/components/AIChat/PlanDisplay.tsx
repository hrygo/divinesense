/**
 * PlanDisplay - Component for displaying Orchestrator task plans
 *
 * Shows the task decomposition and execution progress to the user,
 * providing transparency into the multi-agent orchestration process.
 *
 * @see docs/research/orchestrator-workers-research.md
 * @see Issue #169
 */

import { CheckCircle2, Circle, Loader2, XCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { getTaskProgress, getTaskSummary } from "@/hooks/useOrchestratorEvents";
import { cn } from "@/lib/utils";
import type { OrchestratorTask, TaskStatus } from "@/types/parrot";

interface PlanDisplayProps {
  analysis: string | null;
  tasks: OrchestratorTask[];
  isParallel: boolean;
  className?: string;
}

/**
 * Get icon for task status
 */
function TaskStatusIcon({ status }: { status: TaskStatus }) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="size-4 text-green-500" />;
    case "running":
      return <Loader2 className="size-4 text-blue-500 animate-spin" />;
    case "failed":
      return <XCircle className="size-4 text-red-500" />;
    default:
      return <Circle className="size-4 text-gray-400" />;
  }
}

/**
 * Get background color for task status
 */
function getTaskStatusBg(status: TaskStatus): string {
  switch (status) {
    case "completed":
      return "bg-green-50 dark:bg-green-950/30";
    case "running":
      return "bg-blue-50 dark:bg-blue-950/30";
    case "failed":
      return "bg-red-50 dark:bg-red-950/30";
    default:
      return "bg-gray-50 dark:bg-gray-800/50";
  }
}

/**
 * Get agent display name
 */
function getAgentDisplayName(agent: string): string {
  const agentNames: Record<string, string> = {
    memo: "📝 灰灰",
    schedule: "📅 时巧",
  };
  return agentNames[agent] || agent;
}

/**
 * PlanDisplay component
 */
export function PlanDisplay({ analysis, tasks, isParallel, className }: PlanDisplayProps) {
  const { t } = useTranslation();
  const progress = getTaskProgress(tasks);
  const summary = getTaskSummary(tasks);

  if (tasks.length === 0) {
    return null;
  }

  return (
    <div className={cn("rounded-lg border border-border bg-card p-4", className)}>
      {/* Analysis section */}
      {analysis && (
        <div className="mb-4">
          <h4 className="text-sm font-medium text-muted-foreground mb-1">{t("ai.orchestrator.analysis", "任务分析")}</h4>
          <p className="text-sm text-foreground">{analysis}</p>
        </div>
      )}

      {/* Progress bar */}
      <div className="mb-4">
        <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
          <span>
            {isParallel ? t("ai.orchestrator.parallelExecution", "并行执行中") : t("ai.orchestrator.sequentialExecution", "顺序执行中")}
          </span>
          <span>
            {summary.completed}/{summary.total} {t("ai.orchestrator.completed", "完成")}
          </span>
        </div>
        <div className="h-2 bg-muted rounded-full overflow-hidden">
          <div className="h-full bg-primary transition-all duration-300" style={{ width: `${progress}%` }} />
        </div>
      </div>

      {/* Task list */}
      <div className="space-y-2">
        {tasks.map((task, index) => (
          <div
            key={`${task.agent}-${index}`}
            className={cn("flex items-start gap-3 p-3 rounded-md transition-colors", getTaskStatusBg(task.status))}
          >
            <TaskStatusIcon status={task.status} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span className="text-xs font-medium text-muted-foreground">{getAgentDisplayName(task.agent)}</span>
                {isParallel && <span className="text-xs text-muted-foreground">#{index + 1}</span>}
              </div>
              <p className="text-sm text-foreground truncate">{task.purpose}</p>
              {task.error && <p className="text-xs text-red-500 mt-1">{task.error}</p>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default PlanDisplay;
