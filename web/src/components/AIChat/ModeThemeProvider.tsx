import { useEffect } from "react";
import { useAIChat } from "@/contexts/AIChatContext";
import type { AIMode } from "@/types/aichat";

const GEEK_MODE_BODY_CLASS = "geek-mode";
const EVOLUTION_MODE_BODY_CLASS = "evolution-mode";

/**
 * Mode Theme Provider - 统一模式主题提供器
 *
 * 根据当前 AI 模式动态添加 CSS class 到 document.body：
 * - normal: 无特殊 class（默认样式）
 * - geek: 添加 `.geek-mode` class（绿色终端主题）
 * - evolution: 添加 `.evolution-mode` class（紫色进化主题）
 *
 * 应放置在 AIChatProvider 内部，靠近应用根节点。
 */
export function ModeThemeProvider() {
  const { state } = useAIChat();
  const currentMode = state.currentMode || "normal";

  useEffect(() => {
    if (typeof document === "undefined") return;

    const body = document.body;

    // Remove all mode classes first
    body.classList.remove(GEEK_MODE_BODY_CLASS, EVOLUTION_MODE_BODY_CLASS);

    // Add appropriate class based on current mode
    if (currentMode === "geek") {
      body.classList.add(GEEK_MODE_BODY_CLASS);
    } else if (currentMode === "evolution") {
      body.classList.add(EVOLUTION_MODE_BODY_CLASS);
    }

    // Cleanup function
    return () => {
      body.classList.remove(GEEK_MODE_BODY_CLASS, EVOLUTION_MODE_BODY_CLASS);
    };
  }, [currentMode]);

  return null;
}

/**
 * Hook to check if a specific mode is active
 */
export function useActiveMode(mode: AIMode): boolean {
  const { state } = useAIChat();
  return state.currentMode === mode;
}
