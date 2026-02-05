/**
 * Chat Configuration Constants
 * 聊天相关配置常量 - 统一管理，避免散落在各处
 */

/** Message cache configuration */
export const MESSAGE_CONFIG = {
  /** Maximum MSG messages to cache per conversation */
  CACHE_LIMIT: 100,
  /** Maximum length of non-JSON output to log */
  MAX_NON_JSON_OUTPUT_LENGTH: 100,
} as const;

/** Scroll behavior configuration */
export const SCROLL_CONFIG = {
  /** Distance from bottom to trigger auto-scroll (px) */
  THRESHOLD: 150,
  /** Throttle delay for scroll events (ms) */
  THROTTLE_MS: 50,
} as const;

/** Typing indicator configuration */
export const TYPING_CONFIG = {
  /** Delay before showing typing indicator (ms) */
  INDICATOR_DELAY: 300,
  /** Minimum content increase to trigger scroll during streaming */
  CONTENT_INCREASE_THRESHOLD: 50,
} as const;

/** Animation configuration */
export const ANIMATION_CONFIG = {
  /** Logo breathe animation duration (ms) */
  LOGO_BREATHE_DURATION: 3000,
  /** Logo glitch animation duration (ms) */
  LOGO_GLITCH_DURATION: 1500,
  /** Logo evolution animation duration (ms) */
  LOGO_EVOLUTION_DURATION: 2500,
} as const;

/** Stream configuration */
export const STREAM_CONFIG = {
  /** Scanner buffer size for CLI output parsing */
  SCANNER_INITIAL_BUF_SIZE: 256 * 1024, // 256 KB
  SCANNER_MAX_BUF_SIZE: 1024 * 1024, // 1 MB
} as const;

/** Cost configuration (USD per million tokens) */
export const COST_CONFIG = {
  /** DeepSeek V3 input cost */
  DEEPSEEK_INPUT_COST_PER_MILLION: 0.27,
  /** DeepSeek V3 output cost */
  DEEPSEEK_OUTPUT_COST_PER_MILLION: 2.25,
} as const;

/** Combined chat configuration */
export const CHAT_CONFIG = {
  ...MESSAGE_CONFIG,
  ...SCROLL_CONFIG,
  ...TYPING_CONFIG,
  ...ANIMATION_CONFIG,
  ...STREAM_CONFIG,
  ...COST_CONFIG,
} as const;
