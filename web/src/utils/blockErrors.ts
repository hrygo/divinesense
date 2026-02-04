/**
 * Block Error Handling Utilities
 *
 * Provides error classification, recovery strategies, and user-friendly messages
 * for Block-related operations.
 *
 * @see docs/specs/unified-block-model.md
 */

// ============================================================================
// Error Types
// ============================================================================

/**
 * Base error class for Block operations
 */
export class BlockError extends Error {
  constructor(
    message: string,
    public code: string,
    public recoverable: boolean = true,
    public retryable: boolean = true
  ) {
    super(message);
    this.name = "BlockError";
  }
}

/**
 * Network error - connection issues, timeouts
 */
export class NetworkError extends BlockError {
  constructor(message: string, public readonly statusCode?: number) {
    super(message, "NETWORK_ERROR", true, true);
    this.name = "NetworkError";
  }
}

/**
 * Validation error - invalid input data
 */
export class ValidationError extends BlockError {
  constructor(message: string, public readonly field?: string) {
    super(message, "VALIDATION_ERROR", false, false);
    this.name = "ValidationError";
  }
}

/**
 * Conflict error - concurrent modification
 */
export class ConflictError extends BlockError {
  constructor(message: string, public readonly currentVersion?: number) {
    super(message, "CONFLICT_ERROR", true, true);
    this.name = "ConflictError";
  }
}

/**
 * Not found error - block or conversation doesn't exist
 */
export class NotFoundError extends BlockError {
  constructor(message: string, public readonly resourceType?: string) {
    super(message, "NOT_FOUND", false, false);
    this.name = "NotFoundError";
  }
}

/**
 * Permission error - user lacks access
 */
export class PermissionError extends BlockError {
  constructor(message: string) {
    super(message, "PERMISSION_ERROR", false, false);
    this.name = "PermissionError";
  }
}

/**
 * Quota exceeded error - rate limit or storage limit
 */
export class QuotaError extends BlockError {
  constructor(message: string, public readonly retryAfter?: number) {
    super(message, "QUOTA_ERROR", true, true);
    this.name = "QuotaError";
  }
}

// ============================================================================
// Error Classification
// ============================================================================

/**
 * Error severity levels
 */
export enum ErrorSeverity {
  LOW = "low",       // User can continue, data is safe
  MEDIUM = "medium", // Functionality affected but recoverable
  HIGH = "high",     // Critical error, user intervention needed
}

/**
 * Classify an error by its type and properties
 */
export function classifyError(error: unknown): {
  type: string;
  severity: ErrorSeverity;
  recoverable: boolean;
  retryable: boolean;
} {
  // BlockError instances
  if (error instanceof BlockError) {
    const severity = getSeverityForCode(error.code);
    return {
      type: error.code,
      severity,
      recoverable: error.recoverable,
      retryable: error.retryable,
    };
  }

  // Network errors (AbortError, TypeError for fetch failures)
  if (error instanceof DOMException && error.name === "AbortError") {
    return {
      type: "REQUEST_CANCELLED",
      severity: ErrorSeverity.LOW,
      recoverable: true,
      retryable: true,
    };
  }

  // Fetch API errors
  if (error instanceof TypeError && error.message.includes("fetch")) {
    return {
      type: "NETWORK_ERROR",
      severity: ErrorSeverity.MEDIUM,
      recoverable: true,
      retryable: true,
    };
  }

  // Unknown errors
  return {
    type: "UNKNOWN_ERROR",
    severity: ErrorSeverity.HIGH,
    recoverable: false,
    retryable: false,
  };
}

function getSeverityForCode(code: string): ErrorSeverity {
  switch (code) {
    case "NETWORK_ERROR":
    case "QUOTA_ERROR":
      return ErrorSeverity.MEDIUM;
    case "VALIDATION_ERROR":
    case "PERMISSION_ERROR":
      return ErrorSeverity.LOW;
    case "NOT_FOUND":
      return ErrorSeverity.MEDIUM;
    case "CONFLICT_ERROR":
      return ErrorSeverity.HIGH;
    default:
      return ErrorSeverity.HIGH;
  }
}

// ============================================================================
// Error Recovery Strategies
// ============================================================================

/**
 * Recovery action for a given error
 */
export interface RecoveryAction {
  label: string;          // User-facing action label
  action: () => void | Promise<void>;  // Action to execute
  primary?: boolean;      // Whether this is the primary action
}

/**
 * Get recovery actions for a given error
 */
export function getRecoveryActions(error: unknown, context?: {
  conversationId?: number;
  blockId?: number;
  retryCount?: number;
}): RecoveryAction[] {
  const classification = classifyError(error);
  const actions: RecoveryAction[] = [];

  switch (classification.type) {
    case "NETWORK_ERROR":
    case "QUOTA_ERROR":
      if (classification.retryable && (context?.retryCount ?? 0) < 3) {
        actions.push({
          label: "重试",
          action: () => {/* Trigger retry via mutation */},
          primary: true,
        });
      }
      actions.push({
        label: "检查网络连接",
        action: () => {/* Show network status */},
      });
      break;

    case "VALIDATION_ERROR":
      actions.push({
        label: "修改输入",
        action: () => {/* Focus input field */},
        primary: true,
      });
      break;

    case "CONFLICT_ERROR":
      actions.push({
        label: "刷新数据",
        action: () => {/* Invalidate and refetch */},
        primary: true,
      });
      actions.push({
        label: "保留本地修改",
        action: () => {/* Show conflict resolution */},
      });
      break;

    case "NOT_FOUND":
      actions.push({
        label: "返回列表",
        action: () => {/* Navigate back */},
        primary: true,
      });
      if (context?.conversationId) {
        actions.push({
          label: "创建新对话",
          action: () => {/* Start new conversation */},
        });
      }
      break;

    case "PERMISSION_ERROR":
      actions.push({
        label: "联系管理员",
        action: () => {/* Show contact info */},
        primary: true,
      });
      break;

    case "REQUEST_CANCELLED":
      actions.push({
        label: "重新发送",
        action: () => {/* Resubmit */},
        primary: true,
      });
      break;

    default:
      actions.push({
        label: "刷新页面",
        action: () => window.location.reload(),
        primary: true,
      });
  }

  return actions;
}

