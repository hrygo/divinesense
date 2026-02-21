import { Check, Circle } from "lucide-react";
import { cn } from "@/lib/utils";
import { SCHEDULE_PHASES } from "./phaseConfig";

interface PhaseProgressProps {
  currentPhase: number;
  isComplete?: boolean;
  hasError?: boolean;
  className?: string;
}

export function PhaseProgress({ currentPhase, isComplete, hasError, className }: PhaseProgressProps) {
  const phases = SCHEDULE_PHASES;

  return (
    <div className={cn("flex items-center justify-between w-full", className)}>
      {phases.map((phase, index) => {
        const isCompleted = index < currentPhase || isComplete;
        const isCurrent = index === currentPhase && !isComplete && !hasError;
        const isPending = index > currentPhase && !isComplete;

        return (
          <div key={phase.key} className="flex items-center flex-1 last:flex-none">
            <div className="flex flex-col items-center">
              <div
                className={cn(
                  "w-8 h-8 rounded-full flex items-center justify-center transition-all duration-300",
                  isCompleted && "bg-green-500 text-white",
                  isCurrent && !hasError && "bg-primary text-primary-foreground animate-pulse",
                  isPending && "bg-muted text-muted-foreground",
                  hasError && index <= currentPhase && "bg-destructive text-destructive-foreground",
                )}
              >
                {isCompleted ? <Check className="w-4 h-4" /> : <Circle className={cn("w-3 h-3", isCurrent && "animate-pulse")} />}
              </div>
            </div>
            {index < phases.length - 1 && (
              <div className="flex-1 h-0.5 mx-2">
                <div
                  className={cn(
                    "h-full transition-all duration-300",
                    isCompleted ? "bg-green-500" : "bg-muted",
                    hasError && index < currentPhase && "bg-destructive",
                  )}
                />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
