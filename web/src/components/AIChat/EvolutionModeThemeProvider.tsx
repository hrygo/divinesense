import { useEffect } from "react";
import { useAIChat } from "@/contexts/AIChatContext";

const EVOLUTION_MODE_BODY_CLASS = "evolution-mode";

/**
 * Evolution Mode Theme Provider
 *
 * Applies `.evolution-mode` class to document.body when Evolution Mode is enabled.
 * This enables all Evolution Mode visual styles defined in themes/evolution-mode.css.
 *
 * Should be placed near the root of the app, within AIChatProvider.
 */
export function EvolutionModeThemeProvider() {
  const { state } = useAIChat();
  const { evolutionMode } = state;

  useEffect(() => {
    if (typeof document === "undefined") return;

    // Track whether we added the class for proper cleanup
    let wasAdded = false;

    if (evolutionMode) {
      document.body.classList.add(EVOLUTION_MODE_BODY_CLASS);
      wasAdded = true;
    }

    // Cleanup on unmount - only remove if we added it
    return () => {
      if (wasAdded) {
        document.body.classList.remove(EVOLUTION_MODE_BODY_CLASS);
      }
    };
  }, [evolutionMode]);

  return null;
}
