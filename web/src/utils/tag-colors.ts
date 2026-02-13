/**
 * Tag Color System - Sticky Note Design DNA
 *
 * 便签颜色系统：基于标签自动映射便签背景颜色
 *
 * Design Philosophy:
 * - Apple Notes 风格彩色便签纸
 * - 6 色调色板：amber / rose / sky / emerald / violet / orange
 * - 标签规则映射 + 哈希回退
 * - 深色模式适配
 */

// ============================================================================
// Types
// ============================================================================

export type StickyColorKey = "amber" | "rose" | "sky" | "emerald" | "violet" | "orange";

export interface StickyColorScheme {
  bg: string;
  border: string;
  text: string;
  muted: string;
  tag: string;
}

// ============================================================================
// Color Palette - Apple Notes Style
// ============================================================================

export const STICKY_PALETTE: Record<StickyColorKey, { light: StickyColorScheme; dark: StickyColorScheme }> = {
  amber: {
    light: {
      bg: "bg-amber-50",
      border: "border-amber-200",
      text: "text-amber-900",
      muted: "text-amber-700/70",
      tag: "bg-amber-100/80",
    },
    dark: {
      bg: "dark:bg-amber-950/30",
      border: "dark:border-amber-800/40",
      text: "dark:text-amber-100",
      muted: "dark:text-amber-300/70",
      tag: "dark:bg-amber-900/30",
    },
  },
  rose: {
    light: {
      bg: "bg-rose-50",
      border: "border-rose-200",
      text: "text-rose-900",
      muted: "text-rose-700/70",
      tag: "bg-rose-100/80",
    },
    dark: {
      bg: "dark:bg-rose-950/30",
      border: "dark:border-rose-800/40",
      text: "dark:text-rose-100",
      muted: "dark:text-rose-300/70",
      tag: "dark:bg-rose-900/30",
    },
  },
  sky: {
    light: {
      bg: "bg-sky-50",
      border: "border-sky-200",
      text: "text-sky-900",
      muted: "text-sky-700/70",
      tag: "bg-sky-100/80",
    },
    dark: {
      bg: "dark:bg-sky-950/30",
      border: "dark:border-sky-800/40",
      text: "dark:text-sky-100",
      muted: "dark:text-sky-300/70",
      tag: "dark:bg-sky-900/30",
    },
  },
  emerald: {
    light: {
      bg: "bg-emerald-50",
      border: "border-emerald-200",
      text: "text-emerald-900",
      muted: "text-emerald-700/70",
      tag: "bg-emerald-100/80",
    },
    dark: {
      bg: "dark:bg-emerald-950/30",
      border: "dark:border-emerald-800/40",
      text: "dark:text-emerald-100",
      muted: "dark:text-emerald-300/70",
      tag: "dark:bg-emerald-900/30",
    },
  },
  violet: {
    light: {
      bg: "bg-violet-50",
      border: "border-violet-200",
      text: "text-violet-900",
      muted: "text-violet-700/70",
      tag: "bg-violet-100/80",
    },
    dark: {
      bg: "dark:bg-violet-950/30",
      border: "dark:border-violet-800/40",
      text: "dark:text-violet-100",
      muted: "dark:text-violet-300/70",
      tag: "dark:bg-violet-900/30",
    },
  },
  orange: {
    light: {
      bg: "bg-orange-50",
      border: "border-orange-200",
      text: "text-orange-900",
      muted: "text-orange-700/70",
      tag: "bg-orange-100/80",
    },
    dark: {
      bg: "dark:bg-orange-950/30",
      border: "dark:border-orange-800/40",
      text: "dark:text-orange-100",
      muted: "dark:text-orange-300/70",
      tag: "dark:bg-orange-900/30",
    },
  },
};

// ============================================================================
// Tag → Color Mapping Rules
// ============================================================================

/**
 * 标签颜色映射规则
 * 优先级：精确匹配 > 模糊匹配 > 哈希回退
 */
