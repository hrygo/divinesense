import { MessageSquarePlus } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { ConversationSummary } from "@/types/aichat";
import { ConversationItem } from "./ConversationItem";

interface ConversationHistoryPanelProps {
  className?: string;
  onSelectConversation?: (id: string) => void;
}

/**
 * 会话历史面板 - 统一入口设计
 *
 * 设计原则：
 * - 会话按时间分组，提升回溯效率
 * - 新建对话按钮已移至 Sidebar 顶部
 */
export function ConversationHistoryPanel({ className, onSelectConversation }: ConversationHistoryPanelProps) {
  const { t } = useTranslation();
  const {
    conversationSummaries,
    conversations,
    state,
    deleteConversation,
    selectConversation,
    updateConversationTitle,
    loadBlocks,
    refreshConversations,
  } = useAIChat();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [conversationToDelete, setConversationToDelete] = useState<string | null>(null);
  const [refreshingId, setRefreshingId] = useState<string | null>(null);

  // Use backend-provided messageCount instead of local messages array
  const loadedConversationIds = useMemo(
    () => new Set(conversations.filter((c) => (c.messageCount ?? 0) > 0).map((c) => c.id)),
    [conversations],
  );

  // 按时间分组会话
  const groupedConversations = useMemo(() => {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const yesterday = today - 24 * 60 * 60 * 1000;
    const thisWeek = today - 7 * 24 * 60 * 60 * 1000;

    const groups: { key: string; label: string; conversations: ConversationSummary[] }[] = [
      { key: "today", label: t("ai.aichat.sidebar.time-group-today"), conversations: [] },
      { key: "yesterday", label: t("ai.aichat.sidebar.time-group-yesterday"), conversations: [] },
      { key: "thisWeek", label: t("ai.aichat.sidebar.time-group-this-week"), conversations: [] },
      { key: "earlier", label: t("ai.aichat.sidebar.time-group-earlier"), conversations: [] },
    ];

    conversationSummaries.forEach((conv) => {
      const timestamp = conv.updatedAt;
      if (timestamp >= today) {
        groups[0].conversations.push(conv);
      } else if (timestamp >= yesterday) {
        groups[1].conversations.push(conv);
      } else if (timestamp >= thisWeek) {
        groups[2].conversations.push(conv);
      } else {
        groups[3].conversations.push(conv);
      }
    });

    // 只返回有内容的分组
    return groups.filter((g) => g.conversations.length > 0);
  }, [conversationSummaries, t]);

  const handleSelectConversation = (id: string) => {
    selectConversation(id);
    onSelectConversation?.(id);
  };

  const handleDeleteClick = (id: string) => {
    setConversationToDelete(id);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (conversationToDelete) {
      deleteConversation(conversationToDelete);
    }
    setDeleteDialogOpen(false);
    setConversationToDelete(null);
  };

  const handleTitleChange = (id: string, newTitle: string) => {
    updateConversationTitle(id, newTitle);
  };

  const handleRefresh = async (id: string) => {
    setRefreshingId(id);
    try {
      // Refresh conversation blocks from backend
      await loadBlocks(id);
      // Also refresh conversations to get latest messageCount
      await refreshConversations();
      // Re-select to trigger UI update
      if (state.currentConversationId === id) {
        selectConversation(id);
      }
    } catch (error) {
      console.error("Failed to refresh conversation:", error);
    } finally {
      setRefreshingId(null);
    }
  };

  const hasConversations = conversationSummaries.length > 0;

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* 会话列表 */}
      <div className="flex-1 overflow-y-auto">
        {hasConversations ? (
          <div className="flex flex-col py-1">
            {groupedConversations.map((group) => (
              <div key={group.key} className="mb-1">
                {/* 时间分组标签 */}
                <div className="px-3 py-1.5 text-xs font-medium text-muted-foreground uppercase tracking-wide">{group.label}</div>
                {/* 会话列表 */}
                <div className="flex flex-col gap-0.5 px-2">
                  {group.conversations.map((conversation) => (
                    <ConversationItem
                      key={conversation.id}
                      conversation={conversation}
                      isActive={conversation.id === state.currentConversationId}
                      onSelect={handleSelectConversation}
                      onDelete={handleDeleteClick}
                      onRefresh={handleRefresh}
                      onTitleChange={handleTitleChange}
                      isLoaded={loadedConversationIds.has(conversation.id)}
                      isRefreshing={refreshingId === conversation.id}
                    />
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState />
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={(open) => {
          setDeleteDialogOpen(open);
          if (!open) setConversationToDelete(null);
        }}
        onConfirm={handleConfirmDelete}
        title={t("ai.aichat.delete-conversation-title")}
        description={t("ai.aichat.delete-conversation-confirm")}
        confirmLabel={t("common.delete")}
        cancelLabel={t("common.cancel")}
        confirmVariant="destructive"
      />
    </div>
  );
}

function EmptyState() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center h-full p-4 text-center">
      <div className="w-12 h-12 rounded-2xl bg-muted flex items-center justify-center mb-3">
        <MessageSquarePlus className="w-5 h-5 text-muted-foreground" />
      </div>
      <h3 className="text-sm font-medium text-foreground mb-1">{t("ai.aichat.sidebar.no-conversations")}</h3>
      <p className="text-xs text-muted-foreground">{t("ai.aichat.sidebar.start-new-chat")}</p>
    </div>
  );
}
