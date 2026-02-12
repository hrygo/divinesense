export const LOCALSTORAGE_DEBOUNCE_DELAY = 500;

export const FOCUS_MODE_STYLES = {
  backdrop: "fixed inset-0 bg-black/20 backdrop-blur-sm z-[60]",
  container: {
    // Fixed at bottom, expand upward to full screen
    base: "fixed z-[70] bottom-0 left-0 right-0 shadow-lg border-border h-screen overflow-y-auto animate-in slide-in-from-bottom duration-300",
    spacing: "p-4 sm:p-6 md:p-8",
  },
  transition: "transition-all duration-300 ease-in-out",
  exitButton: "absolute top-2 right-2 z-10 opacity-60 hover:opacity-100",
} as const;

export const EDITOR_HEIGHT = {
  // Max height for normal mode - focus mode uses flex-1 to grow dynamically
  normal: "max-h-[50vh]",
} as const;

export const TOOLBAR_BUTTON_STYLES = {
  base: "h-9 w-9 shrink-0 rounded-xl transition-all duration-200",
  ghost: "hover:bg-muted active:scale-95",
  primary: "bg-primary text-primary-foreground shadow-md shadow-primary/20 hover:bg-primary/90 active:scale-95 transition-all duration-200",
} as const;