const TAG_COLOR_RULES: Record<string, StickyColorKey> = {
  // Work / 工作
  work: "sky",
  工作: "sky",
  office: "sky",
  办公: "sky",
  project: "sky",
  项目: "sky",

  // Personal / 个人
  personal: "rose",
  个人: "rose",
  life: "rose",
  生活: "rose",
  family: "rose",
  家庭: "rose",
  diary: "rose",
  日记: "rose",

  // Health / 健康
  health: "emerald",
  健康: "emerald",
  fitness: "emerald",
  运动: "emerald",
  exercise: "emerald",
  habit: "emerald",
  习惯: "emerald",

  // Idea / 创意
  idea: "violet",
  创意: "violet",
  creative: "violet",
  灵感: "violet",
  inspiration: "violet",
  design: "violet",
  设计: "violet",

  // Urgent / 紧急
  urgent: "orange",
  紧急: "orange",
  todo: "orange",
  待办: "orange",
  important: "orange",
  重要: "orange",
  asap: "orange",

  // Learning / 学习
  learning: "sky",
  学习: "sky",
  study: "sky",
  读书: "sky",
  book: "sky",

  // Finance / 财务
  finance: "amber",
  财务: "amber",
  money: "amber",
  钱: "amber",
  budget: "amber",
  预算: "amber",
};

/**
 * Pre-compiled word boundary regex rules for performance
 * Avoids creating new RegExp objects on every function call
 */
const COMPILED_WORD_BOUNDARY_RULES: Array<{
  regex: RegExp;
  color: StickyColorKey;
}> = Object.entries(TAG_COLOR_RULES).map(([keyword, color]) => ({
  regex: new RegExp(`\\b${keyword.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`, "i"),
  color,
}));

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Simple string hash function
 */
function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
}

/**
 * Get color key from tag using hash
 */
function hashToColor(tag: string): StickyColorKey {
  const colors: StickyColorKey[] = ["amber", "rose", "sky", "emerald", "violet", "orange"];
  const hash = hashString(tag.toLowerCase());
  return colors[hash % colors.length];
}

/**
 * Find matching color rule for a tag
 * Uses word boundary matching to avoid false positives (e.g., "worker" matching "work")
 */
function findColorRule(tag: string): StickyColorKey | null {
  const normalizedTag = tag.toLowerCase().trim();

  // Exact match
  if (TAG_COLOR_RULES[normalizedTag]) {
    return TAG_COLOR_RULES[normalizedTag];
  }

  // Word boundary match using pre-compiled regexes (more precise than simple includes)
  // Only match if the keyword appears as a complete word in the tag
  for (const { regex, color } of COMPILED_WORD_BOUNDARY_RULES) {
    if (regex.test(normalizedTag)) {
      return color;
    }
  }

  return null;
}

// ============================================================================
// Main API
// ============================================================================

/**
 * Get sticky note color based on memo tags
 *
 * @param tags - Array of tag strings from memo
 * @returns Color key for the sticky note
 *
 * Priority:
 * 1. First tag exact/partial match
 * 2. Hash-based color from first tag
 * 3. Default amber color (no tags)
 */
export function getMemoColor(tags: string[] | undefined): StickyColorKey {
  if (!tags || tags.length === 0) {
    return "amber"; // Default yellow for no tags
  }

  const firstTag = tags[0];

  // Try rule matching first
  const ruleColor = findColorRule(firstTag);
  if (ruleColor) {
    return ruleColor;
  }

  // Fallback to hash-based color
  return hashToColor(firstTag);
}

/**
 * Get complete color scheme for a sticky note
 *
 * @param tags - Array of tag strings from memo
 * @returns Combined light + dark color scheme classes
 */
export function getMemoColorClasses(tags: string[] | undefined): {
  key: StickyColorKey;
  bg: string;
  border: string;
  text: string;
  muted: string;
  tag: string;
} {
  const colorKey = getMemoColor(tags);
  const palette = STICKY_PALETTE[colorKey];

  return {
    key: colorKey,
    bg: `${palette.light.bg} ${palette.dark.bg}`,
    border: `${palette.light.border} ${palette.dark.border}`,
    text: `${palette.light.text} ${palette.dark.text}`,
    muted: `${palette.light.muted} ${palette.dark.muted}`,
    tag: `${palette.light.tag} ${palette.dark.tag}`,
  };
}

/**
 * Get all available color keys
 */
export function getAvailableColors(): StickyColorKey[] {
  return Object.keys(STICKY_PALETTE) as StickyColorKey[];
}
