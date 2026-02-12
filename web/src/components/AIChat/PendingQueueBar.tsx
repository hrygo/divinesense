import { ListVideo, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";

interface PendingQueueBarProps {
  className?: string;
}

export function PendingQueueBar({ className }: PendingQueueBarProps) {
  const { t } = useTranslation();
  const { state, removeFromPendingQueue, clearPendingQueue } = useAIChat();
  const { pendingQueue } = state;

  if (pendingQueue.messages.length === 0) {
    return null;
  }

  return (
    <div
      className={cn(
        "flex flex-col gap-2 p-3 bg-muted/50 border-t border-border animate-in slide-in-from-bottom-2 fade-in duration-200",
        className,
      )}
    >
      <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
        <div className="flex items-center gap-1.5 font-medium text-primary">
          <ListVideo className="w-3.5 h-3.5" />
          <span>{t("ai.pending_queue.title", { count: pendingQueue.messages.length })}</span>
        </div>
        <Button variant="ghost" size="sm" className="h-5 px-2 text-[10px] hover:text-destructive" onClick={clearPendingQueue}>
          {t("ai.pending_queue.clear_all")}
        </Button>
      </div>

      <div className="flex flex-col gap-1.5 max-h-[120px] overflow-y-auto pr-1">
        {pendingQueue.messages.map((msg) => (
          <div
            key={msg.id}
            className="flex items-start justify-between gap-2 p-2 rounded-md bg-background border border-border/50 text-sm group"
          >
            <span className="line-clamp-2 text-foreground/80 break-all">{msg.content}</span>
            <button
              onClick={() => removeFromPendingQueue(msg.id)}
              className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
