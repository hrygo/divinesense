import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { Routes } from "@/router";
import type { AIMode } from "@/types/aichat";

interface RouteHeaderImageProps {
  mode?: AIMode;
}

const RouteHeaderImage = ({ mode = "normal" }: RouteHeaderImageProps) => {
  const location = useLocation();
  const { i18n } = useTranslation();
  const path = location.pathname;

  // Detect if language is Chinese (simplified or traditional)
  const isZh = i18n.language.startsWith("zh");
  const suffix = isZh ? "-zh" : "";

  let headerName = "";

  if (path === Routes.HOME) headerName = "memos";
  else if (path.startsWith(Routes.EXPLORE)) headerName = "explore";
  else if (path.startsWith(Routes.ARCHIVED)) headerName = "memos";
  else if (path.startsWith("/u/"))
    headerName = "memos"; // Profile
  else if (path.startsWith(Routes.CHAT)) headerName = "ai";
  else if (path.startsWith(Routes.SCHEDULE)) headerName = "schedule";
  else if (path.startsWith(Routes.REVIEW)) headerName = "review";
  else if (path.startsWith(Routes.KNOWLEDGE_GRAPH)) headerName = "knowledge";
  else if (path.startsWith(Routes.ATTACHMENTS)) headerName = "files";
  else if (path.startsWith(Routes.INBOX)) headerName = "inbox";
  else if (path.startsWith(Routes.SETTING)) headerName = "memos";
  else if (path.startsWith("/memos/")) headerName = "memos"; // Detail

  if (!headerName) return null;

  const headerSrc = `/headers/header-${headerName}${suffix}.svg`;

  // 根据模式应用动效
  const getAnimationClass = (modeParam: AIMode) => {
    switch (modeParam) {
      case "geek":
        return "divine-logo-geek";
      case "evolution":
        return "divine-logo-evolution";
      default:
        return "divine-logo-normal";
    }
  };

  return (
    <img
      src={headerSrc}
      alt="Page Header"
      className={cn("h-8 w-auto object-contain select-none opacity-90 dark:opacity-100", getAnimationClass(mode))}
    />
  );
};

export default RouteHeaderImage;
