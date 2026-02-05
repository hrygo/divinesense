import { MessageSquarePlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";
import { ParrotAgentType } from "@/types/parrot";
import { ConversationHistoryPanel } from "./ConversationHistoryPanel";
import { ReferencedMemosPanel } from "./ReferencedMemosPanel";
import { SidebarTabs } from "./SidebarTabs";

interface AIChatSidebarProps {
  className?: string;
  onClose?: () => void;
}

/**
 * AI Chat Sidebar - 统一入口设计
 *
 * 设计原则：
 * - 新建对话置顶，作为最重要的操作
 * - 移除能力切换按钮，完全依赖智能路由
 * - 用户无需理解系统内部能力边界
 */
export function AIChatSidebar({ className, onClose }: AIChatSidebarProps) {
  const { t } = useTranslation();
  const { state, setSidebarTab, createConversation } = useAIChat();
  const { sidebarTab } = state;

  const handleTabChange = (tab: SidebarTab) => {
    setSidebarTab(tab);
  };

  // 统一入口：直接创建会话，使用 AUTO（由后端三层路由决定使用哪个代理）
  const handleStartNewChat = () => {
    createConversation(ParrotAgentType.AUTO);
    onClose?.();
  };

  return (
    <aside className={cn("flex flex-col h-full bg-sidebar", className)}>
      {/* 新建对话按钮 - 置顶 */}
      <div className="px-4 pt-4 pb-2 shrink-0">
        <button
          onClick={handleStartNewChat}
          className={cn(
            "w-full flex items-center justify-center gap-2 px-3 py-2",
            "bg-sidebar-primary hover:bg-sidebar-primary/90",
            "text-sidebar-primary-foreground text-sm font-medium rounded-lg transition-all",
            "shadow-sm active:scale-[0.98]",
          )}
        >
          <MessageSquarePlus className="w-4 h-4" />
          {t("ai.aichat.sidebar.new-chat")}
        </button>
      </div>

      {/* Tabs */}
      <div className="px-4 pb-2 flex-shrink-0">
        <SidebarTabs activeTab={sidebarTab} onTabChange={handleTabChange} />
      </div>

      {/* Panel Content */}
      <div className="flex-1 overflow-hidden min-h-0">
        {sidebarTab === "history" && <ConversationHistoryPanel className="h-full" onSelectConversation={onClose} />}
        {sidebarTab === "memos" && <ReferencedMemosPanel className="h-full" />}
      </div>
    </aside>
  );
}
