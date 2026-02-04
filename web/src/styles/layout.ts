/**
 * Layout Spacing Constants
 *
 * Unified spacing specification for DivineSense layouts.
 * All Layout components should follow these constants for consistency.
 *
 * @see docs/research/layout-spacing-unification.md
 */

/**
 * Standard layout spacing values
 *
 * Token definitions:
 * - sidebarWidth: Standard desktop sidebar width (288px)
 * - sidebarPadding: Internal padding for sidebar content
 * - mobileHeaderHeight: Fixed mobile header height (56px)
 * - paddingTop: Top spacing for main content (responsive)
 * - paddingBottom: Bottom spacing for main content
 * - paddingX: Horizontal padding for main content (responsive)
 * - maxWidth: Maximum content width for readability (1600px)
 */
export const LAYOUT_SPACING = {
  /** Sidebar width: 288px (18rem) - unified across all layouts */
  sidebarWidth: "w-72",

  /** Sidebar internal padding: horizontal 12px, vertical 24px */
  sidebarPadding: "px-3 py-6",

  /** Mobile header height: 56px (3.5rem) */
  mobileHeaderHeight: "h-14",

  /** Main content top padding: mobile 16px, desktop 24px */
  paddingTopMobile: "pt-4",
  paddingTopDesktop: "pt-6",

  /** Main content bottom padding: 32px */
  paddingBottom: "pb-8",

  /** Main content horizontal padding: mobile 16px, desktop 24px */
  paddingXMobile: "px-4",
  paddingXDesktop: "sm:px-6",

  /** Maximum content width: 1600px (100rem) for readability */
  maxWidth: "max-w-[100rem]",
} as const;

/**
 * Helper function to build spacing class string
 *
 * This function can be used in future refactoring to replace hardcoded className strings.
 * Currently, Layout components use inline strings for compatibility with existing patterns.
 *
 * Usage example (for future use):
 * ```tsx
 * const spacingClass = buildSpacingClass({
 *   width: "w-full",
 *   mx: "mx-auto",
 *   paddingX: { mobile: "px-4", desktop: "sm:px-6" },
 *   paddingTop: { mobile: "pt-4", desktop: "sm:pt-6" },
 *   paddingBottom: "pb-8",
 *   maxWidth: "max-w-[100rem]",
 * });
 * ```
 */
export function buildSpacingClass(config: {
  width?: string;
  mx?: string;
  paddingX?: { mobile: string; desktop: string } | string;
  paddingTop?: { mobile: string; desktop: string } | string;
  paddingBottom?: string;
  maxWidth?: string;
}): string {
  const classes: string[] = [];

  if (config.width) classes.push(config.width);
  if (config.mx) classes.push(config.mx);

  if (config.paddingX) {
    if (typeof config.paddingX === "string") {
      classes.push(config.paddingX);
    } else {
      classes.push(config.paddingX.mobile, config.paddingX.desktop);
    }
  }

  if (config.paddingTop) {
    if (typeof config.paddingTop === "string") {
      classes.push(config.paddingTop);
    } else {
      classes.push(config.paddingTop.mobile, config.paddingTop.desktop);
    }
  }

  if (config.paddingBottom) classes.push(config.paddingBottom);
  if (config.maxWidth) classes.push(config.maxWidth);

  return classes.join(" ");
}
