import { Clock, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { SidebarTab } from "@/types/aichat";

interface SidebarTabsProps {
  activeTab: SidebarTab;
  onTabChange: (tab: SidebarTab) => void;
  className?: string;
}

/**
 * Modern segmented control tabs for AI Chat sidebar
 * Clean pill-style design with smooth transitions
 */
export function SidebarTabs({ activeTab, onTabChange, className }: SidebarTabsProps) {
  const { t } = useTranslation();

  return (
    <div className={cn("w-full", className)}>
      {/* Segmented Control Container */}
      <div className="flex p-1 bg-muted rounded-lg">
        <TabButton
          active={activeTab === "history"}
          onClick={() => onTabChange("history")}
          icon={<Clock className="w-4 h-4" />}
          label={t("ai.aichat.sidebar.history")}
        />
        <TabButton
          active={activeTab === "memos"}
          onClick={() => onTabChange("memos")}
          icon={<FileText className="w-4 h-4" />}
          label={t("ai.aichat.sidebar.memos")}
        />
      </div>
    </div>
  );
}

interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}

function TabButton({ active, onClick, icon, label }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex-1 flex items-center justify-center gap-2",
        "py-2 px-3 rounded-lg",
        "text-sm font-medium",
        "transition-all duration-200 ease-in-out",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-blue-500",
        active ? "bg-background text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
      )}
      aria-pressed={active}
    >
      {icon}
      <span className="truncate">{label}</span>
    </button>
  );
}