// ============================================================================
// User-Facing Error Messages
// ============================================================================

/**
 * Get user-friendly error message
 */
export function getUserMessage(error: unknown): string {
  const classification = classifyError(error);

  // Direct message from BlockError
  if (error instanceof BlockError) {
    return error.message;
  }

  // Network errors
  if (classification.type === "NETWORK_ERROR") {
    return "网络连接失败，请检查您的网络连接后重试。";
  }

  // Request cancelled
  if (classification.type === "REQUEST_CANCELLED") {
    return "请求已取消。";
  }

  // Default message
  return "操作失败，请稍后重试。如果问题持续存在，请联系支持团队。";
}

/**
 * Get error title for UI display
 */
export function getErrorTitle(error: unknown): string {
  const classification = classifyError(error);

  switch (classification.type) {
    case "NETWORK_ERROR":
      return "网络错误";
    case "VALIDATION_ERROR":
      return "输入验证失败";
    case "CONFLICT_ERROR":
      return "数据冲突";
    case "NOT_FOUND":
      return "未找到";
    case "PERMISSION_ERROR":
      return "权限不足";
    case "QUOTA_ERROR":
      return "已达限制";
    case "REQUEST_CANCELLED":
      return "请求已取消";
    default:
      return "操作失败";
  }
}

// ============================================================================
// Retry Strategy
// ============================================================================

/**
 * Retry configuration
 */
export interface RetryConfig {
  maxRetries: number;
  retryDelay: (attempt: number) => number;
  shouldRetry: (error: unknown) => boolean;
}

/**
 * Default retry configuration with exponential backoff
 */
export const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 30000), // 1s, 2s, 4s, max 30s
  shouldRetry: (error) => {
    const classification = classifyError(error);
    return classification.retryable;
  },
};

/**
 * Calculate retry delay with jitter to avoid thundering herd
 */
export function getRetryDelay(attempt: number, baseDelay: number = 1000): number {
  const exponentialDelay = baseDelay * 2 ** attempt;
  const jitter = Math.random() * 0.3 * exponentialDelay; // ±15% jitter
  return Math.min(exponentialDelay + jitter, 30000); // Max 30s
}

/**
 * Check if operation should be retried
 */
export function shouldRetry(error: unknown, attempt: number, maxRetries: number = 3): boolean {
  if (attempt >= maxRetries) {
    return false;
  }

  const classification = classifyError(error);
  return classification.retryable;
}

// ============================================================================
// Error Logging
// ============================================================================

interface ErrorLogContext {
  operation?: string;
  conversationId?: number;
  blockId?: number;
  userId?: number;
  timestamp?: number;
  stack?: string;
}

const errorLog: Array<{ error: unknown; context: ErrorLogContext }> = [];

/**
 * Log error for debugging and monitoring
 */
export function logError(error: unknown, context: ErrorLogContext = {}): void {
  const logEntry = {
    error,
    context: {
      ...context,
      timestamp: Date.now(),
      stack: error instanceof Error ? error.stack : undefined,
    },
  };

  errorLog.push(logEntry);

  // Keep only last 100 errors
  if (errorLog.length > 100) {
    errorLog.shift();
  }

  // In development, log to console
  if (import.meta.env.DEV) {
    console.error("[BlockError]", logEntry);
  }

  // TODO: Send to error tracking service (e.g., Sentry)
}

/**
 * Get recent errors for debugging
 */
export function getRecentErrors(count: number = 10): Array<{ error: unknown; context: ErrorLogContext }> {
  return errorLog.slice(-count);
}

/**
 * Clear error log
 */
export function clearErrorLog(): void {
  errorLog.length = 0;
}

// ============================================================================
// Utilities
// ============================================================================

/**
 * Wrap an async function with error handling
 */
export function withErrorHandling<T extends (...args: any[]) => Promise<any>>(
  fn: T,
  context?: ErrorLogContext
): T {
  return (async (...args: any[]) => {
    try {
      return await fn(...args);
    } catch (error) {
      logError(error, context);
      throw error;
    }
  }) as T;
}

/**
 * Parse API error response
 */
export function parseApiError(response: Response): Promise<BlockError> {
  const status = response.status;

  switch (status) {
    case 400:
      return new ValidationError("请求数据格式错误");
    case 401:
    case 403:
      return new PermissionError("您没有权限执行此操作");
    case 404:
      return new NotFoundError("请求的资源不存在");
    case 409:
      return new ConflictError("数据已被修改，请刷新后重试");
    case 429:
      const retryAfter = response.headers.get("Retry-After");
      return new QuotaError("请求过于频繁，请稍后再试", retryAfter ? parseInt(retryAfter) : undefined);
    case 500:
    case 502:
    case 503:
    case 504:
      return new NetworkError("服务暂时不可用，请稍后重试", status);
    default:
      return new BlockError("未知错误", "UNKNOWN_ERROR", false, false);
  }
}

/**
 * Check if error is recoverable
 */
export function isRecoverable(error: unknown): boolean {
  const classification = classifyError(error);
  return classification.recoverable;
}

/**
 * Check if error should be retried
 */
export function isRetryable(error: unknown): boolean {
  const classification = classifyError(error);
  return classification.retryable;
}
