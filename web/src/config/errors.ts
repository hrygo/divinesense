/**
 * API Error Types
 * API 错误类型定义 - 提供类型安全的错误处理
 */

/** Standard API error structure */
export interface ApiError {
  /** HTTP status code (if available) */
  status?: number;
  /** Error code from backend */
  code?: string;
  /** Human-readable error message */
  message?: string;
  /** Additional error details */
  details?: unknown;
  /** Whether this is a network error */
  isNetworkError?: boolean;
  /** Whether this is a timeout error */
  isTimeoutError?: boolean;
}

/** Error categories for retry logic */
export enum ErrorCategory {
  /** Network errors - should retry with exponential backoff */
  NETWORK = "network",
  /** Timeout errors - should retry immediately */
  TIMEOUT = "timeout",
  /** Server errors (5xx) - don't retry, may be persistent */
  SERVER = "server",
  /** Client errors (4xx) - don't retry, need user action */
  CLIENT = "client",
  /** Unknown error - use default retry strategy */
  UNKNOWN = "unknown",
}

/**
 * Classify an error into a category for retry logic
 *
 * @param error - The error to classify
 * @returns The error category
 */
export function classifyError(error: unknown): ErrorCategory {
  const apiError = error as ApiError;

  // Check for timeout
  if (apiError.isTimeoutError || apiError.code === "TIMEOUT") {
    return ErrorCategory.TIMEOUT;
  }

  // Check for network error
  if (apiError.isNetworkError || apiError.code === "NETWORK_ERROR") {
    return ErrorCategory.NETWORK;
  }

  // Check for server error (5xx)
  if (apiError.status && apiError.status >= 500 && apiError.status < 600) {
    return ErrorCategory.SERVER;
  }

  // Check for client error (4xx)
  if (apiError.status && apiError.status >= 400 && apiError.status < 500) {
    return ErrorCategory.CLIENT;
  }

  return ErrorCategory.UNKNOWN;
}

/**
 * Check if an error should be retried
 *
 * @param error - The error to check
 * @param failureCount - Number of retries already attempted
 * @param maxRetries - Maximum number of retries allowed
 * @returns Whether to retry the request
 */
export function shouldRetryError(error: unknown, failureCount: number, maxRetries: number = 3): boolean {
  const category = classifyError(error);

  switch (category) {
    case ErrorCategory.SERVER:
      // Don't retry on 500 errors (server-side issues that won't be fixed by retrying)
      return false;
    case ErrorCategory.CLIENT:
      // Don't retry on client errors (need user action)
      return false;
    case ErrorCategory.TIMEOUT:
      // Retry timeout errors immediately
      return failureCount < maxRetries;
    case ErrorCategory.NETWORK:
    case ErrorCategory.UNKNOWN:
      // Retry network and unknown errors with exponential backoff
      return failureCount < maxRetries;
    default:
      return false;
  }
}

/**
 * Create a retry delay function for exponential backoff
 *
 * @param baseDelay - Base delay in milliseconds
 * @param maxDelay - Maximum delay in milliseconds
 * @returns A function that takes attempt index and returns delay
 */
export function createRetryDelay(baseDelay: number = 1000, maxDelay: number = 30000) {
  return (attemptIndex: number): number => {
    const delay = baseDelay * 2 ** attemptIndex;
    return Math.min(delay, maxDelay);
  };
}
