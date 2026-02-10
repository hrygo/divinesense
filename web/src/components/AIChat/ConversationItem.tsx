import { Pencil, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { translateTitle } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import type { ConversationSummary } from "@/types/aichat";
import { TitleEditDialog } from "./TitleEditDialog";

interface ConversationItemProps {
  conversation: ConversationSummary;
  isActive: boolean;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  onRefresh?: (id: string) => void;
  onTitleChange?: (id: string, newTitle: string) => void;
  className?: string;
  isLoaded?: boolean; // Whether this conversation has been loaded with messages
  isRefreshing?: boolean; // Whether this conversation is currently being refreshed
}

export function ConversationItem({
  conversation,
  isActive,
  onSelect,
  onDelete,
  onRefresh,
  onTitleChange,
  className,
  isLoaded = false,
  isRefreshing = false,
}: ConversationItemProps) {
  const { t } = useTranslation();
  const [editDialogOpen, setEditDialogOpen] = useState(false);

  // Display message count: show "..." if not loaded yet, 0 if truly empty
  const displayMessageCount = isLoaded ? conversation.messageCount : "...";

  const handleTitleChange = (newTitle: string) => {
    onTitleChange?.(conversation.id, newTitle);
  };

  return (
    <>
      <div className={cn("group relative rounded-lg transition-all", isActive ? "bg-accent" : "hover:bg-muted", className)}>
        <button
          onClick={() => onSelect(conversation.id)}
          className="w-full text-left px-3 py-2.5 pr-20"
          aria-label={`Select conversation: ${translateTitle(conversation.title, t)}`}
        >
          <div className="flex flex-col min-w-0">
            <h3 className="font-medium text-sm text-foreground truncate group-hover:text-primary transition-colors">
              {translateTitle(conversation.title, t)}
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              {displayMessageCount === "..."
                ? t("ai.aichat.sidebar.message-count", { count: 0 })
                : t("ai.aichat.sidebar.message-count", { count: displayMessageCount })}{" "}
              · {formatTime(conversation.updatedAt, t)}
            </p>
          </div>
        </button>

        {/* Action Buttons - Show on hover */}
        <div className="absolute right-2 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
          <RefreshButton conversationId={conversation.id} onRefresh={onRefresh} isRefreshing={isRefreshing} />
          <EditButton onEdit={() => setEditDialogOpen(true)} />
          <DeleteButton conversationId={conversation.id} onDelete={onDelete} />
        </div>
      </div>

      {/* Title Edit Dialog */}
      <TitleEditDialog conversation={conversation} open={editDialogOpen} onOpenChange={setEditDialogOpen} onSuccess={handleTitleChange} />
    </>
  );
}

interface EditButtonProps {
  onEdit: () => void;
}

interface RefreshButtonProps {
  conversationId: string;
  onRefresh?: ((id: string) => void) | undefined;
  isRefreshing?: boolean;
}

function RefreshButton({ conversationId, onRefresh, isRefreshing = false }: RefreshButtonProps) {
  const { t } = useTranslation();

  if (!onRefresh) return null;

  const handleRefresh = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await onRefresh(conversationId);
  };

  return (
    <button
      onClick={handleRefresh}
      disabled={isRefreshing}
      className={cn(
        "flex items-center justify-center",
        "w-8 h-8 rounded-lg",
        "text-muted-foreground",
        "hover:text-primary",
        "hover:bg-primary/10",
        "transition-all duration-200",
      )}
      aria-label={t("common.refresh")}
      title={t("common.refresh")}
    >
      <RefreshCw className={cn("w-4 h-4", isRefreshing && "animate-spin")} />
    </button>
  );
}

function EditButton({ onEdit }: EditButtonProps) {
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onEdit();
      }}
      className={cn(
        "flex items-center justify-center",
        "w-8 h-8 rounded-lg",
        "text-muted-foreground",
        "hover:text-primary",
        "hover:bg-primary/10",
        "transition-all duration-200",
      )}
      aria-label="Edit title"
      title="Edit title"
    >
      <Pencil className="w-4 h-4" />
    </button>
  );
}

interface DeleteButtonProps {
  conversationId: string;
  onDelete: (id: string) => void;
}

function DeleteButton({ conversationId, onDelete }: DeleteButtonProps) {
  const { t } = useTranslation();

  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onDelete(conversationId);
      }}
      className={cn(
        "flex items-center justify-center",
        "w-8 h-8 rounded-lg",
        "text-muted-foreground",
        "hover:text-destructive",
        "hover:bg-destructive/10",
        "transition-all duration-200",
      )}
      aria-label={t("common.delete")}
      title={t("common.delete")}
    >
      <Trash2 className="w-4 h-4" />
    </button>
  );
}

function formatTime(timestamp: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return t("ai.aichat.sidebar.time-just-now");
  if (diffMins < 60) return t("ai.aichat.sidebar.time-minutes-ago", { count: diffMins });
  if (diffHours < 24) return t("ai.aichat.sidebar.time-hours-ago", { count: diffHours });
  if (diffDays < 7) return t("ai.aichat.sidebar.time-days-ago", { count: diffDays });

  return date.toLocaleDateString();
}
